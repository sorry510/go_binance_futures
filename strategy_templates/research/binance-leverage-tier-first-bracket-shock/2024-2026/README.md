# Binance Leverage-Tier First-Bracket Shock

Hypothesis: exchange-imposed changes to the maximum leverage available in the smallest-notional USD-M tier alter immediate risk capacity. Increases were pre-registered LONG; decreases SHORT.

Exact 1m results:
- **2024 discovery:** 9 trades, 6 TP / 3 SL, PF **2.0625**, normalized net **+24.42%**, avg +2.71%.
- **2025 OOS1:** 10 trades, 6 TP / 4 SL, PF **1.4967**, normalized net **+15.94%**, avg +1.59%.
- **2026 OOS2:** 10 trades, 3 TP / 7 SL, PF **0.4334**, normalized net **-33.99%**, avg -3.40%.

The mechanism passed discovery and first OOS but failed hard in the second forward period. 2026 is not rescued by splitting directions, reversing tightening events, changing holding horizon, or lowering eligibility thresholds after seeing returns.

Decision: **freeze / not a production candidate / no DB import**.
