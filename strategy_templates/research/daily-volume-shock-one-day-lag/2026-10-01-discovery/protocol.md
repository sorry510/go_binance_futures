# Protocol

1. Freeze strategy JSON before reading returns.
2. Identify a fresh daily quote-volume expansion on completed day [2]: Amount[2] above the preceding 20-day mean and Amount[3] not above its own preceding 20-day mean.
3. Direction is the candle sign of the shock day [2].
4. Day [1] is a full intervening day and is not used for tuning or direction.
5. On the current day, require a completed 4h candle aligned with the shock direction and a subsequent break of its extreme.
6. Exact exits are ROI >= 8 || ROI <= -6.
7. Discovery uses only SOL/DOGE/LTC/AVAX/UNI/ZEC.
8. Failure means no 2d/3d lag scan, no alternate volume baseline, no filters, and no reversal.
9. Holdout remains unread unless discovery passes.
