package futures

type Config struct {
	ApiKey string
	SecretKey string
	ProxyUrl string
	BaseUrl string
	WsBaseUrl string
}

type AccountAsset struct {
	Asset                  string `json:"asset"`
	InitialMargin          string `json:"initialMargin"`
	MaintMargin            string `json:"maintMargin"`
	MarginBalance          string `json:"marginBalance"`
	MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	WalletBalance          string `json:"walletBalance"`
	CrossWalletBalance     string `json:"crossWalletBalance"`
	CrossUnPnl             string `json:"crossUnPnl"`
	AvailableBalance       string `json:"availableBalance"`
	MarginAvailable        bool   `json:"marginAvailable"`
	UpdateTime             int64  `json:"updateTime"`
}

// AccountPosition define account position
type AccountPosition struct {
	Isolated               bool             `json:"isolated"`
	Leverage               string           `json:"leverage"`
	InitialMargin          string           `json:"initialMargin"`
	MaintMargin            string           `json:"maintMargin"`
	OpenOrderInitialMargin string           `json:"openOrderInitialMargin"`
	PositionInitialMargin  string           `json:"positionInitialMargin"`
	Symbol                 string           `json:"symbol"`
	UnrealizedProfit       string           `json:"unrealizedProfit"`
	EntryPrice             string           `json:"entryPrice"`
	MaxNotional            string           `json:"maxNotional"`
	PositionSide           string           `json:"positionSide"`
	PositionAmt            string           `json:"positionAmt"`
	Notional               string           `json:"notional"`
	BidNotional            string           `json:"bidNotional"`
	AskNotional            string           `json:"askNotional"`
	UpdateTime             int64            `json:"updateTime"`
}

type Account struct {
	Assets                      []*AccountAsset    `json:"assets"`
	FeeTier                     int                `json:"feeTier"`
	CanTrade                    bool               `json:"canTrade"`
	CanDeposit                  bool               `json:"canDeposit"`
	CanWithdraw                 bool               `json:"canWithdraw"`
	UpdateTime                  int64              `json:"updateTime"`
	MultiAssetsMargin           bool               `json:"multiAssetsMargin"`
	TotalInitialMargin          string             `json:"totalInitialMargin"`
	TotalMaintMargin            string             `json:"totalMaintMargin"`
	TotalWalletBalance          string             `json:"totalWalletBalance"`
	TotalUnrealizedProfit       string             `json:"totalUnrealizedProfit"`
	TotalMarginBalance          string             `json:"totalMarginBalance"`
	TotalPositionInitialMargin  string             `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin string             `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     string             `json:"totalCrossWalletBalance"`
	TotalCrossUnPnl             string             `json:"totalCrossUnPnl"`
	AvailableBalance            string             `json:"availableBalance"`
	MaxWithdrawAmount           string             `json:"maxWithdrawAmount"`
	Positions                   []*AccountPosition `json:"positions"`
}

type PositionParams struct {
	Symbol string
}

type Position struct {
	EntryPrice       string `json:"entryPrice"`
	BreakEvenPrice   string `json:"breakEvenPrice"`
	MarginType       string `json:"marginType"`
	IsAutoAddMargin  string `json:"isAutoAddMargin"`
	IsolatedMargin   string `json:"isolatedMargin"`
	Leverage         string `json:"leverage"`
	LiquidationPrice string `json:"liquidationPrice"`
	MarkPrice        string `json:"markPrice"`
	MaxNotionalValue string `json:"maxNotionalValue"`
	PositionAmt      string `json:"positionAmt"`
	Symbol           string `json:"symbol"`
	UnRealizedProfit string `json:"unRealizedProfit"`
	PositionSide     string `json:"positionSide"`
	Notional         string `json:"notional"`
	IsolatedWallet   string `json:"isolatedWallet"`
}

type DepthParams struct {
	Symbol string
	Limit int
}

type Depth struct {
	LastUpdateID int64 `json:"lastUpdateId"`
	Time         int64 `json:"time"`
	TradeTime    int64 `json:"timeTime"`
	Bids         []PriceLevel `json:"bids"`
	Asks         []PriceLevel `json:"asks"`
}

type PriceLevel struct {
	Price    string
	Quantity string
}

type TicketPriceParams struct {
	Symbol string
}

type TickerPrice struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type KlineParams struct {
	Symbol string
	Interval string
	Limit int
}

type Kline struct {
	OpenTime                 int64  `json:"openTime"`
	Open                     string `json:"open"`
	High                     string `json:"high"`
	Low                      string `json:"low"`
	Close                    string `json:"close"`
	Volume                   string `json:"volume"`
	CloseTime                int64  `json:"closeTime"`
	QuoteAssetVolume         string `json:"quoteAssetVolume"`
	TradeNum                 int64  `json:"tradeNum"`
	TakerBuyBaseAssetVolume  string `json:"takerBuyBaseAssetVolume"`
	TakerBuyQuoteAssetVolume string `json:"takerBuyQuoteAssetVolume"`
}

type FundingRateParams struct {
	Symbol    string
	StartTime int64
	EndTime   int64
	Limit     int
}

type FundingRate struct {
	Symbol               string `json:"symbol"`
    MarkPrice            string `json:"markPrice"`
    IndexPrice           string `json:"indexPrice"`
    EstimatedSettlePrice string `json:"estimatedSettlePrice"`
    LastFundingRate      string `json:"lastFundingRate"`
    NextFundingTime      int64  `json:"nextFundingTime"`
    InterestRate         string `json:"interestRate"`
    Time                 int64  `json:"time"`
}

type FundingRateHistoryParams struct {
	FundingRateParams
}

type FundingRateHistory struct {
	Symbol      string `json:"symbol"`
    FundingRate string `json:"fundingRate"`
    FundingTime int64  `json:"fundingTime"`
    MarkPrice   string `json:"markPrice"`
}

// 下单
type CreateOrderParams struct {
	Symbol string
	Quantity float64
	Price float64
	PositionSide string
	Side string
	Type string
	StopPrice float64
}

type CreateOrder struct {
	Symbol                  string           `json:"symbol"`                      //
	OrderID                 int64            `json:"orderId"`                     //
	ClientOrderID           string           `json:"clientOrderId"`               //
	Price                   string           `json:"price"`                       //
	OrigQuantity            string           `json:"origQty"`                     //
	ExecutedQuantity        string           `json:"executedQty"`                 //
	CumQuote                string           `json:"cumQuote"`                    //
	ReduceOnly              bool             `json:"reduceOnly"`                  //
	Status                  string           `json:"status"`                      //
	StopPrice               string           `json:"stopPrice"`                   // please ignore when order type is TRAILING_STOP_MARKET
	TimeInForce             string           `json:"timeInForce"`                 //
	Type                    string           `json:"type"`                        //
	Side                    string           `json:"side"`                        //
	UpdateTime              int64            `json:"updateTime"`                  // update time
	WorkingType             string           `json:"workingType"`                 //
	ActivatePrice           string           `json:"activatePrice"`               // activation price, only return with TRAILING_STOP_MARKET order
	PriceRate               string           `json:"priceRate"`                   // callback rate, only return with TRAILING_STOP_MARKET order
	AvgPrice                string           `json:"avgPrice"`                    //
	PositionSide            string           `json:"positionSide"`                //
	ClosePosition           bool             `json:"closePosition"`               // if Close-All
	PriceProtect            bool             `json:"priceProtect"`                // if conditional order trigger is protected
	PriceMatch              string           `json:"priceMatch"`                  // price match mode
	SelfTradePreventionMode string           `json:"selfTradePreventionMode"`     // self trading prevention mode
	GoodTillDate            int64            `json:"goodTillDate"`                // order pre-set auto cancel time for TIF GTD order
	CumQty                  string           `json:"cumQty"`                      //
	OrigType                string           `json:"origType"`                    //
	RateLimitOrder10s       string           `json:"rateLimitOrder10s,omitempty"` //
	RateLimitOrder1m        string           `json:"rateLimitOrder1m,omitempty"`  //
}

type CancelOrderParams struct {
	Symbol string
	OrderID int64
}

type CancelOrder struct {
	ClientOrderID    string           `json:"clientOrderId"`
	CumQuantity      string           `json:"cumQty"` // deprecated: use ExecutedQuantity instead
	CumQuote         string           `json:"cumQuote"`
	ExecutedQuantity string           `json:"executedQty"`
	OrderID          int64            `json:"orderId"`
	OrigQuantity     string           `json:"origQty"`
	Price            string           `json:"price"`
	ReduceOnly       bool             `json:"reduceOnly"`
	Side             string           `json:"side"`
	Status           string           `json:"status"`
	StopPrice        string           `json:"stopPrice"`
	Symbol           string           `json:"symbol"`
	TimeInForce      string           `json:"timeInForce"`
	Type             string           `json:"type"`
	UpdateTime       int64            `json:"updateTime"`
	WorkingType      string           `json:"workingType"`
	ActivatePrice    string           `json:"activatePrice"`
	PriceRate        string           `json:"priceRate"`
	OrigType         string           `json:"origType"`
	PositionSide     string           `json:"positionSide"`
	PriceProtect     bool             `json:"priceProtect"`
}

type OrderParams struct {
	Symbol    string
	OrderID   int64
	StartTime int64
	EndTime   int64
	Limit     int
}

type Order struct {
	Symbol                  string           `json:"symbol"`
	OrderID                 int64            `json:"orderId"`
	ClientOrderID           string           `json:"clientOrderId"`
	Price                   string           `json:"price"`
	ReduceOnly              bool             `json:"reduceOnly"`
	OrigQuantity            string           `json:"origQty"`
	ExecutedQuantity        string           `json:"executedQty"`
	CumQuantity             string           `json:"cumQty"` // deprecated: use ExecutedQuantity instead
	CumQuote                string           `json:"cumQuote"`
	Status                  string           `json:"status"`
	TimeInForce             string           `json:"timeInForce"`
	Type                    string           `json:"type"`
	Side                    string           `json:"side"`
	StopPrice               string           `json:"stopPrice"`
	Time                    int64            `json:"time"`
	UpdateTime              int64            `json:"updateTime"`
	WorkingType             string           `json:"workingType"`
	ActivatePrice           string           `json:"activatePrice"`
	PriceRate               string           `json:"priceRate"`
	AvgPrice                string           `json:"avgPrice"`
	OrigType                string           `json:"origType"`
	PositionSide            string           `json:"positionSide"`
	PriceProtect            bool             `json:"priceProtect"`
	ClosePosition           bool             `json:"closePosition"`
	PriceMatch              string           `json:"priceMatch"`
	SelfTradePreventionMode string           `json:"selfTradePreventionMode"`
	GoodTillDate            int64            `json:"goodTillDate"`
}

type LeverageParams struct {
	Symbol string
	Leverage int
}

type Leverage struct {
	Leverage         int    `json:"leverage"`
	MaxNotionalValue string `json:"maxNotionalValue"`
	Symbol           string `json:"symbol"`
}

type MarginTypeParams struct {
	Symbol string
	MarginType string
}

type RateLimit struct {
    RateLimitType string `json:"rateLimitType"`
    Interval      string `json:"interval"`
    IntervalNum   int64  `json:"intervalNum"`
    Limit         int64  `json:"limit"`
}

type Symbol struct {
	Symbol                string                   `json:"symbol"`
	Pair                  string                   `json:"pair"`
	ContractType          string                   `json:"contractType"`
	DeliveryDate          int64                    `json:"deliveryDate"`
	OnboardDate           int64                    `json:"onboardDate"`
	Status                string                   `json:"status"`
	MaintMarginPercent    string                   `json:"maintMarginPercent"`
	RequiredMarginPercent string                   `json:"requiredMarginPercent"`
	PricePrecision        int                      `json:"pricePrecision"`
	QuantityPrecision     int                      `json:"quantityPrecision"`
	BaseAssetPrecision    int                      `json:"baseAssetPrecision"`
	QuotePrecision        int                      `json:"quotePrecision"`
	UnderlyingType        string                   `json:"underlyingType"`
	UnderlyingSubType     []string                 `json:"underlyingSubType"`
	SettlePlan            int64                    `json:"settlePlan"`
	TriggerProtect        string                   `json:"triggerProtect"`
	OrderType             []string                 `json:"orderType"`
	TimeInForce           []string                 `json:"timeInForce"`
	Filters               []map[string]interface{} `json:"filters"`
	QuoteAsset            string                   `json:"quoteAsset"`
	MarginAsset           string                   `json:"marginAsset"`
	BaseAsset             string                   `json:"baseAsset"`
	LiquidationFee        string                   `json:"liquidationFee"`
	MarketTakeBound       string                   `json:"marketTakeBound"`
}

type ExchangeInfo struct {
	Timezone        string        `json:"timezone"`
	ServerTime      int64         `json:"serverTime"`
	RateLimits      []RateLimit   `json:"rateLimits"`
	ExchangeFilters []interface{} `json:"exchangeFilters"`
	Symbols         []Symbol      `json:"symbols"`
}