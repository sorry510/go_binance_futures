# v124 Funding Sign Flip + 12h Breakout — 2026-10-01 Discovery

Hypothesis: a settled funding-rate sign flip can mark a change in perpetual positioning pressure. Requiring a completed 1h candle to cross the prior 12 completed-hour range boundary in the same direction tests whether that positioning regime change is accompanied by sufficiently strong price displacement for the fixed TP8/SL6 objective.

LONG: latest settled funding rate > 0 and previous settled funding rate <= 0; trigger 1h crosses above the prior 12h high; current price then exceeds the trigger-hour high. SHORT is symmetric for positive-to-negative funding and a downside range cross.

This setup is fully executable by the existing project DSL. No MarketCondition, Benchmark, NowTime %, external exchange, symbol-specific threshold, or funding magnitude threshold is used. Exact exits are ROI >= 8 || ROI <= -6.

Discovery is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2024-12-31. 2025 is OOS1 and 2026 through 2026-09-01 is OOS2; neither may be read unless the previous gate passes.

Promotion gate: PF >= 1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no obvious 2023/2024 regime conflict.
