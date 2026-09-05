package modelgateway

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go_binance_futures/llm"
)

type fakeStore struct {
	settings llm.RouterSettings
	configs  []llm.RoutingConfig
}

func (store fakeStore) RouterSettings(context.Context) (llm.RouterSettings, error) {
	return store.settings, nil
}
func (store fakeStore) RoutingConfigs(context.Context) ([]llm.RoutingConfig, error) {
	return store.configs, nil
}

type fakeClient struct {
	provider llm.Provider
	response *llm.Response
	err      error
	calls    int
}

func (client *fakeClient) Provider() llm.Provider { return client.provider }
func (client *fakeClient) Generate(context.Context, llm.Request) (*llm.Response, error) {
	client.calls++
	if client.err != nil {
		return nil, client.err
	}
	copy := *client.response
	return &copy, nil
}

func routingConfig(id int64, primary bool, provider llm.Provider, model string, profile llm.ModelProfile) llm.RoutingConfig {
	return llm.RoutingConfig{Config: llm.Config{ID: id, Provider: provider, Model: model, APIURL: "http://test", Timeout: time.Second}, Name: model, Primary: primary, Profile: profile}
}

func newTestRouter(store fakeStore, clients map[int64]*fakeClient) *Router {
	return &Router{Store: store, Health: NewHealthRegistry(), NewClient: func(cfg llm.Config) (llm.Client, error) {
		client := clients[cfg.ID]
		if client == nil {
			return nil, errors.New("missing fake client")
		}
		return client, nil
	}}
}

func TestRouterDisabledUsesPrimaryOnly(t *testing.T) {
	primary := routingConfig(1, true, llm.ProviderOpenAI, "primary", llm.ModelProfile{StructuredOutput: true, JSONReliability: 80})
	secondary := routingConfig(2, false, llm.ProviderDeepSeek, "secondary", llm.ModelProfile{StructuredOutput: true, JSONReliability: 100})
	clients := map[int64]*fakeClient{1: {provider: llm.ProviderOpenAI, response: &llm.Response{Model: "primary", Content: "ok"}}, 2: {provider: llm.ProviderDeepSeek, response: &llm.Response{Model: "secondary", Content: "ok"}}}
	router := newTestRouter(fakeStore{settings: llm.DefaultRouterSettings(), configs: []llm.RoutingConfig{primary, secondary}}, clients)
	client, decision, err := router.Route(context.Background(), llm.RouteRequest{Skill: "test", Requirements: llm.ModelRequirements{StructuredOutput: true}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Enabled || decision.Selected.ConfigID != 1 || len(decision.Candidates) != 1 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if _, err := client.Generate(context.Background(), llm.Request{}); err != nil {
		t.Fatal(err)
	}
	if clients[1].calls != 1 || clients[2].calls != 0 {
		t.Fatalf("calls primary=%d secondary=%d", clients[1].calls, clients[2].calls)
	}
}

func TestRouterCapabilitySelectsMatchingCandidate(t *testing.T) {
	primary := routingConfig(1, true, llm.ProviderOpenAI, "primary", llm.ModelProfile{StructuredOutput: true, Reasoning: false, JSONReliability: 90})
	reasoning := routingConfig(2, false, llm.ProviderDeepSeek, "reasoning", llm.ModelProfile{StructuredOutput: true, Reasoning: true, JSONReliability: 85})
	settings := llm.DefaultRouterSettings()
	settings.Enabled = 1
	clients := map[int64]*fakeClient{1: {provider: llm.ProviderOpenAI, response: &llm.Response{}}, 2: {provider: llm.ProviderDeepSeek, response: &llm.Response{}}}
	router := newTestRouter(fakeStore{settings: settings, configs: []llm.RoutingConfig{primary, reasoning}}, clients)
	_, decision, err := router.Route(context.Background(), llm.RouteRequest{Skill: "strategy_builder", Requirements: llm.ModelRequirements{StructuredOutput: true, Reasoning: true, MinJSONReliability: 70}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Selected.ConfigID != 2 || len(decision.Candidates) != 1 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestGatewayFallbackOn429(t *testing.T) {
	primary := routingConfig(1, true, llm.ProviderOpenAI, "primary", llm.ModelProfile{StructuredOutput: true, JSONReliability: 90})
	secondary := routingConfig(2, false, llm.ProviderDeepSeek, "secondary", llm.ModelProfile{StructuredOutput: true, JSONReliability: 80})
	settings := llm.DefaultRouterSettings()
	settings.Enabled = 1
	clients := map[int64]*fakeClient{1: {provider: llm.ProviderOpenAI, err: &llm.HTTPError{StatusCode: 429, Body: "busy"}}, 2: {provider: llm.ProviderDeepSeek, response: &llm.Response{Model: "secondary", Content: "ok"}}}
	router := newTestRouter(fakeStore{settings: settings, configs: []llm.RoutingConfig{primary, secondary}}, clients)
	client, _, err := router.Route(context.Background(), llm.RouteRequest{Skill: "test", Requirements: llm.ModelRequirements{StructuredOutput: true}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Generate(context.Background(), llm.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if response.ConfigID != 2 || response.Provider != llm.ProviderDeepSeek {
		t.Fatalf("unexpected response route: %+v", response)
	}
	if response.RouteTrace == nil || len(response.RouteTrace.Attempts) != 2 {
		t.Fatalf("missing fallback trace: %+v", response.RouteTrace)
	}
}

func TestHealthCircuitOpensAndHalfOpens(t *testing.T) {
	health := NewHealthRegistry()
	cooldown := 20 * time.Millisecond
	health.Record(7, false, "5xx", time.Millisecond, 20, 2, cooldown)
	health.Record(7, false, "5xx", time.Millisecond, 20, 2, cooldown)
	if health.Allow(7, cooldown) {
		t.Fatal("open circuit must reject request")
	}
	time.Sleep(25 * time.Millisecond)
	if !health.Allow(7, cooldown) {
		t.Fatal("expired circuit must allow one half-open request")
	}
	if health.Allow(7, cooldown) {
		t.Fatal("half-open circuit must allow only one in-flight request")
	}
	health.Record(7, true, "", time.Millisecond, 20, 2, cooldown)
	if !health.Allow(7, cooldown) {
		t.Fatal("successful half-open request must close circuit")
	}
	if state := health.Snapshot(7).State; state != "closed" {
		t.Fatalf("state=%s", state)
	}
}
func TestHealthClientErrorsDoNotOpenCircuit(t *testing.T) {
	health := NewHealthRegistry()
	cooldown := time.Minute
	for i := 0; i < 5; i++ {
		health.Record(8, false, "http_400", time.Millisecond, 20, 3, cooldown)
	}
	snapshot := health.Snapshot(8)
	if snapshot.State != "closed" || snapshot.ConsecutiveFailures != 0 {
		t.Fatalf("client errors must not open circuit: %+v", snapshot)
	}
	if !health.Allow(8, cooldown) {
		t.Fatal("client errors must leave model available")
	}
}

func TestRouterReportsCircuitOpenSeparatelyFromCapabilityMismatch(t *testing.T) {
	primary := routingConfig(1, true, llm.ProviderOpenAI, "primary", llm.ModelProfile{StructuredOutput: true, Reasoning: false, JSONReliability: 90})
	reasoning := routingConfig(2, false, llm.ProviderDeepSeek, "reasoning", llm.ModelProfile{StructuredOutput: true, Reasoning: true, JSONReliability: 85})
	settings := llm.DefaultRouterSettings()
	settings.Enabled = 1
	router := newTestRouter(fakeStore{settings: settings, configs: []llm.RoutingConfig{primary, reasoning}}, map[int64]*fakeClient{1: {provider: llm.ProviderOpenAI, response: &llm.Response{}}, 2: {provider: llm.ProviderDeepSeek, response: &llm.Response{}}})
	router.Health.Record(2, false, "5xx", time.Millisecond, 20, 1, time.Minute)
	_, _, err := router.Route(context.Background(), llm.RouteRequest{Skill: "strategy_builder", Requirements: llm.ModelRequirements{StructuredOutput: true, Reasoning: true, MinJSONReliability: 75}})
	if err == nil || !strings.Contains(err.Error(), "circuit_open=[2]") {
		t.Fatalf("expected circuit-open diagnostic, got %v", err)
	}
}
