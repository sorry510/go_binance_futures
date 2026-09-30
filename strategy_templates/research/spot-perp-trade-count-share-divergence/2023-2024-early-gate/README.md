# Spot-vs-Perp Trade-Count Share Divergence — 2023–2024 Early Gate

Hypothesis: an extreme increase in USD-M perpetual trade-arrival intensity relative to spot reflects leveraged crowding and should mean-revert; an extreme spot-side dominance should predict LONG USD-M.

Signal: hourly log(perp trade_count / spot trade_count), standardized against the prior 720 aligned hours. First |z| >= 3 after re-arm at |z| < 1. z > 3 => SHORT; z < -3 => LONG. Entry at next 1h open.

Frozen early gate: BTC/ETH/BNB/XRP, 2023–2024; require 12h signed mean >= +0.10% and at least 3/4 symbols positive.

Result: 341 events. 1h +0.0116%, 4h +0.0362%, 12h -0.0888%. Only ETH was positive at 12h (1/4 symbols).

Decision: freeze. Do not reverse, lower z threshold, change 720h lookback, expand symbols, or inspect OOS.
