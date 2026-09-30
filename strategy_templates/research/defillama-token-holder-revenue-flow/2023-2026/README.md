# DeFiLlama Token-Holder Revenue Flow — 2023–2026

Mechanism: direct economic value returned to protocol tokens via holder/staker distributions, buybacks, burns, or fee sharing. The source map was frozen before return inspection from DeFiLlama adapters exposing `HoldersRevenue`, with ticker collisions and non-holder-revenue adapters excluded before returns.

Signal: log(latest 7 completed UTC days holder revenue / preceding 7 days holder revenue), zero-cross LONG/SHORT, next UTC daily open; event-time Binance USD-M history >=2 years and signal-day QuoteVolume >=5m USDT.

Boundary audit correction: the initial discovery implementation admitted late-December 2024 signals whose 7d endpoint extended into 2025-01-01..07. Those initial discovery and OOS outputs are preserved as `legacy_boundary_leak_*` for audit only. Canonical discovery now requires every 7d endpoint to end no later than 2024-12-31.

Corrected canonical discovery: 547 events / 10 triggering symbols; 1d +0.2449%, 3d +0.2436%, 7d +1.2513%. 2023 +0.9338%, 2024 +1.5326%, but only 5/10 symbols have positive mean7, so breadth is 50% versus the frozen 60% requirement.

Decision: discovery gate fails on breadth. OOS is therefore not admissible for promotion; previously generated OOS remains legacy audit evidence only. Freeze the family before exact TP8/SL6. Do not narrow the pre-return source map, delete weak symbols, change the 7v7 window, reverse direction, or use a smaller post-hoc universe to rescue it.
