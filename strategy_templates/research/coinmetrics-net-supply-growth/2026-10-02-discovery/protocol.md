# Discovery Protocol

1. Universe is frozen to the eight feasibility-passing assets: BTC, ETH, XRP, ADA, BCH, LTC, DOGE, ZEC. LINK/UNI are excluded only because Community SplyCur is constant over the full feasibility period.
2. Source is CoinMetrics Community daily SplyCur. Use complete UTC daily observations only.
3. Define current 7d net-supply growth g_now = log(S_t / S_{t-7}) and previous 7d growth g_prev = log(S_{t-7} / S_{t-14}).
4. Supply-growth acceleration a_t = g_now - g_prev.
5. Zero-cross only: a_{t-1} <= 0 and a_t > 0 -> SHORT (dilution accelerating); a_{t-1} >= 0 and a_t < 0 -> LONG (dilution decelerating). No magnitude threshold, z-score, smoothing or re-arm threshold.
6. Signal uses completed UTC day t; entry is next UTC day's Binance USD-M open.
7. Dynamic production eligibility at every event: same-name USD-M history >=730 days at signal time and signal-day QuoteVolume >=5M USDT.
8. Measure signed 1d/3d/7d log returns. The 7d endpoint must remain in the same calendar year as signal to prevent 2024 labels reading 2025.
9. Discovery is 2023-2024 only. 2025+ CoinMetrics values and price outcomes remain unread unless the gate passes.
10. Gate: event-weighted 7d signed mean >= +0.25%; >=60% of triggered-symbol means positive; >=0.30 eligible events/symbol/week; and both 2023/2024 7d signed means >0.
11. Failure freezes this family. Do not scan 3d/14d/30d supply windows, add magnitude thresholds, delete one direction, switch to issuance-only assets, or reverse the economic mapping.
