# USDC-vs-USDT Perpetual Dislocation Catch-Up

Discovery 2024-2025 passed endpoint gates: 1,154 events, 12h signed mean +0.2158%, 17/27 symbols positive; 2024 +0.4418%, 2025 +0.0525%. Untouched 2026-01..08 OOS was stronger: 578 events, 12h +0.2849%, 19/27 symbols positive.

Frozen exact 1m replay with single-position semantics materially failed after real project costs. Of 1,732 signals, 32 overlapping same-symbol signals were skipped, leaving 1,700 trades and zero data errors. Discovery exact: 1,140 trades, PF 0.9923, net -30.86%. OOS exact: 560 trades, PF 0.9177, net -151.77%. Combined PF 0.9688, net -182.64%. OOS LONG PF 0.8634 and SHORT PF 0.9610, so neither side survives.

Audit: exact-traded signals retain positive endpoint edge (discovery 12h +0.2156%, OOS +0.2864%). OOS gross after 5bps-per-side slippage but before fees has PF 1.0421 / mean +0.1294% equity, while round-trip fees at 4x average about 0.3999% equity; funding is negligible. Therefore this is a genuine transaction-cost-margin failure, not signal alignment or parser failure.

Decision: freeze / no DB import / no threshold, leverage, direction or symbol post-selection.
