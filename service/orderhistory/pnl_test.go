package orderhistory

import (
	"context"
	"testing"
)

func TestDisplayedPNLUsesAllFilteredCyclesNotPageAndPreservesCertifiedAudit(t *testing.T) {
	o, store := historyDB(t)
	closedID := fullCycle(t, o)
	activeID := positionFixture(t, o, "main", "auto_strategy", "BTCUSDT", "LONG", "active", "open", false)
	if _, err := o.Raw("UPDATE futures_managed_positions SET managed_qty=2 WHERE id=?", activeID).Exec(); err != nil {
		t.Fatal(err)
	}
	managedOrder(t, o, activeID, "main", "auto_strategy", "BTCUSDT", "LONG", "second-open", "open", "501", 2)
	fill(t, o, "main", "BTCUSDT", "LONG", "501", "501", "second-open", "open", "2", "100", "0.1", "0", "USDT", activeID, false)
	if _, err := o.Raw("INSERT INTO symbols (symbol,close,updateTime,type) VALUES (?,?,?,?)", "BTCUSDT", "110", 1800000010000, "USDT").Exec(); err != nil {
		t.Fatal(err)
	}
	for pageNum := 1; pageNum <= 2; pageNum++ {
		page, err := store.List(context.Background(), Filter{AccountID: "main", Page: pageNum, Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != 2 || len(page.List) != 1 {
			t.Fatalf("wrong pagination: %+v", page)
		}
		got := page.FilteredPnL["USDT"]
		if got.Realized != "110" || got.Unrealized != "20" || got.Total != "130" || page.PNLUnpriced != 0 {
			t.Fatalf("all-filtered summary depends on page %d: %+v", pageNum, page)
		}
		cycle := page.List[0]
		if cycle.CurrentPrice == nil || *cycle.CurrentPrice != "110" {
			t.Fatalf("symbol quote missing: %+v", cycle)
		}
		switch cycle.PositionID {
		case activeID:
			if cycle.RealizedPnL == nil || *cycle.RealizedPnL != "0" ||
				cycle.UnrealizedPnL == nil || *cycle.UnrealizedPnL != "20" ||
				cycle.TotalPnL == nil || *cycle.TotalPnL != "20" ||
				cycle.NetPnL != nil {
				t.Fatalf("active cycle PnL incorrect or fabricated certified net: %+v", cycle)
			}
		case closedID:
			if cycle.RealizedPnL == nil || *cycle.RealizedPnL != "110" ||
				cycle.UnrealizedPnL == nil || *cycle.UnrealizedPnL != "0" ||
				cycle.TotalPnL == nil || *cycle.TotalPnL != "110" ||
				cycle.NetPnL == nil || *cycle.NetPnL != "109.2" {
				t.Fatalf("closed cycle indicative PnL or certified net incorrect: %+v", cycle)
			}
		default:
			t.Fatalf("unrelated position in filter: %d", cycle.PositionID)
		}
	}
	outside, err := store.List(context.Background(), Filter{AccountID: "main", Page: 99, Limit: 1})
	if err != nil || outside.Total != 2 || len(outside.List) != 0 || outside.FilteredPnL["USDT"].Total != "130" {
		t.Fatalf("out-of-range page lost whole-filter PnL: %+v %v", outside, err)
	}
	closed, err := store.List(context.Background(), Filter{AccountID: "main", Status: "closed", Page: 1, Limit: 1})
	if err != nil || closed.Total != 1 || closed.FilteredPnL["USDT"].Total != "110" {
		t.Fatalf("filtered status not respected: %+v, %v", closed, err)
	}
	active, err := store.List(context.Background(), Filter{AccountID: "main", Status: "active", Page: 1, Limit: 1})
	if err != nil || active.Total != 1 || active.FilteredPnL["USDT"].Total != "20" {
		t.Fatalf("filtered status not respected: %+v, %v", active, err)
	}
}

func TestShortPartialCloseAndMissingPriceDoNotMisstateFilteredTotal(t *testing.T) {
	o, store := historyDB(t)
	id := positionFixture(t, o, "main", "auto_strategy", "BTCUSDT", "SHORT", "active", "open", false)
	if _, err := o.Raw("UPDATE futures_managed_positions SET managed_qty=1 WHERE id=?", id).Exec(); err != nil {
		t.Fatal(err)
	}
	managedOrder(t, o, id, "main", "auto_strategy", "BTCUSDT", "SHORT", "short-open", "open", "701", 3)
	managedOrder(t, o, id, "main", "auto_strategy", "BTCUSDT", "SHORT", "short-close", "close", "702", 2)
	fill(t, o, "main", "BTCUSDT", "SHORT", "701", "701", "short-open", "open", "3", "100", "0.01", "0", "USDT", id, false)
	fill(t, o, "main", "BTCUSDT", "SHORT", "702", "702", "short-close", "close", "2", "90", "0.01", "20", "USDT", id, false)
	if _, err := o.Raw("INSERT INTO symbols (symbol,close,updateTime,type) VALUES (?,?,?,?)", "BTCUSDT", "90", 1800000010000, "USDT").Exec(); err != nil {
		t.Fatal(err)
	}
	page, err := store.List(context.Background(), Filter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	p := page.List[0]
	if p.RealizedPnL == nil || *p.RealizedPnL != "20" ||
		p.UnrealizedPnL == nil || *p.UnrealizedPnL != "10" ||
		p.TotalPnL == nil || *p.TotalPnL != "30" ||
		page.FilteredPnL["USDT"].Total != "30" ||
		p.NetPnL != nil {
		t.Fatalf("short partial close or uncertified display incorrect: %+v %+v", p, page)
	}
	if _, err := o.Raw("DELETE FROM symbols WHERE symbol=?", "BTCUSDT").Exec(); err != nil {
		t.Fatal(err)
	}
	missing, err := store.List(context.Background(), Filter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if missing.List[0].TotalPnL != nil || missing.PNLUnpriced != 1 || len(missing.FilteredPnL) != 0 {
		t.Fatalf("missing live price silently fabricated aggregate: %+v", missing)
	}
}
