package governance

import (
	"strings"
	"testing"
)

func TestLimiterEnforcesMinuteBudget(t *testing.T) {
	limiter := New(func() Limits { return Limits{PerMinute: 1, PerHour: 2} })
	if err := limiter.Admit("symbol_analysis"); err != nil {
		t.Fatal(err)
	}
	err := limiter.Admit("symbol_analysis")
	if err == nil || !strings.Contains(err.Error(), "budget exceeded") {
		t.Fatalf("expected budget rejection, got %v", err)
	}
	status := limiter.Status()
	if status.Accepted != 1 || status.Rejected != 1 || status.RecentMinute != 1 {
		t.Fatalf("unexpected limiter status: %+v", status)
	}
	skill := status.Skills["symbol_analysis"]
	if skill.Accepted != 1 || skill.Rejected != 1 {
		t.Fatalf("unexpected skill status: %+v", skill)
	}
}
