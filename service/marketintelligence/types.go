package marketintelligence

import (
	"context"
	"encoding/json"
)

const (
	FreshnessFresh   = "fresh"
	FreshnessStale   = "stale"
	FreshnessUnknown = "unknown"

	SeverityInfo     = "info"
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

const (
	EventTypeAnnouncement = "announcement"
	EventTypeAlphaListing = "alpha_listing"
	EventTypeNews         = "news"
	EventTypeSignal       = "signal"

	FactTypeFunding      = "funding"
	FactTypeOpenInterest = "open_interest"
	FactTypeTaker        = "taker"
	FactTypeDepth        = "depth"
	FactTypeLiquidation  = "liquidation"
)

type EventInput struct {
	DedupKey   string          `json:"dedup_key,omitempty"`
	Type       string          `json:"type"`
	Category   string          `json:"category"`
	Symbols    []string        `json:"symbols"`
	EventTime  int64           `json:"event_time"`
	ObservedAt int64           `json:"observed_at,omitempty"`
	Source     string          `json:"source"`
	SourceRef  string          `json:"source_ref,omitempty"`
	Headline   string          `json:"headline,omitempty"`
	Summary    string          `json:"summary,omitempty"`
	Severity   string          `json:"severity,omitempty"`
	Confidence float64         `json:"confidence,omitempty"`
	RawRef     string          `json:"raw_ref,omitempty"`
	Raw        json.RawMessage `json:"raw,omitempty"`
}

type FactInput struct {
	DedupKey   string          `json:"dedup_key,omitempty"`
	Type       string          `json:"type"`
	Category   string          `json:"category"`
	Symbol     string          `json:"symbol"`
	EventTime  int64           `json:"event_time"`
	ObservedAt int64           `json:"observed_at,omitempty"`
	Source     string          `json:"source"`
	SourceRef  string          `json:"source_ref,omitempty"`
	Severity   string          `json:"severity,omitempty"`
	Confidence float64         `json:"confidence,omitempty"`
	RawRef     string          `json:"raw_ref,omitempty"`
	Data       json.RawMessage `json:"data"`
}

type EventSource struct {
	Source     string `json:"source"`
	SourceRef  string `json:"source_ref,omitempty"`
	RawRef     string `json:"raw_ref,omitempty"`
	ObservedAt int64  `json:"observed_at"`
}

type Event struct {
	ID          int64         `json:"id"`
	EventKey    string        `json:"event_key"`
	Type        string        `json:"type"`
	Category    string        `json:"category"`
	Symbols     []string      `json:"symbols"`
	EventTime   int64         `json:"event_time"`
	ObservedAt  int64         `json:"observed_at"`
	Source      string        `json:"source"`
	SourceRef   string        `json:"source_ref,omitempty"`
	Headline    string        `json:"headline,omitempty"`
	Summary     string        `json:"summary,omitempty"`
	Severity    string        `json:"severity"`
	Confidence  float64       `json:"confidence"`
	Freshness   string        `json:"freshness"`
	FreshnessMs int64         `json:"freshness_age_ms"`
	RawRef      string        `json:"raw_ref,omitempty"`
	Sources     []EventSource `json:"sources"`
}

type Fact struct {
	ID          int64           `json:"id"`
	FactKey     string          `json:"fact_key"`
	Type        string          `json:"type"`
	Category    string          `json:"category"`
	Symbol      string          `json:"symbol"`
	EventTime   int64           `json:"event_time"`
	ObservedAt  int64           `json:"observed_at"`
	Source      string          `json:"source"`
	SourceRef   string          `json:"source_ref,omitempty"`
	Severity    string          `json:"severity"`
	Confidence  float64         `json:"confidence"`
	Freshness   string          `json:"freshness"`
	FreshnessMs int64           `json:"freshness_age_ms"`
	RawRef      string          `json:"raw_ref,omitempty"`
	Data        json.RawMessage `json:"data"`
}

type ListOptions struct {
	Symbol         string
	Type           string
	Category       string
	StartTime      int64
	EndTime        int64
	ObservedBefore int64
	Limit          int
}

type Snapshot struct {
	Symbol      string         `json:"symbol"`
	AsOf        string         `json:"as_of"`
	WindowStart int64          `json:"window_start"`
	Events      []Event        `json:"events"`
	Facts       []Fact         `json:"facts"`
	Sources     []SourceStatus `json:"sources"`
	DataMissing []string       `json:"data_missing"`
}

type SourceStatus struct {
	Source        string `json:"source"`
	Status        string `json:"status"`
	LastSuccessAt int64  `json:"last_success_at,omitempty"`
	LastErrorAt   int64  `json:"last_error_at,omitempty"`
	LastError     string `json:"last_error,omitempty"`
}

type Provider interface {
	Name() string
	Fetch(context.Context, int64) ([]EventInput, error)
}

type ProviderSyncResult struct {
	Source   string `json:"source"`
	Fetched  int    `json:"fetched"`
	Inserted int    `json:"inserted"`
	Merged   int    `json:"merged"`
	Error    string `json:"error,omitempty"`
}
