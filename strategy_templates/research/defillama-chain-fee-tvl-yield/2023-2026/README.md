# DeFiLlama Chain Fee Yield / Capital Productivity

Mechanism: weekly chain user fees relative to average chain TVL measure fee-generating productivity per unit of locked capital. Rising productivity is fixed LONG the native token; falling productivity is fixed SHORT.

Signal was frozen before returns: compute latest 7 completed UTC days fee sum / same 7-day average TVL and compare with the preceding 7-day window. Zero-cross upward LONG, downward SHORT. Entry is next UTC daily open. Eligibility requires >=2 years Binance USD-M history and signal-day QuoteVolume >=5m USDT.

Discovery partition is 2023-2024 and the full 7d endpoint must remain inside it. Frozen gate: aggregate 7d signed mean >=+0.25%, >=60% symbols positive, and both 2023 and 2024 aggregate mean7 >0.

Discovery passed strongly: 780 events / 12 symbols, mean7 +0.8336%, 10/12 positive. 2023 +1.3448%; 2024 +0.3841%.

Untouched OOS weakened materially. 2025: 588 events / 13 symbols, mean7 +0.1465%, 8/13 positive. 2026: 415 events / 13 symbols, mean7 +0.1468%, only 7/13 positive.

Decision: freeze before exact replay. OOS retains the sign but fails economic strength and 2026 breadth. Do not tune the 7v7 window, threshold, direction, or drop weak symbols to rescue the family.
