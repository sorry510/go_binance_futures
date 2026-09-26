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
