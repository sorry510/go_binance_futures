# v146 20h Quote-Volume-Weighted Close Reclaim — 2026-10-01

Hypothesis: a rolling value reference weighted by each hour's quote notional may react differently from an unweighted moving average. The reference is explicitly a quote-volume-weighted close, not true VWAP: QVWC20 = sum(Close * QuoteAssetVolume) / sum(QuoteAssetVolume) over the prior 20 completed 1h bars. A completed 1h close must newly cross QVWC20; current price must then break the trigger bar's extreme.

Project-native only. DSL fields are `kline_1h.Close` and `kline_1h.Amount`, where Amount is QuoteAssetVolume. No EMA/ADX/funding/taker/time filter is used. Exact exits are `ROI >= 8 || ROI <= -6`.

Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only. OOS1/OOS2 stay unread unless the preceding frozen gate passes.

## Final result

Strict Engine discovery:
- **6,202 trades**, LONG 3,230 / SHORT 2,972.
- normalized PF **0.822533**.
- normalized net **-11.478695**.
- **0/6 symbols positive**.
- frequency **9.898313 trades/symbol/week**.
- 2023: 2,618 trades, PF **0.813249**.
- 2024: 3,584 trades, PF **0.829465**.
- exits: 3,617 stop-loss / 2,583 take-profit / 2 end-of-data.

Per-symbol PF: SOL 0.8162, DOGE 0.8606, LTC 0.7953, AVAX 0.8688, UNI 0.8001, ZEC 0.7947.

Decision: **freeze v146**. The signal is high frequency but stably negative across every symbol and both discovery years. Do not scan QVWC period, substitute true VWAP, add trend/volume filters, delete one direction, or reverse the rule. 2025/2026 remain unread; no DB write.
