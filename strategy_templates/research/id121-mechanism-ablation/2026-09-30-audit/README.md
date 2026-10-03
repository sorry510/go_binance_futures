# ID121 Mechanism Ablation Audit

Purpose: explain which frozen ID121 components contribute on the original 15-symbol cohort. This audit is diagnostic only; it is not a candidate-selection or retuning stage.

Exact base:
- 773 trades, PF **1.2112**, 12/15 positive, frequency **0.5327/symbol/week**.
- 2024/2025/2026 PF: **1.2413 / 1.1791 / 1.2457**.

Single-component removals:
- no daily regime: 1,502 trades, PF **1.0185**, 6/15 positive; 2024 PF 0.9254, 2025 PF 0.9917.
- no freshness: 1,034 trades, PF **1.0301**, 6/15 positive.
- no funding: 1,261 trades, PF **1.0582**, 8/15 positive.
- no impulse: 1,193 trades, PF **1.0656**, 9/15 positive.
- no 4h strength: 1,121 trades, PF **1.0746**, 7/15 positive; 2024 PF 0.9695.
- no no-chase: 1,112 trades, PF **1.1423**, 11/15 positive; 2024/2025/2026 PF 1.1171/1.1129/1.1908.

## Interpretation

The higher-frequency variants are not free improvements. Removing daily regime, freshness, funding, impulse, or 4h strength increases trade count substantially but compresses PF toward 1.0 and weakens cross-symbol breadth. Daily regime and freshness are the largest degradations by PF; funding, impulse and 4h strength are also material.

The no-chase condition is the least essential of the six: removing it raises frequency to 0.766/week while retaining PF 1.142 and 11/15 positive. However, this is an in-cohort diagnostic observation, not an untouched candidate. It cannot be promoted or retuned from this audit.

Decision: retain canonical ID121 unchanged. **Freeze the “remove filters to gain frequency” route.** Any simplified ID121 candidate would require a separately preregistered untouched validation universe/time window; this audit cannot supply it.
