package alertpipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	agentapp "go_binance_futures/agent/app"
	agentevent "go_binance_futures/agent/event"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	marketintelligence "go_binance_futures/service/marketintelligence"
	signalservice "go_binance_futures/service/signal"

	"github.com/beego/beego/v2/core/logs"
)

type ConfigProvider func() models.Config

type RuntimeStatus struct {
	Settings Settings                  `json:"settings"`
	EventBus agentevent.Stats          `json:"event_bus"`
	Signal   signalservice.EngineStats `json:"signal_engine"`
	Pipeline Stats                     `json:"pipeline"`
	Traces   []Trace                   `json:"traces"`
}

var defaultOnce sync.Once
var defaultPipeline *Pipeline
var defaultEngine *signalservice.Engine
var defaultProvider ConfigProvider
var defaultErr error

func StartDefault(ctx context.Context, provider ConfigProvider) error {
	defaultOnce.Do(func() {
		if provider == nil {
			defaultErr = fmt.Errorf("alert pipeline config provider is required")
			return
		}
		defaultProvider = provider
		manager, err := agentapp.DefaultManager()
		if err != nil {
			defaultErr = err
			return
		}
		bus := agentevent.DefaultBus()
		settings := func() Settings { return SettingsFromConfig(provider()) }
		defaultPipeline, err = New(Config{
			Bus: bus, Settings: settings,
			StartTask: manager.Start, GetTask: manager.Get,
			Notify: func(params notify.AgentAlertParams) (int64, error) {
				return notify.GetNotifyChannel().AgentAlert(params)
			},
			QueueSize: 512, Workers: 4, TraceStore: ORMTraceStore{},
		})
		if err != nil {
			defaultErr = err
			return
		}
		defaultEngine, err = signalservice.NewEngine(bus,
			func() signalservice.Settings { return settings().Signal }, func(value signalservice.Signal) bool {
				go persistMarketIntelligenceSignal(value)
				return defaultPipeline.Emit(value)
			})
		if err != nil {
			defaultErr = err
			return
		}
		defaultPipeline.Start(ctx)
		bus.Start(ctx)
	})
	return defaultErr
}

func DefaultStatus(traceLimit int) RuntimeStatus {
	status := RuntimeStatus{EventBus: agentevent.DefaultBus().Stats()}
	if defaultProvider != nil {
		status.Settings = SettingsFromConfig(defaultProvider())
	} else {
		status.Settings = DefaultSettings()
	}
	if defaultEngine != nil {
		status.Signal = defaultEngine.Stats()
	}
	if defaultPipeline != nil {
		status.Pipeline = defaultPipeline.Stats()
		status.Traces = defaultPipeline.Traces(traceLimit)
	}
	return status
}

func DefaultPipeline() *Pipeline {
	return defaultPipeline
}

func persistMarketIntelligenceSignal(value signalservice.Signal) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, _, err := marketintelligence.DefaultService().IngestSignal(ctx, value); err != nil {
		logs.Warning("persist market intelligence signal:", err)
	}
}
