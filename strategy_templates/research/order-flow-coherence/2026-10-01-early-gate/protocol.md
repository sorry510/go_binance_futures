# Protocol

1. Freeze SOL/DOGE/LTC/AVAX/UNI/ZEC and discovery=2023-2024 before reading returns.
2. Use only completed project-local USD-M 1m Klines. Repository source=nil; no remote fetch/import/DB write.
3. For each completed minute m, signed_flow_m = 2*TakerBuyQuoteVolume_m - QuoteVolume_m.
4. Over the trailing 60 completed minutes, net = sum(signed_flow), gross = sum(abs(signed_flow)), coherence = abs(net)/gross.
5. A signal fires only when coherence crosses from <=0.5 to >0.5. Threshold 0.5 and 60m window are frozen before returns and are not scanned.
6. Direction is LONG if net>0, SHORT if net<0. No price trend, QPS, funding, volatility, or symbol-specific filter.
7. Entry is the next 1m open. Measure causal signed 1h/4h/12h forward returns.
8. 2024-end signals whose 12h endpoint would cross into 2025 are excluded; 2025+ remains untouched.
9. Early gate: 12h signed mean >=+0.20%, >=4/6 symbol means positive, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: no 30m/120m window scan, no 0.4/0.6 threshold scan, no price/taker-volume magnitude filter, no direction reversal.
11. Strict TP8/SL6 Engine is run only after early-gate promotion.
