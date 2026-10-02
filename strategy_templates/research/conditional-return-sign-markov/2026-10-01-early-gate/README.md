# v140 Conditional Return-Sign Markov Predictor — 2026-10-01 Early Gate

Mechanism:
- Each completed 1h bar is +1 when close>open and -1 when close<open.
- Use the latest 24 fully observed sign transitions to estimate the 2x2 transition matrix.
- Given the current completed-hour sign, compute the conditional expected next sign.
- Predictor zero up-cross emits LONG; zero down-cross emits SHORT; enter next complete 1h open.

This is distinct from streak rules and unconditional return autocorrelation: the forecast is explicitly conditioned on the current state using empirical transition probabilities.

Frozen discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024.
Gate: 12h signed mean >=+0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both years positive.

## Result

33,866 events; 12h signed mean **-0.0065%**; 2/6 positive; 54.050 events/symbol/week.
2023 12h **+0.0091%**; 2024 **-0.0226%**.
LONG 12h +0.0901%, SHORT -0.1049% is audit-only and cannot justify deleting SHORT.

Decision: **early gate failed / freeze**. No transition-window scan, probability threshold, side deletion, reversal, strict TP8/SL6 Engine, or 2025+ OOS.
