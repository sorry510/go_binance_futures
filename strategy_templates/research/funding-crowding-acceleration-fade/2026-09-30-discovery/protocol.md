# Protocol

1. Freeze JSON before reading returns.
2. FundingRate.Data[0] is newest. LONG crowd condition: Data[0] < Data[1] < Data[2] < 0. SHORT crowd condition is symmetric above zero.
3. Price must first reject the crowd on a completed 4h candle, then current price breaks that 4h extreme.
4. Exact CLOSE rules: ROI >= 8 || ROI <= -6.
5. Discovery only SOL/DOGE/LTC/AVAX/UNI/ZEC.
6. Promotion gate PF>=1.15, >=4/6 positive, >=0.30 trades/symbol/week, no clear multi-year instability.
7. Failure => no 2/4-settlement variant, no absolute funding threshold, no added trend/QPS/taker filter, no direction reversal.
8. Fresh holdout stays unread unless discovery passes.
