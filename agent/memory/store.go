package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/security"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type ORMStore struct{ Alias string }

func NewORMStore() *ORMStore { return &ORMStore{} }

func (store *ORMStore) Create(ctx context.Context, input CreateInput) (Memory, error) {
	if err := ctx.Err(); err != nil {
		return Memory{}, err
	}
	input, err := normalizeCreate(input, time.Now().UTC())
	if err != nil {
		return Memory{}, err
	}
	now := time.Now().UTC()
	storedContent := security.RedactText(input.Content)
	row := models.AgentMemory{
		Type: string(input.Type), ScopeUser: input.Scope.User, ScopeSkill: input.Scope.Skill,
		ScopeSymbol: input.Scope.Symbol, ScopeStrategy: input.Scope.Strategy,
		SourceTaskID: strings.TrimSpace(input.SourceTaskID), Confidence: input.Confidence,
		Status: input.Status, Content: storedContent, ContentHash: hashContent(storedContent),
		CreatedAt: now.UnixMilli(), UpdatedAt: now.UnixMilli(), ExpiresAt: input.ExpiresAt,
	}
	if _, err := store.orm().Insert(&row); err != nil {
		return Memory{}, fmt.Errorf("insert agent memory: %w", err)
	}
	return fromModel(row), nil
}

func (store *ORMStore) Update(ctx context.Context, id int64, input UpdateInput) (Memory, error) {
	row, err := store.getModel(ctx, id)
	if err != nil {
		return Memory{}, err
	}
	input.Scope = normalizeScope(input.Scope)
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" {
		return Memory{}, fmt.Errorf("memory content is required")
	}
	if input.Confidence < 0 || input.Confidence > 1 {
		return Memory{}, fmt.Errorf("memory confidence must be between 0 and 1")
	}
	if Type(row.Type) == TypeMarketHypothesis && input.ExpiresAt <= time.Now().UTC().UnixMilli() {
		return Memory{}, fmt.Errorf("market_hypothesis requires a future expires_at")
	}
	row.ScopeUser, row.ScopeSkill = input.Scope.User, input.Scope.Skill
	row.ScopeSymbol, row.ScopeStrategy = input.Scope.Symbol, input.Scope.Strategy
	storedContent := security.RedactText(input.Content)
	row.Confidence, row.Content = input.Confidence, storedContent
	row.ContentHash, row.ExpiresAt, row.UpdatedAt = hashContent(storedContent), input.ExpiresAt, time.Now().UTC().UnixMilli()
	if _, err := store.orm().Update(&row); err != nil {
		return Memory{}, fmt.Errorf("update agent memory: %w", err)
	}
	return fromModel(row), nil
}

func (store *ORMStore) Get(ctx context.Context, id int64) (Memory, error) {
	row, err := store.getModel(ctx, id)
	if err != nil {
		return Memory{}, err
	}
	return fromModel(row), nil
}

func (store *ORMStore) SetStatus(ctx context.Context, id int64, status string) (Memory, error) {
	if status != StatusActive && status != StatusDisabled && status != StatusCandidate {
		return Memory{}, fmt.Errorf("unsupported memory status %q", status)
	}
	row, err := store.getModel(ctx, id)
	if err != nil {
		return Memory{}, err
	}
	if status == StatusActive && row.ExpiresAt > 0 && row.ExpiresAt <= time.Now().UTC().UnixMilli() {
		return Memory{}, fmt.Errorf("expired memory cannot be enabled")
	}
	row.Status, row.UpdatedAt = status, time.Now().UTC().UnixMilli()
	if _, err := store.orm().Update(&row, "Status", "UpdatedAt"); err != nil {
		return Memory{}, err
	}
	return fromModel(row), nil
}

func (store *ORMStore) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	count, err := store.orm().QueryTable(new(models.AgentMemory)).Filter("id", id).Delete()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("memory %d not found", id)
	}
	return nil
}

func (store *ORMStore) List(ctx context.Context, options ListOptions) (ListResult, error) {
	if err := store.Expire(ctx, time.Now().UTC()); err != nil {
		return ListResult{}, err
	}
	page, limit := normalizePage(options.Page, options.Limit)
	query := store.orm().QueryTable(new(models.AgentMemory))
	if value := strings.TrimSpace(options.Type); value != "" {
		query = query.Filter("type", value)
	}
	if value := strings.TrimSpace(options.Status); value != "" {
		query = query.Filter("status", value)
	}
	if value := strings.TrimSpace(options.User); value != "" {
		query = query.Filter("scope_user", value)
	}
	if value := strings.TrimSpace(options.Skill); value != "" {
		query = query.Filter("scope_skill", value)
	}
	if value := strings.ToUpper(strings.TrimSpace(options.Symbol)); value != "" {
		query = query.Filter("scope_symbol", value)
	}
	if value := strings.TrimSpace(options.Strategy); value != "" {
		query = query.Filter("scope_strategy", value)
	}
	if value := strings.TrimSpace(options.SourceTaskID); value != "" {
		query = query.Filter("source_task_id", value)
	}
	if !options.IncludeExpired {
		query = query.Exclude("status", StatusExpired)
	}
	total, err := query.Count()
	if err != nil {
		return ListResult{}, err
	}
	var rows []models.AgentMemory
	if _, err := query.OrderBy("-updated_at", "-id").Limit(limit, (page-1)*limit).All(&rows); err != nil {
		return ListResult{}, err
	}
	result := make([]Memory, 0, len(rows))
	for _, row := range rows {
		result = append(result, fromModel(row))
	}
	return ListResult{Page: page, Limit: limit, Total: total, List: result}, nil
}

func (store *ORMStore) Query(ctx context.Context, scope QueryScope) ([]Memory, error) {
	now := scope.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := store.Expire(ctx, now); err != nil {
		return nil, err
	}
	limit := scope.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	scope.User = strings.TrimSpace(scope.User)
	if scope.User == "" {
		scope.User = DefaultUserScope
	}
	scope.Skill = strings.TrimSpace(scope.Skill)
	scope.Symbol = strings.ToUpper(strings.TrimSpace(scope.Symbol))
	scope.Strategy = strings.TrimSpace(scope.Strategy)
	query := store.orm().QueryTable(new(models.AgentMemory)).Filter("status", StatusActive)
	query = query.Filter("scope_user__in", "", scope.User)
	query = query.Filter("scope_skill__in", "", scope.Skill)
	query = query.Filter("scope_symbol__in", "", scope.Symbol)
	query = query.Filter("scope_strategy__in", "", scope.Strategy)
	var rows []models.AgentMemory
	if _, err := query.OrderBy("-updated_at", "-id").Limit(limit).All(&rows); err != nil {
		return nil, err
	}
	result := make([]Memory, 0, limit)
	for _, row := range rows {
		result = append(result, fromModel(row))
	}
	return result, nil
}

func (store *ORMStore) Expire(ctx context.Context, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := store.orm().QueryTable(new(models.AgentMemory)).Filter("status__in", StatusActive, StatusCandidate, StatusDisabled).Filter("expires_at__gt", 0).Filter("expires_at__lte", now.UTC().UnixMilli()).Update(orm.Params{"status": StatusExpired, "updated_at": now.UTC().UnixMilli()})
	return err
}

func (store *ORMStore) ContextBlocks(ctx context.Context, scope QueryScope) ([]contextengine.ContextBlock, error) {
	items, err := store.Query(ctx, scope)
	if err != nil {
		return nil, err
	}
	blocks := make([]contextengine.ContextBlock, 0, len(items))
	for _, item := range items {
		blocks = append(blocks, contextengine.ContextBlock{
			ID: fmt.Sprintf("memory:%d", item.ID), Type: contextengine.BlockMemory, Source: "long_term_memory:" + string(item.Type),
			Role: "user", AsOf: item.UpdatedAt.UTC().Format(time.RFC3339), Priority: contextengine.DefaultPriority(contextengine.BlockMemory),
			Freshness: contextengine.FreshnessFresh, ContentHash: item.ContentHash,
			Content: formatContextMemory(item),
		})
	}
	return blocks, nil
}

func (store *ORMStore) getModel(ctx context.Context, id int64) (models.AgentMemory, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentMemory{}, err
	}
	var row models.AgentMemory
	if err := store.orm().QueryTable(new(models.AgentMemory)).Filter("id", id).One(&row); err != nil {
		if err == orm.ErrNoRows {
			return row, fmt.Errorf("memory %d not found", id)
		}
		return row, err
	}
	return row, nil
}

func (store *ORMStore) orm() orm.Ormer {
	if strings.TrimSpace(store.Alias) != "" {
		return orm.NewOrmUsingDB(store.Alias)
	}
	return orm.NewOrm()
}

func normalizeCreate(input CreateInput, now time.Time) (CreateInput, error) {
	input.Scope = normalizeScope(input.Scope)
	input.SourceTaskID = strings.TrimSpace(input.SourceTaskID)
	input.Content = strings.TrimSpace(input.Content)
	if !validType(input.Type) {
		return input, fmt.Errorf("unsupported memory type %q", input.Type)
	}
	if input.Content == "" {
		return input, fmt.Errorf("memory content is required")
	}
	if input.Confidence < 0 || input.Confidence > 1 {
		return input, fmt.Errorf("memory confidence must be between 0 and 1")
	}
	if input.Status == "" {
		input.Status = StatusActive
	}
	if input.Status != StatusActive && input.Status != StatusCandidate {
		return input, fmt.Errorf("new memory status must be active or candidate")
	}
	if input.Type == TypeUserPreference && input.Scope.User == "" {
		input.Scope.User = DefaultUserScope
	}
	if input.Type == TypeMarketHypothesis {
		if input.ExpiresAt == 0 {
			input.ExpiresAt = now.Add(DefaultMarketHypothesisTTL).UnixMilli()
		}
		if input.ExpiresAt <= now.UnixMilli() {
			return input, fmt.Errorf("market_hypothesis requires a future expires_at")
		}
	}
	return input, nil
}

func validType(value Type) bool {
	switch value {
	case TypeUserPreference, TypeStrategyFact, TypeMarketHypothesis, TypeTaskSummary, TypeLesson:
		return true
	default:
		return false
	}
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

func formatContextMemory(item Memory) string {
	return fmt.Sprintf("LONG_TERM_MEMORY reference_only=true type=%s confidence=%.2f scope_user=%s scope_skill=%s scope_symbol=%s scope_strategy=%s\n%s", item.Type, item.Confidence, item.Scope.User, item.Scope.Skill, item.Scope.Symbol, item.Scope.Strategy, item.Content)
}

func fromModel(row models.AgentMemory) Memory {
	item := Memory{ID: row.ID, Type: Type(row.Type), Scope: Scope{User: row.ScopeUser, Skill: row.ScopeSkill, Symbol: row.ScopeSymbol, Strategy: row.ScopeStrategy}, SourceTaskID: row.SourceTaskID, Confidence: row.Confidence, Status: row.Status, Content: row.Content, ContentHash: row.ContentHash, CreatedAt: time.UnixMilli(row.CreatedAt).UTC(), UpdatedAt: time.UnixMilli(row.UpdatedAt).UTC()}
	if row.ExpiresAt > 0 {
		value := time.UnixMilli(row.ExpiresAt).UTC()
		item.ExpiresAt = &value
	}
	return item
}

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
