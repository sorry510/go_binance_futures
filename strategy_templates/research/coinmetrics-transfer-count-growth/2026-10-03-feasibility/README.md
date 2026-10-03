# CoinMetrics Transfer-Count Growth — Feasibility Only

Candidate metric: CoinMetrics Community `TxTfrCnt`, the count of asset transfers. This is distinct from `TxCnt` (ledger transaction count) at the raw-data level.

## Coverage audit

An outcome-independent set of mature same-name Binance USD-M candidates was checked for:
- current USD-M perpetual status = TRADING;
- Community `TxTfrCnt` 1d history across 2023-2024;
- enough dates naturally satisfying >=730 days since the same-name USD-M first daily Kline.

**18 symbols pass the feasibility gate:**
BTC, ETH, XRP, ADA, DOGE, LTC, LINK, BCH, ETC, TRX, XLM, UNI, AAVE, ZEC, MANA, ALGO, SNX, COMP.

Coverage is therefore not a blocker.

## Research-family boundary

No forward return was read. Earlier CoinMetrics activity research explicitly froze further enumeration of adjacent on-chain activity metrics after TxCnt, Active Address and Holder-Base families showed discovery/OOS instability. `TxTfrCnt` is a different raw metric, but it remains an adjacent activity-count hypothesis.

Decision: **coverage passes, but discovery is intentionally not opened because it would violate the existing activity-family freeze.** Preserve the audit for future use only if a genuinely new economic hypothesis is preregistered independently of these outcomes.
