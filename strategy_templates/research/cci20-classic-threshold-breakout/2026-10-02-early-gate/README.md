# v156 CCI(20) Classic Threshold Breakout — Early Gate

Project-native standalone CCI test. CCI is calculated identically to the project's CalculateCCI implementation. A completed 1h CCI crossing from <=+100 to >+100 emits LONG; crossing from >=-100 to <-100 emits SHORT. Entry is the next 1h open.

Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only. No price/trend/funding/taker/QPS filter.

## Final result

- **9,990 events**, LONG 4,997 / SHORT 4,993.
- Frequency **15.9439 events/symbol/week**.
- 1h signed mean **+0.0029%**.
- 4h signed mean **-0.0261%**.
- 12h signed mean **-0.0312%**.
- Breadth **2/6 positive**.
- 2023 12h **-0.0580%**.
- 2024 12h **-0.0048%**.
- LONG 12h **+0.0603%**; SHORT 12h **-0.1227%** (audit only; no post-hoc side deletion).

Decision: **freeze v156 / entire CCI family**. Do not scan periods or +/-50 / +/-200 thresholds, switch to threshold re-entry/mean-reversion, delete a side, reverse, or add filters. 2025+ remains unread; no strict Engine strategy and no DB import.
