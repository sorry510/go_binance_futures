# v142 Interpretable Two-Split Fast-Move Discovery

Purpose: test whether a very small, interpretable nonlinear interaction among project-native 1h features can isolate fast TP8/SL6-compatible paths after the earlier unified linear model failed.

Frozen protocol:
- SOL/DOGE/LTC/AVAX/UNI/ZEC.
- 2023 train; 2024 validation; 2025/2026 unread.
- One symmetric model receives both LONG and SHORT candidates through side-coded directional features.
- Tree depth <=2; thresholds only 2023 q10/q25/q50/q75/q90; minimum child 2,000 resolved samples.
- Nine fixed DSL-translatable features are documented in protocol.md.
- Entry label uses next 1m open with 5bps adverse entry slippage, then future 1m CLOSE path for <=24h.
- Gross 4x ROI TP +8 => reward +8; SL -6 => reward -6; unresolved labels are excluded.
- Training leaves require mean reward >= +1.0 before they are even eligible for 2024 validation.
- Frozen validation gate: >=1,000 selected resolved samples, mean reward >= +0.5, and >=4/6 positive symbols.
- Exact production exits, if ever promoted, remain `ROI >= 8 || ROI <= -6`.

Pre-outcome protocol correction: the initial skeleton named `trade_ratio_8h`, but the current production KLinePrice/DSL does not expose trade_count. Before any return run, it was replaced by `taker_delta_8h_side`. No outcome from either feature set had been observed.

## Final result

Resolved labels:
- 2023 train: **91,379** of 104,844 candidates; 13,465 unresolved.
- 2024 validation pool: **99,827** of 105,132 candidates; 5,305 unresolved.
- 2025/2026: **not read**.

The frozen depth-2 tree used only directional return features:
- root: `ret12_side <= -0.02973393`
- left child: `ret1_side <= -0.00364252`
- right child: `ret4_side <= -0.01658932`

Four train leaves:
1. n=4,116, mean reward **+0.299320**
2. n=5,022, mean reward **-0.028674**
3. n=5,084, mean reward **+0.030685**
4. n=77,157, mean reward **-0.311598**

The best leaf is only +0.299 reward, far below the preregistered +1.0 training-selection threshold. Therefore **selected leaves = 0**. With no train-qualified rule, there is nothing valid to test as a selected 2024 strategy; the validation gate necessarily fails.

Interpretation: allowing a fixed two-level nonlinear interaction does not uncover a sufficiently strong fast-move subset in the nine project-native features. The tree falls back to recent directional returns, but even its best 2023 region has too little gross TP/SL reward to justify promotion.

Decision: **freeze v142**. Do not increase tree depth, tune quantile grid, lower +1.0/+0.5 reward gates, reduce leaf size, alter the nine features, change the 24h label, or use 2024/2025/2026 to redesign the rule. No strict Engine candidate is generated and no DB write occurs.
