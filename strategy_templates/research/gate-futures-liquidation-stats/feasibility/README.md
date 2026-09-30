# Gate Futures Liquidation / OI Contract Stats — Feasibility

Mechanism: long-vs-short liquidation USD imbalance and open-interest stress on Gate could act as an external deleveraging signal for the same Binance USD-M symbol.

Gate public endpoint GET /api/v4/futures/usdt/contract_stats exposes long_liq_usd, short_liq_usd and open_interest and accepts a from timestamp.

Historical-depth audit on 2026-09-29 requested BTC_USDT and ETH_USDT from 2023-01-01 and BTC_USDT from 2024-01-01. Every request returned "from time exceeds 180-day limit".

Decision: historical OOS/discovery blocked before viewing returns. This source remains usable only for future forward collection; do not weaken the discovery/OOS protocol.
