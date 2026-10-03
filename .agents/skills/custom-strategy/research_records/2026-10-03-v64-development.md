# v64 complementary range-entry experiment

Status: development hypothesis, not a validated release. All full candidates and diagnostic controls are retained under `temp_strategy/v64`; no template is written or activated.

## Frozen design before replay

- Base: `temp_strategy/v29/03-remove-acceleration.json` (v29C). Preserve its existing long/short trend entries and signal-confirmed trend exits.
- Failure motivating the study: v62 reached the frequency target in the original development universe but lost money in five of the first six cross-coin runs. The high-frequency breakout supplement did not establish a general edge. Those results are not a basis for assuming an inverse strategy is profitable.
- Complement: a closed 1h candle sweeps a previous 20h Donchian boundary and reclaims the range with favorable candle position, turnover/activity, taker direction, RSI recovery, and weak/non-accelerating 4h ADX. Use forming `[0]` only for no-chase entry and price-confirmed exits. No new indicator implementation is required.
- v64a: trend + range entries, exits bound to the exact range opening rule hash (`OpenStrategyHash`). Range positions close on price-confirmed mean reversion or a confirmed range failure; trend positions retain v29C exits. Current daily regime must not silently change the entry family's risk model.
- v64b: one logical change from v64a: forbid the range entry against a strong daily EMA/DI trend. All other prices, costs, exits, and parameters are identical.
- v64a portable SHA-256: `a935f49187f37cddc6158de3cc34b94a52a77d18154c3c9c1df1c0240ad064be`.
- v64b portable SHA-256: `1f8466488e054944d5819104b341bd7720c429f1939800666412872870d97238`.
- Development universe: BTCUSDT, ETHUSDT, SOLUSDT, XRPUSDT, all already inspected. ADA/AVAX/BNB/DOGE/LTC/NEAR and any subsequent inspected v62 validation coins are not untouched holdouts for v64.
- UTC period: 2022-09-01..2026-08-31, four complete Sep–Aug years, actual standard 1m project engine.
- Costs/risk: initial equity 1,000 USDT per coin; 10% of current available cash as margin, 8x leverage, +/-5% outer ROI gates, 0.05% fees and 5bps slippage per side, actual historical funding. No ledger-only counterfactual returns.
- Source: checksum-verified public execution and indicator archives, `-minute-repair verified-archive`, correction version `20261003-v2`, independent v2 cache/output. ARM supplies templates, read-only independent repair references, and funding. Neither `conf/app.conf` nor any database table is modified.
- Gates unchanged: every coin >=0.9 trades/full-calendar week, positive cost-adjusted four-year net, no unresolved losing year, cross-coin validation, minute-liquidity audit. Never promote a research candidate merely because syntax passes.

## Close-decision contract

| Entry family | Side | Gate alone, no confirmation | Supported normal close | Emergency exception |
|---|---|---|---|---|
| Range | LONG | false at +5/-5 | ROI >=5 with price at channel midpoint; ROI <=-5 with confirmed downside range break | ROI <=-12 |
| Range | SHORT | false at +5/-5 | ROI >=5 with price at channel midpoint; ROI <=-5 with confirmed upside range break | ROI <=-12 |
| Trend | LONG | false at +5/-5 and at internal gates without signals | Existing v29C reversal/momentum/volume confirmation | ROI <=-20 |
| Trend | SHORT | false at +5/-5 and at internal gates without signals | Existing v29C reversal/momentum/volume confirmation | ROI <=-20 |

Range profit above +8 still requires weakening momentum if the channel midpoint has not been reached. A 24h failure exit additionally requires reliable local position creation time and price on the adverse side of EMA20; the outer +/-5 gate continues to apply. Blank or unrecognized opening hash uses the original trend exit conservatively; release requires verifying hash propagation on the deployed server. Changing an opening expression requires regenerating its bound exit hashes.

Full matching replays and synthetic branch checks are recorded separately below after completion. No additional repository test files are created.

### Compile-only correction before strategy returns

The initial v64a replay stopped at the first evaluation: `floor` cannot redeclare the Expr built-in. Neither initial v64 candidate supplied a strategy return. Preserve the original full files and failed checkpoint. Rename only local `floor`/`ceiling` variables to `range_low`/`range_high`, then regenerate the exact opening-hash bindings through the helper; no threshold or hypothesis changes.

- v64a2: `temp_strategy/v64/03-trend-plus-range-entry-bound-exits-exprfix.json`, SHA-256 `c90ee19fcfffb4b18d6056fbe8e9c975e8f3c2f81fb0a5d4b50d40bd2faf138f`.
- v64b2: `temp_strategy/v64/04-trend-plus-range-daily-guard-exprfix.json`, SHA-256 `6aaa205bbc706d81404a2822df6dcc99427283c6b108a4fe4154471bac445957`.

These are the frozen executable revisions used for development comparison. The earlier BTC v29C control alone returned 101 trades, 0.484/week and +670.737 USDT, which still fails frequency.

## Completed comparison

Twelve full four-year runs (v29C/v64a2/v64b2 × four coins) completed. Both v64 variants have positive total net in all four coins, but BTC/ETH/XRP frequency stays below 0.9/week and losing annual windows remain. Neither qualifies for fresh cross-coin release validation. The 72-case real-struct Expr matrix passed; the mutually exclusive guard-mode negative check rejected conflicting flags without writing output.

Minute-liquidity audit: v64a2 713 trades, v64b2 639 trades; both have one zero-activity BTC base-entry fill at 2024-10-28 20:01 UTC. This is a standard engine assumption, not evidence of a realizable order. Do not remove that ledger row and claim corrected counterfactual PnL. Full metrics and caveats appear in `2026-10-03-arm-v62-v65-summary.md`.
