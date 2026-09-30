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
