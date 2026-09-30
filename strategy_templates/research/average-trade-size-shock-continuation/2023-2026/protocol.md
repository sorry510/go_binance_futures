# Protocol

Use log average trade notional = log(quote_volume/trade_count). Z-score versus the previous 168 completed 1h bars. Trigger first z>=3; re-arm below 1. Direction follows the completed trigger-hour return. Enter at next 1h open and inspect signed 1h/4h/12h movement.

This is a standalone nonlinear tail test, distinct from the earlier linear discovery feature. No neighboring z/window/reversal search.
