# Binance USD-M Liquidation History — Feasibility

Hypothesis: USD-M liquidation cascades may contain forced-flow continuation or exhaustion information.

The official Binance Vision `data/futures/um/daily/` directory was enumerated. It exposes aggTrades, bookDepth, bookTicker, indexPriceKlines, klines, markPriceKlines, metrics, premiumIndexKlines and trades, but no USD-M `liquidationSnapshot` / forced-order archive. The analogous COIN-M liquidationSnapshot family has already been researched separately.

Decision: historical-data blocked before returns. Do not use the recent REST forced-order window as a substitute for 2023-2024 discovery history.