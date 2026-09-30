# Protocol DEX Market-Share Flow — Frozen Protocol

Hypothesis: persistent gains/losses in a DEX protocol token's share of aggregate DeFi DEX volume reflect protocol-level competitive usage changes that may lead the protocol token.

Universe construction is mechanical. DeFiLlama DEX entries are mapped through defillamaId -> protocol.id -> protocol.symbol. All DEX versions sharing the same token symbol are aggregated before any signal is calculated. Symbols require Binance USD-M history and dynamic >=2-year history at each signal.

Signal uses daily UTC data only. share_t = aggregated protocol DEX dailyVolume / aggregate DEX dailyVolume. flow_t = log(mean(share[t-6:t]) / mean(share[t-13:t-7])). A first zero-cross upward is LONG; a first zero-cross downward is SHORT. Entry is the next UTC day's Binance USD-M open.

No threshold scan, no symbol-specific rules, no category filtering after returns are observed, and no post-hoc direction inversion are allowed.

Discovery = 2023-2024. OOS1 = 2025. OOS2 = 2026 through available data. Discovery gate is mean signed 7d endpoint return >= +0.25% AND >=60% of eligible symbols have positive mean 7d return. Only if discovery passes may OOS be used to decide continuation.

Exact TP8/SL6 replay is allowed only if discovery and untouched OOS endpoint gates remain positive. Exact execution must use 4x leverage, fee 0.0005/side, slippage 5bps/side, funding, single position, and the fixed next-day entry boundary.
