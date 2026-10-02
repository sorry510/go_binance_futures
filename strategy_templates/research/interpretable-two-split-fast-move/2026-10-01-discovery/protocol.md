# Protocol

- 2023 train, 2024 validation; 2025/2026 remain unread.
- Six symbols: SOL/DOGE/LTC/AVAX/UNI/ZEC USDT perpetuals. No symbol feature.
- Every completed 1h state generates a LONG and SHORT candidate. Directional features are multiplied by side (+1 LONG, -1 SHORT), so one common tree learns symmetric cross-symbol rules.
- Entry is the next 1m open after the completed 1h bar, with adverse entry slippage 5bps. Label path uses future 1m CLOSE only for up to 24h. Gross 4x ROI reaching +8 first gives reward +8; reaching -6 first gives reward -6. Unresolved samples are excluded from fitting/evaluation.
- Signals whose 24h label path would cross a train/validation year boundary are excluded.
- Fixed nine DSL-translatable features, calculated only from completed 1h bars:
  1. ret1_side = side * log(C0/O0).
  2. ret4_side = side * log(C0/C4).
  3. ret12_side = side * log(C0/C12).
  4. body_range_side = side * (C0-O0)/(H0-L0), zero when range is zero.
  5. taker_imb_side = side * (2*TakerBuyQuoteVolume0/QuoteVolume0 - 1).
  6. quote_ratio_8h = QuoteVolume0 / mean(QuoteVolume1..8).
  7. range_ratio_8h = log(H0/L0) / mean(log(H/L)1..8).
  8. taker_delta_8h_side = side * (TakerBuyRatio0 - mean(TakerBuyRatio1..8)).
  9. close_loc_12h_side = side * (2*(C0-min(L0..11))/(max(H0..11)-min(L0..11)) - 1).
- Protocol correction before any return run: the initial skeleton named trade_ratio_8h, but current production DSL does not expose trade_count. It is replaced pre-outcome by taker_delta_8h_side; no result from either feature set has been observed.
- Tree depth exactly 2 when valid splits exist. Candidate thresholds are only 2023 resolved-sample q10/q25/q50/q75/q90 for each feature. Minimum child size is 2000.
- CART-style regression split criterion is fixed. Train leaves are selected only when their mean reward (+8/-6) >= +1.0.
- Frozen 2024 validation gate: >=1000 resolved selected samples, mean reward >= +0.5, and >=4/6 symbols with positive selected-sample mean reward.
- Failure freezes the route without tuning depth, quantile grid, features, leaf size, reward thresholds, label horizon, or side handling.
- Only if validation passes may 2025/2026 be read and a strict Engine strategy be generated. Exact production exits must be `ROI >= 8 || ROI <= -6`, leverage=4, fee=0.0005/side, slippage=5bps/side, single-position.
