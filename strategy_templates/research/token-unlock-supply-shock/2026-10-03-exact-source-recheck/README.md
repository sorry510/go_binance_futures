# Token Unlock Supply-Shock — 2026-10-03 Exact-Source Recheck

The frozen production-eligible cohort remains unchanged at 9 events / 8 symbols:
FTM 2024-03-01, IMX 2024-02-14, AXS 2024-10-25, RON 2024-10-28,
APT 2024-11-12 and 2025-01-12, ID 2025-03-22, FET 2025-03-14, SEI 2025-08-15.

No token return was read in this recheck.

## New source audit

Current public search confirms that exact unlock timestamps do exist in commercial/event datasets:
- CryptoRank's unlock API response schema contains a millisecond `time` field, but unlock-calendar history is an authenticated/pro product.
- Tokenomist's API is authenticated with an x-api-key and full historical release-event access is premium.
- CoinMarketCal can preserve exact UTC times from token.unlocks.app for selected historical events.

However, the frozen cohort still cannot be reconstructed 9/9 from one accessible auditable source. More importantly, independent current historical pages disagree with the frozen author-curated CSV on some event dates:
- the frozen CSV has AXS 2024-10-25, while a current tokenomics historical schedule lists the major AXS unlock on 2024-10-14;
- the frozen CSV has APT 2024-11-12, while public event calendars place the November 2024 APT unlock around Nov 11/12 depending on source/time-zone representation.

This means assigning a time to the frozen CSV date would create false precision before the actual event identity/date is reconciled.

## Decision

**Timestamp/source-identity coverage remains blocked.**
- Do not shrink to events for which an exact time can be found.
- Do not mix different vendors event-by-event.
- Do not infer midnight/noon from recurring schedules.
- Do not silently replace frozen event dates after seeing alternative histories.
- Do not read any new price/PF/TP-SL outcome.

The family may reopen only with a complete, consistently sourced, auditable 9/9 historical dataset (or a future genuinely point-in-time cohort collected prospectively).
