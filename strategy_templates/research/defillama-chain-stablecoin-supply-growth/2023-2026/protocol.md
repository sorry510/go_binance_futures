# Chain Stablecoin Supply Growth — Corrected Frozen Protocol

This run reopens the 2026-09-28 feasibility-only family because the prior target-chain audit omitted mechanically eligible native chains, including Tron, Polygon and Fantom. No returns were inspected in the superseded feasibility run, so correcting the universe does not use outcome information.

Use the same 15 native-chain map as the later chain-level research universe. DeFiLlama source is stablecoincharts/{chain}, field totalCirculatingUSD.peggedUSD. Binance execution asset is the chain native-token USD-M perpetual.

For each UTC day t, g_t = log(supply_t / supply_{t-7}). First zero-cross upward is LONG; first zero-cross downward is SHORT. Entry is next UTC day's Binance USD-M open. No threshold scan or alternate window is allowed.

Dynamic eligibility at each signal: Binance USD-M history >=2 years and signal-day quote volume >=5M USDT. Lookback is 7 days.

Discovery is 2023-2024. Frozen discovery gate: mean signed 7d endpoint return >= +0.25% AND >=60% of eligible symbols have positive mean 7d return.

Only after a passing discovery gate may 2025 OOS1 and 2026 OOS2 returns be evaluated. Exact TP8/SL6 replay is allowed only after discovery and untouched OOS remain positive. No post-hoc direction inversion, chain filtering, threshold tuning, or window tuning.
