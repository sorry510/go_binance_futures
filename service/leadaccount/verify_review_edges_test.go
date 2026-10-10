package leadaccount

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	binanceapi "go_binance_futures/feature/api/binance"
)

func TestStage41ReviewReportIsolationAndJSON(t *testing.T) {
	store := newTestEncryptedStore(t)
	mgr := newManager(store, func(Credentials) (AccountReader, error) { return validFakeReader(), nil })
	if err := mgr.SaveCredentials(Credentials{APIKey: "audit-key-ABCD", APISecret: "audit-only-secret"}); err != nil {
		t.Fatal(err)
	}
	report, err := mgr.VerifyReadOnly(context.Background())
	if err != nil || !report.ReadOnlyChecksPassed || report.TradingReady || report.PortfolioBindingConfirmed {
		t.Fatalf("incorrect read-only state %+v: %v", report, err)
	}
	if report.KeyHint != "****ABCD" {
		t.Fatalf("key hint %q", report.KeyHint)
	}
	report.BlockingReasons[0] = "tampered"
	saved := mgr.Status()
	if saved.BlockingReasons[0] == "tampered" {
		t.Fatal("report mutated internal state")
	}
	saved.BlockingReasons[0] = "also-tampered"
	if mgr.Status().BlockingReasons[0] == "also-tampered" {
		t.Fatal("Status is not copied")
	}
	out, err := json.Marshal(mgr.Status())
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"audit-key-ABCD", "audit-only-secret", "tampered"} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf("unsafe report JSON")
		}
	}
}
func TestStage41ReviewWhitelistFilterAndMultiAsset(t *testing.T) {
	f := validFakeReader()
	f.whitelist = []binanceapi.LeadTradingSymbol{
		{Symbol: "BTCUSDT", QuoteAsset: "USDT"},
		{Symbol: "btcusdt", QuoteAsset: "USDT"},
		{Symbol: "ETHUSDT"},
		{Symbol: "SOLUSDC", QuoteAsset: "USDC"},
		{Symbol: "ETHBTC", QuoteAsset: "BTC"},
		{Symbol: "XRPUSDT", QuoteAsset: "USDC"},
	}
	r := verifyReader(context.Background(), f)
	if !r.ReadOnlyChecksPassed || r.USDTWhitelistCount != 2 {
		t.Fatalf("invalid whitelist filtering: %+v", r)
	}
	f.account.Assets = append(f.account.Assets, &futures.AccountAsset{Asset: "BTC", MarginBalance: "1"})
	r = verifyReader(context.Background(), f)
	if r.ReadOnlyChecksPassed || !hasReason(r.BlockingReasons, "multi_asset_account_unconfirmed") {
		t.Fatalf("non-USDT margin permitted: %+v", r)
	}
	f.account.Assets = f.account.Assets[:1]
	f.account.MultiAssetsMargin = true
	r = verifyReader(context.Background(), f)
	if r.ReadOnlyChecksPassed || !hasReason(r.BlockingReasons, "lead_multi_asset_margin_mode") {
		t.Fatalf("multi-asset setting permitted: %+v", r)
	}
}
