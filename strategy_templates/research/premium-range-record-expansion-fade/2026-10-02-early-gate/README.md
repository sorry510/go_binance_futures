# v157 Premium-Range Record Expansion Fade — Early Gate

Mechanism: use Binance USD-M 1h premiumIndexKlines. Premium range is High-Low. A signal occurs only when the current completed 1h premium range is strictly greater than the maximum of the previous 24 completed hours and the prior hour was not itself in record-expansion state. Positive premium close is faded with SHORT; negative premium close with LONG. Entry is the next complete USD-M 1h open.

The 24h record boundary and zero sign boundary were frozen before returns. All 26 premium points needed for the state transition must be strictly hourly-contiguous. No funding/OI/taker/price-trend/QPS filters or premium thresholds.

## Final result

SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024:
- **3,460 events**.
- Frequency: **5.5221 events/symbol/week**.
- 1h signed mean: **+0.0295%**.
- 4h signed mean: **+0.0472%**.
- 12h signed mean: **+0.0359%**.
- Breadth: **3/6 symbols positive**.
- 2023 12h: **+0.0928%**.
- 2024 12h: **-0.0192%**.
- LONG 12h: **+0.2205%**.
- SHORT 12h: **-0.2774%** (audit only; no side deletion permitted).

The combined economic magnitude is far below the +0.20% gate, breadth fails, and 2024 reverses sign.

Decision: **freeze v157**. Do not scan 12h/48h lookbacks, replace record with z-score/multiplier, delete SHORT, add funding/OI filters, or reverse the rule. 2025+ and strict Engine remain unread/unrun; no DB import.
