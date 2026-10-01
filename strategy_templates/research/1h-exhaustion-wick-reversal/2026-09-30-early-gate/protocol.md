# Protocol

Frozen before returns:
- Core-4 BTC/ETH/BNB/XRP, 2023-01-01 to 2026-09-01.
- 1h signal timeframe using existing local history.
- Fresh 20-bar extreme, >=7/9 incoming-direction candles, rejection wick/body >0.66.
- Current 1h breaks signal candle in reversal direction.
- Fixed 4x / TP8 / SL6 / fee0.0005/side / 5bps/side / single position.
- No MarketCondition, Benchmark, NowTime modulo.
- Gate: PF>=1.15, >=3/4 symbols positive, frequency>=0.3/symbol/week, no obvious full-year collapse.
- On failure: no tuning of 20, 7/9, 0.66, timeframe, or filters.

## Audit correction

The prior run used conditional CLOSE logic and is non-canonical for the project fixed-exit research constraint. It is preserved under `legacy/conditional-exit/`. Entry logic is unchanged. Canonical rerun uses `ROI >= 8 || ROI <= -6` for both CLOSE sides.

## Strict TP8/SL6 audit correction

The original run used a conditional CLOSE expression. In this Engine, RunConfig TP/SL values gate close-rule evaluation but do not force closure. The original result is therefore non-canonical and is preserved under legacy/conditional-exit/. The canonical rerun keeps entry logic unchanged and uses ROI >= 8 || ROI <= -6 for both close rules.
