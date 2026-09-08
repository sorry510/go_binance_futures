package symbolanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	marketintelligence "go_binance_futures/service/marketintelligence"
)

func persistMarketIntelligence(ctx context.Context, service marketintelligence.Service, value Context) (marketintelligence.Snapshot, error) {
	asOf, err := time.Parse(time.RFC3339, value.AsOf)
	if err != nil {
		return marketintelligence.Snapshot{}, fmt.Errorf("parse symbol context as_of: %w", err)
	}
	eventTime := asOf.UnixMilli()
	bucket := eventTime / int64(time.Minute/time.Millisecond)
	facts := []struct {
		kind   string
		source string
		value  any
	}{
		{marketintelligence.FactTypeFunding, "binance_futures", value.Funding},
		{marketintelligence.FactTypeOpenInterest, "binance_futures", value.OpenInterest},
		{marketintelligence.FactTypeTaker, "binance_futures", value.Taker},
		{marketintelligence.FactTypeDepth, "binance_futures", value.Depth},
		{marketintelligence.FactTypeLiquidation, "local_liquidation_store", value.Liquidations},
	}
	for _, fact := range facts {
		if fact.value == nil {
			continue
		}
		raw, marshalErr := json.Marshal(fact.value)
		if marshalErr != nil {
			continue
		}
		_, _, ingestErr := service.IngestFact(ctx, marketintelligence.FactInput{
			DedupKey: fmt.Sprintf("symbol-context:%s:%s:%d", value.Symbol, fact.kind, bucket),
			Type:     fact.kind, Category: "derivatives", Symbol: value.Symbol,
			EventTime: eventTime, ObservedAt: eventTime, Source: fact.source,
			Severity: marketintelligence.SeverityInfo, Confidence: 1, Data: raw,
		})
		if ingestErr != nil {
			return marketintelligence.Snapshot{}, ingestErr
		}
	}
	return service.Snapshot(ctx, value.Symbol, 24*time.Hour, 100)
}
