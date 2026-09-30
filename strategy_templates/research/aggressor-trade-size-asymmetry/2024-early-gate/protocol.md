# Aggressor Buy/Sell Average Trade-Size Asymmetry — Frozen 2024 Early Gate

Mechanism: unusually large buyer-initiated trades relative to seller-initiated trades may identify informed/large aggressor pressure not visible in aggregate taker volume.

Universe is frozen before return inspection: TRXUSDT, ATOMUSDT, LTCUSDT, UNIUSDT. They are mature USD-M contracts and were selected only for manageable 2024 Binance Vision aggTrades archive size among established liquid contracts.

For each UTC hour, reconstruct raw-trade count per aggTrade as last_trade_id - first_trade_id + 1. buyer_is_maker=false is aggressive buy; true is aggressive sell. Compute average notional on each side and A=log(avg_buy_notional/avg_sell_notional).

Signal uses a prior-720-complete-hour causal z-score of A. First z>=3 after re-arm => LONG; first z<=-3 => SHORT; re-arm only after |z|<1. Entry is next 1h open.

Eligibility at signal: contract history >=2 years and trailing completed 24h USD-M QuoteVolume >=5m USDT. Evaluate signed 1h/4h/12h endpoint returns.

2024 early gate: >=20 events, 12h signed mean >=+0.10%, and >=3/4 symbols with positive 12h mean. Only a pass permits broader-symbol expansion and untouched OOS. No threshold/window/direction changes after results.
