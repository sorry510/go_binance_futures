# CoinMetrics SOPR / NUPL — Feasibility

Both metrics are economically distinct from prior MVRV, active-address, transaction-count and fee studies:
- SOPR uses realized profit/loss of spent outputs and has a natural break-even level of 1.
- NUPL measures unrealized profit/loss relative to market capitalization.

Access audit on 2026-10-02 used the Coin Metrics Community API asset-metrics endpoint, with PriceUSD as a control:
- PriceUSD: available.
- SOPR: HTTP 403 Forbidden.
- NUPL: HTTP 403 Forbidden.

Existing project audit already found NVTAdj / TxTfrValAdjUSD are also community-access blocked. No proxy reconstruction, paid-data assumption, or BTC-only exception is introduced.

Decision: **data-access blocked / freeze feasibility**. No signal parameters were fitted and no market returns were read.
