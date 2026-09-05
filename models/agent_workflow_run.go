package models

// AgentWorkflowRun is the parent lifecycle record for V2-11 business workflows.
type AgentWorkflowRun struct {
	ID               string `orm:"column(id);pk;size(64)" json:"id"`
	Workflow         string `orm:"column(workflow);size(64);index" json:"workflow"`
	SchemaVersion    string `orm:"column(schema_version);size(96)" json:"schema_version"`
	Status           string `orm:"column(status);size(32);index" json:"status"`
	Stage            string `orm:"column(stage);size(64);index" json:"stage"`
	InputJSON        string `orm:"column(input_json);type(text)" json:"-"`
	ResultJSON       string `orm:"column(result_json);type(text);null" json:"-"`
	Error            string `orm:"column(error);type(text);null" json:"error,omitempty"`
	ChildTaskIDsJSON string `orm:"column(child_task_ids_json);type(text);null" json:"-"`
	CreatedAt        int64  `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt        int64  `orm:"column(updated_at);index" json:"updated_at"`
	CompletedAt      int64  `orm:"column(completed_at);index" json:"completed_at,omitempty"`
}

func (*AgentWorkflowRun) TableName() string { return "agent_workflow_runs" }
