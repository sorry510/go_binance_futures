# Binance Range Bound Underlying Support — Feasibility

Official Binance 2023 Range Bound corpus contains 19 articles: the initial product launch plus 18 recurring "New Range Bound Products Launched" batches.

All article bodies were mechanically audited. Every recurring batch explicitly lists the same three Range Bound underlyings:
- BTC
- ETH
- BNB

The launch article is a generic product introduction and does not define an additional underlying-addition event.

Coverage gate was preregistered at >=8 unique crypto underlyings before any market-return work. Actual unique underlyings = **3**.

Decision: **coverage blocked / freeze feasibility**. Do not treat weekly price-range/settlement-date refreshes as new asset-support events, and do not lower the unique-token gate. No market returns, OOS, exact replay or DB write.
