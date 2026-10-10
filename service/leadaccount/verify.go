package leadaccount

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binanceapi "go_binance_futures/feature/api/binance"
)

// AccountReader contains READ operations only. No order, leverage, margin,
// cancellation, or stream-start service is accessible from the verifier.
type AccountReader interface {
	LeadTraderStatus(context.Context) (binanceapi.LeadTraderStatus, error)
	LeadTradingSymbols(context.Context) ([]binanceapi.LeadTradingSymbol, error)
	GetFuturesAccountContext(context.Context) (*futures.Account, error)
	GetPositionFreshContext(context.Context, binanceapi.PositionParams) ([]*futures.PositionRisk, error)
	GetOpenOrderFreshContext(context.Context, ...string) ([]*futures.Order, error)
	GetPositionModeContext(context.Context) (*futures.PositionMode, error)
}

type readerFactory func(Credentials) (AccountReader, error)
type credentialSource interface {
	Load() (Credentials, error)
	Save(Credentials) error
}

type VerificationReport struct {
	AccountID                 string    `json:"account_id"`
	PortfolioLabel            string    `json:"portfolio_label"`
	KeyHint                   string    `json:"key_hint"`
	CheckedAt                 time.Time `json:"checked_at"`
	LeadIdentityOK            bool      `json:"lead_identity_ok"`
	USDTWhitelistCount        int       `json:"usdt_whitelist_count"`
	PortfolioBindingConfirmed bool      `json:"portfolio_binding_confirmed"`
	FuturesCanTrade           bool      `json:"futures_can_trade"`
	HedgeMode                 bool      `json:"hedge_mode"`
	USDTBalance               float64   `json:"usdt_balance"`
	USDTAvailable             float64   `json:"usdt_available"`
	PositionCount             int       `json:"position_count"`
	OpenOrderCount            int       `json:"open_order_count"`
	ReadOnlyChecksPassed      bool      `json:"read_only_checks_passed"`
	TradingReady              bool      `json:"trading_ready"`
	BlockingReasons           []string  `json:"blocking_reasons"`
}

func initialReport() VerificationReport {
	return VerificationReport{
		AccountID:       string(binanceapi.LeadAccountID),
		BlockingReasons: []string{"lead_disabled", "new_opens_paused", "credentials_not_verified", "portfolio_binding_not_confirmed", "risk_gate_not_implemented", "ws_not_ready", "stage7_live_authorization_required"},
	}
}

// Manager is deliberately not registered globally. Stage 4-1 has no
// background polling, HTTP routes or live trade runner. Stage 6 may provide
// a strictly-authenticated UI/service facade around it.
type Manager struct {
	mu         sync.RWMutex
	verifyMu   sync.Mutex
	store      credentialSource
	factory    readerFactory
	generation uint64
	last       VerificationReport
}

func NewManager(store *EncryptedFileStore) *Manager {
	return newManager(store, func(c Credentials) (AccountReader, error) {
		return binanceapi.NewLeadAccountClient(c.APIKey, c.APISecret, nil)
	})
}
func newManager(store credentialSource, factory readerFactory) *Manager {
	return &Manager{store: store, factory: factory, last: initialReport()}
}
func (m *Manager) Status() VerificationReport {
	if m == nil {
		return initialReport()
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := m.last
	s.BlockingReasons = append([]string(nil), s.BlockingReasons...)
	return s
}
func (m *Manager) SaveCredentials(c Credentials) error {
	if m == nil || m.store == nil || m.factory == nil {
		return ErrCredentialsUnavailable
	}
	if !c.Valid() {
		return ErrCredentialsUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// All Stage 4-1 configurations stay stopped. Rotation cannot silently
	// activate a Lead trading loop, and invalidates every earlier verification.
	if err := m.store.Save(c); err != nil {
		return err
	}
	m.generation++
	m.last = initialReport()
	m.last.KeyHint = c.MaskedKey()
	m.last.PortfolioLabel = c.PortfolioLabel
	return nil
}

// VerifyReadOnly is explicitly invoked by an authorized future API handler.
// It does not run on application startup, use Main's credentials, or make any
// Binance mutation. Its report NEVER enables live trade, even on success.
func (m *Manager) VerifyReadOnly(ctx context.Context) (VerificationReport, error) {
	if m == nil || m.store == nil || m.factory == nil {
		return initialReport(), ErrCredentialsUnavailable
	}
	m.verifyMu.Lock()
	defer m.verifyMu.Unlock()
	m.mu.RLock()
	generation := m.generation
	m.mu.RUnlock()
	// An already canceled verification must revoke the previous successful
	// read-only snapshot, not call Binance or return an ambiguous nil error.
	if err := ctx.Err(); err != nil {
		m.invalidateVerification(generation, verificationContextReason(err))
		return m.Status(), err
	}
	c, err := m.store.Load()
	if err != nil {
		m.invalidateVerification(generation, "credentials_load_failed")
		return initialReport(), err
	}
	reader, err := m.factory(c)
	if err != nil {
		m.invalidateVerification(generation, "lead_read_client_failed")
		return initialReport(), errors.New("lead read-only client initialization failed")
	}
	if reader == nil {
		m.invalidateVerification(generation, "lead_read_client_failed")
		return initialReport(), errors.New("lead read-only account client unavailable")
	}
	requestCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	report := verifyReader(requestCtx, reader)
	// The reader normally honors the context, but even a broken/slow
	// implementation returning a seemingly successful snapshot after
	// cancellation must never get a pass or produce a nil error.
	verificationErr := requestCtx.Err()
	if verificationErr != nil {
		report.ReadOnlyChecksPassed = false
		report.BlockingReasons = append(report.BlockingReasons, verificationContextReason(verificationErr))
	}
	report.KeyHint = c.MaskedKey()
	report.PortfolioLabel = c.PortfolioLabel // untrusted display-only
	m.mu.Lock()
	defer m.mu.Unlock()
	if generation != m.generation {
		return initialReport(), errors.New("lead credentials changed during verification")
	}
	m.last = report
	return m.StatusUnlocked(), verificationErr
}

// These are fixed, non-sensitive reason codes that a future UI may display.
func verificationContextReason(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "verification_deadline_exceeded"
	}
	return "verification_canceled"
}

func (m *Manager) invalidateVerification(generation uint64, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.generation != generation {
		return
	}
	oldHint, oldLabel := m.last.KeyHint, m.last.PortfolioLabel
	m.last = initialReport()
	m.last.KeyHint = oldHint
	m.last.PortfolioLabel = oldLabel
	m.last.BlockingReasons = append(m.last.BlockingReasons, reason)
}
func (m *Manager) StatusUnlocked() VerificationReport {
	r := m.last
	r.BlockingReasons = append([]string(nil), r.BlockingReasons...)
	return r
}

func verifyReader(ctx context.Context, reader AccountReader) VerificationReport {
	r := initialReport()
	r.CheckedAt = time.Now().UTC()
	r.BlockingReasons = []string{}
	block := func(reason string) { r.BlockingReasons = append(r.BlockingReasons, reason) }
	// Do not continue into /fapi with a key that is not a confirmed Lead.
	status, err := reader.LeadTraderStatus(ctx)
	if err != nil {
		block("lead_identity_request_failed")
		return finishVerification(r)
	}
	if !status.Success {
		block("lead_identity_response_invalid")
		return finishVerification(r)
	}
	if !status.Data.IsLeadTrader {
		block("not_lead_trader")
		return finishVerification(r)
	}
	r.LeadIdentityOK = true
	whitelist, err := reader.LeadTradingSymbols(ctx)
	if err != nil {
		block("lead_symbol_request_failed")
	} else {
		unique := map[string]struct{}{}
		for _, symbol := range whitelist {
			name := strings.ToUpper(strings.TrimSpace(symbol.Symbol))
			quote := strings.ToUpper(strings.TrimSpace(symbol.QuoteAsset))
			if strings.HasSuffix(name, "USDT") && (quote == "" || quote == "USDT") {
				unique[name] = struct{}{}
			}
		}
		r.USDTWhitelistCount = len(unique)
		if len(unique) == 0 {
			block("lead_usdt_whitelist_empty")
		}
	}
	account, err := reader.GetFuturesAccountContext(ctx)
	if err != nil || account == nil {
		block("lead_futures_account_request_failed")
	} else {
		r.FuturesCanTrade = account.CanTrade
		if !account.CanTrade {
			block("lead_futures_trade_permission_missing")
		}
		// Portfolio is USDT-single-asset. Do not pass an account view whose margin
		// data is incomplete or not parseable.
		usdtFound := false
		for _, asset := range account.Assets {
			if asset == nil {
				continue
			}
			if asset.Asset == "USDT" {
				balance, b1 := validAmount(asset.WalletBalance)
				available, b2 := validAmount(asset.AvailableBalance)
				if !b1 || !b2 {
					block("lead_usdt_balance_invalid")
				} else {
					r.USDTBalance = balance
					r.USDTAvailable = available
					usdtFound = true
				}
			} else if asset.Asset != "" {
				if b, ok := validAmount(asset.MarginBalance); ok && b > 0 {
					block("multi_asset_account_unconfirmed")
					break
				}
			}
		}
		if !usdtFound {
			block("lead_usdt_asset_missing")
		}
		if account.MultiAssetsMargin {
			block("lead_multi_asset_margin_mode")
		}
	}
	positions, err := reader.GetPositionFreshContext(ctx, binanceapi.PositionParams{})
	if err != nil {
		block("lead_positions_request_failed")
	} else {
		for _, p := range positions {
			if p != nil {
				qty, valid := strconv.ParseFloat(p.PositionAmt, 64)
				if !validParse(qty, valid) {
					block("lead_position_quantity_invalid")
					break
				}
				if math.Abs(qty) > 1e-9 {
					r.PositionCount++
				}
			}
		}
	}
	orders, err := reader.GetOpenOrderFreshContext(ctx)
	if err != nil {
		block("lead_open_orders_request_failed")
	} else {
		r.OpenOrderCount = len(orders)
	}
	mode, err := reader.GetPositionModeContext(ctx)
	if err != nil || mode == nil {
		block("lead_position_mode_request_failed")
	} else {
		r.HedgeMode = mode.DualSidePosition
		if !r.HedgeMode {
			block("lead_hedge_mode_required")
		}
	}
	return finishVerification(r)
}
func validAmount(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f, err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) && f >= 0
}
func validParse(f float64, err error) bool { return err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) }
func finishVerification(r VerificationReport) VerificationReport {
	r.ReadOnlyChecksPassed = len(r.BlockingReasons) == 0
	// This service cannot prove /fapi is scoped to the intended Portfolio from
	// SAPI userStatus alone. Real portfolio view comparison is a separate gate.
	r.BlockingReasons = append(r.BlockingReasons,
		"portfolio_binding_not_confirmed", "risk_gate_not_implemented", "ws_not_ready", "stage7_live_authorization_required", "lead_disabled", "new_opens_paused")
	r.PortfolioBindingConfirmed = false
	r.TradingReady = false
	return r
}

// A helper for future application bootstrap. It only constructs local objects,
// performs no remote calls, and will never change Main's FutureEnable switch.
func NewManagerFromEnv() (*Manager, error) {
	store, err := NewEncryptedFileStoreFromEnv()
	if err != nil {
		return nil, fmt.Errorf("lead credential store unavailable: %w", err)
	}
	return NewManager(store), nil
}

// Ensure the live factory never takes an externally supplied shared HTTP
// client or accesses main config credentials.
var _ AccountReader = (*binanceapi.AccountClient)(nil)
