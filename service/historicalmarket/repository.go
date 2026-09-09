package historicalmarket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
)

type Repository struct {
	Source Source
	Now    func() time.Time
}

func NewRepository(source Source) *Repository { return &Repository{Source: source, Now: time.Now} }
func DefaultRepository() *Repository          { return NewRepository(BinanceSource{}) }

func (repo *Repository) LoadKlines(ctx context.Context, market, symbol, interval string, start, end int64) ([]Kline, error) {
	if err := validateRange(market, symbol, start, end); err != nil {
		return nil, err
	}
	if _, err := KlineTable(interval); err != nil {
		return nil, err
	}
	rows, err := repo.queryKlines(market, symbol, interval, start, end)
	if err != nil {
		return nil, err
	}
	missing, err := missingKlineRanges(interval, start, end, rows)
	if err != nil {
		return nil, err
	}
	if len(missing) > 0 && repo.Source != nil {
		for _, gap := range missing {
			remote, err := repo.Source.Klines(ctx, market, symbol, interval, gap[0], gap[1])
			if err != nil {
				return nil, err
			}
			if len(remote) > 0 {
				if _, err := repo.Import(ctx, ImportRequest{Source: SourceBinanceREST, SourceRef: fmt.Sprintf("%s:%s:%d-%d", symbol, interval, gap[0], gap[1]), Klines: remote}); err != nil {
					return nil, err
				}
			}
		}
		rows, err = repo.queryKlines(market, symbol, interval, start, end)
		if err != nil {
			return nil, err
		}
		missing, err = missingKlineRanges(interval, start, end, rows)
		if err != nil {
			return nil, err
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("historical K-line gaps remain for %s %s: %v", symbol, interval, missing)
	}
	return rows, nil
}

func (repo *Repository) LoadFunding(ctx context.Context, market, symbol string, start, end int64) ([]FundingRate, error) {
	if err := validateRange(market, symbol, start, end); err != nil {
		return nil, err
	}
	rows, err := repo.queryFunding(market, symbol, start, end)
	if err != nil {
		return nil, err
	}
	missing := fundingMissingRanges(start, end, rows)
	if len(missing) > 0 && repo.Source != nil {
		for _, gap := range missing {
			remote, err := repo.Source.Funding(ctx, market, symbol, gap[0], gap[1])
			if err != nil {
				return nil, err
			}
			if len(remote) > 0 {
				if _, err := repo.Import(ctx, ImportRequest{Source: SourceBinanceREST, SourceRef: fmt.Sprintf("%s:funding:%d-%d", symbol, gap[0], gap[1]), Funding: remote}); err != nil {
					return nil, err
				}
			}
		}
		rows, err = repo.queryFunding(market, symbol, start, end)
		if err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (repo *Repository) Import(ctx context.Context, request ImportRequest) (ImportResult, error) {
	if err := ctx.Err(); err != nil {
		return ImportResult{}, err
	}
	request.Source = strings.TrimSpace(request.Source)
	if request.Source == "" {
		request.Source = "external"
	}
	result := ImportResult{BatchID: newBatchID(), TotalRows: len(request.Klines) + len(request.Funding)}
	now := repo.nowMillis()
	batch := models.MarketDataImportBatch{BatchID: result.BatchID, Source: request.Source, Kind: importKind(request), Status: "running", TotalRows: result.TotalRows, SourceRef: request.SourceRef, CreatedAt: now}
	if _, err := orm.NewOrm().Insert(&batch); err != nil {
		return result, err
	}
	finish := func(status string, err error) {
		params := orm.Params{"status": status, "written_rows": result.WrittenRows, "invalid_rows": result.InvalidRows, "completed_at": repo.nowMillis()}
		if err != nil {
			params["error"] = err.Error()
		}
		_, _ = orm.NewOrm().QueryTable(new(models.MarketDataImportBatch)).Filter("batch_id", result.BatchID).Update(params)
	}
	for i := range request.Klines {
		if strings.TrimSpace(request.Klines[i].Market) == "" {
			request.Klines[i].Market = MarketFuturesUSDT
		}
		request.Klines[i].Symbol = strings.ToUpper(strings.TrimSpace(request.Klines[i].Symbol))
		if request.Klines[i].CloseTime == 0 && request.Klines[i].OpenTime > 0 {
			closeTime, closeErr := barClose(request.Klines[i].Interval, request.Klines[i].OpenTime)
			if closeErr != nil {
				result.InvalidRows++
				finish("failed", closeErr)
				return result, closeErr
			}
			request.Klines[i].CloseTime = closeTime
		}
		request.Klines[i].Source = request.Source
		if request.Klines[i].SourceRef == "" {
			request.Klines[i].SourceRef = request.SourceRef
		}
		if err := validateKline(request.Klines[i]); err != nil {
			result.InvalidRows++
			finish("failed", err)
			return result, err
		}
	}
	for i := range request.Funding {
		if strings.TrimSpace(request.Funding[i].Market) == "" {
			request.Funding[i].Market = MarketFuturesUSDT
		}
		request.Funding[i].Symbol = strings.ToUpper(strings.TrimSpace(request.Funding[i].Symbol))
		request.Funding[i].Source = request.Source
		if request.Funding[i].SourceRef == "" {
			request.Funding[i].SourceRef = request.SourceRef
		}
		if err := validateFunding(request.Funding[i]); err != nil {
			result.InvalidRows++
			finish("failed", err)
			return result, err
		}
	}
	if len(request.Klines) > 0 {
		written, err := repo.upsertKlines(request.Klines)
		result.WrittenRows += written
		if err != nil {
			finish("failed", err)
			return result, err
		}
	}
	if len(request.Funding) > 0 {
		written, err := repo.upsertFunding(request.Funding)
		result.WrittenRows += written
		if err != nil {
			finish("failed", err)
			return result, err
		}
	}
	finish("succeeded", nil)
	return result, nil
}

func (repo *Repository) queryKlines(market, symbol, interval string, start, end int64) ([]Kline, error) {
	table, err := KlineTable(interval)
	if err != nil {
		return nil, err
	}
	first, last, ok, err := expectedKlineBounds(interval, start, end)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []Kline{}, nil
	}
	var rows []models.MarketKline1m
	query := "SELECT market,symbol,open_time,close_time,open_price,high_price,low_price,close_price,volume,quote_volume,trade_count,taker_buy_base_volume,taker_buy_quote_volume,source,source_ref,created_at,updated_at FROM " + table + " WHERE market=? AND symbol=? AND open_time>=? AND open_time<=? ORDER BY open_time"
	if _, err := orm.NewOrm().Raw(query, market, strings.ToUpper(symbol), first, last).QueryRows(&rows); err != nil {
		return nil, err
	}
	out := make([]Kline, 0, len(rows))
	for _, row := range rows {
		out = append(out, Kline{Market: row.Market, Symbol: row.Symbol, Interval: interval, OpenTime: row.OpenTime, CloseTime: row.CloseTime, Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume, QuoteVolume: row.QuoteVolume, TradeCount: row.TradeCount, TakerBuyBaseVolume: row.TakerBuyBaseVolume, TakerBuyQuoteVolume: row.TakerBuyQuoteVolume, Source: row.Source, SourceRef: row.SourceRef})
	}
	return out, nil
}

func (repo *Repository) queryFunding(market, symbol string, start, end int64) ([]FundingRate, error) {
	var rows []models.MarketFundingRate
	_, err := orm.NewOrm().QueryTable(new(models.MarketFundingRate)).Filter("market", market).Filter("symbol", strings.ToUpper(symbol)).Filter("funding_time__gte", start).Filter("funding_time__lte", end).OrderBy("funding_time").All(&rows)
	if err != nil {
		return nil, err
	}
	out := make([]FundingRate, 0, len(rows))
	for _, row := range rows {
		out = append(out, FundingRate{Market: row.Market, Symbol: row.Symbol, FundingTime: row.FundingTime, FundingRate: row.FundingRate, MarkPrice: row.MarkPrice, Source: row.Source, SourceRef: row.SourceRef})
	}
	return out, nil
}

func (repo *Repository) upsertKlines(rows []Kline) (int, error) {
	groups := map[string][]Kline{}
	for _, row := range rows {
		groups[row.Interval] = append(groups[row.Interval], row)
	}
	written := 0
	for interval, group := range groups {
		table, err := KlineTable(interval)
		if err != nil {
			return written, err
		}
		for start := 0; start < len(group); start += repo.chunkSize(17) {
			end := start + repo.chunkSize(17)
			if end > len(group) {
				end = len(group)
			}
			query, args := buildKlineUpsert(table, group[start:end], repo.mysql(), repo.nowMillis())
			if _, err := orm.NewOrm().Raw(query, args...).Exec(); err != nil {
				return written, err
			}
			written += end - start
		}
	}
	return written, nil
}

func (repo *Repository) upsertFunding(rows []FundingRate) (int, error) {
	written := 0
	for start := 0; start < len(rows); start += repo.chunkSize(10) {
		end := start + repo.chunkSize(10)
		if end > len(rows) {
			end = len(rows)
		}
		query, args := buildFundingUpsert(rows[start:end], repo.mysql(), repo.nowMillis())
		if _, err := orm.NewOrm().Raw(query, args...).Exec(); err != nil {
			return written, err
		}
		written += end - start
	}
	return written, nil
}

func buildKlineUpsert(table string, rows []Kline, mysql bool, now int64) (string, []interface{}) {
	values := make([]string, 0, len(rows))
	args := make([]interface{}, 0, len(rows)*17)
	for _, row := range rows {
		values = append(values, "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
		args = append(args, row.Market, strings.ToUpper(row.Symbol), row.OpenTime, row.CloseTime, row.Open, row.High, row.Low, row.Close, row.Volume, row.QuoteVolume, row.TradeCount, row.TakerBuyBaseVolume, row.TakerBuyQuoteVolume, row.Source, row.SourceRef, now, now)
	}
	base := "INSERT INTO " + table + " (market,symbol,open_time,close_time,open_price,high_price,low_price,close_price,volume,quote_volume,trade_count,taker_buy_base_volume,taker_buy_quote_volume,source,source_ref,created_at,updated_at) VALUES " + strings.Join(values, ",")
	if mysql {
		return base + " ON DUPLICATE KEY UPDATE close_time=VALUES(close_time),open_price=VALUES(open_price),high_price=VALUES(high_price),low_price=VALUES(low_price),close_price=VALUES(close_price),volume=VALUES(volume),quote_volume=VALUES(quote_volume),trade_count=VALUES(trade_count),taker_buy_base_volume=VALUES(taker_buy_base_volume),taker_buy_quote_volume=VALUES(taker_buy_quote_volume),source=VALUES(source),source_ref=VALUES(source_ref),updated_at=VALUES(updated_at)", args
	}
	return base + " ON CONFLICT(market,symbol,open_time) DO UPDATE SET close_time=excluded.close_time,open_price=excluded.open_price,high_price=excluded.high_price,low_price=excluded.low_price,close_price=excluded.close_price,volume=excluded.volume,quote_volume=excluded.quote_volume,trade_count=excluded.trade_count,taker_buy_base_volume=excluded.taker_buy_base_volume,taker_buy_quote_volume=excluded.taker_buy_quote_volume,source=excluded.source,source_ref=excluded.source_ref,updated_at=excluded.updated_at", args
}

func buildFundingUpsert(rows []FundingRate, mysql bool, now int64) (string, []interface{}) {
	values := make([]string, 0, len(rows))
	args := make([]interface{}, 0, len(rows)*9)
	for _, row := range rows {
		values = append(values, "(?,?,?,?,?,?,?,?,?)")
		args = append(args, row.Market, strings.ToUpper(row.Symbol), row.FundingTime, row.FundingRate, row.MarkPrice, row.Source, row.SourceRef, now, now)
	}
	base := "INSERT INTO market_funding_rates (market,symbol,funding_time,funding_rate,mark_price,source,source_ref,created_at,updated_at) VALUES " + strings.Join(values, ",")
	if mysql {
		return base + " ON DUPLICATE KEY UPDATE funding_rate=VALUES(funding_rate),mark_price=VALUES(mark_price),source=VALUES(source),source_ref=VALUES(source_ref),updated_at=VALUES(updated_at)", args
	}
	return base + " ON CONFLICT(market,symbol,funding_time) DO UPDATE SET funding_rate=excluded.funding_rate,mark_price=excluded.mark_price,source=excluded.source,source_ref=excluded.source_ref,updated_at=excluded.updated_at", args
}

func missingKlineRanges(interval string, start, end int64, rows []Kline) ([][2]int64, error) {
	first, last, ok, err := expectedKlineBounds(interval, start, end)
	if err != nil || !ok {
		return nil, err
	}
	byOpen := make(map[int64]Kline, len(rows))
	for _, row := range rows {
		byOpen[row.OpenTime] = row
	}
	missing := make([][2]int64, 0)
	var gapStart int64
	for cursor := first; cursor <= last; {
		_, exists := byOpen[cursor]
		if !exists && gapStart == 0 {
			gapStart = cursor
		}
		next, err := nextOpen(interval, cursor)
		if err != nil {
			return nil, err
		}
		if exists && gapStart != 0 {
			prev, _ := previousOpen(interval, cursor)
			close, _ := barClose(interval, prev)
			missing = append(missing, [2]int64{gapStart, close})
			gapStart = 0
		}
		if next <= cursor {
			return nil, fmt.Errorf("invalid interval progression %s", interval)
		}
		cursor = next
	}
	if gapStart != 0 {
		close, _ := barClose(interval, last)
		missing = append(missing, [2]int64{gapStart, close})
	}
	return missing, nil
}

func fundingMissingRanges(start, end int64, rows []FundingRate) [][2]int64 {
	if len(rows) == 0 {
		return [][2]int64{{start, end}}
	}
	const maxGap = int64(12 * time.Hour / time.Millisecond)
	result := make([][2]int64, 0)
	if rows[0].FundingTime-start > maxGap {
		result = append(result, [2]int64{start, rows[0].FundingTime - 1})
	}
	for i := 1; i < len(rows); i++ {
		if rows[i].FundingTime-rows[i-1].FundingTime > maxGap {
			result = append(result, [2]int64{rows[i-1].FundingTime + 1, rows[i].FundingTime - 1})
		}
	}
	if end-rows[len(rows)-1].FundingTime > maxGap {
		result = append(result, [2]int64{rows[len(rows)-1].FundingTime + 1, end})
	}
	return result
}

func expectedKlineBounds(interval string, start, end int64) (int64, int64, bool, error) {
	first, err := ceilOpen(interval, start)
	if err != nil {
		return 0, 0, false, err
	}
	candidate, err := floorOpen(interval, end)
	if err != nil {
		return 0, 0, false, err
	}
	close, err := barClose(interval, candidate)
	if err != nil {
		return 0, 0, false, err
	}
	if close > end {
		candidate, err = previousOpen(interval, candidate)
		if err != nil {
			return 0, 0, false, err
		}
	}
	if candidate < first {
		return 0, 0, false, nil
	}
	return first, candidate, true, nil
}

func ceilOpen(interval string, value int64) (int64, error) {
	floor, err := floorOpen(interval, value)
	if err != nil {
		return 0, err
	}
	if floor < value {
		return nextOpen(interval, floor)
	}
	return floor, nil
}
func floorOpen(interval string, value int64) (int64, error) {
	t := time.UnixMilli(value).UTC()
	if interval == "1M" {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).UnixMilli(), nil
	}
	if interval == "1w" {
		days := (int(t.Weekday()) + 6) % 7
		return time.Date(t.Year(), t.Month(), t.Day()-days, 0, 0, 0, 0, time.UTC).UnixMilli(), nil
	}
	d, err := fixedDuration(interval)
	if err != nil {
		return 0, err
	}
	ms := d.Milliseconds()
	return value - positiveMod(value, ms), nil
}
func nextOpen(interval string, open int64) (int64, error) {
	if interval == "1M" {
		return time.UnixMilli(open).UTC().AddDate(0, 1, 0).UnixMilli(), nil
	}
	d, err := fixedDuration(interval)
	if err != nil {
		return 0, err
	}
	return open + d.Milliseconds(), nil
}
func previousOpen(interval string, open int64) (int64, error) {
	if interval == "1M" {
		return time.UnixMilli(open).UTC().AddDate(0, -1, 0).UnixMilli(), nil
	}
	d, err := fixedDuration(interval)
	if err != nil {
		return 0, err
	}
	return open - d.Milliseconds(), nil
}
func barClose(interval string, open int64) (int64, error) {
	next, err := nextOpen(interval, open)
	if err != nil {
		return 0, err
	}
	return next - 1, nil
}
func fixedDuration(interval string) (time.Duration, error) {
	switch interval {
	case "1m":
		return time.Minute, nil
	case "3m":
		return 3 * time.Minute, nil
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "30m":
		return 30 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "2h":
		return 2 * time.Hour, nil
	case "4h":
		return 4 * time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "8h":
		return 8 * time.Hour, nil
	case "12h":
		return 12 * time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	case "3d":
		return 72 * time.Hour, nil
	case "1w":
		return 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported historical K-line interval %q", interval)
	}
}
func positiveMod(value, mod int64) int64 {
	r := value % mod
	if r < 0 {
		r += mod
	}
	return r
}
func validateRange(market, symbol string, start, end int64) error {
	if strings.TrimSpace(market) == "" || strings.TrimSpace(symbol) == "" || start <= 0 || end <= 0 || start > end {
		return fmt.Errorf("market, symbol and valid historical range are required")
	}
	return nil
}
func validateKline(row Kline) error {
	if err := validateRange(row.Market, row.Symbol, row.OpenTime, row.CloseTime); err != nil {
		return err
	}
	if _, err := KlineTable(row.Interval); err != nil {
		return err
	}
	if row.Open <= 0 || row.Close <= 0 || row.High < row.Low || row.Open < row.Low || row.Open > row.High || row.Close < row.Low || row.Close > row.High || row.Volume < 0 || row.QuoteVolume < 0 || row.TakerBuyBaseVolume < 0 || row.TakerBuyQuoteVolume < 0 || math.IsNaN(row.Close) {
		return fmt.Errorf("invalid historical K-line %s %s at %d", row.Symbol, row.Interval, row.OpenTime)
	}
	return nil
}
func validateFunding(row FundingRate) error {
	if strings.TrimSpace(row.Market) == "" || strings.TrimSpace(row.Symbol) == "" || row.FundingTime <= 0 || math.IsNaN(row.FundingRate) || math.IsInf(row.FundingRate, 0) || row.MarkPrice < 0 {
		return fmt.Errorf("invalid historical funding row %s at %d", row.Symbol, row.FundingTime)
	}
	return nil
}
func importKind(request ImportRequest) string {
	if len(request.Klines) > 0 && len(request.Funding) > 0 {
		return "mixed"
	}
	if len(request.Klines) > 0 {
		return "kline"
	}
	return "funding"
}
func newBatchID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err == nil {
		return "mdi_" + hex.EncodeToString(b)
	}
	return fmt.Sprintf("mdi_%d", time.Now().UnixNano())
}
func (repo *Repository) nowMillis() int64 {
	if repo.Now != nil {
		return repo.Now().UnixMilli()
	}
	return time.Now().UnixMilli()
}
func (repo *Repository) mysql() bool {
	if db, err := orm.GetDB("default"); err == nil {
		driverType := strings.ToLower(fmt.Sprintf("%T", db.Driver()))
		if strings.Contains(driverType, "mysql") {
			return true
		}
		if strings.Contains(driverType, "sqlite") || strings.Contains(driverType, "pq") || strings.Contains(driverType, "postgres") {
			return false
		}
	}
	driver, _ := config.String("database::driver")
	return strings.EqualFold(strings.TrimSpace(driver), "mysql")
}
func (repo *Repository) chunkSize(columns int) int {
	if repo.mysql() {
		return 200
	}
	size := 900 / columns
	if size < 1 {
		return 1
	}
	return size
}

func SortKlines(rows []Kline) {
	sort.Slice(rows, func(i, j int) bool { return rows[i].OpenTime < rows[j].OpenTime })
}
