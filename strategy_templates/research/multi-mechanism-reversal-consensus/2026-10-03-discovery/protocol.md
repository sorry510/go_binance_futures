# v197 Multi-Mechanism Reversal Consensus — Protocol

1. This is a meta-hypothesis trained on already-read 2023-2024 discovery evidence. It is NOT independent validation.
2. Freeze source families exactly to v190, v192, v194 and v196. Do not add/remove a family after combined returns are read.
3. Universe remains SOL/DOGE/LTC/AVAX/UNI/ZEC; discovery remains 2023-2024.
4. For each symbol and completed signal hour, collect source-family events.
5. Emit LONG only if at least two distinct source families emit LONG and zero emit SHORT at that same hour.
6. Emit SHORT only if at least two distinct source families emit SHORT and zero emit LONG at that same hour.
7. Any mixed-direction hour is discarded rather than net-voted. Duplicate source events are impossible by construction and would be deduplicated by family.
8. Entry/forward-return semantics are inherited unchanged: next complete USD-M 1h open and signed 1h/4h/12h closes.
9. No family weights, score magnitudes, 3-of-4 alternative, one-hour tolerance window, symbol-specific rules, time-of-day, funding/OI filters, or source-side deletion.
10. Discovery gate: 12h signed mean >= +0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and both 2023 and 2024 means >0.
11. If discovery fails, freeze. Do not try 1-of-4/3-of-4, weighted voting, source removal, or timestamp tolerance.
12. If discovery passes, freeze this exact rule first and only then compute fresh 2025 OOS signals from the underlying source formulas. 2026 remains unread until 2025 passes.
