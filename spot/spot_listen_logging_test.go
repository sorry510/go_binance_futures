package spot

import (
	"testing"
	"time"

	"go_binance_futures/models"
)

func TestSpotListenLogAllowedOnlyOnTransitionsOrMinute(t *testing.T) {
	now := time.Unix(1730000000, 0)
	previous := spotListenLogState{outcome: spotListenNoMatch, at: now}
	if !spotListenLogAllowed(previous, false, spotListenNoMatch, now) {
		t.Fatal("first occurrence must be logged")
	}
	if spotListenLogAllowed(previous, true, spotListenNoMatch, now.Add(10*time.Second)) {
		t.Fatal("unchanged status within a minute should not flood logs")
	}
	if !spotListenLogAllowed(previous, true, spotListenCooldown, now.Add(10*time.Second)) {
		t.Fatal("a status change must be logged immediately")
	}
	if !spotListenLogAllowed(previous, true, spotListenNoMatch, now.Add(time.Minute)) {
		t.Fatal("unchanged status after a minute must be logged")
	}
}

func TestSpotListenerOutcomesWithoutMarketRequests(t *testing.T) {
	tests := []struct {
		name     string
		coin     models.ListenSymbols
		expected spotListenOutcome
	}{
		{
			name:     "no strategy configured",
			coin:     models.ListenSymbols{Symbol: "BTCUSDT"},
			expected: spotListenNoStrategy,
		},
		{
			name: "cooldown takes precedence",
			coin: models.ListenSymbols{
				Symbol: "BTCUSDT", Strategy: "[]", LastNoticeTime: time.Now().UnixMilli(),
				NoticeLimitMin: 5,
			},
			expected: spotListenCooldown,
		},
		{
			name: "no enabled strategy",
			coin: models.ListenSymbols{
				Symbol:   "BTCUSDT",
				Strategy: "[{\"name\":\"disabled\",\"code\":\"true\",\"enable\":false,\"type\":\"long\"}]",
			},
			expected: spotListenNoActiveRules,
		},
		{
			name: "unconfigured rule code",
			coin: models.ListenSymbols{
				Symbol:   "BTCUSDT",
				Strategy: "[{\"name\":\"empty\",\"code\":\"\",\"enable\":true,\"type\":\"short\"}]",
			},
			expected: spotListenNoActiveRules,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := spotKlineCustomListen(test.coin); got != test.expected {
				t.Fatalf("got status %q, want %q", got, test.expected)
			}
		})
	}
}
