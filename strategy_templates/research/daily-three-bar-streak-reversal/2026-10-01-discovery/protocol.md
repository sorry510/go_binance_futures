# Protocol

1. Freeze strategy JSON before reading returns.
2. LONG regime requires the prior three completed daily candles all bearish.
3. SHORT regime requires the prior three completed daily candles all bullish.
4. A completed 4h candle in the opposite direction is the reversal trigger.
5. Current price must break that 4h candle's high/low in the reversal direction.
6. CLOSE_LONG and CLOSE_SHORT are exactly ROI >= 8 || ROI <= -6.
7. Discovery uses only SOL/DOGE/LTC/AVAX/UNI/ZEC.
8. Failure => no 2-day/4-day streak scan, no added trend/volume/funding filters, no reversal into continuation.
9. Pass => freeze parameters and evaluate fresh holdout ALGO/INJ/LDO/PENDLE/PYTH.
