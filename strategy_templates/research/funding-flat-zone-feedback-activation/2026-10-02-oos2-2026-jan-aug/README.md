# v154 Funding Flat-Zone Feedback Activation — 2026 OOS2

OOS2 reused the discovery/OOS1 rule unchanged. Period was frozen to 2026-01-01 through 2026-09-01 because on 2026-10-02 Binance Vision's September 2026 fundingRate monthly archive existed but the September 1h kline monthly archive still returned 404; no partial-month or REST substitute was mixed into the test.

## OOS2 result

- **377 events**
- 1h signed mean **-0.0008%**
- 4h signed mean **-0.0856%**
- 12h signed mean **-0.1872%**
- **0/6 symbols positive**
- frequency **1.8100 events/symbol/week**
- LONG 375 events: **-0.1789%**
- SHORT 2 events: **-1.7317%**

The preregistered OOS2 gate fails decisively. The failure is not caused only by the tiny SHORT subset: LONG itself is negative.

Across discovery + OOS1 + OOS2 there are 2,489 events with event-weighted 12h mean about +0.1535%, but the chronological 2026 hard failure invalidates promotion.

Decision: **freeze v154**. Do not change the flat-state equality, add magnitude/premium/OI filters, delete a side, alter entry/horizon, or run strict TP8/SL6 after the OOS2 failure. No DB import.
