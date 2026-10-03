# Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use Binance Vision USD-M fundingRate settlements plus USD-M 1h klines. Only normal 8h funding cadence is eligible; adjacent settlements must be 7.5h-8.5h apart.
3. Binance's standard 8h interest component is 0.01% (=0.0001). Under the official +/-0.05% clamp, funding remains exactly 0.01% while the average premium remains inside the flat zone.
4. A signal occurs only when the previous eligible settlement funding rate equals 0.0001 and the current eligible settlement first leaves that flat funding state.
5. Current funding >0.0001 emits SHORT; current funding <0.0001 emits LONG. Exact 0.0001 emits no signal.
6. Equality to 0.0001 is tested at stored funding-rate precision; no tolerance band, z-score, magnitude threshold, or tuned premium threshold is introduced.
7. Enter at the next complete 1h open after the triggering settlement. Measure signed 1h/4h/12h returns; the 12h endpoint must remain in the signal calendar year.
8. No price trend, OI, taker, QPS, premium magnitude, volatility or symbol-specific filter.
9. Early gate: 12h signed mean >=+0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and both 2023 and 2024 12h means >0.
10. Failure freezes this family: do not scan tolerance/magnitude thresholds, use 1h premium closes as a substitute for the settlement-average premium, delete a side, reverse the direction, add filters, or inspect OOS.
11. This is distinct from v148 premium zero-cross and prior funding extreme/zero-cross/streak/volatility studies: the state boundary is the exchange-defined 0.01% flat funding state, not an empirically chosen premium/funding threshold.
