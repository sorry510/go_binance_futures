# Protocol

Frozen before returns:
- BTC/ETH/BNB/XRP core-4, 2023-01-01 through 2026-09-01.
- LONG: Close[1]<Close[2]<Close[3] while TakerBuyRatio[1]>[2]>[3], then current 1h breaks High[1].
- SHORT: exact mirror.
- No TakerBuyRatio level threshold, QPS/volume filter, higher-timeframe trend, MarketCondition, Benchmark, or NowTime modulo.
- Fixed 4x / TP8 / SL6 / fee0.0005/side / slippage5bps/side / single position.
- Early gate: PF>=1.15, >=3/4 positive, frequency>=0.3/symbol/week, no obvious full-year collapse.
- Failure => freeze without changing 3h length, adding thresholds, reversing sign, or adding filters.

## Audit correction

The prior run used conditional CLOSE logic and is non-canonical for the project fixed-exit research constraint. It is preserved under `legacy/conditional-exit/`. Entry logic is unchanged. Canonical rerun uses `ROI >= 8 || ROI <= -6` for both CLOSE sides.

## Strict TP8/SL6 audit correction

The original run used a conditional CLOSE expression. In this Engine, RunConfig TP/SL values gate close-rule evaluation but do not force closure. The original result is therefore non-canonical and is preserved under legacy/conditional-exit/. The canonical rerun keeps entry logic unchanged and uses ROI >= 8 || ROI <= -6 for both close rules.
