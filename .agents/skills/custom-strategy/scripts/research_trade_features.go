package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"go_binance_futures/service/backtest"
)

type featureTradeRow struct {
	Symbol            string  `json:"symbol"`
	Side              string  `json:"side"`
	EntryTime         int64   `json:"entry_time"`
	Cycle             int     `json:"cycle"`
	NetPnL            float64 `json:"net_pnl"`
	ExitReason        string  `json:"exit_reason"`
	RelativeVolume    float64 `json:"relative_volume"`
	SideTakerRatio    float64 `json:"side_taker_ratio"`
	ClosePosition     float64 `json:"close_position"`
	AgainstWick       float64 `json:"against_wick"`
	BodyRangeFraction float64 `json:"body_range_fraction"`
	FiveDayTrend      bool    `json:"five_day_trend"`
	FourHourReturn    float64 `json:"four_hour_return"`
}

func main() {
	if err := writeTradeFeatures(); err != nil {
		fmt.Fprintln(os.Stderr, "trade features error:", err)
		os.Exit(1)
	}
}

func writeTradeFeatures() error {
	cacheDefault, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	resultsPath := flag.String("results", "", "historical research results JSON")
	versionPrefix := flag.String("version-prefix", "", "candidate version prefix")
	outputPath := flag.String("output", "", "output feature JSON")
	cacheRoot := flag.String("cache-root", filepath.Join(cacheDefault, "go-binance-strategy-research"), "research cache root")
	flag.Parse()
	if *resultsPath == "" || *versionPrefix == "" || *outputPath == "" {
		return errors.New("-results, -version-prefix, and -output are required")
	}
	content, err := os.ReadFile(*resultsPath)
	if err != nil {
		return err
	}
	var study struct {
		StartTime int64    `json:"start_time"`
		EndTime   int64    `json:"end_time"`
		Intervals []string `json:"intervals"`
		Runs      []struct {
			CandidateVersion string           `json:"candidate_version"`
			Symbol           string           `json:"symbol"`
			Trades           []backtest.Trade `json:"trades"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(content, &study); err != nil {
		return err
	}
	rows := make([]featureTradeRow, 0)
	matchedRuns := 0
	for _, run := range study.Runs {
		if !strings.HasPrefix(run.CandidateVersion, *versionPrefix) {
			continue
		}
		matchedRuns++
		path := researchCachePath(*cacheRoot, run.Symbol, study.StartTime, study.EndTime, study.Intervals)
		dataset, _, hit, err := readResearchDatasetCache(path, run.Symbol, study.StartTime, study.EndTime, study.Intervals)
		if err != nil {
			return err
		}
		if !hit {
			return fmt.Errorf("dataset cache missing for %s", run.Symbol)
		}
		hours := dataset.Bars[backtest.BarSeriesKey(run.Symbol, "1h")]
		fourHours := dataset.Bars[backtest.BarSeriesKey(run.Symbol, "4h")]
		if len(hours) == 0 || len(fourHours) == 0 {
			return fmt.Errorf("1h/4h bars missing for %s", run.Symbol)
		}
		for _, trade := range run.Trades {
			i := sort.Search(len(hours), func(i int) bool { return hours[i].CloseTime >= trade.EntryTime }) - 1
			j := sort.Search(len(fourHours), func(j int) bool { return fourHours[j].CloseTime >= trade.EntryTime }) - 1
			if i < 8 || j < 30 {
				return fmt.Errorf("insufficient pre-entry bars for %s at %d", run.Symbol, trade.EntryTime)
			}
			bar := hours[i]
			if bar.QuoteVolume <= 0 || bar.High <= bar.Low {
				return fmt.Errorf("invalid signal candle for %s at %d", run.Symbol, trade.EntryTime)
			}
			meanVolume, meanClose := 0.0, 0.0
			for k := i - 8; k < i; k++ {
				meanVolume += hours[k].QuoteVolume / 8
			}
			for k := j - 30; k < j; k++ {
				meanClose += fourHours[k].Close / 30
			}
			if meanVolume <= 0 || meanClose <= 0 {
				return fmt.Errorf("invalid pre-entry mean for %s at %d", run.Symbol, trade.EntryTime)
			}
			span := bar.High - bar.Low
			closePosition := (bar.Close - bar.Low) / span
			againstWick := (bar.High - max(bar.Open, bar.Close)) / span
			taker := bar.TakerBuyQuoteVolume / bar.QuoteVolume
			trend := fourHours[j].Close >= meanClose
			if trade.Side == "SHORT" {
				closePosition = 1 - closePosition
				againstWick = (min(bar.Open, bar.Close) - bar.Low) / span
				taker = 1 - taker
				trend = !trend
			}
			entryDate := time.UnixMilli(trade.EntryTime).UTC()
			cycle := entryDate.Year()
			if entryDate.Month() < time.September {
				cycle--
			}
			rows = append(rows, featureTradeRow{Symbol: run.Symbol, Side: trade.Side, EntryTime: trade.EntryTime, Cycle: cycle,
				NetPnL: trade.NetPnL, ExitReason: trade.ExitReason, RelativeVolume: bar.QuoteVolume / meanVolume,
				SideTakerRatio: taker, ClosePosition: closePosition, AgainstWick: againstWick,
				BodyRangeFraction: (max(bar.Open, bar.Close) - min(bar.Open, bar.Close)) / span,
				FiveDayTrend:      trend, FourHourReturn: fourHours[j].Close/fourHours[j-6].Close - 1})
		}
		fmt.Printf("%s feature rows=%d\n", run.Symbol, len(run.Trades))
		dataset = backtest.Dataset{}
		runtime.GC()
	}
	if matchedRuns == 0 {
		return errors.New("no run matched the candidate prefix")
	}
	output, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(*outputPath, output, 0o600); err != nil {
		return err
	}
	fmt.Printf("wrote %d feature rows to %s\n", len(rows), *outputPath)
	return nil
}
