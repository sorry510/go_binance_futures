package leadaccount

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

type fakeLeadSnapshotReader struct {
	account   *futures.Account
	positions []*futures.PositionRisk
	orders    []*futures.Order
	mode      *futures.PositionMode
	income    func(startMs, endMs, limit int64) ([]*futures.IncomeHistory, error)
	calls     []string
}

func (f *fakeLeadSnapshotReader) GetFuturesAccountContext(context.Context) (*futures.Account, error) {
	f.calls = append(f.calls, "balance")
	return f.account, nil
}
func (f *fakeLeadSnapshotReader) GetPositionFreshContext(context.Context, binance.PositionParams) ([]*futures.PositionRisk, error) {
	f.calls = append(f.calls, "positions")
	return f.positions, nil
}
func (f *fakeLeadSnapshotReader) GetOpenOrderFreshContext(context.Context, ...string) ([]*futures.Order, error) {
	f.calls = append(f.calls, "orders")
	return f.orders, nil
}
func (f *fakeLeadSnapshotReader) GetPositionModeContext(context.Context) (*futures.PositionMode, error) {
	f.calls = append(f.calls, "mode")
	return f.mode, nil
}
func (f *fakeLeadSnapshotReader) GetIncomeHistoryContext(_ context.Context, startMs, endMs, limit int64) ([]*futures.IncomeHistory, error) {
	f.calls = append(f.calls, "income")
	if f.income != nil {
		return f.income(startMs, endMs, limit)
	}
	return nil, nil
}
func sampleLeadEvidenceReader() *fakeLeadSnapshotReader {
	return &fakeLeadSnapshotReader{
		account: &futures.Account{CanTrade: true, Assets: []*futures.AccountAsset{{Asset: "USDT", WalletBalance: "300", AvailableBalance: "150", MarginBalance: "310"}}},
		positions: []*futures.PositionRisk{
			{Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmt: "1", MarkPrice: "100", UnRealizedProfit: "-1", Leverage: "4"},
			{Symbol: "ETHUSDT", PositionSide: "SHORT", PositionAmt: "0", MarkPrice: "0"},
		},
		orders: []*futures.Order{
			{Symbol: "ETHUSDT", PositionSide: futures.PositionSideTypeShort, Side: futures.SideTypeSell, Type: futures.OrderTypeLimit, Status: futures.OrderStatusTypeNew, OrigQuantity: "3", ExecutedQuantity: "1", Price: "20"},
			{Symbol: "BTCUSDT", PositionSide: futures.PositionSideTypeLong, Side: futures.SideTypeSell, Type: futures.OrderTypeLimit, Status: futures.OrderStatusTypeNew, OrigQuantity: "1", ExecutedQuantity: "0", Price: "105"},
		},
		mode: &futures.PositionMode{DualSidePosition: true},
	}
}
func TestStage43ReadOnlyEvidenceTotalsAndOwnAccount(t *testing.T) {
	now := time.Date(2026, 10, 10, 11, 0, 0, 0, time.UTC)
	reader := sampleLeadEvidenceReader()
	reader.income = func(start, end, limit int64) ([]*futures.IncomeHistory, error) {
		if start != now.Truncate(24*time.Hour).UnixMilli() || end != now.UnixMilli() || limit != 1000 {
			t.Fatalf("bad PNL range: %d %d %d", start, end, limit)
		}
		return []*futures.IncomeHistory{
			{TranID: 101, Time: now.UnixMilli() - 1000, Asset: "USDT", IncomeType: "REALIZED_PNL", Income: "-12.3"},
			{TranID: 102, Time: now.UnixMilli() - 900, Asset: "USDT", IncomeType: "COMMISSION", Income: "-0.5"},
			{TranID: 103, Time: now.UnixMilli() - 800, Asset: "USDT", IncomeType: "FUNDING_FEE", Income: "0.1"},
			{TranID: 104, Time: now.UnixMilli() - 700, Asset: "USDT", IncomeType: "TRANSFER", Income: "50"},
		}, nil
	}
	source := &RiskEvidenceSource{AccountID: binance.LeadAccountID, Reader: reader, Now: func() time.Time { return now }}
	snapshot, err := source.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.PositionsComplete || !snapshot.OrdersComplete || !snapshot.PNLComplete || snapshot.Reconciled || !snapshot.UnknownOrders {
		t.Fatalf("false reconciliation authorization: %+v", snapshot)
	}
	if snapshot.AvailableUSDT != 150 || snapshot.EquityUSDT != 310 || len(snapshot.Positions) != 1 || len(snapshot.PendingOpens) != 1 ||
		snapshot.PendingOpens[0].RemainingQuantity != 2 || math.Abs(snapshot.DailyNetRealizedPNLUSDT+12.7) > 1e-8 {
		t.Fatalf("incorrect Lead evidence %+v", snapshot)
	}
	if strings.Join(reader.calls, ",") != "balance,mode,positions,orders,income" {
		t.Fatalf("unexpected/private calls=%v", reader.calls)
	}
}
func TestStage43EvidenceMissingOrMalformedAlwaysFailsClosed(t *testing.T) {
	base := sampleLeadEvidenceReader()
	tests := []struct {
		name   string
		change func(*fakeLeadSnapshotReader)
	}{
		{"missing_account", func(f *fakeLeadSnapshotReader) { f.account = nil }},
		{"cantrade_disabled", func(f *fakeLeadSnapshotReader) { f.account.CanTrade = false }},
		{"not_hedge", func(f *fakeLeadSnapshotReader) { f.mode.DualSidePosition = false }},
		{"nan_wallet", func(f *fakeLeadSnapshotReader) { f.account.Assets[0].WalletBalance = "NaN" }},
		{"broken_position", func(f *fakeLeadSnapshotReader) { f.positions[0].MarkPrice = "NaN" }},
		{"broken_order", func(f *fakeLeadSnapshotReader) { f.orders[0].ExecutedQuantity = "30" }},
		{"unvalued_market", func(f *fakeLeadSnapshotReader) { f.orders[0].Price = "0" }},
		{"income_bad_page", func(f *fakeLeadSnapshotReader) {
			f.income = func(int64, int64, int64) ([]*futures.IncomeHistory, error) { return nil, errors.New("failed") }
		}},
	}
	_ = base
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := sampleLeadEvidenceReader()
			tt.change(fake)
			source := &RiskEvidenceSource{AccountID: binance.LeadAccountID, Reader: fake}
			snap, err := source.Snapshot(context.Background())
			if err == nil || snap.Reconciled || !snap.UnknownOrders {
				t.Fatalf("bad snapshot unexpectedly usable %+v err=%v", snap, err)
			}
		})
	}
}
func TestStage43IncomePaginationOverlapAndAmbiguity(t *testing.T) {
	start := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC).UnixMilli()
	reader := sampleLeadEvidenceReader()
	page := 0
	reader.income = func(cursor, _, _ int64) ([]*futures.IncomeHistory, error) {
		page++
		rows := make([]*futures.IncomeHistory, 0, 1000)
		if page == 1 {
			for i := int64(1); i <= 1000; i++ {
				rows = append(rows, &futures.IncomeHistory{TranID: i, Time: start + i, Asset: "USDT", IncomeType: "REALIZED_PNL", Income: "-1"})
			}
		} else {
			if cursor != start+1000 {
				t.Fatalf("overlap cursor was %d", cursor)
			}
			rows = append(rows, &futures.IncomeHistory{TranID: 1000, Time: start + 1000, Asset: "USDT", IncomeType: "REALIZED_PNL", Income: "-1"})
			rows = append(rows, &futures.IncomeHistory{TranID: 1001, Time: start + 1001, Asset: "USDT", IncomeType: "COMMISSION", Income: "-2"})
		}
		return rows, nil
	}
	pnl, err := readLeadDailyNetPNL(context.Background(), reader, start, start+2000)
	if err != nil || pnl != -1002 || page != 2 {
		t.Fatalf("pagination missed/duplicated rows pnl=%v pages=%d err=%v", pnl, page, err)
	}
	reader.income = func(_, _, _ int64) ([]*futures.IncomeHistory, error) {
		rows := make([]*futures.IncomeHistory, 1000)
		for i := range rows {
			rows[i] = &futures.IncomeHistory{TranID: int64(i + 1), Time: start, Asset: "USDT", IncomeType: "COMMISSION", Income: "-1"}
		}
		return rows, nil
	}
	if _, err := readLeadDailyNetPNL(context.Background(), reader, start, start+5000); err == nil {
		t.Fatal("ambiguous identical millisecond income page was treated as complete")
	}
}
func TestStage43SnapshotMainCannotBeBound(t *testing.T) {
	account, err := binance.NewAccountClient(binance.MainAccountID, futures.NewClient("fake-main", "fake-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewRiskEvidenceSource(account); err == nil {
		t.Fatal("main account accepted as Lead risk source")
	}
}
