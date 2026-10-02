# Protocol

1. Freeze six-symbol 2023-2024 discovery before reading returns.
2. For adjacent completed 1h bars, overlap ratio = intersection length of high-low ranges divided by the smaller bar range; clamp at zero when ranges do not overlap.
3. For each current pair, compare its overlap ratio with the mean of the previous 24 completed pair-overlap ratios, excluding the current pair.
4. Event occurs only on a fresh crossing from previous pair overlap >= its own prior-24 mean to current pair overlap < current prior-24 mean.
5. Direction is LONG when current high-low midpoint > previous midpoint; SHORT when lower. Equal midpoint gives no event.
6. Enter next complete 1h open.
7. No volume, taker, trend, funding, ATR, candle-body, gap, or symbol-specific filter.
8. Early gate: 12h signed mean >=+0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, both 2023/2024 12h means >0.
9. Failure freezes this family: no lookback scan, overlap fixed threshold, body/wick filter, or reversal.
10. 2025+ OOS remains unread unless early gate passes.
