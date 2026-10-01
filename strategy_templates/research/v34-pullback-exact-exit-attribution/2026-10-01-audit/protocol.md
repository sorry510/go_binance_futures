# Protocol

1. Freeze source v34 entry code.
2. FULL changes only both CLOSE rules to ROI >= 8 || ROI <= -6.
3. BASE is identical to FULL except funding_pullback_resume_short_v34 is disabled.
4. Run both variants over the same 15-symbol eligibility schedule used by the ID121 exact-exit audit.
5. Compare aggregate PF, by-symbol/b-year results, and paired trade cohorts.
6. Pullback mechanism is considered useful only if its marginal cohort is positive expectancy and does not degrade aggregate quality.
7. No funding threshold, EMA, ADX, RSI, pullback touch, or resume condition may be changed after results.
8. No DB writes, commit, or push.
