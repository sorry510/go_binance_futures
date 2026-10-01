# Protocol

Before reading returns, freeze:

- Discovery/early gate symbols: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT.
- Window: 2023-01-01 through 2026-09-01.
- LONG: previous completed day closes above the day before's high and is bullish; current day trades above previous day's high; 4h EMA20>EMA50, EMA20 rising, ADX14>=20, +DI>-DI.
- SHORT: exact mirror.
- No MarketCondition, Benchmark, or NowTime modulo.
- Execution: 4x leverage, TP8, SL6, fee 0.0005/side, slippage 5bps/side, single position.
- Early gate: overall standardized PF >= ~1.15, at least 3/4 symbols positive, frequency >=0.3 trades/symbol/week, and no obvious full-year collapse.
- If the early gate fails, freeze the family. Do not tune daily acceptance, EMA/ADX, direction, or add filters.
- If it passes, freeze parameters and expand before any holdout inspection.

## Audit correction

The prior run used conditional CLOSE logic and is non-canonical for the project fixed-exit research constraint. It is preserved under `legacy/conditional-exit/`. Entry logic is unchanged. Canonical rerun uses `ROI >= 8 || ROI <= -6` for both CLOSE sides.

## Strict TP8/SL6 audit correction

The original run used a conditional CLOSE expression. In this Engine, RunConfig TP/SL values gate close-rule evaluation but do not force closure. The original result is therefore non-canonical and is preserved under legacy/conditional-exit/. The canonical rerun keeps entry logic unchanged and uses ROI >= 8 || ROI <= -6 for both close rules.
