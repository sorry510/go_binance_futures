# Protocol

Normalized permutation entropy on the latest 48 hourly returns, ordinal order 3. Trigger first H<=0.85, re-arm only at H>=0.90. Direction is the sign of the already-completed trailing 12h return. Entry proxy is next 1h open; inspect signed 1h/4h/12h movement.

No entropy/window/order/direction search after observing results.
