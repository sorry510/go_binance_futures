package feature

import (
	"sort"
	"strings"
	"sync"
	"time"

	"go_binance_futures/feature/api/binance"
)

const mainEntryDiagnosticsCap = 200

// MainEntryDiagnostic represents the last REAL decision observed by the
// existing Main strategy loop. Order-submitted is an ACK, never a verified Fill.
// No Binance, ORM, strategy evaluation or history mutation is performed here.
type MainEntryDiagnostic struct {
	Symbol               string `json:"symbol"`
	PositionSide         string `json:"position_side"`
	ReasonCode           string `json:"reason_code"`
	ObservedAt           int64  `json:"observed_at"`
	LastOrderSubmittedAt int64  `json:"last_order_submitted_at"`
}

type MainEntryDiagnosticsSnapshot struct {
	AccountID    string                `json:"account_id"`
	GlobalReason string                `json:"global_reason"`
	ObservedAt   int64                 `json:"observed_at"`
	List         []MainEntryDiagnostic `json:"list"`
	Total        int                   `json:"total"`
	Scope        string                `json:"scope"`
}

type mainEntryDiagnosticStore struct {
	sync.RWMutex
	generation    uint64
	globalReason  string
	observedAt    int64
	rows          map[string]MainEntryDiagnostic
	lastSubmitted map[string]int64
	now           func() time.Time
}

func newMainEntryDiagnosticStore() *mainEntryDiagnosticStore {
	return &mainEntryDiagnosticStore{
		rows: make(map[string]MainEntryDiagnostic), lastSubmitted: make(map[string]int64),
		now: time.Now,
	}
}

var mainEntryDiagnostics = newMainEntryDiagnosticStore()

func mainEntryDiagnosticKey(symbol, side string) string {
	return symbol + "/" + side
}

func (s *mainEntryDiagnosticStore) begin(enabled bool) uint64 {
	s.Lock()
	defer s.Unlock()
	s.generation++
	s.rows = make(map[string]MainEntryDiagnostic)
	s.observedAt = s.now().UnixMilli()
	if enabled {
		s.globalReason = "running"
	} else {
		s.globalReason = "global_off"
	}
	return s.generation
}

func (s *mainEntryDiagnosticStore) global(generation uint64, reason string) {
	s.Lock()
	defer s.Unlock()
	if s.generation != generation {
		return
	}
	s.globalReason = reason
	s.observedAt = s.now().UnixMilli()
}

func (s *mainEntryDiagnosticStore) record(generation uint64, symbol, side, reason string) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" || (side != "LONG" && side != "SHORT") {
		return
	}
	key := mainEntryDiagnosticKey(symbol, side)
	s.Lock()
	defer s.Unlock()
	if s.generation != generation {
		return
	}
	now := s.now().UnixMilli()
	if _, present := s.rows[key]; !present && len(s.rows) >= mainEntryDiagnosticsCap {
		// Bounded memory even with thousands of configured contracts.
		var oldestKey string
		var oldestAt int64
		for k, v := range s.rows {
			if oldestKey == "" || v.ObservedAt < oldestAt || (v.ObservedAt == oldestAt && k < oldestKey) {
				oldestKey, oldestAt = k, v.ObservedAt
			}
		}
		delete(s.rows, oldestKey)
	}
	if reason == "order_submitted" {
		if _, has := s.lastSubmitted[key]; !has && len(s.lastSubmitted) >= mainEntryDiagnosticsCap {
			var oldestKey string
			var oldestAt int64
			for k, v := range s.lastSubmitted {
				if oldestKey == "" || v < oldestAt || (v == oldestAt && k < oldestKey) {
					oldestKey, oldestAt = k, v
				}
			}
			delete(s.lastSubmitted, oldestKey)
		}
		s.lastSubmitted[key] = now
	}
	s.rows[key] = MainEntryDiagnostic{
		Symbol: symbol, PositionSide: side, ReasonCode: reason, ObservedAt: now,
		LastOrderSubmittedAt: s.lastSubmitted[key],
	}
}

func (s *mainEntryDiagnosticStore) recordBoth(generation uint64, symbol, reason string) {
	s.record(generation, symbol, "LONG", reason)
	s.record(generation, symbol, "SHORT", reason)
}

func (s *mainEntryDiagnosticStore) snapshot(symbol, side string) MainEntryDiagnosticsSnapshot {
	s.RLock()
	defer s.RUnlock()
	result := MainEntryDiagnosticsSnapshot{
		AccountID: string(binance.MainAccountID), GlobalReason: s.globalReason, ObservedAt: s.observedAt,
		List: []MainEntryDiagnostic{}, Scope: "current_process_last_main_cycle",
	}
	if s.generation == 0 {
		result.GlobalReason = "not_started"
	}
	for _, item := range s.rows {
		if symbol != "" && item.Symbol != symbol {
			continue
		}
		if side != "" && item.PositionSide != side {
			continue
		}
		result.List = append(result.List, item)
	}
	sort.Slice(result.List, func(i, j int) bool {
		a, b := result.List[i], result.List[j]
		if a.Symbol != b.Symbol {
			return a.Symbol < b.Symbol
		}
		return a.PositionSide < b.PositionSide
	})
	result.Total = len(result.List)
	return result
}

func MainEntryDiagnostics(symbol, side string) MainEntryDiagnosticsSnapshot {
	return mainEntryDiagnostics.snapshot(symbol, side)
}

// Main runner only; a Lead evaluation must never overwrite the Main report.
type mainEntryObserver struct {
	generation uint64
	enabled    bool
}

func beginMainEntryObservation(r *accountTradeRunner) mainEntryObserver {
	if r == nil || r.AccountID != binance.MainAccountID {
		return mainEntryObserver{}
	}
	return mainEntryObserver{generation: mainEntryDiagnostics.begin(r.AllowNewOpens), enabled: r.AllowNewOpens}
}
func (o mainEntryObserver) global(reason string) {
	if o.generation != 0 {
		mainEntryDiagnostics.global(o.generation, reason)
	}
}
func (o mainEntryObserver) record(symbol, side, reason string) {
	if o.generation != 0 && o.enabled {
		mainEntryDiagnostics.record(o.generation, symbol, side, reason)
	}
}
func (o mainEntryObserver) both(symbol, reason string) {
	if o.generation != 0 && o.enabled {
		mainEntryDiagnostics.recordBoth(o.generation, symbol, reason)
	}
}
