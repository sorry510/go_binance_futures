# v81 Rolling-7d Range Acceptance — 2026-09-30 Early Gate

Hypothesis: acceptance outside the highest/lowest price of the prior seven completed daily bars can produce a stronger continuation move than shorter intraday breakouts, while remaining fully project-native.

Entry uses only existing 1d and 1h Kline series. LONG: completed 1h opens at/below the prior-7d high and closes above it; current price then exceeds the signal-hour high. SHORT is symmetric at the prior-7d low.

Frozen early-gate universe: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01. Execution is the project fixed standard: 4x, TP8, SL6, 0.0005 fee/side, 5bps slippage/side, single position.

Promotion gate: normalized PF >=1.15, >=3/4 symbols positive, >=0.30 trades/symbol/week, with no clear multi-year instability. On failure, freeze without changing 7d horizon, acceptance rule, confirmation, or direction.

## Final result

Early gate failed decisively: 819 trades, normalized PF 0.866646, 0/4 symbols positive, frequency 1.070388 trades/symbol/week. 2025 PF was 0.590212. The family is frozen. Per protocol, no 5d/10d horizon tuning, no trend/volume/funding filter, no direction reversal, and no symbol expansion.

## Canonical strict-exit rerun

After correcting CLOSE semantics to exact fixed `ROI >= 8 || ROI <= -6`, the canonical result is 1069 trades, normalized PF 0.890524, 0/4 positive, frequency 1.397125/symbol/week. The earlier conditional-exit result is non-canonical and preserved under `legacy/conditional-exit/`. Family remains frozen.

## Canonical strict-exit correction

The original result used a conditional CLOSE expression and is superseded. Canonical rerun keeps entry logic unchanged and uses exact fixed exits `ROI >= 8 || ROI <= -6`. Result: 1069 trades, PF 0.890524, positive symbols 0/4, frequency 1.397125/symbol/week. Years: 2023 PF 0.918477, 2024 PF 0.931696, 2025 PF 0.821612, 2026 PF 0.902314. Final decision remains **frozen**.
