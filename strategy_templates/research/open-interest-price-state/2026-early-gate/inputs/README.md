# Reconstructing source inputs

Raw Binance Vision metrics archives should not be committed as canonical evidence.

Source convention:
- https://data.binance.vision/data/futures/um/daily/metrics/{SYMBOL}/{SYMBOL}-metrics-{YYYY-MM-DD}.zip
- Symbols: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT.
- Stage 1 fixed days: 2026-01-05..07, 2026-04-06..08, 2026-07-06..08.
- Stage 2: every available calendar day in 2025.
- Required columns: create_time, sum_open_interest, sum_open_interest_value.

replay.py reconstructs Stage 1. validate_2025.py reconstructs Stage 2.
