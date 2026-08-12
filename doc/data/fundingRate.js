// https://binance-docs.github.io/apidocs/futures/cn/#69f9b0b2f3
// 最新标记价格和资金费率
const data = [
    {
        "symbol": "BTCUSDT",            // 交易对
        "markPrice": "11793.63104562",  // 标记价格
        "indexPrice": "11781.80495970", // 指数价格
        "estimatedSettlePrice": "11781.16138815",  // 预估结算价,仅在交割开始前最后一小时有意义
        "lastFundingRate": "0.00038246",    // 最近更新的预估资金费率
        "nextFundingTime": 1597392000000,   // 下次资金费时间
        "interestRate": "0.00010000",       // 标的资产基础利率
        "time": 1597370495002               // 更新时间
    }
]