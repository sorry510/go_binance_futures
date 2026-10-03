# Mandatory 2023–2026-09 Cycle Audit Backfill — v190 / v192 / v194 / v196 / v197

This audit implements the global research-protocol change made on 2026-10-03: every sufficiently covered mechanism must be evaluated over the full **2023-01-01 through 2026-09-30** span and reported by calendar year.

No strategy parameter, direction, threshold, window, symbol rule or execution rule was changed.

## Data and parity audit

The backfill independently downloaded **552 Binance Vision USD-M monthly Kline archives**:
- 6 symbols;
- 2022-12 through 2026-09;
- both 1h and 1m.

2023-2024 reproduction:
- v190: exact archived event-key parity after restoring its original rule excluding 1h bars with non-positive QuoteVolume/TradeCount.
- v194: exact archived event-key parity.
- v192: all 23,871 archived events reproduced; Binance Vision additionally exposed 5 valid events.
- v196: all 14,267 archived events reproduced; Binance Vision additionally exposed 2 valid events.

The v192/v196 extras were audited rather than silently accepted. Their cause is a missing local `market_klines_1h` row at **2024-02-13 00:00 UTC** for DOGE/LTC/AVAX/UNI/ZEC. Local 1m history is complete and identical to Binance Vision around the incident. Aggregating the 60 official 1m bars reproduces the official Binance Vision 1h OHLC/volume/trade-count row exactly.

For the canonical cycle summary below, **strict-parity mode** preserves the original archived 2023-2024 events exactly and appends only newly evaluated 2025-2026 events. v197 reproduces its original 8,524 discovery event keys exactly before adding later years. The full Vision coverage-corrected variant is preserved separately in `results/summary.json`.

## Strict-parity full-cycle results

| Family | Events | Full 12h mean | Positive symbols | Positive years | 2023 | 2024 | 2025 | 2026 Jan-Sep |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| v190 Trading-Invariant Stress Reversal | 7,090 | +0.0373% | 4/6 | 3/4 | +0.1331% | +0.1058% | +0.0051% | **-0.1457%** |
| v192 Intrahour Volatility-Signature | 44,968 | +0.0451% | **6/6** | 3/4 | +0.0424% | +0.1174% | +0.0159% | **-0.0087%** |
| v194 3-Bar FVG Fill Reversal | 35,217 | **-0.0362%** | 0/6 | 2/4 | +0.0415% | +0.0453% | **-0.1416%** | **-0.1006%** |
| v196 Open-Reference Recross Reversal | 27,494 | +0.0023% | 4/6 | 2/4 | +0.0712% | +0.0212% | **-0.0505%** | **-0.0380%** |
| v197 Fixed 2-of-4 Consensus | 16,717 | **-0.0044%** | 3/6 | 2/4 | +0.0783% | +0.0214% | **-0.0235%** | **-0.1095%** |

## Interpretation

The full-cycle audit materially changes the reading of the 2023-2024 screen:

- v194, v196 and v197 looked directionally positive in both 2023 and 2024 but reverse in both 2025 and 2026.
- v190 decays from +0.13%/+0.11% in 2023/2024 to approximately flat in 2025 and materially negative in 2026.
- v192 is the most stable of the group and remains positive across all six symbols in the aggregate, but its 12h effect is only +0.045% over the full period and becomes approximately zero/slightly negative in 2026.
- None approaches the required economic magnitude, and the later-year decay confirms substantial regime dependence.

These are forward-return diagnostic studies, not strict TP8/SL6 production candidates. No family is promoted.

## Decision

All five families remain frozen. The important methodological change is permanent: **future research may no longer stop after a 2023-2024 failure or success. 2025 and 2026 Jan-Sep are mandatory cycle diagnostics.**

No DB write, app.conf change, commit or push occurred.
