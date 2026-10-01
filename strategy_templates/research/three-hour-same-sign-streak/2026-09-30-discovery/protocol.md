# Protocol

1. Freeze JSON before reading returns.
2. LONG requires the last three completed 1h candles to be bullish; SHORT requires all three bearish.
3. Current price must break the most recent completed 1h high/low.
4. CLOSE_LONG and CLOSE_SHORT are exactly ROI >= 8 || ROI <= -6.
5. Discovery uses only SOL/DOGE/LTC/AVAX/UNI/ZEC.
6. Promotion gate: PF>=1.15, >=4/6 positive, >=0.30 trades/symbol/week, no clear multi-year instability.
7. Failure => no 2-bar/4-bar streak scan, no body-size threshold, no trend/volume/taker/funding filters, no reversal.
8. Fresh holdout remains unread unless discovery passes.
