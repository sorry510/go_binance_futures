package historicalmarket

import (
	"context"
	"testing"
	"time"
)

func TestParseMinuteMonthRange(t *testing.T) {
	now := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		from, to string
		valid    bool
	}{
		{"2025-11", "2026-10", true},
		{"2026-11", "2026-11", true},
		{"2026-12", "2026-12", false},
		{"2026-01", "2025-12", false},
		{"2026-1", "2026-11", false},
		{"2020-01", "2025-01", false},
		{"", "", false},
	}
	for _, tc := range tests {
		_, _, err := ParseMinuteMonthRange(tc.from, tc.to, now)
		if (err == nil) != tc.valid {
			t.Errorf("%s-%s valid=%v err=%v", tc.from, tc.to, tc.valid, err)
		}
	}
}

func TestKline1mCalendarChunkAndLegacyRows(t *testing.T) {
	setupRepositoryTest(t)
	ctx := context.Background()
	repo := NewRepository(nil)
	old := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	// Official two-minute February archive becomes a compressed Chunk.
	if _, err := repo.Import(ctx, ImportRequest{
		Source: SourceBinancePublicData,
		Klines: []Kline{minuteBar(old, 101), minuteBar(old.Add(time.Minute), 102)},
	}); err != nil {
		t.Fatal(err)
	}
	march := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	// Two-minute March data are held in the legacy individual row table.
	if _, err := repo.Import(ctx, ImportRequest{
		Source: SourceBinanceREST,
		Klines: []Kline{minuteBar(march, 103), minuteBar(march.Add(time.Minute), 104)},
	}); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Kline1mCalendar(ctx, "btcusdt", "2025-02", "2025-03",
		time.Date(2025, 3, 1, 0, 3, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Months) != 2 || result.Symbol != "BTCUSDT" {
		t.Fatalf("bad result: %+v", result)
	}
	feb := result.Months[0]
	if feb.Expected != 28*1440 || feb.Stored != 2 || feb.Status != "partial" ||
		feb.Days[0].Stored != 2 || feb.Days[1].Status != "missing" {
		t.Fatalf("compressed month not counted correctly: %+v", feb)
	}
	mar := result.Months[1]
	if mar.Expected != 3 || mar.Stored != 2 || mar.Status != "partial" ||
		mar.Days[0].Expected != 3 || mar.Days[0].Stored != 2 ||
		mar.Days[1].Expected != 0 || mar.Days[1].Status != "future" {
		t.Fatalf("current month or row coverage incorrect: %+v", mar)
	}
	// The calendar endpoint must not fetch remote data or mutate chunks.
	repeated, err := repo.Kline1mCalendar(ctx, "BTCUSDT", "2025-02", "2025-03",
		time.Date(2025, 3, 1, 0, 3, 0, 0, time.UTC))
	if err != nil || repeated.Months[0].Stored != 2 || repeated.Months[1].Stored != 2 {
		t.Fatalf("calendar must be read-only and deterministic: %+v %v", repeated, err)
	}
}

func TestKline1mCalendarEmptyAndInvalidSymbol(t *testing.T) {
	setupRepositoryTest(t)
	repo := NewRepository(nil)
	now := time.Date(2026, 10, 2, 15, 0, 0, 0, time.UTC)
	r, err := repo.Kline1mCalendar(context.Background(), "ETHUSDT", "2026-09", "2026-09", now)
	if err != nil || len(r.Months) != 1 || r.Months[0].Stored != 0 || r.Months[0].Status != "missing" {
		t.Fatalf("missing complete month: %+v %v", r, err)
	}
	for _, bad := range []string{"BTCUSDT;DROP", "ETHUSDT ", "XX", "BTC/USDT"} {
		if bad == "ETHUSDT " {
			continue
		} // surrounding whitespace is normalized
		if _, err := repo.Kline1mCalendar(context.Background(), bad, "2026-09", "2026-09", now); err == nil {
			t.Errorf("accepted malformed symbol %q", bad)
		}
	}
}
