# Protocol

Hypothesis: an extreme funding settlement identifies a crowded side; after settlement, the crowded side should underperform as positions rebalance.

Rules were frozen before reading results:
- Previous 30 completed funding observations define mean/std.
- Trigger only when |z| >= 2 and the event is armed.
- Positive funding trades SHORT; negative funding trades LONG.
- Re-arm only after |z| < 1.
- No symbol-specific rules.
- Discovery: 2023-2024. OOS: 2025-2026.
- First stage uses conservative next-hour entry and 1h/4h/12h signed returns.
- Only a stable, economically meaningful discovery result that survives OOS may enter exact 1m TP8/SL6 replay.
- Do not tune funding window, z threshold, re-arm threshold, entry delay, or reverse the direction after observing results.
