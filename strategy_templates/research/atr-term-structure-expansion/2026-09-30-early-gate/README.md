# v82 ATR Term-Structure Expansion — 2026-09-30 Early Gate

Project-native hypothesis: when 1h ATR14, scaled by sqrt(24), crosses above 1d ATR14, short-horizon volatility has expanded relative to the daily regime. Follow the direction of the completed trigger hour only after current price breaks that trigger hour's extreme.

Frozen before returns. No threshold scan: scale is sqrt(24), crossing level is exactly 1. Core-4 early gate uses BTC/ETH/BNB/XRP, 2023-01-01 through 2026-09-01. Fixed execution is 4x, TP8, SL6, fee 0.0005/side, slippage 5bps/side, single position.

Promotion gate: PF>=1.15, >=3/4 symbols positive, frequency>=0.30 trades/symbol/week, with no obvious multi-year instability. Failure freezes the family without changing ATR periods, scale, cross level, or direction.

## Final result

Early gate failed on economic magnitude despite decent breadth and year-by-year sign: 758 trades, normalized PF 1.090303, 3/4 symbols positive, frequency 0.990665/week. All four year PFs were slightly above 1, but the preregistered PF>=1.15 gate was not met. Freeze without tuning ATR period, sqrt(24) scale, cross level, or adding filters.

## Canonical strict-exit rerun

After correcting CLOSE semantics to exact fixed `ROI >= 8 || ROI <= -6`, the canonical result is 936 trades, normalized PF 0.853439, 0/4 positive, frequency 1.223301/symbol/week. The earlier conditional-exit result is non-canonical and preserved under `legacy/conditional-exit/`. Family remains frozen.

## Canonical strict-exit correction

The original result used a conditional CLOSE expression and is superseded. Canonical rerun keeps entry logic unchanged and uses exact fixed exits `ROI >= 8 || ROI <= -6`. Result: 936 trades, PF 0.853439, positive symbols 0/4, frequency 1.223301/symbol/week. Years: 2023 PF 0.845737, 2024 PF 0.928144, 2025 PF 0.906238, 2026 PF 0.693261. Final decision remains **frozen**.
