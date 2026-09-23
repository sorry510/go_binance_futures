package scanner

import (
	"context"
	"sort"
	"strings"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

const (
	DefaultSmartLocalV2PoolSize       = 60
	DefaultSmartLocalV2Limit          = DefaultSmartLocalV2PoolSize
	DefaultSmartLocalV2CooldownMinute = int64(5)
	DefaultSmartLocalV2MaxDataAgeMs   = int64(30 * time.Second / time.Millisecond)
	DefaultSmartLocalV2MinQuoteVolume = 5_000_000
)

type SmartLocalV2Mode string

const (
	SmartLocalV2ModeTrade SmartLocalV2Mode = "trade"
	SmartLocalV2ModeTest  SmartLocalV2Mode = "test"
)

type SmartLocalV2Options struct {
	Limit           int     `json:"limit"`
	CooldownMinute  int64   `json:"cooldown_minute"`
	MaxDataAgeMs    int64   `json:"max_data_age_ms"`
	MinQuoteVolume  float64 `json:"min_quote_volume"`
	IncludeExcluded bool    `json:"include_excluded"`
}

type SmartLocalV2Result struct {
	PrefilterResult
	Selector string `json:"selector"`

	// NextBatch and Rotation are preview/debug fields. The Candidate Service
	// itself leaves them empty; preview callers must populate them with Peek
	// so reading the preview never advances a live/test round-robin scope.
	NextBatch []PrefilterCandidate      `json:"next_batch,omitempty"`
	Rotation  SmartLocalV2RotationState `json:"rotation,omitempty"`
}

func SmartLocalV2(ctx context.Context, opts SmartLocalV2Options) (*SmartLocalV2Result, error) {
	return SmartLocalV2ForMode(ctx, SmartLocalV2ModeTrade, opts)
}

func SmartLocalV2ForMode(ctx context.Context, mode SmartLocalV2Mode, opts SmartLocalV2Options) (*SmartLocalV2Result, error) {
	return smartLocalV2WithOrm(ctx, orm.NewOrm(), mode, opts)
}

func smartLocalV2WithOrm(ctx context.Context, o orm.Ormer, mode SmartLocalV2Mode, opts SmartLocalV2Options) (*SmartLocalV2Result, error) {
	var symbols []*models.Symbols
	if _, err := o.QueryTable(new(models.Symbols)).OrderBy("ID").All(&symbols); err != nil {
		return nil, err
	}
	return smartLocalV2FromSymbolsWithOrm(ctx, o, symbols, mode, opts)
}

func SmartLocalV2FromSymbolsWithLocalCooldown(ctx context.Context, symbols []*models.Symbols, opts SmartLocalV2Options) (*SmartLocalV2Result, error) {
	return SmartLocalV2FromSymbolsWithModeCooldown(ctx, symbols, SmartLocalV2ModeTrade, opts)
}

func SmartLocalV2FromSymbolsWithModeCooldown(ctx context.Context, symbols []*models.Symbols, mode SmartLocalV2Mode, opts SmartLocalV2Options) (*SmartLocalV2Result, error) {
	return smartLocalV2FromSymbolsWithOrm(ctx, orm.NewOrm(), symbols, mode, opts)
}

func smartLocalV2FromSymbolsWithOrm(ctx context.Context, o orm.Ormer, symbols []*models.Symbols, mode SmartLocalV2Mode, opts SmartLocalV2Options) (*SmartLocalV2Result, error) {
	cooldownMinute := opts.CooldownMinute
	if cooldownMinute <= 0 {
		cooldownMinute = DefaultSmartLocalV2CooldownMinute
	}
	var (
		cooldown map[string]bool
		err      error
	)
	if mode == SmartLocalV2ModeTest {
		cooldown, err = recentTestClosedSymbolsWithOrm(ctx, o, cooldownMinute)
	} else {
		cooldown, err = recentClosedSymbolsWithOrm(ctx, o, cooldownMinute)
	}
	if err != nil {
		return nil, err
	}
	result := SmartLocalV2FromSymbols(symbols, cooldown, opts)
	result.Meta["mode"] = string(mode)
	return &result, nil
}

func SmartLocalV2FromSymbols(symbols []*models.Symbols, cooldown map[string]bool, opts SmartLocalV2Options) SmartLocalV2Result {
	requestedLimit := opts.Limit
	limit := requestedLimit
	if limit <= 0 {
		limit = DefaultSmartLocalV2Limit
	} else if limit > DefaultSmartLocalV2PoolSize {
		limit = DefaultSmartLocalV2PoolSize
	}
	maxDataAgeMs := opts.MaxDataAgeMs
	if maxDataAgeMs <= 0 {
		maxDataAgeMs = DefaultSmartLocalV2MaxDataAgeMs
	}
	minQuoteVolume := opts.MinQuoteVolume
	if minQuoteVolume <= 0 {
		minQuoteVolume = DefaultSmartLocalV2MinQuoteVolume
	}

	now := time.Now().UnixMilli()
	eligible := make([]*models.Symbols, 0, len(symbols))
	excluded := make([]PrefilterExclusion, 0)
	for _, item := range symbols {
		if item == nil {
			continue
		}
		symbol := strings.ToUpper(strings.TrimSpace(item.Symbol))
		switch {
		case item.Enable != 1:
			if opts.IncludeExcluded {
				excluded = append(excluded, PrefilterExclusion{Symbol: symbol, Reason: "币种未启用"})
			}
		case strings.TrimSpace(item.Type) != "USDT" || !strings.HasSuffix(symbol, "USDT"):
			if opts.IncludeExcluded {
				excluded = append(excluded, PrefilterExclusion{Symbol: symbol, Reason: "非 USDT 本位合约"})
			}
		case item.UpdateTime <= 0:
			if opts.IncludeExcluded {
				excluded = append(excluded, PrefilterExclusion{Symbol: symbol, Reason: "缺少本地行情更新时间"})
			}
		case now-item.UpdateTime > maxDataAgeMs:
			if opts.IncludeExcluded {
				excluded = append(excluded, PrefilterExclusion{Symbol: symbol, Reason: "本地行情数据过旧"})
			}
		case cooldown[symbol]:
			if opts.IncludeExcluded {
				excluded = append(excluded, PrefilterExclusion{Symbol: symbol, Reason: "最近交易冷却中"})
			}
		default:
			eligible = append(eligible, item)
		}
	}

	prefilter := PrefilterTop30FromSymbols(eligible, PrefilterOptions{
		Limit:                  limit,
		MaxLimit:               DefaultSmartLocalV2PoolSize,
		MinQuoteVolume:         minQuoteVolume,
		MaxDataAgeMs:           maxDataAgeMs,
		IncludeBenchmarks:      true,
		UseTradeCountScore:     true,
		StableSymbolTieBreak:   true,
		SymmetricChangeScoring: true,
	})
	if opts.IncludeExcluded {
		prefilter.Excluded = append(excluded, prefilter.Excluded...)
		sort.SliceStable(prefilter.Excluded, func(i, j int) bool {
			if prefilter.Excluded[i].Symbol == prefilter.Excluded[j].Symbol {
				return prefilter.Excluded[i].Reason < prefilter.Excluded[j].Reason
			}
			return prefilter.Excluded[i].Symbol < prefilter.Excluded[j].Symbol
		})
	} else {
		prefilter.Excluded = nil
	}
	prefilter.Meta["selector"] = "smart_local_v2"
	prefilter.Meta["requested_limit"] = requestedLimit
	prefilter.Meta["effective_limit"] = limit
	prefilter.Meta["pool_limit"] = limit
	prefilter.Meta["pool_size"] = len(prefilter.Candidates)
	prefilter.Meta["batch_size"] = DefaultSmartLocalV2BatchSize
	prefilter.Meta["cooldown_minute"] = optsOrDefaultCooldown(opts.CooldownMinute)
	prefilter.Meta["max_data_age_ms"] = maxDataAgeMs
	prefilter.Meta["rest_api_used"] = false

	return SmartLocalV2Result{
		PrefilterResult: prefilter,
		Selector:        "smart_local_v2",
	}
}

func recentClosedSymbols(ctx context.Context, cooldownMinute int64) (map[string]bool, error) {
	return recentClosedSymbolsWithOrm(ctx, orm.NewOrm(), cooldownMinute)
}

func recentClosedSymbolsWithOrm(ctx context.Context, o orm.Ormer, cooldownMinute int64) (map[string]bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cooldownMinute <= 0 {
		cooldownMinute = DefaultSmartLocalV2CooldownMinute
	}
	startTime := time.Now().UnixMilli() - cooldownMinute*60*1000
	var orders []models.Order
	_, err := o.QueryTable("order").
		Filter("UpdateTime__gte", startTime).
		Filter("Side", "close").
		All(&orders, "Symbol")
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(orders))
	for _, order := range orders {
		symbol := strings.ToUpper(strings.TrimSpace(order.Symbol))
		if symbol != "" {
			result[symbol] = true
		}
	}
	return result, nil
}

func recentTestClosedSymbolsWithOrm(ctx context.Context, o orm.Ormer, cooldownMinute int64) (map[string]bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cooldownMinute <= 0 {
		cooldownMinute = DefaultSmartLocalV2CooldownMinute
	}
	startTime := time.Now().UnixMilli() - cooldownMinute*60*1000
	var rows []models.TestStrategyResults
	_, err := o.QueryTable(new(models.TestStrategyResults)).
		Filter("UpdateTime__gte", startTime).
		Exclude("ClosePrice", "0").
		All(&rows, "Symbol")
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(rows))
	for _, row := range rows {
		symbol := strings.ToUpper(strings.TrimSpace(row.Symbol))
		if symbol != "" {
			result[symbol] = true
		}
	}
	return result, nil
}

func optsOrDefaultCooldown(value int64) int64 {
	if value <= 0 {
		return DefaultSmartLocalV2CooldownMinute
	}
	return value
}
