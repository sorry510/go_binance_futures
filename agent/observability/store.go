package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/security"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type TraceListOptions struct {
	TaskID     string
	Skill      string
	Type       string
	Status     string
	ToolSource string
	StartTime  int64
	EndTime    int64
	Page       int
	Limit      int
}

type TraceListResult struct {
	Page  int                       `json:"page"`
	Limit int                       `json:"limit"`
	Total int64                     `json:"total"`
	List  []models.AgentObservation `json:"list"`
}

type ChangeListOptions struct {
	Category   string
	EntityType string
	EntityName string
	ChangeType string
	Status     string
	StartTime  int64
	EndTime    int64
	Page       int
	Limit      int
}

type ChangeListResult struct {
	Page  int                       `json:"page"`
	Limit int                       `json:"limit"`
	Total int64                     `json:"total"`
	List  []models.AgentChangeEvent `json:"list"`
}

type Store struct{ Alias string }

func (s Store) orm() orm.Ormer {
	if strings.TrimSpace(s.Alias) != "" {
		return orm.NewOrmUsingDB(s.Alias)
	}
	return orm.NewOrm()
}

func (s Store) InsertObservation(ctx context.Context, value agentruntime.Observation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	row := observationModel(value)
	_, err := s.orm().Insert(&row)
	return err
}

func observationModel(value agentruntime.Observation) models.AgentObservation {
	return models.AgentObservation{
		TaskID: strings.TrimSpace(value.TaskID), ConversationID: strings.TrimSpace(value.ConversationID), Type: strings.TrimSpace(value.Type),
		StepID: strings.TrimSpace(value.StepID), StepType: strings.TrimSpace(value.StepType), Skill: strings.TrimSpace(value.Skill),
		Provider: strings.TrimSpace(value.Provider), Model: strings.TrimSpace(value.Model), Tool: strings.TrimSpace(value.Tool), ToolSource: strings.TrimSpace(value.ToolSource),
		ProviderRef: security.RedactText(strings.TrimSpace(value.ProviderRef)), ProtocolVersion: strings.TrimSpace(value.ProtocolVersion), CatalogHash: strings.TrimSpace(value.CatalogHash), SchemaHash: strings.TrimSpace(value.SchemaHash),
		Status: strings.TrimSpace(value.Status), ErrorType: strings.TrimSpace(value.ErrorType), Error: security.RedactText(value.Error), Round: value.Round, DurationMs: value.DurationMs,
		CacheHit: value.CacheHit, Partial: value.Partial, RawSize: value.RawSize, ContentHash: strings.TrimSpace(value.ContentHash),
		InputTokens: value.Usage.InputTokens, OutputTokens: value.Usage.OutputTokens, TotalTokens: value.Usage.TotalTokens,
		ContextTokens: value.ContextTokens, ContextBlocks: value.ContextBlocks, TrimmedBlocks: value.TrimmedBlocks,
		MemorySelected: value.MemorySelected, MemoryTrimmed: value.MemoryTrimmed, EvidenceCount: value.EvidenceCount, EvalCase: strings.TrimSpace(value.EvalCase), EvalScore: value.EvalScore, CreatedAt: time.Now().UTC().UnixMilli(),
	}
}

func persistObservation(ctx context.Context, value agentruntime.Observation) {
	defer func() { recover() }()
	_ = (Store{}).InsertObservation(ctx, value)
}

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit
}

func (s Store) ListTraces(ctx context.Context, options TraceListOptions) (TraceListResult, error) {
	if err := ctx.Err(); err != nil {
		return TraceListResult{}, err
	}
	page, limit := normalizePage(options.Page, options.Limit)
	q := s.orm().QueryTable(new(models.AgentObservation))
	if v := strings.TrimSpace(options.TaskID); v != "" {
		q = q.Filter("task_id", v)
	}
	if v := strings.TrimSpace(options.Skill); v != "" {
		q = q.Filter("skill", v)
	}
	if v := strings.TrimSpace(options.Type); v != "" {
		q = q.Filter("type", v)
	}
	if v := strings.TrimSpace(options.Status); v != "" {
		q = q.Filter("status", v)
	}
	if v := strings.TrimSpace(options.ToolSource); v != "" {
		q = q.Filter("tool_source", v)
	}
	if options.StartTime > 0 {
		q = q.Filter("created_at__gte", options.StartTime)
	}
	if options.EndTime > 0 {
		q = q.Filter("created_at__lte", options.EndTime)
	}
	total, err := q.Count()
	if err != nil {
		return TraceListResult{}, err
	}
	var rows []models.AgentObservation
	if _, err := q.OrderBy("-created_at", "-id").Limit(limit, (page-1)*limit).All(&rows); err != nil {
		return TraceListResult{}, err
	}
	return TraceListResult{Page: page, Limit: limit, Total: total, List: rows}, nil
}

func (s Store) RecordChange(ctx context.Context, event ChangeEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	detail := event.Detail
	if detail == nil {
		detail = map[string]any{}
	}
	raw, _ := json.Marshal(detail)
	row := models.AgentChangeEvent{Category: strings.TrimSpace(event.Category), EntityType: strings.TrimSpace(event.EntityType), EntityID: event.EntityID,
		EntityName: security.RedactText(strings.TrimSpace(event.EntityName)), ChangeType: strings.TrimSpace(event.ChangeType), FromVersion: strings.TrimSpace(event.FromVersion), ToVersion: strings.TrimSpace(event.ToVersion),
		BeforeHash: strings.TrimSpace(event.BeforeHash), AfterHash: strings.TrimSpace(event.AfterHash), Status: strings.TrimSpace(event.Status), DetailJSON: security.RedactText(string(raw)), CreatedAt: time.Now().UTC().UnixMilli()}
	if row.Status == "" {
		row.Status = "success"
	}
	_, err := s.orm().Insert(&row)
	return err
}

func (s Store) ListChanges(ctx context.Context, options ChangeListOptions) (ChangeListResult, error) {
	if err := ctx.Err(); err != nil {
		return ChangeListResult{}, err
	}
	page, limit := normalizePage(options.Page, options.Limit)
	q := s.orm().QueryTable(new(models.AgentChangeEvent))
	if v := strings.TrimSpace(options.Category); v != "" {
		q = q.Filter("category", v)
	}
	if v := strings.TrimSpace(options.EntityType); v != "" {
		q = q.Filter("entity_type", v)
	}
	if v := strings.TrimSpace(options.EntityName); v != "" {
		q = q.Filter("entity_name__icontains", v)
	}
	if v := strings.TrimSpace(options.ChangeType); v != "" {
		q = q.Filter("change_type", v)
	}
	if v := strings.TrimSpace(options.Status); v != "" {
		q = q.Filter("status", v)
	}
	if options.StartTime > 0 {
		q = q.Filter("created_at__gte", options.StartTime)
	}
	if options.EndTime > 0 {
		q = q.Filter("created_at__lte", options.EndTime)
	}
	total, err := q.Count()
	if err != nil {
		return ChangeListResult{}, err
	}
	var rows []models.AgentChangeEvent
	if _, err := q.OrderBy("-created_at", "-id").Limit(limit, (page-1)*limit).All(&rows); err != nil {
		return ChangeListResult{}, err
	}
	return ChangeListResult{Page: page, Limit: limit, Total: total, List: rows}, nil
}

type ChangeEvent struct {
	Category    string
	EntityType  string
	EntityID    int64
	EntityName  string
	ChangeType  string
	FromVersion string
	ToVersion   string
	BeforeHash  string
	AfterHash   string
	Status      string
	Detail      map[string]any
}

func RecordChange(ctx context.Context, event ChangeEvent) {
	defer func() { recover() }()
	_ = (Store{}).RecordChange(ctx, event)
}

func RecordEval(ctx context.Context, caseName, skill, status string, score float64, durationMs int64, err string) {
	persistObservation(ctx, agentruntime.Observation{Type: "eval", Skill: strings.TrimSpace(skill), Status: strings.TrimSpace(status), Error: err, EvalCase: strings.TrimSpace(caseName), EvalScore: score, DurationMs: durationMs})
}

func percentile64(values []int64, ratio float64) int64 {
	if len(values) == 0 {
		return 0
	}
	copyValues := append([]int64(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i] < copyValues[j] })
	idx := int(float64(len(copyValues)-1) * ratio)
	return copyValues[idx]
}

func safeRatio(num, denom int64) float64 {
	if denom <= 0 {
		return 0
	}
	return float64(num) / float64(denom)
}

func validateWindow(start, end int64) error {
	if start > 0 && end > 0 && start > end {
		return fmt.Errorf("start_time must be <= end_time")
	}
	return nil
}
