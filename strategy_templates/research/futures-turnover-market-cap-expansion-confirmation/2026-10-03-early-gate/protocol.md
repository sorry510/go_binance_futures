# v168 Futures Turnover / Market-Cap Expansion Confirmation — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC for 2023-2024; 2025+ remains unread unless the frozen gate passes.
2. Market cap uses CoinMetrics Community daily CapMrktEstUSD uniformly for all six assets.
3. Daily USD-M turnover numerator is the sum of QuoteAssetVolume across the 24 complete Binance USD-M 1h bars of the same UTC date.
4. Daily turnover ratio = USD-M daily QuoteVolume / CapMrktEstUSD. Both values must be positive and aligned to the same complete UTC date.
5. Baseline is the arithmetic mean of the previous 30 complete UTC-day turnover ratios, excluding the current date.
6. score = log(current turnover ratio / baseline). Trigger only on a fresh zero up-cross from <=0 to >0.
7. Direction is preregistered activity-confirmed momentum: trailing 7 complete UTC-day USD-M close-to-close return >0 -> LONG; <0 -> SHORT; exact zero -> no signal.
8. Entry is next UTC-day USD-M open. Measure signed 1d/3d/7d returns. The 7d endpoint must remain in the signal calendar year.
9. No turnover magnitude threshold, z-score, funding, OI, taker, basis, volatility, time-of-day, or symbol-specific filter.
10. Early gate: 7d signed mean >= +0.25%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 7d means >0.
11. Failure freezes this family: do not scan 7/14/60d baselines, add turnover thresholds, delete one side, reverse to fade, or combine with v167 OI/market-cap.
12. This is distinct from v167: v167 used OI notional / market cap as a leverage stock and faded expansion; v168 uses actual USD-M traded notional / market cap as turnover flow and tests trend confirmation.
