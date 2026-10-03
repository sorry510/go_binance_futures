# Binance Spot Tick-Size Increase -> SHORT — 2021-2022 Untouched OOT

Purpose: validate the one-sided hypothesis generated from the 2023-2024 discovery subset, where tick-size increase -> SHORT had 7 exact trades, 6TP/1SL, PF 7.1198. This earlier 2021-2022 window was untouched: no returns had been inspected.

Official Binance archive:
- 18 tick-size notice articles.
- Body parser was updated only for older article formatting; no signal rule changed.
- 241 BASE/USDT tick-size rows reconstructed.
- **161 tick-size increase -> SHORT events / 140 unique symbols / 4 independent effective-time batches**.

Frozen production eligibility:
- same-name USD-M history >=730 days at effective time;
- prior 24 complete 1h QuoteVolume >=5M USDT;
- only then exact replay.

## Final result

**0 eligible events.**

Exclusions:
- 104 events: USD-M existed but had <730 days history.
- 57 events: no USD-M event-month history.

No post-event return, exact TP/SL path, funding, or PnL was read.

Decision: **validation unavailable / coverage blocked**. The 2023-2024 increase->SHORT subset remains an unvalidated generated hypothesis and must not be promoted. Do not lower the two-year rule, use Spot history as a substitute for USD-M history, backfill synthetic Futures history, or select later-starting symbols after outcomes.
