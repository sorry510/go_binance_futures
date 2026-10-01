# Protocol

1. Freeze the strategy JSON before reading returns.
2. Squeeze-on: completed bar [2] has Bollinger upper below Keltner upper and Bollinger lower above Keltner lower.
3. Release: completed bar [1] is no longer fully inside Keltner.
4. Direction = completed release-bar candle sign.
5. Entry requires current price to break the release-bar high/low.
6. CLOSE_LONG and CLOSE_SHORT are exactly ROI >= 8 || ROI <= -6.
7. Core-4 early gate only.
8. Failure => no Boll/KC parameter scan, no 4h variant, no EMA/ADX/QPS/funding filter, no reversal.
9. Pass => freeze parameters and expand symbols before holdout.
