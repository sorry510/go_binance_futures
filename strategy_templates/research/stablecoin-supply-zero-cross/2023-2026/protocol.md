# Protocol
Use DefiLlama aggregate USD-pegged stablecoin circulating USD. Calculate the sign of the 7-day change using completed daily records. Trigger only when the sign crosses zero. Positive crossing is LONG; negative crossing SHORT. The record labeled day d is not tradable until d+1 00:00 UTC.

Contract-age eligibility is based on Binance Vision, not the local DB first cached bar. Nine symbols already have Jan-2021 USD-M monthly data; ZEC is conservatively eligible from Jun-2023 based on Jun-2021 monthly availability.

This stage only tests signed forward 24h/72h moves. No reverse-direction rescue or threshold/window search after results.
