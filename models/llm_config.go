package models

type LLMConfig struct {
	ID                int64   `orm:"column(id);auto" json:"id"`
	Name              string  `orm:"column(name);size(128);unique" json:"name"`
	Provider          string  `orm:"column(provider);size(32)" json:"provider"`
	APIURL            string  `orm:"column(api_url);type(text)" json:"api_url"`
	APIKey            string  `orm:"column(api_key);type(text)" json:"-"`
	Model             string  `orm:"column(model);size(256)" json:"model"`
	APIVersion        string  `orm:"column(api_version);size(64)" json:"api_version,omitempty"`
	TimeoutSeconds    int     `orm:"column(timeout_seconds);default(60)" json:"timeout_seconds"`
	Temperature       float64 `orm:"column(temperature);digits(6);decimals(3);default(0.2)" json:"temperature"`
	Enabled           int     `orm:"column(enabled);default(0)" json:"enabled"`
	RouterCandidate   int     `orm:"column(router_candidate);default(0)" json:"router_candidate"`
	StructuredOutput  int     `orm:"column(structured_output);default(1)" json:"structured_output"`
	NativeToolCalling int     `orm:"column(native_tool_calling);default(0)" json:"native_tool_calling"`
	Reasoning         int     `orm:"column(reasoning);default(0)" json:"reasoning"`
	LongContext       int     `orm:"column(long_context);default(0)" json:"long_context"`
	JSONReliability   int     `orm:"column(json_reliability);default(80)" json:"json_reliability"`
	MaxContextTokens  int     `orm:"column(max_context_tokens);default(0)" json:"max_context_tokens"`
	CostClass         string  `orm:"column(cost_class);size(16);default(medium)" json:"cost_class"`
	LatencyClass      string  `orm:"column(latency_class);size(16);default(medium)" json:"latency_class"`
	CreatedAt         int64   `orm:"column(created_at)" json:"created_at"`
	UpdatedAt         int64   `orm:"column(updated_at)" json:"updated_at"`
	Deleted           int     `orm:"column(deleted);default(0)" json:"-"`
}

func (*LLMConfig) TableName() string {
	return "llm_configs"
}

type LLMRouterSetting struct {
	ID               int64 `orm:"column(id);pk" json:"id"`
	Enabled          int   `orm:"column(enabled);default(0)" json:"enabled"`
	FallbackEnabled  int   `orm:"column(fallback_enabled);default(1)" json:"fallback_enabled"`
	FailureThreshold int   `orm:"column(failure_threshold);default(3)" json:"failure_threshold"`
	CooldownSeconds  int   `orm:"column(cooldown_seconds);default(60)" json:"cooldown_seconds"`
	HealthWindow     int   `orm:"column(health_window);default(20)" json:"health_window"`
	UpdatedAt        int64 `orm:"column(updated_at)" json:"updated_at"`
}

func (*LLMRouterSetting) TableName() string { return "llm_router_settings" }
