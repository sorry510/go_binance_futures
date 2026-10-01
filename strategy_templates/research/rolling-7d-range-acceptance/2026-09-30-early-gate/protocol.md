# Protocol

1. Freeze strategy JSON and gate before reading returns.
2. Use only existing project 1m replay plus aggregated 1h/1d bars. No DB backfill.
3. Boundary uses the prior seven completed daily bars: max High[1:8] / min Low[1:8].
4. Acceptance requires a completed 1h candle to cross from inside to a close outside the boundary.
5. Entry requires current price to continue beyond the signal-hour high/low.
6. Fixed engine execution: leverage4, TP8, SL6, fee0.0005/side, slippage5bps/side, single position.
7. Early gate: BTC/ETH/BNB/XRP, 2023-01-01 to 2026-09-01.
8. If gate fails, do not tune 7d, confirmation horizon, add trend/volume/funding filters, or reverse direction.
9. If gate passes, freeze all parameters and run an expansion set before any holdout.

## Audit correction

The initial run used a conditional CLOSE expression. In this Engine, RunConfig TP/SL values gate close-rule evaluation but do not force closure. Therefore that run did not implement strict fixed TP8/SL6 and is non-canonical. Its files are preserved under `legacy/conditional-exit/`. The canonical rerun keeps entry logic unchanged and uses `ROI >= 8 || ROI <= -6` for both close rules.
