package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/service/backtest"
	"go_binance_futures/service/historicalmarket"
	strategyservice "go_binance_futures/service/strategy"
	"go_binance_futures/technology"
)

type researchCandidate struct {
	Path           string          `json:"path"`
	Name           string          `json:"name"`
	FileSHA256     string          `json:"file_sha256"`
	TechnologyJSON json.RawMessage `json:"technology"`
	StrategyJSON   json.RawMessage `json:"strategy"`
	Version        string          `json:"version"`
	Intervals      []string        `json:"intervals"`
}

type researchRun struct {
	CandidateVersion string           `json:"candidate_version"`
	Symbol           string           `json:"symbol"`
	DatasetID        string           `json:"dataset_id"`
	DataHash         string           `json:"data_hash"`
	DatasetSource    datasetSource    `json:"dataset_source"`
	EngineVersion    string           `json:"engine_version"`
	ResolutionMode   string           `json:"resolution_mode"`
	Metrics          backtest.Metrics `json:"metrics"`
	GrossPnL         float64          `json:"gross_pnl"`
	FrequencyPerWeek float64          `json:"frequency_per_week"`
	AnnualNet        []float64        `json:"annual_net"`
	AnnualTrades     []int            `json:"annual_trades"`
	Trades           []backtest.Trade `json:"trades"`
	ElapsedMs        int64            `json:"elapsed_ms"`
}

type researchStudy struct {
	CreatedAtUTC string              `json:"created_at_utc"`
	Config       backtest.RunConfig  `json:"config"`
	StartTime    int64               `json:"start_time"`
	EndTime      int64               `json:"end_time"`
	Intervals    []string            `json:"intervals"`
	Candidates   []researchCandidate `json:"candidates"`
	Runs         []researchRun       `json:"runs"`
}

func main() {
	if err := runResearch(); err != nil {
		fmt.Fprintln(os.Stderr, "research replay error:", err)
		os.Exit(1)
	}
}

func runResearch() error {
	cacheDefault, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	confPath := flag.String("conf", "conf/app.conf", "configuration containing the commented ARM block")
	fileList := flag.String("strategy-files", "", "comma-separated portable strategy JSON paths")
	extraIntervals := flag.String("extra-intervals", "", "comma-separated intervals to retain for a shared cached dataset")
	symbolList := flag.String("symbols", "BTCUSDT,ETHUSDT,SOLUSDT,XRPUSDT", "comma-separated symbol names")
	startText := flag.String("start", "2022-09-01", "first UTC trade date")
	endText := flag.String("end", "2026-08-31", "last UTC trade date")
	cacheRoot := flag.String("cache-root", filepath.Join(cacheDefault, "go-binance-strategy-research"), "persistent research cache")
	outputPath := flag.String("output", "", "required result JSON path")
	flag.Parse()
	if strings.TrimSpace(*fileList) == "" || strings.TrimSpace(*outputPath) == "" {
		return errors.New("-strategy-files and -output are required")
	}
	startDate, err := time.Parse("2006-01-02", *startText)
	if err != nil {
		return err
	}
	endDate, err := time.Parse("2006-01-02", *endText)
	if err != nil {
		return err
	}
	start, end := startDate.UnixMilli(), endDate.AddDate(0, 0, 1).UnixMilli()-1
	if start >= end || end > time.Now().UTC().UnixMilli() {
		return errors.New("invalid historical date range")
	}
	candidates := make([]researchCandidate, 0)
	intervalSet := map[string]bool{"1m": true}
	for _, path := range strings.Split(*fileList, ",") {
		item, err := loadResearchCandidate(strings.TrimSpace(path))
		if err != nil {
			return err
		}
		candidates = append(candidates, item)
		for _, interval := range item.Intervals {
			intervalSet[interval] = true
		}
	}
	if strings.TrimSpace(*extraIntervals) != "" {
		for _, raw := range strings.Split(*extraIntervals, ",") {
			interval := strings.TrimSpace(raw)
			if _, err := researchIntervalMs(interval); err != nil {
				return err
			}
			intervalSet[interval] = true
		}
	}
	intervals := make([]string, 0, len(intervalSet))
	for interval := range intervalSet {
		intervals = append(intervals, interval)
	}
	sort.Slice(intervals, func(i, j int) bool {
		left, _ := researchIntervalMs(intervals[i])
		right, _ := researchIntervalMs(intervals[j])
		return left < right
	})
	symbols := make([]string, 0)
	pattern := regexp.MustCompile(`^[A-Z0-9]+USDT$`)
	for _, raw := range strings.Split(*symbolList, ",") {
		symbol := strings.ToUpper(strings.TrimSpace(raw))
		if !pattern.MatchString(symbol) {
			return fmt.Errorf("invalid symbol %q", raw)
		}
		symbols = append(symbols, symbol)
	}
	if len(symbols) == 0 {
		return errors.New("at least one symbol is required")
	}
	config := backtest.RunConfig{InitialEquity: 1000, PositionSizePct: 0.1, Leverage: 8,
		FeeRate: 0.0005, SlippageBps: 5, StopLossPct: 5, TakeProfitPct: 5}
	study := researchStudy{CreatedAtUTC: time.Now().UTC().Format(time.RFC3339), Config: config,
		StartTime: start, EndTime: end, Intervals: intervals, Candidates: candidates, Runs: []researchRun{}}
	if previous, err := readResearchStudy(*outputPath); err != nil {
		return err
	} else if previous != nil {
		if previous.StartTime != start || previous.EndTime != end || previous.Config != config || strings.Join(previous.Intervals, ",") != strings.Join(intervals, ",") || len(previous.Candidates) != len(candidates) {
			return errors.New("existing output has different study identity")
		}
		for i := range candidates {
			if previous.Candidates[i].Version != candidates[i].Version {
				return errors.New("existing output has different candidate snapshots")
			}
		}
		study = *previous
	}
	armCfg, err := parseArmResearchConfig(*confPath)
	if err != nil {
		return err
	}
	db, err := openArmResearchDB(armCfg, "go_binance")
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect go_binance: %w", err)
	}
	archiveDir := filepath.Join(*cacheRoot, "archives")
	client, err := historicalmarket.NewPublicDataClient(historicalmarket.PublicDataClientConfig{CacheDir: archiveDir, Timeout: 5 * time.Minute, MaxRetries: 3})
	if err != nil {
		return err
	}
	defer client.Close()
	for _, symbol := range symbols {
		if allResearchRunsPresent(study, candidates, symbol) {
			fmt.Printf("%s all runs already recorded\n", symbol)
			continue
		}
		dataset, source, err := buildResearchDataset(ctx, db, client, symbol, start, end, intervals, *cacheRoot)
		if err != nil {
			return fmt.Errorf("dataset %s: %w", symbol, err)
		}
		for _, candidate := range candidates {
			if hasResearchRun(study, candidate.Version, symbol, dataset.DataHash) {
				fmt.Printf("%s %s already recorded\n", symbol, candidate.Name)
				continue
			}
			fmt.Printf("%s %s replay start\n", symbol, candidate.Name)
			started := time.Now()
			lastMilestone := -1
			result, err := (backtest.Engine{}).RunWithResolution(ctx, dataset,
				backtest.StrategySnapshot{TemplateName: candidate.Name, TechnologyJSON: string(candidate.TechnologyJSON), StrategyJSON: string(candidate.StrategyJSON), Version: candidate.Version},
				config, backtest.ResolutionModeStandard, func(completed, total int) {
					if total <= 0 {
						return
					}
					milestone := completed * 10 / total
					if milestone != lastMilestone {
						lastMilestone = milestone
						fmt.Printf("%s %s replay %d%%\n", symbol, candidate.Name, milestone*10)
					}
				})
			if err != nil {
				return fmt.Errorf("replay %s %s: %w", symbol, candidate.Name, err)
			}
			run := summarizeResearchRun(candidate, symbol, dataset, source, result, start, end, time.Since(started))
			study.Runs = append(study.Runs, run)
			if err := writeResearchStudy(*outputPath, study); err != nil {
				return err
			}
			fmt.Printf("%s %s trades=%d freq=%.3f/wk net=%.3f dd=%.2f%% elapsed=%s\n", symbol, candidate.Name, run.Metrics.TradeCount, run.FrequencyPerWeek, run.Metrics.NetPnL, run.Metrics.MaxDrawdownPct, time.Since(started).Round(time.Second))
			runtime.GC()
		}
	}
	return nil
}

func loadResearchCandidate(path string) (researchCandidate, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return researchCandidate{}, err
	}
	var raw struct {
		Name       string          `json:"name"`
		Technology json.RawMessage `json:"technology"`
		Strategy   json.RawMessage `json:"strategy"`
	}
	if err := json.Unmarshal(content, &raw); err != nil {
		return researchCandidate{}, err
	}
	if strings.TrimSpace(raw.Name) == "" || len(raw.Technology) == 0 || len(raw.Strategy) == 0 {
		return researchCandidate{}, fmt.Errorf("strategy %s lacks a required field", path)
	}
	var technologyConfig technology.TechnologyConfig
	if err := json.Unmarshal(raw.Technology, &technologyConfig); err != nil {
		return researchCandidate{}, err
	}
	if err := line.ValidateTechnologyConfig(technologyConfig); err != nil {
		return researchCandidate{}, fmt.Errorf("strategy %s technology: %w", path, err)
	}
	var rules []backtest.Rule
	if err := json.Unmarshal(raw.Strategy, &rules); err != nil {
		return researchCandidate{}, err
	}
	types := map[string]int{}
	for _, rule := range rules {
		if rule.Enable {
			if strings.TrimSpace(rule.Name) == "" || strings.TrimSpace(rule.Code) == "" {
				return researchCandidate{}, fmt.Errorf("strategy %s has empty enabled rule", path)
			}
			if strings.Contains(rule.Code, "MarketCondition") {
				return researchCandidate{}, fmt.Errorf("strategy %s uses removed MarketCondition contract", path)
			}
			types[rule.Type]++
		}
	}
	for _, required := range []string{"long", "short", "close_long", "close_short"} {
		if types[required] == 0 {
			return researchCandidate{}, fmt.Errorf("strategy %s lacks enabled %s rule", path, required)
		}
	}
	var groups map[string][]struct {
		Enable        bool   `json:"enable"`
		KlineInterval string `json:"kline_interval"`
	}
	if err := json.Unmarshal(raw.Technology, &groups); err != nil {
		return researchCandidate{}, err
	}
	seen := map[string]bool{}
	for _, items := range groups {
		for _, item := range items {
			if item.Enable {
				if _, err := researchIntervalMs(item.KlineInterval); err != nil {
					return researchCandidate{}, err
				}
				seen[item.KlineInterval] = true
			}
		}
	}
	intervals := make([]string, 0, len(seen))
	for interval := range seen {
		intervals = append(intervals, interval)
	}
	sort.Slice(intervals, func(i, j int) bool {
		left, _ := researchIntervalMs(intervals[i])
		right, _ := researchIntervalMs(intervals[j])
		return left < right
	})
	return researchCandidate{Path: path, Name: raw.Name, FileSHA256: researchSHA256(content), TechnologyJSON: raw.Technology, StrategyJSON: raw.Strategy,
		Version: strategyservice.StrategySnapshotHash(string(raw.Technology), string(raw.Strategy)), Intervals: intervals}, nil
}

func summarizeResearchRun(candidate researchCandidate, symbol string, dataset backtest.Dataset, source datasetSource, result backtest.Result, start, end int64, elapsed time.Duration) researchRun {
	annual := []float64{0, 0, 0, 0}
	annualTrades := []int{0, 0, 0, 0}
	gross := 0.0
	for _, trade := range result.Trades {
		gross += trade.GrossPnL
		for index := 0; index < 4; index++ {
			begin := time.UnixMilli(start).UTC().AddDate(index, 0, 0).UnixMilli()
			finish := time.UnixMilli(start).UTC().AddDate(index+1, 0, 0).UnixMilli()
			if trade.ExitTime >= begin && trade.ExitTime < finish {
				annual[index] += trade.NetPnL
				annualTrades[index]++
				break
			}
		}
	}
	weeks := float64(end-start+1) / float64(7*24*time.Hour/time.Millisecond)
	return researchRun{CandidateVersion: candidate.Version, Symbol: symbol, DatasetID: dataset.DatasetID,
		DataHash: dataset.DataHash, DatasetSource: source, EngineVersion: result.EngineVersion,
		ResolutionMode: result.ResolutionMode, Metrics: result.Metrics, GrossPnL: gross,
		FrequencyPerWeek: float64(result.Metrics.TradeCount) / weeks, AnnualNet: annual, AnnualTrades: annualTrades,
		Trades: result.Trades, ElapsedMs: elapsed.Milliseconds()}
}

func hasResearchRun(study researchStudy, version, symbol, dataHash string) bool {
	for _, run := range study.Runs {
		if run.CandidateVersion == version && run.Symbol == symbol && run.DataHash == dataHash {
			return true
		}
	}
	return false
}

func allResearchRunsPresent(study researchStudy, candidates []researchCandidate, symbol string) bool {
	for _, candidate := range candidates {
		found := false
		for _, run := range study.Runs {
			if run.CandidateVersion == candidate.Version && run.Symbol == symbol {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func readResearchStudy(path string) (*researchStudy, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var study researchStudy
	if err := json.Unmarshal(data, &study); err != nil {
		return nil, err
	}
	return &study, nil
}

func writeResearchStudy(path string, study researchStudy) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(study, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".research-result-*.part")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func researchSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
