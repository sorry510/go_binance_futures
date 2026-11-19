package orderhistory

import (
	"context"
	"testing"

	"go_binance_futures/models"
)

func TestOracle1ReadOnlyDisplaySeparatesExecutionFromFinancialSettlement(t *testing.T) {
	o, store := historyDB(t)
	if _, err := o.Raw("CREATE TABLE IF NOT EXISTS agent_trade_proposals (proposal_id TEXT PRIMARY KEY,symbol TEXT,side TEXT,status TEXT,updated_at BIGINT)").Exec(); err != nil {
		t.Fatal(err)
	}
	clean := fullCycle(t, o)
	if _, err := o.Raw("UPDATE futures_managed_positions SET settlement_status='incomplete',settlement_processed_at=0,settlement_pnl_sign=0 WHERE id=?", clean).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("UPDATE futures_trade_fills SET commission_asset='BNB',pnl_asset='' WHERE position_id=?", clean).Exec(); err != nil {
		t.Fatal(err)
	}
	// Another incomplete cycle is genuinely missing exchange fills. It
	// remains visible under financial verification, never marked settled.
	broken := positionFixture(t, o, "main", "auto_strategy", "ETHUSDT", "SHORT", "closed", "incomplete", true)
	// Actual execution-reconciliation errors remain on the default page.
	actual := positionFixture(t, o, "main", "auto_strategy", "XRPUSDT", "LONG", "reconcile_required", "open", false)

	old, err := store.Anomalies(context.Background(), AnomalyFilter{AccountID: "main", Page: 1, Limit: 10})
	if err != nil || old.Total != 3 {
		t.Fatalf("legacy API must retain three results, got %+v %v", old, err)
	}

	ui, err := store.Anomalies(context.Background(), AnomalyFilter{
		AccountID: "main", ExcludeSettlementPending: true, Page: 1, Limit: 10,
	})
	if err != nil || ui.Total != 1 || len(ui.List) != 1 ||
		ui.List[0].PositionID == nil || *ui.List[0].PositionID != actual ||
		ui.SettlementPendingTotal != 2 || ui.WithBNBCommissionTotal != 1 {
		t.Fatalf("execution anomalies mixed with financial pending/BNB counts: %+v %v", ui, err)
	}
	settlement, err := store.Anomalies(context.Background(), AnomalyFilter{
		AccountID: "main", Category: "settlement_incomplete",
		ExcludeSettlementPending: true, Page: 1, Limit: 10,
	})
	if err != nil || settlement.Total != 2 || len(settlement.List) != 2 ||
		settlement.SettlementPendingTotal != 2 || settlement.WithBNBCommissionTotal != 1 {
		t.Fatalf("explicit financial review must show BNB and true missing fill, no hiding: %+v %v", settlement, err)
	}
	seen := map[int64]bool{}
	for _, item := range settlement.List {
		if item.PositionID == nil || item.Category != "settlement_incomplete" {
			t.Fatalf("invalid financial classification: %+v", item)
		}
		seen[*item.PositionID] = true
	}
	if !seen[clean] || !seen[broken] {
		t.Fatalf("missing financial cycles: %+v", seen)
	}
	var position models.FuturesManagedPosition
	if err := o.QueryTable(new(models.FuturesManagedPosition)).Filter("id", clean).One(&position); err != nil {
		t.Fatal(err)
	}
	if position.SettlementStatus != "incomplete" || position.SettlementProcessedAt != 0 {
		t.Fatalf("read-only classified a BNB fee as settled: %+v", position)
	}
	var pnlAsset string
	if err := o.Raw("SELECT pnl_asset FROM futures_trade_fills WHERE position_id=? LIMIT 1", clean).QueryRow(&pnlAsset); err != nil {
		t.Fatal(err)
	}
	if pnlAsset != "" {
		t.Fatalf("display-only query wrote an unauthorized PnL denomination %q", pnlAsset)
	}
}

func TestOracle1ReadOnlyCountersAreAccountOwnerSymbolSideScoped(t *testing.T) {
	o, store := historyDB(t)
	if _, err := o.Raw("CREATE TABLE IF NOT EXISTS agent_trade_proposals (proposal_id TEXT PRIMARY KEY,symbol TEXT,side TEXT,status TEXT,updated_at BIGINT)").Exec(); err != nil {
		t.Fatal(err)
	}
	main := positionFixture(t, o, "main", "auto_strategy", "BTCUSDT", "SHORT", "closed", "incomplete", true)
	lead := positionFixture(t, o, "lead", "agent_trade", "BTCUSDT", "LONG", "closed", "incomplete", true)
	for _, id := range []int64{main, lead} {
		account := "main"
		if id == lead {
			account = "lead"
		}
		if _, err := o.Raw("INSERT INTO futures_trade_fills (market,account_id,symbol,order_id,trade_id,position_id,client_order_id,intent,position_side,qty,price,commission,commission_asset,realized_pnl,pnl_asset,trade_time,rest_verified) VALUES ('USDM',?,'BTCUSDT',?,?,?,'local-open','open',?,'1','100','0.002','BNB','0','',1800000002000,1)", account, "readonly-order", account+"-trade", id, map[bool]string{true: "LONG", false: "SHORT"}[id == lead]).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		q            AnomalyFilter
		pending, bnb int64
	}{
		{AnomalyFilter{AccountID: "main", ExcludeSettlementPending: true, Page: 1, Limit: 10}, 1, 1},
		{AnomalyFilter{AccountID: "lead", ExcludeSettlementPending: true, Page: 1, Limit: 10}, 1, 1},
		{AnomalyFilter{AccountID: "main", Owner: "agent_trade", ExcludeSettlementPending: true, Page: 1, Limit: 10}, 0, 0},
		{AnomalyFilter{AccountID: "main", PositionSide: "LONG", ExcludeSettlementPending: true, Page: 1, Limit: 10}, 0, 0},
		{AnomalyFilter{AccountID: "main", Symbol: "ETHUSDT", ExcludeSettlementPending: true, Page: 1, Limit: 10}, 0, 0},
	}
	for _, tc := range cases {
		result, err := store.Anomalies(context.Background(), tc.q)
		if err != nil || result.Total != 0 || result.SettlementPendingTotal != tc.pending || result.WithBNBCommissionTotal != tc.bnb {
			t.Fatalf("account scoped pending counts invalid for %+v: %+v %v", tc.q, result, err)
		}
	}
}
