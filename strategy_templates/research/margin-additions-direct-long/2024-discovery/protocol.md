# Protocol

The discovery universe is all 2024 official Binance Margin announcements whose title starts with `Binance Margin Adds`, parsed into token-level events from explicitly added borrowable assets or Margin trading pairs.

Before examining outcome metrics, the following were fixed: LONG direction, 4x leverage, TP 8%, SL 6%, fee 0.0005 per side, 5 bps slippage per side, 72h maximum hold, funding inclusion, single-position semantics, production eligibility requiring at least two years of USD-M 1m history, and entry at the next complete 1m open after the official publish timestamp.

Eligibility is determined without using outcome data. Every event remains in `inputs/eligibility.json`, including excluded events and their reason.

The discovery gate is intentionally simple: if the 2024 sample is materially negative after production costs and does not support the preregistered mechanism, freeze the family immediately. Only a successful discovery result would permit inspection of a later untouched year as OOS.

The 2024 discovery failed with PF 0.5284822667154679 and normalized net -116.44202292291679%, so the family was frozen before 2025 OOS. The failed LONG result must not be converted into a SHORT strategy, and no symbol/month/quote/subtype/reaction/liquidity subgroup may be selected after seeing returns.
