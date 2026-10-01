# v124 20h Quote-Volume-Weighted Close Reclaim — 2026-10-01

Hypothesis: a rolling value reference weighted by each hour's quote notional may react differently from an unweighted moving average. The reference is explicitly a quote-volume-weighted close, not true VWAP: QVWC20 = sum(Close * QuoteAssetVolume) / sum(QuoteAssetVolume) over the prior 20 completed 1h bars. A completed 1h close must newly cross QVWC20; current price must then break the trigger bar's extreme.

Project-native only. The DSL fields are kline_1h.Close and kline_1h.Amount, where Amount is QuoteAssetVolume. No base volume or external data is required. No EMA/ADX/funding/taker/time filter is added. Exact exits are ROI >= 8 || ROI <= -6.

Time protocol is frozen before returns:
- Discovery: 2023-01-01 through 2025-01-01
- OOS1: 2025-01-01 through 2026-01-01, only if discovery passes
- OOS2: 2026-01-01 through 2026-09-01, only if OOS1 passes

Universe: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT.
Discovery gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no severe 2023/2024 instability.
