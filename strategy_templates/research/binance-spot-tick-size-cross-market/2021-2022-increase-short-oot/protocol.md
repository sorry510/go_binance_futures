# 2021-2022 Untouched OOT Protocol — Tick-Size Increase -> SHORT

1. This stage validates only the generated 2023-2024 discovery hypothesis: Spot BASE/USDT tick-size increase -> SHORT USD-M. No decrease/LONG outcome is inspected.
2. The 18 official 2021-2022 Binance tick-size notice articles were archived before any 2021-2022 return was inspected.
3. Parse each official article body mechanically. Use only BASE/USDT rows and the article's explicit effective UTC timestamp; exclude leveraged UP/DOWN tokens.
4. Before any return, freeze the complete increase-event set and require same-name USD-M history >=730 days at effective time plus prior 24 complete 1h QuoteVolume >=5M USDT.
5. No minimum event count is imposed after seeing the archive; report coverage exactly. If zero eligible increase events exist, validation is unavailable rather than failed alpha.
6. Exact replay, if eligible events exist: next 1m open after effective time; 4x leverage; TP8/SL6; fee 0.0005/side; slippage 5bps/side; funding included; max hold 72h; single-position semantics per event.
7. Validation passes only if PF>1, aggregate net>0, and >=60% independent effective-time batches have positive mean net. This is intentionally weaker than a fresh discovery gate because the hypothesis was generated from only 7 discovery trades; no parameter tuning is allowed.
8. Failure freezes increase->SHORT permanently. Do not alter direction, TP/SL, eligibility, effective time, or select symbols/batches.
