# Spot Quant Trading

[中文](../zh-CN/03-现货量化交易.md) · [Back to English README](../../../README.EN.md)

## Automated Spot Quant Trading (06-7)

Spot Quant uses the existing **two-second strategy polling loop**, with separate Spot symbols and indicators and `LONG → BUY` / `CLOSE_LONG → SELL` rules. **Only the two existing business switches** are needed; no third mode, extra environment variable, or Task Center is required.

**1. Configure the trading environment (`[binance]` in `conf/app.conf`)**

| `testnet` | Spot REST | API credentials | Notes |
| --- | --- | --- | --- |
| `true` | `https://testnet.binance.vision` | `testnet_spot_api_key` / `testnet_spot_api_secret` | Spot Testnet; **does not use** Futures Testnet `testnet_api_key` / `testnet_api_secret` |
| `false` | `https://api.binance.com` | `api_key` / `api_secret` | **Live Spot mainnet: real money and real orders** |

Set `testnet_binance_spot_base_url` to `https://testnet.binance.vision` without `/api/v3` (the SDK adds it). `testnet` is the existing Binance environment switch. **Restart the service** after changing `app.conf`, and check other features in the same process that use this setting.

**2. Enable automated trading**

1. Under **Configuration Center → Spot Quant Trading**, enable the master switch (`config.spot_quant_enable=1`).
2. Under **Spot Quant → Spot Trading**, configure each symbol's quote-asset budget (`usdt`), technical indicators, `long` / `close_long` rules, and take-profit/stop-loss thresholds; then enable the symbol (`spot_symbols.spot_quant_enabled=1`).
3. **New BUY entries** require both the global and per-symbol switches, a ready signed user-data WS, a valid strategy signal, and balance, precision and exchange-rule checks. Disabling either switch stops **new BUYs only**. Existing Managed positions continue to evaluate take-profit / stop-loss and **SELL** rules, and unsettled orders are reconciled. Disabling a switch **does not immediately force-sell** a position. Automatic SELLs still require verified Fill ownership and available balance.

**Important:** The default `profit=100` and `loss=100` mean **+100% / −100% unleveraged ROI**, not the commonly used +8% / −6%. `close_long` grants permission to sell only after the exit ROI threshold is reached; **a true rule alone does not trigger an immediate close**. Confirm thresholds and budget before enabling trading.

### Spot scanning and API rate-limit optimization

- The scheduler still ticks every **two seconds** without overlapping Runner cycles. Actual cycle duration depends on the symbols and Binance response times.
- Each cycle checks Managed positions and unsettled orders **first**, then scans other `spot_quant_enabled=1` symbols. There is no global Spot coin-selection strategy.
- Spot 24-hour market snapshots are cached for **four seconds** per trading environment. Kline data is cached for up to **seven seconds**, never past the latest exchange candle boundary. Identical concurrent requests are coalesced; failed responses are not cached as valid data.
- Mainnet and Testnet caches are isolated. Mainnet may use a fresh matching-environment market WebSocket price; Testnet never uses Mainnet market data.
- All real REST requests still pass through the existing Binance API weight budget and `429/418` backoff. Caching does not guarantee that rate limits cannot occur; monitor Spot `USED-WEIGHT-1M` and endpoint usage on the System Dashboard.

### What disabling switches means and how recovery works

**Disabling the global or per-symbol entry switch is not an emergency stop for all automatic orders.** It stops new BUYs, but an existing Fill-verified Managed position may still submit a MARKET SELL when its exit strategy is satisfied. Disabling a switch does not sell immediately.

- Managed exits and unsettled orders still require Spot REST and an authenticated User Data WebSocket. Until signed WS subscription ACK confirms readiness, **both automatic BUY and SELL are paused**. Existing orders can still be reconciled using read-only REST; there is no unapproved REST-only SELL fallback.
- When entry switches are OFF and no Managed positions or pending orders remain, the authenticated user WS is closed on the next work-state check; only a low-frequency local database check continues. The stream resumes if work returns.
- When a disabled entry route repeatedly produces `ErrNeedsReview`, recovery attempts back off at 5, 10, 20, and up to 30 seconds to reduce unnecessary exchange requests. Inspect Binance orders, Fills, managed inventory, and discrepancies manually. After correction, recovery resumes at the next retry.
- To completely stop this process from submitting any automatic orders, stop the trading service through the external process supervisor. **This also stops automatic exits and reconciliation of existing positions**; inspect exchange positions, open orders and protections first. No separate Spot Quant kill switch currently exists.
- Each market cache is bounded to 256 entries and uses least-recently-used eviction. This reduces repeated market REST requests; it does not replace Binance weight-budget controls or order-recovery backoff.

Live Mainnet execution still requires the project's separate production release checks. Passing offline tests does not imply production readiness.
