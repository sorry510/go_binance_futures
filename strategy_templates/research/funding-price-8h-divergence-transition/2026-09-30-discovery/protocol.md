# Protocol

1. Freeze strategy JSON before returns.
2. 8h return is Close[1]/Close[3]-1 on completed 4h bars; prior observation is Close[2]/Close[4]-1.
3. LONG only if latest funding <0 and 8h return crosses <=0 to >0; SHORT only if funding >0 and return crosses >=0 to <0.
4. Current price must break the trigger 4h high/low.
5. Exact CLOSE rules: ROI >= 8 || ROI <= -6.
6. Discovery only SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. Promotion gate PF>=1.15, >=4/6 positive, >=0.30 trades/symbol/week, no clear multi-year instability.
8. Failure => no 4h/12h window, no funding magnitude threshold, no ADX/QPS filter, no sign reversal.
9. Fresh holdout remains unread unless discovery passes.
