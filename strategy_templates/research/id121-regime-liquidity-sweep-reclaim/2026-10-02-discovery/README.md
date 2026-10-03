# v165 ID121-Regime Liquidity-Sweep Reclaim — Discovery

Purpose: test a genuinely different entry structure inside the frozen ID121 quality regime. All ID121 funding, daily regime, 4h trend/ADX/DI, RSI, ATR body bounds and no-chase logic were preserved. Only the fresh breakout/breakdown setup was replaced by an opposite-side 12h liquidity sweep and reclaim.

LONG: in the bullish ID121 regime, the completed 1h bar makes a strict new low below the prior 12 completed 1h lows, closes back above the prior-12h minimum, is a fresh sweep, and still satisfies the canonical bullish impulse/no-chase conditions. SHORT is the exact mirror.

Exact Engine: leverage=4, TP8, SL6, fee=0.0005/side, slippage=5bps/side, single-position, exact exits `ROI >= 8 || ROI <= -6`.

## 2023-2024 discovery

- **17 trades**, LONG 10 / SHORT 7.
- normalized PF **0.356110**.
- normalized net **-14.4503%**.
- breadth **2/6 positive symbols**.
- frequency **0.0271 trades/symbol/week**.
- 2023: 7 trades, PF **0.186859**.
- 2024: 10 trades, PF **0.504249**.
- exits: 13 stop-loss / 4 take-profit.
- per symbol: SOL 6 trades PF0.599; DOGE 1 winning trade; LTC 0; AVAX 5 all-loss sample; UNI 2 PF1.100; ZEC 3 all-loss sample.

The setup misses every promotion dimension: expectancy, breadth, frequency and cross-year stability.

Decision: **freeze v165**. Do not scan 6h/24h sweep windows, add wick/depth thresholds, remove no-chase, alter RSI/ADX/ATR, add a confirmation/retest bar, or inspect the fresh-symbol holdout.
