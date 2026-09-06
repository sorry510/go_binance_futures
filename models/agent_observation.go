package models

type AgentObservation struct {
	ID              int64   `orm:"column(id);auto" json:"id"`
	TaskID          string  `orm:"column(task_id);size(64);index" json:"task_id"`
	ConversationID  string  `orm:"column(conversation_id);size(64);null;index" json:"conversation_id,omitempty"`
	Type            string  `orm:"column(type);size(32);index" json:"type"`
	StepID          string  `orm:"column(step_id);size(64);null;index" json:"step_id,omitempty"`
	StepType        string  `orm:"column(step_type);size(32);null;index" json:"step_type,omitempty"`
	Skill           string  `orm:"column(skill);size(96);index" json:"skill"`
	Provider        string  `orm:"column(provider);size(64);null;index" json:"provider,omitempty"`
	Model           string  `orm:"column(model);size(191);null;index" json:"model,omitempty"`
	Tool            string  `orm:"column(tool);size(191);null;index" json:"tool,omitempty"`
	ToolSource      string  `orm:"column(tool_source);size(16);null;index" json:"tool_source,omitempty"`
	ProviderRef     string  `orm:"column(provider_ref);size(255);null" json:"provider_ref,omitempty"`
	ProtocolVersion string  `orm:"column(protocol_version);size(32);null" json:"protocol_version,omitempty"`
	CatalogHash     string  `orm:"column(catalog_hash);size(64);null" json:"catalog_hash,omitempty"`
	SchemaHash      string  `orm:"column(schema_hash);size(64);null" json:"schema_hash,omitempty"`
	Status          string  `orm:"column(status);size(32);null;index" json:"status,omitempty"`
	ErrorType       string  `orm:"column(error_type);size(64);null;index" json:"error_type,omitempty"`
	Error           string  `orm:"column(error);type(text);null" json:"error,omitempty"`
	Round           int     `orm:"column(round);default(0)" json:"round,omitempty"`
	DurationMs      int64   `orm:"column(duration_ms);default(0)" json:"duration_ms,omitempty"`
	CacheHit        bool    `orm:"column(cache_hit);default(false);index" json:"cache_hit,omitempty"`
	Partial         bool    `orm:"column(partial);default(false);index" json:"partial,omitempty"`
	RawSize         int     `orm:"column(raw_size);default(0)" json:"raw_size,omitempty"`
	ContentHash     string  `orm:"column(content_hash);size(64);null" json:"content_hash,omitempty"`
	InputTokens     int     `orm:"column(input_tokens);default(0)" json:"input_tokens,omitempty"`
	OutputTokens    int     `orm:"column(output_tokens);default(0)" json:"output_tokens,omitempty"`
	TotalTokens     int     `orm:"column(total_tokens);default(0)" json:"total_tokens,omitempty"`
	ContextTokens   int     `orm:"column(context_tokens);default(0)" json:"context_tokens,omitempty"`
	ContextBlocks   int     `orm:"column(context_blocks);default(0)" json:"context_blocks,omitempty"`
	TrimmedBlocks   int     `orm:"column(trimmed_blocks);default(0)" json:"trimmed_blocks,omitempty"`
	MemorySelected  int     `orm:"column(memory_selected);default(0)" json:"memory_selected,omitempty"`
	MemoryTrimmed   int     `orm:"column(memory_trimmed);default(0)" json:"memory_trimmed,omitempty"`
	EvidenceCount   int     `orm:"column(evidence_count);default(0)" json:"evidence_count,omitempty"`
	EvalCase        string  `orm:"column(eval_case);size(191);null;index" json:"eval_case,omitempty"`
	EvalScore       float64 `orm:"column(eval_score);digits(8);decimals(3);default(0)" json:"eval_score,omitempty"`
	CreatedAt       int64   `orm:"column(created_at);index" json:"created_at"`
}

func (*AgentObservation) TableName() string { return "agent_observations" }
