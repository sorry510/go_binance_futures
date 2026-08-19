---
name: custom-strategy
description: Design, validate, export, persist, evaluate, and optimize efficient expr-lang custom futures strategies for this go_binance_futures project and its go_binance_futrues_new_ui frontend. Use when the user asks to create, revise, simplify, inspect, test, export, insert, or update a strategy_templates custom strategy, or to judge and optimize a tested strategy from test_strategy_results.
---

# Custom Strategy

Use this project-specific workflow to produce a strategy that matches the real evaluator, current indicator fields, frontend JSON contract, and database schema.

## Establish the current contract

1. Read `AGENTS.md`, `feature/strategy/line/line_custom.go`, `feature/strategy/line/parse_technology_config.go`, `technology/define.go`, and `STRATEGY.CN.md` completely.
2. Discover code with codebase-memory graph tools first. Trace open evaluation, close evaluation, environment construction, K-line ordering, and the outer `profit/loss` gates in `feature/feature.go`.
3. Inspect the active frontend at `/Users/zhz/work/binance/go_binance_futrues_new_ui`. Verify indicator tabs, serialization, validation, autocomplete, and the strategy-test API before relying on a field.
4. Treat `[0]` as the current/forming K-line and `[1]` as the latest closed K-line. Preserve the user's intentional use of live `[0]` values.
5. Re-read current code instead of assuming the indicator list is unchanged. Current families include MA, EMA, MACD, ADX/DMI, MFI, OBV, CCI, ROC, KDJ, RSI, KC, BOLL, Donchian, ATR, and Supertrend.

## Audit existing templates safely

1. Read database connection values from `[database]` in `conf/app.conf` without printing credentials or changing the file.
2. Query `strategy_templates` read-only before designing. Summarize each template by enabled indicators and enabled `long`, `short`, `close_long`, and `close_short` rules.
3. Expand relevant `technology` and `strategy` JSON completely. Check for duplicated names, asymmetric rules, impossible conditions, distant indexes, excessive intervals, and stale comments.
4. Never write during the audit. Treat database insertion, template assignment, trading enablement, and order placement as separate authorization boundaries.

The table stores `id`, `name`, `technology`, `strategy`, `createTime`, and `updateTime`. The two JSON columns contain strings in the API/database but remain structured objects in the portable artifact. The repository artifact may stay pretty-printed for review, but database JSON must always be stored in its single-line compact form.

## Choose a small strategy architecture

Define the horizon, market scope, entry regime, exit regime, and risk assumptions. If the request is vague, state conservative assumptions and proceed.

Prefer the smallest useful set:

- one higher timeframe for direction;
- one lower timeframe for entry;
- one momentum or volume filter only when it removes a specific false signal;
- ATR or a channel for adaptive exits.

For short-term strategies that should run efficiently, target two distinct K-line intervals and three or four enabled indicators. Reusing an interval is cheaper because `ParseTechnologyConfig` caches its fetched K-lines. Do not add indicators merely because they exist.

Useful pairings:

- Supertrend or EMA for direction;
- Donchian `High[1]`/`Low[1]` for a live breakout, because the current channel includes the current candle;
- RSI, ROC, MFI, or OBV change for one focused confirmation;
- ATR for price-scale-independent stop and take-profit distances.

## Write four cohesive rules

Include one enabled rule for each type unless the user narrows the request:

- `long`: higher-timeframe bullish direction, lower-timeframe trigger, and no-chase filter;
- `short`: structurally symmetric bearish conditions;
- `close_long`: hard loss, hard profit, developed-profit protection, setup invalidation, and optional time exit;
- `close_short`: symmetric short-side exits.

Keep each expr program readable and make its last expression Boolean. Use short `let` bindings, valid array indexes, and no unnecessary loops over long ranges. Prefer one cohesive close rule per side because multiple close rules have OR semantics.

Before finalizing close rules, compare every internal ROI trigger with the target symbols' outer `profit` and `loss` settings. The evaluator does not run the close expression while ROI is inside `(-loss, profit)`. Therefore a profit-protection trigger below `profit`, a setup-failure trigger near zero, or a stop whose magnitude is smaller than `loss` cannot fire at its intended level. Align the operational settings or redesign the thresholds, and report the required `profit/loss` values with the strategy.

Do not claim that a rule is profitable or effective merely because it compiles. This project supports current-snapshot testing and forward simulation, not historical backtesting.

## Create the portable JSON

If the user requests design or review only, keep the proposal in the response and do not create a file. Create or edit an artifact only when the request authorizes repository changes.

Write the artifact under `strategy_templates/<descriptive-name>.json`:

```json
{
  "name": "example",
  "technology": {
    "ma": [], "ema": [], "macd": [], "adx": [], "mfi": [],
    "obv": [], "cci": [], "roc": [], "kdj": [], "rsi": [],
    "kc": [], "boll": [], "donchian": [], "atr": [], "supertrend": []
  },
  "strategy": [
    {
      "name": "example_long_open",
      "type": "long",
      "code": "true",
      "fullScreen": false,
      "enable": true
    }
  ]
}
```

Use unique expression-safe indicator names. Revalidate period and multiplier limits from backend and frontend code before saving.

## Validate before persistence

1. Validate JSON shape with `jq`.
2. Run every enabled rule separately through the frontend-equivalent endpoint:

```bash
rtk bash .agents/skills/custom-strategy/scripts/validate_strategy.sh \
  /Users/zhz/work/binance/go_binance_futures \
  /absolute/path/to/strategy.json \
  BTCUSDT
```

3. If the local service is unavailable, report the API check as blocked and compile/run every rule with the real expr package and actual Go environment structs using deterministic synthetic arrays.
4. Interpret `code: 200` only as compile/runtime success. `pass: false` is a valid current-snapshot result.
5. The current test controller responds inside the first enabled-rule iteration and injects one fixed mock position. Validate one rule per request, and do not treat that mock as proof of correct long/short position semantics. Cover all four rule types with real Go environment structs and deterministic LONG/SHORT cases.

## Evaluate forward-test results

Read `references/test-strategy-results.md` completely before deciding whether a tested strategy is effective or needs optimization.

1. Query `test_strategy_results` read-only. Do not judge from the frontend's current-page profit total because it does not represent the complete filtered cohort. Record an `as_of` timestamp and fetch each live database cohort with one SELECT when possible. Do not combine pages collected while open rows are still being closed; if data changes during inspection, rerun the final metrics from a fresh single-statement snapshot.
2. Select one exact strategy version by canonicalized `technology` and `strategy` snapshots plus the requested test window. Do not mix JSON formatting variants, revised rules, or different indicator configurations, and do not identify a cohort only by template name.
3. Separate open rows where `close_price == "0"` from closed rows. Treat open rows as censored observations: report their count and age, but never count them as wins, losses, or realized profit.
4. For closed rows, report sample count, wins/losses/breakeven, gross profit and loss, gross PnL, win rate, profit factor, expectancy, average and median margin return, median and p90 holding time, maximum closed-trade drawdown, and maximum losing streak. Segment at least by `LONG`/`SHORT`, symbol, and test period.
5. Calculate margin return as `close_profit / usdt * 100` only when both values parse correctly and `usdt > 0`. Reject malformed rows from numeric metrics and report how many were rejected.
6. Treat `close_profit` as gross simulated USDT PnL. It excludes trading fees, funding, and additional close slippage. Apply an explicit cost assumption when the user supplies one; otherwise report gross results and state that net effectiveness remains unproven.
7. Include the simulator's behavior in the diagnosis: open prices already contain a 0.1% adverse adjustment, and close rules are evaluated only after the symbol's outer `profit` or `loss` gate is crossed. `close_strategy` stores the complete matched close rule, not the internal Boolean branch that caused the exit.
8. Return exactly one verdict: `insufficient evidence`, `promising under tested conditions`, `needs optimization`, or `invalidated`. Use 20 closed trades as the default minimum for a preliminary verdict and 50 for a more stable verdict, but raise the requirement for concentrated symbols, one-sided samples, or a narrow market regime.
9. Optimize only when the evidence points to a specific failure mode. Change one logical dimension at a time, create a new named strategy version, preserve the old template and results, and compare non-overlapping version cohorts. Never update templates, delete test rows, or enable trading without explicit authorization.

Forward simulation is stronger evidence than compile and current-snapshot checks, but it is not a historical backtest or live-trading proof. Do not claim general profitability from one test window.

## Persist only with explicit authorization

When the user explicitly requests a database write:

1. Finish and validate the portable JSON file first.
2. For DBX writes, read [references/dbx-strategy-template-persistence.md](references/dbx-strategy-template-persistence.md) completely and follow its exact-name check, compact UTF-8 hex encoding, direct `INSERT ... VALUES`, and strict readback workflow.
3. Reject a duplicate instead of overwriting it. Update an existing row only when the user specifically asks to replace or revise that row.
4. Never assign the template to symbols, change `strategy_type`, enable trading, or place orders unless separately authorized.

## Report runtime caveats

Always explain these project-specific constraints:

- The symbol must use `strategy_type=custom` and reference the template before it affects trading.
- `CanOrderComplete` is called only after ROI crosses the symbol's outer `profit` or `loss` threshold. Trend exits, ATR exits, and time exits can therefore be delayed.
- `AutoStopOrder` for custom strategies currently always returns false.
- `Position.CreateTime` time exits are reliable only for positions with a recorded local creation time.
- Database insertion does not constitute assignment, activation, backtesting, or evidence of returns.

Deliver the JSON path, database result when authorized, concise rule explanations, passed checks, blocked checks, and required operational settings.
