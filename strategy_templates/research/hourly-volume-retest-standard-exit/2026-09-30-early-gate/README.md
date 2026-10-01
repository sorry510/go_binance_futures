# Hourly Volume Retest — Standard Exit

Status: pre-registered early gate; results not read at creation time.

This study preserves the entry logic of existing `strategy_templates/hourly-volume-retest-strength-v4.json`: 1h volume/OBV impulse, controlled lower-volume retest, reclaim, 4h EMA34 trend and 1h ADX direction. Original non-standard CLOSE logic is disabled so all exits follow the fixed research execution.

## Final result

Canonical strict fixed-exit early gate completed: 1771 trades, normalized PF 0.852652, 0/4 symbols positive, 2.314600 trades/symbol/week. Every yearly PF is below 1. The family is frozen; no entry-threshold tuning or six-symbol expansion.
