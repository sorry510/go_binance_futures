package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
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
const researchArchiveRepairVersion = "20261003-v2"

type armResearchConfig struct {
	host, port, username, password string
	databases                      []string
}

type datasetSource struct {
	Database           string                    `json:"database"`
	AsOfMS             int64                     `json:"as_of_ms"`
	CacheHit           bool                      `json:"cache_hit"`
	ArchiveCount       int                       `json:"archive_count"`
	ExecutionSource    string                    `json:"execution_source,omitempty"`
	IndicatorSource    string                    `json:"indicator_source,omitempty"`
	RESTGapBars        map[string]int            `json:"rest_gap_bars,omitempty"`
	BarCoverage        map[string]seriesCoverage `json:"bar_coverage"`
	Funding            seriesCoverage            `json:"funding_coverage"`
	AggregateChecks    map[string]aggregateCheck `json:"aggregate_checks,omitempty"`
	MinuteRepairPolicy string                    `json:"minute_repair_policy,omitempty"`
	RepairVersion      string                    `json:"repair_version,omitempty"`
	MinuteRepairs      []researchMinuteRepair    `json:"minute_repairs,omitempty"`
	IndicatorRepairs   []researchIndicatorRepair `json:"indicator_repairs,omitempty"`
}

type researchIndicatorRepair struct {
	Interval      string       `json:"interval"`
	ArchiveURL    string       `json:"archive_url"`
	ArchiveSHA256 string       `json:"archive_sha256"`
	Verification  string       `json:"verification"`
	Before        backtest.Bar `json:"before"`
	After         backtest.Bar `json:"after"`
}

type researchMinuteRepair struct {
	HourMS           int64        `json:"hour_ms"`
	ChangedMinutes   int          `json:"changed_minutes"`
	ArchiveURL       string       `json:"archive_url"`
	ArchiveSHA256    string       `json:"archive_sha256"`
	Verification     string       `json:"verification"`
	TradeIDSpanHoles int64        `json:"trade_id_span_holes,omitempty"`
	Before           backtest.Bar `json:"before"`
	After            backtest.Bar `json:"after"`
}

type aggregateCheck struct {
	MatchedPrices                int     `json:"matched_prices"`
	CanonicalVolumeDifferences   int     `json:"canonical_volume_differences"`
	FirstVolumeDifference        int64   `json:"first_volume_difference_ms,omitempty"`
	MaximumQuoteVolumeDifference float64 `json:"maximum_relative_quote_volume_difference,omitempty"`
	ZeroTradeMinutes             int     `json:"zero_trade_minutes"`
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

func aggregateResearchMinutes(minutes []backtest.Bar, interval string) (backtest.Bar, int) {
	if len(minutes) == 0 {
		return backtest.Bar{}, 0
	}
	aggregate := minutes[0]
	aggregate.Interval = interval
	aggregate.CloseTime = minutes[len(minutes)-1].CloseTime
	aggregate.Close = minutes[len(minutes)-1].Close
	aggregate.Volume, aggregate.QuoteVolume = 0, 0
	aggregate.TradeCount, aggregate.TakerBuyQuoteVolume = 0, 0
	priceStarted, zeroMinutes := false, 0
	for _, minute := range minutes {
		// Empty carry bars are observations, not traded OHLC extremes.
		if minute.TradeCount > 0 || minute.Volume > 0 || minute.QuoteVolume > 0 {
			if !priceStarted {
				aggregate.Open, aggregate.High, aggregate.Low = minute.Open, minute.High, minute.Low
				priceStarted = true
			} else {
				aggregate.High = math.Max(aggregate.High, minute.High)
				aggregate.Low = math.Min(aggregate.Low, minute.Low)
			}
			aggregate.Close = minute.Close
		} else {
			zeroMinutes++
		}
		aggregate.Volume += minute.Volume
		aggregate.QuoteVolume += minute.QuoteVolume
		aggregate.TradeCount += minute.TradeCount
		aggregate.TakerBuyQuoteVolume += minute.TakerBuyQuoteVolume
	}
	return aggregate, zeroMinutes
}

func sameResearchPrices(left, right backtest.Bar) bool {
	left.Volume, left.QuoteVolume = right.Volume, right.QuoteVolume
	left.TradeCount, left.TakerBuyQuoteVolume = right.TradeCount, right.TakerBuyQuoteVolume
	return sameResearchBar(left, right)
}

func researchArchiveKlines(ctx context.Context, client *historicalmarket.PublicDataClient, archive historicalmarket.PublicDataArchive, symbol, interval string, start, end int64) ([]backtest.Bar, error) {
	rows, err := client.ParseKlines(ctx, archive, start, end)
	if err != nil {
		return nil, err
	}
	bars := make([]backtest.Bar, 0, len(rows))
	for _, row := range rows {
		bars = append(bars, backtest.Bar{Symbol: symbol, Interval: interval, OpenTime: row.OpenTime, CloseTime: row.CloseTime,
			Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume, QuoteVolume: row.QuoteVolume,
			TradeCount: row.TradeCount, TakerBuyQuoteVolume: row.TakerBuyQuoteVolume})
	}
	return bars, nil
}

func reconstructResearchMinutes(original []backtest.Bar, trades []historicalmarket.PublicDataTrade) ([]backtest.Bar, int64, error) {
	if len(original) != 60 || len(trades) == 0 {
		return nil, 0, fmt.Errorf("trade repair requires one complete hour and positive trade coverage")
	}
	bars := append([]backtest.Bar(nil), original...)
	seenIDs := make(map[int64]bool, len(trades))
	active := make([]bool, len(bars))
	minimumID, maximumID := trades[0].TradeID, trades[0].TradeID
	for _, trade := range trades {
		if trade.TradeTime < original[0].OpenTime || trade.TradeTime > original[len(original)-1].CloseTime {
			return nil, 0, fmt.Errorf("trade outside repair hour")
		}
		if seenIDs[trade.TradeID] || trade.Price <= 0 || trade.Quantity <= 0 || trade.QuoteQuantity <= 0 {
			return nil, 0, fmt.Errorf("invalid/duplicate trade in minute repair")
		}
		seenIDs[trade.TradeID] = true
		minimumID, maximumID = min(minimumID, trade.TradeID), max(maximumID, trade.TradeID)
		i := int((trade.TradeTime - original[0].OpenTime) / 60_000)
		if i < 0 || i >= len(bars) {
			return nil, 0, fmt.Errorf("trade outside repair hour")
		}
		bar := &bars[i]
		if !active[i] {
			bar.Open, bar.High, bar.Low = trade.Price, trade.Price, trade.Price
			bar.Volume, bar.QuoteVolume, bar.TradeCount, bar.TakerBuyQuoteVolume = 0, 0, 0, 0
			active[i] = true
		}
		bar.High, bar.Low = max(bar.High, trade.Price), min(bar.Low, trade.Price)
		bar.Close = trade.Price
		bar.Volume += trade.Quantity
		// USD-M turnover is price times base quantity; archive quoteQty may be stale.
		quote := trade.Price * trade.Quantity
		bar.QuoteVolume += quote
		bar.TradeCount++
		if !trade.IsBuyerMaker {
			bar.TakerBuyQuoteVolume += quote
		}
	}
	for i, bar := range original {
		if !active[i] && (bar.TradeCount > 0 || bar.QuoteVolume > 0 || bar.Volume > 0) {
			return nil, 0, fmt.Errorf("trade archive lacks an active minute at %d", bar.OpenTime)
		}
	}
	return bars, maximumID - minimumID + 1 - int64(len(seenIDs)), nil
}

func repairResearchArchiveMinutes(ctx context.Context, db *sql.DB, client *historicalmarket.PublicDataClient, symbol string, series map[string][]backtest.Bar, start, end int64, cacheDir string) ([]researchMinuteRepair, int, error) {
	minutes, hours := series["1m"], series["1h"]
	if len(minutes) == 0 || len(hours) == 0 {
		return nil, 0, fmt.Errorf("verified minute repair requires both 1m and 1h series")
	}
	repairs := make([]researchMinuteRepair, 0)
	archives, changedTotal := 0, 0
	for _, hour := range hours {
		if hour.OpenTime < start || hour.CloseTime > end {
			continue
		}
		first := sort.Search(len(minutes), func(i int) bool { return minutes[i].OpenTime >= hour.OpenTime })
		if first+60 > len(minutes) || minutes[first].OpenTime != hour.OpenTime || minutes[first+59].CloseTime != hour.CloseTime {
			return nil, archives, fmt.Errorf("minute repair hour is incomplete at %d", hour.OpenTime)
		}
		original := minutes[first : first+60]
		before, _ := aggregateResearchMinutes(original, "1h")
		// Independent canonical volume values are audited, not automatically repaired.
		if sameResearchPrices(before, hour) {
			continue
		}
		if len(repairs) >= 32 {
			return nil, archives, fmt.Errorf("more than 32 anomalous hours for %s; stop for source audit", symbol)
		}
		date := time.UnixMilli(hour.OpenTime).UTC()
		archive, err := fetchResearchArchive(ctx, client, historicalmarket.PublicDataArchiveSpec{Kind: historicalmarket.ArchiveKindKlines,
			Period: historicalmarket.ArchivePeriodDaily, Symbol: symbol, Interval: "1m", Date: date}, cacheDir)
		if err != nil {
			return nil, archives, err
		}
		archives++
		replacement, err := researchArchiveKlines(ctx, client, archive, symbol, "1m", hour.OpenTime, hour.CloseTime)
		if err != nil {
			return nil, archives, err
		}
		if err := validateResearchBars(symbol, "1m", replacement, hour.OpenTime, hour.CloseTime); err != nil {
			return nil, archives, err
		}
		after, _ := aggregateResearchMinutes(replacement, "1h")
		verification, holes := "daily_1m_exact_canonical_1h", int64(0)
		if !sameResearchBar(after, hour) {
			tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
			if err != nil {
				return nil, archives, err
			}
			armHours, readErr := readArmSeries(ctx, tx, symbol, "1h", hour.OpenTime, hour.CloseTime)
			commitErr := tx.Commit()
			if readErr != nil {
				return nil, archives, readErr
			}
			if commitErr != nil {
				return nil, archives, commitErr
			}
			// A stale higher bar must not force a valid minute stream to change.
			if sameResearchBar(after, before) && len(armHours) == 1 && sameResearchBar(after, armHours[0]) {
				continue
			}
			archive, err = fetchResearchArchive(ctx, client, historicalmarket.PublicDataArchiveSpec{Kind: historicalmarket.ArchiveKindTrades,
				Period: historicalmarket.ArchivePeriodDaily, Symbol: symbol, Date: date}, cacheDir)
			if err != nil {
				return nil, archives, err
			}
			archives++
			trades, err := client.ParseTrades(ctx, archive, hour.OpenTime, hour.CloseTime)
			if err != nil {
				return nil, archives, err
			}
			replacement, holes, err = reconstructResearchMinutes(original, trades)
			if err != nil {
				return nil, archives, err
			}
			after, _ = aggregateResearchMinutes(replacement, "1h")
			verification = "daily_trades_exact_canonical_1h"
			if !sameResearchBar(after, hour) {
				if len(armHours) != 1 || !sameResearchBar(after, armHours[0]) {
					return nil, archives, fmt.Errorf("trade reconstruction volume/count lacks independent 1h confirmation at %d", hour.OpenTime)
				}
				verification = "daily_trades_exact_arm_1h"
			}
		}
		changed := 0
		for i := range replacement {
			if !sameResearchBar(original[i], replacement[i]) {
				changed++
			}
		}
		if changed == 0 {
			return nil, archives, fmt.Errorf("verified minute repair made no change at %d", hour.OpenTime)
		}
		changedTotal += changed
		if changedTotal > 1000 {
			return nil, archives, fmt.Errorf("more than 1000 repaired minutes for %s; stop for source audit", symbol)
		}
		copy(original, replacement)
		repairs = append(repairs, researchMinuteRepair{HourMS: hour.OpenTime, ChangedMinutes: changed, ArchiveURL: archive.URL,
			ArchiveSHA256: archive.SHA256, Verification: verification, TradeIDSpanHoles: holes, Before: before, After: after})
		fmt.Printf("%s minute_repair hour=%d changed=%d proof=%s trade_id_span_holes=%d\n", symbol, hour.OpenTime, changed, verification, holes)
	}
	return repairs, archives, nil
}

func repairResearchIndicatorPrices(ctx context.Context, db *sql.DB, client *historicalmarket.PublicDataClient, symbol string, series map[string][]backtest.Bar, start, end int64, cacheDir string) ([]researchIndicatorRepair, int, error) {
	minutes := series["1m"]
	repairs := make([]researchIndicatorRepair, 0)
	archives := 0
	for _, interval := range []string{"1h", "4h", "1d"} {
		higher := series[interval]
		duration, _ := researchIntervalMs(interval)
		for i, bar := range higher {
			if bar.OpenTime < start || bar.CloseTime > end {
				continue
			}
			first := sort.Search(len(minutes), func(i int) bool { return minutes[i].OpenTime >= bar.OpenTime })
			last := first + int(duration/60_000)
			if last > len(minutes) || minutes[first].OpenTime != bar.OpenTime || minutes[last-1].CloseTime != bar.CloseTime {
				return nil, archives, fmt.Errorf("indicator repair minute coverage missing at %d", bar.OpenTime)
			}
			aggregate, _ := aggregateResearchMinutes(minutes[first:last], interval)
			if sameResearchPrices(aggregate, bar) {
				continue
			}
			if len(repairs) >= 48 {
				return nil, archives, fmt.Errorf("more than 48 indicator price anomalies for %s; stop for audit", symbol)
			}
			archive, err := fetchResearchArchive(ctx, client, historicalmarket.PublicDataArchiveSpec{Kind: historicalmarket.ArchiveKindKlines,
				Period: historicalmarket.ArchivePeriodDaily, Symbol: symbol, Interval: interval, Date: time.UnixMilli(bar.OpenTime).UTC()}, cacheDir)
			if err != nil {
				return nil, archives, err
			}
			archives++
			rows, err := researchArchiveKlines(ctx, client, archive, symbol, interval, bar.OpenTime, bar.CloseTime)
			if err != nil {
				return nil, archives, err
			}
			verification := "daily_archive_exact_minute_aggregate"
			if len(rows) != 1 || !sameResearchBar(aggregate, rows[0]) {
				if interval != "1h" {
					return nil, archives, fmt.Errorf("daily %s archive cannot independently verify repaired aggregate at %d", interval, bar.OpenTime)
				}
				// Confirm a stale hourly archive against ARM plus an independent daily tape.
				proofArchive, err := fetchResearchArchive(ctx, client, historicalmarket.PublicDataArchiveSpec{Kind: historicalmarket.ArchiveKindKlines,
					Period: historicalmarket.ArchivePeriodDaily, Symbol: symbol, Interval: "1m", Date: time.UnixMilli(bar.OpenTime).UTC()}, cacheDir)
				if err != nil {
					return nil, archives, err
				}
				archives++
				proofMinutes, err := researchArchiveKlines(ctx, client, proofArchive, symbol, "1m", bar.OpenTime, bar.CloseTime)
				if err != nil {
					return nil, archives, err
				}
				proof, _ := aggregateResearchMinutes(proofMinutes, interval)
				verification = "daily_1m_and_arm_exact_minute_aggregate"
				if !sameResearchBar(aggregate, proof) {
					proofArchive, err = fetchResearchArchive(ctx, client, historicalmarket.PublicDataArchiveSpec{Kind: historicalmarket.ArchiveKindTrades,
						Period: historicalmarket.ArchivePeriodDaily, Symbol: symbol, Date: time.UnixMilli(bar.OpenTime).UTC()}, cacheDir)
					if err != nil {
						return nil, archives, err
					}
					archives++
					trades, err := client.ParseTrades(ctx, proofArchive, bar.OpenTime, bar.CloseTime)
					if err != nil {
						return nil, archives, err
					}
					proofMinutes, _, err = reconstructResearchMinutes(minutes[first:last], trades)
					if err != nil {
						return nil, archives, err
					}
					proof, _ = aggregateResearchMinutes(proofMinutes, interval)
					verification = "daily_trades_and_arm_exact_minute_aggregate"
				}
				tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
				if err != nil {
					return nil, archives, err
				}
				arm, readErr := readArmSeries(ctx, tx, symbol, interval, bar.OpenTime, bar.CloseTime)
				commitErr := tx.Commit()
				if readErr != nil {
					return nil, archives, readErr
				}
				if commitErr != nil {
					return nil, archives, commitErr
				}
				if len(arm) != 1 || !sameResearchBar(aggregate, proof) || !sameResearchBar(aggregate, arm[0]) {
					return nil, archives, fmt.Errorf("hourly source repair lacks exact daily/ARM confirmation at %d", bar.OpenTime)
				}
				rows, archive = arm, proofArchive
			}
			higher[i] = rows[0]
			repairs = append(repairs, researchIndicatorRepair{Interval: interval, ArchiveURL: archive.URL, ArchiveSHA256: archive.SHA256, Verification: verification, Before: bar, After: rows[0]})
			fmt.Printf("%s indicator_repair interval=%s at=%d proof=%s\n", symbol, interval, bar.OpenTime, verification)
		}
	}
	return repairs, archives, nil
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

func fetchResearchArchive(ctx context.Context, client *historicalmarket.PublicDataClient, spec historicalmarket.PublicDataArchiveSpec, cacheDir string) (historicalmarket.PublicDataArchive, error) {
	url, err := client.ArchiveURL(spec)
	if err != nil {
		return historicalmarket.PublicDataArchive{}, err
	}
	key := sha256.Sum256([]byte(url))
	manifest := filepath.Join(cacheDir, "research-manifests", hex.EncodeToString(key[:])+".json")
	if content, err := os.ReadFile(manifest); err == nil {
		var archive historicalmarket.PublicDataArchive
		if err := json.Unmarshal(content, &archive); err != nil {
			return archive, fmt.Errorf("read archive manifest: %w", err)
		}
		if archive.URL != url || !strings.HasPrefix(filepath.Clean(archive.Path), filepath.Clean(cacheDir)+string(os.PathSeparator)) || len(archive.SHA256) != 64 {
			return archive, fmt.Errorf("archive manifest identity mismatch for %s", url)
		}
		file, err := os.Open(archive.Path)
		if err != nil {
			return archive, err
		}
		hash := sha256.New()
		size, hashErr := io.Copy(hash, file)
		closeErr := file.Close()
		if hashErr != nil {
			return archive, hashErr
		}
		if closeErr != nil {
			return archive, closeErr
		}
		if size != archive.Size || !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), archive.SHA256) {
			return archive, fmt.Errorf("verified archive cache hash/size changed for %s", url)
		}
		archive.FromCache = true
		return archive, nil
	} else if !os.IsNotExist(err) {
		return historicalmarket.PublicDataArchive{}, err
	}
	fmt.Printf("%s %s archive_fetch=%s\n", spec.Symbol, spec.Interval, spec.Date.Format("2006-01"))
	archive, err := client.FetchArchive(ctx, spec)
	if err != nil {
		return archive, err
	}
	content, err := json.Marshal(archive)
	if err != nil {
		return archive, err
	}
	if err := os.MkdirAll(filepath.Dir(manifest), 0o700); err != nil {
		return archive, err
	}
	if err := os.WriteFile(manifest+".part", content, 0o600); err != nil {
		return archive, err
	}
	if err := os.Rename(manifest+".part", manifest); err != nil {
		return archive, err
	}
	return archive, nil
}

func loadArchivePrefix(ctx context.Context, client *historicalmarket.PublicDataClient, symbol, interval string, start, end int64, cacheDir string) ([]backtest.Bar, int, error) {
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
		archive, err := fetchResearchArchive(ctx, client, spec, cacheDir)
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
		fmt.Printf("%s %s archive=%s verified_sha256=%s rows=%d cache=%t\n", symbol, interval, month.Format("2006-01"), archive.SHA256[:12], len(rows), archive.FromCache)
		month = month.AddDate(0, 1, 0)
	}
	return out, archives, nil
}

func loadResearchArchiveSeries(ctx context.Context, client *historicalmarket.PublicDataClient, symbol string, intervals []string, starts map[string]int64, end int64, cacheDir string) (map[string][]backtest.Bar, int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		interval string
		bars     []backtest.Bar
		archives int
		err      error
	}
	results := make(chan result, len(intervals))
	for _, interval := range intervals {
		go func(interval string) {
			bars, archives, err := loadArchivePrefix(ctx, client, symbol, interval, starts[interval], end, cacheDir)
			results <- result{interval: interval, bars: bars, archives: archives, err: err}
		}(interval)
	}
	series := make(map[string][]backtest.Bar, len(intervals))
	count := 0
	var firstErr error
	for range intervals {
		item := <-results
		if item.err != nil && firstErr == nil {
			firstErr = item.err
			cancel()
		}
		series[item.interval] = item.bars
		count += item.archives
	}
	return series, count, firstErr
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

func buildResearchDataset(ctx context.Context, db *sql.DB, client *historicalmarket.PublicDataClient, symbol string, start, end int64, intervals []string, cacheRoot, executionSource, indicatorSource, minuteRepairPolicy, archiveDir string) (backtest.Dataset, datasetSource, error) {
	if executionSource != "arm" && executionSource != "public-archive" {
		return backtest.Dataset{}, datasetSource{}, fmt.Errorf("unsupported execution source %q", executionSource)
	}
	if indicatorSource != "arm" && indicatorSource != "public-archive" {
		return backtest.Dataset{}, datasetSource{}, fmt.Errorf("unsupported indicator source %q", indicatorSource)
	}
	if minuteRepairPolicy != "none" && minuteRepairPolicy != "verified-archive" {
		return backtest.Dataset{}, datasetSource{}, fmt.Errorf("unsupported minute repair policy %q", minuteRepairPolicy)
	}
	if minuteRepairPolicy != "none" && (executionSource != "public-archive" || indicatorSource != "public-archive") {
		return backtest.Dataset{}, datasetSource{}, fmt.Errorf("verified minute repair requires full public-archive sources")
	}
	cachePath := researchCachePath(cacheRoot, symbol, start, end, intervals)
	if dataset, source, hit, err := readResearchDatasetCache(cachePath, symbol, start, end, intervals); err != nil {
		return backtest.Dataset{}, datasetSource{}, err
	} else if hit {
		priorExecution, priorIndicator := source.ExecutionSource, source.IndicatorSource
		if priorExecution == "" {
			priorExecution = "arm"
		}
		if priorIndicator == "" {
			priorIndicator = "arm"
		}
		priorRepair := source.MinuteRepairPolicy
		if priorRepair == "" {
			priorRepair = "none"
		}
		if priorExecution != executionSource || priorIndicator != indicatorSource || priorRepair != minuteRepairPolicy {
			return backtest.Dataset{}, datasetSource{}, fmt.Errorf("dataset cache source differs from request; select a separate -cache-root for the new source")
		}
		if minuteRepairPolicy != "none" && source.RepairVersion != researchArchiveRepairVersion {
			return backtest.Dataset{}, datasetSource{}, fmt.Errorf("dataset cache repair version differs; select a separate -cache-root")
		}
		return dataset, source, nil
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return backtest.Dataset{}, datasetSource{}, err
	}
	defer tx.Rollback()
	source := datasetSource{Database: "go_binance", ExecutionSource: executionSource, IndicatorSource: indicatorSource, MinuteRepairPolicy: minuteRepairPolicy, RESTGapBars: make(map[string]int), BarCoverage: make(map[string]seriesCoverage)}
	if minuteRepairPolicy != "none" {
		source.RepairVersion = researchArchiveRepairVersion
	}
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
		if (interval == "1m" && executionSource == "public-archive") || (interval != "1m" && indicatorSource == "public-archive") {
			series[interval] = nil
			fmt.Printf("%s %s source=Binance checksum-verified public monthly archives\n", symbol, interval)
			continue
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
	if executionSource == "public-archive" && indicatorSource == "public-archive" {
		publicSeries, archives, err := loadResearchArchiveSeries(ctx, client, symbol, intervals, starts, end, archiveDir)
		if err != nil {
			return backtest.Dataset{}, datasetSource{}, err
		}
		series = publicSeries
		source.ArchiveCount += archives
	}
	for _, interval := range intervals {
		arm := series[interval]
		if len(arm) == 0 || arm[0].OpenTime > starts[interval] {
			prefixEnd := end
			if len(arm) > 0 && arm[0].OpenTime+31*86_400_000 < end {
				// Keep a month of overlap to verify that public archives match ARM rows.
				prefixEnd = arm[0].OpenTime + 31*86_400_000
			}
			prefix, archives, err := loadArchivePrefix(ctx, client, symbol, interval, starts[interval], prefixEnd, archiveDir)
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
	if minuteRepairPolicy == "verified-archive" {
		repairs, archives, err := repairResearchArchiveMinutes(ctx, db, client, symbol, series, start, end, archiveDir)
		if err != nil {
			return backtest.Dataset{}, datasetSource{}, err
		}
		source.MinuteRepairs = repairs
		source.ArchiveCount += archives
		indicatorRepairs, indicatorArchives, err := repairResearchIndicatorPrices(ctx, db, client, symbol, series, start, end, archiveDir)
		if err != nil {
			return backtest.Dataset{}, datasetSource{}, err
		}
		source.IndicatorRepairs = indicatorRepairs
		source.ArchiveCount += indicatorArchives
	}
	if executionSource == "public-archive" {
		checks, archives, err := validateResearchMinuteAggregates(ctx, client, symbol, series, start, end, indicatorSource == "public-archive", archiveDir)
		if err != nil {
			return backtest.Dataset{}, datasetSource{}, err
		}
		source.AggregateChecks = checks
		source.ArchiveCount += archives
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

func validateResearchMinuteAggregates(ctx context.Context, client *historicalmarket.PublicDataClient, symbol string, series map[string][]backtest.Bar, start, end int64, indicatorsCanonical bool, cacheDir string) (map[string]aggregateCheck, int, error) {
	minutes := series["1m"]
	checks := make(map[string]aggregateCheck)
	archives := 0
	for interval, higher := range series {
		if interval == "1m" {
			continue
		}
		duration, err := researchIntervalMs(interval)
		if err != nil {
			return nil, archives, err
		}
		expected := make(map[int64]backtest.Bar, len(higher))
		for _, bar := range higher {
			expected[bar.OpenTime] = bar
		}
		check := aggregateCheck{}
		canonicalMonths := make(map[int64]map[int64]backtest.Bar)
		for first := 0; first < len(minutes); {
			bucket := minutes[first].OpenTime / duration * duration
			last := first + 1
			for last < len(minutes) && minutes[last].OpenTime < bucket+duration {
				last++
			}
			if bucket >= start && bucket+duration-1 <= end && int64(last-first) == duration/60_000 {
				aggregate, zeroMinutes := aggregateResearchMinutes(minutes[first:last], interval)
				check.ZeroTradeMinutes += zeroMinutes
				bar, exists := expected[bucket]
				priceOnly := aggregate
				priceOnly.Volume, priceOnly.QuoteVolume = bar.Volume, bar.QuoteVolume
				priceOnly.TradeCount, priceOnly.TakerBuyQuoteVolume = bar.TradeCount, bar.TakerBuyQuoteVolume
				if !exists || !sameResearchBar(priceOnly, bar) {
					return nil, archives, fmt.Errorf("minute/indicator price aggregate mismatch %s %s at %d: minute=%+v higher=%+v", symbol, interval, bucket, aggregate, bar)
				}
				if !sameResearchBar(aggregate, bar) {
					date := time.UnixMilli(bucket).UTC()
					month := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
					canonical, ok := canonicalMonths[month.UnixMilli()]
					if indicatorsCanonical {
						canonical, ok = expected, true
					}
					if !ok {
						items, fetched, err := loadArchivePrefix(ctx, client, symbol, interval, month.UnixMilli(), month.AddDate(0, 1, 0).UnixMilli()-1, cacheDir)
						if err != nil {
							return nil, archives, err
						}
						archives += fetched
						canonical = make(map[int64]backtest.Bar, len(items))
						for _, item := range items {
							canonical[item.OpenTime] = item
						}
						canonicalMonths[month.UnixMilli()] = canonical
					}
					official, ok := canonical[bucket]
					if !ok || !sameResearchBar(official, bar) {
						return nil, archives, fmt.Errorf("ARM indicator differs from canonical archive %s %s at %d: official=%+v ARM=%+v", symbol, interval, bucket, official, bar)
					}
					check.CanonicalVolumeDifferences++
					if check.FirstVolumeDifference == 0 {
						check.FirstVolumeDifference = bucket
					}
					check.MaximumQuoteVolumeDifference = math.Max(check.MaximumQuoteVolumeDifference, math.Abs(aggregate.QuoteVolume-bar.QuoteVolume)/math.Max(1, math.Abs(bar.QuoteVolume)))
				}
				check.MatchedPrices++
			}
			first = last
		}
		if check.MatchedPrices == 0 {
			return nil, archives, fmt.Errorf("no complete minute aggregates matched %s %s", symbol, interval)
		}
		checks[interval] = check
		fmt.Printf("%s %s minute prices verified=%d canonical_volume_differences=%d max_quote_diff=%.6f\n", symbol, interval, check.MatchedPrices, check.CanonicalVolumeDifferences, check.MaximumQuoteVolumeDifference)
	}
	return checks, archives, nil
}
