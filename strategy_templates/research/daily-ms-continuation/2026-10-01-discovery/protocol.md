# Protocol

1. Freeze the v119 strategy before returns.
2. LONG requires three completed daily bars with rising highs and rising lows; SHORT mirrors with falling highs and lows.
3. A completed 4h candle must align with the daily structure direction.
4. Current price must break the aligned 4h candle extreme.
5. CLOSE_LONG and CLOSE_SHORT are exactly ROI >= 8 || ROI <= -6.
6. Discovery uses SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT only.
7. Failure means no 2-bar/4-bar structure scan, no extra filters, and no reversal.
8. Only a discovery pass may unlock ALGOUSDT, INJUSDT, LDOUSDT, PENDLEUSDT, PYTHUSDT holdout.
