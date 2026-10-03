# Mandatory 2023–2026-09 Cycle Audit Backfill — v179–v188

This bundle extends the recent 2023-2024-only diagnostics to the mandatory full cycle **2023-01-01 through 2026-09-30** without changing any strategy rule.

Universe is fixed to **SOLUSDT / DOGEUSDT / LTCUSDT / AVAXUSDT / UNIUSDT / ZECUSDT**.

Canonical construction:
- archived 2023-2024 event files are preserved exactly;
- the original replay programs were copied and changed only to read 2025-01-01 through 2026-09-30;
- full-cycle summaries concatenate the untouched archived events with the later-year events;
- no threshold, direction, lookback, universe, or signal mapping was retuned after reading later years.

Data:
- v179-v186/v188 later years: Binance Vision USD-M monthly 1h Klines;
- v187 later years: Binance Vision USD-M daily 5m Futures metrics plus monthly 1h Klines;
- raw downloads are temporary cache only and are not committed.

## Full-cycle results

| Version | Mechanism | Events | 12h signed mean | Positive symbols | Positive years | 2023 | 2024 | 2025 | 2026 Jan-Sep |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| v179 | Return-Energy Concentration | 17,280 | **-0.0278%** | 1/6 | 2/4 | +0.0287% | -0.0713% | -0.1132% | +0.0792% |
| v180 | 24h Range Occupancy | 16,588 | **-0.0317%** | 2/6 | 3/4 | +0.0038% | +0.0034% | -0.1345% | +0.0114% |
| v181 | Activity–Volatility Coupling | 837 | **+0.1144%** | 4/6 | 3/4 | +0.0508% | -0.0466% | +0.1997% | +0.3733% |
| v182 | Per-Trade Volatility Impact | 14,903 | **+0.0018%** | 3/6 | 3/4 | +0.0250% | +0.0399% | -0.0829% | +0.0334% |
| v183 | 24h Price Monotonicity | 8,576 | **+0.0947%** | **6/6** | **4/4** | +0.0149% | +0.1704% | +0.0411% | +0.1784% |
| v184 | Notional-vs-Trade-Arrival Concentration | 1,453 | **-0.0584%** | 3/6 | 2/4 | -0.0728% | -0.2218% | +0.0007% | +0.2407% |
| v185 | Participation–Ticket-Size Coupling | 413 | **+0.0030%** | 3/6 | 2/4 | -0.0091% | +0.1225% | +0.1365% | -0.3857% |
| v186 | Rolling-4h Extreme Order | 48,841 | **-0.0085%** | 1/6 | 1/4 | -0.0058% | -0.0190% | -0.0082% | +0.0012% |
| v187 | Taker-vs-Global Account Skew | 30,152 | **+0.0088%** | 5/6 | **4/4** | +0.0081% | +0.0089% | +0.0104% | +0.0080% |
| v188 | Corwin–Schultz Liquidity Stress | 7,842 | **+0.0324%** | 4/6 | **4/4** | +0.0044% | +0.0628% | +0.0008% | +0.0725% |

## Interpretation

### v183 is the strongest stability result in this batch, but not economically strong enough

v183 is positive on all six symbols and in all four calendar-year slices. Per-symbol 12h means are:
- SOL +0.1390%
- DOGE +0.1115%
- LTC +0.0382%
- AVAX +0.1612%
- UNI +0.0452%
- ZEC +0.0755%

Both directions are positive in aggregate: LONG +0.0760%, SHORT +0.1133%.

However the full-cycle mean is only **+0.0947% per event**, below the preregistered **+0.20%** raw diagnostic gate. Under the project's fixed fee/slippage and 4x execution semantics this does not provide enough pre-cost margin to justify a strict TP8/SL6 Engine promotion.

### v181 is regime-dependent rather than stable

v181 rises sharply in 2025 and 2026, but 2024 is negative and only 4/6 symbols are positive over the full span. The aggregate side split is LONG -0.1313% versus SHORT +0.3514%; this is post-result attribution only. Deleting LONG or remapping the family after seeing this split is prohibited.

### v187 is consistent but economically negligible

v187 is positive in all four year slices and 5/6 symbols, but the full-cycle 12h mean is only **+0.0088%**. The stability does not compensate for essentially zero economic magnitude.

v179, v180, v182, v184, v185, and v186 fail by magnitude, breadth, or year stability. v188 is directionally consistent but still only +0.0324%.

## Decision

All ten mechanisms remain frozen. No strict TP8/SL6 Engine promotion, no strategy-template import, and no DB write.

The main useful result is methodological: extending the test window materially changes several apparent 2023-2024 readings, while v183 survives the cycle test directionally but still lacks enough edge magnitude.

No app.conf modification, commit, or push occurred.
