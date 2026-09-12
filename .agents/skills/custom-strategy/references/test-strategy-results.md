# Test Strategy Results Evaluation

Evaluate immutable forward-test snapshots, not template names or the latest UI page. This contract was verified against the repository and all three arm databases on 2026-09-06; recheck schemas and writers on later runs.

## Source and field contract

Read these current sources for results reviews:
- `models/tableStruct.go`: `TestStrategyResults`.
- `feature/feature_test_strategy.go`: `createTestResult`, `CheckTestResults`.
- `service/strategy/identity.go`: template attribution, snapshot/rule hashes, system exit identifiers.
- `service/strategy/service.go`: fee/PnL calculation and full-filter API queries.
- `service/strategy/stats.go`: grouping, realized-only statistics, win-rate denominator.
- `controllers/testStrategyResult.go` and frontend `src/views/order/testOrder.vue` when comparing displayed statistics.

Prefer graph discovery, but check current files when the graph returns stale line ranges or omits new fields. Results-only reviews do not require re-reading unrelated indicator implementations.

### Identity relationships

| Field | Meaning and use |
|---|---|
| `strategy_template_id` | Source template ID; join to `strategy_templates.id` within the same database. IDs are not globally unique. Zero/missing/deleted templates do not invalidate the stored strategy snapshot. |
| `strategy_template_name` | Recorded attribution label (up to 128 characters), not immutable version identity. Preserve it alongside the current template name. |
| `technology`, `strategy` | Authoritative configuration at entry, retained even if the template later changes. |
| `strategy_snapshot_hash` | SHA256 of `TrimSpace(technology) + "\n" + TrimSpace(strategy)` in UTF-8. It is NOT a hash of canonicalized JSON. |
| `open_strategy_name/type/hash` | Matched entry program; type normally `long` or `short`. |
| `close_strategy_name/type/hash` | Matched exit program; type normally `close_long`, `close_short`, or `system`. |
| `open_strategy`, `close_strategy` | Program text, not its evaluated environment or the internal Boolean branch. Rule hash is SHA256 of trimmed code; empty code has empty hash. |
| `open_fee_rate`, `close_fee_rate` | Decimal fee-rate snapshots. 0.0005 means 0.05%, not 0.0005%. |
| `profit`, `loss`, `leverage`, `usdt` | Entry-time operational settings; keep separate cohorts when they differ. |

Use `(database, template ID or unresolved attribution, snapshot hash)` as the initial version key. Check stored hashes against source strings and compare historical snapshots against the current template. Different raw hashes can reflect JSON key order/spacing only: canonicalize parsed objects for comparison, preserving array order and exact expression strings, then record any formatting-only aliases explicitly. Do not merge changed code, indicator parameters, enable flags, or names without reporting what differs. A rule hash alone is not a strategy version: identical close rules may be shared by several templates.

Validate side/type consistency and that named rules exist in the saved strategy. A name or template-ID match alone does not prove that the live template still contains the tested version. Identity fields may have been backfilled: complete metadata does not establish when the new writer was deployed.

## Read a complete, stable cohort

1. Refresh DBX connections and resolve the requested connection (historically `arm`) instead of trusting cached IDs. Inspect both tables in each requested database; feature availability may differ across deployments.
2. Record `as_of`, database timezone, row count, minimum/maximum IDs and entry/close times. Use `UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3))*1000` for epoch time; do not pass UTC_TIMESTAMP through UNIX_TIMESTAMP under an unverified non-UTC session.
3. Fetch every required row in one SELECT when practical, including prices, signed quantity, fee rates, stake, leverage, gates, identity fields, side, and timestamps. Fetch complete JSON snapshots once per distinct version and verify identity rather than repeatedly exporting large expressions.
4. DBX currently caps results at 100 rows and cells at 4000 characters. For bounded cohorts, pack small JSON arrays of rows per cell with ROW_NUMBER and JSON_ARRAYAGG, then verify payload lengths, batch count, row count, unique IDs and no truncation. Long JSON/programs can be explicitly expanded by HEX/SUBSTRING chunks with lengths checked. These are read-only SELECTs.
5. If more than one page is necessary, use a DBX pinned session with a supported read-only consistent transaction, then close it. A fixed maximum ID alone does not freeze open rows that may close between pages. If a consistent transaction is unavailable, use one-statement aggregate metrics; do not present mixed live pages as one snapshot.
6. Separate snapshot acquisition from subsequent identity diagnostics. Keep the original snapshot metrics stable; record later queries as later checks. Three separately read databases have separate observation times and are not a synchronized distributed snapshot.
7. Apply the intended entry window to `createTime`. A close-date view answers a different question. Report both when needed, label timezone, and compare A/B/C on their common entry window and a common observation cutoff without using outcomes learned after that cutoff.

Example compact export for MySQL 8 (adjust selected columns and batch size after checking limits):

```sql
WITH numbered AS (
  SELECT r.*, ROW_NUMBER() OVER (ORDER BY id) AS rn
  FROM test_strategy_results r
)
SELECT FLOOR((rn - 1) / 8) AS batch,
       JSON_ARRAYAGG(JSON_ARRAY(
         id, symbol, price, close_price, position_amt, usdt, leverage,
         profit, loss, close_profit, open_fee_rate, close_fee_rate,
         createTime, updateTime, strategy_template_id, strategy_snapshot_hash,
         position_side, open_strategy_hash, close_strategy_hash, close_strategy_type
       )) AS payload,
       COUNT(*) AS batch_count,
       COUNT(*) OVER () AS total_batches,
       CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3))*1000 AS UNSIGNED) AS as_of_ms
FROM numbered
GROUP BY FLOOR((rn - 1) / 8)
ORDER BY batch;
```

Do not infer the order within JSON_ARRAYAGG; sort decoded rows by ID or close time explicitly. If more than 100 batches would be returned or any cell is truncated, change the retrieval plan before analyzing.

## Recompute PnL and reconcile accounting

For a valid closed row, with entry E, exit X, signed quantity Q, leverage L and fee snapshots fo/fc:

```text
gross_pnl = (X - E) * Q
open_fee = E * abs(Q) * fo
close_fee = X * abs(Q) * fc
fees = open_fee + close_fee
net_pnl = gross_pnl - fees
margin_return_pct = net_pnl / usdt * 100
evaluator_ROI = gross_pnl / (abs(Q) * X) * L * 100
evaluator_NetROI = net_pnl / (abs(Q) * X) * L * 100
holding_hours = (updateTime - createTime) / 3_600_000
```

Use signed SHORT quantity in gross PnL and absolute quantity for both fees. Leverage is already reflected in Q: do not multiply PnL by leverage again. The margin return and evaluator ROI denominators differ; do not use them interchangeably to infer which stop threshold fired.

- Current close writers store fee-adjusted `close_profit` rounded to 3 decimals. Recompute from prices and fee snapshots for metrics, matching the current statistics API, and retain the stored value for reconciliation. Never subtract fees from the stored net value a second time.
- Flag `abs(stored - recomputed_net) > 0.000501` USDT per row as an accounting discrepancy. Report count, IDs, period, fee coverage, and aggregate difference. Investigate old code/deployment/backfill rather than silently correcting database rows. A stored sum may differ slightly from summing unrounded reconstructed PnL.
- Zero fee rates in legacy rows mean fee modeling is absent or unverified unless there is explicit evidence otherwise. Report zero/nonzero-rate segments separately. Do not fill old fees with today's rate without authorization; a labeled sensitivity scenario is distinct from observed results.
- Entry prices already include an adverse 0.1% adjustment before tick/quantity rounding; do not deduct that adjustment again. Funding and additional exit slippage are not included. Call results “fee-adjusted simulated net PnL,” not fully costed live-trading net profit.
- Validate nonempty finite numeric values, E > 0, X >= 0, nonzero Q, L > 0, positive stake for margin metrics, sensible nonnegative fees, signed side consistency and nonnegative durations. Reject invalid records from affected metrics and report them separately. Invalid closed rows are not open positions.
- Open rows with zero close price are censored. Report count, oldest/median/p90 age. Never use their default zero close_profit as breakeven. If mark-to-market is requested, use explicitly timestamped prices and keep unrealized PnL separate.

## Metrics and attribution

For each independently identified version, calculate:
- total/open/valid-closed/rejected rows; wins, losses and breakeven;
- signed gross PnL, fees, net PnL, positive-net sum and absolute negative-net sum;
- net profit factor = positive-net sum / absolute negative-net sum (undefined with no losing trades);
- net expectancy per closed trade;
- win rate = wins / valid closed count, matching current API (includes breakeven in denominator); label a non-breakeven win rate separately if used;
- mean/median net margin return, median/p90 holding time;
- maximum drawdown of cumulative net closed PnL ordered by (updateTime, id), starting from zero; this is USDT closed-trade drawdown, not account equity or intratrade drawdown;
- longest loss streak in the same order, with a win or breakeven ending the streak.

Break down side, symbol, entry day/period, fee regime, stake/leverage/gates, entry rule and exit rule. Nest rule groupings inside template version (or include its key) to avoid combining shared close logic across different entry strategies. Show top-profit contribution and results excluding the best symbol/trade before attributing general effectiveness.

Current API caveats:
- `stats` is computed from all filtered records, not only the page. Its realized metrics exclude open rows.
- `current_profit` is also full-filter, but adds closed PnL and currently marked open PnL, rounded per row. It is not closed realized PnL.
- API `gross_profit` is signed gross PnL, not the positive-win sum used in profit factor.
- API rule groupings may combine shared rule identities across templates; re-group by template version for controlled comparisons.
- API list/count/profit queries are separate statements, not an atomic snapshot. Malformed price rows may be classified as open by the API; an independent review must flag them as invalid.

## Exit attribution is not execution proof

The current system fallback identity is:
- name: `system_roi_10pct_fallback`
- type: `system`
- code: `ROI > 10 || ROI < -10`
- hash: `RuleHash(code)`

Separate system, normal rule and unknown legacy exits. Blank legacy close text is not by itself proof of fallback. Neither a whole-rule hash nor an ROI near a threshold identifies the internal branch that fired.

In current `CheckTestResults`, `findStrategy` becomes true after a matching enabled close rule compiles and executes successfully, even when the result is false. Fallback requires `!findStrategy`. Therefore:
- system exits alongside a saved enabled matching close rule require investigation of runtime compilation/evaluation failures, missing indicators/environment, deployed code differences, or historical backfill;
- do not infer that “normal rules were not satisfied” should trigger fallback;
- do not treat system-exit profits as proof the intended profit-taking branch worked;
- do not infer that deleting losing exits and retaining profitable system exits would recreate those outcomes: exit groups are selected by outcomes and execution paths, not randomized control groups;
- do not change strategy parameters solely to compensate for an unresolved execution-path mismatch.

Outer eligibility is checked against current symbol profit/loss settings (default 3 if unavailable), not necessarily the entry snapshot's gates. The simulator then refreshes the mark price for expr ROI/PnL. Record this drift risk. It can close only one normal-rule position per scan because it returns after an ordinary close; fallback closes can continue in the loop. Scan delays and price gaps can overshoot nominal stops.

## Limits, verdict and optimization

The current table still has no rule-branch ID, indicator snapshot, MFE/MAE, scan timestamp, missed-signal count, or funding/exit-slippage records. Do not reconstruct historical indicator values from today's data.

Give exactly one verdict per version:
- `insufficient evidence`: fewer than 20 valid closes, concentration, substantial censoring, a narrow window, unknown costs or unresolved execution attribution.
- `promising under tested conditions`: adequate fee-covered and attributable samples, positive net expectancy/PF, acceptable observed risk, and no single symbol/exit artifact explains the profit.
- `needs optimization`: adequate reliable evidence isolates a specific weakness.
- `invalidated`: adequate diverse reliable evidence is persistently negative without an isolated remediable cause.

Twenty closed trades is preliminary and fifty more stable; neither overrides poor coverage or execution anomalies. Long-only strategies are not penalized for intentionally having no shorts, but cannot establish a two-sided edge. Compare fee/risk settings, symbol universe, scan/position limits, overlapping time windows and deployment before ranking control groups.

Only optimize a demonstrated failure mode, one logical dimension at a time, using a new named version. Preserve old templates/results. A results review or skill update does not authorize database writes, strategy reassignment, enabling trading or order placement.

## Reusable analyzer

`scripts/analyze_results.py` reads an already captured JSON snapshot; it makes no database connection or writes. Input:

```json
{"databases":{"go_binance":{"as_of_ms":1788696926069,"rows":[{"id":1,"symbol":"BTCUSDT","price":"100","close_price":"101","position_amt":"1","usdt":"20","leverage":5,"profit":"5","loss":"5","close_profit":"0.899","open_fee_rate":"0.0005","close_fee_rate":"0.0005","createTime":1788690000000,"updateTime":1788691000000,"strategy_template_id":1,"strategy_snapshot_hash":"verified-hash","position_side":"LONG","open_strategy_hash":"entry-hash","close_strategy_hash":"exit-hash","close_strategy_type":"close_long"}]}}}
```

```bash
rtk proxy python3 .agents/skills/custom-strategy/scripts/analyze_results.py /absolute/path/snapshot.json --summary
```

Omit `--summary` for symbol, day, side, fee/settings and rule breakdowns. Hash/snapshot verification, template names, runtime investigation, historical-regime coverage and the final verdict remain review steps; the script does not claim to perform them. It uses Decimal arithmetic and reports invalid rows and stored-PnL discrepancies.
