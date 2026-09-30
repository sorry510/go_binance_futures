# Protocol DEX Share Momentum — Frozen Protocol

Research question: does protocol-level DEX market-share improvement lead its token, after removing broad DEX-market activity by using relative share?

Universe is frozen before return inspection: DeFiLlama category=Dexs, protocol ID mapped to protocol.symbol, matching Binance USD-M Vision history. Candidate symbols: UNI, RAY, CRV, SUSHI, DODO, BAL, RUNE, WOO, KNC, LIT, INJ.

For each symbol, aggregate all DeFiLlama DEX adapters sharing that mapped protocol token symbol. Compute trailing-7-day protocol DEX volume divided by trailing-7-day global DEX volume. Signal variable is log(share7[t] / share7[t-7]).

Signal: first daily zero-cross only. Negative/non-positive to positive => LONG; positive/non-negative to negative => SHORT. Entry is next UTC day open. No parameter scan, no direction reversal, no symbol-specific tuning.

Eligibility: signal date must be at least two years after first Binance USD-M daily K-line and signal-day USD-M QuoteVolume must be >= 5,000,000 USDT.

Discovery is strictly 2023-2024. Gate: 7d signed mean >= +0.25% and >=60% of symbols have positive 7d mean. Only if discovery passes may 2025 and 2026 be read. Exact TP8/SL6 replay is allowed only after untouched OOS confirms the endpoint signal.
