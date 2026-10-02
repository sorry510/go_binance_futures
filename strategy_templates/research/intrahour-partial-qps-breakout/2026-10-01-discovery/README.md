# v132 Intrahour Partial-QPS Previous-Hour Breakout — 2026-10-01 Discovery

Hypothesis: a move that breaks the previous completed hour while the current, still-forming hour has already accumulated at least a typical full hour's quote-volume rate is a high-intensity displacement event. This may produce faster path expansion than completed-bar momentum signals.

Mechanism:
- Compare current partial 1h QPS with the mean QPS of the previous 8 completed 1h bars.
- Require current partial QPS >= that baseline.
- LONG only on a fresh 1m close cross above previous completed 1h high.
- SHORT mirrors below previous completed 1h low.
- No trend, taker, funding, ATR, RSI, ADX, or symbol-specific filter.

EMA(2) entries for 1m/1h in technology config are data-loading scaffolding only; entry/exit rules do not reference them.
Exact exits: `ROI >= 8 || ROI <= -6`.
Discovery only: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT, 2023-01-01 through 2025-01-01.

## Final result

Discovery failed decisively: 7,188 trades, normalized PF 0.832938, 0/6 symbols positive, 11.471956 trades/symbol/week. 2023 PF 0.834772 and 2024 PF 0.831508. LONG PF 0.786282; SHORT PF 0.879678. Every symbol is below PF 0.865.

The first preflight run stopped before any return was computed because kline_1m was not present in the strategy VM. Adding unused EMA(2) scaffolding for the 1m series fixed only data exposure; signal logic and preregistered gate were unchanged. The failed preflight is preserved as results/preflight_compile.log.

Interpretation: current-hour volume completion plus previous-hour extreme crossing creates very high frequency but stable negative expectancy under the fixed TP8/SL6 cost structure. Freeze the family. Do not reverse it, scan QPS multipliers, alter the 8h baseline, or add taker/trend filters. 2025 OOS1 and 2026 OOS2 were not evaluated.
