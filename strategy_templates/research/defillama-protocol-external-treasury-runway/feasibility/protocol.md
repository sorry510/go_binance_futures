# Protocol

1. Build the treasury universe mechanically from DeFiLlama catalog entries with a treasury adapter and unique token ticker.
2. Use only non-OwnTokens treasury branches to avoid own-token price mark-to-market contamination.
3. Before inspecting returns require >=8 symbols that can satisfy Binance USD-M history >=2 years during 2023-2024.
4. If coverage fails, do not supplement the universe by hand, do not inspect returns, and do not define/tune a signal.
