# v80 Previous-Week Range Acceptance — 2026-09-30 Early Gate

Hypothesis: a completed 1h candle that crosses and closes beyond the prior completed weekly high/low represents acceptance outside the prior-week value range. If price then continues through the signal-hour extreme, continuation may be strong enough for the project's fixed TP8/SL6 execution.

This study is project-native: only existing Binance USD-M 1m replay, aggregated 1h/1w klines, NowPrice and the current Backtest Engine are used. No external data pipeline is required.

Discovery/early-gate universe is frozen before results: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT, 2023-01-01 through 2026-09-01. Fixed execution: leverage 4, TP 8%, SL 6%, fee 0.0005/side, slippage 5bps/side, single position.

Promotion gate: overall normalized PF >= 1.15, >=3/4 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. If the gate fails, freeze the family without changing weekly window, acceptance definition, confirmation rule, or direction.

## Decision

**DATA BLOCKED BEFORE RETURNS.** The current project supports a 1w backtest interval in code, but the existing local historical store does not contain a complete BTCUSDT 1w series. Dataset build failed before Engine.Run, so no return/PF evidence was inspected. Per project rules no historical backfill or DB write was performed. The calendar-week family is frozen in its current form; a separate rolling-7d family may be tested using existing 1d data.
