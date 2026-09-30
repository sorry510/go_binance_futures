# Protocol

- Source: Binance Vision Spot 1h klines for trade_count; local go_bn_test market_klines_1h for USD-M futures trade_count and prices.
- Core symbols: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT.
- Discovery-only early gate: 2023-01-01 through 2024-12-31.
- Feature: log(perp trade_count / spot trade_count).
- Causal standardization: prior 720 aligned hours only.
- Trigger: first |z| >= 3, re-arm only after |z| < 1.
- Direction: perp dominance z>3 => SHORT; spot dominance z<-3 => LONG.
- Entry: next 1h open; diagnostics at 1h, 4h, 12h.
- Gate: 12h signed mean >= +0.10% and >=3/4 symbols positive.
- No OOS or exact TP8/SL6 replay if the early gate fails.
