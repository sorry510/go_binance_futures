# DeFiLlama Chain Stablecoin Supply Growth — Corrected Discovery

This run supersedes the earlier feasibility-only freeze. The original coverage audit omitted mechanically eligible native chains, including Tron, Polygon and Fantom. Because the earlier run inspected no returns, the universe correction is outcome-independent.

The corrected universe uses the same 15 native-chain mapping as later chain-level research. Twelve chains have 2023-2024 stablecoin history; BTC has no endpoint, Sui begins in 2024, and XRPL begins in 2025. Dynamic Binance USD-M >=2-year eligibility further controls signal participation.

Frozen signal: g_t = log(chain stablecoin supply_t / supply_t-7d), using DeFiLlama totalCirculatingUSD.peggedUSD. Upward zero-cross LONG, downward zero-cross SHORT, next UTC day Binance USD-M open. Signal-day quote volume must be >=5M USDT.

Discovery 2023-2024 produced 1086 events across 11 symbols with events. Mean signed returns: 1d -0.1924%, 3d -0.0759%, 7d +0.1777%. Seven of 11 symbols (63.6%) had positive mean 7d return.

The pre-return discovery gate required BOTH 7d mean >= +0.25% and >=60% positive symbols. Breadth passed, economic magnitude failed. Therefore the family is frozen. 2025/2026 OOS was not evaluated and exact TP8/SL6 replay was not entered.

No threshold adjustment, symbol filtering, alternate window, or post-hoc direction inversion is permitted.

Canonical evidence:
- protocol.md / config.json: corrected frozen protocol.
- inputs/coverage_correction.json: 15-chain source coverage correction.
- inputs/universe.json: dynamic USD-M eligibility and event coverage.
- results/events.json: discovery events only.
- results/summary.json: gate failure and oos_evaluated=false.
- replay.py: stage-gated replay.
