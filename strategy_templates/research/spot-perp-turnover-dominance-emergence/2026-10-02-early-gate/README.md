# v151 Spot-Turnover Dominance Emergence — 2026-10-02 Early Gate

Hypothesis: when trailing-24h Binance Spot QuoteVolume newly exceeds same-symbol USD-M perpetual QuoteVolume, spot activity has become the dominant venue; the sign of the same 24h spot return would define LONG/SHORT continuation.

Frozen definition:
- 24 aligned completed 1h Spot and USD-M bars.
- ratio = Spot QuoteVolume24 / Perp QuoteVolume24.
- trigger only on ratio cross from <=1 to >1.
- direction = sign of same-window Spot 24h log return.
- next USD-M 1h open; intended 1h/4h/12h early gate.
- no z-score, threshold relaxation, reverse-cross signal, or other filter.

## Final result

Binance Vision coverage is complete for all six symbols over the requested monthly range:
- Spot rows: 18,287 per symbol.
- Perp rows: 18,288 per symbol.
- Missing months: none.

Eligible signals in 2023-2024:
- SOL 0
- DOGE 0
- LTC 0
- AVAX 0
- UNI 0
- ZEC 0
- **Total: 0 events; frequency 0.0.**

This is a structural coverage failure, not a return failure: on this liquid USD-M universe, trailing-24h Spot QuoteVolume never crossed from <= Perp to > Perp during discovery.

Decision: **freeze v151**. Do not lower the natural 1.0 dominance boundary, scan alternate turnover windows, reinterpret the reverse cross as directional, or inspect 2025+ to manufacture events. No returns were available to evaluate, no strict Engine candidate, no DB write.
