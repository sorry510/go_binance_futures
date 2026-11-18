# System Overview

[中文](../zh-CN/00-系统概况.md) · [Back to English README](../../../README.EN.md)

---

## Features
- Automated futures trading
- Automated spot trading
- Real-time notifications
- Custom trading strategies
- Custom monitoring-event alerts
- Historical futures backtesting
- Futures paper trading
- AI Agents
- New-coin rush trading

---

### Real-time Notifications
> DingTalk, Slack, and in-app WebSocket notifications

- DingTalk
![DingTalk notification 1](../../../img/zh/dingding_future1.jpg)
![DingTalk notification 2](../../../img/zh/dingding3.jpg)
![DingTalk notification 3](../../../img/zh/dingding4.jpg)

- Slack
![Slack](../../../img/zh/listen_slack.jpg)

- In-app WebSocket notifications
![WebSocket notification](../../../img/ui/ws-notification1.png)


## Web UI

Open `http://<server-ip>:<web.port>/zmkm/index.html` and use the left-hand menu after signing in. The username and password come from `web.username` and `web.password` in `conf/app.conf`.

| Menu | Page and purpose |
| --- | --- |
| System Dashboard | System health indicators and operational statistics |
| Configuration Center | Global trading switches and runtime settings |
| Futures Trading → Futures Trading | Browse, add, search, batch-edit, enable, or disable symbol configurations |
| Futures Trading → Futures Orders | Order history, including bulk hiding/restoring within filters and manual annotations |
| Futures Trading → Futures Positions | Local position ownership, Managed orders, manual claims, reconciliation, and repair history |
| Futures Trading page → Copy Trading button next to Funding Rate | Dedicated Lead account read-only management: credentials and verification, risk controls, pause, managed positions/orders, local audit, and WS/API status. **Live Lead order submission is not authorized.** |
| Futures Trading → Futures Account | Assets, local positions, and local open orders |
| Futures Trading → Strategy Templates | Create and maintain technical-indicator and strategy-method templates |
| Futures Trading → Test Results | Search real-time simulated trades by symbol and date |
| Futures Trading → Historical Backtest | Replay strategies on historical 1m bars, Funding, and MarketCondition; Standard 1m and Adaptive Resolution, trades, equity curves, audit events, and drill-down statistics |
| COIN-M Futures → COIN-M Symbols | USD-inverse perpetual contracts, per-symbol budget/leverage/STOP/TP, LONG/SHORT/CLOSE strategy and entry switches; deploying with both gates enabled can submit live orders |
| COIN-M Futures → Local Account | Read-only Managed cycles, real Fills, asset-separated BTC/ETH PnL and independent read-only COIN-M wallet balances |
| COIN-M Futures → Strategy Templates | Dedicated four-direction COIN-M templates and indicators, separate from USD-M and Spot |
| COIN-M Futures → Positions & Controlled Trading | Local runtime/Managed inventory and unsettled orders, no manual submit/cancel/close endpoints; [documentation](10-coin-m-futures.md) |
| Spot Trading → Spot Trading | Independent Spot symbol configuration, LONG / CLOSE_LONG strategies, indicators, filtering, and enable switches |
| Spot Trading → Local Spot Account | Locally managed positions, orders, actual exchange Fills, and PnL by quote asset; full net PnL is unavailable with stale prices or unvalued fees |
| Spot Trading → Spot Strategy Templates | Spot-specific indicators and strategy templates, separate from Futures |
| Spot Trading → Positions & Controlled Trading | View system-owned inventory and anomalies; manually rebuild local aggregates. Manual selling, cancellation, and exchange reconciliation UI actions are not yet available |
| Coin Alerts → Spot / Futures Alerts | Target-price alerts with optional automatic trading |
| Market Monitoring → Spot / Futures Monitoring | K-line, threshold, indicator, and custom-strategy monitoring |
| AI → Chat | Multi-turn Agent conversations, symbol selection, model switching, and mounting multiple Skills |
| AI → Market Analysis | **Symbol Analysis / Opportunities** tabs: structured TradingPlans, historical analysis, opportunity review, and controlled-trade proposals |
| AI → Business Workflows | Five manually executable workflows. Strategy Draft Validation compiles expressions and tests synthetic scenarios only; **it is not a historical backtest**. Qualified market-scan candidates are sent to opportunity analysis asynchronously |
| AI → Task Center | **Tasks / Scheduler & Governance / Alert Pipeline** tabs: execution details, scheduling status, metrics, and alert history |
| AI → AI Settings | **AI Configuration / Models / Skills / MCP / Memory** tabs, opening on AI Configuration to manage opportunity monitoring, alerts, scheduling, trade risk, and AI runtime settings |
| Notifications | Review notification history and configure delivery |
| New-Coin Rush | Configure launch-time Spot, mining-token, and Futures trades |
| System Configuration | Edit the configuration file with Save and Restart to apply |
| Logs | View application logs |

---

## UI Screenshots

#### System Dashboard
![System Dashboard 1](../../../img/ui/dashboard1.png)
![System Dashboard 2](../../../img/ui/dashboard2.png)

#### Configuration Center
![Configuration Center 1](../../../img/ui/config1.png)
![Configuration Center 2](../../../img/ui/config2.png)
![Configuration Center 3](../../../img/ui/config3.png)

#### Futures Trading
![Futures Trading](../../../img/ui/futures-symbols.png)

#### Futures Orders
![Futures Orders](../../../img/ui/futures-orders.png)

#### Futures Positions
![Futures Positions](../../../img/ui/futures-position.png)

#### Lead Copy Trading
![Lead Copy Trading](../../../img/ui/futures-position.png)

#### Futures Account
![Futures Account](../../../img/ui/futures-account.png)

#### Strategy Templates
![Strategy Templates](../../../img/ui/strategy-templates.png)

#### Test Results
![Test Results](../../../img/ui/futures-test-results.png)

#### Historical Backtest
![Historical Backtest 1](../../../img/ui/futures-trade-history1.png)
![Historical Backtest 2](../../../img/ui/futures-trade-history2.png)

#### Spot Alerts
![Spot Alerts](../../../img/ui/spot-alerts.png)

#### Futures Alerts
![Futures Alerts](../../../img/ui/futures-alerts.png)

#### Spot Monitoring
![Spot Monitoring](../../../img/ui/spot-monitoring.png)

#### Futures Monitoring
![Futures Monitoring](../../../img/ui/futures-monitoring.png)

#### AI – Chat
![AI Chat](../../../img/ui/ai-symbol-analysis.png)

#### AI – Market Analysis
![AI Market Analysis](../../../img/ui/ai-market-symbols.png)

#### AI – Settings
![AI Settings](../../../img/ui/ai-config.png)

#### Notification Configuration
![Notification Configuration](../../../img/ui/notification-config.png)

#### Notification History
![Notification History](../../../img/ui/notification-history.png)

#### New-Coin Rush
![New-Coin Rush](../../../img/ui/new-coin-rush.png)

#### System Configuration
![System Configuration](../../../img/ui/system-config.png)

#### Logs
![Application Logs](../../../img/ui/service-logs.png)
