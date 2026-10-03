# ID121 no-no-chase — Fresh-Symbol Formal Validation Protocol

1. Hypothesis source is the diagnostic-only ID121 mechanism-ablation audit on its original 15-symbol cohort:
   ADA, AVAX, BNB, BTC, DOGE, ETH, LTC, NEAR, SOL, UNI, XRP, ZEC, 1000PEPE, SUI, ONDO.
2. The only allowed simplification is exactly the archived `strategies/no_no_chase.json` snapshot (SHA-256 6398522cecdddc67acc2926034f79db3bdae7c44e1ba3c950310fcfa204d632e). No other ID121 condition may change.
3. Fresh validation universe is frozen before any no-no-chase returns are read:
   LINKUSDT, BCHUSDT, ETCUSDT, TRXUSDT, XLMUSDT, AAVEUSDT.
   None participated in the source ablation cohort.
4. Validation interval is 2023-01-01T00:00:00Z through 2026-09-01T00:00:00Z, using only existing historical data. Each symbol must have established same-name USD-M history before 2023-01-01; any incomplete symbol blocks itself rather than being repaired.
5. Canonical strategy semantics remain leverage=4, fee=0.0005/side, slippage=5bps/side, single position, exact strategy closes `ROI >= 8 || ROI <= -6`.
6. Repository source must remain nil/read-only for validation. No REST gap repair/import, DB write, app.conf edit, commit or push.
7. Promotion gate: aggregate normalized PF >=1.15; >=4/6 symbols net-positive; frequency >=0.30 trades/symbol/week; and no full calendar year with material sample PF <0.90.
8. Passing this gate makes no-no-chase a candidate only; it does not replace canonical ID121 automatically. Failing freezes the simplification permanently.
9. On failure do not restore a partial no-chase condition, change funding/ADX/freshness/impulse, delete a side, remove losing fresh symbols, or reuse source-cohort results as rescue evidence.
