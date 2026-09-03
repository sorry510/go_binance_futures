package toolruntime

import (
	"encoding/json"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/permission"
)

type SourceType string

const (
	SourceNative SourceType = "native"
	SourceMCP    SourceType = "mcp"
)

type CachePolicy struct {
	Enabled bool   `json:"enabled"`
	TTLms   int64  `json:"ttl_ms,omitempty"`
	Scope   string `json:"scope,omitempty"`
}

func NewCachePolicy(ttl time.Duration) CachePolicy {
	return CachePolicy{Enabled: ttl > 0, TTLms: ttl.Milliseconds(), Scope: "task"}
}

type ToolDescriptor struct {
	CanonicalName  string               `json:"canonical_name"`
	SourceType     SourceType           `json:"source_type"`
	Description    string               `json:"description,omitempty"`
	InputSchema    json.RawMessage      `json:"input_schema,omitempty"`
	OutputSchema   json.RawMessage      `json:"output_schema,omitempty"`
	Risk           permission.RiskLevel `json:"risk"`
	Idempotent     bool                 `json:"idempotent"`
	TimeoutMs      int64                `json:"timeout_ms,omitempty"`
	CachePolicy    CachePolicy          `json:"cache_policy"`
	ProviderRef    string               `json:"provider_ref,omitempty"`
	MaxResultBytes int                  `json:"max_result_bytes,omitempty"`
}

type ToolResultEnvelope struct {
	Data        any       `json:"data,omitempty"`
	Source      string    `json:"source"`
	AsOf        string    `json:"as_of,omitempty"`
	DurationMs  int64     `json:"duration_ms"`
	CacheHit    bool      `json:"cache_hit"`
	Partial     bool      `json:"partial"`
	Warnings    []string  `json:"warnings,omitempty"`
	ErrorType   ErrorType `json:"error_type,omitempty"`
	RawSize     int       `json:"raw_size"`
	ContentHash string    `json:"content_hash,omitempty"`
}

type Trace struct {
	CanonicalName string               `json:"canonical_name"`
	SourceType    SourceType           `json:"source_type"`
	Risk          permission.RiskLevel `json:"risk"`
	Idempotent    bool                 `json:"idempotent"`
	TimeoutMs     int64                `json:"timeout_ms,omitempty"`
	CacheTTLms    int64                `json:"cache_ttl_ms,omitempty"`
	ArgumentsHash string               `json:"arguments_hash,omitempty"`
	CallIndex     int                  `json:"call_index,omitempty"`
	CallBudget    int                  `json:"call_budget,omitempty"`
	DurationMs    int64                `json:"duration_ms"`
	CacheHit      bool                 `json:"cache_hit"`
	Partial       bool                 `json:"partial"`
	ErrorType     ErrorType            `json:"error_type,omitempty"`
	RawSize       int                  `json:"raw_size"`
	ContentHash   string               `json:"content_hash,omitempty"`
	AsOf          string               `json:"as_of,omitempty"`
	Warnings      []string             `json:"warnings,omitempty"`
}

type ExecuteRequest struct {
	SkillName      string
	AllowedTools   map[string]bool
	ToolName       string
	Arguments      json.RawMessage
	MaxResultBytes int
	CallIndex      int
	CallBudget     int
}

type ExecuteResult struct {
	Descriptor   ToolDescriptor
	Envelope     ToolResultEnvelope
	Trace        Trace
	Value        any
	Raw          json.RawMessage
	Evidence     []contextengine.Evidence
	ContextBlock contextengine.ContextBlock
	ToolError    error
}
