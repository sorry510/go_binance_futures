# Protocol

1. Freeze strategy JSON before reading returns.
2. Define term ratio = ATR14(1h)*sqrt(24)/ATR14(1d).
3. Trigger only on a completed-hour cross from <=1 to >1.
4. Direction is the completed trigger-hour candle direction; current price must break the trigger-hour extreme.
5. Use fixed project execution: 4x, TP8, SL6, fee/slippage, single position.
6. Early gate BTC/ETH/BNB/XRP, 2023-01-01..2026-09-01.
7. On failure: no ATR period change, no 0.8/1.2 threshold scan, no trend/volume/funding filters, no sign reversal.
8. On pass: freeze parameters and expand symbols before holdout.

## Audit correction

The initial run used a conditional CLOSE expression. In this Engine, RunConfig TP/SL values gate close-rule evaluation but do not force closure. Therefore that run did not implement strict fixed TP8/SL6 and is non-canonical. Its files are preserved under `legacy/conditional-exit/`. The canonical rerun keeps entry logic unchanged and uses `ROI >= 8 || ROI <= -6` for both close rules.
