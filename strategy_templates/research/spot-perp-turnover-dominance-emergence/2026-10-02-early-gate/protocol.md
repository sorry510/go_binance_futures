# Protocol

1. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only. 2025+ remains unread unless the frozen gate passes.
2. Use Binance Vision Spot and USD-M Futures 1h Klines only; align exact completed UTC hours.
3. For each completed hour t, compute trailing-24h Spot QuoteVolume and trailing-24h USD-M QuoteVolume from the same 24 aligned completed hours.
4. turnover_ratio = Spot_QV24 / Perp_QV24. Trigger only when prior ratio <=1 and current ratio >1: spot turnover has newly become dominant.
5. Direction is the sign of Spot 24h log return over the same completed 24h window: positive -> LONG, negative -> SHORT; exact zero -> no signal.
6. No signal is assigned to the reverse cross back below 1. Perp dominance has no natural bullish/bearish direction and is not forced into a mirror trade.
7. Enter at the next complete USD-M 1h open; measure signed 1h/4h/12h returns.
8. Require all 24 aligned Spot/Perp hours and all forward USD-M bars to be contiguous. A 12h endpoint must remain inside the same calendar year.
9. No z-score, QV magnitude threshold, taker/funding/OI/trend filter, symbol-specific rule, or alternative return window.
10. Early gate: 12h signed mean >=+0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and 2023/2024 12h means both >0.
11. Failure freezes the family: do not scan 12h/48h turnover windows, shift the 1.0 ratio boundary, add price/flow filters, use the reverse cross, or inspect OOS.
12. This is distinct from prior Spot/Perp trade-count-share and average-trade-notional 720h z-score studies.
