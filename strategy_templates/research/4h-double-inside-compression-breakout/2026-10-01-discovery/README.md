# v130 4h Double-Inside Compression Breakout — 2026-10-01 Discovery

Hypothesis: two nested completed 4h inside bars compress price inside a mother bar; the first completed 1h close outside the mother range marks structural release, and current price must then break that 1h extreme.

Definition:
- 4h[2] fully inside 4h[3].
- 4h[1] fully inside 4h[2].
- LONG: 1h[2] high <= mother high and 1h[1] close > mother high, then current price > 1h[1] high.
- SHORT is symmetric.

Exact exits: ROI >= 8 || ROI <= -6. No EMA/ADX/volume/funding/taker filters and no parameter scan.

Discovery only: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT, 2023-01-01 through 2025-01-01.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear 2023/2024 instability. Failure means no 1/3-inside variant, no range-ratio threshold, and OOS remains unread.

## Final result

Discovery failed decisively: 49 trades, PF 0.441123, 0/6 symbols positive, 0.078203 trades/symbol/week. 2023 PF 0.478528; 2024 PF 0.407032. OOS was not evaluated. Freeze without changing inside-bar count, compression definition, adding filters, or reversing the family.
