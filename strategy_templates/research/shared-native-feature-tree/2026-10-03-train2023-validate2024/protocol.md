# v189 Shared Depth-3 Native-Feature Tree — Frozen Protocol

## Purpose
Test whether a very small, cross-symbol nonlinear interaction among already-available native features can produce a stronger path signal than the many failed one-factor rules. This is not a hyperparameter search.

## Universe and chronology
- One shared model across BTC/ETH/BNB/XRP/SOL/DOGE/LTC/AVAX/UNI/ZEC.
- Train only on 2023.
- Freeze the fitted tree and all thresholds before reading any 2024 target/strategy return.
- 2025/2026 remain unread until 2024 raw and strict gates pass.

## Features at completed hour t
All use completed USD-M 1h bars only, maximum lookback 24h:
1. ret4 = log(C_t/C_(t-4))
2. ret12 = log(C_t/C_(t-12))
3. ret24 = log(C_t/C_(t-24))
4. rv24 = sqrt(sum of squared 1h log returns over prior 24h)
5. path_eff12 = abs(ret12) / sum(abs(1h returns) over prior 12h), zero if denominator zero
6. qv4_vs20 = log(mean QuoteVolume over latest4h / mean QuoteVolume over preceding20h)
7. taker4 = mean(2*TakerBuyQuoteVolume/QuoteVolume - 1) over latest4h
8. trade4_vs20 = log(mean TradeCount latest4h / mean TradeCount preceding20h)
9. ticket4_vs20 = log((sum QuoteVolume/sum TradeCount latest4h) / (sum QuoteVolume/sum TradeCount preceding20h))

No symbol ID, calendar/time feature, benchmark, MarketCondition, funding, OI, external data or cross-symbol feature.

## Model
- self-contained Go CART regressor, standard greedy squared-error split
- max_depth=3
- min_samples_leaf=2000
- deterministic tie-break: feature order, then lower threshold
- ordinary squared-error objective
- no sample filtering except continuous/valid input
- no feature scaling needed
- target = log(C_(t+12h)/Open_(t+1h)); target endpoint must remain in 2023 for training.

No tuning of depth, leaf size, features, target horizon, split criterion or loss after outcomes. The Go implementation replaces unavailable sklearn without changing the frozen model class.

## Trading interpretation after model freeze
For validation hour t, compute frozen-tree prediction p_t.
- previous p <=0 and current p >0: LONG
- previous p >=0 and current p <0: SHORT
- otherwise no new signal
- entry = next complete 1h open
- raw validation signed return measured at +1h/+4h/+12h.

2024 raw gate:
- 12h signed mean >= +0.20%
- >=6/10 symbols positive
- >=0.30 signals/symbol/week
- 2024 H1 and H2 12h means both >0

If and only if all pass, run the unchanged 2024 events through strict project semantics: 4x, TP8, SL6, fee 0.0005/side, slippage 5bps/side, funding, single position. Strict validation must also show PF>1, >=6/10 positive symbols and not collapse frequency. Only then may 2025 OOS be read.

Failure freezes this entire tree family. Do not tune depth/min leaf, remove weak features, choose a profitable leaf, add a prediction-magnitude threshold, retain one side only, or retrain on 2024.
