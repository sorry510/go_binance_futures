# GitHub Core Commit Activity — Feasibility

Hypothesis: sustained core-repository commit activity may proxy protocol development intensity.

Decision: timestamp-semantics blocked before return inspection. Git/GitHub commit author/committer timestamps do not reliably represent when a commit became visible on the repository default branch. Replaying monthly counts by those timestamps can introduce look-ahead when commits are merged/pushed later.

A valid revisit requires an event-time source such as GH Archive PushEvent history (or another auditable default-branch visibility timestamp). Do not backtest from git log timestamps.