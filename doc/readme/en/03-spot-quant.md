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
3. Both switches must be enabled, the signed user-data WS must be ready, the strategy signal must hold, and balance, quantity precision and exchange-rule checks must pass before an automatic BUY or SELL. Disabling either switch prevents **new automatic orders**; it **does not force-sell existing strategy positions**.

**Important:** The default `profit=100` and `loss=100` mean **+100% / −100% unleveraged ROI**, not the commonly used +8% / −6%. `close_long` grants permission to sell only after the exit ROI threshold is reached; **a true rule alone does not trigger an immediate close**. Confirm thresholds and budget before enabling trading.
