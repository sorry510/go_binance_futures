package modelgateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"go_binance_futures/llm"
)

type routingStore interface {
	RouterSettings(context.Context) (llm.RouterSettings, error)
	RoutingConfigs(context.Context) ([]llm.RoutingConfig, error)
}

type Router struct {
	Store     routingStore
	Health    *HealthRegistry
	NewClient func(llm.Config) (llm.Client, error)
}

var defaultHealth = NewHealthRegistry()
var defaultRouter = &Router{Store: llm.Store{}, Health: defaultHealth, NewClient: llm.NewClient}

func Default() *Router               { return defaultRouter }
func DefaultHealth() *HealthRegistry { return defaultHealth }

type routedCandidate struct {
	route  llm.RouteCandidate
	config llm.Config
	client llm.Client
}

func (router *Router) Route(ctx context.Context, request llm.RouteRequest) (llm.Client, llm.RouteDecision, error) {
	if router == nil || router.Store == nil {
		return nil, llm.RouteDecision{}, fmt.Errorf("model gateway store is required")
	}
	if router.Health == nil {
		router.Health = NewHealthRegistry()
	}
	if router.NewClient == nil {
		router.NewClient = llm.NewClient
	}
	settings, err := router.Store.RouterSettings(ctx)
	if err != nil {
		return nil, llm.RouteDecision{}, err
	}
	configs, err := router.Store.RoutingConfigs(ctx)
	if err != nil {
		return nil, llm.RouteDecision{}, err
	}
	if len(configs) == 0 {
		return nil, llm.RouteDecision{}, fmt.Errorf("no enabled LLM configuration; configure one in AI -> LLM 配置")
	}
	primary := -1
	for index := range configs {
		if configs[index].Primary {
			primary = index
			break
		}
	}
	if primary < 0 {
		return nil, llm.RouteDecision{}, fmt.Errorf("model gateway requires one enabled primary LLM configuration")
	}
	if settings.Enabled != 1 {
		return router.single(configs[primary], request, "router disabled; using enabled primary model")
	}
	return router.routed(configs, settings, request)
}

func (router *Router) single(config llm.RoutingConfig, request llm.RouteRequest, reason string) (llm.Client, llm.RouteDecision, error) {
	client, err := router.NewClient(config.Config)
	if err != nil {
		return nil, llm.RouteDecision{}, err
	}
	candidate := makeCandidate(config, scoreCandidate(config, request.Requirements, HealthSnapshot{}))
	decision := llm.RouteDecision{Enabled: false, Reason: reason, Candidates: []llm.RouteCandidate{candidate}, Selected: candidate}
	return &gatewayClient{provider: candidate.Provider, initialConfigID: candidate.ConfigID, settings: llm.DefaultRouterSettings(), health: router.Health, candidates: []routedCandidate{{route: candidate, config: config.Config, client: client}}, reason: reason}, decision, nil
}

func (router *Router) routed(configs []llm.RoutingConfig, settings llm.RouterSettings, request llm.RouteRequest) (llm.Client, llm.RouteDecision, error) {
	candidates := make([]routedCandidate, 0, len(configs))
	capabilityMatches := 0
	circuitOpen := make([]int64, 0)
	initFailures := make([]string, 0)
	for _, config := range configs {
		if !matches(config.Profile, request.Requirements) {
			continue
		}
		capabilityMatches++
		if !router.Health.Available(config.Config.ID) {
			circuitOpen = append(circuitOpen, config.Config.ID)
			continue
		}
		health := router.Health.Snapshot(config.Config.ID)
		route := makeCandidate(config, scoreCandidate(config, request.Requirements, health))
		client, err := router.NewClient(config.Config)
		if err != nil {
			initFailures = append(initFailures, fmt.Sprintf("config_id=%d: %v", config.Config.ID, err))
			continue
		}
		candidates = append(candidates, routedCandidate{route: route, config: config.Config, client: client})
	}
	if len(candidates) == 0 {
		if capabilityMatches == 0 {
			return nil, llm.RouteDecision{}, fmt.Errorf("no LLM model satisfies requirements for skill %q", request.Skill)
		}
		return nil, llm.RouteDecision{}, fmt.Errorf("%d LLM model(s) satisfy requirements for skill %q but none are currently available (circuit_open=%v init_failures=%v)", capabilityMatches, request.Skill, circuitOpen, initFailures)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].route.Score == candidates[j].route.Score {
			return candidates[i].route.ConfigID < candidates[j].route.ConfigID
		}
		return candidates[i].route.Score > candidates[j].route.Score
	})
	routes := make([]llm.RouteCandidate, len(candidates))
	for i := range candidates {
		routes[i] = candidates[i].route
	}
	selected := routes[0]
	reason := fmt.Sprintf("router enabled; skill=%s matched=%d selected=%s/%s score=%d", request.Skill, len(routes), selected.Provider, selected.Model, selected.Score)
	client := &gatewayClient{provider: selected.Provider, initialConfigID: selected.ConfigID, settings: settings, health: router.Health, candidates: candidates, reason: reason}
	return client, llm.RouteDecision{Enabled: true, Reason: reason, Candidates: routes, Selected: selected}, nil
}

func makeCandidate(config llm.RoutingConfig, score int) llm.RouteCandidate {
	return llm.RouteCandidate{ConfigID: config.Config.ID, Name: config.Name, Provider: config.Config.Provider, Model: config.Config.Model, Primary: config.Primary, Score: score, Profile: config.Profile}
}

func matches(profile llm.ModelProfile, requirements llm.ModelRequirements) bool {
	if requirements.StructuredOutput && !profile.StructuredOutput {
		return false
	}
	if requirements.NativeToolCalling && !profile.NativeToolCalling {
		return false
	}
	if requirements.Reasoning && !profile.Reasoning {
		return false
	}
	if requirements.LongContext && !profile.LongContext {
		return false
	}
	if requirements.MinJSONReliability > 0 && profile.JSONReliability < requirements.MinJSONReliability {
		return false
	}
	if requirements.MinContextTokens > 0 && profile.MaxContextTokens < requirements.MinContextTokens {
		return false
	}
	return true
}

func scoreCandidate(config llm.RoutingConfig, requirements llm.ModelRequirements, health HealthSnapshot) int {
	score := config.Profile.JSONReliability
	if config.Primary {
		score += 35
	}
	if requirements.Reasoning && config.Profile.Reasoning {
		score += 30
	}
	if requirements.LongContext && config.Profile.LongContext {
		score += 20
	}
	if requirements.PreferLowLatency {
		score += classScore(config.Profile.LatencyClass, 20, 10)
	}
	if requirements.PreferLowCost {
		score += classScore(config.Profile.CostClass, 15, 7)
	}
	if health.Samples > 0 {
		score += int(health.SuccessRate * 30)
		if health.AverageLatencyMs > 10000 {
			score -= 10
		}
	}
	return score
}

func classScore(value string, low, medium int) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return low
	case "medium":
		return medium
	default:
		return 0
	}
}

type gatewayClient struct {
	mu              sync.Mutex
	provider        llm.Provider
	initialConfigID int64
	settings        llm.RouterSettings
	health          *HealthRegistry
	candidates      []routedCandidate
	reason          string
	attempts        []llm.RouteAttempt
}

func (client *gatewayClient) Provider() llm.Provider { return client.provider }
func (client *gatewayClient) ConfigID() int64        { return client.initialConfigID }
func (client *gatewayClient) RouteDecision() llm.RouteDecision {
	routes := make([]llm.RouteCandidate, len(client.candidates))
	for index := range client.candidates {
		routes[index] = client.candidates[index].route
	}
	decision := llm.RouteDecision{Enabled: client.settings.Enabled == 1, Reason: client.reason, Candidates: routes}
	if len(routes) > 0 {
		decision.Selected = routes[0]
	}
	return decision
}

func (client *gatewayClient) Generate(ctx context.Context, request llm.Request) (*llm.Response, error) {
	if len(client.candidates) == 0 {
		return nil, fmt.Errorf("model gateway has no candidates")
	}
	cooldown := time.Duration(client.settings.CooldownSeconds) * time.Second
	var lastErr error
	for index, candidate := range client.candidates {
		if index > 0 && client.settings.FallbackEnabled != 1 {
			break
		}
		if !client.health.Allow(candidate.route.ConfigID, cooldown) {
			client.appendAttempt(llm.RouteAttempt{ConfigID: candidate.route.ConfigID, Provider: candidate.route.Provider, Model: candidate.route.Model, Status: "circuit_open", ErrorType: "circuit_open"})
			continue
		}
		started := time.Now()
		response, err := candidate.client.Generate(ctx, request)
		duration := time.Since(started)
		kind, fallback := classifyRouteError(err)
		if err == nil {
			client.health.Record(candidate.route.ConfigID, true, "", duration, client.settings.HealthWindow, client.settings.FailureThreshold, cooldown)
			attempt := llm.RouteAttempt{ConfigID: candidate.route.ConfigID, Provider: candidate.route.Provider, Model: candidate.route.Model, Status: "success", DurationMs: duration.Milliseconds()}
			client.appendAttempt(attempt)
			if response == nil {
				response = &llm.Response{}
			}
			response.Provider = candidate.route.Provider
			response.ConfigID = candidate.route.ConfigID
			response.RouteTrace = client.trace(candidate.route.ConfigID)
			return response, nil
		}
		lastErr = err
		client.health.Record(candidate.route.ConfigID, false, kind, duration, client.settings.HealthWindow, client.settings.FailureThreshold, cooldown)
		client.appendAttempt(llm.RouteAttempt{ConfigID: candidate.route.ConfigID, Provider: candidate.route.Provider, Model: candidate.route.Model, Status: "error", ErrorType: kind, Error: err.Error(), DurationMs: duration.Milliseconds()})
		if !fallback {
			return nil, fmt.Errorf("model gateway candidate %s/%s failed with non-fallback error: %w", candidate.route.Provider, candidate.route.Model, err)
		}
		if client.settings.FallbackEnabled != 1 {
			return nil, fmt.Errorf("model gateway fallback disabled after %s/%s failed: %w", candidate.route.Provider, candidate.route.Model, err)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("all model gateway candidates are unavailable")
	}
	return nil, fmt.Errorf("model gateway exhausted candidates: %w", lastErr)
}

func (client *gatewayClient) appendAttempt(value llm.RouteAttempt) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.attempts = append(client.attempts, value)
}

func (client *gatewayClient) trace(finalConfigID int64) *llm.RouteTrace {
	client.mu.Lock()
	defer client.mu.Unlock()
	return &llm.RouteTrace{InitialConfigID: client.initialConfigID, FinalConfigID: finalConfigID, Reason: client.reason, Attempts: append([]llm.RouteAttempt(nil), client.attempts...)}
}

func classifyRouteError(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return "network", true
	}
	var networkError net.Error
	if errors.As(err, &networkError) && (networkError.Timeout() || networkError.Temporary()) {
		return "timeout", true
	}
	var httpError *llm.HTTPError
	if errors.As(err, &httpError) {
		switch {
		case httpError.StatusCode == 429:
			return "429", true
		case httpError.StatusCode >= 500:
			return "5xx", true
		default:
			return fmt.Sprintf("http_%d", httpError.StatusCode), false
		}
	}
	return "request", false
}
