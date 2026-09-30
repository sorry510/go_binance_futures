# Protocol

Event source is official Binance leverage/margin-tier announcements for USD-M perpetual contracts.

Classification is mechanical and frozen before return inspection:
- smallest-notional tier maximum leverage increases -> LONG;
- decreases -> SHORT;
- unchanged maximum leverage -> exclude, even if notional capacity or maintenance-margin tiers change elsewhere.

Eligibility at event time: contract history >=730 days and previous 24 completed 1h QuoteVolume >=5M USDT.

Execution: next 1m open, 4x, TP8, SL6, fee 0.0005 per side, 5bps slippage per side, funding included, max hold 24h. TP/SL uses minute-close trigger and next-minute-open execution.

2026-09-25 occurred in an unclosed Binance Vision monthly archive. Contract age and prior-24h liquidity were reconstructed from Binance Vision daily 1h files; exact 1m and funding for the current month were filled from Binance public USD-M REST. No threshold or direction changed.

Promotion required positive discovery, OOS1, and OOS2. No post-hoc separation of loosening/tightening directions.
