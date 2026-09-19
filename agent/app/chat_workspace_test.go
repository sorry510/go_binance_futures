package app

import (
	"strings"
	"testing"
)

func TestAutoSelectChatSkillUsesOnlyAttachedSkills(t *testing.T) {
	catalog := map[string]ChatSkill{
		"general_chat":    {Name: "general_chat", DisplayName: "通用对话"},
		"symbol_analysis": {Name: "symbol_analysis", DisplayName: "单币分析"},
		"market_scan":     {Name: "market_scan", DisplayName: "市场扫描"},
	}
	attached := map[string]bool{
		"general_chat":    true,
		"symbol_analysis": true,
	}
	if got := autoSelectChatSkill("分析 BTCUSDT 的技术走势和支撑阻力", attached, catalog); got != "symbol_analysis" {
		t.Fatalf("auto selected %q, want symbol_analysis", got)
	}
	if got := autoSelectChatSkill("扫描市场机会并给我候选币", attached, catalog); got != "general_chat" {
		t.Fatalf("unattached market_scan must not be selected, got %q", got)
	}
}

func TestAutoSelectChatSkillFallsBackToGeneralChat(t *testing.T) {
	catalog := map[string]ChatSkill{
		"general_chat":    {Name: "general_chat", DisplayName: "通用对话"},
		"symbol_analysis": {Name: "symbol_analysis", DisplayName: "单币分析"},
		"market_scan":     {Name: "market_scan", DisplayName: "市场扫描"},
	}
	attached := map[string]bool{
		"general_chat": true,
		"market_scan":  true,
	}
	if got := autoSelectChatSkill("扫描市场机会并给我候选币", attached, catalog); got != "market_scan" {
		t.Fatalf("auto selected %q, want market_scan", got)
	}
	if got := autoSelectChatSkill("帮我解释一下这段文字", attached, catalog); got != "general_chat" {
		t.Fatalf("auto selected %q, want general_chat", got)
	}
}

func TestAutoSelectChatSkillScoreBoundaries(t *testing.T) {
	catalog := map[string]ChatSkill{
		"general_chat": {Name: "general_chat", DisplayName: "通用对话"},
		"market_scan":  {Name: "market_scan", DisplayName: "市场扫描"},
		"portable_one": {Name: "portable_one", DisplayName: "自定义研究"},
	}
	attached := map[string]bool{
		"general_chat": true,
		"market_scan":  true,
		"portable_one": true,
	}
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "single keyword stays below threshold", content: "扫描", want: "general_chat"},
		{name: "two keywords reach threshold", content: "扫描并选币", want: "market_scan"},
		{name: "skill name is explicit enough", content: "运行 market_scan", want: "market_scan"},
		{name: "display name is explicit enough", content: "请执行自定义研究", want: "portable_one"},
		{name: "slash name has highest priority", content: "/market_scan", want: "market_scan"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := autoSelectChatSkill(tt.content, attached, catalog); got != tt.want {
				t.Fatalf("auto selected %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChatRoutingTextIncludesSelectedSymbol(t *testing.T) {
	got := chatRoutingText("分析这个币", "btcusdt")
	if got != "分析这个币 BTCUSDT" {
		t.Fatalf("routing text = %q", got)
	}
}

func TestAutoSelectChatSkillUsesSelectedSymbolSignal(t *testing.T) {
	catalog := map[string]ChatSkill{
		"general_chat":    {Name: "general_chat", DisplayName: "通用对话"},
		"symbol_analysis": {Name: "symbol_analysis", DisplayName: "单币分析"},
	}
	attached := map[string]bool{
		"general_chat":    true,
		"symbol_analysis": true,
	}
	got := autoSelectChatSkill(chatRoutingText("分析这个币", "BTCUSDT"), attached, catalog)
	if got != "symbol_analysis" {
		t.Fatalf("auto selected %q, want symbol_analysis", got)
	}
}

func TestGeneralChatInputPinsSelectedSymbol(t *testing.T) {
	got := generalChatInput("分析这个币", "ethusdt")
	if !strings.Contains(got, "Selected Binance USDT futures symbol: ETHUSDT") {
		t.Fatalf("selected symbol missing from general chat input: %q", got)
	}
	if !strings.Contains(got, "Do not substitute a different symbol from conversation history") {
		t.Fatalf("history override guard missing: %q", got)
	}
}
