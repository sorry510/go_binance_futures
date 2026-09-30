# Binance Options IV / Skew Feasibility
Binance Vision options history exposes BVOLIndex only for BTC and ETH. EOHSummary covers BTC, ETH, BNB, XRP and DOGE: five underlyings total. This is below the minimum cross-symbol coverage of eight old assets. Using BTC/ETH options as a broad market factor would also violate the project constraint against Benchmark/MarketCondition-style logic.

Decision: coverage blocked before viewing returns. No five-symbol exception and no DB import.
