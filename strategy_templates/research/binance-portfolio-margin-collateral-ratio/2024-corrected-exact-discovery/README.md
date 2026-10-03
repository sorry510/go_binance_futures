# Binance Portfolio Margin Collateral-Ratio — 2024 Corrected Exact Discovery

Frozen universe: exactly 8 eligibility-passing events / 8 assets / 4 independent official article batches.

Direction remains preregistered:
- collateral-ratio increase -> LONG;
- decrease -> SHORT.

Exact replay:
- effective time signal;
- next complete 1m open;
- leverage 4;
- TP8 / SL6;
- fee 0.0005/side;
- adverse slippage 5bps/side;
- funding included;
- max hold 72h;
- minute-close trigger, next-minute-open exit.

Discovery gate: PF >=1.15, aggregate normalized net >0, and >=3/4 independent article batches positive.

## Final result

- **8 trades: 4 TP / 4 SL**.
- PF **1.3560**.
- normalized net **+9.8082%**.
- SHORT: 5 trades, PF **2.1979**, net **+16.2712%**.
- LONG: 3 trades, PF **0.5372**, net **-6.4630%**.
- Independent positive article batches: **2/4**, below the required 3/4.

The aggregate PF/net conditions pass, but the mandatory independent-batch breadth gate fails. The side split is audit-only; deleting LONG after returns is not allowed.

Decision: **freeze**. Do not convert the family to SHORT-only, change ratio-change thresholds, alter TP/SL or entry semantics, or inspect 2025+ to redesign. No DB import.
