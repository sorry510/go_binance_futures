# Protocol

1. Freeze the v80 JSON before reading any return output.
2. Use only current project Backtest Engine and existing historical data; do not backfill or write DB for this study.
3. Prior completed week is `kline_1w[1]`.
4. LONG acceptance: completed 1h opens at/below prior-week high and closes above it; current price must exceed that signal hour's high.
5. SHORT is symmetric at the prior-week low.
6. Close rules only invalidate acceptance; fixed engine TP8/SL6 remains authoritative.
7. Early gate is evaluated on BTC/ETH/BNB/XRP only.
8. If gate fails, do not tune 1w/1h horizons, add trend/volume filters, reverse direction, or expand symbols.
9. If gate passes, freeze parameters and proceed to an expansion set before any holdout.
