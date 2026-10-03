# v150 DeFiLlama Chain TVL Momentum — 2026-10-02 Discovery

Hypothesis: persistent whole-chain capital inflow/outflow may provide directional information for the native token, distinct from protocol-level TVL and from chain TVL residualized against price.

Frozen signal:
- Reuse the exact 15 native-chain -> Binance USD-M mapping audited by the existing chain-fee/TVL research.
- DeFiLlama historicalChainTvl, complete UTC dates only.
- score = log(TVL_t / TVL_{t-7}); require all dates t-8..t to exist with TVL > 0.
- zero up-cross -> LONG; zero down-cross -> SHORT.
- next UTC-day USD-M open; signed 1d/3d/7d returns.
- dynamic eligibility: USD-M history >=730 days and signal-day QuoteVolume >=5M USDT.
- 7d endpoint must remain inside the same calendar year.
- no magnitude threshold, price residualization, fee/stablecoin/DEX filter, chain-category filter, or symbol-specific rule.

## Final result

2023-2024 discovery:
- **1,240 events / 13 triggered symbols**.
- frequency **0.9134 events/symbol/week**.
- 1d signed mean **+0.1106%**.
- 3d signed mean **-0.0593%**.
- 7d signed mean **-0.1584%**.
- breadth **5/13 positive symbols**.
- 2023: 601 events, 7d **+0.4124%**, 7/10 positive.
- 2024: 639 events, 7d **-0.6954%**, 3/13 positive.

The direction reverses materially between discovery years and overall 7d expectancy is negative. ARB/SUI correctly produce zero discovery events because their pre-audited two-year USD-M eligibility begins only in 2025. XRP chain TVL starts in 2024-03 and therefore contributes only where source history exists.

Decision: **freeze v150**. Do not scan TVL windows, add residual/fee/stablecoin filters, remove weak chains/directions, reverse the rule, or inspect 2025+ for redesign. No exact TP8/SL6 candidate and no DB write.
