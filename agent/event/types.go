package event

import (
	"fmt"
	"hash/fnv"
	"strings"
	"sync/atomic"
	"time"
)

type Type string

const (
	TypePriceTick   Type = "price_tick"
	TypeKlineClosed Type = "kline_closed"
	TypeLiquidation Type = "liquidation"
	TypeFundingRate Type = "funding_rate"
	TypePosition    Type = "position"
	TypeWsHealth    Type = "ws_health"
)

var eventSequence atomic.Uint64

type Metadata struct {
	EventID   string `json:"event_id"`
	Type      Type   `json:"type"`
	Symbol    string `json:"symbol"`
	EventTime int64  `json:"event_time"`
	Source    string `json:"source"`
}

type Event interface {
	Metadata() Metadata
}

type PriceTickEvent struct {
	Meta             Metadata `json:"meta"`
	Price            float64  `json:"price"`
	PercentChange24h float64  `json:"percent_change_24h"`
}

func (e PriceTickEvent) Metadata() Metadata { return e.Meta }

type LiquidationEvent struct {
	Meta            Metadata `json:"meta"`
	OrderSide       string   `json:"order_side"`
	Price           float64  `json:"price"`
	Quantity        float64  `json:"quantity"`
	Notional        float64  `json:"notional"`
	LiquidationSide string   `json:"liquidation_side"`
}

func (e LiquidationEvent) Metadata() Metadata { return e.Meta }

type WsHealthEvent struct {
	Meta          Metadata `json:"meta"`
	Status        string   `json:"status"`
	NoDataMinutes float64  `json:"no_data_minutes"`
}

func (e WsHealthEvent) Metadata() Metadata { return e.Meta }

type KlineClosedEvent struct {
	Meta     Metadata `json:"meta"`
	Interval string   `json:"interval"`
	Open     float64  `json:"open"`
	High     float64  `json:"high"`
	Low      float64  `json:"low"`
	Close    float64  `json:"close"`
	Volume   float64  `json:"volume"`
}

func (e KlineClosedEvent) Metadata() Metadata { return e.Meta }

type FundingRateEvent struct {
	Meta    Metadata `json:"meta"`
	RatePct float64  `json:"rate_pct"`
	Price   float64  `json:"price"`
}

func (e FundingRateEvent) Metadata() Metadata { return e.Meta }

type PositionEvent struct {
	Meta       Metadata `json:"meta"`
	Side       string   `json:"side"`
	Amount     float64  `json:"amount"`
	EntryPrice float64  `json:"entry_price"`
	MarkPrice  float64  `json:"mark_price"`
}

func (e PositionEvent) Metadata() Metadata { return e.Meta }

func NewMetadata(eventType Type, symbol string, eventTime int64, source string) Metadata {
	if eventTime <= 0 {
		eventTime = time.Now().UnixMilli()
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	return Metadata{
		EventID:   fmt.Sprintf("evt_%d_%d", eventTime, eventSequence.Add(1)),
		Type:      eventType,
		Symbol:    symbol,
		EventTime: eventTime,
		Source:    strings.TrimSpace(source),
	}
}

func NewPriceTick(symbol, source string, eventTime int64, price, percentChange24h float64) PriceTickEvent {
	return PriceTickEvent{
		Meta:             NewMetadata(TypePriceTick, symbol, eventTime, source),
		Price:            price,
		PercentChange24h: percentChange24h,
	}
}

func NewWsHealth(source, status string, eventTime int64, noDataMinutes float64) WsHealthEvent {
	return WsHealthEvent{
		Meta:          NewMetadata(TypeWsHealth, "FuturesWS", eventTime, source),
		Status:        strings.TrimSpace(status),
		NoDataMinutes: noDataMinutes,
	}
}

func NewLiquidation(symbol, source string, eventTime int64, orderSide string, price, quantity, notional float64, liquidationSide string) LiquidationEvent {
	return LiquidationEvent{
		Meta:      NewMetadata(TypeLiquidation, symbol, eventTime, source),
		OrderSide: strings.ToUpper(strings.TrimSpace(orderSide)),
		Price:     price, Quantity: quantity, Notional: notional,
		LiquidationSide: strings.ToLower(strings.TrimSpace(liquidationSide)),
	}
}

func StableID(prefix, key string) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(strings.TrimSpace(key)))
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "evt"
	}
	return fmt.Sprintf("%s_%x", prefix, hash.Sum64())
}
