# v134 Intrahour Taker-Aligned Previous-Hour Breakout — 2026-10-01 Discovery

Hypothesis: previous-hour range breaks are more likely to produce a fast TP8/SL6-compatible displacement when the currently forming hour's accumulated aggressive flow already points in the same direction.

Definition:
- LONG: current partial 1h TakerBuyRatio > 0.5 and the latest completed 1m close freshly crosses above the previous completed 1h high.
- SHORT: current partial 1h TakerBuyRatio < 0.5 and the latest completed 1m close freshly crosses below the previous completed 1h low.
- No completed-hour taker cross, QPS gate, trend filter, funding filter, or external data.

EMA(2) 1m/1h technology entries are only scaffolding to expose the two Kline series to the strategy VM.
Exact exits: `ROI >= 8 || ROI <= -6`.
Discovery only: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT, 2023-01-01 through 2025-01-01.

This differs from v67/v76 because those use completed 1h taker information; repository audit found no existing LONG/SHORT rule using `TakerBuyRatio[0]`.

## Final result

Discovery failed decisively: 13,912 trades, normalized PF 0.821814, 0/6 symbols positive, 22.203374 trades/symbol/week. 2023 PF 0.830443 and 2024 PF 0.815927. LONG PF 0.803503; SHORT PF 0.841424. Every symbol is below PF 0.902.

Interpretation: causal live aggressive-flow alignment does not rescue previous-hour breakouts under the fixed TP8/SL6 cost structure; it creates very high turnover with stable negative expectancy. Freeze the family. Do not scan 0.52/0.48 or 0.55/0.45, add QPS/trend/funding filters, or reverse it. 2025 OOS1 and 2026 OOS2 were not evaluated.
