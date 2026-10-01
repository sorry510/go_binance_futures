# Protocol

Frozen before returns:

- Core-4 early gate: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01.
- 3m bars are reconstructed by the current DatasetBuilder/Engine from existing 1m history.
- Exhaustion run: at least 7 of the preceding 9 completed 3m candles share the incoming direction.
- Reversal candle: latest completed 3m makes a new 20-bar extreme, remains the incoming-direction body color, and rejection wick/body >0.66.
- Entry: current 3m breaks the reversal candle opposite the incoming run.
- Fixed execution: 4x, TP8, SL6, fee 0.0005/side, slippage 5bps/side, single position.
- No MarketCondition, Benchmark, or NowTime modulo.
- Gate: standardized PF >=1.15, >=3/4 symbols positive, frequency >=0.3 trades/symbol/week, no obvious full-year collapse.
- Failure => freeze without changing 20 bars, 7/9, 0.66, timeframe, direction, or adding higher-timeframe filters.

## Strict TP8/SL6 audit correction

The original run used a conditional CLOSE expression. In this Engine, RunConfig TP/SL values gate close-rule evaluation but do not force closure. The original result is therefore non-canonical and is preserved under legacy/conditional-exit/. The canonical rerun keeps entry logic unchanged and uses ROI >= 8 || ROI <= -6 for both close rules.
