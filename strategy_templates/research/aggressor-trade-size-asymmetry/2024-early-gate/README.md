# Aggressor Buy/Sell Average Trade-Size Asymmetry — 2024 Early Gate

This run resolves the prior data-cost feasibility blocker with a bounded, pre-frozen 2024 gate on TRXUSDT, ATOMUSDT, LTCUSDT, and UNIUSDT. These mature liquid contracts were selected before return inspection because their Binance Vision aggTrades archives were manageable.

For each UTC hour, raw trade count per aggregate trade is reconstructed as last_trade_id - first_trade_id + 1. buyer_is_maker=false is aggressive buy and true is aggressive sell. Average notional is computed separately by side; A=log(avg_buy_notional/avg_sell_notional).

Signal: prior-720-complete-hour causal z-score of A. First z>=3 after re-arm => LONG; first z<=-3 => SHORT; re-arm at |z|<1. Entry is next 1h open.

Frozen gate: >=20 events, 12h signed mean >=+0.10%, and >=3/4 symbols positive.

Result: 288 events. 1h +0.0517%, 4h +0.0625%, 12h -0.0326%; only TRX and UNI were positive at 12h (2/4).

Decision: early gate failed. Freeze the family. Do not expand to BTC/ETH, alter z/window/direction, or consume untouched OOS.
