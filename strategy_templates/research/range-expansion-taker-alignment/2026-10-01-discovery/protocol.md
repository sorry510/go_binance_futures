# Protocol

1. Freeze strategy JSON before reading returns.
2. Trigger-hour range must exceed the arithmetic mean of the prior 8 completed 1h ranges.
3. LONG requires bullish trigger candle and TakerBuyRatio > 0.5; SHORT requires bearish trigger candle and TakerBuyRatio < 0.5.
4. Current price must break the trigger-hour high/low.
5. Exact fixed exits are TP8/SL6 via the strategy close rules.
6. Discovery only: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-01-01 to 2025-01-01.
7. Failure means do not read OOS1/OOS2 and do not change the 8h baseline, 0.5 flow center, add filters, or reverse direction.
8. Pass means freeze parameters and evaluate 2025 OOS1; only an OOS1 pass unlocks 2026 OOS2.
