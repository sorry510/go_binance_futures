# Protocol

1. Use Binance official CMS catalogId=49 and audit all 2023-2024 titles containing Loan/Loans.
2. Keep only explicit asset removals, delistings, or forced closures of loanable/collateral support; additions, rate changes, product upgrades, NFT Loan, and generic rules are excluded.
3. Parse token names only from explicit affected-asset text. Deduplicate a token within one announcement.
4. Stablecoins are excluded from directional token trading.
5. Before any return is read, require the target Binance USD-M symbol to have at least 730 days of history at announcement time and at least 5M USDT QuoteVolume in the prior 24 complete 1h bars.
6. Coverage gate is at least 8 eligible event-symbols and at least 8 unique tokens. If coverage fails, freeze without reading post-event returns.
7. If coverage passes, direction is preregistered SHORT because removal reduces borrowing/collateral utility and may force position closure.
8. No post-event price reaction, symbol filtering, or event subtype filtering may be used to rescue a failed coverage/discovery stage.
