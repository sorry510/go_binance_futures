# v89 TTM Squeeze Release — 2026-09-30 Early Gate

Hypothesis: volatility release after a standard TTM-style squeeze can create directional continuation large enough for the project's fixed TP8/SL6 exits. Squeeze-on is defined as Bollinger(20,2) fully inside Keltner(20,1.5) on a completed 1h bar. Release occurs when the next completed 1h bar is no longer inside the Keltner channel. Direction is the release bar's candle sign; entry requires current price to break that release bar's extreme.

This is a clean project-native test. It does not use NowTime %, EMA/ADX/Funding/QPS filters, parameter scans, or conditional exits.

Core-4 early gate: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01. Exact exits: ROI >= 8 || ROI <= -6. Fixed leverage 4, fee 0.0005/side, slippage 5bps/side, single position.

Promotion gate: normalized PF >=1.15, >=3/4 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Failure freezes the family without changing Bollinger/Keltner parameters, timeframe, release definition, or adding filters.

## Final result

Canonical strict TP8/SL6 result: 1690 trades, PF 0.841711, 0/4 symbols positive, 2.208738 trades/symbol/week; yearly PF 0.955206 / 0.782989 / 0.850029 / 0.763467. The standard 1h Bollinger(20,2)-inside-Keltner(20,1.5) release definition is frozen without parameter changes or added filters.
