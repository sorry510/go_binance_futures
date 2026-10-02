# Order-Flow Coherence — 2026-10-01 Early Gate

Hypothesis: aggregate hourly taker imbalance hides whether active buying/selling was persistent or merely the residual of rapidly alternating flow. A high net/gross coherence ratio should identify genuinely one-sided aggressive flow.

Frozen definition:
- For each completed 1m bar: signed_flow = 2*TakerBuyQuoteVolume - QuoteVolume.
- Trailing 60 complete minutes: net = sum(signed_flow), gross = sum(abs(signed_flow)).
- coherence = abs(net)/gross.
- Fire only on the first cross from <=0.5 to >0.5.
- net>0 -> LONG, net<0 -> SHORT.
- No price trend, QPS, funding, volatility, or symbol-specific filter.
- Entry is next 1m open.
- Discovery only SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024; 12h endpoints may not cross into 2025.

## Result

**33,067 events**: 9,558 LONG / 23,509 SHORT. Frequency **52.7745 events/symbol/week**.

Event-weighted signed returns:
- 1h: **-0.0228%**
- 4h: **-0.0359%**
- 12h: **-0.0751%**

Breadth: **0/6** symbols positive at 12h.
- SOL -0.1006%
- DOGE -0.1048%
- LTC -0.0626%
- AVAX -0.0376%
- UNI -0.1125%
- ZEC -0.0374%

Cross-year:
- 2023: -0.0396%
- 2024: -0.1067%

Direction split is diagnostic only and cannot be used to rescue the family:
- LONG: 9,558 events, +0.1007% 12h
- SHORT: 23,509 events, -0.1466% 12h

The preregistered combined family fails economic magnitude, breadth, and sign. The strong post-hoc directional asymmetry is not permission to delete SHORT after seeing returns.

Decision: **freeze**. Do not scan 30m/120m windows or 0.4/0.6 thresholds, delete the losing direction, reverse the family, or add trend/QPS/funding filters. Strict TP8/SL6 Engine was not run; 2025+ was not read.
