// https://binance-docs.github.io/apidocs/futures/cn/#31dbeb24c4
// https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/market-data/rest-api/Get-Funding-Rate-History
// 查询资金费率历史
const data = [   
    {
        "symbol": "BTCUSDT",            // 交易对
        "fundingTime": 1698768000000,   // 资金费时间
        "fundingRate": "0.00010000",    // 资金费率
        "markPrice": "34287.54619963"   // 资金费对应标记价格
    },
    {
        "symbol": "BTCUSDT",
        "fundingTime": 1698796800000,
        "fundingRate": "0.00010000",
        "markPrice": "34651.40000000"
    }
]