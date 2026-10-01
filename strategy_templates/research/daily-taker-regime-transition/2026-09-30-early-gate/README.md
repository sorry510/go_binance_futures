# v86 Daily Taker Regime Transition — 2026-09-30 Early Gate

Hypothesis: a completed daily TakerBuyRatio cross of the neutral 0.5 level marks a next-day directional taker-flow regime. During that current day, aligned completed 1h candles are eligible only when the next hour breaks their extreme.

Frozen before returns. No EMA/ADX/QPS/funding filters, no threshold scan, no time modulo. Core-4 early gate: BTC/ETH/BNB/XRP, 2023-01-01..2026-09-01. Fixed project execution: 4x, TP8, SL6, fee0.0005/side, slippage5bps/side, single position.

Promotion gate: PF>=1.15, >=3/4 symbols positive, frequency>=0.30 trades/symbol/week, no clear multi-year instability.

## Final result

Failed decisively: 1367 trades, normalized PF 0.851866, 0/4 positive, and every yearly PF below 1. Freeze without threshold changes or added filters.

## Canonical strict-exit rerun

After correcting CLOSE semantics to exact fixed `ROI >= 8 || ROI <= -6`, the canonical result is 3710 trades, normalized PF 0.789004, 0/4 positive, frequency 4.848768/symbol/week. The earlier conditional-exit result is non-canonical and preserved under `legacy/conditional-exit/`. Family remains frozen.

## Canonical strict-exit correction

The original result used a conditional CLOSE expression and is superseded. Canonical rerun keeps entry logic unchanged and uses exact fixed exits `ROI >= 8 || ROI <= -6`. Result: 3710 trades, PF 0.789004, positive symbols 0/4, frequency 4.848768/symbol/week. Years: 2023 PF 0.726912, 2024 PF 0.721337, 2025 PF 0.920370, 2026 PF 0.770767. Final decision remains **frozen**.
