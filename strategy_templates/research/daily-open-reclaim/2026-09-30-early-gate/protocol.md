# Protocol

1. Freeze entry and exit JSON before returns.
2. Reference price = current daily bar Open[0].
3. LONG trigger: completed 1h opens at/below daily open and closes above it; current price must break that 1h high.
4. SHORT is symmetric.
5. Exact close rule on both sides: ROI >= 8 || ROI <= -6.
6. Core-4 early gate only.
7. Failure => no previous-day-open variant, no EMA/ADX/QPS/funding filters, no direction reversal.
8. Pass => freeze parameters and expand symbols before holdout.
