# Chain Stablecoin Velocity — Frozen Protocol

Hypothesis: DEX turnover per dollar of chain stablecoin liquidity measures speculative/transactional velocity. Rising velocity can reflect increasing use of available on-chain dollar liquidity and may lead the native token; falling velocity may indicate activity decay.

Universe is the same mechanically defined 15 native-chain map used by other chain-level research. Require both DeFiLlama chain DEX dailyVolume and stablecoincharts totalCirculatingUSD.peggedUSD. Dynamic Binance USD-M eligibility is >=2 years plus signal-day quote volume >=5M USDT.

For each day, weekly velocity = sum(DEX dailyVolume over 7 consecutive UTC days) / mean(stablecoin supply over the same 7 days). flow_t = log(recent 7d velocity / prior non-overlapping 7d velocity).

First zero-cross upward of flow is LONG; first zero-cross downward is SHORT. Entry is next UTC day's Binance USD-M open. Lookback is 14 days. No threshold scan, alternate window, symbol-specific filter, or direction inversion.

Discovery = 2023-2024. Frozen gate: mean signed 7d endpoint return >= +0.25% AND >=60% of eligible symbols have positive mean 7d return. OOS 2025/2026 may be evaluated only after discovery passes. Exact TP8/SL6 replay is allowed only after discovery and untouched OOS remain positive.
