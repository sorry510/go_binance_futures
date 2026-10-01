# ID121 Exact TP8/SL6 Early OOT — 2021H2–2022 Audit

Purpose: re-evaluate the known pre-2023 weakness of ID121 after correcting the Engine exit semantics. This is not alpha discovery and no entry rule is changed.

Canonical strategy is the frozen ID121 exact-exit snapshot from the 2026-09-30 candidate audit. Symbols are BTCUSDT and ETHUSDT only because local 1m history exists from 2021-01 and supports the required daily-indicator warmup. Evaluation window is 2021-07-20 inclusive through 2023-01-01 exclusive.

Execution remains leverage 4, exact CLOSE ROI >= 8 || ROI <= -6, fee 0.0005/side, slippage 5bps/side, single position. No DB write, no threshold adjustment, no comparison-driven tuning.

## Final result

Strict exact-exit early OOT remains decisively weak: BTC 42 trades PF 0.603733; ETH 37 trades PF 0.569195; combined 79 trades PF 0.587214, 0/2 symbols positive, 0.521698 trades/symbol/week. 2021H2 PF 0.303450 and 2022 PF 0.697847. Exit-semantic correction does not remove the pre-2023 regime failure. ID121 may remain a current-regime candidate, but it is not supported as an all-cycle strategy.
