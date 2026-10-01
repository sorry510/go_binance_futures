# Protocol

1. Freeze JSON before reading returns.
2. Previous completed daily bar must be NR4: its range <= ranges of the three prior completed daily bars.
3. LONG: completed 4h opens at/below NR4 high and closes above it; current price must break that 4h high.
4. SHORT is symmetric at NR4 low.
5. Exact exits: ROI >= 8 || ROI <= -6.
6. Discovery only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. Failure => no NR7, no range-ratio threshold, no EMA/volume/funding filters, no reversal.
8. Pass => freeze and evaluate preregistered fresh holdout.
