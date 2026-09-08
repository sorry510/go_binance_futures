package marketintelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	signalservice "go_binance_futures/service/signal"
)

func (service Service) Snapshot(ctx context.Context, symbol string, window time.Duration, limit int) (Snapshot, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" || !strings.HasSuffix(symbol, "USDT") {
		return Snapshot{}, fmt.Errorf("symbol must be a USDT futures contract")
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	now := service.now()
	start := now.Add(-window).UnixMilli()
	options := ListOptions{Symbol: symbol, StartTime: start, EndTime: now.UnixMilli(), Limit: limit}
	events, err := service.ListEvents(ctx, options)
	if err != nil {
		return Snapshot{}, err
	}
	facts, err := service.ListFacts(ctx, options)
	if err != nil {
		return Snapshot{}, err
	}
	sources, err := service.SourceStatuses(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	missing := []string{}
	for _, source := range sources {
		if source.Status == "error" {
			missing = append(missing, "source:"+source.Source)
		}
	}
	return Snapshot{
		Symbol: symbol, AsOf: now.Format(time.RFC3339), WindowStart: start,
		Events: events, Facts: facts, Sources: sources, DataMissing: missing,
	}, nil
}

func (service Service) Timeline(ctx context.Context, symbol string, startTime, endTime int64, limit int) (Snapshot, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" || !strings.HasSuffix(symbol, "USDT") {
		return Snapshot{}, fmt.Errorf("symbol must be a USDT futures contract")
	}
	if startTime <= 0 || endTime <= 0 || startTime > endTime {
		return Snapshot{}, fmt.Errorf("valid start_time and end_time are required")
	}
	asOf := time.UnixMilli(endTime).UTC()
	replay := service
	replay.Now = func() time.Time { return asOf }
	options := ListOptions{Symbol: symbol, StartTime: startTime, EndTime: endTime, ObservedBefore: endTime, Limit: limit}
	events, err := replay.ListEvents(ctx, options)
	if err != nil {
		return Snapshot{}, err
	}
	facts, err := replay.ListFacts(ctx, options)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Symbol: symbol, AsOf: asOf.Format(time.RFC3339), WindowStart: startTime, Events: events, Facts: facts, Sources: []SourceStatus{}, DataMissing: []string{}}, nil
}

// IngestSignal mirrors a deterministic local Signal into the unified event
// stream. The original Signal ID is the deduplication identity.
func (service Service) IngestSignal(ctx context.Context, signal signalservice.Signal) (Event, bool, error) {
	if strings.TrimSpace(signal.SignalID) == "" {
		return Event{}, false, fmt.Errorf("signal_id is required")
	}
	raw, _ := json.Marshal(signal)
	findings := make([]string, 0, len(signal.Evidence))
	for _, evidence := range signal.Evidence {
		if text := strings.TrimSpace(evidence.Finding); text != "" {
			findings = append(findings, text)
		}
	}
	eventTime := signal.CreatedAt
	if eventTime <= 0 {
		eventTime = service.now().UnixMilli()
	}
	return service.IngestEvent(ctx, EventInput{
		DedupKey: "signal:" + signal.SignalID,
		Type:     EventTypeSignal, Category: string(signal.Type), Symbols: []string{signal.Symbol},
		EventTime: eventTime, ObservedAt: service.now().UnixMilli(), Source: "signal_engine", SourceRef: signal.SignalID,
		Headline: strings.TrimSpace(signal.Symbol) + " " + string(signal.Type), Summary: strings.Join(findings, "; "),
		Severity: string(signal.Severity), Confidence: 1, Raw: raw,
	})
}
