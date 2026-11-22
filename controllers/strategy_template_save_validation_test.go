package controllers

import (
	"strings"
	"testing"

	"go_binance_futures/models"
)

const editableTechnology = `{"ma":[{"name":"ma_1h_3","enable":true,"kline_interval":"1h","period":3}]}`

func editableStrategyTemplate(technology, strategy string) models.StrategyTemplates {
	return models.StrategyTemplates{Name: "draft and live", Technology: technology, Strategy: strategy}
}

func TestEditableStrategyTemplateAllowsDraftSteps(t *testing.T) {
	tests := []models.StrategyTemplates{
		{Name: "new draft"},
		editableStrategyTemplate(editableTechnology, ""),
		editableStrategyTemplate(editableTechnology, "[]"),
		editableStrategyTemplate("", `[{"name":"","type":"","code":"invalid :::","enable":false,"fullScreen":false}]`),
		editableStrategyTemplate(editableTechnology, `[{"name":"disabled","type":"long","code":"invalid :::","enable":false}]`),
	}
	for _, candidate := range tests {
		if err := validateEditableStrategyTemplate(candidate); err != nil {
			t.Fatalf("valid editable draft rejected: %+v: %v", candidate, err)
		}
	}
}

func TestEditableStrategyTemplateValidatesEnabledRulesAndIndicators(t *testing.T) {
	tests := []struct{ name, technology, strategy, errorPart string }{
		{"valid entry", editableTechnology, `[{"name":"long","type":"long","code":"ma_1h_3.Data[1] > 0 && kline_1h.Close[1] > 0","enable":true}]`, ""},
		{"valid close", editableTechnology, `[{"name":"exit","type":"close_short","code":"ROI > 6 && Position.EntryPrice > 0","enable":true}]`, ""},
		{"invalid tech", `{"ma":[{"name":"BAD-DASH","enable":true,"kline_interval":"1h","period":3}]}`, "", "technology"},
		{"malformed tech", "{invalid", "", "JSON"},
		{"malformed rules", editableTechnology, "{invalid", "strategy"},
		{"non array rules", editableTechnology, `{"name":"bad"}`, "策略规则数组"},
		{"missing indicator", `{"ema":[{"name":"ema_4h_3","enable":true,"kline_interval":"4h","period":3}]}`, `[{"name":"long","type":"long","code":"kline_1h.Close[1] > 0","enable":true}]`, "kline_1h"},
		{"undefined symbol", editableTechnology, `[{"name":"long","type":"long","code":"UNKNOWN > 10","enable":true}]`, "编译失败"},
		{"invalid expr syntax", editableTechnology, `[{"name":"long","type":"long","code":"NowPrice >","enable":true}]`, "编译失败"},
		{"non boolean", editableTechnology, `[{"name":"long","type":"long","code":"NowPrice + 1","enable":true}]`, "编译失败"},
		{"invalid rule type", editableTechnology, `[{"name":"x","type":"invalid","code":"NowPrice > 0","enable":true}]`, "type"},
		{"missing enabled name", editableTechnology, `[{"type":"long","code":"NowPrice > 0","enable":true}]`, "name"},
		{"empty enabled code", editableTechnology, `[{"name":"x","type":"long","code":"","enable":true}]`, "code"},
		{"no active indicators", "{}", `[{"name":"x","type":"long","code":"NowPrice > 0","enable":true}]`, "已启用的技术指标"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEditableStrategyTemplate(editableStrategyTemplate(tc.technology, tc.strategy))
			if tc.errorPart == "" && err != nil {
				t.Fatalf("valid template rejected: %v", err)
			}
			if tc.errorPart != "" && (err == nil || !strings.Contains(err.Error(), tc.errorPart)) {
				t.Fatalf("got %v, expected error containing %q", err, tc.errorPart)
			}
		})
	}
}

func TestEditableStrategyTemplateRejectsRemovingReferencedIndicator(t *testing.T) {
	enabled := `[{"name":"entry","type":"long","code":"ma_1h_3.Data[1] > 0","enable":true}]`
	if err := validateEditableStrategyTemplate(editableStrategyTemplate(editableTechnology, enabled)); err != nil {
		t.Fatal(err)
	}
	if err := validateEditableStrategyTemplate(editableStrategyTemplate("{}", enabled)); err == nil {
		t.Fatal("enabled strategy must not retain undefined indicator after removal")
	}
}
