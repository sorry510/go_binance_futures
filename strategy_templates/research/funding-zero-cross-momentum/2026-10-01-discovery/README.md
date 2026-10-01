# v125 Funding Zero-Cross Momentum — 2026-10-01 Discovery

Hypothesis: a settled funding-rate zero-cross can mark a directional regime change in perpetual demand. Negative-to-positive funding is treated as bullish momentum; positive-to-negative funding as bearish momentum.

To avoid repeated triggering through the whole funding interval, the event is eligible only when the latest settled funding timestamp is 1h to <2h old. Therefore kline_1h[1] is the first fully completed hour after settlement. That hour must align with the new funding sign, and current price must break its extreme.

This uses NowTime only as an event-age difference; there is no NowTime modulo or clock-time window.

Exact exits: ROI >= 8 || ROI <= -6. Discovery only: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2025-01-01.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, with no clear year instability. Failure means no threshold tuning, no sign reversal into a crowding trade, and 2025/2026 OOS remain unread.

## Final result

Discovery failed decisively: 369 trades, PF 0.717792, 0/6 symbols positive, 0.588919 trades/symbol/week. 2023 PF 0.520228; 2024 PF 0.932157. 2025 OOS1 and 2026 OOS2 were not evaluated. Freeze without funding-magnitude tuning, opposite-direction reinterpretation, or event-age-window changes.

## Final result

Discovery failed decisively: 369 trades, PF 0.717792, 0/6 symbols positive, 0.588919 trades/symbol/week. 2023 PF 0.520228; 2024 PF 0.932157. 2025 OOS1 and 2026 OOS2 were not evaluated. Freeze without funding-threshold tuning, event-window tuning, or sign reversal.
