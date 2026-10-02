# Funding Sign Persistence Reversal — 2026-10-01 Early Gate

Hypothesis: persistent one-sided funding over a full normal 24h cycle indicates crowded leveraged positioning. The first time three consecutive normal ~8h settlements share the same sign, trade against the crowd: positive funding streak -> SHORT, negative funding streak -> LONG.

Frozen mechanics:
- Only regular funding cadence is accepted: adjacent settlement gaps must be 7.5h..8.5h.
- Three consecutive same-sign settlements define the event; the immediately preceding settlement must not extend the same regular-cadence streak.
- Zero funding breaks the streak.
- Funding magnitude is ignored.
- Three settlements is fixed as one normal 24h cycle; no streak-length search.
- Entry is the next full 1h open after the third settlement.
- 2024 year-end events whose 12h diagnostic endpoint would enter 2025 are excluded, preserving untouched OOS.
- Discovery cohort: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only.

## Result

580 events; 143 LONG / 437 SHORT. Frequency **0.925673 events/symbol/week**.

Event-weighted signed returns:
- 1h: **-0.0131%**
- 4h: **+0.0424%**
- 12h: **+0.0168%**

Breadth: **3/6** symbols positive at 12h.
- SOL +0.2727%
- DOGE +0.3449%
- LTC -0.6631%
- AVAX -0.0786%
- UNI +0.2960%
- ZEC -0.3830%

Cross-year:
- 2023: **-0.0253%**
- 2024: **+0.0697%**

By direction (diagnostic only; not used to rescue the family):
- positive-funding streak -> SHORT: 437 events, 12h +0.0084%
- negative-funding streak -> LONG: 143 events, 12h +0.0424%

The preregistered early gate required >=+0.20% 12h mean, >=4/6 positive symbols, >=0.30 events/symbol/week, and both years positive. Frequency passes, but economic magnitude, breadth, and cross-year stability fail.

Decision: **freeze**. Do not scan 2/4/5-settlement streaks, add funding magnitude thresholds, split/revive one direction, add trend/taker/OI filters, or reverse to continuation. Strict TP8/SL6 Engine was not run; 2025+ OOS was not read.
