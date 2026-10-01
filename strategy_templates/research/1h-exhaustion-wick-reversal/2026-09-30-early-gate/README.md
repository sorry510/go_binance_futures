# 1h Exhaustion Wick Reversal

Status: **FROZEN after early gate**.

Mechanism: 20-bar fresh extreme after >=7/9 incoming-direction 1h candles, rejection wick/body >0.66, followed by confirmed reversal breakout.

Core-4 2023-01-01 through 2026-09-01: 495 trades, PF 0.652453, 0/4 positive, 0.646938 trades/symbol/week. Every year PF <1.

Decision: stable negative expectancy; freeze without changing lookback, 7/9 rule, wick threshold, timeframe, direction, or adding filters. No DB write.

## Canonical strict-exit audit rerun

With entry logic unchanged and exact fixed `ROI >= 8 || ROI <= -6` exits, canonical result is 516 trades, normalized PF 0.682287, 0/4 positive, frequency 0.674384/symbol/week. The old conditional-exit evidence is non-canonical and preserved under `legacy/conditional-exit/`. Family remains frozen.

## Canonical strict-exit correction

v79 canonical rerun uses exact fixed exits ROI >= 8 || ROI <= -6. Result: 516 trades, PF 0.682287, positive symbols 0/4, frequency 0.674384/symbol/week. The prior conditional-exit result is non-canonical and preserved under legacy/conditional-exit/. Final decision: frozen.

## Canonical strict-exit correction

Canonical rerun uses exact fixed exits ROI >= 8 || ROI <= -6 with entry logic unchanged. Result: 516 trades, PF 0.682287, positive symbols 0/4, frequency 0.674384/symbol/week. Years: 2023 PF 0.920361, 2024 PF 0.573741, 2025 PF 0.634967, 2026 PF 0.752225. Original conditional-exit result is superseded. Final decision: frozen.
