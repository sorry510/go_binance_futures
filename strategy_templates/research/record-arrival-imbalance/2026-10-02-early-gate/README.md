# v160 Record-Arrival Imbalance — Early Gate

Mechanism: scan the trailing 24 completed 1h closes chronologically. Count strict new closing-price highs and strict new closing-price lows after the first seed bar. Score = up-record count - down-record count. Zero up-cross -> LONG; zero down-cross -> SHORT; entry at next 1h open.

No magnitude threshold, smoothing, return filter, funding/OI/taker/volume input or symbol-specific rule.

## 2023-2024 discovery

- **12,931 events**
- frequency **20.6377 events/symbol/week**
- 1h signed mean **+0.0236%**
- 4h signed mean **+0.0177%**
- 12h signed mean **-0.0016%**
- breadth **1/6 positive symbols**
- 2023 12h **+0.0137%**
- 2024 12h **-0.0183%**
- LONG 6,472 events: 12h **+0.0739%**
- SHORT 6,459 events: 12h **-0.0772%**

The preregistered economic and breadth gates fail and annual sign flips.

Decision: **freeze v160**. Do not scan 12h/48h windows, require record-count gaps, delete SHORT, reverse direction, or add trend/volume filters. 2025+ and strict Engine remain unread/unrun.
