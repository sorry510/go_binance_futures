package leadaccount

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	binanceapi "go_binance_futures/feature/api/binance"
)

type fakeReader struct {
	status        binanceapi.LeadTraderStatus
	whitelist     []binanceapi.LeadTradingSymbol
	account       *futures.Account
	positions     []*futures.PositionRisk
	orders        []*futures.Order
	mode          *futures.PositionMode
	fail          string
	calls         []string
	mu            sync.Mutex
	statusStarted chan struct{}
	statusRelease chan struct{}
}

func (f *fakeReader) mark(s string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, s)
	if f.fail == s {
		return errors.New("private-secret-value-simulated-error")
	}
	return nil
}
func (f *fakeReader) LeadTraderStatus(context.Context) (binanceapi.LeadTraderStatus, error) {
	if err := f.mark("lead_status"); err != nil {
		return binanceapi.LeadTraderStatus{}, err
	}
	if f.statusStarted != nil {
		close(f.statusStarted)
		<-f.statusRelease
	}
	return f.status, nil
}
func (f *fakeReader) LeadTradingSymbols(context.Context) ([]binanceapi.LeadTradingSymbol, error) {
	if err := f.mark("whitelist"); err != nil {
		return nil, err
	}
	return f.whitelist, nil
}
func (f *fakeReader) GetFuturesAccountContext(context.Context) (*futures.Account, error) {
	if err := f.mark("account"); err != nil {
		return nil, err
	}
	return f.account, nil
}
func (f *fakeReader) GetPositionFreshContext(context.Context, binanceapi.PositionParams) ([]*futures.PositionRisk, error) {
	if err := f.mark("positions"); err != nil {
		return nil, err
	}
	return f.positions, nil
}
func (f *fakeReader) GetOpenOrderFreshContext(context.Context, ...string) ([]*futures.Order, error) {
	if err := f.mark("orders"); err != nil {
		return nil, err
	}
	return f.orders, nil
}
func (f *fakeReader) GetPositionModeContext(context.Context) (*futures.PositionMode, error) {
	if err := f.mark("hedge_mode"); err != nil {
		return nil, err
	}
	return f.mode, nil
}
func validFakeReader() *fakeReader {
	f := &fakeReader{
		whitelist: []binanceapi.LeadTradingSymbol{{Symbol: "BTCUSDT", QuoteAsset: "USDT"}, {Symbol: "ETHUSDT", QuoteAsset: "USDT"}},
		account:   &futures.Account{CanTrade: true, Assets: []*futures.AccountAsset{{Asset: "USDT", WalletBalance: "250", AvailableBalance: "100", MarginBalance: "250"}}},
		positions: []*futures.PositionRisk{{Symbol: "BTCUSDT", PositionAmt: "0.5"}},
		orders:    []*futures.Order{{Symbol: "ETHUSDT"}},
		mode:      &futures.PositionMode{DualSidePosition: true},
	}
	f.status.Success = true
	f.status.Data.IsLeadTrader = true
	return f
}
func hasReason(reasons []string, reason string) bool {
	for _, r := range reasons {
		if r == reason {
			return true
		}
	}
	return false
}
func TestStage41LeadVerificationReadOnlySuccessStillBlocksTrades(t *testing.T) {
	reader := validFakeReader()
	report := verifyReader(context.Background(), reader)
	if !report.LeadIdentityOK || report.USDTWhitelistCount != 2 || !report.ReadOnlyChecksPassed || !report.HedgeMode {
		t.Fatalf("incorrect read-only result: %+v", report)
	}
	if report.USDTBalance != 250 || report.USDTAvailable != 100 || report.PositionCount != 1 || report.OpenOrderCount != 1 {
		t.Fatalf("snapshot incorrect: %+v", report)
	}
	if report.TradingReady || report.PortfolioBindingConfirmed {
		t.Fatal("read-only checks must never authorize live trades")
	}
	if !hasReason(report.BlockingReasons, "portfolio_binding_not_confirmed") || !hasReason(report.BlockingReasons, "risk_gate_not_implemented") || !hasReason(report.BlockingReasons, "stage7_live_authorization_required") {
		t.Fatalf("missing blocking gates: %+v", report)
	}
	if !reflect.DeepEqual(reader.calls, []string{"lead_status", "whitelist", "account", "positions", "orders", "hedge_mode"}) {
		t.Fatalf("read-only calls unexpected: %v", reader.calls)
	}
}
func TestStage41LeadVerificationIdentityFailureDoesNotReadFutures(t *testing.T) {
	f := validFakeReader()
	f.status.Data.IsLeadTrader = false
	report := verifyReader(context.Background(), f)
	if report.ReadOnlyChecksPassed || report.TradingReady || !hasReason(report.BlockingReasons, "not_lead_trader") {
		t.Fatalf("non lead accepted: %+v", report)
	}
	if !reflect.DeepEqual(f.calls, []string{"lead_status"}) {
		t.Fatalf("non lead performed further API calls: %v", f.calls)
	}
}
func TestStage41LeadVerificationErrorsAlwaysFailClosed(t *testing.T) {
	for _, stage := range []string{"lead_status", "whitelist", "account", "positions", "orders", "hedge_mode"} {
		t.Run(stage, func(t *testing.T) {
			f := validFakeReader()
			f.fail = stage
			report := verifyReader(context.Background(), f)
			if report.ReadOnlyChecksPassed || report.TradingReady {
				t.Fatalf("failure accepted: %+v", report)
			}
			if strings.Contains(strings.Join(report.BlockingReasons, ","), "private-secret-value") {
				t.Fatal("private API error leaked")
			}
		})
	}
}
func TestStage41LeadVerificationNonHedgeAndBadBalanceBlocked(t *testing.T) {
	f := validFakeReader()
	f.mode.DualSidePosition = false
	f.account.Assets[0].WalletBalance = "NaN"
	report := verifyReader(context.Background(), f)
	if !hasReason(report.BlockingReasons, "lead_hedge_mode_required") || !hasReason(report.BlockingReasons, "lead_usdt_balance_invalid") {
		t.Fatalf("missing safety gates: %+v", report)
	}
}
func TestStage41ManagerCredentialsRotationInvalidatesReport(t *testing.T) {
	store := newTestEncryptedStore(t)
	reader := validFakeReader()
	var calls atomic.Int32
	manager := newManager(store, func(c Credentials) (AccountReader, error) {
		calls.Add(1)
		if c.APIKey != "key-a" {
			t.Errorf("unexpected key %q", c.APIKey)
		}
		return reader, nil
	})
	if manager.Status().TradingReady {
		t.Fatal("new manager enabled")
	}
	if err := manager.SaveCredentials(Credentials{APIKey: "key-a", APISecret: "secret-a", PortfolioLabel: "untrusted label"}); err != nil {
		t.Fatal(err)
	}
	report, err := manager.VerifyReadOnly(context.Background())
	if err != nil || !report.ReadOnlyChecksPassed {
		t.Fatalf("verify=%+v error=%v", report, err)
	}
	if report.PortfolioLabel != "untrusted label" || report.KeyHint != "****ey-a" || report.TradingReady {
		t.Fatalf("verification incorrectly activated: %+v", report)
	}
	if err := manager.SaveCredentials(Credentials{APIKey: "key-b", APISecret: "secret-b"}); err != nil {
		t.Fatal(err)
	}
	now := manager.Status()
	if now.ReadOnlyChecksPassed || now.TradingReady || !hasReason(now.BlockingReasons, "credentials_not_verified") {
		t.Fatalf("rotation reused stale verification: %+v", now)
	}
	if calls.Load() != 1 {
		t.Fatalf("unexpected API calls=%d", calls.Load())
	}
}
func TestStage41ManagerStaleInFlightVerificationNeverRestoresOldKey(t *testing.T) {
	store := newTestEncryptedStore(t)
	reader := validFakeReader()
	reader.statusStarted = make(chan struct{})
	reader.statusRelease = make(chan struct{})
	manager := newManager(store, func(Credentials) (AccountReader, error) { return reader, nil })
	if err := manager.SaveCredentials(Credentials{APIKey: "old-key", APISecret: "old-secret"}); err != nil {
		t.Fatal(err)
	}
	errChan := make(chan error, 1)
	go func() { _, err := manager.VerifyReadOnly(context.Background()); errChan <- err }()
	<-reader.statusStarted
	if err := manager.SaveCredentials(Credentials{APIKey: "new-key", APISecret: "new-secret"}); err != nil {
		t.Fatal(err)
	}
	close(reader.statusRelease)
	if err := <-errChan; err == nil {
		t.Fatal("old verification succeeded after credential rotation")
	}
	status := manager.Status()
	if status.ReadOnlyChecksPassed || status.TradingReady {
		t.Fatalf("old verification restored readiness: %+v", status)
	}
}

// A failed re-verification must revoke a previous successful read-only state.
func TestStage41ManagerFailedReverificationClearsEarlierSuccess(t *testing.T) {
	store := newTestEncryptedStore(t)
	reader := validFakeReader()
	failFactory := false
	m := newManager(store, func(Credentials) (AccountReader, error) {
		if failFactory {
			return nil, errors.New("secret-and-raw-http-error")
		}
		return reader, nil
	})
	if err := m.SaveCredentials(Credentials{APIKey: "key", APISecret: "secret"}); err != nil {
		t.Fatal(err)
	}
	first, err := m.VerifyReadOnly(context.Background())
	if err != nil || !first.ReadOnlyChecksPassed {
		t.Fatalf("first verification=%v %+v", err, first)
	}
	failFactory = true
	if _, err := m.VerifyReadOnly(context.Background()); err == nil || strings.Contains(err.Error(), "secret-and-raw-http-error") {
		t.Fatalf("raw factory error leaked: %v", err)
	}
	if status := m.Status(); status.ReadOnlyChecksPassed || status.TradingReady ||
		!hasReason(status.BlockingReasons, "lead_read_client_failed") {
		t.Fatalf("stale verification survived failed retry: %+v", status)
	}
}
