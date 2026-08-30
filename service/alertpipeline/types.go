package alertpipeline

import (
	"time"

	"go_binance_futures/models"
	signalservice "go_binance_futures/service/signal"
)

type Settings struct {
	Enabled       bool                   `json:"enabled"`
	AIEnabled     bool                   `json:"ai_enabled"`
	MinSeverity   signalservice.Severity `json:"min_severity"`
	Cooldown      time.Duration          `json:"-"`
	MaxConcurrent int                    `json:"max_concurrent"`
	MaxPerMinute  int                    `json:"max_per_minute"`
	Signal        signalservice.Settings `json:"-"`
}

type Trace struct {
	EventID        string                 `json:"event_id"`
	SignalID       string                 `json:"signal_id"`
	TaskID         string                 `json:"task_id,omitempty"`
	NotificationID int64                  `json:"notification_id,omitempty"`
	Symbol         string                 `json:"symbol"`
	Type           signalservice.Type     `json:"type"`
	Severity       signalservice.Severity `json:"severity"`
	Action         string                 `json:"action,omitempty"`
	Status         string                 `json:"status"`
	Fallback       bool                   `json:"fallback"`
	Error          string                 `json:"error,omitempty"`
	CreatedAt      int64                  `json:"created_at"`
	UpdatedAt      int64                  `json:"updated_at"`
}

func DefaultSettings() Settings {
	return Settings{
		Enabled:       false,
		AIEnabled:     false,
		MinSeverity:   signalservice.SeverityMedium,
		Cooldown:      15 * time.Minute,
		MaxConcurrent: 2,
		MaxPerMinute:  6,
		Signal:        signalservice.DefaultSettings(),
	}
}

func SettingsFromConfig(config models.Config) Settings {
	settings := DefaultSettings()
	settings.Enabled = config.AgentAlertPipelineEnable == 1
	settings.AIEnabled = config.AgentAlertAnalysisEnable == 1
	if value := signalservice.Severity(config.AgentAlertMinSeverity); signalservice.SeverityAtLeast(value, signalservice.SeverityLow) {
		settings.MinSeverity = value
	}
	if config.AgentAlertCooldownSec > 0 {
		settings.Cooldown = time.Duration(config.AgentAlertCooldownSec) * time.Second
	}
	if config.AgentAlertMaxConcurrent > 0 {
		settings.MaxConcurrent = config.AgentAlertMaxConcurrent
	}
	if config.AgentAlertMaxPerMinute > 0 {
		settings.MaxPerMinute = config.AgentAlertMaxPerMinute
	}

	settings.Signal.FastMoveEnabled = config.WsFuturesFastMoveEnable == 1
	if config.WsFuturesFastMoveThreshold > 0 {
		settings.Signal.FastMoveThresholdPct = float64(config.WsFuturesFastMoveThreshold)
	}
	if config.WsFuturesFastMoveRecover > 0 {
		settings.Signal.FastMoveRecoverPct = float64(config.WsFuturesFastMoveRecover)
	}
	if config.WsFuturesFastMoveCooldownSec > 0 {
		settings.Signal.FastMoveCooldown = time.Duration(config.WsFuturesFastMoveCooldownSec) * time.Second
	}
	settings.Signal.FastMoveWindows = signalservice.ParseWindows(config.WsFuturesFastMoveWindows)
	settings.Signal.LiquidationEnabled = config.WsFuturesLiquidationEnable == 1
	if config.WsFuturesLiquidationAlertWindowSec > 0 {
		settings.Signal.LiquidationWindow = time.Duration(config.WsFuturesLiquidationAlertWindowSec) * time.Second
	}
	if config.WsFuturesLiquidationAlertNotionalThreshold > 0 {
		settings.Signal.LiquidationThreshold = config.WsFuturesLiquidationAlertNotionalThreshold
	}
	if config.WsFuturesLiquidationAlertCooldownSec > 0 {
		settings.Signal.LiquidationCooldown = time.Duration(config.WsFuturesLiquidationAlertCooldownSec) * time.Second
	}
	return settings
}

type Stats struct {
	SignalsReceived    uint64 `json:"signals_received"`
	SignalsDropped     uint64 `json:"signals_dropped"`
	ShadowSignals      uint64 `json:"shadow_signals"`
	SuppressedCooldown uint64 `json:"suppressed_cooldown"`
	AITasksStarted     uint64 `json:"ai_tasks_started"`
	AIFallbacks        uint64 `json:"ai_fallbacks"`
	Notifications      uint64 `json:"notifications"`
	ActiveAI           int    `json:"active_ai"`
}
