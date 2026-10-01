# ID121 Exact-Exit Robustness Audit — 2026-10-01

Purpose: audit the frozen canonical ID121 exact TP8/SL6 candidate without changing any strategy rule.

Source evidence is the 773 canonical exact-exit trades from `id121-v54-exact-exit/2026-09-30-audit`. This audit measures normalized-return concentration by symbol and calendar block, leave-one-symbol-out PF, monthly/quarterly PF, and a reproducible natural-month block bootstrap that preserves contemporaneous cross-symbol dependence within each sampled month.

No parameter selection, symbol deletion, DB write, or strategy modification is allowed. Bootstrap seed is fixed at 121 and uses 10,000 resamples of calendar-month blocks.

## Final result

Canonical ID121 contains 773 exact-exit trades with normalized PF 1.211221. Leave-one-symbol-out PF ranges from 1.147744 (remove XRP) to 1.257755 (remove ZEC), so no single symbol is solely responsible for portfolio profitability. 15/25 calendar months and 8/9 calendar quarters have positive normalized net; the only negative quarter is 2026-Q2 with PF 0.857713.

The top three normalized-net contributors are XRP, ETH and BTC. They account for 61.64% of total positive-symbol net and 69.60% of overall portfolio net, so contribution concentration remains a material monitoring risk.

Natural-month block bootstrap, 10,000 resamples with seed 121: PF q01=0.930815, q05=1.015463, q10=1.057228, median=1.207487, q90=1.362973, q95=1.405628. Only 3.71% of bootstrap samples have PF <=1. This supports the existence of a 2024+ block-level edge, while the separate 2021H2-2022 OOT failure still prevents any all-regime claim.
