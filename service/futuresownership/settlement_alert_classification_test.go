package futuresownership

import (
	"context"
	"strconv"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

func TestHistoricalClosedSnapshotPendingIsTrackedWithoutRepeatedAlarm(t *testing.T) {
	pos := models.FuturesManagedPosition{
		ID: 12345, AccountID: "main", Owner: OwnerAutoStrategy,
		Symbol: "UNIUSDT", PositionSide: "SHORT",
		Status: PositionClosed, SettlementStatus: SettlementIncomplete,
		ManagedQty: 0, ClosedAt: 1_800_000_000_000,
	}
	defer clearIncompleteAlarm(pos.AccountID, pos.ID)
	const snapshotReason = "symbol-scoped fresh position snapshot missing or ambiguous"
	const bnbReason = "fully verified BNB commission requires FX valuation"
	now := pos.ClosedAt + 20*60*1000
	for _, reason := range []string{snapshotReason, bnbReason} {
		for k := 0; k < 4; k++ {
			summary := ReconcileSummary{}
			alarmIncompletePosition(pos, now+int64(k)*15*60*1000, reason, &summary)
			if summary.IncompleteAged != 1 || summary.OldestIncompleteAgeSec < 20*60 {
				t.Fatalf("historical backlog must remain visible in monitoring: %+v", summary)
			}
		}
		incompleteAlertMu.Lock()
		_, alerted := incompleteAlertAt["main/12345"]
		incompleteAlertMu.Unlock()
		if alerted {
			t.Fatalf("historical financial backlog emitted a warning: %s", reason)
		}
	}
	// A real exposure or order/Fill failure retains the Warning path.
	for _, reason := range []string{"exchange still has nonzero position", "closed-cycle trade backfill: REST missing",
		"empty snapshot and unresolved exchange orders", "commission in a different or unknown asset requires FX valuation"} {
		clearIncompleteAlarm(pos.AccountID, pos.ID)
		summary := ReconcileSummary{}
		alarmIncompletePosition(pos, now, reason, &summary)
		incompleteAlertMu.Lock()
		_, alerted := incompleteAlertAt["main/12345"]
		incompleteAlertMu.Unlock()
		if !alerted || summary.IncompleteAged != 1 {
			t.Fatalf("genuine issue was suppressed: %s %+v", reason, summary)
		}
	}
	// Even the formerly quiet snapshot reason must warn if position isn't truly
	// a closed, zero-quantity, financially incomplete historical cycle.
	for _, modify := range []func(*models.FuturesManagedPosition){
		func(p *models.FuturesManagedPosition) { p.AccountID = "lead" },
		func(p *models.FuturesManagedPosition) { p.Owner = OwnerAgentTrade },
		func(p *models.FuturesManagedPosition) { p.Status = PositionActive },
		func(p *models.FuturesManagedPosition) { p.ManagedQty = 1 },
		func(p *models.FuturesManagedPosition) { p.SettlementStatus = SettlementOpen },
	} {
		changed := pos
		changed.ID++
		modify(&changed)
		clearIncompleteAlarm(changed.AccountID, changed.ID)
		summary := ReconcileSummary{}
		alarmIncompletePosition(changed, now, snapshotReason, &summary)
		incompleteAlertMu.Lock()
		_, alerted := incompleteAlertAt[changed.AccountID+"/"+strconv.FormatInt(changed.ID, 10)]
		incompleteAlertMu.Unlock()
		clearIncompleteAlarm(changed.AccountID, changed.ID)
		if !alerted {
			t.Fatalf("non-historical snapshot problem silenced: %+v", changed)
		}
	}
}

func TestVerifiedBNBFeeIsFinancialBacklogOnlyAfterFullFillProof(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	svc := testService()
	pos := settlementFilled(t, svc, IntentOpen, "bnblog_open", "1101", 1, 100, "0", "0.002")
	pos = settlementFilled(t, svc, IntentClose, "bnblog_close", "1201", 1, 110, "10", "0.003")
	o := orm.NewOrm()
	if _, err := o.Raw("UPDATE futures_trade_fills SET commission_asset='BNB',pnl_asset='' WHERE order_id IN ('1101','1201')").Exec(); err != nil {
		t.Fatal(err)
	}
	check := func(want string) {
		t.Helper()
		_, _, _, reason, err := verifiedSettlementFacts(o, pos, "USDT")
		if err != nil || reason != want {
			t.Fatalf("reason=%q, want %q, err=%v", reason, want, err)
		}
	}
	const bnbReason = "fully verified BNB commission requires FX valuation"
	check(bnbReason)
	before := loadSettlementConfig(t)
	result, err := svc.settleVerifiedFlat(context.Background(), pos.ID, "USDT")
	if err != nil || result.Status != SettlementIncomplete || result.Reason != bnbReason {
		t.Fatalf("BNB fee was incorrectly certified: %+v %v", result, err)
	}
	after := loadSettlementPosition(t, pos.ID)
	cfg := loadSettlementConfig(t)
	if after.SettlementStatus != SettlementIncomplete || after.SettlementProcessedAt != 0 ||
		cfg.SettlementConsecutiveLosses != before.SettlementConsecutiveLosses ||
		cfg.SettlementScaleBalance != before.SettlementScaleBalance {
		t.Fatalf("estimate affected durable settlement or risk: position=%+v cfg=%+v", after, cfg)
	}
	// Any missing REST provenance must take precedence over the BNB fee warning
	// downgrade: it is a real execution verification problem.
	if _, err := o.Raw("UPDATE futures_trade_fills SET rest_verified=0 WHERE order_id='1201'").Exec(); err != nil {
		t.Fatal(err)
	}
	check("trade fee and realized PnL not confirmed by Binance REST")
	if _, err := o.Raw("UPDATE futures_trade_fills SET rest_verified=1, commission_asset='MISSING' WHERE order_id='1201'").Exec(); err != nil {
		t.Fatal(err)
	}
	check("commission in a different or unknown asset requires FX valuation")
	if _, err := o.Raw("UPDATE futures_trade_fills SET commission_asset='BNB',qty='0.9' WHERE order_id='1201'").Exec(); err != nil {
		t.Fatal(err)
	}
	check("some order fills are missing or exceed cumulative execution")
}
