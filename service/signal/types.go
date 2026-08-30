package signal

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Type string

type Severity string

const (
	TypeFastMove              Type = "fast_move"
	TypeVolumeSpike           Type = "volume_spike"
	TypeOISpike               Type = "oi_spike"
	TypeLiquidationSpike      Type = "liquidation_spike"
	TypeFundingExtreme        Type = "funding_extreme"
	TypeBreakout              Type = "breakout"
	TypeBreakdown             Type = "breakdown"
	TypeShortSqueezeCandidate Type = "short_squeeze_candidate"
	TypeLongSqueezeCandidate  Type = "long_squeeze_candidate"
)

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

var signalSequence atomic.Uint64

type Evidence struct {
	Source  string `json:"source"`
	Finding string `json:"finding"`
}

type Signal struct {
	SignalID  string             `json:"signal_id"`
	EventID   string             `json:"event_id"`
	Symbol    string             `json:"symbol"`
	Type      Type               `json:"type"`
	Severity  Severity           `json:"severity"`
	Window    string             `json:"window"`
	Metrics   map[string]float64 `json:"metrics"`
	Labels    map[string]string  `json:"labels"`
	Evidence  []Evidence         `json:"evidence"`
	CreatedAt int64              `json:"created_at"`
}

type Window struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"-"`
}

type Settings struct {
	FastMoveEnabled         bool
	FastMoveThresholdPct    float64
	FastMoveRecoverPct      float64
	FastMoveCooldown        time.Duration
	FastMoveWindows         []Window
	FastMoveExcludedSymbols map[string]bool
	LiquidationEnabled      bool
	LiquidationSymbol       string
	LiquidationWindow       time.Duration
	LiquidationThreshold    float64
	LiquidationCooldown     time.Duration
}

func DefaultSettings() Settings {
	return Settings{
		FastMoveEnabled:      true,
		FastMoveThresholdPct: 20,
		FastMoveRecoverPct:   18,
		FastMoveCooldown:     15 * time.Minute,
		FastMoveWindows: []Window{
			{Name: "3m", Duration: 3 * time.Minute},
			{Name: "5m", Duration: 5 * time.Minute},
			{Name: "10m", Duration: 10 * time.Minute},
		},
		FastMoveExcludedSymbols: map[string]bool{"USDCUSDT": true, "XAGUSDT": true, "XAUUSDT": true},
		LiquidationEnabled:      true,
		LiquidationSymbol:       "BTCUSDT",
		LiquidationWindow:       time.Minute,
		LiquidationThreshold:    5_000_000,
		LiquidationCooldown:     5 * time.Minute,
	}
}

func NewSignal(eventID, symbol string, signalType Type, severity Severity, window string) Signal {
	now := time.Now().UnixMilli()
	return Signal{
		SignalID:  fmt.Sprintf("sig_%d_%d", now, signalSequence.Add(1)),
		EventID:   strings.TrimSpace(eventID),
		Symbol:    strings.ToUpper(strings.TrimSpace(symbol)),
		Type:      signalType,
		Severity:  severity,
		Window:    strings.TrimSpace(window),
		Metrics:   map[string]float64{},
		Labels:    map[string]string{},
		Evidence:  []Evidence{},
		CreatedAt: now,
	}
}

func ParseWindows(raw string) []Window {
	if strings.TrimSpace(raw) == "" {
		return append([]Window(nil), DefaultSettings().FastMoveWindows...)
	}
	seen := map[string]bool{}
	result := make([]Window, 0)
	for _, item := range strings.Split(raw, ",") {
		name := strings.TrimSpace(item)
		if name == "" || seen[name] {
			continue
		}
		duration, ok := parseWindow(name)
		if !ok {
			continue
		}
		seen[name] = true
		result = append(result, Window{Name: name, Duration: duration})
	}
	if len(result) == 0 {
		return append([]Window(nil), DefaultSettings().FastMoveWindows...)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Duration < result[j].Duration })
	return result
}

func parseWindow(value string) (time.Duration, bool) {
	if len(value) < 2 {
		return 0, false
	}
	amount, err := strconv.Atoi(value[:len(value)-1])
	if err != nil || amount <= 0 {
		return 0, false
	}
	switch value[len(value)-1] {
	case 's':
		return time.Duration(amount) * time.Second, true
	case 'm':
		return time.Duration(amount) * time.Minute, true
	case 'h':
		return time.Duration(amount) * time.Hour, true
	default:
		return 0, false
	}
}

func SeverityAtLeast(value, minimum Severity) bool {
	return severityRank(value) >= severityRank(minimum)
}

func severityRank(value Severity) int {
	switch value {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

func SeverityForRatio(ratio float64) Severity {
	if ratio >= 2 {
		return SeverityCritical
	}
	if ratio >= 1.5 {
		return SeverityHigh
	}
	return SeverityMedium
}
