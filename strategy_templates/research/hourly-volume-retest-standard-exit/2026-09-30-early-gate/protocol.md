# Protocol

Frozen before returns:
- Entry logic copied unchanged from `hourly-volume-retest-strength-v4.json`.
- BTC/ETH/BNB/XRP, 2023-01-01 through 2026-09-01.
- 1h impulse uses average of prior six volumes, ATR14 body bounds, OBV direction, ADX/DMI; controlled retest and reclaim; 4h EMA34 trend.
- CLOSE_LONG/CLOSE_SHORT set false; Engine exits fixed at leverage4, TP8, SL6, fee0.0005/side, slippage5bps/side, single position.
- No MarketCondition, Benchmark, NowTime modulo.
- Gate: PF>=1.15, >=3/4 positive, frequency>=0.3/symbol/week, no obvious full-year collapse.
- Failure => freeze; no volume/ATR/ADX/EMA34/retest threshold tuning.
- Pass => parameters stay frozen; expand to untouched six old symbols before any symbol holdout.

## Pre-result exit-semantics correction

Before any return output was read, the old protocol wording `CLOSE=false` was found to be inconsistent with Engine semantics: RunConfig TP/SL values gate close-rule evaluation but do not force closure. The already-created strategy snapshot used `true`, which would close at the gate. For audit clarity the canonical snapshot now writes the equivalent explicit fixed rule `ROI >= 8 || ROI <= -6` for both close sides. Entry logic and all thresholds remain unchanged.
