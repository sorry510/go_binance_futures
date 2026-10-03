# v165 ID121-Regime Liquidity-Sweep Reclaim — Discovery Protocol

1. Purpose: test a genuinely different entry mechanism inside the already-frozen ID121 quality regime, not retune ID121 breakout thresholds.
2. Preserve ID121 funding logic, daily regime, 4h EMA/ADX/DI strength, RSI thresholds, ATR body bounds, and no-chase logic exactly.
3. Replace only the fresh breakout/breakdown setup with an opposite-side liquidity sweep and reclaim:
   - LONG: completed 1h bar makes a strict new low below the previous 12 completed 1h lows, but closes back above the previous-12h minimum.
   - SHORT: completed 1h bar makes a strict new high above the previous 12 completed 1h highs, but closes back below the previous-12h maximum.
4. Require freshness: the immediately preceding completed 1h bar must not already satisfy the same sweep/reclaim condition.
5. The sweep/reclaim bar must itself satisfy the canonical directional impulse condition: LONG bullish body + RSI>=55 + canonical ADX acceleration/body bounds; SHORT bearish body + RSI<=45 + canonical body bounds.
6. Current execution price must still satisfy the canonical no-chase window around the sweep bar close.
7. No wick-ratio threshold, volume/taker/QPS condition, sweep-depth threshold, retest delay, extra confirmation bar, time-of-day rule, or symbol-specific rule.
8. Exact Engine exits remain `ROI >= 8 || ROI <= -6`; leverage=4, fee=0.0005/side, slippage=5bps/side, single-position.
9. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only. Gate: PF>=1.15, >=4/6 positive symbols, >=0.30 trades/symbol/week, and both 2023/2024 PF>1.
10. Failure freezes this setup: do not scan 6h/24h sweep windows, add sweep-depth/wick thresholds, remove no-chase, change RSI/ADX/ATR, or add filters.
11. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless the frozen discovery gate passes.
