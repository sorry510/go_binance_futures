# Protocol

1. Freeze strategy JSON before returns.
2. Use only completed 1h bars.
3. Compute four-hour upside excursion as sum(High-Open) and downside excursion as sum(Open-Low).
4. LONG requires ratio up/down crossing from <=1 to >1; SHORT mirrors from >=1 to <1.
5. Current price must break the most recent completed 1h high/low.
6. Exact exit rules are ROI >= 8 || ROI <= -6.
7. Discovery uses only SOL/DOGE/LTC/AVAX/UNI/ZEC.
8. Failure means no 2h/8h window scan, no threshold change, no added filters, and no reversal.
9. Pass means freeze parameters and evaluate fresh holdout ALGO/INJ/LDO/PENDLE/PYTH.
