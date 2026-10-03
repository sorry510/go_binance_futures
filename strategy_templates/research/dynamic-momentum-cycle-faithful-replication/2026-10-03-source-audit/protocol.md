# Protocol

1. This audit is source-identification only; it must not read Binance forward returns.
2. A faithful implementation requires a unique deterministic mapping from price observations + kappa to confirmed turning points.
3. Descriptions such as "moving-average smoothing filter" or figures showing resulting turning points are insufficient.
4. If the exact filter is unavailable, stop before implementation. Do not replace it with SMA slope flips, centered extrema, ZigZag thresholds, or another turning-point detector.
5. If the primary algorithm is later recovered, freeze its exact causal implementation and kappa=5 before reading Binance discovery returns.
