# Protocol

1. External event source is GitHub Releases from a fixed canonical core-node/client repository per token.
2. A valid event is a Release object with `draft=false`, `prerelease=false`, and a non-null `published_at`. Tags/commits are not substituted for repositories that do not use Releases.
3. Multiple stable Releases for the same symbol on one UTC date are collapsed to the earliest `published_at` before any return inspection.
4. Fixed direction is LONG: a stable core software release is treated as a protocol-delivery catalyst. Entry is the next complete Binance USD-M 1h open after `published_at`.
5. Eligibility at the event timestamp: actual Binance USD-M history >=2 years and trailing complete 24h QuoteVolume >=5m USDT.
6. Discovery is 2023-01-01 through 2024-12-31 only. Endpoint diagnostics are signed 1h/4h/12h returns from the frozen entry.
7. Frozen discovery gate: >=80 eligible events, >=8 triggering symbols, aggregate 12h mean >=+0.25%, >=60% of symbols have positive mean12, and aggregate 2023 and 2024 mean12 are both positive.
8. OOS 2025-01-01 through 2026-08-31 stays return-untouched unless the discovery gate passes. Exact 1m TP8/SL6 with fees/slippage/funding is allowed only after OOS passes.
9. No post-hoc SHORT reversal, repo substitution, tag-based backfill, release-title filtering, semver filtering, or event-window tuning after returns are seen.
