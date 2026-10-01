# 3h Price-Taker Flow Divergence Reversal

Status: **FROZEN after early gate**.

Three completed 1h price closes trend one way while TakerBuyRatio trends the opposite way; current 1h must break the latest completed candle in the reversal direction.

Core-4 result: 1396 trades, PF 0.862791, only 1/4 positive, 1.824496 trades/symbol/week. All yearly PFs 2023–2026 are below 1.

Decision: stable negative expectancy. Freeze; no sign reversal, no 2h/4h window search, no thresholds or extra filters, no DB write.

## Canonical strict-exit audit rerun

With entry logic unchanged and exact fixed `ROI >= 8 || ROI <= -6` exits, canonical result is 1517 trades, normalized PF 0.828099, 0/4 positive, frequency 1.982636/symbol/week. The old conditional-exit evidence is non-canonical and preserved under `legacy/conditional-exit/`. Family remains frozen.

## Canonical strict-exit correction

v80 canonical rerun uses exact fixed exits ROI >= 8 || ROI <= -6. Result: 1517 trades, PF 0.828099, positive symbols 0/4, frequency 1.982636/symbol/week. The prior conditional-exit result is non-canonical and preserved under legacy/conditional-exit/. Final decision: frozen.

## Canonical strict-exit correction

Canonical rerun uses exact fixed exits ROI >= 8 || ROI <= -6 with entry logic unchanged. Result: 1517 trades, PF 0.828099, positive symbols 0/4, frequency 1.982636/symbol/week. Years: 2023 PF 0.993759, 2024 PF 0.834531, 2025 PF 0.708459, 2026 PF 0.819391. Original conditional-exit result is superseded. Final decision: frozen.
