# Reconstructing inputs

Raw Binance Vision ZIP archives are intentionally not stored.

Premium Index source:
- monthly USD-M premiumIndexKlines
- interval: 5m
- symbols: BTC, ETH, BNB, XRP, SOL, DOGE, LTC, AVAX, UNI, ZEC USDT perpetuals
- months required: 2023-12 through 2026-09

Price source:
- local research DB go_bn_test
- table market_klines_1h
- market futures_usdt
- 2024-01-01 through 2026-12-31

export_prices.go reconstructs the local 1h price input. replay.py downloads the Premium Index archives and recreates the study.
