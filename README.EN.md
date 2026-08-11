<p align="center">
    <a href="./README.md">简体中文</a>
    ·
    <a href="./README.EN.md">English </a>
</p>

# Binance-trade-bot

## update database
> If it is the first time to use the program, you can just run it directly. If you have been using it for a while, when downloading the new version of the program, you can directly place the old database file in the `db` directory and then run the program.

## peculiarity

## Pusher
> dingding, slack

- dingding
![钉钉推送1](./img/en/dinding_future1.jpg)

- slack
![slack](./img/en/listen_slack.jpg)

## custom strategy

In the new UI, maintain indicators and strategy methods under **Futures Trade → Strategy Templates** (`合约交易 → 策略模板`), then assign a global or custom strategy to each symbol under **Futures Trade → Futures Trade** (`合约交易 → 合约交易`).

<a href="./STRATEGY.md">custom strategy details</a>

# DISCLAIMER

```
I am not responsible for anything done with this bot.
You use it at your own risk.
There are no warranties or guarantees expressed or implied.
You assume all responsibility and liability.
```


# Features <!-- omit in toc -->

## New Web UI

Open `http://<server-ip>:<web.port>/zmkm/index.html`. The login username and password come from `web.username` and `web.password` in `conf/app.conf`.

| Menu | Pages and purpose |
| --- | --- |
| Configuration Center (`配置中心`) | Global switches and runtime settings for futures trading, WebSocket, new-coin rush, price alerts, market monitoring, funding-rate monitoring, notifications, debug push, and external links |
| Futures Trade → Futures Trade (`合约交易 → 合约交易`) | Browse `Favorites`, `USDT`, and `USDC` symbols; add, search, batch-edit, enable all, or disable all symbol configurations |
| Futures Trade → Futures Orders (`合约交易 → 合约订单`) | Search real futures order history by symbol and time range |
| Futures Trade → Futures Account (`合约交易 → 合约账户`) | View Binance futures assets, positions, and open orders |
| Futures Trade → Local Futures Account (`合约交易 → 本地合约账户`) | View assets, positions, and orders recorded locally by the application |
| Futures Trade → Strategy Templates (`合约交易 → 策略模板`) | Create and maintain indicator and strategy-method templates |
| Futures Trade → Test Results (`合约交易 → 测试结果`) | Search simulated-trading results by symbol and time range |
| Futures Trade → Market Volatility Logs (`合约交易 → 市场波动日志`) | Search and group fast-market-movement records |
| Coin Alerts → Spot/Futures Alerts (`币种提醒 → 现货提醒 / 合约提醒`) | Configure target-price alerts and optional automatic trades |
| Market Monitoring → Spot/Futures Monitoring (`市场监听 → 现货监听 / 合约监听`) | Configure K-line, threshold, indicator, and custom-strategy monitoring |
| Funding Rate Monitoring (`资金费率监听`) | View funding rates and configure automatic trading |
| New-Coin Rush (`抢新配置`) | Configure spot, mining-token, and futures launch trades |
| System Configuration (`系统配置`) | Edit `conf/app.conf` online and access Save, Restart Service, and Stop Service actions |
| Logs (`日志`) | Display the service logs returned by `web.commend_log` |

### New UI screenshots

These screenshots use the current UI. Pages that may contain balances, orders, configuration secrets, or log text show only a safe area.

#### Configuration Center
![New UI - Configuration Center](./img/ui/dashboard.jpg)

#### Futures Trade
![New UI - Futures Trade](./img/ui/futures-symbols.jpg)

#### Futures Orders
![New UI - Futures Orders](./img/ui/futures-orders.jpg)

#### Futures Account
![New UI - Futures Account](./img/ui/futures-account.jpg)

#### Local Futures Account
![New UI - Local Futures Account](./img/ui/local-futures-account.jpg)

#### Strategy Templates
![New UI - Strategy Templates](./img/ui/strategy-templates.jpg)

#### Test Results
![New UI - Test Results](./img/ui/test-results.jpg)

#### Market Volatility Logs
![New UI - Market Volatility Logs](./img/ui/market-volatility-logs.jpg)

#### Spot Alerts
![New UI - Spot Alerts](./img/ui/spot-alerts.jpg)

#### Futures Alerts
![New UI - Futures Alerts](./img/ui/futures-alerts.jpg)

#### Spot Monitoring
![New UI - Spot Monitoring](./img/ui/spot-monitoring.jpg)

#### Futures Monitoring
![New UI - Futures Monitoring](./img/ui/futures-monitoring.jpg)

#### Funding Rate Monitoring
![New UI - Funding Rate Monitoring](./img/ui/funding-rate.jpg)

#### New-Coin Rush
![New UI - New-Coin Rush](./img/ui/new-coin-rush.jpg)

#### Notification Configuration
![New UI - Notification Configuration](./img/ui/notification-config.jpg)

#### System Configuration
![New UI - System Configuration](./img/ui/system-config.jpg)

#### Logs
![New UI - Logs](./img/ui/service-logs.jpg)

## futures-trade

The **Futures Trade → Futures Trade** page supports per-symbol settings for strategy type, indicators, strategy methods, margin mode, USDT amount, leverage, take-profit rate, stop-loss rate, and enabled status.

### Enabling futures trading

1. Under **Configuration Center → Futures Trade** (`配置中心 → 合约交易`), enable the futures master switch and `WebSocket`.
2. Enable `Allow Long` and/or `Allow Short`, then configure the global strategies, position limits, and order type.
3. Open **Futures Trade → Futures Trade** and verify the target symbol's parameters and enabled status.

### Global settings

- **24h movement and fast-movement alerts:** configure the movement threshold, recovery threshold, cooldown, and monitoring windows. Review triggered events under **Market Volatility Logs**.
- **Position profit sign-change notification:** sends a notification when a position changes between profit and loss. Enabling it for many positions or symbols increases API traffic.
- **Allow Long / Allow Short:** direction-level master switches; disabled directions will not open even when a strategy matches.
- **Trading Strategy / Coin Selection Strategy:** used when a symbol's strategy type is `global`; custom symbols use their own configuration.
- **Maximum Positions / Maximum Losing Positions:** prevent new automatic positions after a limit is reached. Automatic scaling can adjust the losing-position limit after consecutive wins or losses.
- **Market Trend:** select it manually or enable automatic market-trend updates for strategy evaluation.
- **Excluded Symbols:** symbols selected in the UI are excluded from automatic trading. Add existing manual positions here if the bot must not manage them.
- **Order Type:** `LIMIT` places a limit order; `MARKET` executes a market order.

## simulated custom-strategy trading (no backtesting)

Enable `WebSocket` and `Test Strategy` under **Configuration Center → Futures Trade**. Simulation follows the same strategies and limits as real automatic trading but does not operate the real futures account. Open the result from the `View Test Results` button or **Futures Trade → Test Results**.

When the `Test Auto-Conversion Count Limit` is non-zero, consecutive simulated wins can switch to real trading; consecutive real-trading losses can switch back to test mode.

## futures-trade-order

- **Futures Orders:** automatic-trading order history. Profit is estimated from order data and may differ slightly from Binance's final result.
- **Futures Account:** assets, positions, and open orders returned by Binance.
- **Local Futures Account:** the application's local account state, useful for comparing local data with exchange data.

## strategy-templates

Create and maintain templates under **Futures Trade → Strategy Templates**. A template contains indicators and strategy methods and can be reused by futures symbols and monitoring rules. See [Custom Strategy Details](./STRATEGY.md) for syntax.

## new-coin-rush

- spot rush buy
- mining rush sell
- futures rush long
- futures rush short

First enable the corresponding switch under **Configuration Center → New-Coin Rush**, then add the symbol rule on the **New-Coin Rush** page.

## coin-notice

### spot-notice

- alarm notification for reaching the preset price
- automatic buying or selling

### futures-notice

- alarm notification for reaching the preset price
- optional automatic trading with margin mode, USDT, leverage, take-profit price, and stop-loss price

Enable the master switch under **Configuration Center → Coin Alerts**, then maintain rules under **Coin Alerts → Spot Alerts** or **Futures Alerts**.

## market-listen

### spot-listen

- K-line change monitoring
- configurable threshold, notification interval, and enabled status

### futures-listen

- K-line and indicator monitoring
- custom strategy

Enable the master switch under **Configuration Center → Market Monitoring**, then add rules under **Spot Monitoring** or **Futures Monitoring**.

## funding-rate

- funding rate search and history
- funding rate change listen

Enable the master switch under **Configuration Center → Funding Rate Monitoring**, then query and configure symbols on the **Funding Rate Monitoring** page.

## system-config

- **Configuration Center:** changes database-backed runtime switches and parameters.
- **System Configuration:** edits `conf/app.conf` online. After `Save`, startup-time settings still require `Restart Service` or a manual application restart.
- **Logs:** displays output from the command configured in `web.commend_log`.

The System Configuration page contains sensitive values such as API keys, database passwords, and notification tokens. Never include it in screenshots, issues, or Git commits.

## important
- The network must be located outside the mainland (as the Binance interface cannot be accessed normally in mainland China). The proxy configuration for Binance API has been added (websocket has no proxy configuration due to component usage issues, and is only used to update the latest contract currency prices in the background). If there are available proxies, they can also be used normally
-Apply for api_key address: [Binance API Management Page]（ https://www.binance.com/cn/usercenter/settings/api-management )
- If the account already has futures positions, select symbols that the bot must not manage under **Configuration Center → Futures Trade → Excluded Symbols**. Otherwise, the bot may close them according to its strategy
- After modifying `conf/app.conf`, restart the application before expecting startup-time settings to take effect
-Please ensure that your account has sufficient USDT, otherwise placing an order will result in an error
- Do not exceed 20 notifications within 1 minute of DingTalk push, otherwise the IP address will be blocked for a period of time and the push will not be successful
-Adjusting too many parameters (such as using multiple combinations of functions under the same IP) may cause the Binance API request frequency to exceed the limit and disable the IP for a period of time

## how to use
> in https://github.com/sorry510/go_binance_futures/releases page download or use `golang` compile

### FAQ (new UI)

1. **Where do I enable automatic trading?** Open **Configuration Center → Futures Trade** and verify the futures master switch, `WebSocket`, and `Allow Long/Allow Short`. Then verify the symbol's enabled status under **Futures Trade → Futures Trade**.
2. **Where can I view simulated trades?** Enable `Test Strategy`, then use `View Test Results` or open **Futures Trade → Test Results**.
3. **Why did a configuration change not take effect?** Configuration Center updates runtime settings. System Configuration edits `conf/app.conf`; after `Save`, startup-time settings still require an application restart.
4. **Why are futures prices delayed?** Prices are updated through WebSocket. Check the `WebSocket` switch, network quality, and proxy stability.
5. **Why can the account not open a position?** Check direction switches, maximum-position limits, maximum losing positions, excluded symbols, symbol enabled status, and available USDT. Binance may also restrict some IP regions.
6. **Why is the UI slow?** For larger datasets, use MySQL; SQLite is better suited to smaller deployments.
7. **What should I do after an API rate-limit error?** Reduce enabled symbols, monitoring rules, and high-frequency notifications, then wait for Binance's restriction to clear.
8. **Where can I view service logs?** Open **Logs**. It runs the command configured in `web.commend_log`.

### edit config

```
cp conf/app.conf.example conf/app.conf
```

#### database config

##### use sqlite

- app.conf
```
[database]
driver = "sqlite"
path = "./db/coin.db?_journal_mode=WAL&_busy_timeout=5000"
```

##### use mysql

```
[database]
driver = "mysql"
username = ""
password = ""
host= ""
port= ""
dbname = ""
```

### how to run
> !!!Please note that after modifying the `app.conf` configuration, the program must be restarted, otherwise the configuration will not take effect!!!

```
./go_binance_futures
```

### web page

- Server URL: `http://<server-ip>:<web.port>/zmkm/index.html`
- Local development URL: `http://localhost:3333/zmkm/index.html`
- Login credentials: `web.username` and `web.password` in `conf/app.conf`
- After login, the UI opens **Configuration Center**. Use the left navigation menu; there is no need to manually build `#/...` routes

### Trading Strategy
> Refer to the `feature/strategy` folder and [Custom Strategy Details](./STRATEGY.md)

### Common actions in the new UI

- **Futures Trade → Enable All / Disable All:** batch-updates the enabled status of the current symbol list. Verify the active filter first.
- **Futures Trade → Batch Edit:** changes settings for multiple symbols.
- **System Configuration → Save:** saves the `conf/app.conf` content shown in the editor.
- **System Configuration → Restart Service:** runs `web.commend_start`; configure this command in `conf/app.conf` first.
- **System Configuration → Stop Service:** runs `web.commend_stop`; configure this command in `conf/app.conf` first.
- **Logs:** runs `web.commend_log` and displays its output.

### new coin rush config

#### spot rush buy

| coin  |  trade_type | coin_type  | usdt  | step_size  | enable  |
| ------------ | ------------ | ------------ | ------------ | ------------ | ------------ |
| ABCUSDT  | buy  | spot  | 10  |0.1(if you don't know, please fill in 0)   | open   |

#### spot mining rush sell
> ps:Binance has a minimum transaction limit, and if the quantity is too small (such as 5 USDT), it cannot be conducted

| coin  |  trade_type | coin_type  | step_size  | amount  | enable  |
| ------------ | ------------ | ------------ | ------------ | ------------ | ------------ |
| ABCUSDT  | sell  | spot  | 0.1(if you don't know, please fill in 0)   | 80(Quantity of mining income) | open  |

#### futures rush buy long

| coin  |  trade_type | coin_type  | margin_type | usdt|  leverage | step_size  |  enable  |
| ------------ | ------------ | ------------ | ------------ | ------------ | ------------ |------------ | ------------ |
| ABCUSDT  | buy_long  | futures  | ISOLATED or CROSSED| 10|3 | 0.1(if you don't know, please fill in 0)  | open   |

#### futures rush buy short
| coin  |  trade_type | coin_type  |margin_type| usdt|  leverage | step_size  |  enable  |
| ------------ | ------------ | ------------ | ------------ | ------------ | ------------ |------------ | ------------ |
| ABCUSDT   | buy_short  | futures  | ISOLATED or CROSSED | 10|3 | 0.1(if you don't know, please fill in 0)   | open   |

## donate

### qrcode
![usdt](./img/bsc-usdt.jpg)

#### bsc-usdt

```
0x170197328b6e73597bc29a1b059f29d4e111e1e8
```


## how to develop
>install golang

## technology function
>futures/strategy/line/technology.go

## config file

```
cp ./conf/app.conf.example app.conf
```

### install bee
> Remember to add `GOPAT/bin` to the environment variable `PATH`, otherwise the `bee` command cannot be used globally
> Use `go env GOPATH` to view the `GOPATH` path

```
go install github.com/beego/bee/v2@latest
```

### Install dependencies

```
go mod tidy
```

### how to run
> go to http://localhost:3333/zmkm/index.html

```
bee run
```

### pack

#### pack to `windows`

```
bee pack -be GOOS=windows
```

## web ui
> https://github.com/sorry510/binance_bot_ui
