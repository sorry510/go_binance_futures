package backtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"go_binance_futures/service/historicalmarket"
)

type PriceDirection string

const (
	PriceDirectionGTE PriceDirection = ">="
	PriceDirectionLTE PriceDirection = "<="
)

type PriceEvent struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	TriggerPrice float64        `json:"trigger_price"`
	Direction    PriceDirection `json:"direction"`
	KnownAt      int64          `json:"known_at"`
	Priority     int            `json:"priority"`
}

type IntrabarDecision struct {
	Resolved       bool                                 `json:"resolved"`
	Event          PriceEvent                           `json:"event,omitempty"`
	Resolution     string                               `json:"resolution"`
	HitTime        int64                                `json:"hit_time,omitempty"`
	HitPrice       float64                              `json:"hit_price,omitempty"`
	Candidates     []PriceEvent                         `json:"candidates,omitempty"`
	SecondEvidence *historicalmarket.ResolutionEvidence `json:"second_evidence,omitempty"`
	TradeEvidence  *historicalmarket.ResolutionEvidence `json:"trade_evidence,omitempty"`
}

type IntrabarResolver struct {
	Provider historicalmarket.ResolutionProvider
}

func (resolver IntrabarResolver) Resolve(ctx context.Context, minute historicalmarket.Kline, events []PriceEvent) (IntrabarDecision, error) {
	if err := ctx.Err(); err != nil {
		return IntrabarDecision{}, err
	}
	if minute.OpenTime <= 0 || minute.CloseTime < minute.OpenTime || minute.High < minute.Low {
		return IntrabarDecision{}, fmt.Errorf("invalid minute bar for intrabar resolution")
	}
	candidates := make([]PriceEvent, 0, len(events))
	requiresTimeResolution := false
	for _, event := range events {
		if err := validatePriceEvent(event); err != nil {
			return IntrabarDecision{}, err
		}
		if event.KnownAt > minute.CloseTime {
			continue
		}
		if !eventTouched(event, minute.High, minute.Low) {
			continue
		}
		candidates = append(candidates, event)
		if event.KnownAt > minute.OpenTime {
			requiresTimeResolution = true
		}
	}
	sortPriceEvents(candidates)
	if len(candidates) == 0 {
		return IntrabarDecision{Resolved: false, Resolution: "1m", Candidates: candidates}, nil
	}
	if len(candidates) == 1 && !requiresTimeResolution {
		event := candidates[0]
		return IntrabarDecision{Resolved: true, Event: event, Resolution: "1m", HitTime: minute.CloseTime, HitPrice: event.TriggerPrice, Candidates: candidates}, nil
	}
	if resolver.Provider == nil {
		return IntrabarDecision{}, fmt.Errorf("intrabar resolution requires a high-resolution provider")
	}

	seconds, evidence, err := resolver.Provider.SecondBars(ctx, minute.Market, minute.Symbol, minute.OpenTime, minute.CloseTime)
	if err != nil {
		return IntrabarDecision{}, fmt.Errorf("load second bars: %w", err)
	}
	for _, second := range seconds {
		if second.CloseTime < minute.OpenTime || second.OpenTime > minute.CloseTime {
			continue
		}
		hits := make([]PriceEvent, 0, len(candidates))
		knownInsideSecond := false
		for _, event := range candidates {
			if event.KnownAt > second.CloseTime {
				continue
			}
			if !eventTouched(event, second.High, second.Low) {
				continue
			}
			hits = append(hits, event)
			if event.KnownAt > second.OpenTime {
				knownInsideSecond = true
			}
		}
		if len(hits) == 0 {
			continue
		}
		if len(hits) == 1 && !knownInsideSecond {
			event := hits[0]
			return IntrabarDecision{Resolved: true, Event: event, Resolution: "1s", HitTime: second.CloseTime, HitPrice: event.TriggerPrice, Candidates: candidates, SecondEvidence: &evidence}, nil
		}
		decision, tradeEvidence, err := resolver.resolveWithTrades(ctx, minute.Market, minute.Symbol, second, hits)
		if err != nil {
			return IntrabarDecision{}, err
		}
		decision.Candidates = candidates
		decision.SecondEvidence = &evidence
		decision.TradeEvidence = &tradeEvidence
		return decision, nil
	}
	return IntrabarDecision{Resolved: false, Resolution: "1s", Candidates: candidates, SecondEvidence: &evidence}, nil
}

func (resolver IntrabarResolver) resolveWithTrades(ctx context.Context, market, symbol string, second historicalmarket.Kline, events []PriceEvent) (IntrabarDecision, historicalmarket.ResolutionEvidence, error) {
	trades, evidence, err := resolver.Provider.Trades(ctx, market, symbol, second.OpenTime, second.CloseTime)
	if err != nil {
		return IntrabarDecision{}, historicalmarket.ResolutionEvidence{}, fmt.Errorf("load trades for intrabar conflict: %w", err)
	}
	for _, trade := range trades {
		hits := make([]PriceEvent, 0, len(events))
		for _, event := range events {
			if trade.TradeTime < event.KnownAt || !eventSatisfied(event, trade.Price) {
				continue
			}
			hits = append(hits, event)
		}
		if len(hits) == 0 {
			continue
		}
		sortPriceEvents(hits)
		return IntrabarDecision{Resolved: true, Event: hits[0], Resolution: "trades", HitTime: trade.TradeTime, HitPrice: trade.Price}, evidence, nil
	}
	return IntrabarDecision{Resolved: false, Resolution: "trades"}, evidence, nil
}

func validatePriceEvent(event PriceEvent) error {
	if strings.TrimSpace(event.ID) == "" {
		return fmt.Errorf("price event id is required")
	}
	if strings.TrimSpace(event.Type) == "" {
		return fmt.Errorf("price event %s type is required", event.ID)
	}
	if event.TriggerPrice <= 0 {
		return fmt.Errorf("price event %s trigger price must be positive", event.ID)
	}
	if event.KnownAt <= 0 {
		return fmt.Errorf("price event %s known_at must be positive", event.ID)
	}
	if event.Direction != PriceDirectionGTE && event.Direction != PriceDirectionLTE {
		return fmt.Errorf("price event %s has unsupported direction %q", event.ID, event.Direction)
	}
	return nil
}

func eventTouched(event PriceEvent, high, low float64) bool {
	switch event.Direction {
	case PriceDirectionGTE:
		return high >= event.TriggerPrice
	case PriceDirectionLTE:
		return low <= event.TriggerPrice
	default:
		return false
	}
}

func eventSatisfied(event PriceEvent, price float64) bool {
	switch event.Direction {
	case PriceDirectionGTE:
		return price >= event.TriggerPrice
	case PriceDirectionLTE:
		return price <= event.TriggerPrice
	default:
		return false
	}
}

func sortPriceEvents(events []PriceEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Priority != events[j].Priority {
			return events[i].Priority < events[j].Priority
		}
		if events[i].KnownAt != events[j].KnownAt {
			return events[i].KnownAt < events[j].KnownAt
		}
		if events[i].ID != events[j].ID {
			return events[i].ID < events[j].ID
		}
		return events[i].Type < events[j].Type
	})
}

// MergeResolutionEvidenceHash extends the canonical V3-3 data hash only with
// high-resolution evidence that was actually consumed by the adaptive run.
// Evidence order and duplicate references do not affect the resulting hash.
func MergeResolutionEvidenceHash(base string, evidence ...historicalmarket.ResolutionEvidence) string {
	hashes := make([]string, 0, len(evidence))
	seen := map[string]struct{}{}
	for _, item := range evidence {
		value := strings.TrimSpace(item.EvidenceHash)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		hashes = append(hashes, value)
	}
	if len(hashes) == 0 {
		return base
	}
	sort.Strings(hashes)
	h := sha256.New()
	_, _ = h.Write([]byte("base:" + base + "\n"))
	for _, value := range hashes {
		_, _ = h.Write([]byte("evidence:" + value + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}
