# DeFiLlama Chain Stablecoin Source Composition — Frozen Protocol

1. Reuse the audited chain -> native-token mapping and actual Binance USD-M first-history dates from Chain Stablecoin Supply Growth; no outcome-based chain selection.
2. Source: DeFiLlama stablecoincharts/{chain}, daily completed UTC observations.
3. Only days with both totalMintedUSD.peggedUSD > 0 and totalBridgedToUSD.peggedUSD > 0 are valid.
4. composition = log(totalBridgedToUSD.peggedUSD / totalMintedUSD.peggedUSD). This is equivalent to the log-odds of bridge-source share bridged/(bridged+minted) and avoids misuse of totalCirculating as a strict source denominator.
5. flow7 = composition_t - composition_{t-7d}, requiring continuous daily history.
6. Zero-cross upward => LONG native-token Binance USD-M; zero-cross downward => SHORT.
7. Entry = next UTC daily open.
8. Dynamic eligibility at signal: Binance USD-M actual history >=2 years and signal-day QuoteVolume >=5m USDT.
9. Discovery = 2023–2024 only; complete r7 endpoint must remain inside discovery.
10. Frozen gate: aggregate signed 7d mean >= +0.25% and >=60% of triggering symbols have positive 7d mean.
11. If discovery fails, 2025–2026 OOS remains untouched. No reversal, window change, or source-share threshold sweep.
