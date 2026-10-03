# v189 Shared Depth-3 Native-Feature Tree — 2023 Train / 2024 Validation

A deliberately tiny, fully shared nonlinear model was used to test whether interactions among existing project-native features could recover stronger alpha than the many failed one-factor rules.

## Frozen design

- One model for BTC/ETH/BNB/XRP/SOL/DOGE/LTC/AVAX/UNI/ZEC.
- No symbol ID, calendar/time feature, MarketCondition, Benchmark or cross-symbol input.
- Native 1h features only: ret4/12/24, rv24, 12h path efficiency, 4h-vs-prior20h QuoteVolume, 4h taker flow, TradeCount and average-ticket ratios.
- Self-contained Go CART squared-error regressor, max depth 3, min leaf 2000; no hyperparameter search.
- 2023 only used for training; model hash was frozen before any 2024 strategy return was read.
- Model SHA-256: `88289990d5a3936dfa0d11e4b5f11a8111f6b330a40a8319c7e1184b17526eaf`.

Training: **87,480 samples**, exactly 8,748 per symbol. The fitted tree primarily used rv24, taker4, ret24, qv4_vs20 and ret4. Only one terminal leaf had a negative prediction.

## Untouched 2024 validation

Signals are only fresh zero-crosses of the frozen tree prediction.

- **3,747 events**.
- Frequency **7.1664 events/symbol/week**.
- 1h signed mean **+0.0061%**.
- 4h signed mean **-0.0046%**.
- 12h signed mean **+0.00038%** (economically zero).
- Breadth **7/10 positive symbols**, but most symbol means are only a few bp or less.
- H1 12h **-0.00039%**.
- H2 12h **+0.00107%**.

The preregistered +0.20% raw economic gate fails by orders of magnitude and H1 is not positive.

Decision: **freeze v189 before strict TP8/SL6 and before 2025/2026 OOS**. Do not tune depth/min-leaf, remove features, pick individual profitable leaves, add a prediction-magnitude threshold, keep only one side, retrain on 2024 or use a stronger black-box model as a rescue.
