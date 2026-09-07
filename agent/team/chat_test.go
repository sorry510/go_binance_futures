package team

import "testing"

func TestBuildChatInputUsesExplicitLiteralAndPreviousSymbol(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		explicit string
		previous []string
		want     string
	}{
		{name: "explicit", content: "分析一下", explicit: "btcusdt", want: "BTCUSDT"},
		{name: "literal", content: "分析 ongusdt 当前趋势", want: "ONGUSDT"},
		{name: "continue", content: "继续分析这个币", previous: []string{`{"symbol":"ETHUSDT","prompt":"first"}`}, want: "ETHUSDT"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input, err := BuildChatInput(tc.content, tc.previous, tc.explicit)
			if err != nil {
				t.Fatal(err)
			}
			if input.Symbol != tc.want || input.Prompt != tc.content {
				t.Fatalf("unexpected input: %+v", input)
			}
		})
	}
}

func TestBuildChatInputRequiresSymbol(t *testing.T) {
	if _, err := BuildChatInput("分析一下", nil, ""); err == nil {
		t.Fatal("expected missing symbol error")
	}
	if _, err := BuildChatInput("分析一下", nil, "BTC"); err == nil {
		t.Fatal("expected non-USDT symbol error")
	}
}
