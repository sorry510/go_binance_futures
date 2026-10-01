# ID121 vs v52 Precompression Attribution — 2026-10-01

Purpose: determine whether v52 contributes independent alpha or merely filters ID121.

Code inspection before returns shows:
- v52 SHORT is identical to ID121 SHORT.
- v52 LONG is ID121 LONG plus one extra fixed condition: prior 12h range width must be >=2 ATR and <3 ATR.

Therefore this is descriptive attribution, not a new candidate search.

Frozen comparison:
- Universe: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT.
- Window: 2023-01-01 through 2025-01-01.
- Both strategies use exact ROI >= 8 || ROI <= -6 exits, leverage 4, fee 0.0005/side, slippage 5bps/side, single position.
- Compare aggregate/symbol/year PF and exact entry overlap from generated trades.csv.
- Do not tune the 2-3 ATR filter and do not unlock 2025/2026 OOS from this audit.

## Final attribution

On the same six symbols and 2023-2024 window, canonical ID121 produced 211 trades at PF 1.058755, while v52 produced 151 trades at PF 1.297779. Exact common entries were 150 trades at PF 1.316104. The 61 ID121-only trades removed by v52 had PF 0.613443 and were all LONG; the 31 removed 2024 trades had PF 0.381821. Thus the fixed 2-3ATR precompression condition is removing a materially negative long cohort on this sample. However v52 still fails the preregistered frequency gate (0.240994/week) and 2023 stability (PF 0.978071), so this does not unlock OOS or justify retuning.
