# Test Strategy Results Evaluation

Use this reference to evaluate one exact custom-strategy version from `test_strategy_results`.

## Verified record lifecycle

- An open trigger inserts one row with the current `technology`, complete `strategy`, matched `open_strategy`, `close_price = "0"`, and `close_profit = "0"`.
- LONG quantity is positive and SHORT quantity is negative.
- The simulator adjusts a LONG open price upward by 0.1% and a SHORT open price downward by 0.1% before insertion.
- Only open rows are checked for closing.
- A close writes the mark price to `close_price`, gross unrealized USDT PnL to `close_profit`, the complete matched expression to `close_strategy`, and the close timestamp to `updateTime`.
- Close expressions are not evaluated while ROI remains inside the symbol's outer `loss` and `profit` thresholds.
- If no executable close rule exists, the simulator has a fallback close beyond positive or negative 10% ROI. Rows closed by that fallback have no normal matched `close_strategy` and must be reported separately.

## Build an uncontaminated cohort

1. Canonicalize both JSON snapshots before comparison so pretty and compact forms of identical JSON match.
2. Group by the pair `(canonical technology, canonical strategy)`, not by `open_strategy` alone.
3. Apply the requested `createTime` window after fixing the version pair.
4. Keep closed and open rows separate.
5. Split fallback exits from normal strategy exits.
6. Report the number of malformed numeric rows and exclude them from calculated metrics.
7. Compare the row or symbol `profit/loss` gates with every internal ROI exit threshold. Mark branches that cannot be reached at their intended ROI because the outer gate delays expression evaluation.

The table has no template ID or template name. Exact JSON snapshots are the strategy-version identity available in the data.

## Calculate closed-trade metrics

For each valid closed row, define:

- `pnl = numeric(close_profit)`
- `margin_return_pct = pnl / numeric(usdt) * 100`, requiring `usdt > 0`
- `holding_ms = updateTime - createTime`, requiring a nonnegative value

Then calculate:

- wins: `pnl > 0`; losses: `pnl < 0`; breakeven: `pnl == 0`
- win rate: `wins / (wins + losses)`, excluding breakeven
- gross profit: sum of positive PnL
- gross loss: absolute sum of negative PnL
- profit factor: `gross_profit / gross_loss`; report undefined when gross loss is zero
- expectancy: average PnL per closed trade
- average and median margin return
- median and p90 holding time
- maximum drawdown from cumulative closed PnL ordered by `updateTime`
- maximum consecutive losing trades in close-time order

Also show the same core metrics by side and symbol. Flag concentration when one symbol or one side dominates the sample or PnL.

## Account for incomplete and missing evidence

- Open rows are censored. Report open count, oldest age, and age distribution; do not convert them to realized outcomes.
- Current mark-to-market PnL for open rows requires a current price from `symbols` or another explicit source and remains unrealized.
- `close_profit` excludes fees, funding, and additional close slippage. Never label gross PnL as net PnL.
- The table does not record maximum adverse excursion, maximum favorable excursion, signal opportunities that did not open, or the internal close-condition branch that fired.
- Results are affected by symbol enablement, scan order and frequency, open-position limits, notification cooldown, price availability, and the outer close gate.
- The table is live: rows can move from open to closed during analysis. Record the snapshot time and obtain the final cohort with one database statement or a supported consistent read. Never merge changing paginated results into one verdict.
- The frontend defaults to a paginated result set and its displayed profit sum covers only loaded rows. Query the full database cohort for evaluation.

## Decide and optimize

Use these conservative default verdicts:

- `insufficient evidence`: fewer than 20 valid closed rows, severe concentration, only one narrow regime, or too many old open rows;
- `promising under tested conditions`: sufficient closed rows, positive gross expectancy and profit factor, acceptable drawdown and losing streak, and no single segment explains nearly all profit;
- `needs optimization`: adequate evidence shows a localized weakness such as one side, symbol group, holding-time bucket, or exit behavior causing losses;
- `invalidated`: adequate and reasonably diverse evidence shows persistently negative expectancy or unacceptable risk without one isolated cause.

Treat 20 closed trades as preliminary and 50 as more stable, not as universal statistical guarantees. State the observation window and concentration alongside every verdict.

When optimizing, change only one evidence-backed dimension, such as entry confirmation, no-chase filter, one side's enablement, exit threshold, or time exit. Give the revision a new versioned name, preserve the old rows, and evaluate the new cohort independently before comparing them.
