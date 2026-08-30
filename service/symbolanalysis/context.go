package symbolanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/models"
	liquidationservice "go_binance_futures/service/liquidation"
	marketservice "go_binance_futures/service/market"
	symbolservice "go_binance_futures/service/symbol"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Dependencies struct {
	GetSymbol                 func(context.Context, string) (models.Symbols, error)
	GetMarketCondition        func(context.Context) (marketservice.Condition, error)
	GetKlines                 func(context.Context, string, string, int) ([]*futures.Kline, error)
	GetFundingRate            func(context.Context, string) ([]*futures.PremiumIndex, error)
	GetOpenInterest           func(context.Context, string) (*futures.OpenInterest, error)
	GetOpenInterestStatistics func(context.Context, string, string, int) ([]*futures.OpenInterestStatistic, error)
	GetTakerLongShortRatio    func(context.Context, string, string, int) ([]*futures.TakerLongShortRatio, error)
	GetDepth                  func(context.Context, string, int) (*futures.DepthResponse, error)
	ListLiquidations          func(context.Context, liquidationservice.ListOptions) (liquidationservice.ListResult, error)
	ListHistory               func(context.Context, HistoryListOptions) (HistoryListResult, error)
	Now                       func() time.Time
}
type Snapshot struct {
	Symbol           string  `json:"symbol"`
	Price            float64 `json:"price"`
	PercentChange24h float64 `json:"percent_change_24h"`
	Open24h          float64 `json:"open_24h"`
	High24h          float64 `json:"high_24h"`
	Low24h           float64 `json:"low_24h"`
	QuoteVolume24h   float64 `json:"quote_volume_24h"`
	TradeCount24h    float64 `json:"trade_count_24h"`
	UpdatedAtMs      int64   `json:"updated_at_ms"`
	DataAgeMs        int64   `json:"data_age_ms"`
}
type KlineFeature struct {
	Interval          string  `json:"interval"`
	Samples           int     `json:"samples"`
	LatestClosedPrice float64 `json:"latest_closed_price"`
	ChangePct         float64 `json:"change_pct"`
	RangePct          float64 `json:"range_pct"`
	VolatilityPct     float64 `json:"volatility_pct"`
	TakerBuySharePct  float64 `json:"taker_buy_share_pct"`
	Trend             string  `json:"trend"`
}
type FundingFeature struct {
	RatePct         float64 `json:"rate_pct"`
	MarkPrice       float64 `json:"mark_price"`
	NextFundingTime int64   `json:"next_funding_time"`
}
type OpenInterestFeature struct {
	Current   float64 `json:"current"`
	ChangePct float64 `json:"change_pct"`
	Period    string  `json:"period"`
}
type TakerFeature struct {
	BuySellRatio float64 `json:"buy_sell_ratio"`
	BuySharePct  float64 `json:"buy_share_pct"`
	Period       string  `json:"period"`
	Timestamp    uint64  `json:"timestamp"`
}
type DepthFeature struct {
	BidNotional  float64 `json:"bid_notional"`
	AskNotional  float64 `json:"ask_notional"`
	ImbalancePct float64 `json:"imbalance_pct"`
	SpreadPct    float64 `json:"spread_pct"`
}
type LiquidationFeature struct {
	WindowMinutes   int     `json:"window_minutes"`
	Count           int     `json:"count"`
	LongNotional    float64 `json:"long_notional"`
	ShortNotional   float64 `json:"short_notional"`
	LargestNotional float64 `json:"largest_notional"`
}
type PreviousAnalysis struct {
	TaskID          string          `json:"task_id"`
	CreatedAt       int64           `json:"created_at"`
	Direction       string          `json:"direction"`
	Confidence      float64         `json:"confidence"`
	MarketCondition int             `json:"market_condition"`
	AnalysisPrice   float64         `json:"analysis_price"`
	CurrentPrice    float64         `json:"current_price"`
	PriceChangePct  float64         `json:"price_change_pct"`
	Summary         string          `json:"summary"`
	Plan            json.RawMessage `json:"plan,omitempty"`
}
type Context struct {
	Symbol           string                   `json:"symbol"`
	AsOf             string                   `json:"as_of"`
	Snapshot         Snapshot                 `json:"snapshot"`
	MarketCondition  *marketservice.Condition `json:"market_condition,omitempty"`
	Klines           []KlineFeature           `json:"klines"`
	Funding          *FundingFeature          `json:"funding,omitempty"`
	OpenInterest     *OpenInterestFeature     `json:"open_interest,omitempty"`
	Taker            *TakerFeature            `json:"taker,omitempty"`
	Depth            *DepthFeature            `json:"depth,omitempty"`
	Liquidations     *LiquidationFeature      `json:"liquidations,omitempty"`
	PreviousAnalyses []PreviousAnalysis       `json:"previous_analyses"`
	DataMissing      []string                 `json:"data_missing"`
}

func DefaultDependencies() Dependencies {
	s := symbolservice.Service{}
	m := marketservice.Service{}
	l := liquidationservice.Service{}
	h := HistoryService{}
	return Dependencies{GetSymbol: s.Snapshot, GetMarketCondition: m.MarketCondition, GetKlines: m.Klines, GetFundingRate: m.FundingRate, GetOpenInterest: m.OpenInterest, GetOpenInterestStatistics: m.OpenInterestStatistics, GetTakerLongShortRatio: m.TakerLongShortRatio, GetDepth: m.Depth, ListLiquidations: l.List, ListHistory: h.List, Now: time.Now}
}
func Build(ctx context.Context, symbol string, d Dependencies) (Context, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if !strings.HasSuffix(symbol, "USDT") {
		return Context{}, fmt.Errorf("symbol must be a USDT futures contract")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	item, err := d.GetSymbol(ctx, symbol)
	if err != nil {
		return Context{}, err
	}
	now := d.Now().UTC()
	price, e := strconv.ParseFloat(item.Close, 64)
	if e != nil || price <= 0 {
		return Context{}, fmt.Errorf("invalid symbol snapshot price")
	}
	out := Context{Symbol: symbol, AsOf: now.Format(time.RFC3339), Snapshot: Snapshot{Symbol: symbol, Price: price, PercentChange24h: item.PercentChange, Open24h: f(item.Open), High24h: f(item.High), Low24h: f(item.Low), QuoteVolume24h: item.QuoteVolume, TradeCount24h: item.TradeCount, UpdatedAtMs: item.UpdateTime, DataAgeMs: max64(0, now.UnixMilli()-item.UpdateTime)}, Klines: []KlineFeature{}, PreviousAnalyses: []PreviousAnalysis{}, DataMissing: []string{}}
	if d.ListHistory != nil {
		history, historyErr := d.ListHistory(ctx, HistoryListOptions{Symbol: symbol, Status: "succeeded", Page: 1, Limit: 3})
		if historyErr != nil {
			out.DataMissing = append(out.DataMissing, "analysis_history")
		} else {
			out.PreviousAnalyses = compactPreviousAnalyses(history.List)
		}
	}
	if out.Snapshot.DataAgeMs > (3 * time.Minute).Milliseconds() {
		out.DataMissing = append(out.DataMissing, "symbol_snapshot_stale")
	}
	intervals := []string{"5m", "15m", "1h", "4h"}
	krs := make([][]*futures.Kline, 4)
	kes := make([]error, 4)
	var cond marketservice.Condition
	var ce error
	var fund []*futures.PremiumIndex
	var fe error
	var oi *futures.OpenInterest
	var ois []*futures.OpenInterestStatistic
	var oiErr error
	var oiStatsErr error
	var tak []*futures.TakerLongShortRatio
	var te error
	var dep *futures.DepthResponse
	var de error
	var liq liquidationservice.ListResult
	var le error
	var wg sync.WaitGroup
	wg.Add(10)
	go func() { defer wg.Done(); cond, ce = d.GetMarketCondition(ctx) }()
	for i, it := range intervals {
		go func(i int, it string) { defer wg.Done(); krs[i], kes[i] = d.GetKlines(ctx, symbol, it, 60) }(i, it)
	}
	go func() { defer wg.Done(); fund, fe = d.GetFundingRate(ctx, symbol) }()
	go func() {
		defer wg.Done()
		oi, oiErr = d.GetOpenInterest(ctx, symbol)
		ois, oiStatsErr = d.GetOpenInterestStatistics(ctx, symbol, "5m", 6)
	}()
	go func() { defer wg.Done(); tak, te = d.GetTakerLongShortRatio(ctx, symbol, "5m", 6) }()
	go func() { defer wg.Done(); dep, de = d.GetDepth(ctx, symbol, 20) }()
	go func() {
		defer wg.Done()
		liq, le = d.ListLiquidations(ctx, liquidationservice.ListOptions{Symbol: symbol, StartTime: now.Add(-time.Hour).UnixMilli(), EndTime: now.UnixMilli(), Limit: 200, DefaultLimit: 200, MaxLimit: 200})
	}()
	wg.Wait()
	if ce == nil {
		out.MarketCondition = &cond
	} else {
		out.DataMissing = append(out.DataMissing, "market_condition")
	}
	for i := range intervals {
		if kes[i] == nil {
			if x, e := summarizeKlines(intervals[i], krs[i]); e == nil {
				out.Klines = append(out.Klines, x)
			} else {
				out.DataMissing = append(out.DataMissing, "kline_"+intervals[i])
			}
		} else {
			out.DataMissing = append(out.DataMissing, "kline_"+intervals[i])
		}
	}
	if fe == nil && len(fund) > 0 {
		out.Funding = &FundingFeature{RatePct: f(fund[0].LastFundingRate) * 100, MarkPrice: f(fund[0].MarkPrice), NextFundingTime: fund[0].NextFundingTime}
	} else {
		out.DataMissing = append(out.DataMissing, "funding_rate")
	}
	if oiErr == nil && oi != nil {
		x := &OpenInterestFeature{Current: f(oi.OpenInterest), Period: "5m"}
		if oiStatsErr == nil {
			sort.Slice(ois, func(i, j int) bool { return ois[i].Timestamp < ois[j].Timestamp })
			if len(ois) > 1 {
				x.ChangePct = pct(f(ois[len(ois)-1].SumOpenInterest), f(ois[0].SumOpenInterest))
			}
		} else {
			out.DataMissing = append(out.DataMissing, "open_interest_change")
		}
		out.OpenInterest = x
	} else {
		out.DataMissing = append(out.DataMissing, "open_interest")
	}
	if te == nil && len(tak) > 0 {
		sort.Slice(tak, func(i, j int) bool { return tak[i].Timestamp < tak[j].Timestamp })
		x := tak[len(tak)-1]
		b, s := f(x.BuyVol), f(x.SellVol)
		out.Taker = &TakerFeature{BuySellRatio: f(x.BuySellRatio), BuySharePct: share(b, b+s), Period: "5m", Timestamp: x.Timestamp}
	} else {
		out.DataMissing = append(out.DataMissing, "taker_ratio")
	}
	if de == nil && dep != nil {
		out.Depth = summarizeDepth(dep)
	} else {
		out.DataMissing = append(out.DataMissing, "depth")
	}
	if le == nil {
		out.Liquidations = summarizeLiquidations(liq)
	} else {
		out.DataMissing = append(out.DataMissing, "liquidations")
	}
	return out, nil
}

func compactPreviousAnalyses(items []HistoryItem) []PreviousAnalysis {
	result := make([]PreviousAnalysis, 0, len(items))
	for _, item := range items {
		result = append(result, PreviousAnalysis{
			TaskID: item.TaskID, CreatedAt: item.CreatedAt, Direction: item.Direction,
			Confidence: item.Confidence, MarketCondition: item.MarketCondition,
			AnalysisPrice: item.AnalysisPrice, CurrentPrice: item.CurrentPrice,
			PriceChangePct: item.PriceChangePct, Summary: item.Summary,
			Plan: append(json.RawMessage(nil), item.Result...),
		})
	}
	return result
}

func summarizeKlines(interval string, ks []*futures.Kline) (KlineFeature, error) {
	if len(ks) < 3 {
		return KlineFeature{}, fmt.Errorf("insufficient klines")
	}
	xs := ks[1:]
	if len(xs) > 20 {
		xs = xs[:20]
	}
	latest := f(xs[0].Close)
	oldest := f(xs[len(xs)-1].Close)
	hi, lo := 0.0, math.MaxFloat64
	returns := []float64{}
	q, tb := 0.0, 0.0
	for i, k := range xs {
		h, l := f(k.High), f(k.Low)
		if h > hi {
			hi = h
		}
		if l < lo {
			lo = l
		}
		q += f(k.QuoteAssetVolume)
		tb += f(k.TakerBuyQuoteAssetVolume)
		if i+1 < len(xs) {
			returns = append(returns, pct(f(xs[i].Close), f(xs[i+1].Close)))
		}
	}
	ch := pct(latest, oldest)
	tr := "sideways"
	if ch > 0.5 {
		tr = "bullish"
	} else if ch < -0.5 {
		tr = "bearish"
	}
	return KlineFeature{Interval: interval, Samples: len(xs), LatestClosedPrice: latest, ChangePct: ch, RangePct: pct(hi, lo), VolatilityPct: stddev(returns), TakerBuySharePct: share(tb, q), Trend: tr}, nil
}
func summarizeDepth(d *futures.DepthResponse) *DepthFeature {
	b, a := 0.0, 0.0
	for _, x := range d.Bids {
		b += f(x.Price) * f(x.Quantity)
	}
	for _, x := range d.Asks {
		a += f(x.Price) * f(x.Quantity)
	}
	sp := 0.0
	if len(d.Bids) > 0 && len(d.Asks) > 0 {
		sp = pct(f(d.Asks[0].Price), f(d.Bids[0].Price))
	}
	return &DepthFeature{BidNotional: b, AskNotional: a, ImbalancePct: share(b-a, b+a), SpreadPct: sp}
}
func summarizeLiquidations(r liquidationservice.ListResult) *LiquidationFeature {
	x := &LiquidationFeature{WindowMinutes: 60, Count: len(r.List)}
	for _, v := range r.List {
		if v.Side == "SELL" {
			x.LongNotional += v.Notional
		} else if v.Side == "BUY" {
			x.ShortNotional += v.Notional
		}
		if v.Notional > x.LargestNotional {
			x.LargestNotional = v.Notional
		}
	}
	return x
}
func f(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
func pct(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return (a - b) / b * 100
}
func share(a, t float64) float64 {
	if t == 0 {
		return 0
	}
	return a / t * 100
}
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
func stddev(x []float64) float64 {
	if len(x) == 0 {
		return 0
	}
	m := 0.0
	for _, v := range x {
		m += v
	}
	m /= float64(len(x))
	s := 0.0
	for _, v := range x {
		d := v - m
		s += d * d
	}
	return math.Sqrt(s / float64(len(x)))
}
