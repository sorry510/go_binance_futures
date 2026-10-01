# Previous-Day Range Acceptance Continuation

Status: **FROZEN after early gate**.

Hypothesis: when the previous completed daily candle closes beyond the prior day's high/low and the current day extends beyond that accepted range in the same direction, a fast continuation move may be large enough to match fixed 4x / TP8 / SL6 execution.

The family is distinct from old daily-regime + 4h Donchian templates: the event is explicit previous-day range acceptance followed by next-day continuation.

Early gate, BTC/ETH/BNB/XRP, 2023-01-01 through 2026-09-01: 273 trades, standardized PF 1.070523, 2/4 symbols positive, 0.356796 trades/symbol/week. 2023 PF 0.983705 and 2024 PF 0.928613. Gate failed.

Decision: freeze. No tuning of acceptance semantics, EMA/ADX, direction, or filters; no expansion; no DB write.

Canonical evidence: `strategy.json`, `protocol.md`, `config.json`, `results/summary.json`, `results/engine.log`, `replay.go`, `provenance.json`, `manifest.sha256`.

## Canonical strict-exit audit rerun

With entry logic unchanged and exact fixed `ROI >= 8 || ROI <= -6` exits, canonical result is 2856 trades, normalized PF 0.859759, 0/4 positive, frequency 3.732636/symbol/week. The old conditional-exit evidence is non-canonical and preserved under `legacy/conditional-exit/`. Family remains frozen.

## Canonical strict-exit correction

v77 canonical rerun uses exact fixed exits ROI >= 8 || ROI <= -6. Result: 2856 trades, PF 0.859759, positive symbols 0/4, frequency 3.732636/symbol/week. The prior conditional-exit result is non-canonical and preserved under legacy/conditional-exit/. Final decision: frozen.

## Canonical strict-exit correction

Strict fixed TP8/SL6 rerun using ROI >= 8 || ROI <= -6: 2856 trades, PF 0.859759, 0/4 positive, frequency 3.732636/symbol/week; yearly PF 0.834642 / 0.903357 / 0.850932 / 0.809999. The original conditional-exit result is superseded and preserved under legacy/conditional-exit/. Final decision: frozen.
