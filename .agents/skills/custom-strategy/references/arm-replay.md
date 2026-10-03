# Programmatic ARM replay

Use this workflow for read-only template snapshots and historical strategy research, especially when App UI is prohibited. Parse only the commented `# arm` block in `conf/app.conf`; the scripts whitelist `go_binance`, `go_bn_oracle1`, and `go_bn_oracle2`. Keep configuration and databases unchanged unless the user separately authorizes a database mutation.

## Snapshot and diagnostics

Run `go run scripts/research_arm_snapshot.go scripts/research_arm_data.go -conf <project>/conf/app.conf -output <cache>/snapshot.json`. Paths to scripts are relative to the skill directory. The snapshot reads all three databases in read-only repeatable-read transactions. Optional `-v29-export <cache>/v29.json` exports the single v29 template from `go_binance`; compare canonical JSON rather than SQL ids or whitespace.

For a complete live-table review, add `-table-export <cache>/tables.json` to the snapshot command. It captures the complete current column names/types and every row from both tables inside each database's read-only repeatable-read transaction, verifies row counts and unique IDs, and records separate per-database cutoff times. Output permissions are 0600. Pass the resulting `databases.{name}.rows` artifact to `scripts/analyze_results.py`; separately verify snapshot/rule hashes, saved rule membership, current-template semantic identity, and system-exit attribution. A positive short forward window under different leverage/gates does not establish the historical release contract.

If an ARM minute read fails or stalls, use `-diagnose-minute-symbol ADAUSDT` to inspect active minute-query state, chunk sizes/coverage, and the composite-index query plan. A terminal `unexpected EOF`/`bad connection` is an acquisition failure and supplies no strategy evidence. Poll the same live handle; restart only after authoritative termination or a diagnosed repair to the owned process. Preserve completed result checkpoints and verified archive manifests.

## Replay and source selection

Run `go run scripts/research_arm_replay.go scripts/research_arm_data.go -conf <project>/conf/app.conf -strategy-files <candidate.json> -symbols <symbols> -start <UTC date> -end <UTC date> -output <cache>/results.json`. The harness uses the actual standard 1-minute project engine and records cost assumptions, strategy snapshots/hashes, dataset hashes, per-coin frequency, fees/funding, drawdown, sides, and Sep–Aug annual windows. Confirm its current config when different dates or costs are requested.

Default `-execution-source arm -indicator-source arm` reads ARM minute chunks/raw bars and higher intervals, filling the earlier prefix from official Binance archives. If large ARM payload transfers fail, or current canonical higher-interval bars differ from ARM values, use both exact flags:

```text
-execution-source public-archive -indicator-source public-archive
```

Select a separate `-cache-root` for a different source, such as `<cache>/public-canonical`; preserve existing ARM caches/results. The harness rejects a cache whose recorded source differs from the request. Re-run development and validation coins on the same source before comparing strategy versions; do not mix old ARM development returns with canonical holdout returns as release proof. ARM still provides the frozen template source and historical funding.

Official monthly files require their published SHA-256 checksum. Per-file local manifests store URL, path, size, and checksum; reuse recomputes hash and size before parsing. Parallel downloads are limited to requested interval streams. `-archive-timeout` defaults to 45 seconds per request with bounded retries; `-archive-proxy` may select a tested route without changing `app.conf`. Do not route around venue rate limits.

Require exact timestamp continuity and coverage for every execution/indicator interval and historical funding. Only bounded, exact historical REST bars may repair small archive gaps; never interpolate or turn missing coverage into zero trades. Preserve warmup and every candidate's required interval union.

## Canonical cross-interval volume differences

Official minute sums and official hourly/daily archives can have different volume/trade-count values despite equal prices. Do not overwrite canonical interval values with minute aggregates or assume every such difference is ARM corruption. When ARM indicator values disagree with minute sums, verify the corresponding checksum-verified official interval archive. If ARM differs from that archive, keep the original database/cache intact and switch to an independent full-canonical dataset for research.

The canonical loader enforces aggregate OHLC/time consistency, records independent canonical volume/trade-count discrepancy counts and maximum relative quote-volume difference, and retains canonical interval volumes. In price aggregation, zero-activity carry minutes do not create traded OHLC extremes, but their timestamps remain mandatory. A price mismatch aborts ordinary acquisition; independently proven source corrections may be applied only through the explicit workflow below. Hash changes, unverified corrections, or unresolved coverage gaps still abort. Preserve `dataset_source.aggregate_checks` and explain any effect on signal comparability.

## Verified price-source corrections

When a checksum-valid monthly file contains stale or empty observations, first run `research_audit_source.go` with `research_arm_data.go`, `-conf`, `-symbol`, aligned RFC3339 UTC `-start`, `-interval 1h|4h`, `-cache-root`, and `-output`. Optional `-trades` adds the checksum-verified daily tape. This reads the ARM window in a read-only transaction and compares ARM, official monthly, and official daily streams; it does not mutate data. A valid checksum proves file integrity, not correctness of every historical value. Binance documents that archives may be revised: <https://github.com/binance/binance-public-data#updates>.

For demonstrated price defects, add this exact replay switch alongside full public-archive source flags:

```text
-minute-repair verified-archive
```

Use a separate `-cache-root` and `-output` for repaired studies and for a changed repair version; preserve original ARM and uncorrected canonical results. Share only checksum-verified downloads through `-archive-cache-root <existing-cache>/archives`. The dataset cache and completed-result checkpoint both reject source/repair-policy or repair-version mismatches. Re-run development and validation on one correction version before ranking candidates.

The correction evidence must be independent and match all OHLC, timestamps, base/quote volume, trade count, and taker turnover fields within existing numeric precision. A daily minute archive can replace stale monthly minutes when its complete-hour aggregate matches the official hourly reference. Otherwise a daily trade reconstruction must match an official or read-only ARM hourly reference exactly. A higher-interval price correction requires either a checksum-verified daily interval bar exactly matching the minute aggregate, or, for hourly repairs only, an ARM bar plus a separately verified daily minute/tape aggregate both matching exactly. Never manufacture a higher bar merely to pass its own minute check. If independent proof is absent, stop acquisition rather than accept the discrepancy or report zero trades.

For USD-M tape reconstruction, turnover is `price * base quantity`; supplied trade `quoteQty` can contain stale values. Retain the original archive/checksum, reject duplicate IDs or invalid prices/quantities, and verify the reconstructed whole-hour fields against the independent reference. Trade-ID span holes are recorded, not filled with invented trades; they remain a source-quality caveat, not proof of a missing execution bar. Preserve all before/after bars, source URLs/checksums, proof type, changed-minute counts, and ID-span observations under `dataset_source.minute_repairs` and `indicator_repairs`. Repairs are bounded; only price defects trigger correction, while price-consistent canonical volume differences remain separately reported.

Keep the candidate hash, universe, costs and dates frozen during source repair. These corrections change research data only, never `app.conf`, database tables, production indicators, live rules, or release gates.

## Sizing and execution audit

Inspect the actual engine before calling sizing fixed: current entry notional is `max(cash, 0) * PositionSizePct * Leverage`. With `PositionSizePct=0.1`, margin uses 10% of current available cash and compounds; it is not 10% of initial equity at every trade.

Run `research_trade_features.go` with `research_arm_data.go`, `-results`, `-version-prefix`, `-cache-root`, and `-output` to audit recorded entry/exit minutes and rule attribution. It requires the ledger's data hash to match the immutable dataset and records minute trade counts/quote volumes plus `zero_liquidity_fill`. The standard engine does not itself reject fills on zero-activity carry bars; audit actual fills before claiming executable evidence. A clean minute-liquidity check is not a historical order-book fill guarantee. Feature-ledger filtering is descriptive attribution, never realized counterfactual PnL.

Keep full portable candidate JSON, including failures, in `temp_strategy`; keep large datasets, trade ledgers, and replay results in the research cache. A source repair before any returns are inspected does not contaminate an untouched holdout; record the repair and keep candidate/universe/cost parameters frozen.

## Entry-family-bound comparisons

When combining complementary entry families whose exits should not change with the current market regime, run `research_combine_entries.go` with `-base`, `-supplement`, `-name`, `-output`, and `-entry-bound-exits`. The supplement must have exactly one enabled long and short entry and one close per side. The helper uses the project's `RuleHash` and wraps mutually exclusive exit guards around `OpenStrategyHash`. It rejects conflicting guard modes and identical base/supplement entry hashes. Confirm that both the deployed live evaluator and the historical environment populate the hash; blank/unknown hashes conservatively take base exits and are not proof of correct supplement routing. Regenerate bound exits after every opening-expression change. Keep all four rule types and save every full candidate/control, including failed syntax, under `temp_strategy`.

Compile/run each enabled expression with actual project Go structs before a long replay. Local names cannot shadow Expr built-ins such as `floor`; preserve failed files, correct names in a new revision, and regenerate hash bindings. Exercise reachable LONG/SHORT entries, ordinary ROI gate without a signal (false), confirmed profit/loss (true), the explicitly deeper emergency stop (true), and wrong-entry-hash rejection. Test opposite directions rather than infer a profitable inverse from losses. A clean compile/branch matrix does not establish profits.
