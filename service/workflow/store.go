package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/agent/security"
	"go_binance_futures/models"
)

type Run struct {
	ID            string          `json:"id"`
	Workflow      string          `json:"workflow"`
	SchemaVersion string          `json:"schema_version"`
	Status        string          `json:"status"`
	Stage         string          `json:"stage"`
	Input         json.RawMessage `json:"input,omitempty"`
	Result        json.RawMessage `json:"result,omitempty"`
	Error         string          `json:"error,omitempty"`
	ChildTaskIDs  []string        `json:"child_task_ids"`
	CreatedAt     int64           `json:"created_at"`
	UpdatedAt     int64           `json:"updated_at"`
	CompletedAt   int64           `json:"completed_at,omitempty"`
}

type ListOptions struct {
	Workflow, Status string
	Page, Limit      int
}
type ListResult struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
	List  []Run `json:"list"`
}
type Store struct{ Alias string }

func (s Store) orm() orm.Ormer {
	if strings.TrimSpace(s.Alias) != "" {
		return orm.NewOrmUsingDB(s.Alias)
	}
	return orm.NewOrm()
}
func (s Store) Save(ctx context.Context, run Run) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(run.ID) == "" {
		return fmt.Errorf("workflow run id is required")
	}
	row := toModel(run)
	o := s.orm()
	var existing models.AgentWorkflowRun
	err := o.QueryTable(new(models.AgentWorkflowRun)).Filter("id", run.ID).One(&existing)
	if err == orm.ErrNoRows {
		_, err = o.Insert(&row)
		return err
	}
	if err != nil {
		return err
	}
	_, err = o.Update(&row)
	return err
}
func (s Store) Get(ctx context.Context, id string) (Run, error) {
	if err := ctx.Err(); err != nil {
		return Run{}, err
	}
	var row models.AgentWorkflowRun
	err := s.orm().QueryTable(new(models.AgentWorkflowRun)).Filter("id", strings.TrimSpace(id)).One(&row)
	if err != nil {
		return Run{}, err
	}
	return fromModel(row), nil
}
func (s Store) List(ctx context.Context, opt ListOptions) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	if opt.Page < 1 {
		opt.Page = 1
	}
	if opt.Limit < 1 {
		opt.Limit = 20
	}
	if opt.Limit > 100 {
		opt.Limit = 100
	}
	q := s.orm().QueryTable(new(models.AgentWorkflowRun))
	if v := strings.TrimSpace(opt.Workflow); v != "" {
		q = q.Filter("workflow", v)
	}
	if v := strings.TrimSpace(opt.Status); v != "" {
		q = q.Filter("status", v)
	}
	total, err := q.Count()
	if err != nil {
		return ListResult{}, err
	}
	var rows []models.AgentWorkflowRun
	_, err = q.OrderBy("-created_at").Limit(opt.Limit, (opt.Page-1)*opt.Limit).All(&rows)
	if err != nil {
		return ListResult{}, err
	}
	out := make([]Run, 0, len(rows))
	for _, r := range rows {
		out = append(out, fromModel(r))
	}
	return ListResult{Page: opt.Page, Limit: opt.Limit, Total: total, List: out}, nil
}
func toModel(r Run) models.AgentWorkflowRun {
	ids, _ := json.Marshal(r.ChildTaskIDs)
	return models.AgentWorkflowRun{ID: r.ID, Workflow: r.Workflow, SchemaVersion: r.SchemaVersion, Status: r.Status, Stage: r.Stage, InputJSON: security.RedactText(string(r.Input)), ResultJSON: security.RedactText(string(r.Result)), Error: security.RedactText(r.Error), ChildTaskIDsJSON: string(ids), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, CompletedAt: r.CompletedAt}
}
func fromModel(r models.AgentWorkflowRun) Run {
	out := Run{ID: r.ID, Workflow: r.Workflow, SchemaVersion: r.SchemaVersion, Status: r.Status, Stage: r.Stage, Input: json.RawMessage(r.InputJSON), Result: json.RawMessage(r.ResultJSON), Error: r.Error, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, CompletedAt: r.CompletedAt, ChildTaskIDs: []string{}}
	_ = json.Unmarshal([]byte(r.ChildTaskIDsJSON), &out.ChildTaskIDs)
	return out
}
func newRunID() string { return fmt.Sprintf("wf_%d", time.Now().UTC().UnixNano()) }
