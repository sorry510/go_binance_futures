# v154 Funding Flat-Zone Feedback Activation — Discovery

Mechanism: Binance standard 8h funding has a 0.01% interest component and a +/-0.05% clamp. v154 uses the **actual funding settlement rate**, not 1h premium close. When the previous normal 8h settlement is exactly 0.0001 and the current normal 8h settlement first leaves 0.0001, >0.0001 emits SHORT and <0.0001 emits LONG.

Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024. No magnitude threshold, z-score, premium filter, OI/taker/trend filter or symbol-specific rule.

## Discovery result

- **1,422 events**
- 12h signed mean **+0.2058%**
- **4/6 symbols positive**
- frequency **2.2695 events/symbol/week**
- 2023 12h **+0.2121%**
- 2024 12h **+0.1980%**
- LONG 1,046 events: **+0.1979%**
- SHORT 376 events: **+0.2277%**

The preregistered discovery gate passed. Rule and gate were frozen and promoted unchanged to 2025 OOS1. Final family decision is determined by the later OOS bundles; do not reinterpret this discovery result in isolation.
