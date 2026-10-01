# Protocol

1. Freeze strategy JSON before returns.
2. Require two nested completed 4h inside bars: [2] inside [3], then [1] inside [2].
3. Mother range is 4h[3].
4. LONG requires the first completed 1h close outside mother high, then current price breaks that 1h high. SHORT mirrors at mother low.
5. Exact close rules are ROI >= 8 || ROI <= -6.
6. Discovery uses SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only.
7. Failure => no inside-bar count scan, no compression-ratio threshold, no filters, no reversal.
8. 2025 OOS1 and 2026 OOS2 remain unread unless prior stage passes.
