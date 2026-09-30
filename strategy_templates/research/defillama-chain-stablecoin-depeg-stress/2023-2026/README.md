# DeFiLlama Chain Stablecoin Depeg Stress

Hypothesis: rising chain-level stablecoin depeg stress signals deteriorating dollar-liquidity / credit conditions for the native token; easing stress is supportive.

The signal uses DeFiLlama chain stablecoin charts. For peggedUSD assets, implicit basket peg = totalCirculatingUSD / totalCirculating. Daily stress = abs(log(implicit peg)). Flow = log(mean stress over latest 7 completed UTC days / mean stress over preceding 7 days).

Frozen direction: flow zero-cross upward => SHORT native-token USD-M; zero-cross downward => LONG. Entry is next UTC daily open.

Eligibility is inherited from the audited chain stablecoin universe: actual Binance USD-M history >=2 years at signal date and signal-day QuoteVolume >=5m USDT. Discovery is strictly 2023-2024 and the full 7d endpoint must remain inside the discovery years.

Frozen discovery gate: aggregate 7d signed mean >= +0.25% and >=60% of triggering symbols have positive 7d mean.

Result: 996 events / 11 symbols. Mean signed returns: 1d +0.1224%, 3d +0.0634%, 7d -0.4077%; only 4/11 symbols positive at 7d. Gate failed.

Decision: freeze. OOS was not evaluated. Do not reverse direction, change the 7v7 window, or substitute individual USDT/USDC selection.
