# Protocol

Hypothesis: an extreme Premium Index identifies perpetual-vs-index crowding; prices should subsequently mean-revert against the premium sign.

Frozen before reading results:
- Binance Vision USD-M premiumIndexKlines, 5m.
- Previous 288 completed 5m closes define the 24h causal mean/std.
- Trigger first |z| >= 3; re-arm only after |z| < 1.
- Positive premium -> SHORT; negative premium -> LONG.
- Enter at the next full 1h open after the completed premium bar.
- Fixed ten old symbols; no symbol-specific filters.
- Discovery 2024, OOS1 2025, OOS2 2026.
- No threshold/window/direction tuning after results.
- Exact 1m TP8/SL6 replay only if discovery and both OOS periods show economically meaningful, cross-symbol stable edge.
