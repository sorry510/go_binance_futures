# Protocol

1. Chain-to-native-token mapping and >=2y futures eligibility are reused from the pre-existing audited Chain Stablecoin Supply Growth universe.
2. Source: DeFiLlama stablecoincharts/{chain}.
3. implicit_peg = totalCirculatingUSD.peggedUSD / totalCirculating.peggedUSD.
4. stress = abs(log(implicit_peg)).
5. stress_flow_7v7 = log(mean latest 7 completed days stress / mean preceding 7 completed days stress).
6. Zero-cross upward => SHORT; zero-cross downward => LONG.
7. Entry: next UTC daily open.
8. Signal-day futures QuoteVolume >= 5m USDT.
9. Discovery: 2023-2024 only, with the complete 7d endpoint also inside those years.
10. Gate: mean7 >= +0.25% and >=60% symbols positive.
11. If discovery fails, do not compute 2025-2026 OOS or exact TP8/SL6.
