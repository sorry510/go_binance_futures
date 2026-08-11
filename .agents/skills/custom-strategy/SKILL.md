---
name: custom-strategy
description: Design, validate, export, and optionally persist efficient expr-lang custom futures strategies for this go_binance_futures project and its go_binance_futrues_new_ui frontend. Use when the user asks to create, revise, simplify, inspect, test, export, insert, or update a strategy_templates custom strategy with technical indicators, K-line conditions, and long/short open and close rules.
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

## Persist only with explicit authorization

When the user explicitly requests a database write:

1. Finish and validate the portable JSON file first.
2. Check for a conflicting template name.
3. Before every `INSERT` or `UPDATE`, compact the `technology` and `strategy` subtrees with `json.Compact`, `json.Marshal`, or `jq -c`. Never store pretty-printed JSON or structural line breaks and indentation in these columns.
4. Use a transaction and prepared parameters to persist `name`, the compact JSON strings, and Unix-millisecond timestamps. Never concatenate JSON into an SQL statement.
5. Reject a duplicate instead of overwriting it. Update an existing row only when the user specifically asks to replace or revise that row.
6. Commit the transaction, then query the row back by name. Require both columns to be valid JSON and byte-for-byte equal to the compact inputs before reporting the row ID and success.
7. Never assign the template to symbols, change `strategy_type`, enable trading, or place orders unless separately authorized.

## Report runtime caveats

Always explain these project-specific constraints:

- The symbol must use `strategy_type=custom` and reference the template before it affects trading.
- `CanOrderComplete` is called only after ROI crosses the symbol's outer `profit` or `loss` threshold. Trend exits, ATR exits, and time exits can therefore be delayed.
- `AutoStopOrder` for custom strategies currently always returns false.
- `Position.CreateTime` time exits are reliable only for positions with a recorded local creation time.
- Database insertion does not constitute assignment, activation, backtesting, or evidence of returns.

Deliver the JSON path, database result when authorized, concise rule explanations, passed checks, blocked checks, and required operational settings.
