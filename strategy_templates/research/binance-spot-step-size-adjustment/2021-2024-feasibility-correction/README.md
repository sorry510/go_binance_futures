# Binance Spot Step-Size Adjustment — 2021-2024 Feasibility Correction

This audit supersedes the 2026-09-29 feasibility note that incorrectly concluded the 2021-08 mixed Tick Size + Step Size article was the only 2021-2024 Step Size notice.

Corrected official census:
- 2021-08-12: `Updates on Tick Size and Step Size for Spot Trading Pairs`.
- 2024-04-29: `Updates on Step Size for Spot Trading Pairs`.
- 2024-06-19: `Updates on Step Size for Spot Trading Pairs`.

Only the Step Size / quantity-increment tables are parsed. Tick Size rows in the mixed 2021 notice are excluded. BASE/USDT rows are used; leveraged tokens and stable/fiat bases are excluded.

Pre-registered mapping before returns:
- Step-size decrease -> LONG USD-M.
- Step-size increase -> SHORT USD-M.

Raw audit:
- **229 token-events / 223 unique tokens / 3 official articles**.
- No post-event return was read.

Production eligibility:
- same-name USD-M history >=730 days at effective time;
- prior 24 complete 1h QuoteVolume >=5M USDT.

## Final result

Only **3 events / 3 tokens / 2 independent articles** are eligible:
- SOL — 2024-04-29 — 0.01 -> 0.001 — LONG.
- LINA — 2024-06-19 — 0.01 -> 1 — SHORT.
- TRB — 2024-06-19 — 0.01 -> 0.001 — LONG.

The other 226 events fail the fixed two-year USD-M history rule. No eligible event fails the QV threshold.

Coverage gate requires >=8 events, >=8 unique tokens and >=3 independent eligible articles, so it fails before any return is inspected.

Decision: **coverage blocked / freeze**. The old “only one article” rationale is superseded, but the no-return conclusion remains. Do not split effective sub-times inside one article into independent batches, lower the two-year rule, merge Tick Size changes, or inspect the three eligible returns.
