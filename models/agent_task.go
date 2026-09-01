package models

type AgentTask struct {
	ID                    string `orm:"column(id);pk;size(64)" json:"id"`
	Skill                 string `orm:"column(skill);size(64);index" json:"skill"`
	ConversationID        string `orm:"column(conversation_id);size(64);index" json:"conversation_id"`
	Status                string `orm:"column(status);size(32);index" json:"status"`
	Stage                 string `orm:"column(stage);size(64)" json:"stage"`
	Progress              int    `orm:"column(progress)" json:"progress"`
	InputJSON             string `orm:"column(input_json);type(text)" json:"-"`
	ResultJSON            string `orm:"column(result_json);type(text)" json:"-"`
	Error                 string `orm:"column(error);type(text)" json:"error"`
	Round                 int    `orm:"column(round)" json:"round"`
	MaxRounds             int    `orm:"column(max_rounds)" json:"max_rounds"`
	Provider              string `orm:"column(provider);size(64)" json:"provider"`
	Model                 string `orm:"column(model);size(128)" json:"model"`
	RuntimeVersion        string `orm:"column(runtime_version);size(64)" json:"runtime_version"`
	SkillVersion          string `orm:"column(skill_version);size(64)" json:"skill_version"`
	PromptVersion         string `orm:"column(prompt_version);size(64)" json:"prompt_version"`
	PromptHash            string `orm:"column(prompt_hash);size(64)" json:"prompt_hash"`
	ModelConfigID         int64  `orm:"column(model_config_id);index" json:"model_config_id"`
	InputContractVersion  string `orm:"column(input_contract_version);size(96)" json:"input_contract_version"`
	OutputContractVersion string `orm:"column(output_contract_version);size(96)" json:"output_contract_version"`
	SkillSource           string `orm:"column(skill_source);size(32);default(native)" json:"skill_source"`
	SkillSourceVersion    string `orm:"column(skill_source_version);size(128)" json:"skill_source_version"`
	InputTokens           int    `orm:"column(input_tokens)" json:"input_tokens"`
	OutputTokens          int    `orm:"column(output_tokens)" json:"output_tokens"`
	TotalTokens           int    `orm:"column(total_tokens)" json:"total_tokens"`
	CreatedAt             int64  `orm:"column(created_at);index" json:"created_at"`
	StartedAt             int64  `orm:"column(started_at);index" json:"started_at"`
	UpdatedAt             int64  `orm:"column(updated_at);index" json:"updated_at"`
	CompletedAt           int64  `orm:"column(completed_at);index" json:"completed_at"`
}

func (*AgentTask) TableName() string { return "agent_tasks" }

type AgentTaskEvent struct {
	ID         int64  `orm:"column(id);auto" json:"id"`
	TaskID     string `orm:"column(task_id);size(64);index" json:"task_id"`
	Sequence   int    `orm:"column(sequence);index" json:"sequence"`
	Stage      string `orm:"column(stage);size(64);index" json:"stage"`
	Progress   int    `orm:"column(progress)" json:"progress"`
	Round      int    `orm:"column(round)" json:"round"`
	Message    string `orm:"column(message);type(text)" json:"message"`
	Skill      string `orm:"column(skill);size(64)" json:"skill"`
	Tool       string `orm:"column(tool);size(128)" json:"tool"`
	Status     string `orm:"column(status);size(32)" json:"status"`
	DurationMs int64  `orm:"column(duration_ms)" json:"duration_ms"`
	EventTime  int64  `orm:"column(event_time);index" json:"event_time"`
}

func (*AgentTaskEvent) TableName() string { return "agent_task_events" }
