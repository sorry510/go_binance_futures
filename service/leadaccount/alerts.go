package leadaccount

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"

	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/notify"
)

var ErrLeadAlertBackpressure = errors.New("Lead fault alert queue is full")

type LeadAlertQueue struct {
	events             chan LeadFaultEvent
	dropped            atomic.Uint64
	droppedCritical    atomic.Uint64
	criticalDropLogged atomic.Bool
	warningDropLogged  atomic.Bool
	mu                 sync.Mutex
	running            bool
}

func NewLeadAlertQueue(capacity int) *LeadAlertQueue {
	if capacity <= 0 {
		capacity = 64
	}
	if capacity > 2048 {
		capacity = 2048
	}
	return &LeadAlertQueue{events: make(chan LeadFaultEvent, capacity)}
}

// Publish is bounded, nonblocking and safe from the trading hot path.
// The guard retains safety state even when an alert cannot be queued.
func (q *LeadAlertQueue) Publish(event LeadFaultEvent) error {
	if q == nil || event.AccountID != binance.LeadAccountID {
		return ErrLeadAlertBackpressure
	}
	if _, _, ok := faultDefinition(event.Code); !ok {
		return ErrLeadAlertBackpressure
	}
	select {
	case q.events <- event:
		return nil
	default:
		q.dropped.Add(1)
		if event.Severity == SeverityCritical {
			q.droppedCritical.Add(1)
			if q.criticalDropLogged.CompareAndSwap(false, true) {
				log.Print("lead_trading: critical fault alert dropped due to queue backpressure; Lead safety gate remains active")
			}
		} else if q.warningDropLogged.CompareAndSwap(false, true) {
			log.Print("lead_trading: noncritical fault alert dropped due to queue backpressure; Lead safety gate remains active")
		}
		return ErrLeadAlertBackpressure
	}
}
func (q *LeadAlertQueue) Dropped() uint64 {
	if q == nil {
		return 0
	}
	return q.dropped.Load()
}

// LeadAlertQueueSnapshot is a lock-free, read-only local telemetry projection.
// Stage 6 can expose it through an authenticated Lead-only status endpoint.
// It is not a production route and cannot activate an account.
type LeadAlertQueueSnapshot struct {
	Depth           int    `json:"depth"`
	Capacity        int    `json:"capacity"`
	Dropped         uint64 `json:"dropped"`
	DroppedCritical uint64 `json:"dropped_critical"`
}

func (q *LeadAlertQueue) Snapshot() LeadAlertQueueSnapshot {
	if q == nil {
		return LeadAlertQueueSnapshot{}
	}
	return LeadAlertQueueSnapshot{
		Depth: len(q.events), Capacity: cap(q.events),
		Dropped: q.dropped.Load(), DroppedCritical: q.droppedCritical.Load(),
	}
}

type LeadAlertSend func(context.Context, LeadFaultEvent) error

// Run is explicitly opt-in. There is no global alert goroutine and no
// blocking IO on RecordFault. Callers own cancellation and worker lifetime.
func (q *LeadAlertQueue) Run(ctx context.Context, send LeadAlertSend) error {
	if q == nil || send == nil {
		return ErrLeadAlertBackpressure
	}
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return errors.New("Lead alert worker already running")
	}
	q.running = true
	q.mu.Unlock()
	defer func() { q.mu.Lock(); q.running = false; q.mu.Unlock() }()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-q.events:
			// Handler errors never loosen the breaker.
			_ = send(ctx, event)
		}
	}
}

// SendLeadAlertToConfiguredChannel routes fixed sanitized account reasons into
// the EXISTING alert channel. The channel is only opened when the explicit
// alert worker runs, and it cannot submit/cancel Binance orders.
func SendLeadAlertToConfiguredChannel(ctx context.Context, event LeadFaultEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.AccountID != binance.LeadAccountID {
		return ErrLeadAlertBackpressure
	}
	if _, _, ok := faultDefinition(event.Code); !ok {
		return ErrLeadAlertBackpressure
	}
	sev := string(event.Severity)
	if sev != "critical" {
		sev = "warning"
	}
	_, err := notify.GetNotifyChannel().AgentAlert(notify.AgentAlertParams{
		Module: "lead_trading", Source: "LEAD", SignalType: "account_fault",
		Severity: sev, Summary: string(event.Action) + ": " + string(event.Code),
		EventTime: event.OccurredAt.UnixMilli(), Risks: []string{string(event.Code)},
	})
	return err
}
