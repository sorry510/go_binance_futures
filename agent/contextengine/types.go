package contextengine

import (
	"context"
	"time"
)

type BlockType string

const (
	BlockSystem           BlockType = "system"
	BlockTask             BlockType = "task"
	BlockMarket           BlockType = "market"
	BlockHistory          BlockType = "history"
	BlockMemory           BlockType = "memory"
	BlockTool             BlockType = "tool"
	BlockSkillInstruction BlockType = "skill_instruction"
	BlockMCPResource      BlockType = "mcp_resource"
)

type Freshness string

const (
	FreshnessFresh   Freshness = "fresh"
	FreshnessStale   Freshness = "stale"
	FreshnessMissing Freshness = "missing"
	FreshnessUnknown Freshness = "unknown"
)

type ContextBlock struct {
	ID              string    `json:"id"`
	Type            BlockType `json:"type"`
	Source          string    `json:"source"`
	Role            string    `json:"role,omitempty"`
	AsOf            string    `json:"as_of,omitempty"`
	Priority        int       `json:"priority"`
	EstimatedTokens int       `json:"estimated_tokens"`
	Freshness       Freshness `json:"freshness"`
	Sensitive       bool      `json:"sensitive,omitempty"`
	Required        bool      `json:"required,omitempty"`
	Content         string    `json:"content"`
	ContentHash     string    `json:"content_hash"`
	EvidenceIDs     []string  `json:"evidence_ids,omitempty"`
	Order           int       `json:"order"`
}

type Evidence struct {
	ID           string            `json:"id"`
	SourceType   string            `json:"source_type"`
	Source       string            `json:"source"`
	AsOf         string            `json:"as_of,omitempty"`
	ObservedAt   string            `json:"observed_at"`
	ContentHash  string            `json:"content_hash"`
	Freshness    Freshness         `json:"freshness"`
	FreshnessAge int64             `json:"freshness_age_ms,omitempty"`
	StaleReason  string            `json:"stale_reason,omitempty"`
	KeyFields    map[string]string `json:"key_fields,omitempty"`
	DataMissing  []string          `json:"data_missing,omitempty"`
}

type TrimRecord struct {
	BlockID         string    `json:"block_id"`
	Type            BlockType `json:"type"`
	Source          string    `json:"source"`
	EstimatedTokens int       `json:"estimated_tokens"`
	Reason          string    `json:"reason"`
}

type BuildTrace struct {
	BudgetTokens     int          `json:"budget_tokens"`
	BudgetBytes      int          `json:"budget_bytes"`
	SystemTokens     int          `json:"system_tokens"`
	SelectedTokens   int          `json:"selected_tokens"`
	SelectedBytes    int          `json:"selected_bytes"`
	InputBlocks      int          `json:"input_blocks"`
	SelectedBlocks   int          `json:"selected_blocks"`
	TrimmedBlocks    int          `json:"trimmed_blocks"`
	SelectedBlockIDs []string     `json:"selected_block_ids,omitempty"`
	Trimmed          []TrimRecord `json:"trimmed,omitempty"`
	StaleEvidenceIDs []string     `json:"stale_evidence_ids,omitempty"`
	BuiltAt          string       `json:"built_at"`
}

type Disclosure string

const (
	DisclosureActivation Disclosure = "activation"
	DisclosureOnDemand   Disclosure = "on_demand"
)

type Resource struct {
	ID         string
	Type       BlockType
	Source     string
	Priority   int
	Sensitive  bool
	Disclosure Disclosure
	Load       func(context.Context) (string, error)
}

type BuildOptions struct {
	SystemPrompt string
	Blocks       []ContextBlock
	MaxTokens    int
	MaxBytes     int
	Now          time.Time
}
