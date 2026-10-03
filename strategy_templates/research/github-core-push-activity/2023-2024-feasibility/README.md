# GitHub Core Push Activity — 2023-2024 Feasibility

Goal: reopen the previously blocked GitHub commit-activity hypothesis with a correct point-in-time visibility timestamp.

## Correct source semantics

GH Archive records the GitHub public event stream and exposes `PushEvent.created_at` in hourly archives. This fixes the prior problem where Git commit author/commit timestamps do not establish when the commit became visible on the tracked public repository.

## Access audit

- Required historical discovery window: 2023-2024 for the frozen core-repository universe.
- Direct GH Archive access is bulk hourly JSON.gz. Filtering by repository happens only after the archive file is downloaded.
- This Mac currently has no `bq` or `gcloud` CLI and no configured BigQuery identity/project.
- Therefore there is no available server-side query path to restrict the public GH Archive dataset to the small repository set before transfer.
- Downloading/scanning the full global 2023-2024 hourly archive is incompatible with the project's storage/data-budget constraints.
- GitHub's normal Events API is suitable for prospective collection but does not provide the required two-year historical event stream.

## Decision

**Historical-data-access / storage blocked. No token return was read.**

This is not an alpha failure. The family may be reopened only if a server-side GH Archive query path (for example an authorized BigQuery project or an equivalent indexed archive) becomes available. Do not fall back to commit author dates, commit dates, current repository history, or a partial recent event feed as a substitute for point-in-time historical visibility.
