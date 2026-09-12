package historicalmarket

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

const SparseSecondInterval = "1s"

// LoadSparseSecondBars reads only already-persisted high-resolution rows. Unlike
// LoadKlines it never treats absent seconds as a canonical-history gap and never
// fetches the full requested range from the REST source.
func (repo *Repository) LoadSparseSecondBars(ctx context.Context, market, symbol string, start, end int64) ([]Kline, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateRange(market, symbol, start, end); err != nil {
		return nil, err
	}
	from := " FROM market_klines_1s"
	if repo.mysql() {
		from += " FORCE INDEX (market)"
	}
	query := "SELECT market,symbol,open_time,close_time,open_price,high_price,low_price,close_price,volume,quote_volume,trade_count,taker_buy_base_volume,taker_buy_quote_volume,source,source_ref" + from + " WHERE market=? AND symbol=? AND open_time>=? AND open_time<=? ORDER BY open_time"
	var rows []models.MarketKline1s
	if _, err := orm.NewOrm().Raw(query, market, strings.ToUpper(symbol), start, end).QueryRows(&rows); err != nil {
		return nil, err
	}
	out := make([]Kline, 0, len(rows))
	for _, row := range rows {
		out = append(out, Kline{Market: row.Market, Symbol: row.Symbol, Interval: SparseSecondInterval, Source: row.Source, SourceRef: row.SourceRef, OpenTime: row.OpenTime, CloseTime: row.CloseTime, Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume, QuoteVolume: row.QuoteVolume, TradeCount: row.TradeCount, TakerBuyBaseVolume: row.TakerBuyBaseVolume, TakerBuyQuoteVolume: row.TakerBuyQuoteVolume})
	}
	return out, ctx.Err()
}

func (repo *Repository) StoreSparseSecondBars(ctx context.Context, rows []Kline) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	normalized := make([]Kline, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Market) == "" {
			row.Market = MarketFuturesUSDT
		}
		row.Symbol = strings.ToUpper(strings.TrimSpace(row.Symbol))
		row.Interval = SparseSecondInterval
		if row.CloseTime == 0 && row.OpenTime > 0 {
			row.CloseTime = row.OpenTime + 999
		}
		if row.Source == "" {
			row.Source = SourceBinancePublicData
		}
		if err := validateSparseSecondKline(row); err != nil {
			return 0, err
		}
		normalized = append(normalized, row)
	}
	if len(normalized) == 0 {
		return 0, nil
	}
	tx, err := orm.NewOrm().Begin()
	if err != nil {
		return 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	written := 0
	for start := 0; start < len(normalized); start += repo.chunkSize(17) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		end := start + repo.chunkSize(17)
		if end > len(normalized) {
			end = len(normalized)
		}
		query, args := buildKlineUpsert("market_klines_1s", normalized[start:end], repo.mysql(), repo.nowMillis())
		if _, err := tx.Raw(query, args...).Exec(); err != nil {
			return 0, err
		}
		written += end - start
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	committed = true
	return written, nil
}

func (repo *Repository) LoadSparseTrades(ctx context.Context, market, symbol string, start, end int64) ([]PublicDataTrade, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateRange(market, symbol, start, end); err != nil {
		return nil, err
	}
	var rows []models.MarketTrade
	query := "SELECT market,symbol,trade_id,trade_time,price,quantity,quote_quantity,is_buyer_maker,source,source_ref,archive_sha256 FROM market_trades WHERE market=? AND symbol=? AND trade_time>=? AND trade_time<=? ORDER BY trade_time,trade_id"
	if _, err := orm.NewOrm().Raw(query, market, strings.ToUpper(symbol), start, end).QueryRows(&rows); err != nil {
		return nil, err
	}
	out := make([]PublicDataTrade, 0, len(rows))
	for _, row := range rows {
		out = append(out, PublicDataTrade{Market: row.Market, Symbol: row.Symbol, Source: row.Source, SourceRef: row.SourceRef, ArchiveSHA256: row.ArchiveSHA256, TradeID: row.TradeID, TradeTime: row.TradeTime, Price: row.Price, Quantity: row.Quantity, QuoteQuantity: row.QuoteQuantity, IsBuyerMaker: row.IsBuyerMaker})
	}
	return out, ctx.Err()
}

func (repo *Repository) StoreSparseTrades(ctx context.Context, rows []PublicDataTrade) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	normalized := make([]PublicDataTrade, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Market) == "" {
			row.Market = MarketFuturesUSDT
		}
		row.Symbol = strings.ToUpper(strings.TrimSpace(row.Symbol))
		if row.Source == "" {
			row.Source = SourceBinancePublicData
		}
		if err := validateSparseTrade(row); err != nil {
			return 0, err
		}
		normalized = append(normalized, row)
	}
	if len(normalized) == 0 {
		return 0, nil
	}
	tx, err := orm.NewOrm().Begin()
	if err != nil {
		return 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	written := 0
	for start := 0; start < len(normalized); start += repo.chunkSize(13) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		end := start + repo.chunkSize(13)
		if end > len(normalized) {
			end = len(normalized)
		}
		query, args := buildTradeUpsert(normalized[start:end], repo.mysql(), repo.nowMillis())
		if _, err := tx.Raw(query, args...).Exec(); err != nil {
			return 0, err
		}
		written += end - start
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	committed = true
	return written, nil
}

func buildTradeUpsert(rows []PublicDataTrade, mysql bool, now int64) (string, []interface{}) {
	values := make([]string, 0, len(rows))
	args := make([]interface{}, 0, len(rows)*13)
	for _, row := range rows {
		values = append(values, "(?,?,?,?,?,?,?,?,?,?,?,?,?)")
		args = append(args, row.Market, strings.ToUpper(row.Symbol), row.TradeID, row.TradeTime, row.Price, row.Quantity, row.QuoteQuantity, row.IsBuyerMaker, row.Source, row.SourceRef, row.ArchiveSHA256, now, now)
	}
	base := "INSERT INTO market_trades (market,symbol,trade_id,trade_time,price,quantity,quote_quantity,is_buyer_maker,source,source_ref,archive_sha256,created_at,updated_at) VALUES " + strings.Join(values, ",")
	if mysql {
		return base + " ON DUPLICATE KEY UPDATE trade_time=VALUES(trade_time),price=VALUES(price),quantity=VALUES(quantity),quote_quantity=VALUES(quote_quantity),is_buyer_maker=VALUES(is_buyer_maker),source=VALUES(source),source_ref=VALUES(source_ref),archive_sha256=VALUES(archive_sha256),updated_at=VALUES(updated_at)", args
	}
	return base + " ON CONFLICT(market,symbol,trade_id) DO UPDATE SET trade_time=excluded.trade_time,price=excluded.price,quantity=excluded.quantity,quote_quantity=excluded.quote_quantity,is_buyer_maker=excluded.is_buyer_maker,source=excluded.source,source_ref=excluded.source_ref,archive_sha256=excluded.archive_sha256,updated_at=excluded.updated_at", args
}

func validateSparseSecondKline(row Kline) error {
	if row.Interval != SparseSecondInterval {
		return fmt.Errorf("sparse second K-line interval must be %s", SparseSecondInterval)
	}
	if err := validateRange(row.Market, row.Symbol, row.OpenTime, row.CloseTime); err != nil {
		return err
	}
	if row.CloseTime-row.OpenTime > 999 || row.CloseTime < row.OpenTime {
		return fmt.Errorf("invalid sparse second K-line time range at %d", row.OpenTime)
	}
	if row.Open <= 0 || row.Close <= 0 || row.High < row.Low || row.Open < row.Low || row.Open > row.High || row.Close < row.Low || row.Close > row.High || row.Volume < 0 || row.QuoteVolume < 0 || row.TakerBuyBaseVolume < 0 || row.TakerBuyQuoteVolume < 0 || math.IsNaN(row.Close) || math.IsInf(row.Close, 0) {
		return fmt.Errorf("invalid sparse second K-line %s at %d", row.Symbol, row.OpenTime)
	}
	return nil
}

func validateSparseTrade(row PublicDataTrade) error {
	if strings.TrimSpace(row.Market) == "" || strings.TrimSpace(row.Symbol) == "" || row.TradeID < 0 || row.TradeTime <= 0 || row.Price <= 0 || row.Quantity < 0 || row.QuoteQuantity < 0 || math.IsNaN(row.Price) || math.IsInf(row.Price, 0) {
		return fmt.Errorf("invalid sparse trade %s id=%d time=%d", row.Symbol, row.TradeID, row.TradeTime)
	}
	if row.Source == SourceBinancePublicData && !validSHA256Hex(row.ArchiveSHA256) {
		return fmt.Errorf("public data trade %s id=%d has invalid archive SHA256", row.Symbol, row.TradeID)
	}
	return nil
}

func validSHA256Hex(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}
