package models

// AgentMarketEvent persists normalized discrete market events such as official
// announcements, Alpha listings, external news and deterministic local Signals.
type AgentMarketEvent struct {
	ID          int64   `orm:"column(id);auto" json:"id"`
	EventKey    string  `orm:"column(event_key);size(64);unique" json:"event_key"`
	Type        string  `orm:"column(event_type);size(64);index" json:"type"`
	Category    string  `orm:"column(category);size(64);index" json:"category"`
	SymbolsJSON string  `orm:"column(symbols_json);type(text);null" json:"-"`
	EventTime   int64   `orm:"column(event_time);index" json:"event_time"`
	ObservedAt  int64   `orm:"column(observed_at);index" json:"observed_at"`
	Source      string  `orm:"column(source);size(96);index" json:"source"`
	SourceRef   string  `orm:"column(source_ref);size(512);null" json:"source_ref,omitempty"`
	Headline    string  `orm:"column(headline);size(512);null" json:"headline,omitempty"`
	Summary     string  `orm:"column(summary);type(text);null" json:"summary,omitempty"`
	Severity    string  `orm:"column(severity);size(16);index" json:"severity"`
	Confidence  float64 `orm:"column(confidence);digits(8);decimals(6)" json:"confidence"`
	Freshness   string  `orm:"column(freshness);size(16);index" json:"freshness"`
	RawRef      string  `orm:"column(raw_ref);size(512);null" json:"raw_ref,omitempty"`
	RawJSON     string  `orm:"column(raw_json);type(text);null" json:"-"`
	CreatedAt   int64   `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt   int64   `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentMarketEvent) TableName() string { return "agent_market_events" }

// AgentMarketEventSource keeps all observed sources for one canonical event.
type AgentMarketEventSource struct {
	ID         int64  `orm:"column(id);auto" json:"id"`
	EventID    int64  `orm:"column(event_id);index" json:"event_id"`
	SourceKey  string `orm:"column(source_key);size(64);unique" json:"source_key"`
	Source     string `orm:"column(source);size(96);index" json:"source"`
	SourceRef  string `orm:"column(source_ref);size(512);null" json:"source_ref,omitempty"`
	RawRef     string `orm:"column(raw_ref);size(512);null" json:"raw_ref,omitempty"`
	ObservedAt int64  `orm:"column(observed_at);index" json:"observed_at"`
	CreatedAt  int64  `orm:"column(created_at);index" json:"created_at"`
}

func (*AgentMarketEventSource) TableName() string { return "agent_market_event_sources" }

// AgentMarketFact persists point-in-time deterministic market observations.
type AgentMarketFact struct {
	ID         int64   `orm:"column(id);auto" json:"id"`
	FactKey    string  `orm:"column(fact_key);size(64);unique" json:"fact_key"`
	Type       string  `orm:"column(fact_type);size(64);index" json:"type"`
	Category   string  `orm:"column(category);size(64);index" json:"category"`
	Symbol     string  `orm:"column(symbol);size(32);index" json:"symbol"`
	EventTime  int64   `orm:"column(event_time);index" json:"event_time"`
	ObservedAt int64   `orm:"column(observed_at);index" json:"observed_at"`
	Source     string  `orm:"column(source);size(96);index" json:"source"`
	SourceRef  string  `orm:"column(source_ref);size(512);null" json:"source_ref,omitempty"`
	Severity   string  `orm:"column(severity);size(16);index" json:"severity"`
	Confidence float64 `orm:"column(confidence);digits(8);decimals(6)" json:"confidence"`
	Freshness  string  `orm:"column(freshness);size(16);index" json:"freshness"`
	DataJSON   string  `orm:"column(data_json);type(text)" json:"-"`
	RawRef     string  `orm:"column(raw_ref);size(512);null" json:"raw_ref,omitempty"`
	CreatedAt  int64   `orm:"column(created_at);index" json:"created_at"`
}

func (*AgentMarketFact) TableName() string { return "agent_market_facts" }

// AgentMarketSourceStatus makes provider failures explicit without blocking the
// deterministic market-data path.
type AgentMarketSourceStatus struct {
	ID            int64  `orm:"column(id);auto" json:"id"`
	Source        string `orm:"column(source);size(96);unique" json:"source"`
	Status        string `orm:"column(status);size(32);index" json:"status"`
	LastSuccessAt int64  `orm:"column(last_success_at);index" json:"last_success_at,omitempty"`
	LastErrorAt   int64  `orm:"column(last_error_at);index" json:"last_error_at,omitempty"`
	LastError     string `orm:"column(last_error);type(text);null" json:"last_error,omitempty"`
	UpdatedAt     int64  `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentMarketSourceStatus) TableName() string { return "agent_market_source_status" }
