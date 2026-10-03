# v189 Frozen Native-Feature Tree — Full Temporal Validation

The exact depth-3 shared CART model trained on 2023 was kept frozen and applied forward without retraining or threshold changes.

Model SHA-256: `88289990d5a3936dfa0d11e4b5f11a8111f6b330a40a8319c7e1184b17526eaf`.

Universe: BTC / ETH / BNB / XRP / SOL / DOGE / LTC / AVAX / UNI / ZEC.

## Result

2023 is training data and therefore is not reported as validation performance.

| Validation slice | Events | 12h signed mean |
|---|---:|---:|
| 2024 | 3,747 | +0.00038% |
| 2025 | 3,996 | -0.01191% |
| 2026 Jan-Sep | 2,152 | -0.03992% |
| 2024–2026-09 aggregate | 9,895 | **-0.01335%** |

Aggregate breadth is **5/10 positive symbols**. LONG is +0.04332% while SHORT is -0.07005%; this is post-result attribution only and does not permit removing SHORT.

The original 2024 raw edge was already economically zero. The two later untouched periods turn negative, confirming that the frozen nonlinear interaction model does not contain a durable cross-cycle edge.

Decision: **v189 remains frozen**. No retraining on 2024/2025, no leaf selection, no prediction threshold, no feature pruning, no black-box rescue, no strict TP8/SL6 promotion, and no DB write.
