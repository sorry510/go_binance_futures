# v131 OI Expansion × Taker Alignment — 2026-10-01 Early Gate

## Hypothesis

When 4h open interest changes from non-expanding to expanding, the sign of aggregate taker pressure can distinguish new directional position building from a generic OI state transition. If 4h OI expansion begins while taker buy/sell pressure is net bullish, test LONG continuation; if taker pressure is net bearish, test SHORT continuation.

This is distinct from the frozen OI-price-state family: direction is determined by Binance USD-M taker flow, not by past price direction.

## Frozen early-gate sampling

To avoid downloading two full years before the mechanism shows any signal, use fixed outcome-independent quarterly blocks:
- 2023: Jan 2-4, Apr 3-5, Jul 3-5, Oct 2-4
- 2024: Jan 1-3, Apr 1-3, Jul 1-3, Oct 7-9

Symbols: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT.

Each block automatically fetches one prior calendar day for 4h lookback and one following calendar day for 12h forward coverage. Events are evaluated only on the three pre-registered target days.

## Signal

At each 5m metrics observation:
1. Compute 4h log OI change.
2. Trigger only when 4h OI change crosses from <=0 to >0.
3. Compute mean log(sum_taker_long_short_vol_ratio) over the same trailing 4h.
4. Mean log ratio >0 => LONG; <0 => SHORT. No magnitude threshold.
5. Measure signed 1h/4h/12h forward return using the same metrics price proxy sum_open_interest_value / sum_open_interest.

Early gate: overall signed 4h mean >= +0.10% and at least 4/6 symbols positive at 4h.

Failure freezes this family without changing the OI lookback, adding magnitude thresholds, reversing direction, or selecting symbols. Only an early-gate pass unlocks full 2023-2024 discovery, then exact TP8/SL6 Engine validation.

## Production mapping

Both OI and taker metrics already exist in the project Binance market-data service/API path. No other exchange is required. A production strategy DSL field would be added only after the research stages survive; no Engine or DB change is made in this gate.

## Final result

Early gate failed narrowly on economic magnitude. Across 870 pre-registered quarterly-sample events, signed mean returns were +0.0479% at 1h, +0.0786% at 4h, and +0.2470% at 12h. Four of six symbols had positive 4h mean, so breadth passed, but the frozen 4h economic gate required at least +0.10%. `passes_early_gate=false`; full 2023-2024 discovery was not evaluated. The stronger 12h result is not used to change the pre-registered gate. Freeze without OI/taker thresholds, lookback changes, direction reversal, or symbol filtering.
