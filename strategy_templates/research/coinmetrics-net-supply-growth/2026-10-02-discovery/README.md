# CoinMetrics Net Supply Growth / Dilution Acceleration — Discovery

Universe: the eight feasibility-passing mature assets BTC/ETH/XRP/ADA/BCH/LTC/DOGE/ZEC.

Signal:
- `g_now = log(S_t / S_{t-7})`;
- `g_prev = log(S_{t-7} / S_{t-14})`;
- acceleration = `g_now - g_prev`;
- zero up-cross => SHORT (dilution accelerating);
- zero down-cross => LONG (dilution decelerating);
- enter next UTC-day USD-M open.

Dynamic production eligibility remains USD-M history >=730 days and signal-day QuoteVolume >=5M USDT. No magnitude threshold or smoothing. Discovery is 2023-2024 only.

## Final result

- **946 events / 8 symbols**.
- Frequency **1.1324 events/symbol/week**.
- 1d signed mean **-0.0627%**.
- 3d signed mean **-0.1771%**.
- 7d signed mean **-0.2094%**.
- Breadth **3/8 positive symbols**.
- 2023 7d **-0.3178%**.
- 2024 7d **-0.1115%**.

The economic gate, breadth gate and both annual signs all fail.

Decision: **freeze**. Do not scan 3d/14d/30d supply windows, add acceleration thresholds, delete one direction, switch to issuance-only subsets, or reverse the rule. 2025+ remains unread; no strict Engine and no DB import.
