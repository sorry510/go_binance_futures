# Protocol

1. v149 tests directional fragility balance between top-trader position skew and visible ±1% book capacity.
2. For each UTC hour, use the last valid completed Binance Vision metrics observation and last bookDepth snapshot within that hour.
3. Let R = sum_toptrader_long_short_ratio, Bid = -1% cumulative bid notional, Ask = +1% cumulative ask notional.
4. score = log(R * Ask / Bid). This equals the log ratio of estimated long-side liquidation pressure per bid capacity to short-side pressure per ask capacity, up to a common scale.
5. score <=0 -> >0 means downside/long-liquidation fragility becomes dominant: emit SHORT. score >=0 -> <0 emits LONG.
6. Require current and prior combined hourly observations exactly one hour apart. Missing metrics/depth hours cannot create synthetic crosses.
7. Enter at the next USD-M 1h open; measure signed 1h/4h/12h returns. No price breakout, funding, OI magnitude, taker, volatility or symbol-specific filter.
8. Stage A is 2023 only. Gate: 12h mean >=+0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week.
9. Only if Stage A passes is 2024 downloaded/evaluated as frozen Stage B validation under the identical rule and gate.
10. 2025+ remains unread unless both discovery stages pass.
11. Failure freezes the family: no depth-band substitution, no score magnitude threshold, no smoothing/z-score, no side deletion, no reversal, no added OI/price/funding filter.
12. This differs from prior bookDepth-only, positioning-only, and flow×depth studies: the signal is the structural ratio between top-position directional skew and opposite-side visible depth.
13. Before computing any forward return, each symbol must have combined hourly feature coverage >=95% of the 2023 hours. If any symbol fails, Stage A is data-blocked and returns are not evaluated.
