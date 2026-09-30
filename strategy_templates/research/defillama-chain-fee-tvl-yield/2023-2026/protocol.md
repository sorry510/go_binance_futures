# Protocol

- Sources: DeFiLlama chain fees, DeFiLlama historical chain TVL, Binance Vision USD-M 1d klines.
- Weekly fee yield = sum of latest 7 completed UTC daily fees / average TVL over those same 7 days.
- Signal = log(latest weekly fee yield / preceding weekly fee yield).
- Upward zero-cross => LONG; downward zero-cross => SHORT.
- Entry = next UTC daily open.
- Eligibility at signal: actual USD-M history >=2 years and signal-day QuoteVolume >=5m USDT.
- Discovery = 2023-01-01 through 2024-12-31, with full 7d return endpoint required to remain in discovery.
- Discovery gate = aggregate mean7 >=+0.25%, >=60% positive symbols, 2023 mean7 >0, 2024 mean7 >0.
- OOS = 2025 then 2026, untouched until discovery passes.
- Exact TP8/SL6 is allowed only if OOS retains economic magnitude and breadth. No post-hoc window, threshold, direction or symbol filtering.
