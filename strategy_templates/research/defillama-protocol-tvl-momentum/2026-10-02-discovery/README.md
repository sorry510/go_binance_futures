# DeFiLlama Protocol TVL Momentum — 2026-10-02 Discovery

Hypothesis: persistent protocol-level capital inflow/outflow can carry information distinct from isolated TVL shock events. The frozen signal is seven-day protocol TVL log-growth crossing zero: up-cross LONG, down-cross SHORT, next UTC-day Binance USD-M open.

Universe is copied unchanged from the prior protocol-TVL-shock audit. Dynamic eligibility is USD-M history >=730 days and signal-day QuoteVolume >=5M USDT. Discovery is 2023-2024 only; 2025/2026 remain unevaluated.

A first loader using current Binance REST was invalidated because delisted historical contracts returned zero/400 (for example CVX/SXP/BTCST), creating current-survivorship bias. That preflight output is preserved under `results/invalid_preflight_rest/` and is not decision-valid. The corrected run uses Binance Vision monthly USD-M 1d archives with the signal, universe, and gate unchanged. A transient Vision SSL preflight was also aborted before outcome aggregation and retained as `results/preflight_vision_ssl.log`.

## Final result

Corrected Binance Vision discovery:
- 1,479 eligible events across 17 actually triggered symbols.
- Frequency: **0.8331 events/symbol/week**.
- 1d signed mean: **-0.0640%**.
- 3d signed mean: **-0.4467%**.
- 7d signed mean: **-0.7793%**.
- Breadth: **4/17 positive symbols**.
- 2023 7d: **-0.4588%**.
- 2024 7d: **-1.0865%**.
- LONG: 735 events, 7d **-0.8306%**.
- SHORT: 744 events, 7d **-0.7285%**.

The mechanism fails the preregistered economic, breadth, and annual-sign gates. Both directions are negative, so there is no legitimate one-sided rescue.

Decision: **freeze**. Do not scan 3d/14d/30d TVL windows, add TVL-size/category filters, delete a direction, residualize by price, reverse the rule, or inspect 2025/2026 for redesign. No strict Engine candidate and no DB write.
