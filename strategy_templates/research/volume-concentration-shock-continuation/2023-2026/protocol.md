# Protocol
24h QuoteVolume concentration is measured by HHI=sum(hourly volume share squared), normalized by a causal prior-720h z-score. First z>=3 triggers; z<1 rearms. Direction follows the completed trailing-24h return.
After diagnostic testing, exact replay was frozen to next 1m open, 4x, TP8/SL6, fee/slippage/funding and 12h timeout. Only 2023-2024 exact discovery was required; failure blocks OOS exact replay.
