package contextengine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ToolConversion struct {
	Block      ContextBlock    `json:"block"`
	Evidence   []Evidence      `json:"evidence"`
	ResultJSON json.RawMessage `json:"result_json"`
}

func (engine *Engine) ConvertToolResult(source string, value any, observedAt time.Time) (ToolConversion, error) {
	if engine == nil {
		engine = New()
	}
	observedAt = observedAt.UTC()
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ToolConversion{}, fmt.Errorf("encode tool evidence %q: %w", source, err)
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ToolConversion{}, fmt.Errorf("decode tool evidence %q: %w", source, err)
	}
	asOf := extractAsOf(decoded)
	missing := extractDataMissing(decoded)
	freshness, age, reason := engine.Freshness.Evaluate(source, asOf, observedAt, missing)
	hash := ContentHash(string(raw))
	evidenceID := "ev_" + hash[:20]
	if source != "" {
		sourceHash := ContentHash(source + "|" + hash)
		evidenceID = "ev_" + sourceHash[:20]
	}
	evidence := Evidence{
		ID: evidenceID, SourceType: "tool", Source: source, ObservedAt: observedAt.Format(time.RFC3339),
		ContentHash: hash, Freshness: freshness, FreshnessAge: age.Milliseconds(), StaleReason: reason,
		KeyFields: summarizeKeyFields(decoded), DataMissing: missing,
	}
	if asOf != nil {
		evidence.AsOf = asOf.UTC().Format(time.RFC3339)
	}
	blockType := BlockTool
	if isMarketSource(source) {
		blockType = BlockMarket
	}
	block := ContextBlock{
		ID: "tool-" + evidenceID, Type: blockType, Source: source, Priority: DefaultPriority(blockType),
		AsOf: evidence.AsOf, Freshness: freshness, Content: string(raw), ContentHash: hash,
		EvidenceIDs: []string{evidenceID},
	}
	return ToolConversion{Block: block, Evidence: []Evidence{evidence}, ResultJSON: append(json.RawMessage(nil), raw...)}, nil
}

func isMarketSource(source string) bool {
	switch source {
	case "get_symbol_analysis_context", "get_symbol_snapshot", "get_features", "get_klines", "get_funding_rate", "get_liquidations", "get_market_condition", "scan_symbols":
		return true
	default:
		return false
	}
}

func extractAsOf(value any) *time.Time {
	if parsed := timestampFromMap(value); parsed != nil {
		return parsed
	}
	var latest *time.Time
	walkJSON(value, 0, func(key string, scalar any) {
		if latest != nil && (key == "as_of" || key == "updated_at_ms" || key == "update_time") {
			return
		}
		candidate := parseTimestamp(key, scalar)
		if candidate != nil && (latest == nil || candidate.After(*latest)) {
			copy := candidate.UTC()
			latest = &copy
		}
	})
	return latest
}

func timestampFromMap(value any) *time.Time {
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range []string{"as_of", "updated_at_ms", "updatedAtMs", "update_time", "updateTime", "event_time", "eventTime", "timestamp", "created_at", "createdAt", "closeTime", "openTime"} {
		if raw, exists := object[key]; exists {
			if parsed := parseTimestamp(key, raw); parsed != nil {
				return parsed
			}
		}
	}
	if snapshot, ok := object["snapshot"].(map[string]any); ok {
		if raw, exists := snapshot["updated_at_ms"]; exists {
			return parseTimestamp("updated_at_ms", raw)
		}
	}
	return nil
}

func parseTimestamp(key string, value any) *time.Time {
	if !isTimestampKey(key) {
		return nil
	}
	if text, ok := value.(string); ok {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(text)); err == nil {
			return &parsed
		}
		if numeric, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64); err == nil {
			return unixFlexible(numeric)
		}
	}
	switch number := value.(type) {
	case float64:
		return unixFlexible(int64(number))
	case int64:
		return unixFlexible(number)
	case int:
		return unixFlexible(int64(number))
	}
	return nil
}

func isTimestampKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "as_of", "updated_at_ms", "updatedatms", "update_time", "updatetime",
		"event_time", "eventtime", "timestamp", "created_at", "createdat",
		"closetime", "opentime":
		return true
	default:
		return false
	}
}

func unixFlexible(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	var parsed time.Time
	if value > 100000000000 {
		parsed = time.UnixMilli(value).UTC()
	} else {
		parsed = time.Unix(value, 0).UTC()
	}
	return &parsed
}

func extractDataMissing(value any) []string {
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := object["data_missing"].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			result = append(result, strings.TrimSpace(text))
		}
	}
	sort.Strings(result)
	return result
}

func summarizeKeyFields(value any) map[string]string {
	result := map[string]string{}
	preferred := map[string]bool{
		"symbol": true, "price": true, "close": true, "percent_change_24h": true, "percentChange": true, "market_condition": true,
		"direction": true, "confidence": true, "rate_pct": true, "mark_price": true,
		"count": true, "long_notional": true, "short_notional": true, "interval": true,
	}
	walkJSON(value, 0, func(key string, scalar any) {
		if len(result) >= 12 || !preferred[key] {
			return
		}
		if text := scalarText(scalar); text != "" {
			if _, exists := result[key]; !exists {
				result[key] = text
			}
		}
	})
	return result
}

func scalarText(value any) string {
	switch scalar := value.(type) {
	case string:
		if len(scalar) > 120 {
			return scalar[:120]
		}
		return scalar
	case float64:
		return strconv.FormatFloat(scalar, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(scalar)
	default:
		return ""
	}
}

func walkJSON(value any, depth int, visit func(string, any)) {
	if depth > 4 {
		return
	}
	switch current := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			child := current[key]
			switch child.(type) {
			case map[string]any, []any:
				walkJSON(child, depth+1, visit)
			default:
				visit(key, child)
			}
		}
	case []any:
		limit := len(current)
		if limit > 20 {
			limit = 20
		}
		for index := 0; index < limit; index++ {
			walkJSON(current[index], depth+1, visit)
		}
	}
}
