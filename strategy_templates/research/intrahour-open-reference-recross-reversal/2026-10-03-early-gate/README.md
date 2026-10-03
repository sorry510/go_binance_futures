# v196 Intrahour Open-Reference Recross Activation Reversal — Early Gate

Mechanism: for each fully completed UTC hour, use the prior hh:59 one-minute close as a fixed reference and follow the next 60 one-minute closes. Count sign changes in close-minus-reference across consecutive nonzero deviations. Trigger only when the previous completed hour had zero recrosses and the current hour has at least one. Fade the completed-hour final direction relative to the reference and enter at the next complete USD-M 1h open.

No minimum recross count beyond one, crossing-speed/amplitude threshold, moving reference, volume/taker/funding/OI/spread/ATR/time-of-day or symbol-specific rule.

## 2023-2024 discovery

- **14,267 events**.
- Frequency **22.7699 events/symbol/week**.
- 1h signed mean **+0.0204%**.
- 4h signed mean **+0.0608%**.
- 12h signed mean **+0.0462%**.
- Breadth **4/6 positive symbols**.
- 2023 12h **+0.0712%**.
- 2024 12h **+0.0212%**.
- LONG audit: 12h **+0.1337%**.
- SHORT audit: 12h **-0.0402%**.

Frequency and annual sign consistency are adequate, but the preregistered +0.20% economic threshold fails materially and breadth only meets the minimum. The side split is post-result attribution only.

Decision: **freeze v196**. Do not test >=2/3/5 recrosses, moving-average/VWAP references, 30m/120m windows, continuation, side deletion, additional filters, or inspect 2025+. No strict Engine run and no DB import.
