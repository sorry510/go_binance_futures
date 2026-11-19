package orderhistory

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// UF-6 synthetic benchmark: isolated SQLite fixture ONLY; never reads or
// writes an operational MySQL or Binance account. Trigger explicitly with
// UF6_BENCH_MODE=baseline or UF6_BENCH_MODE=after.
func uf6PopulateBenchmark(t *testing.T, store Store, n int) {
	t.Helper()
	o := store.db()
	if _, err := o.Raw("INSERT INTO symbols (symbol,close,updateTime,type) VALUES (?,?,?,?)", "BTCUSDT", "110", 1800000010000, "USDT").Exec(); err != nil {
		t.Fatal(err)
	}
	tx, err := o.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	const batch = 50
	for off := 0; off < n; off += batch {
		end := off + batch
		if end > n {
			end = n
		}
		var values []string
		var args []any
		for i := off; i < end; i++ {
			values = append(values, "(?,?,?,?,?,?,?,?,?,?)")
			args = append(args, "main", "auto_strategy", "BTCUSDT", "LONG", "active", 1, 100, "open", int64(1800000000000+i), int64(1800000000000+i))
		}
		stmt := `INSERT INTO futures_managed_positions
 (account_id,owner,symbol,position_side,status,managed_qty,entry_price,settlement_status,created_at,updated_at)
 VALUES ` + strings.Join(values, ",")
		if _, err = tx.Raw(stmt, args...).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
func TestUF6SyntheticBenchmark(t *testing.T) {
	mode := os.Getenv("UF6_BENCH_MODE")
	if mode != "baseline" && mode != "after" {
		t.Skip("run explicitly with UF6_BENCH_MODE")
	}
	for _, n := range []int{100, 1000, 10000} {
		t.Run(fmt.Sprintf("n_%d", n), func(t *testing.T) {
			_, store := historyDB(t)
			setup := time.Now()
			uf6PopulateBenchmark(t, store, n)
			fixtureDuration := time.Since(setup)
			ctx := context.Background()
			query := Filter{AccountID: "main", Page: 1, Limit: 20}
			start := time.Now()
			old, err := store.List(ctx, query)
			oldTime := time.Since(start)
			if err != nil {
				t.Fatal(err)
			}
			if old.Total != int64(n) || old.PNLUnpriced != 0 || old.FilteredPnL["USDT"].Total != fmt.Sprint(n*10) {
				t.Fatalf("synthetic old PnL must match exactly, got %+v", old)
			}
			start = time.Now()
			next, err := store.List(ctx, Filter{AccountID: "main", Page: 2, Limit: 20})
			oldFlip := time.Since(start)
			if err != nil || !reflect.DeepEqual(next.FilteredPnL, old.FilteredPnL) {
				t.Fatalf("legacy pagination sum differs: %+v, %v", next, err)
			}
			t.Logf("UF6_BENCH mode=%s n=%d fixture=%v legacy_page1=%v legacy_page2=%v total=%s", mode, n, fixtureDuration, oldTime, oldFlip, old.FilteredPnL["USDT"].Total)
			if mode == "after" {
				start = time.Now()
				page, err := store.ListPageOnly(ctx, query)
				modern := time.Since(start)
				if err != nil || page.Total != old.Total || !reflect.DeepEqual(page.List, old.List) {
					t.Fatalf("new page differs from legacy: %v", err)
				}
				start = time.Now()
				summary, err := store.Summary(ctx, query)
				summaryTime := time.Since(start)
				if err != nil || !reflect.DeepEqual(summary.FilteredPnL, old.FilteredPnL) || summary.PNLUnpriced != old.PNLUnpriced {
					t.Fatalf("new summary differs from legacy: %+v, %v", summary, err)
				}
				start = time.Now()
				page2, err := store.ListPageOnly(ctx, Filter{AccountID: "main", Page: 2, Limit: 20})
				modernFlip := time.Since(start)
				if err != nil || !reflect.DeepEqual(page2.List, next.List) {
					t.Fatalf("new page2 differs from legacy: %v", err)
				}
				t.Logf("UF6_BENCH_SPLIT n=%d list_page1=%v summary_once=%v page2=%v estimated_SQL_roundtrips_per_flip legacy=%d split=5", n, modern, summaryTime, modernFlip, 5+4*((n+99)/100))
			}
		})
	}
}
