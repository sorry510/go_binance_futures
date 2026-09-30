package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go_binance_futures/service/backtest"
	"go_binance_futures/service/historicalmarket"

	"github.com/go-sql-driver/mysql"
	"github.com/klauspost/compress/zstd"
)

const researchMarket = historicalmarket.MarketFuturesUSDT

type armResearchConfig struct {
	host, port, username, password string
	databases                      []string
}

type datasetSource struct {
	Database     string                    `json:"database"`
	AsOfMS       int64                     `json:"as_of_ms"`
	CacheHit     bool                      `json:"cache_hit"`
	ArchiveCount int                       `json:"archive_count"`
	RESTGapBars  map[string]int            `json:"rest_gap_bars,omitempty"`
	BarCoverage  map[string]seriesCoverage `json:"bar_coverage"`
	Funding      seriesCoverage            `json:"funding_coverage"`
}

type seriesCoverage struct {
	Count int   `json:"count"`
	First int64 `json:"first_ms"`
	Last  int64 `json:"last_ms"`
}

type datasetCacheEntry struct {
	Dataset backtest.Dataset
	Source  datasetSource
}

func parseArmResearchConfig(path string) (armResearchConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return armResearchConfig{}, err
	}
	defer file.Close()
	var cfg armResearchConfig
	inArm := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "# arm" {
			inArm = true
			continue
		}
		if !inArm {
			continue
		}
		if line == "" || strings.HasPrefix(line, "[") {
			break
		}
		if !strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(strings.TrimSpace(strings.TrimPrefix(line, "#")), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		switch key {
		case "host":
			cfg.host = value
		case "port":
			cfg.port = value
		case "username":
			cfg.username = value
		case "password":
			cfg.password = value
		case "dbname":
			cfg.databases = append(cfg.databases, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return armResearchConfig{}, err
	}
	if cfg.host == "" || cfg.port == "" || cfg.username == "" || cfg.password == "" {
		return armResearchConfig{}, errors.New("commented # arm block lacks a required connection field")
	}
	wanted := []string{"go_binance", "go_bn_oracle1", "go_bn_oracle2"}
	sort.Strings(wanted)
	sort.Strings(cfg.databases)
	if strings.Join(cfg.databases, "\x00") != strings.Join(wanted, "\x00") {
		return armResearchConfig{}, fmt.Errorf("commented # arm database whitelist differs: %v", cfg.databases)
	}
	return cfg, nil
}

func openArmResearchDB(cfg armResearchConfig, database string) (*sql.DB, error) {
	if database != "go_binance" && database != "go_bn_oracle1" && database != "go_bn_oracle2" {
		return nil, fmt.Errorf("database %q is not in the ARM whitelist", database)
	}
	dsn := mysql.NewConfig()
	dsn.User, dsn.Passwd = cfg.username, cfg.password
	dsn.Net, dsn.Addr, dsn.DBName = "tcp", cfg.host+":"+cfg.port, database
	dsn.Timeout, dsn.ReadTimeout, dsn.WriteTimeout = 10*time.Second, 2*time.Minute, 2*time.Minute
	dsn.Params = map[string]string{"charset": "utf8mb4"}
	db, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open ARM connection: %w", err)
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	return db, nil
}

func researchIntervalMs(interval string) (int64, error) {
	switch interval {
	case "1m":
		return 60_000, nil
	case "1h":
		return 3_600_000, nil
	case "4h":
		return 14_400_000, nil
	case "1d":
		return 86_400_000, nil
	default:
		return 0, fmt.Errorf("research loader does not support interval %q", interval)
	}
}

func readArmSeries(ctx context.Context, tx *sql.Tx, symbol, interval string, start, end int64) ([]backtest.Bar, error) {
	if interval == "1m" {
		return readArmMinutes(ctx, tx, symbol, start, end)
	}
	tables := map[string]string{"1h": "market_klines_1h", "4h": "market_klines_4h", "1d": "market_klines_1d"}
	table, ok := tables[interval]
	if !ok {
		return nil, fmt.Errorf("unsupported ARM interval %q", interval)
	}
	query := "SELECT open_time,close_time,open_price,high_price,low_price,close_price,volume,quote_volume,trade_count,taker_buy_quote_volume" +
		" FROM " + table + " FORCE INDEX (market) WHERE market=? AND symbol=? AND open_time>=? AND open_time<=? ORDER BY open_time"
	rows, err := tx.QueryContext(ctx, query, researchMarket, symbol, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]backtest.Bar, 0)
	for rows.Next() {
		bar := backtest.Bar{Symbol: symbol, Interval: interval}
		if err := rows.Scan(&bar.OpenTime, &bar.CloseTime, &bar.Open, &bar.High, &bar.Low, &bar.Close, &bar.Volume, &bar.QuoteVolume, &bar.TradeCount, &bar.TakerBuyQuoteVolume); err != nil {
			return nil, err
		}
		out = append(out, bar)
	}
	return out, rows.Err()
}

func readArmMinutes(ctx context.Context, tx *sql.Tx, symbol string, start, end int64) ([]backtest.Bar, error) {
	monthStart := time.UnixMilli(start).UTC()
	monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, time.UTC)
	query := "SELECT start_time,end_time,point_count,encoding,compression,checksum,payload" +
		" FROM market_klines_1m_chunks WHERE market=? AND symbol=? AND month_start>=? AND month_start<=? ORDER BY month_start"
	rows, err := tx.QueryContext(ctx, query, researchMarket, symbol, monthStart.UnixMilli(), end)
	if err != nil {
		return nil, err
	}
	out := make([]backtest.Bar, 0)
	for rows.Next() {
		var first, last, checksum int64
		var count int
		var encoding, compression string
		var payload []byte
		if err := rows.Scan(&first, &last, &count, &encoding, &compression, &checksum, &payload); err != nil {
			rows.Close()
			return nil, err
		}
		decoded, err := decodeResearchMinuteChunk(symbol, first, last, count, encoding, compression, checksum, payload)
		if err != nil {
			rows.Close()
			return nil, err
		}
		for _, bar := range decoded {
			if bar.OpenTime >= start && bar.OpenTime <= end {
				out = append(out, bar)
			}
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	rawQuery := "SELECT open_time,close_time,open_price,high_price,low_price,close_price,volume,quote_volume,trade_count,taker_buy_quote_volume" +
		" FROM market_klines_1m FORCE INDEX (market) WHERE market=? AND symbol=? AND open_time>=? AND open_time<=? ORDER BY open_time"
	raw, err := tx.QueryContext(ctx, rawQuery, researchMarket, symbol, start, end)
	if err != nil {
		return nil, err
	}
	rawBars := make([]backtest.Bar, 0)
	for raw.Next() {
		bar := backtest.Bar{Symbol: symbol, Interval: "1m"}
		if err := raw.Scan(&bar.OpenTime, &bar.CloseTime, &bar.Open, &bar.High, &bar.Low, &bar.Close, &bar.Volume, &bar.QuoteVolume, &bar.TradeCount, &bar.TakerBuyQuoteVolume); err != nil {
			raw.Close()
			return nil, err
		}
		rawBars = append(rawBars, bar)
	}
	if err := raw.Close(); err != nil {
		return nil, err
	}
	if len(rawBars) == 0 {
		return out, nil
	}
	merged := make([]backtest.Bar, 0, len(out)+len(rawBars))
	i, j := 0, 0
	for i < len(out) || j < len(rawBars) {
		if j >= len(rawBars) || (i < len(out) && out[i].OpenTime < rawBars[j].OpenTime) {
			merged = append(merged, out[i])
			i++
			continue
		}
		if i >= len(out) || rawBars[j].OpenTime < out[i].OpenTime {
			merged = append(merged, rawBars[j])
			j++
			continue
		}
		if !sameResearchBar(out[i], rawBars[j]) {
			return nil, fmt.Errorf("ARM 1m chunk/raw mismatch at %d", out[i].OpenTime)
		}
		merged = append(merged, out[i])
		i++
		j++
	}
	return merged, nil
}

func decodeResearchMinuteChunk(symbol string, first, last int64, count int, encoding, compression string, checksum int64, payload []byte) ([]backtest.Bar, error) {
	if encoding != "binary_v1" || compression != "zstd" {
		return nil, fmt.Errorf("unsupported minute chunk codec %s/%s", encoding, compression)
	}
	decoder, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
	if err != nil {
		return nil, err
	}
	raw, err := decoder.DecodeAll(payload, nil)
	decoder.Close()
	if err != nil {
		return nil, err
	}
	const header, record = 16, 90
	if len(raw) != header+record*count || string(raw[:8]) != "K1MC0001" || int(binary.LittleEndian.Uint32(raw[8:12])) != count || crc32.ChecksumIEEE(raw) != uint32(checksum) {
		return nil, fmt.Errorf("invalid minute chunk at %d: length/header/count/checksum mismatch", first)
	}
	out := make([]backtest.Bar, count)
	for i := 0; i < count; i++ {
		at := header + i*record
		u64 := func(offset int) uint64 { return binary.LittleEndian.Uint64(raw[at+offset:]) }
		out[i] = backtest.Bar{
			Symbol: symbol, Interval: "1m", OpenTime: int64(u64(0)), CloseTime: int64(u64(8)), TradeCount: int64(u64(16)),
			Open: math.Float64frombits(u64(24)), High: math.Float64frombits(u64(32)), Low: math.Float64frombits(u64(40)),
			Close: math.Float64frombits(u64(48)), Volume: math.Float64frombits(u64(56)),
			QuoteVolume: math.Float64frombits(u64(64)), TakerBuyQuoteVolume: math.Float64frombits(u64(80)),
		}
		if i > 0 && out[i].OpenTime != out[i-1].OpenTime+60_000 {
			return nil, fmt.Errorf("minute chunk gap at %d", out[i].OpenTime)
		}
	}
	if len(out) == 0 || out[0].OpenTime != first || out[len(out)-1].OpenTime != last {
		return nil, fmt.Errorf("minute chunk bounds mismatch at %d", first)
	}
	return out, nil
}

func readArmFunding(ctx context.Context, tx *sql.Tx, symbol string, start, end int64) ([]backtest.Funding, error) {
	query := "SELECT funding_time,funding_rate,mark_price FROM market_funding_rates FORCE INDEX (market)" +
		" WHERE market=? AND symbol=? AND funding_time>=? AND funding_time<=? ORDER BY funding_time"
	rows, err := tx.QueryContext(ctx, query, researchMarket, symbol, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]backtest.Funding, 0)
	for rows.Next() {
		fund := backtest.Funding{Symbol: symbol}
		if err := rows.Scan(&fund.FundingTime, &fund.FundingRate, &fund.MarkPrice); err != nil {
			return nil, err
		}
		out = append(out, fund)
	}
	return out, rows.Err()
}

func fetchResearchFundingPrefix(ctx context.Context, symbol string, start, end int64) ([]historicalmarket.FundingRate, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		rows, err := (historicalmarket.BinanceSource{}).Funding(ctx, researchMarket, symbol, start, end)
		if err == nil {
			return rows, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	return nil, lastErr
}

func sameResearchBar(left, right backtest.Bar) bool {
	if left.OpenTime != right.OpenTime || left.CloseTime != right.CloseTime || left.TradeCount != right.TradeCount {
		return false
	}
	a := []float64{left.Open, left.High, left.Low, left.Close, left.Volume, left.QuoteVolume, left.TakerBuyQuoteVolume}
	b := []float64{right.Open, right.High, right.Low, right.Close, right.Volume, right.QuoteVolume, right.TakerBuyQuoteVolume}
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-8*math.Max(1, math.Abs(a[i])) {
			return false
		}
	}
	return true
}

func mergeResearchBars(symbol, interval string, arm, prefix []backtest.Bar) ([]backtest.Bar, error) {
	out := make([]backtest.Bar, 0, len(arm)+len(prefix))
	i, j := 0, 0
	for i < len(prefix) || j < len(arm) {
		if j >= len(arm) || (i < len(prefix) && prefix[i].OpenTime < arm[j].OpenTime) {
			out = append(out, prefix[i])
			i++
			continue
		}
		if i >= len(prefix) || arm[j].OpenTime < prefix[i].OpenTime {
			out = append(out, arm[j])
			j++
			continue
		}
		if !sameResearchBar(prefix[i], arm[j]) {
			return nil, fmt.Errorf("ARM/public archive mismatch for %s %s at %d", symbol, interval, prefix[i].OpenTime)
		}
		out = append(out, arm[j])
		i++
		j++
	}
	return out, nil
}

func loadArchivePrefix(ctx context.Context, client *historicalmarket.PublicDataClient, symbol, interval string, start, end int64) ([]backtest.Bar, int, error) {
	if end < start {
		return nil, 0, nil
	}
	month := time.UnixMilli(start).UTC()
	month = time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := time.UnixMilli(end).UTC()
	out := make([]backtest.Bar, 0)
	archives := 0
	for !month.After(last) {
		spec := historicalmarket.PublicDataArchiveSpec{Kind: historicalmarket.ArchiveKindKlines, Period: historicalmarket.ArchivePeriodMonthly, Symbol: symbol, Interval: interval, Date: month}
		archive, err := client.FetchArchive(ctx, spec)
		if err != nil {
			return nil, archives, fmt.Errorf("fetch %s %s: %w", interval, month.Format("2006-01"), err)
		}
		rows, err := client.ParseKlines(ctx, archive, start, end)
		if err != nil {
			return nil, archives, fmt.Errorf("parse %s %s: %w", interval, month.Format("2006-01"), err)
		}
		for _, row := range rows {
			out = append(out, backtest.Bar{Symbol: symbol, Interval: interval, OpenTime: row.OpenTime, CloseTime: row.CloseTime,
				Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume,
				QuoteVolume: row.QuoteVolume, TradeCount: row.TradeCount, TakerBuyQuoteVolume: row.TakerBuyQuoteVolume})
		}
		archives++
		month = month.AddDate(0, 1, 0)
	}
	return out, archives, nil
}

func fillResearchBarGaps(ctx context.Context, symbol, interval string, bars []backtest.Bar, start, end int64) ([]backtest.Bar, int, error) {
	duration, err := researchIntervalMs(interval)
	if err != nil {
		return nil, 0, err
	}
	last := end / duration * duration
	filled := make([]backtest.Bar, 0)
	expected := start
	for _, bar := range bars {
		if bar.OpenTime < expected {
			return nil, 0, fmt.Errorf("duplicate/out-of-order %s %s bar at %d", symbol, interval, bar.OpenTime)
		}
		if bar.OpenTime > expected {
			if (bar.OpenTime-expected)/duration > 30 || len(filled)+(int((bar.OpenTime-expected)/duration)) > 100 {
				return nil, 0, fmt.Errorf("%s %s gap too large for REST verification at %d", symbol, interval, expected)
			}
			remote, err := (historicalmarket.BinanceSource{}).Klines(ctx, researchMarket, symbol, interval, expected, bar.OpenTime-1)
			if err != nil {
				return nil, 0, fmt.Errorf("fetch REST gap %s %s at %d: %w", symbol, interval, expected, err)
			}
			for _, row := range remote {
				if row.OpenTime != expected || row.CloseTime != expected+duration-1 {
					return nil, 0, fmt.Errorf("REST gap %s %s returned noncontiguous bar at %d", symbol, interval, row.OpenTime)
				}
				filled = append(filled, backtest.Bar{Symbol: symbol, Interval: interval, OpenTime: row.OpenTime, CloseTime: row.CloseTime,
					Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume,
					QuoteVolume: row.QuoteVolume, TradeCount: row.TradeCount, TakerBuyQuoteVolume: row.TakerBuyQuoteVolume})
				expected += duration
			}
			if expected != bar.OpenTime {
				return nil, 0, fmt.Errorf("REST gap %s %s remains incomplete at %d", symbol, interval, expected)
			}
		}
		expected = bar.OpenTime + duration
	}
	if expected <= last {
		return nil, 0, fmt.Errorf("%s %s trailing gap at %d", symbol, interval, expected)
	}
	if len(filled) == 0 {
		return bars, 0, nil
	}
	merged := make([]backtest.Bar, 0, len(bars)+len(filled))
	merged = append(merged, bars...)
	merged = append(merged, filled...)
	sort.Slice(merged, func(i, j int) bool { return merged[i].OpenTime < merged[j].OpenTime })
	return merged, len(filled), nil
}

func validateResearchBars(symbol, interval string, bars []backtest.Bar, requiredStart, end int64) error {
	duration, err := researchIntervalMs(interval)
	if err != nil {
		return err
	}
	first, last := requiredStart, end/duration*duration
	if first%duration != 0 {
		return fmt.Errorf("unaligned %s required start %d", interval, first)
	}
	want := int((last-first)/duration) + 1
	if len(bars) != want {
		actualFirst, actualLast := int64(0), int64(0)
		gaps := make([]string, 0, 8)
		if len(bars) > 0 {
			actualFirst, actualLast = bars[0].OpenTime, bars[len(bars)-1].OpenTime
			expected := first
			for _, bar := range bars {
				if bar.OpenTime != expected && len(gaps) < 8 {
					gaps = append(gaps, fmt.Sprintf("%s..%s", time.UnixMilli(expected).UTC().Format("2006-01-02"), time.UnixMilli(bar.OpenTime-duration).UTC().Format("2006-01-02")))
				}
				expected = bar.OpenTime + duration
			}
		}
		return fmt.Errorf("%s %s count=%d expected=%d first=%s actual_first=%s last=%s actual_last=%s gaps=%v", symbol, interval, len(bars), want,
			time.UnixMilli(first).UTC().Format(time.RFC3339), time.UnixMilli(actualFirst).UTC().Format(time.RFC3339),
			time.UnixMilli(last).UTC().Format(time.RFC3339), time.UnixMilli(actualLast).UTC().Format(time.RFC3339), gaps)
	}
	for i, bar := range bars {
		if bar.OpenTime != first+int64(i)*duration || bar.CloseTime != bar.OpenTime+duration-1 || bar.High < bar.Low || bar.Open <= 0 || bar.Close <= 0 || bar.QuoteVolume < 0 || bar.TakerBuyQuoteVolume < 0 {
			return fmt.Errorf("invalid/gapped %s %s at index=%d time=%d", symbol, interval, i, bar.OpenTime)
		}
	}
	return nil
}

func researchCachePath(root, symbol string, start, end int64, intervals []string) string {
	key := fmt.Sprintf("arm-go_binance-public-monthly-v1|%s|%d|%d|200|%s", symbol, start, end, strings.Join(intervals, ","))
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(root, "datasets", symbol+"-"+hex.EncodeToString(sum[:12])+".gob.zst")
}

func readResearchDatasetCache(path, symbol string, start, end int64, intervals []string) (backtest.Dataset, datasetSource, bool, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return backtest.Dataset{}, datasetSource{}, false, nil
	}
	if err != nil {
		return backtest.Dataset{}, datasetSource{}, false, err
	}
	defer file.Close()
	decoder, err := zstd.NewReader(file)
	if err != nil {
		return backtest.Dataset{}, datasetSource{}, false, err
	}
	defer decoder.Close()
	var entry datasetCacheEntry
	if err := gob.NewDecoder(decoder).Decode(&entry); err != nil {
		return backtest.Dataset{}, datasetSource{}, false, err
	}
	dataset := entry.Dataset
	if dataset.Symbol != symbol || dataset.StartTime != start || dataset.EndTime != end || strings.Join(dataset.Intervals, ",") != strings.Join(intervals, ",") || dataset.DataHash != backtest.DatasetDataHash(dataset) {
		return backtest.Dataset{}, datasetSource{}, false, fmt.Errorf("cached dataset identity/hash mismatch at %s", path)
	}
	entry.Source.CacheHit = true
	return dataset, entry.Source, true, nil
}

func writeResearchDatasetCache(path string, dataset backtest.Dataset, source datasetSource) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".dataset-*.part")
	if err != nil {
		return err
	}
	part := file.Name()
	defer os.Remove(part)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	encoder, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedFastest), zstd.WithEncoderConcurrency(1))
	if err != nil {
		file.Close()
		return err
	}
	encodeErr := gob.NewEncoder(encoder).Encode(datasetCacheEntry{Dataset: dataset, Source: source})
	closeEncoderErr := encoder.Close()
	closeFileErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	if closeEncoderErr != nil {
		return closeEncoderErr
	}
	if closeFileErr != nil {
		return closeFileErr
	}
	return os.Rename(part, path)
}

func buildResearchDataset(ctx context.Context, db *sql.DB, client *historicalmarket.PublicDataClient, symbol string, start, end int64, intervals []string, cacheRoot string) (backtest.Dataset, datasetSource, error) {
	cachePath := researchCachePath(cacheRoot, symbol, start, end, intervals)
	if dataset, source, hit, err := readResearchDatasetCache(cachePath, symbol, start, end, intervals); err != nil {
		return backtest.Dataset{}, datasetSource{}, err
	} else if hit {
		return dataset, source, nil
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return backtest.Dataset{}, datasetSource{}, err
	}
	defer tx.Rollback()
	source := datasetSource{Database: "go_binance", RESTGapBars: make(map[string]int), BarCoverage: make(map[string]seriesCoverage)}
	if err := tx.QueryRowContext(ctx, "SELECT CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3))*1000 AS UNSIGNED)").Scan(&source.AsOfMS); err != nil {
		return backtest.Dataset{}, datasetSource{}, err
	}
	series := make(map[string][]backtest.Bar, len(intervals))
	starts := make(map[string]int64, len(intervals))
	warmupStart := start
	for _, interval := range intervals {
		duration, err := researchIntervalMs(interval)
		if err != nil {
			return backtest.Dataset{}, datasetSource{}, err
		}
		requiredStart := start - 200*duration
		starts[interval] = requiredStart
		if requiredStart < warmupStart {
			warmupStart = requiredStart
		}
		bars, err := readArmSeries(ctx, tx, symbol, interval, requiredStart, end)
		if err != nil {
			return backtest.Dataset{}, datasetSource{}, fmt.Errorf("read ARM %s %s: %w", symbol, interval, err)
		}
		series[interval] = bars
		fmt.Printf("%s %s ARM rows=%d\n", symbol, interval, len(bars))
	}
	fundingStart := start - 48*3_600_000
	funding, err := readArmFunding(ctx, tx, symbol, fundingStart, end)
	if err != nil {
		return backtest.Dataset{}, datasetSource{}, err
	}
	if err := tx.Commit(); err != nil {
		return backtest.Dataset{}, datasetSource{}, err
	}
	for _, interval := range intervals {
		arm := series[interval]
		if len(arm) == 0 || arm[0].OpenTime > starts[interval] {
			prefixEnd := end
			if len(arm) > 0 && arm[0].OpenTime+31*86_400_000 < end {
				// Keep a month of overlap to verify that public archives match ARM rows.
				prefixEnd = arm[0].OpenTime + 31*86_400_000
			}
			prefix, archives, err := loadArchivePrefix(ctx, client, symbol, interval, starts[interval], prefixEnd)
			if err != nil {
				return backtest.Dataset{}, datasetSource{}, err
			}
			source.ArchiveCount += archives
			series[interval], err = mergeResearchBars(symbol, interval, arm, prefix)
			if err != nil {
				return backtest.Dataset{}, datasetSource{}, err
			}
		}
		merged, restBars, err := fillResearchBarGaps(ctx, symbol, interval, series[interval], starts[interval], end)
		if err != nil {
			return backtest.Dataset{}, datasetSource{}, err
		}
		series[interval] = merged
		if restBars > 0 {
			source.RESTGapBars[interval] = restBars
			fmt.Printf("%s %s REST gap bars=%d\n", symbol, interval, restBars)
		}
		if err := validateResearchBars(symbol, interval, series[interval], starts[interval], end); err != nil {
			return backtest.Dataset{}, datasetSource{}, err
		}
		bars := series[interval]
		source.BarCoverage[interval] = seriesCoverage{Count: len(bars), First: bars[0].OpenTime, Last: bars[len(bars)-1].OpenTime}
		fmt.Printf("%s %s verified rows=%d first=%d last=%d\n", symbol, interval, len(bars), bars[0].OpenTime, bars[len(bars)-1].OpenTime)
	}
	if len(funding) == 0 || funding[0].FundingTime > start-24*3_600_000 {
		prefixEnd := end
		if len(funding) > 0 {
			prefixEnd = funding[0].FundingTime - 1
		}
		remote, err := fetchResearchFundingPrefix(ctx, symbol, fundingStart, prefixEnd)
		if err != nil {
			return backtest.Dataset{}, datasetSource{}, fmt.Errorf("fetch funding prefix %s: %w", symbol, err)
		}
		for _, item := range remote {
			funding = append(funding, backtest.Funding{Symbol: symbol, FundingTime: item.FundingTime, FundingRate: item.FundingRate, MarkPrice: item.MarkPrice})
		}
		sort.Slice(funding, func(i, j int) bool { return funding[i].FundingTime < funding[j].FundingTime })
	}
	if len(funding) == 0 || funding[0].FundingTime > start || funding[len(funding)-1].FundingTime < end-12*3_600_000 {
		return backtest.Dataset{}, datasetSource{}, fmt.Errorf("incomplete funding coverage for %s", symbol)
	}
	for i := 1; i < len(funding); i++ {
		if funding[i].FundingTime <= funding[i-1].FundingTime || funding[i].FundingTime-funding[i-1].FundingTime > 24*3_600_000 {
			return backtest.Dataset{}, datasetSource{}, fmt.Errorf("funding gap/duplicate for %s at %d", symbol, funding[i].FundingTime)
		}
	}
	source.Funding = seriesCoverage{Count: len(funding), First: funding[0].FundingTime, Last: funding[len(funding)-1].FundingTime}
	dataset := backtest.Dataset{Market: researchMarket, Symbol: symbol, ExecutionInterval: "1m", Intervals: intervals,
		StartTime: start, EndTime: end, WarmupStartTime: warmupStart, Bars: make(map[string][]backtest.Bar), Funding: funding}
	for interval, bars := range series {
		dataset.Bars[backtest.BarSeriesKey(symbol, interval)] = bars
	}
	dataset.DatasetSpecHash = backtest.DatasetSpecHash(dataset)
	dataset.DatasetID = "ds_" + dataset.DatasetSpecHash[:24]
	dataset.DataHash = backtest.DatasetDataHash(dataset)
	if err := writeResearchDatasetCache(cachePath, dataset, source); err != nil {
		return backtest.Dataset{}, datasetSource{}, err
	}
	fmt.Printf("%s dataset cached data_hash=%s\n", symbol, dataset.DataHash)
	return dataset, source, nil
}
