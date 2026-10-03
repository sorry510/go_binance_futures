# v191 / v193 Mandatory 2023–2026-09 Cycle Extension

This bundle preserves the archived 2023-2024 events for v191 and v193 and evaluates the unchanged rules on 2025-01-01 through 2026-09-30.

Frozen universe:
SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT.

v191:
- each complete UTC hour uses 60 one-minute close-to-close log returns;
- sample skewness zero up-cross -> SHORT, zero down-cross -> LONG;
- next complete 1h open entry.

v193:
- count exact zero one-minute close-to-close returns within each complete UTC hour;
- previous hour zero-count == 0 and current hour zero-count > 0 activates;
- fade the current hour formation return;
- next complete 1h open entry.

Both retain signed 1h / 4h / 12h diagnostic endpoints and require the 12h endpoint to remain in the same calendar year.

Data for the extension is Binance Vision USD-M monthly 1m plus 1h Klines. No thresholds, direction mapping, minute definition, symbol rule, or endpoint is changed.

Canonical results = archived 2023-2024 events + newly evaluated 2025-2026 events. No DB write, app.conf change, commit, or push.
