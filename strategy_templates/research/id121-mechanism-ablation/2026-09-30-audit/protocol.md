# ID121 Mechanism Ablation Audit Protocol

Purpose: explain which frozen ID121 components contribute to the original 15-symbol cohort. This is diagnostic, not a candidate-selection stage.

Fixed variants before execution:
- base
- no_funding
- no_daily_regime
- no_4h_strength
- no_freshness
- no_impulse
- no_no_chase

Rules:
1. Each variant removes exactly the named mechanism from the already-frozen base strategy; no threshold or replacement condition is introduced.
2. Universe and start dates are fixed in replay.go and contain only the original ID121 cohort. ALGO/INJ/LDO/PENDLE/PYTH fresh holdout is not read.
3. Exact costs/exits remain leverage=4, TP8, SL6, fee=0.0005/side, slippage=5bps/side, single-position semantics.
4. Ablation results are explanatory only. Do not promote the best ablation, combine removals, or use this audit to retune thresholds.
5. Any future simplified candidate would require a separately preregistered untouched validation set. The second fresh-symbol holdout is currently coverage blocked, so this audit alone cannot create a promoted strategy.
