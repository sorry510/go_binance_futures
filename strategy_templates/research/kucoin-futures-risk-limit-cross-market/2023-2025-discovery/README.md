# KuCoin Futures Leverage/Risk-Limit Cross-Market
Official KuCoin adjustment pages were parsed mechanically from before/after maximum leverage and maximum risk-limit tables. Discovery strict eligibility produced 112 events / 86 Binance symbols / 12 batches.

Results: 1h -0.0617%, 4h -0.6097%, 12h -1.0978%; batch-equal 12h -0.8755% with 5/12 positive batches. 2023 +0.6359%, 2024 -2.0438%, 2025 -3.7504%. Only 28/86 symbols positive. LONG and SHORT subsets were both negative.

Decision: freeze discovery family; do not parse/use 2026 OOS, do not rescue by direction, threshold, or sub-period.
