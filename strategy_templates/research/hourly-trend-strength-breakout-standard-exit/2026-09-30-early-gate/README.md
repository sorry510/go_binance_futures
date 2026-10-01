# Hourly Trend Strength Breakout — Standard Exit

Status: **FROZEN for insufficient frequency**.

Existing hourly trend-strength breakout entry logic was preserved exactly; original non-standard close rules were disabled so Engine fixed TP8/SL6 governed exits.

Core-4 produced only four trades total (one per symbol), 0.005228 trades/symbol/week. PF 1.880382 and 3/4 positive are not interpreted as alpha evidence at this sample size.

Decision: freeze without modifying ADX/DI/ATR/Donchian/Supertrend entry thresholds. No DB write.
