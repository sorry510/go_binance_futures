package leadaccount

import (
	"context"
	"errors"
	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"testing"
)

func TestLeadSwitchDefaultOffRequiresManualPersistedToggleAndLimits(t *testing.T) {
	prepareLeadSettingsTestDB(t)
	store, err := LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	settings, err := store.Load()
	if err != nil || settings.AllowNewOpens || settings.AccountID != "lead" {
		t.Fatalf("initial config not disabled: %+v %v", settings, err)
	}
	if err := LeadDurableOpenGate(); !errors.Is(err, ErrLeadLiveGateLocked) {
		t.Fatalf("default did not block: %v", err)
	}
	settings.AllowNewOpens = true
	if err := store.Save(settings); err == nil {
		t.Fatal("enabled without risk caps")
	}
	settings.Limits = LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 150}
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	if err := LeadDurableOpenGate(); err != nil {
		t.Fatalf("manually enabled config remained permanently locked: %v", err)
	}
	loaded, err := store.Load()
	if err != nil || !loaded.AllowNewOpens {
		t.Fatalf("enabled state was not persisted: %+v %v", loaded, err)
	}
	loaded.AllowNewOpens = false
	if err := store.Save(loaded); err != nil {
		t.Fatal(err)
	}
	if err := LeadDurableOpenGate(); !errors.Is(err, ErrLeadLiveGateLocked) {
		t.Fatalf("off switch did not stop new entry: %v", err)
	}
	if err := store.Save(LeadSettings{AccountID: "main", AllowNewOpens: true, Limits: settings.Limits}); err == nil {
		t.Fatal("Main scope was accepted as Lead config")
	}
}

func TestLeadSwitchLiveAdapterDoesNotEnableWithoutDedicatedHealthyWS(t *testing.T) {
	// A persisted ON switch alone does not authorize the actual Binance
	// order adapter without the same dedicated and healthy WS client.
	prepareLeadSettingsTestDB(t)
	cfg, err := LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	settings := DefaultLeadSettings()
	settings.AllowNewOpens = true
	settings.Limits = LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 150}
	if err := cfg.Save(settings); err != nil {
		t.Fatal(err)
	}
	a, _ := stage43Adapter(t, true)
	if err := a.ConfigureLiveSession(settings, 5, 5); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("adapter without a healthy dedicated Lead host was authorized: %v", err)
	}
}

func TestLeadSwitchHealthyDedicatedWSAllowsONAndOFFWithoutPortfolioVerify(t *testing.T) {
	prepareLeadSettingsTestDB(t)
	store, err := LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	settings := DefaultLeadSettings()
	settings.AllowNewOpens = true
	settings.Limits = LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 150}
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	a, _ := stage43Adapter(t, true)
	// The old Stage43 fixture uses a separately created whitelist; live
	// sessions must share the EXACT Lead Client instance.
	cache, err := binance.NewLeadSymbolCache(a.account, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.symbols = cache
	mirror := NewLeadAccountMirror()
	gen := mirror.BeginGeneration()
	if err := mirror.Seed(context.Background(), gen, stage51SeedReader()); err != nil {
		t.Fatal(err)
	}
	stream, err := NewLeadUserStream(a.account, mirror)
	if err != nil {
		t.Fatal(err)
	}
	stream.mu.Lock()
	stream.running = true
	stream.currentGeneration = gen
	stream.mu.Unlock()
	host := &LeadReadOnlyHost{stream: stream, running: true}
	leadSetReadOnlyHost(host)
	t.Cleanup(func() { leadSetReadOnlyHost(nil) })
	if err := a.ConfigureLiveSession(settings, 5, 5); err != nil {
		t.Fatalf("manual ON and dedicated WS failed: %v", err)
	}
	a.risk.mu.RLock()
	state := a.risk.state
	a.risk.mu.RUnlock()
	if !state.Stage7Authorized || !state.AllowNewOpens || !state.WSHealthy || !a.guard.permitsOpen() {
		t.Fatalf("runtime ON did not activate the configured Lead-only adapter: %+v", state)
	}
	// The ordinary OFF switch must keep existing Managed exits possible.
	settings.AllowNewOpens = false
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	if err := a.ConfigureLiveSession(settings, 5, 5); err != nil {
		t.Fatal(err)
	}
	a.risk.mu.RLock()
	state = a.risk.state
	a.risk.mu.RUnlock()
	if state.AllowNewOpens || a.guard.permitsOpen() || !a.guard.permitsExitWrite() {
		t.Fatalf("OFF blocked exits or left new entries enabled: %+v", state)
	}
	// Unexpected WS generation loss revokes the next live authorization.
	mirror.Invalidate(gen)
	if err := a.ConfigureLiveSession(settings, 5, 5); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("unhealthy Lead WS passed authorization: %v", err)
	}
}

func TestLeadSwitchRejectsMainLabeledClientEvenWithHealthyLeadMirror(t *testing.T) {
	prepareLeadSettingsTestDB(t)
	settings := DefaultLeadSettings()
	settings.AllowNewOpens = true
	settings.Limits = LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 150}
	store, err := LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	a, _ := stage43Adapter(t, true)
	a.account, _ = binance.NewAccountClient(binance.MainAccountID, futures.NewClient("main-only", "secret"))
	if err := a.ConfigureLiveSession(settings, 5, 5); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("Main account was accepted as Lead: %v", err)
	}
}
