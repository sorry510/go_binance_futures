# v155 Funding→Premium Feedback Failure Fade

Mechanism: after a normal 8h funding settlement at T, wait for the first complete 1h premium-index bar. If funding and that post-settlement premium close remain the same nonzero sign, the funding payment failed to remove the basis sign. Trigger only on false→true activation of this failure state. Positive premium/funding -> SHORT; negative -> LONG. Entry is the USD-M 1h open at T+1h.

Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC in 2023-2024. No magnitude threshold, z-score, OI/taker/QPS/price/trend filter, smoothing, or symbol-specific rule.

## Final result

- 1,451 events; frequency **2.3158 events/symbol/week**.
- 1h signed mean **+0.0517%**.
- 4h signed mean **+0.0963%**.
- 12h signed mean **+0.1232%**.
- Breadth: **5/6 symbols positive**.
- 2023 12h **+0.1548%**.
- 2024 12h **+0.0835%**.
- LONG: 672 events, 12h **+0.3448%**.
- SHORT: 779 events, 12h **-0.0680%**.

The combined mechanism is directionally stable across both years and five of six symbols, but the preregistered +0.20% economic gate fails. The LONG/SHORT split is audit-only and cannot justify deleting SHORT after outcomes.

Decision: **freeze v155**. Do not change the post-settlement observation window, add premium/funding thresholds, delete a direction, add filters, or inspect 2025+. No strict TP8/SL6 replay and no DB import.
