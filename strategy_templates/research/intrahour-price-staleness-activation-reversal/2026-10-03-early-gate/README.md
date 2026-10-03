# v193 Intrahour Price-Staleness Activation Reversal — Early Gate

Mechanism: within each fully completed UTC hour, use exactly 61 consecutive one-minute closes to form 60 one-minute close-to-close returns. Count returns whose adjacent close prices are exactly equal. Trigger only when the previous completed hour had zero such returns and the current hour has at least one. Fade the completed one-hour price direction and enter at the next complete USD-M 1h open.

No minimum positive zero-count beyond one, no rolling baseline, and no volume/taker/funding/OI/spread/ATR/time-of-day or symbol-specific filter.

## Data / structural audit

All six symbols have 1,052,700 one-minute rows and 17,544 complete formation hours, but the unconditional incidence of zero-return minutes differs strongly by symbol:
- SOL: 8,494 hours with at least one zero-return minute.
- DOGE: 15,276.
- LTC: 16,989.
- AVAX: 12,090.
- UNI: 16,498.
- ZEC: 17,424.

This large cross-symbol base-rate difference is itself evidence that the state is strongly affected by tick/price granularity.

## 2023-2024 discovery

- **8,207 events**.
- Frequency **13.0983 events/symbol/week**.
- 1h signed mean **+0.0216%**.
- 4h signed mean **+0.0261%**.
- 12h signed mean **+0.0264%**.
- Breadth **5/6 positive symbols**.
- 2023 12h **-0.0565%**.
- 2024 12h **+0.0697%**.
- LONG audit: 12h **+0.1491%**.
- SHORT audit: 12h **-0.0887%**.

Frequency and breadth are adequate, but the economic threshold fails badly and the annual sign flips. The side split is post-result attribution only.

Decision: **freeze v193**. Do not test >=2/3/5 zero minutes, 5m zero-return counts, zero-volume conditioning, side deletion, continuation, or inspect 2025+. No strict Engine run and no DB import.
