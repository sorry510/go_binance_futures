# Inputs

No raw market data is copied into this bundle.

The replay reads the existing local research database `go_bn_test`:
- `market_funding_rates`: symbol, funding_time, funding_rate.
- `market_klines_1h`: symbol, open_time, open_price, close_price.
- market = `futures_usdt`.

The event universe is generated mechanically from all available funding observations for the ten fixed old symbols over the configured date range. No events are manually selected or removed.
