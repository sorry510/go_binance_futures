# v84 ADX20 Trend Start — 2026-09-30 Early Gate

Hypothesis: an hourly ADX14 transition from <=20 to >20 marks the start of a directional trend episode. DI+/DI- chooses direction; the current hour must break the completed trigger-hour extreme before entry.

Frozen before returns. No EMA, volume, funding, Donchian, or threshold scan is used. Core-4 early gate: BTC/ETH/BNB/XRP, 2023-01-01..2026-09-01. Fixed execution: 4x, TP8, SL6, fee0.0005/side, slippage5bps/side, single position.

Promotion gate: PF>=1.15, >=3/4 symbols positive, frequency>=0.30 trades/symbol/week, no clear multi-year instability. Failure freezes the family without trying ADX18/25, extra trend filters, or opposite direction.

## Final result

Failed preregistered early gate: 779 trades, normalized PF 1.079869, 3/4 symbols positive, but 2024 PF 0.759475 and overall PF below 1.15. Freeze without ADX threshold tuning or added filters.

## Canonical strict-exit rerun

After correcting CLOSE semantics to exact fixed `ROI >= 8 || ROI <= -6`, the canonical result is 852 trades, normalized PF 1.011205, 3/4 positive, frequency 1.113518/symbol/week. The earlier conditional-exit result is non-canonical and preserved under `legacy/conditional-exit/`. Family remains frozen.

## Canonical strict-exit correction

The original result used a conditional CLOSE expression and is superseded. Canonical rerun keeps entry logic unchanged and uses exact fixed exits `ROI >= 8 || ROI <= -6`. Result: 852 trades, PF 1.011205, positive symbols 3/4, frequency 1.113518/symbol/week. Years: 2023 PF 1.008415, 2024 PF 0.884756, 2025 PF 1.074132, 2026 PF 1.142108. Final decision remains **frozen**.
