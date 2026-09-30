# Return Autocorrelation Regime
Use lag-1 autocorrelation of the latest 24 completed hourly log returns. On a crossing from non-positive to positive autocorrelation, follow the completed trailing 4h return. On a crossing from non-negative to negative autocorrelation, reverse the trailing 4h return. Zero is the only regime threshold.
Discovery 2023-2024: 14,995 events, 12h +0.02355%, 8/10 symbols positive. 2025 OOS1: -0.03076%, 3/10 positive. 2026 OOS2: -0.06671%, 3/10 positive. Full sample: -0.00862%.
Decision: freeze. The discovery effect is tiny and fails both OOS periods. No one-sided regime selection, window search or threshold tuning.
