const data = {
    "timezone": "UTC",
    "serverTime": 1711627208783,
    "rateLimits": [
        {
            "rateLimitType": "REQUEST_WEIGHT",
            "interval": "MINUTE",
            "intervalNum": 1,
            "limit": 2400
        },
        {
            "rateLimitType": "ORDERS",
            "interval": "MINUTE",
            "intervalNum": 1,
            "limit": 1200
        },
        {
            "rateLimitType": "ORDERS",
            "interval": "SECOND",
            "intervalNum": 10,
            "limit": 300
        }
    ],
    "exchangeFilters": [],
    "symbols": [
        {
            "symbol": "BTCUSDT",
            "pair": "BTCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "PoW"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "4529764",
                    "minPrice": "556.80",
                    "tickSize": "0.10"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "120",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "100"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BTC",
            "liquidationFee": "0.012500",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ETHUSDT",
            "pair": "ETHUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "306177",
                    "minPrice": "39.86",
                    "tickSize": "0.01"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "20"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ETH",
            "liquidationFee": "0.012500",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "BCHUSDT",
            "pair": "BCHUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "PoW"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "13.93",
                    "tickSize": "0.01"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "850",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "20"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BCH",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "XRPUSDT",
            "pair": "XRPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Payment"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0143",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "XRP",
            "liquidationFee": "0.012500",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "EOSUSDT",
            "pair": "EOSUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.111",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "120000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "EOS",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "LTCUSDT",
            "pair": "LTCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "PoW"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "3.61",
                    "tickSize": "0.01"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "20"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LTC",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "TRXUSDT",
            "pair": "TRXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00132",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "TRX",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ETCUSDT",
            "pair": "ETCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "1.051",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "20"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ETC",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "LINKUSDT",
            "pair": "LINKUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Oracle"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.464",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "200000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "20"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LINK",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "XLMUSDT",
            "pair": "XLMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00648",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "XLM",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ADAUSDT",
            "pair": "ADAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.01740",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "2000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ADA",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "XMRUSDT",
            "pair": "XMRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Privacy"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "4.36",
                    "tickSize": "0.01"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "XMR",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "DASHUSDT",
            "pair": "DASHUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Payment"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "3.82",
                    "tickSize": "0.01"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "3000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DASH",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ZECUSDT",
            "pair": "ZECUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Privacy"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "2.85",
                    "tickSize": "0.01"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ZEC",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "XTZUSDT",
            "pair": "XTZUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.064",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "XTZ",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BNBUSDT",
            "pair": "BNBUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "6.600",
                    "tickSize": "0.010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BNB",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ATOMUSDT",
            "pair": "ATOMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.251",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ATOM",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ONTUSDT",
            "pair": "ONTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0241",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "250000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ONT",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "IOTAUSDT",
            "pair": "IOTAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0205",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "IOTA",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BATUSDT",
            "pair": "BATUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0134",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "600000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BAT",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "VETUSDT",
            "pair": "VETUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.002080",
                    "tickSize": "0.000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "VET",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "NEOUSDT",
            "pair": "NEOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "1.093",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "NEO",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "QTUMUSDT",
            "pair": "QTUMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.246",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "QTUM",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "IOSTUSDT",
            "pair": "IOSTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000587",
                    "tickSize": "0.000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "80000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "IOST",
            "liquidationFee": "0.012500",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "THETAUSDT",
            "pair": "THETAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.1070",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "THETA",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ALGOUSDT",
            "pair": "ALGOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.0141",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ALGO",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ZILUSDT",
            "pair": "ZILUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.00219",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ZIL",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "KNCUSDT",
            "pair": "KNCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "181",
                    "minPrice": "0.03200",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "KNC",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ZRXUSDT",
            "pair": "ZRXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.0179",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "250000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ZRX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "COMPUSDT",
            "pair": "COMPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "8",
                    "tickSize": "0.01"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "COMP",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "OMGUSDT",
            "pair": "OMGUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.1060",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "OMG",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "DOGEUSDT",
            "pair": "DOGEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Meme"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "30",
                    "minPrice": "0.002440",
                    "tickSize": "0.000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DOGE",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "SXPUSDT",
            "pair": "SXPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Payment"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.0454",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "80000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SXP",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "KAVAUSDT",
            "pair": "KAVAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "653",
                    "minPrice": "0.0593",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "KAVA",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BANDUSDT",
            "pair": "BANDUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Oracle"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "783",
                    "minPrice": "0.1647",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "2000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BAND",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "RLCUSDT",
            "pair": "RLCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Oracle"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "432",
                    "minPrice": "0.1029",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "80000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "RLC",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "WAVESUSDT",
            "pair": "WAVESUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2249",
                    "minPrice": "0.3420",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "16000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "WAVES",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "MKRUSDT",
            "pair": "MKRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "351788",
                    "minPrice": "50",
                    "tickSize": "0.10"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MKR",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "SNXUSDT",
            "pair": "SNXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1144",
                    "minPrice": "0.164",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SNX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "DOTUSDT",
            "pair": "DOTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2398",
                    "minPrice": "0.373",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DOT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "DEFIUSDT",
            "pair": "DEFIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 1,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "INDEX",
            "underlyingSubType": [
                "Index"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "230680",
                    "minPrice": "33.8",
                    "tickSize": "0.1"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DEFI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "YFIUSDT",
            "pair": "YFIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 1,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "3836373",
                    "minPrice": "667",
                    "tickSize": "1"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "500",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "YFI",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BALUSDT",
            "pair": "BALUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "5000",
                    "minPrice": "0.630",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BAL",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "CRVUSDT",
            "pair": "CRVUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.031",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CRV",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "TRBUSDT",
            "pair": "TRBUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Oracle"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.3000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "5138",
                    "minPrice": "1.230",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.7000",
                    "multiplierUp": "1.3000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "TRB",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.30"
        },
        {
            "symbol": "RUNEUSDT",
            "pair": "RUNEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "839",
                    "minPrice": "0.1720",
                    "tickSize": "0.0010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "RUNE",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "SUSHIUSDT",
            "pair": "SUSHIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1301",
                    "minPrice": "0.1430",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "25000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SUSHI",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "SRMUSDT",
            "pair": "SRMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1668486600000,
            "onboardDate": 1569398400000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "722",
                    "minPrice": "0.0920",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SRM",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "EGLDUSDT",
            "pair": "EGLDUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "14295",
                    "minPrice": "1.770",
                    "tickSize": "0.010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "EGLD",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "SOLUSDT",
            "pair": "SOLUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "6857",
                    "minPrice": "0.4200",
                    "tickSize": "0.0010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SOL",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ICXUSDT",
            "pair": "ICXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "400",
                    "minPrice": "0.0234",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ICX",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "STORJUSDT",
            "pair": "STORJUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Storage"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "121",
                    "minPrice": "0.0190",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "STORJ",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BLZUSDT",
            "pair": "BLZUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Storage"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "22",
                    "minPrice": "0.00360",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "350000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BLZ",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "UNIUSDT",
            "pair": "UNIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2684",
                    "minPrice": "0.3730",
                    "tickSize": "0.0010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "25000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "UNI",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "AVAXUSDT",
            "pair": "AVAXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2403",
                    "minPrice": "0.3500",
                    "tickSize": "0.0010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AVAX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "FTMUSDT",
            "pair": "FTMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "41",
                    "minPrice": "0.007700",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "250000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "FTM",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "HNTUSDT",
            "pair": "HNTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1679302800000,
            "onboardDate": 1569398400000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1929",
                    "minPrice": "0.1660",
                    "tickSize": "0.0010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "HNT",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ENJUSDT",
            "pair": "ENJUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "153",
                    "minPrice": "0.02250",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ENJ",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "FLMUSDT",
            "pair": "FLMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.0096",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "FLM",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "TOMOUSDT",
            "pair": "TOMOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1699952400000,
            "onboardDate": 1569398400000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.0250",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "TOMO",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "RENUSDT",
            "pair": "RENUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.00880",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "REN",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "KSMUSDT",
            "pair": "KSMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "28450",
                    "minPrice": "4.060",
                    "tickSize": "0.010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "KSM",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "NEARUSDT",
            "pair": "NEARUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.0480",
                    "tickSize": "0.0010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "24000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "NEAR",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "AAVEUSDT",
            "pair": "AAVEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "38702",
                    "minPrice": "4.340",
                    "tickSize": "0.010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1250",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AAVE",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "FILUSDT",
            "pair": "FILUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Storage"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "10000",
                    "minPrice": "1.381",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "7500",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "FIL",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "RSRUSDT",
            "pair": "RSRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "10",
                    "minPrice": "0.000778",
                    "tickSize": "0.000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "RSR",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "LRCUSDT",
            "pair": "LRCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.00520",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LRC",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "MATICUSDT",
            "pair": "MATICUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "137",
                    "minPrice": "0.00960",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MATIC",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "OCEANUSDT",
            "pair": "OCEANUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "AI"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.00010",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "600000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "OCEAN",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "CVCUSDT",
            "pair": "CVCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1669712400000,
            "onboardDate": 1569398400000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.00517",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "800000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CVC",
            "liquidationFee": "0.025000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BELUSDT",
            "pair": "BELUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "204",
                    "minPrice": "0.03610",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "150000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BEL",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "CTKUSDT",
            "pair": "CTKUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "180",
                    "minPrice": "0.02600",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "88000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CTK",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "AXSUSDT",
            "pair": "AXSUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "6989",
                    "minPrice": "0.08000",
                    "tickSize": "0.00100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "8000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AXS",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ALPHAUSDT",
            "pair": "ALPHAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.01760",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "540000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ALPHA",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ZENUSDT",
            "pair": "ZENUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1569398400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Privacy"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "10000",
                    "minPrice": "1.437",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ZEN",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "SKLUSDT",
            "pair": "SKLUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1598252400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.00544",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SKL",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "GRTUSDT",
            "pair": "GRTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1608274800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Oracle"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.01398",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "150000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "GRT",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "1INCHUSDT",
            "pair": "1INCHUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1608879600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0613",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "1INCH",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "CHZUSDT",
            "pair": "CHZUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1611212400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.00447",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "850000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CHZ",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "SANDUSDT",
            "pair": "SANDUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1610953200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00500",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SAND",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ANKRUSDT",
            "pair": "ANKRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1611558000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001510",
                    "tickSize": "0.000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "60000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ANKR",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BTSUSDT",
            "pair": "BTSUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1660813200000,
            "onboardDate": 1612076400000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00114",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "3000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BTS",
            "liquidationFee": "0.025000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "LITUSDT",
            "pair": "LITUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1613545200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Privacy"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.077",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "150000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LIT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "UNFIUSDT",
            "pair": "UNFIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1612854000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "10000",
                    "minPrice": "0.222",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "8000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "UNFI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "REEFUSDT",
            "pair": "REEFUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1611558000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000516",
                    "tickSize": "0.000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "200000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "REEF",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "RVNUSDT",
            "pair": "RVNUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1614063600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "PoW"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00154",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "3000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "RVN",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "SFPUSDT",
            "pair": "SFPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1614150000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0223",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SFP",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "XEMUSDT",
            "pair": "XEMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1614668400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0035",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "XEM",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BTCSTUSDT",
            "pair": "BTCSTUSDT",
            "contractType": "",
            "deliveryDate": 4133404800000,
            "onboardDate": 1614754800000,
            "status": "PENDING_TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEFI"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.668",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "8500",
                    "multiplierUp": "11500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BTCST",
            "liquidationFee": "0.040000",
            "marketTakeBound": "0.30"
        },
        {
            "symbol": "COTIUSDT",
            "pair": "COTIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1615273200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Payment"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00328",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "800000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "COTI",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "CHRUSDT",
            "pair": "CHRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1615532400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0031",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "250000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CHR",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "MANAUSDT",
            "pair": "MANAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1615705200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0136",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MANA",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ALICEUSDT",
            "pair": "ALICEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1615791600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.120",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ALICE",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "HBARUSDT",
            "pair": "HBARUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1615964400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00279",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "HBAR",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ONEUSDT",
            "pair": "ONEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1615964400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00124",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ONE",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "LINAUSDT",
            "pair": "LINAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1615964400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00087",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LINA",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "STMXUSDT",
            "pair": "STMXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1615964400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Payment"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00048",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "12000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "STMX",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "DENTUSDT",
            "pair": "DENTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1616482800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000070",
                    "tickSize": "0.000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "500000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DENT",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "CELRUSDT",
            "pair": "CELRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1617001200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00050",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "3000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CELR",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "HOTUSDT",
            "pair": "HOTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1617087600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Storage"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000129",
                    "tickSize": "0.000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "HOT",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "MTLUSDT",
            "pair": "MTLUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1617174000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0390",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MTL",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "OGNUSDT",
            "pair": "OGNUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1617174000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0137",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "OGN",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "NKNUSDT",
            "pair": "NKNUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1617865200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00602",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "710000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "NKN",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "SCUSDT",
            "pair": "SCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1655456400000,
            "onboardDate": 1618210800000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Storage"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000368",
                    "tickSize": "0.000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SC",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "DGBUSDT",
            "pair": "DGBUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1618815600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Payment"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00131",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "7200000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DGB",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "1000SHIBUSDT",
            "pair": "1000SHIBUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1620630000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Meme"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000157",
                    "tickSize": "0.000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "800000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "150000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "1000SHIB",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BAKEUSDT",
            "pair": "BAKEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1621321200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0001",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BAKE",
            "liquidationFee": "0.012500",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "GTCUSDT",
            "pair": "GTCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1615791600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.200",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "150000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "GTC",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BTCDOMUSDT",
            "pair": "BTCDOMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1623913200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 1,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "INDEX",
            "underlyingSubType": [
                "Index"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "10",
                    "tickSize": "0.1"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BTCDOM",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "IOTXUSDT",
            "pair": "IOTXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1628665200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00050",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "IOTX",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "AUDIOUSDT",
            "pair": "AUDIOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1629270000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0040",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "420000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AUDIO",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "RAYUSDT",
            "pair": "RAYUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1668484800000,
            "onboardDate": 1629270000000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.010",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "RAY",
            "liquidationFee": "0.025000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "C98USDT",
            "pair": "C98USDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1629702000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0010",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "2500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "C98",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "MASKUSDT",
            "pair": "MASKUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1626418800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Privacy"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0040",
                    "tickSize": "0.0010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "40000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MASK",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ATAUSDT",
            "pair": "ATAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1630306800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Privacy"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0040",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ATA",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "DYDXUSDT",
            "pair": "DYDXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1631170800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.010",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "220000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DYDX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "1000XECUSDT",
            "pair": "1000XECUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1631775600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Payment"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00050",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "1000XEC",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "GALAUSDT",
            "pair": "GALAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1631862000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00050",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "GALA",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "CELOUSDT",
            "pair": "CELOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1632639600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.010",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CELO",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ARUSDT",
            "pair": "ARUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1632812400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Storage"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.010",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AR",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "KLAYUSDT",
            "pair": "KLAYUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1633935600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0010",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "700000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "KLAY",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ARPAUSDT",
            "pair": "ARPAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1634540400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00050",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ARPA",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "CTSIUSDT",
            "pair": "CTSIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1635145200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0010",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "700000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CTSI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "LPTUSDT",
            "pair": "LPTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1636527600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.010",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LPT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ENSUSDT",
            "pair": "ENSUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1638169200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.010",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ENS",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "PEOPLEUSDT",
            "pair": "PEOPLEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1640242800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DAO"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.00010",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1161000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "PEOPLE",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "ANTUSDT",
            "pair": "ANTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1640329200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DAO"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.010",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "15000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ANT",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ROSEUSDT",
            "pair": "ROSEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1640934000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "10000",
                    "minPrice": "0.00010",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ROSE",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "DUSKUSDT",
            "pair": "DUSKUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1641452400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Privacy"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "10000",
                    "minPrice": "0.00010",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DUSK",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "FLOWUSDT",
            "pair": "FLOWUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1644390000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.010",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "150000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "FLOW",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "IMXUSDT",
            "pair": "IMXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1644476400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0010",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "40000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "IMX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "API3USDT",
            "pair": "API3USDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1645426800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.2000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "10000",
                    "minPrice": "0.0010",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8000",
                    "multiplierUp": "1.2000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "API3",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.20"
        },
        {
            "symbol": "GMTUSDT",
            "pair": "GMTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1647241200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "10000",
                    "minPrice": "0.00010",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "150000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "GMT",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "APEUSDT",
            "pair": "APEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1647500400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0010",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "APE",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "WOOUSDT",
            "pair": "WOOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1649314800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "10000",
                    "minPrice": "0.00010",
                    "tickSize": "0.00001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "WOO",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "FTTUSDT",
            "pair": "FTTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1668398400000,
            "onboardDate": 1649919600000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "CEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0010",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "FTT",
            "liquidationFee": "0.025000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "JASMYUSDT",
            "pair": "JASMYUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1650351600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.000010",
                    "tickSize": "0.000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "JASMY",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "DARUSDT",
            "pair": "DARUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1651129200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0010",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "620000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DAR",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "GALUSDT",
            "pair": "GALUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1651734000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "10000",
                    "minPrice": "0.00010",
                    "tickSize": "0.00010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "15040",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "GAL",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "OPUSDT",
            "pair": "OPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1654066800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "OP",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "INJUSDT",
            "pair": "INJUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1660633200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "25000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "INJ",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "STGUSDT",
            "pair": "STGUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1661324400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "190000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "STG",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "FOOTBALLUSDT",
            "pair": "FOOTBALLUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1711443600000,
            "onboardDate": 1661929200000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "INDEX",
            "underlyingSubType": [
                "Index"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.01000",
                    "tickSize": "0.01000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "FOOTBALL",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "SPELLUSDT",
            "pair": "SPELLUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1662361200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "500000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "33840000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SPELL",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "1000LUNCUSDT",
            "pair": "1000LUNCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1662620400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "1000LUNC",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "LUNA2USDT",
            "pair": "LUNA2USDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1662706800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "60000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LUNA2",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "LDOUSDT",
            "pair": "LDOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1663743600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LDO",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "CVXUSDT",
            "pair": "CVXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1663743600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "18000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CVX",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ICPUSDT",
            "pair": "ICPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1627628400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ICP",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "APTUSDT",
            "pair": "APTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1666076400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00100",
                    "tickSize": "0.00100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "APT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "QNTUSDT",
            "pair": "QNTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1666076400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.010000",
                    "tickSize": "0.010000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "QNT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BLUEBIRDUSDT",
            "pair": "BLUEBIRDUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1711443600000,
            "onboardDate": 1667286000000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "INDEX",
            "underlyingSubType": [
                "Index"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.00100",
                    "tickSize": "0.00100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2500",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BLUEBIRD",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "FETUSDT",
            "pair": "FETUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1673766000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "AI"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "250000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "FET",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "FXSUSDT",
            "pair": "FXSUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1674111600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "FXS",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "HOOKUSDT",
            "pair": "HOOKUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1674111600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "95000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "HOOK",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "MAGICUSDT",
            "pair": "MAGICUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1674457200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MAGIC",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "TUSDT",
            "pair": "TUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1675148400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Bitcoin Eco"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "T",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "RNDRUSDT",
            "pair": "RNDRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1675321200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "40000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "RNDR",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "HIGHUSDT",
            "pair": "HIGHUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1675407600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "54000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "HIGH",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "MINAUSDT",
            "pair": "MINAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1675407600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Privacy"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.2000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "62500",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8000",
                    "multiplierUp": "1.2000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MINA",
            "liquidationFee": "0.017500",
            "marketTakeBound": "0.20"
        },
        {
            "symbol": "ASTRUSDT",
            "pair": "ASTRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1676271600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "600000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ASTR",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "AGIXUSDT",
            "pair": "AGIXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1676444400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "AI"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AGIX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "PHBUSDT",
            "pair": "PHBUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1676444400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "23800",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "PHB",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "GMXUSDT",
            "pair": "GMXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1676530800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.010000",
                    "tickSize": "0.010000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "GMX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "CFXUSDT",
            "pair": "CFXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1676790000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "150000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CFX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "STXUSDT",
            "pair": "STXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1676876400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "STX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "COCOSUSDT",
            "pair": "COCOSUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1685005200000,
            "onboardDate": 1676876400000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "COCOS",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "BNXUSDT",
            "pair": "BNXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1677049200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "320000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BNX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ACHUSDT",
            "pair": "ACHUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1676962800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Payment"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "4900000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ACH",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "SSVUSDT",
            "pair": "SSVUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1677135600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.010000",
                    "tickSize": "0.010000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SSV",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "CKBUSDT",
            "pair": "CKBUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1677481200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "PoW"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5580000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CKB",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "PERPUSDT",
            "pair": "PERPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1677999600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "34800",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "PERP",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "TRUUSDT",
            "pair": "TRUUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1678086000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "TRU",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "LQTYUSDT",
            "pair": "LQTYUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1678258800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "67000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LQTY",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "USDCUSDT",
            "pair": "USDCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1678579200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "USDC",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "IDUSDT",
            "pair": "IDUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1679529600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ID",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ARBUSDT",
            "pair": "ARBUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1679554800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ARB",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "JOEUSDT",
            "pair": "JOEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1679961600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "JOE",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "TLMUSDT",
            "pair": "TLMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1680048000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1325000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "TLM",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "AMBUSDT",
            "pair": "AMBUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1680048000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "9300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AMB",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "LEVERUSDT",
            "pair": "LEVERUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1680048000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "47000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LEVER",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "RDNTUSDT",
            "pair": "RDNTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1680566400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "150000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "RDNT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "HFTUSDT",
            "pair": "HFTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1680652800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "HFT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "XVSUSDT",
            "pair": "XVSUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1681257600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "11000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "XVS",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ETHBTC",
            "pair": "ETHBTC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1684972800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Cross Pair"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.000010",
                    "tickSize": "0.000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "0.001"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "BTC",
            "marginAsset": "BTC",
            "baseAsset": "ETH",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "BLURUSDT",
            "pair": "BLURUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1682553600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BLUR",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "EDUUSDT",
            "pair": "EDUUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1682640000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "40000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "EDU",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "IDEXUSDT",
            "pair": "IDEXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1682985600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "800000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "IDEX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "SUIUSDT",
            "pair": "SUIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1683072000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SUI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "1000PEPEUSDT",
            "pair": "1000PEPEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1683244800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Meme"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000001",
                    "tickSize": "0.0000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "800000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "1000PEPE",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "1000FLOKIUSDT",
            "pair": "1000FLOKIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1683331200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Meme"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "638000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "1000FLOKI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "UMAUSDT",
            "pair": "UMAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1683590400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "UMA",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "RADUSDT",
            "pair": "RADUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1683590400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "RAD",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "KEYUSDT",
            "pair": "KEYUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1684800000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "80000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "11000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "KEY",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "COMBOUSDT",
            "pair": "COMBOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1685707200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "82000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "COMBO",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "NMRUSDT",
            "pair": "NMRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1687435200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.010000",
                    "tickSize": "0.010000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "6000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "NMR",
            "liquidationFee": "0.020000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "MAVUSDT",
            "pair": "MAVUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1688040000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "120000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MAV",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "MDTUSDT",
            "pair": "MDTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1688126400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MDT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "XVGUSDT",
            "pair": "XVGUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1688558400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "PoW"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "500000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "15000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "XVG",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "WLDUSDT",
            "pair": "WLDUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1690200000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "AI"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1250000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "WLD",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "PENDLEUSDT",
            "pair": "PENDLEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1690511400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "3000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "PENDLE",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ARKMUSDT",
            "pair": "ARKMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1690512300000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "AI"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ARKM",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "AGLDUSDT",
            "pair": "AGLDUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1690597800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AGLD",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "YGGUSDT",
            "pair": "YGGUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1691206200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "3000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "YGG",
            "liquidationFee": "0.025000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "DODOXUSDT",
            "pair": "DODOXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1691496000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "220000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DODOX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "BNTUSDT",
            "pair": "BNTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1691668800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.2000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "97000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8000",
                    "multiplierUp": "1.2000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BNT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.20"
        },
        {
            "symbol": "OXTUSDT",
            "pair": "OXTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1691755200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "600000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "OXT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "SEIUSDT",
            "pair": "SEIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1692239400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SEI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "CYBERUSDT",
            "pair": "CYBERUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1692619200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CYBER",
            "liquidationFee": "0.025000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "HIFIUSDT",
            "pair": "HIFIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1694874600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "25000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "HIFI",
            "liquidationFee": "0.022500",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ARKUSDT",
            "pair": "ARKUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1695133800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "150000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ARK",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "FRONTUSDT",
            "pair": "FRONTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1695393000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "170000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "FRONT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "GLMRUSDT",
            "pair": "GLMRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1695691800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "GLMR",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "BICOUSDT",
            "pair": "BICOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1695904200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BICO",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "BTCUSDT_240329",
            "pair": "BTCUSDT",
            "contractType": "CURRENT_QUARTER",
            "deliveryDate": 1711699200000,
            "onboardDate": 1695974400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 1,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000000",
                    "minPrice": "576.3",
                    "tickSize": "0.1"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "500",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BTC",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ETHUSDT_240329",
            "pair": "ETHUSDT",
            "contractType": "CURRENT_QUARTER",
            "deliveryDate": 1711699200000,
            "onboardDate": 1695974400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "41.10",
                    "tickSize": "0.01"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ETH",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "STRAXUSDT",
            "pair": "STRAXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 1710493200000,
            "onboardDate": 1696991400000,
            "status": "SETTLING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "90000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "STRAX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "LOOMUSDT",
            "pair": "LOOMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1697034600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "800000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LOOM",
            "liquidationFee": "0.025000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "BIGTIMEUSDT",
            "pair": "BIGTIMEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1697121000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BIGTIME",
            "liquidationFee": "0.022500",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "BONDUSDT",
            "pair": "BONDUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1697380200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "27000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BOND",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ORBSUSDT",
            "pair": "ORBSUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1697522400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ORBS",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "STPTUSDT",
            "pair": "STPTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1697639400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "STPT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "WAXPUSDT",
            "pair": "WAXPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1697640300000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "WAXP",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "BSVUSDT",
            "pair": "BSVUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1697805000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "PoW"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.01000",
                    "tickSize": "0.01000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BSV",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "RIFUSDT",
            "pair": "RIFUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1697891400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Bitcoin Eco"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "540000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "RIF",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "POLYXUSDT",
            "pair": "POLYXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1698201000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "POLYX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "GASUSDT",
            "pair": "GASUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1698201900000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.2000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8000",
                    "multiplierUp": "1.2000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "GAS",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.20"
        },
        {
            "symbol": "POWRUSDT",
            "pair": "POWRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1698373800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "POWR",
            "liquidationFee": "0.012500",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "SLPUSDT",
            "pair": "SLPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1698762600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SLP",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "TIAUSDT",
            "pair": "TIAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1698769800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "25000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "TIA",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "SNTUSDT",
            "pair": "SNTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1698928200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1700000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SNT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "CAKEUSDT",
            "pair": "CAKEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1698929100000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DEX"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "CAKE",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "MEMEUSDT",
            "pair": "MEMEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1699012800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Meme"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MEME",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "TWTUSDT",
            "pair": "TWTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1699021800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "TWT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "TOKENUSDT",
            "pair": "TOKENUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1699023600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "TOKEN",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ORDIUSDT",
            "pair": "ORDIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1699360200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Bitcoin Eco"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ORDI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "STEEMUSDT",
            "pair": "STEEMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1699453800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.000100",
                    "tickSize": "0.000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "2000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "480000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "STEEM",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "BADGERUSDT",
            "pair": "BADGERUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1699533000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Bitcoin Eco"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "21000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BADGER",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ILVUSDT",
            "pair": "ILVUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1699633800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 5,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.01000",
                    "tickSize": "0.01000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ILV",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "NTRNUSDT",
            "pair": "NTRNUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1699963200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "NTRN",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "MBLUSDT",
            "pair": "MBLUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1699974000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "20000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MBL",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "KASUSDT",
            "pair": "KASUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1700186400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "PoW"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "8000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "KAS",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "BEAMXUSDT",
            "pair": "BEAMXUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1700227800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "500000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BEAMX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "1000BONKUSDT",
            "pair": "1000BONKUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1700661600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Meme"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "40000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "6000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "1000BONK",
            "liquidationFee": "0.012500",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "PYTHUSDT",
            "pair": "PYTHUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1700663400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Oracle"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "400000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "70000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "PYTH",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "SUPERUSDT",
            "pair": "SUPERUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1701003600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "NFT"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "250000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "80000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "SUPER",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "USTCUSDT",
            "pair": "USTCUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1701088200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "USTC",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ONGUSDT",
            "pair": "ONGUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1701090000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "130000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ONG",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ETHWUSDT",
            "pair": "ETHWUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1701174600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "PoW"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "23000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ETHW",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "JTOUSDT",
            "pair": "JTOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1702029600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "JTO",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "1000SATSUSDT",
            "pair": "1000SATSUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1702389600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Meme"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "60000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "1000SATS",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "AUCTIONUSDT",
            "pair": "AUCTIONUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1702645200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.2000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.010000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8000",
                    "multiplierUp": "1.2000"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AUCTION",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.20"
        },
        {
            "symbol": "1000RATSUSDT",
            "pair": "1000RATSUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1702648800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Meme"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "75000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "1000RATS",
            "liquidationFee": "0.007500",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ACEUSDT",
            "pair": "ACEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1702893600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2200",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ACE",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "MOVRUSDT",
            "pair": "MOVRUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1703593800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "5000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MOVR",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "NFPUSDT",
            "pair": "NFPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1703680200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "AI"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "NFP",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.30"
        },
        {
            "symbol": "BTCUSDT_240628",
            "pair": "BTCUSDT",
            "contractType": "NEXT_QUARTER",
            "deliveryDate": 1719561600000,
            "onboardDate": 1703836800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 1,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000000",
                    "minPrice": "576.3",
                    "tickSize": "0.1"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "500",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BTC",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ETHUSDT_240628",
            "pair": "ETHUSDT",
            "contractType": "NEXT_QUARTER",
            "deliveryDate": 1719561600000,
            "onboardDate": 1703836800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "41.10",
                    "tickSize": "0.01"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ETH",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "BTCUSDC",
            "pair": "BTCUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1704285000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 1,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "500000",
                    "minPrice": "0.1",
                    "tickSize": "0.1"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "800",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "25",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "100"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "BTC",
            "liquidationFee": "0.012500",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "ETHUSDC",
            "pair": "ETHUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1704285300000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 2,
            "quantityPrecision": 3,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.01",
                    "tickSize": "0.01"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "8000",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "400",
                    "minQty": "0.001",
                    "stepSize": "0.001"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "20"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "ETH",
            "liquidationFee": "0.012500",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "BNBUSDC",
            "pair": "BNBUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1704285600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.010",
                    "tickSize": "0.010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "80000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "400",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "BNB",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "SOLUSDC",
            "pair": "SOLUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1704285900000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.1000",
                    "tickSize": "0.0010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "800000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "SOL",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "XRPUSDC",
            "pair": "XRPUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1704286200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.0500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "100000",
                    "minPrice": "0.0001",
                    "tickSize": "0.0001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "8000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "400000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9500",
                    "multiplierUp": "1.0500"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "XRP",
            "liquidationFee": "0.012500",
            "marketTakeBound": "0.05"
        },
        {
            "symbol": "AIUSDT",
            "pair": "AIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1704717000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.000010",
                    "tickSize": "0.000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "XAIUSDT",
            "pair": "XAIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1704808800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "XAI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "DOGEUSDC",
            "pair": "DOGEUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1705572000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.000010",
                    "tickSize": "0.000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "50000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "25000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "DOGE",
            "liquidationFee": "0.010000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "WIFUSDT",
            "pair": "WIFUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1705587300000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Meme"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "WIF",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "MANTAUSDT",
            "pair": "MANTAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1705586400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "1000",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MANTA",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ONDOUSDT",
            "pair": "ONDOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1705755600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "5000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "250000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ONDO",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "LSKUSDT",
            "pair": "LSKUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1706192100000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "20000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "LSK",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ALTUSDT",
            "pair": "ALTUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1706191200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ALT",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "JUPUSDT",
            "pair": "JUPUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1706725800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Dex"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "50000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "JUP",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ZETAUSDT",
            "pair": "ZETAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1706862600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.000100",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "80000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ZETA",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "RONINUSDT",
            "pair": "RONINUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1707193800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-1"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.001000",
                    "tickSize": "0.000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "RONIN",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "DYMUSDT",
            "pair": "DYMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1707289200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "5000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "DYM",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "SUIUSDC",
            "pair": "SUIUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1707375600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "20000",
                    "minPrice": "0.000100",
                    "tickSize": "0.000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "2000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "SUI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "OMUSDT",
            "pair": "OMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1707832800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "390000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "OM",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "LINKUSDC",
            "pair": "LINKUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1708072200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 3,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001",
                    "tickSize": "0.001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "LINK",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "PIXELUSDT",
            "pair": "PIXELUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1708354800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "PIXEL",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "STRKUSDT",
            "pair": "STRKUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1708448400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "STRK",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "MAVIAUSDT",
            "pair": "MAVIAUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1708509600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "13000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MAVIA",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ORDIUSDC",
            "pair": "ORDIUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1708585200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "200",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "ORDI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "GLMUSDT",
            "pair": "GLMUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1708596000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "GLM",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "PORTALUSDT",
            "pair": "PORTALUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1709218800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Metaverse"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "PORTAL",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "TONUSDT",
            "pair": "TONUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1709296200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "39000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "TON",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "AXLUSDT",
            "pair": "AXLUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1709307000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Infrastructure"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "57000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AXL",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "MYROUSDT",
            "pair": "MYROUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1709627400000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "MEME"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "MYRO",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "1000PEPEUSDC",
            "pair": "1000PEPEUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1709800200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000001",
                    "tickSize": "0.0000001"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "800000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "1000PEPE",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "METISUSDT",
            "pair": "METISUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1710232200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 4,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "Layer-2"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.0100",
                    "tickSize": "0.0100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "METIS",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "AEVOUSDT",
            "pair": "AEVOUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1710336600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.0100000",
                    "tickSize": "0.0010000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "AEVO",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "WLDUSDC",
            "pair": "WLDUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1710399600000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0001000",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "WLD",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "VANRYUSDT",
            "pair": "VANRYUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1710351000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000100",
                    "tickSize": "0.0000100"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "10000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "300000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "VANRY",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "BOMEUSDT",
            "pair": "BOMEUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1710592200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "200",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "200000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "BOME",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "ETHFIUSDT",
            "pair": "ETHFIUSDT",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1710775800000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 1,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "DeFi"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1500",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.0010000",
                    "tickSize": "0.0010000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "1000000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "30000",
                    "minQty": "0.1",
                    "stepSize": "0.1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.8500",
                    "multiplierUp": "1.1500"
                }
            ],
            "quoteAsset": "USDT",
            "marginAsset": "USDT",
            "baseAsset": "ETHFI",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.15"
        },
        {
            "symbol": "AVAXUSDC",
            "pair": "AVAXUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1710918000000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 6,
            "quantityPrecision": 2,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [
                "USDC"
            ],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.001000",
                    "tickSize": "0.001000"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "100000",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10",
                    "minQty": "0.01",
                    "stepSize": "0.01"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "AVAX",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        },
        {
            "symbol": "1000SHIBUSDC",
            "pair": "1000SHIBUSDC",
            "contractType": "PERPETUAL",
            "deliveryDate": 4133404800000,
            "onboardDate": 1711609200000,
            "status": "TRADING",
            "maintMarginPercent": "2.5000",
            "requiredMarginPercent": "5.0000",
            "pricePrecision": 7,
            "quantityPrecision": 0,
            "baseAssetPrecision": 8,
            "quotePrecision": 8,
            "underlyingType": "COIN",
            "underlyingSubType": [],
            "settlePlan": 0,
            "triggerProtect": "0.1000",
            "orderType": null,
            "timeInForce": [
                "GTC",
                "IOC",
                "FOK",
                "GTX",
                "GTD"
            ],
            "filters": [
                {
                    "filterType": "PRICE_FILTER",
                    "maxPrice": "2000",
                    "minPrice": "0.0000010",
                    "tickSize": "0.0000010"
                },
                {
                    "filterType": "LOT_SIZE",
                    "maxQty": "800000000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MARKET_LOT_SIZE",
                    "maxQty": "10000",
                    "minQty": "1",
                    "stepSize": "1"
                },
                {
                    "filterType": "MAX_NUM_ORDERS",
                    "limit": 200
                },
                {
                    "filterType": "MAX_NUM_ALGO_ORDERS",
                    "limit": 10
                },
                {
                    "filterType": "MIN_NOTIONAL",
                    "notional": "5"
                },
                {
                    "filterType": "PERCENT_PRICE",
                    "multiplierDecimal": "4",
                    "multiplierDown": "0.9000",
                    "multiplierUp": "1.1000"
                }
            ],
            "quoteAsset": "USDC",
            "marginAsset": "USDC",
            "baseAsset": "1000SHIB",
            "liquidationFee": "0.015000",
            "marketTakeBound": "0.10"
        }
    ]
}