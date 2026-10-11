package leadaccount

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
)

// Management never owns a live broker. All operational mutations remain
// explicitly guarded and do not authorize Stage 7 trading.
var ErrLeadLiveGateLocked = errors.New("stage7_live_authorization_required")
var ErrLeadConfigUnavailable = errors.New("lead_config_storage_unavailable")

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

const ConfigPathEnv = "BINANCE_LEAD_CONFIG_FILE"

type LeadSettings struct {
	AccountID     string     `json:"account_id"`
	Enabled       bool       `json:"enabled"`
	AllowNewOpens bool       `json:"allow_new_opens"`
	Limits        RiskLimits `json:"limits"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func DefaultLeadSettings() LeadSettings {
	return LeadSettings{AccountID: "lead", Enabled: false, AllowNewOpens: false}
}
func ValidateLeadSettings(s LeadSettings) error {
	if s.AccountID != "lead" || s.Enabled || s.AllowNewOpens {
		return ErrLeadLiveGateLocked
	}
	values := []float64{s.Limits.MaxOrderNotionalUSDT, s.Limits.MaxTotalNotionalUSDT, s.Limits.MaxDailyRealizedLossUSDT, s.Limits.MaxDrawdownPct}
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return fmt.Errorf("invalid_lead_risk_limit")
		}
	}
	if s.Limits.MaxDrawdownPct > 100 || s.Limits.MaxPositions < 0 || s.Limits.MaxLosingPositions < 0 || s.Limits.MaxPositions > 1000 || s.Limits.MaxLosingPositions > 1000 {
		return fmt.Errorf("invalid_lead_risk_limit")
	}
	if s.Limits.MaxTotalNotionalUSDT > 0 && s.Limits.MaxOrderNotionalUSDT > s.Limits.MaxTotalNotionalUSDT {
		return fmt.Errorf("order_limit_exceeds_total_limit")
	}
	return nil
}

type LeadConfigStore struct {
	mu   sync.Mutex
	path string
}

func NewLeadConfigStore(path string) (*LeadConfigStore, error) {
	if !filepath.IsAbs(path) || path == "/" || path == "" {
		return nil, ErrLeadConfigUnavailable
	}
	if err := verifyCredentialPath(path); err != nil {
		return nil, ErrLeadConfigUnavailable
	}
	return &LeadConfigStore{path: path}, nil
}
func LeadConfigStoreFromEnv() (*LeadConfigStore, error) {
	p := os.Getenv(ConfigPathEnv)
	if p == "" {
		return nil, ErrLeadConfigUnavailable
	}
	return NewLeadConfigStore(p)
}
func (s *LeadConfigStore) Load() (LeadSettings, error) {
	if s == nil {
		return DefaultLeadSettings(), ErrLeadConfigUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := verifyCredentialPath(s.path); err != nil {
		return DefaultLeadSettings(), ErrLeadConfigUnavailable
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultLeadSettings(), nil
	}
	if err != nil || len(data) > 4096 {
		return DefaultLeadSettings(), ErrLeadConfigUnavailable
	}
	var value LeadSettings
	if err = json.Unmarshal(data, &value); err != nil || ValidateLeadSettings(value) != nil {
		return DefaultLeadSettings(), ErrLeadConfigUnavailable
	}
	return value, nil
}
func (s *LeadConfigStore) Save(value LeadSettings) error {
	if s == nil {
		return ErrLeadConfigUnavailable
	}
	if err := ValidateLeadSettings(value); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := verifyCredentialPath(s.path); err != nil {
		return ErrLeadConfigUnavailable
	}
	value.Enabled = false
	value.AllowNewOpens = false
	value.UpdatedAt = time.Now().UTC()
	b, err := json.Marshal(value)
	if err != nil {
		return ErrLeadConfigUnavailable
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".lead-settings-*")
	if err != nil {
		return ErrLeadConfigUnavailable
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(b)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return ErrLeadConfigUnavailable
	}
	if err = os.Rename(name, s.path); err != nil {
		return ErrLeadConfigUnavailable
	}
	return nil
}

// LeadDurableOpenGate validates disk state on every attempted future live
// entry. It never returns success in Stage 7-1..7-4: Stage7 human
// authorization has not been implemented and any file/read error is denial.
func LeadDurableOpenGate() error {
	if LeadEmergencyPaused() {
		return ErrLeadLiveGateLocked
	}
	store, err := LeadConfigStoreFromEnv()
	if err != nil {
		return ErrLeadConfigUnavailable
	}
	settings, err := store.Load()
	if err != nil {
		return ErrLeadConfigUnavailable
	}
	if settings.AccountID != "lead" || !settings.Enabled || !settings.AllowNewOpens {
		return ErrLeadLiveGateLocked
	}
	return ErrLeadLiveGateLocked
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
