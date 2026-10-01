# v87 QPS Term-Structure Expansion — 2026-09-30 Early Gate

Hypothesis: an hourly quote-volume-per-second rate crossing above the prior completed day's average quote-volume-per-second marks a transition to unusually active intraday participation. Follow the direction of the completed trigger hour only after current price breaks that hour's extreme.

Frozen before returns. No EMA/ADX/Taker/Funding filters and no threshold scan. Strategy CLOSE rules are deliberately false so exits are isolated to the fixed project TP8/SL6 execution.

Core-4 early gate: BTC/ETH/BNB/XRP, 2023-01-01..2026-09-01. Promotion gate: PF>=1.15, >=3/4 symbols positive, frequency>=0.30 trades/symbol/week, no clear multi-year instability.

## Canonical result

With strict fixed `ROI >= 8 || ROI <= -6` exits: 4777 trades, normalized PF 0.828899, 0/4 symbols positive, all yearly PFs below 1. The initial false-close dry run was invalid and is not evidence. Family frozen.

## Final result

Canonical strict TP8/SL6 rerun: 4777 trades, normalized PF 0.828899, 0/4 symbols positive, 6.243279 trades/symbol/week; yearly PFs 0.762805 / 0.884246 / 0.847831 / 0.790334. The family is frozen. The earlier false-CLOSE dry run was invalid and is not part of the evidence.
