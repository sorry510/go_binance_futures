package marketintelligence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type Service struct {
	Alias string
	Now   func() time.Time
}

func DefaultService() Service { return Service{} }

func (service Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}

func (service Service) ormer() orm.Ormer {
	if strings.TrimSpace(service.Alias) != "" {
		return orm.NewOrmUsingDB(service.Alias)
	}
	return orm.NewOrm()
}

func (service Service) IngestEvent(ctx context.Context, input EventInput) (Event, bool, error) {
	if err := ctx.Err(); err != nil {
		return Event{}, false, err
	}
	input = normalizeEventInput(input, service.now())
	if err := validateEventInput(input); err != nil {
		return Event{}, false, err
	}
	key := eventKey(input)
	o := service.ormer()
	var row models.AgentMarketEvent
	created := false
	err := o.QueryTable(new(models.AgentMarketEvent)).Filter("event_key", key).One(&row)
	if err == orm.ErrNoRows {
		symbolsJSON, _ := json.Marshal(input.Symbols)
		row = models.AgentMarketEvent{
			EventKey: key, Type: input.Type, Category: input.Category, SymbolsJSON: string(symbolsJSON),
			EventTime: input.EventTime, ObservedAt: input.ObservedAt, Source: input.Source, SourceRef: input.SourceRef,
			Headline: input.Headline, Summary: input.Summary, Severity: input.Severity, Confidence: input.Confidence,
			Freshness: freshness(input.Type, input.EventTime, input.ObservedAt), RawRef: input.RawRef, RawJSON: string(input.Raw),
			CreatedAt: input.ObservedAt, UpdatedAt: input.ObservedAt,
		}
		if _, insertErr := o.Insert(&row); insertErr != nil {
			if rereadErr := o.QueryTable(new(models.AgentMarketEvent)).Filter("event_key", key).One(&row); rereadErr != nil {
				return Event{}, false, fmt.Errorf("insert market event: %w", insertErr)
			}
		} else {
			created = true
		}
	} else if err != nil {
		return Event{}, false, fmt.Errorf("load market event: %w", err)
	} else {
		fields := []string{}
		if input.ObservedAt < row.ObservedAt {
			row.ObservedAt = input.ObservedAt
			row.Freshness = freshness(row.Type, row.EventTime, input.ObservedAt)
			fields = append(fields, "ObservedAt", "Freshness")
		}
		if input.ObservedAt > row.UpdatedAt {
			row.UpdatedAt = input.ObservedAt
			fields = append(fields, "UpdatedAt")
		}
		if len(fields) > 0 {
			if _, err := o.Update(&row, fields...); err != nil {
				return Event{}, false, fmt.Errorf("update market event observation: %w", err)
			}
		}
	}
	if err := service.upsertEventSource(ctx, row.ID, key, input); err != nil {
		return Event{}, created, err
	}
	_ = service.RecordSourceSuccess(ctx, input.Source, input.ObservedAt)
	item, convertErr := service.eventFromRow(ctx, row, service.now())
	return item, created, convertErr
}

func (service Service) upsertEventSource(ctx context.Context, eventID int64, eventKeyValue string, input EventInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	o := service.ormer()
	key := hashKey(eventKeyValue + "|" + strings.ToLower(input.Source) + "|" + input.SourceRef + "|" + input.RawRef)
	var row models.AgentMarketEventSource
	err := o.QueryTable(new(models.AgentMarketEventSource)).Filter("source_key", key).One(&row)
	if err == nil {
		if input.ObservedAt < row.ObservedAt {
			row.ObservedAt = input.ObservedAt
			_, err = o.Update(&row, "ObservedAt")
		}
		return err
	}
	if err != orm.ErrNoRows {
		return fmt.Errorf("load market event source: %w", err)
	}
	row = models.AgentMarketEventSource{EventID: eventID, SourceKey: key, Source: input.Source, SourceRef: input.SourceRef, RawRef: input.RawRef, ObservedAt: input.ObservedAt, CreatedAt: input.ObservedAt}
	if _, err := o.Insert(&row); err != nil {
		if o.QueryTable(new(models.AgentMarketEventSource)).Filter("source_key", key).Exist() {
			return nil
		}
		return fmt.Errorf("insert market event source: %w", err)
	}
	return nil
}

func (service Service) IngestFact(ctx context.Context, input FactInput) (Fact, bool, error) {
	if err := ctx.Err(); err != nil {
		return Fact{}, false, err
	}
	input = normalizeFactInput(input, service.now())
	if err := validateFactInput(input); err != nil {
		return Fact{}, false, err
	}
	key := factKey(input)
	o := service.ormer()
	var row models.AgentMarketFact
	err := o.QueryTable(new(models.AgentMarketFact)).Filter("fact_key", key).One(&row)
	created := false
	if err == orm.ErrNoRows {
		row = models.AgentMarketFact{
			FactKey: key, Type: input.Type, Category: input.Category, Symbol: input.Symbol,
			EventTime: input.EventTime, ObservedAt: input.ObservedAt, Source: input.Source, SourceRef: input.SourceRef,
			Severity: input.Severity, Confidence: input.Confidence, Freshness: freshness(input.Type, input.EventTime, input.ObservedAt),
			DataJSON: string(input.Data), RawRef: input.RawRef, CreatedAt: input.ObservedAt,
		}
		if _, insertErr := o.Insert(&row); insertErr != nil {
			if rereadErr := o.QueryTable(new(models.AgentMarketFact)).Filter("fact_key", key).One(&row); rereadErr != nil {
				return Fact{}, false, fmt.Errorf("insert market fact: %w", insertErr)
			}
		} else {
			created = true
		}
	} else if err != nil {
		return Fact{}, false, fmt.Errorf("load market fact: %w", err)
	}
	_ = service.RecordSourceSuccess(ctx, input.Source, input.ObservedAt)
	return factFromRow(row, service.now()), created, nil
}

func (service Service) ListEvents(ctx context.Context, options ListOptions) ([]Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit := normalizeLimit(options.Limit)
	query := service.ormer().QueryTable(new(models.AgentMarketEvent))
	if value := strings.ToUpper(strings.TrimSpace(options.Symbol)); value != "" {
		query = query.Filter("symbols_json__icontains", `"`+value+`"`)
	}
	if value := strings.TrimSpace(options.Type); value != "" {
		query = query.Filter("event_type", value)
	}
	if value := strings.TrimSpace(options.Category); value != "" {
		query = query.Filter("category", value)
	}
	if options.StartTime > 0 {
		query = query.Filter("event_time__gte", options.StartTime)
	}
	if options.EndTime > 0 {
		query = query.Filter("event_time__lte", options.EndTime)
	}
	if options.ObservedBefore > 0 {
		query = query.Filter("observed_at__lte", options.ObservedBefore)
	}
	var rows []models.AgentMarketEvent
	if _, err := query.OrderBy("-event_time", "-observed_at").Limit(limit).All(&rows); err != nil {
		return nil, fmt.Errorf("list market events: %w", err)
	}
	result := make([]Event, 0, len(rows))
	now := service.now()
	for _, row := range rows {
		item, err := service.eventFromRow(ctx, row, now)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (service Service) ListFacts(ctx context.Context, options ListOptions) ([]Fact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit := normalizeLimit(options.Limit)
	query := service.ormer().QueryTable(new(models.AgentMarketFact))
	if value := strings.ToUpper(strings.TrimSpace(options.Symbol)); value != "" {
		query = query.Filter("symbol", value)
	}
	if value := strings.TrimSpace(options.Type); value != "" {
		query = query.Filter("fact_type", value)
	}
	if value := strings.TrimSpace(options.Category); value != "" {
		query = query.Filter("category", value)
	}
	if options.StartTime > 0 {
		query = query.Filter("event_time__gte", options.StartTime)
	}
	if options.EndTime > 0 {
		query = query.Filter("event_time__lte", options.EndTime)
	}
	if options.ObservedBefore > 0 {
		query = query.Filter("observed_at__lte", options.ObservedBefore)
	}
	var rows []models.AgentMarketFact
	if _, err := query.OrderBy("-event_time", "-observed_at").Limit(limit).All(&rows); err != nil {
		return nil, fmt.Errorf("list market facts: %w", err)
	}
	result := make([]Fact, 0, len(rows))
	now := service.now()
	for _, row := range rows {
		result = append(result, factFromRow(row, now))
	}
	return result, nil
}

func (service Service) SourceStatuses(ctx context.Context) ([]SourceStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var rows []models.AgentMarketSourceStatus
	if _, err := service.ormer().QueryTable(new(models.AgentMarketSourceStatus)).OrderBy("source").All(&rows); err != nil {
		return nil, err
	}
	out := make([]SourceStatus, 0, len(rows))
	for _, row := range rows {
		out = append(out, SourceStatus{Source: row.Source, Status: row.Status, LastSuccessAt: row.LastSuccessAt, LastErrorAt: row.LastErrorAt, LastError: row.LastError})
	}
	return out, nil
}

func (service Service) RecordSourceSuccess(ctx context.Context, source string, at int64) error {
	return service.recordSource(ctx, source, "healthy", at, "")
}
func (service Service) RecordSourceFailure(ctx context.Context, source string, at int64, cause error) error {
	message := "provider unavailable"
	if cause != nil {
		message = cause.Error()
	}
	return service.recordSource(ctx, source, "error", at, message)
}
func (service Service) recordSource(ctx context.Context, source, status string, at int64, message string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	if at <= 0 {
		at = service.now().UnixMilli()
	}
	o := service.ormer()
	var row models.AgentMarketSourceStatus
	err := o.QueryTable(new(models.AgentMarketSourceStatus)).Filter("source", source).One(&row)
	if err == orm.ErrNoRows {
		row = models.AgentMarketSourceStatus{Source: source, Status: status, UpdatedAt: at}
		if status == "healthy" {
			row.LastSuccessAt = at
		} else {
			row.LastErrorAt, row.LastError = at, message
		}
		_, err = o.Insert(&row)
		return err
	}
	if err != nil {
		return err
	}
	row.Status, row.UpdatedAt = status, at
	if status == "healthy" {
		row.LastSuccessAt, row.LastError = at, ""
	} else {
		row.LastErrorAt, row.LastError = at, message
	}
	_, err = o.Update(&row)
	return err
}

func (service Service) eventFromRow(ctx context.Context, row models.AgentMarketEvent, now time.Time) (Event, error) {
	if err := ctx.Err(); err != nil {
		return Event{}, err
	}
	var symbols []string
	_ = json.Unmarshal([]byte(row.SymbolsJSON), &symbols)
	var sourceRows []models.AgentMarketEventSource
	if _, err := service.ormer().QueryTable(new(models.AgentMarketEventSource)).Filter("event_id", row.ID).OrderBy("observed_at").All(&sourceRows); err != nil {
		return Event{}, err
	}
	sources := make([]EventSource, 0, len(sourceRows))
	for _, source := range sourceRows {
		sources = append(sources, EventSource{Source: source.Source, SourceRef: source.SourceRef, RawRef: source.RawRef, ObservedAt: source.ObservedAt})
	}
	age := max64(0, now.UnixMilli()-row.EventTime)
	return Event{ID: row.ID, EventKey: row.EventKey, Type: row.Type, Category: row.Category, Symbols: symbols, EventTime: row.EventTime, ObservedAt: row.ObservedAt, Source: row.Source, SourceRef: row.SourceRef, Headline: row.Headline, Summary: row.Summary, Severity: row.Severity, Confidence: row.Confidence, Freshness: freshness(row.Type, row.EventTime, now.UnixMilli()), FreshnessMs: age, RawRef: row.RawRef, Sources: sources}, nil
}
func factFromRow(row models.AgentMarketFact, now time.Time) Fact {
	age := max64(0, now.UnixMilli()-row.EventTime)
	return Fact{ID: row.ID, FactKey: row.FactKey, Type: row.Type, Category: row.Category, Symbol: row.Symbol, EventTime: row.EventTime, ObservedAt: row.ObservedAt, Source: row.Source, SourceRef: row.SourceRef, Severity: row.Severity, Confidence: row.Confidence, Freshness: freshness(row.Type, row.EventTime, now.UnixMilli()), FreshnessMs: age, RawRef: row.RawRef, Data: json.RawMessage(row.DataJSON)}
}

func normalizeEventInput(input EventInput, now time.Time) EventInput {
	input.Type, input.Category, input.Source = cleanLower(input.Type), cleanLower(input.Category), strings.TrimSpace(input.Source)
	input.Symbols = normalizeSymbols(input.Symbols)
	input.Headline = strings.TrimSpace(input.Headline)
	input.Summary = strings.TrimSpace(input.Summary)
	input.SourceRef, input.RawRef, input.DedupKey = strings.TrimSpace(input.SourceRef), strings.TrimSpace(input.RawRef), strings.TrimSpace(input.DedupKey)
	if input.ObservedAt <= 0 {
		input.ObservedAt = now.UnixMilli()
	}
	if input.EventTime <= 0 {
		input.EventTime = input.ObservedAt
	}
	if input.Severity == "" {
		input.Severity = SeverityInfo
	}
	input.Severity = cleanLower(input.Severity)
	if input.Confidence <= 0 {
		input.Confidence = 1
	}
	if input.Confidence > 1 {
		input.Confidence = 1
	}
	return input
}
func normalizeFactInput(input FactInput, now time.Time) FactInput {
	input.Type, input.Category, input.Source = cleanLower(input.Type), cleanLower(input.Category), strings.TrimSpace(input.Source)
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	input.SourceRef, input.RawRef, input.DedupKey = strings.TrimSpace(input.SourceRef), strings.TrimSpace(input.RawRef), strings.TrimSpace(input.DedupKey)
	if input.ObservedAt <= 0 {
		input.ObservedAt = now.UnixMilli()
	}
	if input.EventTime <= 0 {
		input.EventTime = input.ObservedAt
	}
	if input.Severity == "" {
		input.Severity = SeverityInfo
	}
	input.Severity = cleanLower(input.Severity)
	if input.Confidence <= 0 {
		input.Confidence = 1
	}
	if input.Confidence > 1 {
		input.Confidence = 1
	}
	return input
}
func validateEventInput(input EventInput) error {
	if input.Type == "" || input.Category == "" || input.Source == "" {
		return fmt.Errorf("market event type, category and source are required")
	}
	if input.EventTime <= 0 || input.ObservedAt <= 0 {
		return fmt.Errorf("market event time is required")
	}
	if input.Headline == "" && input.SourceRef == "" && input.DedupKey == "" {
		return fmt.Errorf("market event requires headline, source_ref or dedup_key")
	}
	return nil
}
func validateFactInput(input FactInput) error {
	if input.Type == "" || input.Category == "" || input.Symbol == "" || input.Source == "" {
		return fmt.Errorf("market fact type, category, symbol and source are required")
	}
	if input.EventTime <= 0 || input.ObservedAt <= 0 || len(input.Data) == 0 || !json.Valid(input.Data) {
		return fmt.Errorf("market fact requires valid time and data")
	}
	return nil
}
func eventKey(input EventInput) string {
	if input.DedupKey != "" {
		return hashKey("event|" + input.DedupKey)
	}
	headline := normalizeHeadline(input.Headline)
	bucket := input.EventTime / int64(time.Minute/time.Millisecond)
	if headline != "" {
		return hashKey(fmt.Sprintf("event|%s|%s|%s|%d", input.Type, strings.Join(input.Symbols, ","), headline, bucket))
	}
	return hashKey("event|" + input.Type + "|" + input.Category + "|" + input.Source + "|" + input.SourceRef)
}
func factKey(input FactInput) string {
	if input.DedupKey != "" {
		return hashKey("fact|" + input.DedupKey)
	}
	payload := sha256.Sum256(input.Data)
	return hashKey(fmt.Sprintf("fact|%s|%s|%s|%d|%x", input.Type, input.Symbol, input.Source, input.EventTime, payload[:8]))
}
func hashKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func normalizeSymbols(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
func normalizeHeadline(value string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			space = false
		} else if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}
func cleanLower(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
func normalizeLimit(value int) int {
	if value <= 0 {
		return 100
	}
	if value > 500 {
		return 500
	}
	return value
}
func freshness(kind string, eventTime, now int64) string {
	if eventTime <= 0 || now <= 0 {
		return FreshnessUnknown
	}
	age := now - eventTime
	if age < 0 {
		age = 0
	}
	ttl := freshnessTTL(kind)
	if ttl <= 0 {
		return FreshnessUnknown
	}
	if age <= ttl.Milliseconds() {
		return FreshnessFresh
	}
	return FreshnessStale
}
func freshnessTTL(kind string) time.Duration {
	switch kind {
	case FactTypeDepth, FactTypeTaker, FactTypeOpenInterest:
		return 5 * time.Minute
	case FactTypeFunding:
		return 15 * time.Minute
	case FactTypeLiquidation, EventTypeSignal:
		return time.Hour
	case EventTypeAnnouncement, EventTypeAlphaListing, EventTypeNews:
		return 24 * time.Hour
	default:
		return 6 * time.Hour
	}
}
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
