# Protocol

1. Freeze JSON, discovery universe, and gate before returns.
2. Previous completed daily bar must be strictly inside the daily bar before it.
3. LONG acceptance: completed 4h opens at/below inside-day high and closes above it; SHORT is symmetric at the low.
4. Current price must continue beyond the 4h signal candle extreme.
5. Exact fixed exits: ROI >= 8 || ROI <= -6.
6. No trend/ADX/volume/funding/body filters.
7. Discovery SOL/DOGE/LTC/AVAX/UNI/ZEC only.
8. Gate: PF>=1.15, >=4/6 positive, >=0.30 trades/symbol/week, no yearly collapse.
9. Failure => no inside-day looseness, no 1h/8h confirmation change, no added filters, no reversal; fresh holdout untouched.
10. Pass => freeze and test ALGO/INJ/LDO/PENDLE/PYTH only.
