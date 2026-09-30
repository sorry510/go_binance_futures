# DeFiLlama Chain Stablecoin Bridged-Share — 2023–2024 Discovery

Hypothesis: an increasing share of chain stablecoin liquidity that is externally bridged in represents capital entering from outside the chain and should support the native token.

Feature: bridged_share = totalBridgedToUSD.peggedUSD / totalCirculatingUSD.peggedUSD. Use the 7-day change in share. A zero-cross upward is LONG; a zero-cross downward is SHORT. Entry is next UTC daily open.

Universe and production eligibility reuse the frozen 15-chain mapping from chain-stablecoin-supply-growth: actual USD-M history >=2 years and signal-day QuoteVolume >=5m USDT.

Frozen discovery gate: 2023–2024 only, >=80 events, >=8 symbols, aggregate 7d signed mean >=+0.25%, >=60% positive symbols, and both 2023 and 2024 aggregate mean7 >0.

Result: 809 events / 9 triggering symbols. Mean 1d +0.2756%, 3d -0.1169%, 7d -0.1316%; only 4/9 symbols positive. 2023 mean7 -0.5191%, 2024 +0.1714%.

Decision: discovery failed. Freeze the whole bridged-share family; do not reverse direction, tune window, or inspect OOS.
