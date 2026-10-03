# v197 Multi-Mechanism Reversal Consensus — Discovery

This is an explicitly trained meta-hypothesis built only from already-read 2023-2024 discovery evidence. It is not independent validation.

Frozen source families:
- v190 Trading-Invariant Stress Reversal
- v192 Intrahour Volatility-Signature Noise Reversal
- v194 Three-Bar Fair-Value-Gap Fill Reversal
- v196 Intrahour Open-Reference Recross Activation Reversal

Rule:
- same symbol and signal hour;
- at least 2 distinct source families must emit the same side;
- zero source families may emit the opposite side;
- any mixed-direction hour is discarded;
- no weights, source deletion, 3-of-4 variant, time tolerance, or filters.

## 2023-2024 discovery

- **8,524 consensus events**.
- **3,379 mixed-direction hours discarded**.
- Source forward-return consistency check: **0 mismatches**.
- Frequency **13.6042 events/symbol/week**.
- 1h signed mean **+0.0256%**.
- 4h signed mean **+0.0554%**.
- 12h signed mean **+0.0485%**.
- Breadth **4/6 positive symbols**.
- 2023 12h **+0.0783%**.
- 2024 12h **+0.0214%**.
- LONG audit: 12h **+0.1809%**.
- SHORT audit: 12h **-0.0754%**.

Vote counts:
- LONG 2-vote 3,681; 3-vote 437; 4-vote 5.
- SHORT 2-vote 3,911; 3-vote 478; 4-vote 12.

Frequency, minimum breadth and annual sign consistency pass, but the preregistered +0.20% economic threshold fails materially. The LONG/SHORT asymmetry is post-result attribution only and cannot justify deleting SHORT.

Decision: **freeze v197**. Do not try 1-of-4/3-of-4, weighted voting, source removal, timestamp tolerance, side deletion, or additional filters. 2025 OOS remains unread; no strict Engine stage and no DB import.
