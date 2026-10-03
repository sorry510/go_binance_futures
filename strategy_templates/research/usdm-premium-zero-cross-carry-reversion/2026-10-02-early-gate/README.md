# v148 USD-M Premium Sign-Cross Carry/Reversion — 2026-10-02

Hypothesis: the natural zero boundary of Binance USD-M premium index may encode a persistent carry/reversion effect without requiring an extreme z-score. Positive premium crossing above zero is faded with SHORT; negative premium crossing below zero is faded with LONG.

Frozen definition:
- Binance Vision USD-M 1h premiumIndexKlines.
- Require consecutive completed hourly premium bars.
- premium close <=0 -> >0: SHORT.
- premium close >=0 -> <0: LONG.
- Entry: next USD-M 1h open.
- No premium magnitude threshold, z-score, re-arm, trend, funding magnitude, OI, flow or symbol-specific filter.
- Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only.

## Final result

- **14,901 events**; 8,958 LONG / 5,943 SHORT.
- Frequency **23.7818 events/symbol/week**.
- 1h signed mean **+0.0107%**.
- 4h signed mean **+0.0333%**.
- 12h signed mean **+0.0741%**.
- Breadth **5/6 positive symbols**.
- 2023 12h **+0.0493%**.
- 2024 12h **+0.1064%**.
- LONG 12h +0.1961%; SHORT 12h -0.1096% (direction split is post-result audit only and may not be used to delete SHORT).

The preregistered economic gate requires +0.20% 12h signed mean. Breadth, frequency and annual sign are good, but combined magnitude is only ~7.4 bp and therefore insufficient for the fixed 4x TP8/SL6 execution/cost structure.

Decision: **freeze v148**. Do not keep only the LONG side, scan premium thresholds, switch cadence, add funding filters, reverse the signal, or promote to strict Engine based on the post-hoc direction split. 2025+ remains unread; no DB write.
