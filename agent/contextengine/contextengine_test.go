package contextengine

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBuildTrimsHistoryBeforeCurrentTask(t *testing.T) {
	engine := New()
	blocks := []ContextBlock{
		{ID: "old-1", Type: BlockHistory, Source: "conversation", Role: "user", Content: strings.Repeat("old ", 120)},
		{ID: "old-2", Type: BlockHistory, Source: "conversation", Role: "assistant", Content: strings.Repeat("older ", 120)},
		{ID: "market", Type: BlockMarket, Source: "get_symbol_snapshot", Role: "user", Content: "price=1.23", Freshness: FreshnessFresh},
		{ID: "task", Type: BlockTask, Source: "skill_input", Role: "user", Content: "analyze BTCUSDT now", Required: true},
	}
	messages, trace, err := engine.Build(BuildOptions{SystemPrompt: "protocol", Blocks: blocks, MaxTokens: 80, MaxBytes: 4096, Now: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, message := range messages {
		joined += message.Content
	}
	if !strings.Contains(joined, "analyze BTCUSDT now") || !strings.Contains(joined, "price=1.23") {
		t.Fatalf("current task/market context was trimmed: %q", joined)
	}
	if trace.TrimmedBlocks == 0 {
		t.Fatalf("expected explainable trimming: %+v", trace)
	}
	for _, trimmed := range trace.Trimmed {
		if trimmed.BlockID == "task" {
			t.Fatalf("required current task was trimmed: %+v", trace)
		}
	}
}

func TestBuildRejectsOnlyWhenRequiredContextCannotFit(t *testing.T) {
	engine := New()
	_, _, err := engine.Build(BuildOptions{
		SystemPrompt: "protocol",
		Blocks:       []ContextBlock{{ID: "task", Type: BlockTask, Required: true, Content: strings.Repeat("必须保留", 100)}},
		MaxTokens:    10,
	})
	if err == nil || !strings.Contains(err.Error(), "required context block") {
		t.Fatalf("expected required-context budget error, got %v", err)
	}
}

func TestConvertToolResultMarksStaleAndCreatesDeterministicEvidence(t *testing.T) {
	engine := New()
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	value := map[string]any{
		"symbol":   "BTCUSDT",
		"as_of":    now.Add(-10 * time.Minute).Format(time.RFC3339),
		"snapshot": map[string]any{"price": 100.0},
	}
	first, err := engine.ConvertToolResult("get_symbol_analysis_context", value, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := engine.ConvertToolResult("get_symbol_analysis_context", value, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Evidence) != 1 || first.Evidence[0].Freshness != FreshnessStale {
		t.Fatalf("expected stale evidence: %+v", first.Evidence)
	}
	if first.Evidence[0].ID != second.Evidence[0].ID || first.Evidence[0].ContentHash != second.Evidence[0].ContentHash {
		t.Fatalf("evidence identity is not deterministic: %+v %+v", first.Evidence, second.Evidence)
	}
	if first.Block.Type != BlockMarket || len(first.Block.EvidenceIDs) != 1 {
		t.Fatalf("unexpected market context block: %+v", first.Block)
	}
}

func TestConvertSymbolAnalysisContextHonorsExplicitStaleMarker(t *testing.T) {
	engine := New()
	now := time.Now().UTC()
	conversion, err := engine.ConvertToolResult("get_symbol_analysis_context", map[string]any{
		"symbol": "BTCUSDT", "as_of": now.Format(time.RFC3339),
		"data_missing": []string{"symbol_snapshot_stale"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if conversion.Evidence[0].Freshness != FreshnessStale || conversion.Evidence[0].StaleReason != "symbol_snapshot_stale" {
		t.Fatalf("explicit stale marker was lost: %+v", conversion.Evidence[0])
	}
}

func TestProgressiveResourceDisclosure(t *testing.T) {
	engine := New()
	loaded := []string{}
	resources := []Resource{
		{ID: "skill-md", Type: BlockSkillInstruction, Disclosure: DisclosureActivation, Load: func(context.Context) (string, error) {
			loaded = append(loaded, "skill")
			return "skill instructions", nil
		}},
		{ID: "reference-risk", Type: BlockSkillInstruction, Disclosure: DisclosureOnDemand, Load: func(context.Context) (string, error) {
			loaded = append(loaded, "reference")
			return "risk reference", nil
		}},
	}
	blocks, err := engine.LoadResources(context.Background(), resources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || strings.Join(loaded, ",") != "skill" {
		t.Fatalf("on-demand resource loaded eagerly: blocks=%+v loaded=%v", blocks, loaded)
	}
	blocks, err = engine.LoadResources(context.Background(), resources, []string{"reference-risk"})
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 2 || strings.Join(loaded, ",") != "skill,skill,reference" {
		t.Fatalf("requested resource was not disclosed: blocks=%+v loaded=%v", blocks, loaded)
	}
}

func TestConvertSymbolSnapshotUsesCamelCaseUpdateTimeForFreshness(t *testing.T) {
	engine := New()
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	conversion, err := engine.ConvertToolResult("get_symbol_snapshot", map[string]any{
		"symbol": "BTCUSDT", "close": "100", "updateTime": now.Add(-10 * time.Minute).UnixMilli(),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(conversion.Evidence) != 1 || conversion.Evidence[0].Freshness != FreshnessStale {
		t.Fatalf("camelCase updateTime was not used for freshness: %+v", conversion.Evidence)
	}
	if conversion.Evidence[0].FreshnessAge < (9 * time.Minute).Milliseconds() {
		t.Fatalf("unexpected freshness age: %+v", conversion.Evidence[0])
	}
}

func TestConvertToolResultDoesNotTreatOrdinaryNumbersAsTimestamps(t *testing.T) {
	engine := New()
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	conversion, err := engine.ConvertToolResult("get_market_condition", map[string]any{
		"market_condition": 3,
		"price":            100.25,
		"confidence":       0.9,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(conversion.Evidence) != 1 {
		t.Fatalf("unexpected evidence: %+v", conversion.Evidence)
	}
	evidence := conversion.Evidence[0]
	if evidence.AsOf != "" || evidence.Freshness != FreshnessUnknown {
		t.Fatalf("ordinary numeric fields were misclassified as timestamps: %+v", evidence)
	}
}
