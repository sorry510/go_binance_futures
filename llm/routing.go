package llm

import "context"

type ModelProfile struct {
	StructuredOutput  bool   `json:"structured_output"`
	NativeToolCalling bool   `json:"native_tool_calling"`
	Reasoning         bool   `json:"reasoning"`
	LongContext       bool   `json:"long_context"`
	JSONReliability   int    `json:"json_reliability"`
	MaxContextTokens  int    `json:"max_context_tokens"`
	CostClass         string `json:"cost_class"`
	LatencyClass      string `json:"latency_class"`
}

type ModelRequirements struct {
	StructuredOutput   bool `json:"structured_output,omitempty"`
	NativeToolCalling  bool `json:"native_tool_calling,omitempty"`
	Reasoning          bool `json:"reasoning,omitempty"`
	LongContext        bool `json:"long_context,omitempty"`
	MinJSONReliability int  `json:"min_json_reliability,omitempty"`
	MinContextTokens   int  `json:"min_context_tokens,omitempty"`
	PreferLowCost      bool `json:"prefer_low_cost,omitempty"`
	PreferLowLatency   bool `json:"prefer_low_latency,omitempty"`
}

type RouteCandidate struct {
	ConfigID int64        `json:"config_id"`
	Name     string       `json:"name"`
	Provider Provider     `json:"provider"`
	Model    string       `json:"model"`
	Primary  bool         `json:"primary"`
	Score    int          `json:"score"`
	Profile  ModelProfile `json:"profile"`
}

type RouteAttempt struct {
	ConfigID   int64    `json:"config_id"`
	Provider   Provider `json:"provider"`
	Model      string   `json:"model"`
	Status     string   `json:"status"`
	ErrorType  string   `json:"error_type,omitempty"`
	Error      string   `json:"error,omitempty"`
	DurationMs int64    `json:"duration_ms"`
}

type RouteDecision struct {
	Enabled    bool             `json:"enabled"`
	Reason     string           `json:"reason"`
	Candidates []RouteCandidate `json:"candidates"`
	Selected   RouteCandidate   `json:"selected"`
}

type RouteTrace struct {
	InitialConfigID int64          `json:"initial_config_id"`
	FinalConfigID   int64          `json:"final_config_id"`
	Reason          string         `json:"reason"`
	Attempts        []RouteAttempt `json:"attempts"`
}

type RouteRequest struct {
	Skill        string            `json:"skill"`
	Requirements ModelRequirements `json:"requirements"`
}

type Router interface {
	Route(context.Context, RouteRequest) (Client, RouteDecision, error)
}

type RouteDecisionProvider interface {
	RouteDecision() RouteDecision
}

func ClientRouteDecision(client Client) (RouteDecision, bool) {
	provider, ok := client.(RouteDecisionProvider)
	if !ok || provider == nil {
		return RouteDecision{}, false
	}
	return provider.RouteDecision(), true
}

type RouterSettings struct {
	Enabled          int `json:"enabled"`
	FallbackEnabled  int `json:"fallback_enabled"`
	FailureThreshold int `json:"failure_threshold"`
	CooldownSeconds  int `json:"cooldown_seconds"`
	HealthWindow     int `json:"health_window"`
}

type RoutingConfig struct {
	Config  Config
	Name    string
	Primary bool
	Profile ModelProfile
}
