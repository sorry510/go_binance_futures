# Protocol

1. Freeze strategy before returns.
2. r1=Close[1]/Close[2]-1, r2=Close[2]/Close[3]-1, r3=Close[3]/Close[4]-1.
3. LONG requires r1>r2>r3>0; SHORT requires r1<r2<r3<0.
4. Current price must break the newest completed 1h high/low.
5. Exact close rules are ROI >= 8 || ROI <= -6.
6. Discovery only: SOL/DOGE/LTC/AVAX/UNI/ZEC in 2023-2024.
7. Failure => no return-count or magnitude tuning and no added filters.
8. 2025 OOS1 and 2026 OOS2 remain unread unless prior stage passes.
