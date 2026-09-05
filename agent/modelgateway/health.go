package modelgateway

import (
	"sync"
	"time"
)

type healthObservation struct {
	success bool
	kind    string
	latency time.Duration
}

type healthEntry struct {
	observations        []healthObservation
	consecutiveFailures int
	state               string
	openUntil           time.Time
	halfOpenInFlight    bool
}

type HealthSnapshot struct {
	ConfigID            int64   `json:"config_id"`
	State               string  `json:"state"`
	Samples             int     `json:"samples"`
	SuccessRate         float64 `json:"success_rate"`
	Rate429             int     `json:"rate_429"`
	Timeouts            int     `json:"timeouts"`
	ServerErrors        int     `json:"server_errors"`
	AverageLatencyMs    int64   `json:"average_latency_ms"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	OpenUntil           int64   `json:"open_until,omitempty"`
}

type HealthRegistry struct {
	mu      sync.Mutex
	entries map[int64]*healthEntry
}

func NewHealthRegistry() *HealthRegistry {
	return &HealthRegistry{entries: map[int64]*healthEntry{}}
}

func (registry *HealthRegistry) entry(configID int64) *healthEntry {
	item := registry.entries[configID]
	if item == nil {
		item = &healthEntry{state: "closed"}
		registry.entries[configID] = item
	}
	return item
}

func (registry *HealthRegistry) Available(configID int64) bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	item := registry.entry(configID)
	return item.state != "open" || !time.Now().Before(item.openUntil)
}

func (registry *HealthRegistry) Allow(configID int64, cooldown time.Duration) bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	item := registry.entry(configID)
	if item.state == "half_open" {
		return !item.halfOpenInFlight
	}
	if item.state != "open" {
		return true
	}
	if time.Now().Before(item.openUntil) {
		return false
	}
	if item.halfOpenInFlight {
		return false
	}
	item.state = "half_open"
	item.halfOpenInFlight = true
	return true
}

func (registry *HealthRegistry) Record(configID int64, success bool, kind string, latency time.Duration, window, threshold int, cooldown time.Duration) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	item := registry.entry(configID)
	item.observations = append(item.observations, healthObservation{success: success, kind: kind, latency: latency})
	if window < 5 {
		window = 20
	}
	if len(item.observations) > window {
		item.observations = append([]healthObservation(nil), item.observations[len(item.observations)-window:]...)
	}
	if success {
		item.consecutiveFailures = 0
		item.state = "closed"
		item.openUntil = time.Time{}
		item.halfOpenInFlight = false
		return
	}
	if !isCircuitBreakerFailure(kind) {
		item.halfOpenInFlight = false
		if item.state == "half_open" {
			item.consecutiveFailures = 0
			item.state = "closed"
			item.openUntil = time.Time{}
		}
		return
	}
	item.consecutiveFailures++
	item.halfOpenInFlight = false
	if threshold < 1 {
		threshold = 3
	}
	if item.state == "half_open" || item.consecutiveFailures >= threshold {
		item.state = "open"
		item.openUntil = time.Now().Add(cooldown)
	}
}

func (registry *HealthRegistry) Snapshots() []HealthSnapshot {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	result := make([]HealthSnapshot, 0, len(registry.entries))
	for configID, item := range registry.entries {
		result = append(result, snapshot(configID, item))
	}
	return result
}

func snapshot(configID int64, item *healthEntry) HealthSnapshot {
	value := HealthSnapshot{ConfigID: configID, State: item.state, Samples: len(item.observations), ConsecutiveFailures: item.consecutiveFailures}
	if value.State == "" {
		value.State = "closed"
	}
	if !item.openUntil.IsZero() {
		value.OpenUntil = item.openUntil.UnixMilli()
	}
	var successes int
	var latency time.Duration
	for _, observation := range item.observations {
		if observation.success {
			successes++
		}
		switch observation.kind {
		case "429":
			value.Rate429++
		case "timeout":
			value.Timeouts++
		case "5xx":
			value.ServerErrors++
		}
		latency += observation.latency
	}
	if value.Samples > 0 {
		value.SuccessRate = float64(successes) / float64(value.Samples)
		value.AverageLatencyMs = latency.Milliseconds() / int64(value.Samples)
	}
	return value
}

func (registry *HealthRegistry) Snapshot(configID int64) HealthSnapshot {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return snapshot(configID, registry.entry(configID))
}

func isCircuitBreakerFailure(kind string) bool {
	switch kind {
	case "429", "timeout", "network", "5xx":
		return true
	default:
		return false
	}
}
