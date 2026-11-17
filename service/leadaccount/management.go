package leadaccount

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
)

// Management never owns a live broker. The default-OFF persisted setting
// authorizes new-entry attempts only after dedicated Lead session/risk checks.
var ErrLeadLiveGateLocked = errors.New("stage7_live_authorization_required")
var ErrLeadConfigUnavailable = errors.New("lead_config_db_unavailable")

// Stage 6 emergency pause is a one-way process-local latch. It is asserted
// BEFORE any audit/file I/O and cannot be cleared via any management API.
// This is fail-safe even without configured stores, but not durable across
// process restarts. Any future Stage 7 production runner MUST check this
// latch and the persisted halt state before evaluating new Lead entries.
var emergencyLeadPause atomic.Bool

func LatchLeadEmergencyPause() {
	emergencyLeadPause.Store(true)
}

func LeadEmergencyPaused() bool {
	return emergencyLeadPause.Load()
}

// Lead-01 has one configuration format. Unfinished old settings are not
// migrated or read; the trading identity remains independently verified.
type LeadRiskConfig struct {
	MaxOrderNotionalUSDT     float64 `json:"max_order_notional_usdt"`
	MaxTotalNotionalUSDT     float64 `json:"max_total_notional_usdt"`
	MaxDailyRealizedLossUSDT float64 `json:"max_daily_realized_loss_usdt"`
}

type LeadSettings struct {
	AccountID     string         `json:"account_id"`
	AllowNewOpens bool           `json:"allow_new_opens"`
	Limits        LeadRiskConfig `json:"limits"`
	UpdatedAt     time.Time      `json:"updated_at,omitempty"`
}

func DefaultLeadSettings() LeadSettings {
	return LeadSettings{AccountID: "lead"}
}

// Exposure caps must both be configured. Optional daily loss zero disables
// that extra check, but cannot bypass account identity or exchange controls.
func LeadRiskLimitsReady(limits LeadRiskConfig) bool {
	return riskFinitePositive(limits.MaxOrderNotionalUSDT) &&
		riskFinitePositive(limits.MaxTotalNotionalUSDT) &&
		limits.MaxOrderNotionalUSDT <= limits.MaxTotalNotionalUSDT
}

func ValidateLeadSettings(s LeadSettings) error {
	if s.AccountID != "lead" {
		return ErrLeadLiveGateLocked
	}
	if s.AllowNewOpens && !LeadRiskLimitsReady(s.Limits) {
		return fmt.Errorf("lead_risk_limits_required_to_enable")
	}
	for _, value := range []float64{
		s.Limits.MaxOrderNotionalUSDT,
		s.Limits.MaxTotalNotionalUSDT,
		s.Limits.MaxDailyRealizedLossUSDT,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("invalid_lead_risk_limit")
		}
	}
	// The initial default (both zero) is safe for read-only state; only
	// configured positive caps may be accepted by the HTTP save endpoint.
	if (s.Limits.MaxOrderNotionalUSDT == 0) != (s.Limits.MaxTotalNotionalUSDT == 0) {
		return fmt.Errorf("lead_notional_limits_incomplete")
	}
	if s.Limits.MaxTotalNotionalUSDT > 0 && s.Limits.MaxOrderNotionalUSDT > s.Limits.MaxTotalNotionalUSDT {
		return fmt.Errorf("order_limit_exceeds_total_limit")
	}
	return nil
}

// LeadTradeSettingsDDL is the single-table Lead settings schema introduced in v26.
// This same schema is created by command/sql/version/26.sql during sync db.
const LeadTradeSettingsDDL = `CREATE TABLE IF NOT EXISTS lead_trade_settings (
    account_id VARCHAR(16) NOT NULL PRIMARY KEY,
    allow_new_opens SMALLINT NOT NULL DEFAULT 0,
    max_order_notional_usdt DECIMAL(20,8) NOT NULL DEFAULT 0,
    max_total_notional_usdt DECIMAL(20,8) NOT NULL DEFAULT 0,
    max_daily_realized_loss_usdt DECIMAL(20,8) NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT 0
)`

// LeadConfigStore uses the existing application's registered default database.
// There is no JSON risk file, environment variable, implicit migration or
// bypass when the DB is unavailable. Missing rows are default-OFF.
type LeadConfigStore struct{}

func LeadConfigStoreFromDB() (*LeadConfigStore, error) {
	if _, err := orm.GetDB("default"); err != nil {
		return nil, ErrLeadConfigUnavailable
	}
	return &LeadConfigStore{}, nil
}

func (s *LeadConfigStore) Load() (LeadSettings, error) {
	if s == nil {
		return DefaultLeadSettings(), ErrLeadConfigUnavailable
	}
	var account string
	var enabled int
	var order, total, daily float64
	var updated int64
	err := orm.NewOrm().Raw(
		"SELECT account_id, allow_new_opens, max_order_notional_usdt, max_total_notional_usdt, max_daily_realized_loss_usdt, updated_at FROM lead_trade_settings WHERE account_id = ?",
		"lead",
	).QueryRow(&account, &enabled, &order, &total, &daily, &updated)
	if errors.Is(err, orm.ErrNoRows) {
		return DefaultLeadSettings(), nil
	}
	if err != nil {
		return DefaultLeadSettings(), fmt.Errorf("%w: %v", ErrLeadConfigUnavailable, err)
	}
	result := LeadSettings{
		AccountID: account, AllowNewOpens: enabled == 1,
		Limits: LeadRiskConfig{
			MaxOrderNotionalUSDT:     order,
			MaxTotalNotionalUSDT:     total,
			MaxDailyRealizedLossUSDT: daily,
		},
		UpdatedAt: time.UnixMilli(updated).UTC(),
	}
	if (enabled != 0 && enabled != 1) || ValidateLeadSettings(result) != nil {
		return DefaultLeadSettings(), ErrLeadConfigUnavailable
	}
	return result, nil
}

func (s *LeadConfigStore) Save(value LeadSettings) error {
	if s == nil {
		return ErrLeadConfigUnavailable
	}
	if err := ValidateLeadSettings(value); err != nil {
		return err
	}
	tx, err := orm.NewOrm().Begin()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLeadConfigUnavailable, err)
	}
	defer tx.Rollback()
	var count int
	if err = tx.Raw("SELECT COUNT(*) FROM lead_trade_settings WHERE account_id = ?", "lead").QueryRow(&count); err != nil {
		return fmt.Errorf("%w: %v", ErrLeadConfigUnavailable, err)
	}
	allow := 0
	if value.AllowNewOpens {
		allow = 1
	}
	updated := time.Now().UTC().UnixMilli()
	if count == 0 {
		_, err = tx.Raw(
			"INSERT INTO lead_trade_settings (account_id, allow_new_opens, max_order_notional_usdt, max_total_notional_usdt, max_daily_realized_loss_usdt, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			"lead", allow, value.Limits.MaxOrderNotionalUSDT, value.Limits.MaxTotalNotionalUSDT, value.Limits.MaxDailyRealizedLossUSDT, updated,
		).Exec()
	} else if count == 1 {
		_, err = tx.Raw(
			"UPDATE lead_trade_settings SET allow_new_opens=?, max_order_notional_usdt=?, max_total_notional_usdt=?, max_daily_realized_loss_usdt=?, updated_at=? WHERE account_id=?",
			allow, value.Limits.MaxOrderNotionalUSDT, value.Limits.MaxTotalNotionalUSDT, value.Limits.MaxDailyRealizedLossUSDT, updated, "lead",
		).Exec()
	} else {
		return ErrLeadConfigUnavailable
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLeadConfigUnavailable, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", ErrLeadConfigUnavailable, err)
	}
	return nil
}

// LeadDurableOpenGate validates the persisted user switch before every live
// entry. Missing/invalid config and the emergency pause fail closed.
func LeadDurableOpenGate() error {
	if LeadEmergencyPaused() {
		return ErrLeadLiveGateLocked
	}
	store, err := LeadConfigStoreFromDB()
	if err != nil {
		return ErrLeadConfigUnavailable
	}
	settings, err := store.Load()
	if err != nil {
		return ErrLeadConfigUnavailable
	}
	if settings.AccountID != "lead" || !settings.AllowNewOpens || !LeadRiskLimitsReady(settings.Limits) {
		return ErrLeadLiveGateLocked
	}
	// Explicit persisted user switch, disabled by default. No HTTP handler,
	// in-memory flag or old config format can silently enable live entries.
	return nil
}

type LeadDataView struct {
	AccountID  string                          `json:"account_id"`
	Positions  []models.FuturesManagedPosition `json:"positions"`
	Orders     []models.FuturesManagedOrder    `json:"orders"`
	DataSource string                          `json:"data_source"`
	Reconciled bool                            `json:"reconciled"`
}

// Only the immutable Lead account-bound Ownership service is queried. The
// view is local DB evidence, NOT proof of current Binance balance/positions.
func ListLeadManaged(ctx context.Context, owner string, limit int) (LeadDataView, error) {
	value := LeadDataView{AccountID: "lead", Positions: []models.FuturesManagedPosition{}, Orders: []models.FuturesManagedOrder{}, DataSource: "ownership_database", Reconciled: false}
	s, err := futuresownership.BindAccount(binance.LeadAccountID)
	if err != nil {
		return value, err
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	positions, err := s.ListPositions(ctx, owner, false)
	if err != nil {
		return value, err
	}
	orders, err := s.ListOrders(ctx, owner, limit)
	if err != nil {
		return value, err
	}
	value.Positions = positions
	value.Orders = orders
	return value, nil
}
