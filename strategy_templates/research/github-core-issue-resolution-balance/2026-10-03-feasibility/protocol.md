# v177 GitHub Core Issue-Resolution Balance — Feasibility Protocol

1. Reuse exactly the frozen 10 core repositories from v174; no repo substitution.
2. v174 studied issue arrivals only. v177 studies issue resolution versus arrival and is therefore a separate operational-health mechanism.
3. For each repository, use GitHub Search `is:issue closed:<UTC range>`; exclude Pull Requests. The causal timestamp is GitHub server-side `closed_at`.
4. Audit 2022 as warmup and 2023-2024 as discovery-source years. No token return may be read during feasibility.
5. Coverage gate: at least 8/10 repositories must have >=100 closed issues across 2023-2024.
6. GitHub Search's 1,000-result cap must be respected. If a yearly partition exceeds 1,000, split that year mechanically by calendar month before any market return is read.
7. If coverage passes, open the separately frozen discovery stage. Do not use issue titles, labels, bodies, severity, comments or NLP.
