# Protocol

- Sources: DeFiLlama chain DEX dailyVolume, DeFiLlama historical chain TVL, Binance Vision USD-M 1d klines.
- Weekly turnover = sum of latest 7 completed UTC DEX volumes / average TVL over the same 7 days.
- Signal = log(latest weekly turnover / preceding weekly turnover).
- Upward zero-cross => LONG; downward zero-cross => SHORT.
- Entry = next UTC daily open.
- Eligibility: actual Binance USD-M history >=2 years and signal-day QuoteVolume >=5m USDT.
- Discovery = 2023-01-01 through 2024-12-31; full 7d endpoint must remain in discovery.
- Discovery gate = mean7 >=+0.25%, >=60% positive symbols, 2023 mean7 >0, 2024 mean7 >0.
- OOS is 2025 then 2026 and remains untouched until discovery passes.
- Exact replay requires OOS to retain the established economic-strength and breadth standards. No post-hoc relaxation after OOS is seen.
