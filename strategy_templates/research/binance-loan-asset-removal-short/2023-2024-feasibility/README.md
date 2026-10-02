# Binance Loan / VIP Loan Asset Removal → USD-M SHORT — 2023-2024 Feasibility

Hypothesis: explicit removal of a token from Binance Loans / VIP Loan reduces borrowing or collateral utility and can force loan-position closure, so a production-eligible token would be preregistered SHORT after the official announcement.

Source audit:
- Binance official CMS catalogId=49.
- Exhaustive 2023-2024 title scan found 74 Loan-related articles.
- Four explicit removal/forced-close batches were identified: PEPE borrowable removal; selected loanable assets (IRIS/IQ/OAX/JUV/MULTI/ARDR/ATM/MLN); BUSD loan+collateral closure; PLA Margin+VIP Loan removal.
- BUSD is excluded as a stablecoin. This leaves 10 token-events / 10 unique tokens before production eligibility.

Production eligibility was checked before any post-event return:
- Binance USD-M history >=730 days at announcement time.
- Prior 24 complete 1h QuoteVolume >=5M USDT.
- Minimum coverage gate: >=8 eligible events and >=8 unique tokens.

## Final result

Eligible events: **0/10**. All 10 non-stablecoin tokens fail at the first gate because the corresponding USD-M contract either did not exist or did not yet have two years of history at the event timestamp. QuoteVolume and post-event returns were therefore not used for selection, and no return series was evaluated.

Decision: **feasibility blocked / freeze**. Do not lower the two-year production requirement, substitute later history, or merge unrelated Margin/Spot removal events to manufacture coverage. This is not an alpha failure; the event family is structurally incompatible with the current production eligibility rule.
