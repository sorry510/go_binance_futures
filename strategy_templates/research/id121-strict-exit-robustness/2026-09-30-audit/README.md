# ID121 Strict-Exit Robustness Audit — 2026-09-30

Source candidate: ID121 canonical exact-exit audit, using the frozen entry logic and exact close rules ROI >= 8 || ROI <= -6.

Canonical baseline:
- 773 trades
- PF 1.211221
- normalized net 1.420420
- 12/15 symbols positive
- 0.532684 trades/symbol/week
- yearly PF: 2024 1.241297, 2025 1.179101, 2026 1.245726
- LONG PF 1.231696; SHORT PF 1.188182

Robustness findings:
- median symbol PF: about 1.162
- leave-one-symbol-out PF remains above 1.17 except removing XRPUSDT, which lowers PF to 1.147744
- removing ETHUSDT lowers PF to 1.173490
- removing both XRPUSDT and ETHUSDT lowers PF to 1.103143
- XRP contributes about 29.8% of positive symbol net; ETH about 20.6%
- quarterly PF is positive in 8 of 9 observed quarters; 2026-Q2 is the only negative quarter at PF 0.857713
- 15 of 25 active months have positive net
- leave-one-month-out PF minimum is about 1.154, indicating no single month is solely responsible for the edge
- longest losing streak: 8 trades
- maximum additive drawdown in normalized-return units: about 0.497895, spanning roughly 2026-02-05 to 2026-08-16
- rolling 50-trade PF median 1.151, minimum 0.490
- rolling 100-trade PF median 1.217, minimum 0.664
- rolling 150-trade PF median 1.186, minimum 0.704

Block bootstrap, 10,000 deterministic resamples:
- month-block PF median 1.212, p05 1.021, probability PF>1 about 96.6%
- symbol-block PF median 1.209, p05 1.077, probability PF>1 about 99.9%

Interpretation:
ID121 remains a valid formal candidate under strict TP8/SL6. Its edge is broad enough to survive removal of any single symbol, but performance is moderately concentrated in XRP and ETH and can enter prolonged weak regimes. Treat it as a medium-thickness edge rather than a highly redundant alpha source. Bootstrap results are descriptive robustness checks, not formal significance claims.
