# Protocol

1. Use the frozen ID121 canonical trade list from the exact-exit audit; do not rerun or retune the strategy.
2. Preserve exact exits ROI >= 8 or ROI <= -6.
3. Measure concentration by symbol, month, quarter, leave-one-block-out, rolling-trade windows, and additive drawdown.
4. Use deterministic block bootstrap with seed 20260930; month and symbol blocks are descriptive robustness checks only.
5. Do not use these diagnostics to change ID121 parameters or remove weak symbols.
6. Decision is candidate retention/demotion only.
