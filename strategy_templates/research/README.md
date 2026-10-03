# Strategy Research Audit Bundles

`strategy_templates/result.md` remains the human-readable research index. Each material research family should also keep a self-contained audit bundle under this directory so later work can verify the evidence without relying on `/tmp` files or conversation history.

Minimum bundle contents:

- `README.md`: hypothesis, dataset split, final result, freeze/continue decision, and canonical evidence files.
- `config.json`: fixed execution parameters and eligibility/entry/exit semantics used for the run.
- `protocol.md`: discovery/OOS protocol and the decisions frozen before reading results.
- `provenance.json`: research date, repository revision, public data sources, and known limitations.
- `replay.py` or equivalent: executable research logic used to recreate the study.
- `inputs/`: complete event universe plus eligibility decisions and exclusion reasons. Keep excluded events as well as eligible events.
- `results/`: per-trade results, skipped cases, and a machine-readable summary.
- `manifest.sha256`: hashes of the archived evidence files.

Large Binance Vision ZIP/K-line archives should not be stored here. Record the exact source convention and the symbols/months needed to reconstruct them instead.

If an intermediate run is superseded, preserve it only when it explains an audit correction. Put it under `legacy/` and mark it explicitly as non-canonical.

Archived bundles currently include:

- `margin-additions-direct-long/2024-discovery/`: complete event universe, eligibility, replay, and failed 2024 discovery evidence.
- `id121-v54-strict-production/2026-09-24/`: strict same-start ID121/v54 benchmark, exact ID121 DB snapshot, normalized trade export, and raw-vs-normalized attribution separation.
- `monitoring-tag-addition-direct-short/2024-2026/`: preserved three-year historical replay evidence with the funding-parser defect explicitly marked; exact PF/net remains pending corrected-parser replay.

- `open-interest-price-state/2026-early-gate/`: pre-registered 2026 early gate plus full-2025 historical OOS; the apparent OI-contraction reversal edge failed OOS and the family is frozen.

- `funding-settlement-shock-reversal/2023-2026/`: extreme-funding crowding reversal with 2023–2024 discovery and 2025–2026 OOS; OOS edge vanished/reversed, so the family is frozen.

- `funding-interval-compression/2025-2026-feasibility/`: local archive has no 1h funding observations and no production-eligible 8h->4h transition; frozen as currently unverifiable rather than a failed alpha.
- `premium-index-extreme-reversal/2024-2026/`: fixed 24h/3σ Premium Index crowding-reversal study; 2024/2025 edge was economically negligible, so later 2026 strengthening was not backfit and the family is frozen.
- `upbit-new-market-direct-long/2024-discovery/`: official Upbit new-market event universe, Binance production eligibility, and exact 1m direct-LONG replay; 12 trades produced PF 0.096 and the family is frozen.

- upbit-risk-warning-direct-short/2023-2024-discovery/: official Upbit risk-warning SHORT; 12 exact-1m trades, PF 0.452, frozen in discovery.
- permutation-entropy-continuation/2023-2026/: fixed low-permutation-entropy continuation; discovery sample too sparse, frozen without threshold relaxation.
- average-trade-size-shock-continuation/2023-2026/: extreme average-trade-notional continuation; weak breadth and 2026 reversal, frozen.

- vpin-style-toxicity-continuation/2023-2026/: VPIN-style taker toxicity continuation; discovery negative and regime unstable, frozen.
- roll-spread-shock-reversal/2023-2026/: Roll implied-spread shock reversal; cross-year sign instability, frozen.
- volume-concentration-shock-continuation/2023-2026/: 24h volume-HHI continuation; promising diagnostic but exact-1m PF 0.861 after costs, frozen.
- binance-leverage-margin-tier-change/2023-2024-discovery/: official leverage-tier loosening/tightening event study; full family negative, frozen.
- cross-exchange-price-dislocation-catchup/2023-2026/: Binance–Bybit hourly catch-up; negative discovery and 2026 failure, frozen.

- binance-leverage-tier-first-bracket-shock/2024-2026/: official first-bracket leverage-cap changes; PF 2.06 discovery, 1.50 OOS1, then 0.43 OOS2. Frozen after hard 2026 failure.

- binance-portfolio-margin-collateral-ratio/2024-feasibility/: 36 official asset-events but only 7 strict eligible USD-M events; frozen before return inspection due insufficient discovery sample.

- stablecoin-supply-zero-cross/2023-2026/: DefiLlama aggregate USD-stablecoin supply sign-cross; corrected 10-symbol discovery is uniformly negative, frozen without direction reversal.

- volume-concentration-shock-continuation/2023-2026/: HHI volume-concentration diagnostic looked broad, but exact 1m discovery PF0.861; frozen.
- amihud-illiquidity-shock-reversal/2023-2026/: fixed Amihud liquidity-shock reversal failed discovery and OOS; frozen.
- coinm-quarterly-term-structure-reversion/2023-2026/: nearest-quarter COIN-M basis mean reversion failed discovery and 2025 OOS.
- coinm-liquidation-cascade-continuation/2023-2024/: official COIN-M liquidation continuation failed; opposite exhaustion sign is only a future-OOS hypothesis.
- binance-margin-one-hour-interest-waiver-short/2023-2024/: borrow-cost waiver SHORT passed 2023 directionally but flipped negative in 2024 OOS.
- coinm-quarterly-term-structure-convergence/2023-2026/: quarterly-vs-perpetual basis convergence failed discovery and OOS.
- coinm-usdm-oi-share-reversal/2023-2024-early-gate/: coin-margined OI-share stress reversal failed core-4 early gate.
- coinm-usdm-taker-divergence/2023-2024-early-gate/: CM-vs-UM taker-flow divergence failed cross-symbol early gate.
- binance-unplanned-network-suspension/2023-2024-feasibility/: only two clear unplanned incident classes; insufficient sample.
- coinmetrics-active-address-growth/2023-2026/: active-address growth zero-cross failed discovery/OOS.
- coinmetrics-mvrv-extreme-reversal/2023-2026/: MVRV extreme reversal failed discovery and 2026 OOS.
- large-aggressor-order-flow/2026-early-gate/: q99 large-aggressor signed flow; SOL/XRP early gate too sparse/inconsistent, frozen before BTC/ETH expansion.
- binance-loanable-asset-addition-short/2023-2024/: loanable-asset SHORT diagnostic looked positive, but frozen exact 1m discovery PF0.694; frozen.
- `defillama-protocol-dex-market-share/2023-2026/`: protocol-level DEX market-share zero-cross; 607-event 2023-2024 discovery was broadly negative, so OOS remained untouched and the family is frozen.

- defillama-protocol-dex-share-momentum/2023-2026/: protocol-level DEX market-share momentum; identity-corrected 2023-2024 discovery was strongly negative, so OOS was intentionally not read and the family is frozen.
- snapshot-governance-rejection-short/2023-2024-feasibility/: strict binary Snapshot rejection event study; only 2 negative-win events and both LDO, frozen before return inspection.

- `defillama-token-holder-revenue-flow/2023-2026/`: direct token-holder value-accrual flow; strong discovery and positive OOS magnitude, but frozen because OOS breadth was 10/17=58.82%, below the pre-registered 60% gate.
- `defillama-cex-token-reserve-flow/feasibility/`: free CEX API exposes only current exchange-level aggregate inflows/assets; historical per-token reserve/flow data is not available through the free API, so frozen before returns.

- `github-core-commit-activity/feasibility/`: commit timestamps do not provide auditable default-branch visibility time; frozen before returns.
- `binance-futures-index-constituent-adjustment/feasibility/`: no mechanically identifiable 2023-2024 official index constituent/weight announcements; coverage blocked.
- `binance-futures-minimum-notional-adjustment/feasibility/`: only 2 official articles / 6 contracts; frozen before returns for insufficient coverage.
- `binance-margin-dynamic-interest-rate/feasibility/`: only a generic dynamic-rate rule announcement, no asset-level historical event series; frozen before returns.
- `defillama-chain-bluechip-stablecoin-share/2023-2026/`: 754-event USDT+USDC share discovery narrowly passed economic magnitude but failed breadth (5/10); OOS untouched and family frozen.

- `defillama-chain-stablecoin-concentration-hhi/2023-2026/`: complete same-source stablecoin-composition reconstruction; 823-event discovery failed economic gate despite 7/11 breadth, OOS untouched.
- `binance-usdm-liquidation-history/feasibility/`: Binance Vision USD-M has no historical liquidationSnapshot/forceOrders archive; blocked before returns.

- `okx-announcement-event-archive/feasibility/`: deep official history exists, but Help Center pagination is not snapshot-stable and direct OKX API validation is TLS-blocked from the research host; frozen before returns.
- `previous-day-range-acceptance/2026-09-30-early-gate/`: project-native previous-day range acceptance continuation; 273-trade core-4 early gate PF 1.071, 2/4 positive, frozen without tuning.
- `legacy-3m-exhaustion-wick-reversal/2026-09-30-early-gate/`: modern JSON translation of legacy 3m exhaustion/wick logic; blocked before returns because local 3m historical coverage is incomplete; no DB backfill performed.
- `1h-exhaustion-wick-reversal/2026-09-30-early-gate/`: 1h directional-run exhaustion/wick reversal; 495-trade core-4 PF 0.652, 0/4 positive, frozen.
- `price-taker-multihour-divergence/2026-09-30-early-gate/`: 3h price-vs-TakerBuyRatio dynamic divergence reversal; 1396-trade PF 0.863, 1/4 positive, all years <1; frozen.

- `previous-week-range-acceptance/2026-09-30-early-gate/`: project-native calendar-week range acceptance; blocked before returns because local 1w historical series is incomplete; no DB backfill performed.
- `keltner-wide-excursion-reentry/2026-09-30-early-gate/`: modern 4h/1d Keltner wide-excursion re-entry; only 4 core-4 trades, frequency 0.005/week, frozen without parameter relaxation.

- `rolling-7d-range-acceptance/2026-09-30-early-gate/`: project-native prior-7d range acceptance; 819-trade core-4 PF0.867, 0/4 positive, frozen without tuning.
- `triple-ema-trend-start/2026-09-30-early-gate/`: 4h modern adaptation of legacy triple-EMA start; only four core-4 trades, frequency 0.005/week; frozen without relaxation.

- `atr-term-structure-expansion/2026-09-30-early-gate/`: project-native 1h-vs-1d ATR term expansion; 758 trades, PF1.090, 3/4 positive; failed preregistered PF gate and frozen.
- `hourly-trend-strength-breakout-standard-exit/2026-09-30-early-gate/`: existing hourly trend-strength entry under standardized TP8/SL6 exits; only four core-4 trades, frozen for insufficient frequency.

- `low-activity-taker-accumulation/2026-09-30-early-gate/`: two-hour quiet taker-flow accumulation breakout; 3800 trades, PF0.846, 0/4 positive; frozen.

- `adx20-trend-start/2026-09-30-early-gate/`: standalone ADX14 cross-above-20 trend start; 779 trades, PF1.080, 3/4 positive but 2024 PF0.759; frozen.

- `rolling-4h-vwap-reclaim/2026-09-30-early-gate/`: true VWAP setup blocked before returns because strategy DSL does not expose base volume.

- `daily-taker-regime-transition/2026-09-30-early-gate/`: daily TakerBuyRatio neutral-cross regime with hourly confirmation; 1367 trades, PF0.852, 0/4 positive; frozen.

- `qps-term-structure-expansion/2026-09-30-early-gate/`: hourly-vs-daily QPS activity expansion; strict TP8/SL6 4777 trades, PF0.829, 0/4 positive; frozen.

- `record-net-aggressor-flow/2026-09-30-early-gate/`: record absolute 1h net aggressor quote-flow continuation; strict TP8/SL6 4902 trades, PF0.827, 0/4 positive; frozen.

- `1h-full-atr-displacement/2026-09-30-early-gate/`: completed 1h body >= pre-signal ATR continuation; strict TP8/SL6 5570 trades, PF0.825, 0/4 positive; frozen.

- `rolling-24h-return-zero-cross/2026-09-30-early-gate/`: causal rolling-24h return sign transition with hourly confirmation; strict TP8/SL6 4295 trades, PF0.822, 0/4 positive; frozen.

- `hourly-volume-retest-standard-exit/2026-09-30-early-gate/`: existing volume/OBV impulse-retest entry under canonical strict TP8/SL6; 1771 trades, PF0.853, 0/4 positive; frozen.

- `qps-term-structure-expansion/2026-09-30-early-gate/`: hourly QPS crossing above prior-day QPS baseline; strict TP8/SL6 4777 trades, PF0.829, 0/4 positive; frozen.

- `dmi-crossover-breakout/2026-09-30-early-gate/`: standalone 1h DMI crossover with strict TP8/SL6; 2990 trades, PF0.774, 0/4 positive; frozen.

- `ema-spread-reacceleration/2026-09-30-early-gate/`: 1h EMA20/50 spread re-acceleration; strict TP8/SL6 3376 trades, PF0.870, 0/4 positive; frozen.

- `price-obv-3h-divergence/2026-09-30-early-gate/`: project-native 3h price-OBV divergence reversal; strict TP8/SL6 4224 trades, PF0.795, 0/4 positive; frozen.

- `quote-volume-weighted-return-pressure/2026-09-30-early-gate/`: 4h quote-volume-weighted return pressure zero-cross; strict TP8/SL6 6967 trades, PF0.824, 0/4 positive; frozen.

- `adx-volatility-expansion-confluence/2026-09-30-discovery/`: ADX20 trend-start × ATR-term expansion confluence; discovery 485 trades PF0.763, 0/6 positive; fresh holdout untouched; frozen.

- `daily-open-reclaim/2026-09-30-early-gate/`: current-day-open reclaim with hourly confirmation; strict TP8/SL6 5700 trades, PF0.777, 0/4 positive; frozen.

- `ttm-squeeze-release/2026-09-30-early-gate/`: standard 1h Bollinger-inside-Keltner release; strict TP8/SL6 1438 trades, PF0.876, 0/4 positive; frozen.

- `daily-inside-day-4h-acceptance/2026-09-30-discovery/`: daily inside-day contraction then 4h acceptance; strict TP8/SL6 975 trades, PF0.858, 0/6 positive; fresh holdout untouched.

- `daily-outside-day-4h-continuation/2026-09-30-discovery/`: directional daily outside-day then 4h acceptance; strict TP8/SL6 323 trades, PF0.907, 2/6 positive, below frequency gate; frozen.

- `daily-nr4-4h-acceptance/2026-09-30-discovery/`: daily NR4 compression then 4h acceptance; strict TP8/SL6 1435 trades, PF0.878, 1/6 positive; fresh holdout untouched.

- `daily-20d-liquidity-sweep-reversal/2026-09-30-discovery/`: daily 20d failed-breakout reversal with 4h midpoint reclaim; strict TP8/SL6 345 trades, PF0.796, 0/6 positive; fresh holdout untouched.

- `daily-open-reclaim/2026-09-30-early-gate/`: current-day open reclaim with strict TP8/SL6; 5700 trades, PF0.777, 0/4 positive; frozen.

- `ttm-squeeze-release/2026-09-30-early-gate/`: strict standard 1h TTM-style squeeze release; 1690 trades, PF0.842, 0/4 positive; frozen.

- `classic-daily-pivot-r1s1-acceptance/2026-09-30-discovery/`: classic prior-day pivot R1/S1 acceptance; 4846 trades, PF0.815, 0/6 positive; fresh holdout untouched, frozen.

- `daily-full-atr-displacement-continuation/2026-09-30-discovery/`: daily body >= pre-signal ATR with 4h acceptance; 550 trades, PF0.743, 1/6 positive; fresh holdout untouched, frozen.

- `funding-price-8h-divergence-transition/2026-09-30-discovery/`: funding-sign crowd invalidated by 8h return zero-cross; 6671 trades, PF0.815, 0/6 positive; fresh holdout untouched, frozen.

- `funding-crowding-acceleration-fade/2026-09-30-discovery/`: three-settlement same-sign funding acceleration with counter-price confirmation; 1091 trades, PF0.796, 0/6 positive; frozen.

- `classic-daily-pivot-r1s1-acceptance/2026-09-30-discovery/`: classic prior-day Pivot R1/S1 acceptance; 4846 trades, PF0.815, 0/6 positive; fresh holdout untouched, frozen.

- `previous-day-midpoint-reclaim/2026-09-30-discovery/`: previous-day range midpoint reclaim; 7106 trades, PF0.826, 0/6 positive; fresh holdout untouched, frozen.

- `three-hour-same-sign-streak/2026-09-30-discovery/`: three consecutive 1h same-sign candles followed by continuation break; 13874 trades, PF0.779, 0/6 positive; fresh holdout untouched, frozen.

- `three-hour-same-sign-streak/2026-09-30-discovery/`: three consecutive 1h same-sign candles followed by continuation break; 13874 trades, PF0.779, 0/6 positive; fresh holdout untouched, frozen.

- `id121-v54-exact-exit/2026-09-30-audit/`: canonical exact TP8/SL6 candidate audit. ID121 PF1.211, 12/15 positive, 0.533/week retained; v54 PF1.163 but v54-only 60 trades PF0.653, so v54 demoted/frozen.

- `id121-exact-exit-early-oot/2021h2-2022-audit/`: ID121 canonical exact-exit early OOT; BTC/ETH 79 trades, PF0.587, 0/2 positive. Confirms strong pre-2023 regime dependence; no tuning performed.

- donchian-20h-midpoint-regime-cross/2026-09-30-discovery/: v110 strict TP8/SL6 discovery; 10146 trades, PF0.815, 0/6 positive; fresh holdout untouched, frozen.

- daily-quote-volume-expansion-continuation/2026-09-30-discovery/: v111 strict discovery; 2743 trades, PF0.814, 0/6 positive; holdout untouched, frozen.

- `donchian-20h-midpoint-regime-cross/2026-09-30-discovery/`: 20h rolling range midpoint cross; 10146 trades, PF0.815, 0/6 positive; fresh holdout untouched, frozen.
- `directional-excursion-imbalance/2026-10-01-discovery/`: four-hour upside/downside excursion-ratio cross; 15268 trades, PF0.811, 0/6 positive; fresh holdout untouched, frozen.

- `v34-pullback-exact-exit-attribution/2026-10-01-audit/`: v34 full vs pullback-disabled paired audit; FULL-only 356 trades PF0.843, negative marginal expectancy, frozen.
- `daily-full-atr-displacement-continuation/2026-09-30-discovery/`: daily full-ATR impulse continuation; 550 trades PF0.743, 1/6 positive, fresh holdout untouched.
- `daily-first-4h-opening-range-breakout/2026-10-01-discovery/`: first-4h UTC opening-range breakout; 8807 trades PF0.799, 0/6 positive, fresh holdout untouched.

- `donchian-20h-midpoint-regime-cross/2026-09-30-discovery/`: prior-20h range midpoint regime cross; 10146 trades, PF0.815, 0/6 positive; fresh holdout untouched, frozen.

- `v67-entry-strict-exit-audit/2026-10-01-discovery/`: exact v67 entry under canonical fixed TP8/SL6; 3830 trades, PF0.854, 0/6 positive; fresh holdout untouched, frozen.

- `v63-entry-strict-exit-audit/2026-10-01-discovery/`: exact v63c entry under canonical fixed TP8/SL6; 4864 trades, PF0.824, 0/6 positive; frozen.

- `bollinger-band-walk-continuation/2026-10-01-discovery/`: two consecutive 1h closes outside Bollinger(20,2), strict TP8/SL6; PF0.875, 0/6 positive; frozen.

- `keltner-20-2-first-breakout-continuation/2026-10-01-discovery/`: strict 1h Keltner(20,2) first-breakout continuation; 4597 trades, PF0.832, 0/6 positive; fresh holdout untouched, frozen.

- `id121-exact-exit-forward/2026-09-01_2026-09-12-1559z/`: canonical ID121 untouched Sep-2026 forward; 9 trades PF0.533, 7SL/2TP; historical 9-trade windows are this bad or worse 15.56% of the time, so retain candidate and continue untouched forward.

- `id121-exact-exit-forward/2026-09-01_2026-09-12-1559z/`: canonical ID121 untouched Sep-2026 forward; 9 trades PF0.533, 7SL/2TP; canonical historical 9-trade windows are this bad or worse 15.56% of the time, so retain the 2024+ candidate and continue untouched forward.

- `12h-breakout-retest-resume/2026-10-01-discovery/`: fresh 12h breakout followed by one-hour retest hold and resume; strict TP8/SL6 PF0.786, 0/6 positive; frozen.

- `id121-exact-exit-robustness/2026-10-01-audit/`: canonical ID121 leave-one-symbol/month-quarter/block-bootstrap audit; LOO PF1.148-1.258, 8/9 positive quarters, month-block bootstrap q05 PF1.015; confirms 2024+ edge with contribution concentration risk.

- `id121-fresh-symbol-holdout/2026-10-01-audit/`: untouched ALGO/INJ/LDO/PENDLE/PYTH holdout for canonical ID121; 238 trades, PF0.848, 0/5 positive; both long and short sides below 1. Cross-symbol generalization failed.

- `daily-volume-shock-one-day-lag/2026-10-01-discovery/`: one-full-day delayed daily quote-volume shock continuation; 2490 trades, PF0.818, 0/6 positive; frozen.

- `daily-volume-shock-one-day-lag/2026-10-01-discovery/`: one-full-day delayed daily quote-volume shock continuation; 2490 trades, PF0.818, 0/6 positive; frozen.

- `v29-entry-strict-exit-audit/2026-10-01-discovery/`: old v29/ID114 entry under exact TP8/SL6; 242 trades, PF0.989, 4/6 positive but only 0.211/week; frozen.

- daily-three-bar-streak-reversal/2026-10-01-discovery/: 3629 trades, PF0.792, 0/6 positive; fresh holdout untouched, frozen.

- macd-1h-signal-cross/2026-10-01-discovery/: standard 1h MACD(12,26,9) signal cross; 4362 trades, PF0.797, 0/6 positive; OOS untouched, frozen.

- v50-entry-strict-exit-audit/2026-10-01-discovery/: unchanged v50 entry with exact TP8/SL6; 142 trades, PF1.066, 4/6 positive but frequency0.227/week; OOS untouched, frozen.

- v52-entry-strict-exit-audit/2026-10-01-discovery/: unchanged v52 entry with exact TP8/SL6; 151 trades PF1.298, long PF1.483, short PF1.240, but frequency0.241/week and 2023 PF0.978; OOS untouched.

- one-two-three-swing-reversal/2026-10-01-discovery/: strict 1h 1-2-3 reversal; 1608 trades PF0.779, 0/6 positive; OOS untouched, frozen.

- funding-zero-cross-momentum/2026-10-01-discovery/: settled funding zero-cross momentum; 369 trades, PF0.718, 0/6 positive; OOS untouched, frozen.

- id121-second-fresh-symbol-holdout/2026-10-01-feasibility/: deterministic second untouched cohort feasibility; only LITUSDT eligible locally, so blocked before returns.

- body-compression-expansion-release/2026-10-01-discovery/: 2-bar body compression then expansion release; 6433 trades, PF0.758, 0/6 positive; OOS untouched, frozen.

- funding-zero-cross-momentum/2026-10-01-discovery/: settled funding sign flip momentum; 369 trades, PF0.718, 0/6 positive; OOS untouched, frozen.

- 1h-three-return-acceleration/2026-10-01-discovery/: 2166 trades, PF0.810, 0/6 positive; OOS untouched, frozen.

- 4h-double-inside-compression-breakout/2026-10-01-discovery/: 49 trades, PF0.441, 0/6 positive, freq0.078/week; OOS untouched, frozen.

- oi-expansion-taker-alignment/2026-10-01-early-gate/: 870 fixed-sample events; 4h +0.0786%, 4/6 positive; failed frozen +0.10% gate, full discovery untouched.

- v65-entry-strict-exit-audit/2026-10-01-discovery/: unchanged v65 entry with strict TP8/SL6; 3557 trades, PF0.861, 0/6 positive; OOS untouched, frozen.

- 4h-double-inside-compression-breakout/2026-10-01-discovery/: nested double-inside compression release; 49 trades, PF0.441, 0/6 positive, freq0.078; OOS untouched, frozen.

- v64-entry-strict-exit-audit/2026-10-01-discovery/: unchanged v64 entry with strict TP8/SL6; 4479 trades, PF0.788, 0/6 positive; OOS untouched, frozen.

- id121-v52-precompression-attribution/2026-10-01-audit/: six-symbol exact-entry attribution; v52 removes 61 ID121 longs at PF0.613 but fails frequency/year gate.
- id121-v52-original-cohort-attribution/2026-10-01-audit/: original 15-symbol attribution; removed 273 ID121 longs are PF1.200 overall and strongly positive in 2025, proving regime-dependent filter effect; v52 frozen.

- `intrahour-partial-qps-breakout/2026-10-01-discovery/`: current partial 1h QPS >= prior-8h completed baseline plus fresh previous-hour high/low 1m breakout; 7188 trades, PF0.833, 0/6 positive; OOS untouched, frozen.

- `1m-qps-record-burst-continuation/2026-10-01-discovery/`: 1m quote-volume rate record over prior 60m, continuation by burst-minute direction; 17440 trades, PF0.813, 0/6 positive; OOS untouched, frozen.

- `intrahour-taker-aligned-breakout/2026-10-01-discovery/`: current partial 1h taker-buy ratio aligned with fresh previous-hour high/low breakout; 13912 trades, PF0.822, 0/6 positive; OOS untouched, frozen.

- `intrahour-range-expansion-breakout/2026-10-01-discovery/`: current partial 1h range exceeds previous completed 1h range plus fresh previous-hour breakout; 10446 trades, PF0.818, 0/6 positive; OOS untouched, frozen. v132/v134/v135 jointly pause intrahour-acceleration branch.

- `binance-loan-asset-removal-short/2023-2024-feasibility/`: official 74-title Loan audit found 10 non-stablecoin removal token-events, but 0/10 had >=2y USD-M history at event time; post-event returns untouched, feasibility frozen.

- `directional-price-impact-asymmetry/2026-10-01-early-gate/`: 24h upside-vs-downside quote-volume price-impact zero-cross; 9907 events, 12h signed mean +0.0004%, 3/6 positive, 2023/2024 sign flip; OOS untouched, frozen.

- `binance-simple-earn-asset-removal-short/2023-2024-feasibility/`: exhaustive 40-title Simple Earn audit found 0 explicit asset-removal events; CYBER notice only discussed possible future removals. Returns untouched, coverage frozen.

- `binance-deposit-withdrawal-resumption-long/2023-2024-feasibility/`: confirmed Binance service-restoration census found only one TORN deposit-resumption event; production eligibility and returns untouched, coverage frozen.

- `token-unlock-supply-shock/2026-10-01-timestamp-recheck/`: fixed 9-event/8-symbol cohort re-audited; public paper CSV lacks the exact UTC timestamp its documentation claims, and complete old timestamp coverage remains unavailable. Exact replay untouched, blocker retained.

- `binance-unplanned-transfer-suspension-short/2023-2024-feasibility/`: Multichain suspension produced 8 affected tokens but 0/8 met >=2y USD-M history at announcement; returns untouched, coverage frozen.

- `binance-corporate-action-support-long/2023-2024-feasibility/`: 14 Binance-native first-support token swap/migration/rebranding events; only MATIC/TOMO/SXP meet >=2y USD-M history + liquidity, so returns untouched and coverage frozen.

- `lagged-quote-volume-price-lead/2026-10-01-early-gate/`: causal 24h lagged quote-volume/next-return covariance predictor; 59539 events, 12h -0.009%, 1/6 positive; OOS untouched, frozen.

- `signed-price-volume-imbalance-regime/2026-10-01-early-gate/`: 24h rolling signed taker quote-volume imbalance zero-cross; 4877 events, 12h +0.0356%, 5/6 positive but economic gate failed; OOS untouched, frozen.

- `binance-convert-asset-removal-short/2023-2024-feasibility/`: 50-title Convert census; no qualifying non-stablecoin crypto asset-removal event, returns untouched, coverage frozen.

- `lagged-taker-flow-price-lead/2026-10-01-early-gate/`: 24-pair causal taker-flow to next-hour-return covariance predictor; 50484 events, 12h +0.0007%, 3/6 positive with year sign flip; OOS untouched, frozen.

- `binance-auto-invest-asset-addition-long/2023-2024-feasibility/`: 13-title Auto-Invest census leaves one 5-token asset-addition batch; market data/returns untouched, coverage frozen.

- `binance-futures-position-limit-adjustment/2023-2024-feasibility/`: one platform-feature title but zero contract-specific position-cap adjustments; returns untouched, coverage frozen.

- `binance-corporate-action-support-long/2023-2024-feasibility/`: 14 Binance first-support swap/migration/rebrand events, only SXP/TOMO/MATIC pass >=2y USD-M + QV; returns untouched, coverage frozen.

- `binance-pay-asset-addition-long/2023-2024-feasibility/`: Binance Pay token-level asset-addition coverage audit; 3 Pay titles, 0 qualifying token events; no market data/returns read, frozen.

- `directional-taker-flow-energy-asymmetry/2026-10-01-early-gate/`: v139 directional second-moment taker-flow energy; 4742 events, 12h -0.0024%, 2/6 positive; strict/OOS untouched, frozen.

- `funding-sign-persistence-reversal/2026-10-01-early-gate/`: first 3-settlement regular-cadence same-sign funding streak, contrarian direction; 580 events, 12h +0.0168%, 3/6 positive, year sign flip; OOS untouched, frozen.

- `conditional-return-sign-markov/2026-10-01-early-gate/`: v140 24-transition conditional sign model; 33866 events, 12h -0.0065%, 2/6 positive; strict/OOS untouched, frozen.

- `range-overlap-value-migration/2026-10-01-early-gate/`: v141 fresh below-baseline 1h range-overlap migration; 26175 events, 12h -0.0266%, 2/6 positive; strict/OOS untouched, frozen.

- `binance-holder-airdrop-support-long/2023-2024-feasibility/`: 12 Binance airdrop/distribution title candidates collapse to one qualifying external holder-airdrop event (CHZ); production eligibility/returns untouched, coverage frozen.

- `binance-p2p-asset-addition-long/2023-2024-feasibility/`: 29 P2P titles collapse to one ordinary-crypto first-addition event (WLD); market eligibility/returns untouched, coverage frozen.

- `binance-token-trading-fee-change/2023-2024-feasibility/`: eight raw independent token-level fee-policy batches; BETH batch lacks USD-M, so max eligible batches=7<8; no returns read, frozen.

- `order-flow-coherence/2026-10-01-early-gate/`: trailing-60m net/gross taker-flow coherence cross; 33067 events, 12h -0.0751%, 0/6 positive; OOS untouched, frozen.

- `binance-token-burn-support-long/2023-2024-feasibility/`: four burn-keyword titles are all completed recurring BNB Auto-Burn reports; zero qualifying first-support events, returns untouched, frozen.

- `binance-staking-asset-addition-long/2023-2024-feasibility/`: 19 staking titles collapse to one qualifying legacy DeFi-Staking token addition (CVX); market eligibility/returns untouched, coverage frozen.

- `binance-proof-of-reserves-asset-addition-long/2023-2024-feasibility/`: exact 2023-2024 PoR audit found one independent asset-addition batch; market eligibility/returns untouched, coverage frozen.

- `snapshot-governance-approval-long/2023-2024-feasibility/`: 43 clean binary proposals; 41 positive wins but all are LDO in one Snapshot space, so cross-symbol/space coverage fails before market-data access; frozen.

- `interpretable-two-split-fast-move/2026-10-01-discovery/`: fixed depth-2 symmetric tree on nine native features; best 2023 leaf reward +0.299 < +1.0 train gate, selected leaves=0; 2025/2026 unread, frozen.

- `funding-volatility-expansion-fade/2026-10-01-early-gate/`: v143 recent-3 funding std vs prior-21 baseline cross; 863 events, 12h +0.2197%, 5/6 positive, but 2023 negative vs strong 2024; OOS unread, frozen as regime-dependent.

- `binance-borrow-interest-rate-adjustment/2023-2024-feasibility/`: 1071-title official audit found only one generic dynamic-interest policy notice and zero token-level adjustment events; returns untouched, coverage frozen.

- `confirmed-williams-fractal-breakout/2026-10-01-early-gate/`: v144 standard causal 5-bar swing breakout; 10035 events, 12h -0.0776%, 0/6 positive, both years negative; OOS unread, frozen.

- `quote-ease-of-movement-14/2026-10-01-early-gate/`: v145 standard 14-bar quote-EOM zero-cross; 12085 events, 12h -0.0201%, 2/6 positive, annual sign flip; OOS unread, frozen.

- `binance-portfolio-margin-collateral-asset-addition-long/2023-2024-feasibility/`: official 2023-2024 audit found zero qualifying ordinary-crypto Portfolio Margin asset additions; returns untouched, coverage frozen.

- `defillama-protocol-tvl-momentum/2026-10-02-discovery/`: 7d protocol-TVL growth zero-cross; corrected Binance Vision discovery 1479 events, 7d -0.779%, 4/17 positive; OOS 2025+ unevaluated, frozen.

- `quote-volume-weighted-close-20h-reclaim/2026-10-01-discovery/`: v146 QVWC20 reclaim + trigger-bar breakout; strict Engine 6202 trades, PF0.823, 0/6 positive; OOS unread, frozen.

- `top-trader-capital-headcount-skew/2026-10-02-early-gate/`: v147 top-position/top-account skew zero-cross; 1562 events, 12h -0.027%, 4/6 positive, both years negative; OOS unread, frozen.

- `usdm-premium-zero-cross-carry-reversion/2026-10-02-early-gate/`: v148 natural premium sign-cross carry/reversion; 14901 events, 12h +0.074%, 5/6 positive, both years positive but economic gate failed; OOS unread, frozen.

- `top-trader-capital-headcount-skew/2026-10-02-early-gate/`: v147 hourly Top Position / Top Accounts skew zero-cross; 1563 events, 12h -0.0395%, 3/6 positive; OOS unread, frozen.

- `defillama-chain-tvl-momentum/2026-10-02-discovery/`: v148 whole-chain 7d TVL-growth zero-cross; 1240 events, 7d -0.158%, 5/13 positive, 2023/2024 sign flip; OOS unread, frozen.

- `spot-perp-turnover-dominance-emergence/2026-10-02-early-gate/`: v149 natural Spot QV24 > Perp QV24 dominance cross; complete data but 0/6 symbols ever trigger in 2023-2024; structural coverage failure, frozen.

- `top-position-depth-fragility/2026-10-02-discovery/`: v149 Top-Position vs ±1% depth structural fragility; 2023 Stage A 10153 events, 12h +0.0146%, far below gate; 2024/2025+ unread, frozen.

- `binance-usdm-insurance-fund-balance/2026-10-02-feasibility/`: current public insuranceBalance is snapshot-only; historical parameters do not recover 2023-2024 state; returns untouched, data-history blocked.

- `top-trader-capital-headcount-skew/2026-10-02-early-gate/`: v147 hourly zero-cross of top-position vs top-account ratio; 1561 events, 12h -0.0365%, 4/6 positive; OOS unread, frozen.

- `binance-trading-bots-support-long/`: existing-pair Trading Bots support isolated from new-pair announcements; 26 eligible events/13 tokens/13 batches, discovery 12h -1.425%, 4/13 tokens positive; frozen, OOS unread.

- `binance-dual-investment-asset-addition-long/2023-2024-feasibility/`: 61 Dual Investment titles but only 4 true new-underlying batches; coverage blocked before market data/returns.

- `binance-network-integration-expansion-long/2023-2024-feasibility/`: 26 official deposit-network integration events; strict eligibility leaves 8 events/8 batches but only 7 unique tokens, so coverage blocked before returns.

- `binance-earn-wednesday-flexible-activation-long/`: 74 eligible events / 38 tokens; 12h -0.7768%, breadth failed, OOS unread, frozen.
- `dollar-funding-burden-expansion-fade/2026-10-02-early-gate/`: v152 burden expansion fade; 3045 events, 12h +0.0892%, below +0.20% gate, frozen.

- `binance-earn-wednesday-flexible-activation-long/`: state-based activation into weekly Simple Earn Flexible offers; 74 eligible events/38 tokens/43 batches, discovery 12h -0.777% and 15/38 token breadth; frozen, 2025+ unread.

- `binance-liquid-swap-pool-removal-short/2023-2024-feasibility/`: 137 eligible events / 50 tokens / 11 batches, but Binance retired Liquid Swap in Jan-2024; no returns read, production-applicability blocked.

- `oi-funding-crowding-build-fade/2026-10-02-early-gate/`: v153 8h OI expansion transition faded by funding sign; 3379 events, 12h -0.0805%, 1/6 positive; OOS unread, frozen.

- `binance-network-route-support-cessation-short/2023-2024-feasibility/`: 12 token-events but only 3 independent batches; no market returns read, coverage blocked.

- `funding-flat-zone-feedback-activation/`: v154 exchange-defined 0.01% funding flat-state exit; discovery +0.2058%, 2025 OOS1 +0.2317%, but 2026 OOS2 -0.1872% and 0/6 positive; frozen before strict Engine.

- `binance-liquid-swap-pool-removal-short/`: 137 eligible events / 50 tokens; chronological Stage A 66 events, 12h -0.5128%, 1/6 batches positive; Stage B unread, frozen.

- `funding-premium-feedback-failure-fade/2026-10-02-early-gate/`: v155 funding payment fails to remove basis sign; 1451 events, 12h +0.1232%, 5/6 positive, below +0.20% gate; OOS unread, frozen.

- `cci20-classic-threshold-breakout/2026-10-02-early-gate/`: v156 classic CCI20 +/-100 cross; 9990 events, 12h -0.0312%, 2/6 positive; frozen.

- `binance-liquid-swap-pool-removal-short/`: 137 eligible events / 50 tokens / 11 batches; event-weighted 12h +0.531% but batch-equal +0.482% and only 5/11 batches positive; exact/OOS not run, frozen.

- `coinmetrics-realized-cap-growth/feasibility/`: direct CapRealUSD community access returns 403 while control metrics work; no returns read, blocked.

- `premium-range-record-expansion-fade/2026-10-02-early-gate/`: v157 premium-range 24h record expansion fade; 3460 events, 12h +0.0359%, 3/6 positive, 2024 negative; frozen.

- `binance-spot-tick-size-cross-market/2021-2022-increase-short-oot/`: 161 untouched increase events reconstructed, but 0 satisfy event-time >=2y USD-M eligibility; returns unread, validation unavailable.

- `binance-spot-step-size-adjustment/2021-2024-feasibility-correction/`: corrected census finds 3 official articles / 229 raw events, but only SOL/LINA/TRB eligible across 2 articles; no returns read, coverage blocked.

- `binance-futures-tick-size-adjustment/2023-2024-missing-batches-untouched/`: 10 production-eligible untouched events from 5 previously omitted articles; exact 10/10 SL, PF0, permanently frozen.

- `elite-vs-crowd-account-skew/2026-10-02-early-gate/`: v158 TopTrader-account vs global-account zero-cross; 3917 events, 12h +0.0958%, 5/6 positive but below economic gate; frozen.

- `funding-elite-crowd-disagreement-fade/2026-10-02-early-gate/`: v159 funding × elite/global account disagreement; 1282 events, 12h +0.1005%, 5/6 positive but below economic gate; frozen.

- `binance-spot-tick-size-cross-market/2021-2022-increase-short-oot/`: untouched validation for generated increase→SHORT hypothesis; 161 raw events but 0 production-eligible under >=730d USD-M rule, returns unread, coverage blocked.

- `binance-portfolio-margin-collateral-ratio/2024-feasibility-correction/`: corrected coverage 8 events / 8 assets / 4 articles, returns unread.
- `binance-portfolio-margin-collateral-ratio/2024-corrected-exact-discovery/`: exact 8 trades PF1.356, net +9.81%, but only 2/4 positive article batches vs 3/4 gate; frozen.

- `coinmetrics-net-supply-growth/2026-10-02-feasibility/`: SplyCur coverage 8/10 variable mature assets, returns unread, gate passed.
- `coinmetrics-net-supply-growth/2026-10-02-discovery/`: 7d supply-growth acceleration zero-cross; 946 events, 7d -0.2094%, 3/8 positive, both years negative; frozen.

- `binance-spot-tick-size-cross-market/2021-2022-increase-short-oot/`: untouched validation of increase->SHORT; 161 raw events but 0 satisfy >=730d USD-M history; no returns read, validation unavailable.

- `binance-simple-earn-baseline-apr-adjustment/2023-2024-feasibility/`: 25 official APR/yield articles audited; 0 qualifying token-specific durable baseline APR changes; returns unread, frozen.

- `record-arrival-imbalance/2026-10-02-early-gate/`: v160 24h new-high/new-low record-arrival zero-cross; 12931 events, 12h -0.0016%, 1/6 positive, frozen.

- `binance-futures-copy-trading-symbol-support/2023-2024-feasibility/`: 10 official Copy Trading titles, 0 token-specific USD-M support changes; returns unread, frozen.

- `id121-regime-post-breakout-inside-continuation/2026-10-01-discovery/`: v114 exact 46 trades, PF1.205 but 3/6 positive and 0.040/week; fresh holdout unread, frozen.

- `binance-simple-earn-baseline-apr-adjustment/2023-2024-feasibility/`: 25 official APR/yield candidates audited; 0 token-specific durable baseline APR changes; no market returns read, frozen.

- `id121-mechanism-ablation/2026-09-30-audit/`: single-component diagnostic ablation on original 15 symbols; removing daily regime/freshness/funding/impulse/4h strength materially degrades PF/breadth; no-chase least important but not promotable; canonical ID121 retained.

- `spot-perp-basis-momentum/2026-10-02-early-gate/`: v161 24h spot-perp basis-change zero-cross continuation; 42,516 events, 12h -0.0159%, 0/6 positive; OOS unread, frozen.

- `github-core-release-cadence/2026-10-02-discovery/`: v162 release-cadence zero-cross; 174 events/10 symbols, 12h +0.2268% but 2023 negative and below +0.25% gate; OOS unread, frozen.

- `spot-perp-volume-share-expansion-confirmation/2026-10-02-early-gate/`: v163 derivatives-participation expansion confirmation; 2066 events, 12h -0.0602%, 1/6 positive; frozen.

- `large-trade-tail-imbalance/feasibility/`: novel tail-trade imbalance idea blocked by zero local trade history and very large Binance Vision aggTrades archive requirement; no returns read.

- `github-core-release-cadence/2026-10-02-discovery/`: v162 release-cadence zero-cross; 174 eligible events / 10 symbols, 12h +0.2268% but 2023 negative; OOS unread, frozen.

- `perp-excess-volatility-fade/2026-10-02-early-gate/`: v164 fresh perp>spot realized-vol transition faded against 24h price direction; 3606 events, 12h +0.1082%, below +0.20% gate; OOS unread, frozen.

- `id121-regime-liquidity-sweep-reclaim/2026-10-02-discovery/`: v164 opposite-side 12h sweep/reclaim inside fixed ID121 regime; 17 trades, PF0.356, 2/6 positive, very low frequency; frozen.

- `binance-futures-price-protection-adjustment/feasibility/`: official 2023-2024 title corpus has 0 price-protection/price-band adjustment candidates; no returns read.

- `spot-perp-tracking-error-expansion-fade/2026-10-03-early-gate/`: v166 fresh 24h residual-tracking-error expansion fade; 2893 events, 12h +0.0524%, 3/6 positive; OOS unread, frozen. Spot–Perp state branch paused.

- `oi-market-cap-leverage-expansion-fade/2026-10-03-early-gate/`: v167 OI-notional/market-cap expansion fade; 240 events, 7d -1.7403%, 1/6 positive; OOS unread, frozen.

- `futures-turnover-market-cap-expansion-confirmation/2026-10-03-early-gate/`: v168 turnover/market-cap expansion confirms 7d trend; 602 events, 7d +0.0103%, 2023/2024 sign flip; OOS unread, frozen.

- `binance-spot-excess-volatility-fade/2026-10-03-early-gate/`: v169 Binance Spot vs composite Index Price RV up-cross fade; 3841 events, 12h +0.0273%, below gate; OOS unread, frozen.

- `binance-deposit-withdrawal-support-cessation/2023-2024-feasibility/`: only 4 possible ordinary-token batches after stablecoin exclusion; no returns read, frozen.
- `binance-pool-merged-mining-support/2023-2024-feasibility/`: only 4 independent merged-mining support batches; no returns read, frozen.

- `quote-volume-persistence-emergence-momentum/2026-10-03-early-gate/`: v170 48h log-QV autocorrelation zero-up-cross; only 25 events / 0.0399 per symbol-week, frequency blocked; OOS unread, frozen.

- `return-future-volatility-asymmetry-reversal/2026-10-03-early-gate/`: v171 rolling leverage-effect sign transitions; 4653 events, 12h +0.0764%, below gate; OOS unread, frozen.

- `taker-top-position-skew/2026-10-03-early-gate/`: v172 hourly taker-vs-top-position skew zero-cross; 29336 events, 12h +0.0014%, no edge; OOS unread, frozen.

- `volatility-growth-acceleration-regime/2026-10-03-early-gate/`: v173 second-difference of non-overlapping 12h realized volatility; 12184 events, 12h +0.0114%, no edge; OOS unread, frozen.

- `defillama-protocol-open-interest-momentum/2026-10-03-feasibility/`: 137 free OI protocol histories audited; only DYDX plausibly satisfies same-name >=2y Binance USD-M coverage; no returns read, frozen.

- `defillama-chain-derivatives-open-interest/2026-10-03-feasibility/`: historical chain OI rebuilt from protocol breakdowns; only ETH/BNB/SOL/INJ satisfy frozen coverage, no returns read, frozen.

- `taker-flow-volatility-lead-asymmetry-reversal/2026-10-03-early-gate/`: v175 taker-flow to next-hour variance sign transitions; 4880 events, 12h -0.0156%, 2/6 positive; OOS unread, frozen.

- `github-core-issue-backlog/2026-10-03-discovery/`: v174 issue-arrival pressure; 769 eligible signals, 756 replayed, 7d +0.2229% but 2023 negative; LONG-only split is post-hoc, OOS unread, frozen.

- `aggressor-activity-coupling-regime/2026-10-03-early-gate/`: v176 taker-flow vs quote-volume-change correlation zero-cross; 4977 events, 12h +0.0466%, below gate; OOS unread, frozen.

- `taker-flow-volatility-lead-asymmetry-reversal/2026-10-03-early-gate/`: v172 taker-flow to next-hour variance sign-transition reversal; 4880 events, 12h -0.0156%, 2/6 positive; OOS unread, frozen.

- `github-core-issue-resolution-balance/`: v177 7d issue closures-minus-creations zero-cross; 883 eligible / 865 replayed events, 7d -0.1927%, 4/10 positive; OOS unread, frozen.

- `carry-adjusted-momentum-zero-cross/2026-10-03-early-gate/`: v178 24h price return net of realized funding, zero-cross momentum; 10287 events, 12h -0.0448%, 0/6 positive; OOS unread, frozen.

- `return-energy-concentration-regime/2026-10-03-early-gate/`: v179 24h squared-return HHI regime; 9361 events, 12h -0.0238%, 2/6 positive; OOS unread, frozen.

- `github-core-push-activity/2023-2024-feasibility/`: GH Archive PushEvent fixes visibility-time semantics, but no server-side historical filter is available locally; bulk two-year archive scan is storage-blocked, no returns read.

- `range-occupancy-regime-cross/2026-10-03-early-gate/`: v180 24h path-occupancy majority cross; 8808 events, 12h +0.0036%, 3/6 positive; OOS unread, frozen.

- `activity-volatility-coupling-regime/2026-10-03-early-gate/`: v181 rolling activity-change vs squared-return coupling; 453 events, 12h -0.0021%, annual sign flip; OOS unread, frozen.

- `per-trade-volatility-impact-regime/2026-10-03-early-gate/`: v182 24h realized variance per TradeCount regime; 8137 events, 12h +0.0327%, below +0.20% gate; OOS unread, frozen.

- `price-monotonicity-regime-cross/2026-10-03-early-gate/`: v183 24h Spearman time-vs-close rank zero-cross; 4613 events, 12h +0.0906%, 5/6 positive, below +0.20% gate; OOS unread, frozen.

- `notional-vs-trade-arrival-concentration/2026-10-03-early-gate/`: v184 QuoteVolume-HHI vs TradeCount-HHI regime; 1068 events, 12h -0.1091%, 3/6 positive; OOS unread, frozen.

- `participation-ticket-size-coupling-regime/2026-10-03-early-gate/`: v185 TradeCount-change vs average-ticket-change coupling; 295 events, 12h +0.0239%, 3/6 positive; OOS unread, frozen.

- `id121-no-no-chase-forward/2026-09-temporal-holdout/`: preregistered untouched Sep-2026 sanity window; local 1m/funding coverage ends around Sep 12-13, so no strategy returns were read and no backfill was performed.

- `carry-adjusted-momentum-zero-cross/2026-10-03-early-gate/`: v178 24h price return net of realized funding zero-cross; 10287 events, 12h -0.0448%, 0/6 positive; OOS unread, frozen.

- `rolling-4h-extreme-order-continuation/2026-10-03-early-gate/`: v186 rolling four-hour high/low order sign transitions; 26136 events, 12h -0.0124%, 1/6 positive; OOS unread, frozen.

- `taker-global-account-skew/2026-10-03-early-gate/`: v187 Taker-vs-Global-account ratio zero-cross; 13224 events, 12h +0.0085%, 5/6 positive but far below economic gate; positioning pairwise audit closed.
