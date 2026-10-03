# Binance Trading Bots Support → LONG — 2023-2024 Discovery

Frozen universe: the 26 eligibility-passing events from the feasibility audit, covering 13 tokens and 13 independent Binance announcement batches. Direction is LONG. Entry is the first whole-hour USD-M open strictly after official article publishDate.

Early economic gate uses raw 1h/4h/12h log returns. Required before exact replay: event-weighted 12h >=+0.50%, batch-equal 12h >=+0.50%, >=60% token means positive, >=60% batch means positive, and 2024 mean >0. The two 2023 events belong to one BTC/ETH batch and are reported but not used as a standalone annual gate.

## Final result

- 26 events / 13 tokens / 13 batches.
- 1h mean: **-0.5284%**.
- 4h mean: **-1.4130%**.
- 12h mean: **-1.4255%**.
- Batch-equal 12h: **-1.3006%**.
- Positive tokens: **4/13**.
- Positive batches: **4/13**.
- 2023: 2 events from one batch, 12h **+1.7145%**.
- 2024: 24 events, 12h **-1.6871%**, win12 29.17%.

Decision: **freeze**. The main 2024 discovery sample is strongly negative and breadth is poor. Do not split bot types, retain only the 2023 batch, change entry delay/horizon, reverse to SHORT, or inspect 2025+ for redesign. No exact TP8/SL6 replay and no DB import.
