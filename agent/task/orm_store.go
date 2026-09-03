package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/agent/security"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type ORMStore struct {
	Alias string
}

func NewORMStore() *ORMStore {
	return &ORMStore{}
}

func (store *ORMStore) Save(ctx context.Context, item *Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if item == nil || strings.TrimSpace(item.ID) == "" {
		return fmt.Errorf("task store requires a task id")
	}
	o := store.orm()
	row := toModel(item)
	var existing models.AgentTask
	err := o.QueryTable(new(models.AgentTask)).Filter("id", row.ID).One(&existing)
	if err == orm.ErrNoRows {
		if _, err := o.Insert(&row); err != nil {
			return fmt.Errorf("insert agent task: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find agent task: %w", err)
	} else {
		if _, err := o.Update(&row); err != nil {
			return fmt.Errorf("update agent task: %w", err)
		}
	}
	return store.appendMissingEvents(o, item)
}

func (store *ORMStore) Get(ctx context.Context, id string) (*Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("task id is required")
	}
	o := store.orm()
	var row models.AgentTask
	if err := o.QueryTable(new(models.AgentTask)).Filter("id", id).One(&row); err != nil {
		if err == orm.ErrNoRows {
			return nil, fmt.Errorf("task %q not found", id)
		}
		return nil, err
	}
	item := fromModel(row)
	var events []models.AgentTaskEvent
	if _, err := o.QueryTable(new(models.AgentTaskEvent)).Filter("task_id", id).OrderBy("sequence").All(&events); err != nil {
		return nil, fmt.Errorf("load agent task events: %w", err)
	}
	item.Events = make([]Event, 0, len(events))
	for _, event := range events {
		item.Events = append(item.Events, fromEventModel(event))
	}
	return item, nil
}

func (store *ORMStore) List(ctx context.Context, options ListOptions) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	page, limit := normalizeListOptions(options)
	o := store.orm()
	query := o.QueryTable(new(models.AgentTask))
	if skill := strings.TrimSpace(options.Skill); skill != "" {
		query = query.Filter("skill", skill)
	}
	if status := strings.TrimSpace(options.Status); status != "" {
		query = query.Filter("status", status)
	}
	if conversationID := strings.TrimSpace(options.ConversationID); conversationID != "" {
		query = query.Filter("conversation_id", conversationID)
	}
	total, err := query.Count()
	if err != nil {
		return ListResult{}, fmt.Errorf("count agent tasks: %w", err)
	}
	var rows []models.AgentTask
	if _, err := query.OrderBy("-created_at").Limit(limit, (page-1)*limit).All(&rows); err != nil {
		return ListResult{}, fmt.Errorf("list agent tasks: %w", err)
	}
	items := make([]*Task, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromModel(row))
	}
	return ListResult{Page: page, Limit: limit, Total: total, List: items}, nil
}

func (store *ORMStore) MarkInterrupted(ctx context.Context, at time.Time) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	o := store.orm()
	var rows []models.AgentTask
	query := o.QueryTable(new(models.AgentTask)).Filter("status__in",
		string(StatusQueued), string(StatusRunning), string(StatusWaitingLLM), string(StatusWaitingTool), string(StatusValidating))
	if _, err := query.All(&rows); err != nil {
		return 0, fmt.Errorf("list interrupted agent tasks: %w", err)
	}
	for _, row := range rows {
		item, err := store.Get(ctx, row.ID)
		if err != nil {
			return 0, err
		}
		markInterrupted(item, at)
		if err := store.Save(ctx, item); err != nil {
			return 0, err
		}
	}
	return int64(len(rows)), nil
}

func (store *ORMStore) SaveCheckpoint(ctx context.Context, taskID string, checkpoint string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}
	count, err := store.orm().QueryTable(new(models.AgentTask)).Filter("id", taskID).Update(orm.Params{"CheckpointJSON": sanitizePayload(checkpoint)})
	if err != nil {
		return fmt.Errorf("save agent task checkpoint: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("task %q not found", taskID)
	}
	return nil
}

func (store *ORMStore) LoadCheckpoint(ctx context.Context, taskID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var row models.AgentTask
	if err := store.orm().QueryTable(new(models.AgentTask)).Filter("id", strings.TrimSpace(taskID)).One(&row, "CheckpointJSON"); err != nil {
		return "", err
	}
	if strings.TrimSpace(stringValue(row.CheckpointJSON)) == "" {
		return "", fmt.Errorf("task %q has no recovery checkpoint", taskID)
	}
	return stringValue(row.CheckpointJSON), nil
}

func (store *ORMStore) ClearCheckpoint(ctx context.Context, taskID string) error {
	return store.SaveCheckpoint(ctx, taskID, "")
}

func (store *ORMStore) orm() orm.Ormer {
	if strings.TrimSpace(store.Alias) != "" {
		return orm.NewOrmUsingDB(store.Alias)
	}
	return orm.NewOrm()
}

func (store *ORMStore) appendMissingEvents(o orm.Ormer, item *Task) error {
	count, err := o.QueryTable(new(models.AgentTaskEvent)).Filter("task_id", item.ID).Count()
	if err != nil {
		return fmt.Errorf("count agent task events: %w", err)
	}
	if count >= int64(len(item.Events)) {
		return nil
	}
	for index := int(count); index < len(item.Events); index++ {
		row := toEventModel(item.Events[index], index+1)
		if _, err := o.Insert(&row); err != nil {
			return fmt.Errorf("insert agent task event: %w", err)
		}
	}
	return nil
}

func stringPointer(value string) *string {
	copy := value
	return &copy
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func toModel(item *Task) models.AgentTask {
	row := models.AgentTask{
		ID: item.ID, Skill: item.Skill, ConversationID: item.ConversationID, Status: string(item.Status), Stage: item.Stage, Progress: item.Progress,
		InputJSON: sanitizePayload(item.Input), ResultJSON: sanitizePayload(string(item.Result)), Error: sanitizeText(item.Error),
		Round: item.Round, MaxRounds: item.MaxRounds, Provider: item.Provider, Model: item.Model,
		ExecutionMode: item.ExecutionMode, PlanJSON: stringPointer(sanitizePayload(string(item.Plan))), StepsJSON: stringPointer(sanitizePayload(string(item.Steps))), CheckpointJSON: stringPointer(sanitizePayload(item.CheckpointJSON)), ResumeCount: item.ResumeCount,
		RuntimeVersion: item.RuntimeVersion, SkillVersion: item.SkillVersion, PromptVersion: item.PromptVersion,
		PromptHash: item.PromptHash, ModelConfigID: item.ModelConfigID, InputContractVersion: item.InputContractVersion,
		OutputContractVersion: item.OutputContractVersion, SkillSource: item.SkillSource, SkillSourceVersion: item.SkillSourceVersion,
		ToolCatalogHash: item.ToolCatalogHash, SkillPackageHash: item.SkillPackageHash,
		InputTokens: item.Usage.InputTokens, OutputTokens: item.Usage.OutputTokens, TotalTokens: item.Usage.TotalTokens,
		CreatedAt: item.CreatedAt.UnixMilli(), UpdatedAt: item.UpdatedAt.UnixMilli(),
	}
	if item.StartedAt != nil {
		row.StartedAt = item.StartedAt.UnixMilli()
	}
	if item.CompletedAt != nil {
		row.CompletedAt = item.CompletedAt.UnixMilli()
	}
	return row
}

func fromModel(row models.AgentTask) *Task {
	item := &Task{
		ID: row.ID, Skill: row.Skill, ConversationID: row.ConversationID, Status: Status(row.Status), Stage: row.Stage, Progress: row.Progress,
		Input: row.InputJSON, Result: json.RawMessage(row.ResultJSON), Error: row.Error,
		Round: row.Round, MaxRounds: row.MaxRounds, Provider: row.Provider, Model: row.Model,
		ExecutionMode: row.ExecutionMode, Plan: json.RawMessage(stringValue(row.PlanJSON)), Steps: json.RawMessage(stringValue(row.StepsJSON)), CheckpointJSON: stringValue(row.CheckpointJSON), ResumeCount: row.ResumeCount,
		RuntimeVersion: row.RuntimeVersion, SkillVersion: row.SkillVersion, PromptVersion: row.PromptVersion,
		PromptHash: row.PromptHash, ModelConfigID: row.ModelConfigID, InputContractVersion: row.InputContractVersion,
		OutputContractVersion: row.OutputContractVersion, SkillSource: row.SkillSource, SkillSourceVersion: row.SkillSourceVersion,
		ToolCatalogHash: row.ToolCatalogHash, SkillPackageHash: row.SkillPackageHash,
		Usage:     Usage{InputTokens: row.InputTokens, OutputTokens: row.OutputTokens, TotalTokens: row.TotalTokens},
		CreatedAt: time.UnixMilli(row.CreatedAt).UTC(), UpdatedAt: time.UnixMilli(row.UpdatedAt).UTC(),
	}
	if row.StartedAt > 0 {
		value := time.UnixMilli(row.StartedAt).UTC()
		item.StartedAt = &value
	}
	if row.CompletedAt > 0 {
		value := time.UnixMilli(row.CompletedAt).UTC()
		item.CompletedAt = &value
	}
	return item
}

func toEventModel(event Event, sequence int) models.AgentTaskEvent {
	return models.AgentTaskEvent{
		TaskID: event.TaskID, Sequence: sequence, StepID: event.StepID, StepType: event.StepType, Stage: event.Stage, Progress: event.Progress, Round: event.Round,
		Message: sanitizeText(event.Message), Skill: event.Skill, Tool: event.Tool, Status: event.Status, ErrorType: event.ErrorType, Checkpoint: event.Checkpoint,
		DurationMs: event.DurationMs, EventTime: event.Time.UnixMilli(),
	}
}

func fromEventModel(row models.AgentTaskEvent) Event {
	return Event{
		TaskID: row.TaskID, StepID: row.StepID, StepType: row.StepType, Stage: row.Stage, Progress: row.Progress, Round: row.Round, Message: row.Message,
		Skill: row.Skill, Tool: row.Tool, Status: row.Status, ErrorType: row.ErrorType, Checkpoint: row.Checkpoint, DurationMs: row.DurationMs,
		Time: time.UnixMilli(row.EventTime).UTC(),
	}
}

func sanitizeText(value string) string {
	return security.RedactText(value)
}

func sanitizePayload(value string) string {
	return security.RedactPayload(value)
}
