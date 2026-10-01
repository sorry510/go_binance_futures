# v83 Low-Activity Taker Accumulation Breakout — 2026-09-30 Early Gate

Hypothesis: two completed 1h bars with persistent directional taker imbalance but below-normal QPS represent stealth directional accumulation/distribution. A subsequent break of the two-hour range may release a move large enough for fixed TP8/SL6.

Frozen before returns. LONG requires TakerBuyRatio>0.5 for both completed hours; SHORT requires <0.5. Both signal-hour QPS values must be below the mean of the preceding eight completed hours. Entry is confirmed only when current price breaks the two-hour range.

Core-4 early gate: BTC/ETH/BNB/XRP, 2023-01-01..2026-09-01. Fixed execution: leverage4, TP8, SL6, fee0.0005/side, slippage5bps/side, single position. Promotion gate PF>=1.15, >=3/4 symbols positive, frequency>=0.30 trades/symbol/week, no clear multi-year instability.

## Final result

Failed decisively: 3800 trades, normalized PF 0.846211, 0/4 positive, all yearly PFs below 1. Freeze without threshold/window/filter/sign changes.

## Canonical strict-exit rerun

After correcting CLOSE semantics to exact fixed `ROI >= 8 || ROI <= -6`, the canonical result is 5401 trades, normalized PF 0.791651, 0/4 positive, frequency 7.058813/symbol/week. The earlier conditional-exit result is non-canonical and preserved under `legacy/conditional-exit/`. Family remains frozen.

## Canonical strict-exit correction

The original result used a conditional CLOSE expression and is superseded. Canonical rerun keeps entry logic unchanged and uses exact fixed exits `ROI >= 8 || ROI <= -6`. Result: 5401 trades, PF 0.791651, positive symbols 0/4, frequency 7.058813/symbol/week. Years: 2023 PF 0.796669, 2024 PF 0.778185, 2025 PF 0.801633, 2026 PF 0.793469. Final decision remains **frozen**.
