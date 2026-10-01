# Protocol

1. Freeze JSON before returns.
2. Define the UTC-day opening range as the first completed 4h bar whose Open equals the current daily Open.
3. Search only completed 4h bars [1] through [5]; while the first 4h bar is still current [0], no signal is possible.
4. LONG requires a completed 1h bar to cross from inside to above opening-range high; SHORT mirrors below opening-range low.
5. Current price must break the trigger-hour high/low.
6. Exact exits are ROI >= 8 || ROI <= -6.
7. Discovery uses only SOL/DOGE/LTC/AVAX/UNI/ZEC.
8. Failure => no 1h/2h/6h opening-range variants, no filters, no reversal.
9. Pass => freeze and evaluate fresh holdout ALGO/INJ/LDO/PENDLE/PYTH.
