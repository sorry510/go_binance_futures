package models

// MarketKline1m is the canonical global OHLCV shape. Other interval models use
// the same physical columns and only override TableName.
type MarketKline1m struct {
	ID                  int64   `orm:"column(id);auto" json:"id"`
	Market              string  `orm:"column(market);size(32)" json:"market"`
	Symbol              string  `orm:"column(symbol);size(32);index" json:"symbol"`
	OpenTime            int64   `orm:"column(open_time);index" json:"open_time"`
	CloseTime           int64   `orm:"column(close_time);index" json:"close_time"`
	Open                float64 `orm:"column(open_price);digits(30);decimals(12)" json:"open"`
	High                float64 `orm:"column(high_price);digits(30);decimals(12)" json:"high"`
	Low                 float64 `orm:"column(low_price);digits(30);decimals(12)" json:"low"`
	Close               float64 `orm:"column(close_price);digits(30);decimals(12)" json:"close"`
	Volume              float64 `orm:"column(volume);digits(40);decimals(12)" json:"volume"`
	QuoteVolume         float64 `orm:"column(quote_volume);digits(40);decimals(12)" json:"quote_volume"`
	TradeCount          int64   `orm:"column(trade_count)" json:"trade_count"`
	TakerBuyBaseVolume  float64 `orm:"column(taker_buy_base_volume);digits(40);decimals(12)" json:"taker_buy_base_volume"`
	TakerBuyQuoteVolume float64 `orm:"column(taker_buy_quote_volume);digits(40);decimals(12)" json:"taker_buy_quote_volume"`
	Source              string  `orm:"column(source);size(64);index" json:"source"`
	SourceRef           string  `orm:"column(source_ref);size(512);null" json:"source_ref,omitempty"`
	CreatedAt           int64   `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt           int64   `orm:"column(updated_at);index" json:"updated_at"`
}

func (*MarketKline1m) TableName() string       { return "market_klines_1m" }
func (*MarketKline1m) TableUnique() [][]string { return [][]string{{"Market", "Symbol", "OpenTime"}} }

type MarketKline3m MarketKline1m
type MarketKline5m MarketKline1m
type MarketKline15m MarketKline1m
type MarketKline30m MarketKline1m
type MarketKline1h MarketKline1m
type MarketKline2h MarketKline1m
type MarketKline4h MarketKline1m
type MarketKline6h MarketKline1m
type MarketKline8h MarketKline1m
type MarketKline12h MarketKline1m
type MarketKline1d MarketKline1m
type MarketKline3d MarketKline1m
type MarketKline1w MarketKline1m
type MarketKline1mo MarketKline1m

func (*MarketKline3m) TableName() string  { return "market_klines_3m" }
func (*MarketKline5m) TableName() string  { return "market_klines_5m" }
func (*MarketKline15m) TableName() string { return "market_klines_15m" }
func (*MarketKline30m) TableName() string { return "market_klines_30m" }
func (*MarketKline1h) TableName() string  { return "market_klines_1h" }
func (*MarketKline2h) TableName() string  { return "market_klines_2h" }
func (*MarketKline4h) TableName() string  { return "market_klines_4h" }
func (*MarketKline6h) TableName() string  { return "market_klines_6h" }
func (*MarketKline8h) TableName() string  { return "market_klines_8h" }
func (*MarketKline12h) TableName() string { return "market_klines_12h" }
func (*MarketKline1d) TableName() string  { return "market_klines_1d" }
func (*MarketKline3d) TableName() string  { return "market_klines_3d" }
func (*MarketKline1w) TableName() string  { return "market_klines_1w" }
func (*MarketKline1mo) TableName() string { return "market_klines_1mo" }

func (*MarketKline3m) TableUnique() [][]string  { return marketKlineUnique() }
func (*MarketKline5m) TableUnique() [][]string  { return marketKlineUnique() }
func (*MarketKline15m) TableUnique() [][]string { return marketKlineUnique() }
func (*MarketKline30m) TableUnique() [][]string { return marketKlineUnique() }
func (*MarketKline1h) TableUnique() [][]string  { return marketKlineUnique() }
func (*MarketKline2h) TableUnique() [][]string  { return marketKlineUnique() }
func (*MarketKline4h) TableUnique() [][]string  { return marketKlineUnique() }
func (*MarketKline6h) TableUnique() [][]string  { return marketKlineUnique() }
func (*MarketKline8h) TableUnique() [][]string  { return marketKlineUnique() }
func (*MarketKline12h) TableUnique() [][]string { return marketKlineUnique() }
func (*MarketKline1d) TableUnique() [][]string  { return marketKlineUnique() }
func (*MarketKline3d) TableUnique() [][]string  { return marketKlineUnique() }
func (*MarketKline1w) TableUnique() [][]string  { return marketKlineUnique() }
func (*MarketKline1mo) TableUnique() [][]string { return marketKlineUnique() }
func marketKlineUnique() [][]string             { return [][]string{{"Market", "Symbol", "OpenTime"}} }

// MarketFundingRate is globally shared because funding history is much smaller
// than Kline history and has no Kline interval dimension.
type MarketFundingRate struct {
	ID          int64   `orm:"column(id);auto" json:"id"`
	Market      string  `orm:"column(market);size(32)" json:"market"`
	Symbol      string  `orm:"column(symbol);size(32);index" json:"symbol"`
	FundingTime int64   `orm:"column(funding_time);index" json:"funding_time"`
	FundingRate float64 `orm:"column(funding_rate);digits(20);decimals(12)" json:"funding_rate"`
	MarkPrice   float64 `orm:"column(mark_price);digits(30);decimals(12)" json:"mark_price"`
	Source      string  `orm:"column(source);size(64);index" json:"source"`
	SourceRef   string  `orm:"column(source_ref);size(512);null" json:"source_ref,omitempty"`
	CreatedAt   int64   `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt   int64   `orm:"column(updated_at);index" json:"updated_at"`
}

func (*MarketFundingRate) TableName() string { return "market_funding_rates" }
func (*MarketFundingRate) TableUnique() [][]string {
	return [][]string{{"Market", "Symbol", "FundingTime"}}
}

// MarketDataImportBatch records both Binance gap fills and explicit external imports.
type MarketDataImportBatch struct {
	ID          int64  `orm:"column(id);auto" json:"id"`
	BatchID     string `orm:"column(batch_id);size(64);unique" json:"batch_id"`
	Source      string `orm:"column(source);size(64);index" json:"source"`
	Kind        string `orm:"column(kind);size(32);index" json:"kind"`
	Status      string `orm:"column(status);size(32);index" json:"status"`
	TotalRows   int    `orm:"column(total_rows)" json:"total_rows"`
	WrittenRows int    `orm:"column(written_rows)" json:"written_rows"`
	InvalidRows int    `orm:"column(invalid_rows)" json:"invalid_rows"`
	Error       string `orm:"column(error);type(text);null" json:"error,omitempty"`
	SourceRef   string `orm:"column(source_ref);size(512);null" json:"source_ref,omitempty"`
	CreatedAt   int64  `orm:"column(created_at);index" json:"created_at"`
	CompletedAt int64  `orm:"column(completed_at);index" json:"completed_at,omitempty"`
}

func (*MarketDataImportBatch) TableName() string { return "market_data_import_batches" }
