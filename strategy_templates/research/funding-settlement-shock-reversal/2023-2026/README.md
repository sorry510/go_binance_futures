# Extreme Funding Settlement Reversal

## Hypothesis

Extreme funding settlements may identify a crowded side in perpetual futures. If the funding rate is unusually positive, longs are crowded and the trade is SHORT; if unusually negative, the trade is LONG.

This is a standalone funding-event mechanism. It does not use v33/v54 entries, MarketCondition, Benchmark, symbol-specific parameters, or clock-phase rules.

## Discovery — 2023 to 2024

- 1,173 events.
- 1h signed mean: **+0.0677%**.
- 4h signed mean: **+0.0553%**.
- 12h signed mean: **+0.2748%**.
- 7/10 symbols had positive 4h mean.
- But annual 4h mean already flipped: 2023 **+0.1721%**, 2024 **-0.0613%**.

## OOS — 2025 to 2026

- 969 events.
- 1h signed mean: **+0.0030%**.
- 4h signed mean: **-0.0081%**.
- 12h signed mean: **-0.1255%**.
- Only 5/10 symbols had positive 4h mean.
- 2025 4h mean: **+0.0894%**.
- 2026 4h mean: **-0.1593%**.

## Decision

The discovery effect is too small for the project's trading costs and already shows a 2023/2024 regime flip. OOS removes the remaining edge and turns both 4h and 12h expectation negative.

**Freeze the entire extreme-funding-settlement reversal family.** Do not tune the 30-observation window, 2-sigma trigger, 1-sigma re-arm, entry delay, or flip the direction. Do not enter exact 1m TP8/SL6 Engine validation.

Canonical evidence is in `results/summary.json`, `results/events.csv`, and `replay.go`.
