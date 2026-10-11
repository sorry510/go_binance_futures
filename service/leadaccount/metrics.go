package leadaccount

import (
	"time"

	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/binanceapiusage"
)

type LeadUsageMetrics struct {
	AccountID             binance.AccountID `json:"account_id"`
	WindowSeconds         int64             `json:"window_seconds"`
	Requests              int64             `json:"requests"`
	EstimatedWeight       int64             `json:"estimated_weight"`
	Count429              int64             `json:"count_429"`
	Count418              int64             `json:"count_418"`
	WSUpdates             uint64            `json:"ws_updates"`
	RESTSnapshots         uint64            `json:"rest_snapshots"`
	AlertQueuePresent     bool              `json:"alert_queue_present"`
	AlertQueueDepth       int               `json:"alert_queue_depth"`
	AlertQueueCapacity    int               `json:"alert_queue_capacity"`
	AlertsDropped         uint64            `json:"alerts_dropped"`
	CriticalAlertsDropped uint64            `json:"critical_alerts_dropped"`
	// A real saved-REST metric needs endpoint-by-endpoint before/after evidence.
	// Do not present observed WS updates as fabricated API request savings.
	UnverifiedSavedRequests int64 `json:"-"`
}

// LeadUsageSnapshot reads the existing shared V4 collector, retaining the
// unified IP budget. It does not create another market-feed subscription.
func LeadUsageSnapshot(collector *binanceapiusage.Collector, mirror *LeadAccountMirror, now time.Time) LeadUsageMetrics {
	return LeadUsageSnapshotWithAlerts(collector, mirror, nil, now)
}

// LeadUsageSnapshotWithAlerts exposes queue backpressure together with the
// existing Lead metrics. The queue is never shared with Main. Stage 6 may
// present these values, but this method does not register a production route.
func LeadUsageSnapshotWithAlerts(collector *binanceapiusage.Collector, mirror *LeadAccountMirror, alerts *LeadAlertQueue, now time.Time) LeadUsageMetrics {
	out := LeadUsageMetrics{AccountID: binance.LeadAccountID, WindowSeconds: 300}
	if collector == nil {
		collector = binanceapiusage.Default()
	}
	snapshot := collector.Snapshot(now)
	for _, src := range snapshot.Sources5m {
		if src.Source != "lead_trading" && src.Source != "lead_trading_sapi" {
			continue
		}
		out.Requests += src.Count
		out.EstimatedWeight += src.EstimatedWeight
		out.Count429 += src.Count429
		out.Count418 += src.Count418
	}
	if mirror != nil {
		status := mirror.Status()
		out.WSUpdates = status.Metrics.WSUpdates
		out.RESTSnapshots = status.Metrics.RestSnapshots
	}
	if alerts != nil {
		out.AlertQueuePresent = true
		state := alerts.Snapshot()
		out.AlertQueueDepth = state.Depth
		out.AlertQueueCapacity = state.Capacity
		out.AlertsDropped = state.Dropped
		out.CriticalAlertsDropped = state.DroppedCritical
	}
	return out
}
