# Protocol

Hypothesis: a new Upbit market-support announcement creates incremental Korean spot demand and should produce a fast positive cross-exchange price shock in the already-traded Binance USD-M perpetual.

Frozen before reading returns:
- Use Upbit official Trade-category announcements only.
- Use the original first_listed_at timestamp, not later update time.
- Include all new trading-support / new-market additions; do not filter by KRW/BTC/USDT market after seeing returns.
- Map to the corresponding Binance USD-M perpetual, including mechanical 1000-prefix mapping where required.
- At event time the Binance contract must have >=730 days of history.
- Prior 24h Binance quote volume must be >=5M USDT.
- Enter LONG at the next 1m open after announcement.
- 4x, TP8, SL6, fee 0.0005 each side, 5bps slippage each side, funding, max hold 72h.
- 2024 is discovery. Only a positive discovery may advance to 2025 OOS.
- Do not reverse to SHORT or add event/coin/market filters after seeing results.
