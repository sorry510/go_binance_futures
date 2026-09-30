# DeFiLlama Chain DEX Turnover / TVL

Mechanism: weekly DEX trading volume relative to average chain TVL measures capital turnover / trading intensity per unit of DeFi capital. Rising turnover is fixed LONG the native token; falling turnover is fixed SHORT.

Signal was frozen before returns: latest 7 completed UTC days DEX volume sum / same 7-day average TVL, compared with the preceding 7-day window. Zero-cross upward LONG, downward SHORT. Entry is next UTC daily open. Eligibility requires >=2 years Binance USD-M history and signal-day QuoteVolume >=5m USDT.

Discovery partition is 2023-2024; the full 7d endpoint must stay inside discovery. Frozen discovery gate: aggregate mean7 >=+0.25%, >=60% symbols positive, and both 2023 and 2024 aggregate mean7 >0.

Discovery: 922 events / 12 symbols, mean7 +0.8263%, 10/12 positive. 2023 +0.9719%; 2024 +0.6839%.

Untouched OOS remained directionally positive. 2025: 542 events / 13 symbols, mean7 +0.2203%, 8/13 positive. 2026: 388 events / 13 symbols, mean7 +0.3817%, 9/13 positive.

Decision: retain as a near-pass research family, but do not enter exact TP8/SL6. 2025 is below the established +0.25% economic-strength bar; after seeing OOS we may not relax that bar to +0.20%. Do not tune the 7v7 window, threshold, direction, or symbol subset.
