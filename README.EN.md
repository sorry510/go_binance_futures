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
> ![img1](./img/en/te_001.jpg)
> ![alt text](./img/en/strategy_001.png)
> ![alt text](./img/en/strategy_002.png)

<a href="./STRATEGY.md">custom strategy details</a>

# DISCLAIMER

```
I am not responsible for anything done with this bot.
You use it at your own risk.
There are no warranties or guarantees expressed or implied.
You assume all responsibility and liability.
```


# Features <!-- omit in toc -->

- [Futures Trade](#futures-trade)
- [Futures Trade Order](#futures-trade-order)
- [New Coin Rush](#new-coin-rush)
- [Coin Notice](#coin-notice)
    - [Spot Notice](#spot-notice)
    - [Futures Notice](futures-notice)
- [Market Listen](#market-listen)
    - [Spot Listen](#spot-listen)
    - [Futures Listen](#futures-listen)
- [Funding Rate](#funding-rate)
- [System Config](#system-config)

## futures-trade
- Independent configuration for each coin
-  <a href="./STRATEGY.md">custom strategy</a>
![交易币种](./img/en/coins.jpg)
![钉钉推送1](./img/en/dinding_future1.jpg)

## futures-trade-order
- Order history (revenue is estimated based on orders placed, without a Binance query interface, and may differ slightly from actual revenue)
![交易订单](./img/en/order.jpg)

## new-coin-rush
- spot rush buy
- mining rush sell
- futures rush long
- futures rush short
![新币抢购](./img/en/rush.jpg)

## coin-notice
### spot-notice
- alarm notification for reaching the preset price
- automatic buying or selling
![现货通知](./img/en/spot_notice.jpg)

### futures-notice
- alarm notification for reaching the preset price
- automatic buying or selling
![合约通知](./img/en/futures_notice.jpg)

## market-listen

### spot-listen
- kline change listen
![现货监听](./img/en/listen_spot.jpg)

### futures-listen
- kline change listen
- kline keltner channels listen
- custom strategy
![合约监听](./img/en/listen_futures.jpg)
![合约监听chart1](./img/en/listen_chat_kc.jpg)
![合约监听通知1](./img/en/dingding_listen1.jpg)

## funding-rate
- funding rate search and history
- funding rate change listen
![资金费率](./img/en/fundingrate.jpg)
![资金费率历史](./img/en/fundingrate_history.jpg)

## system-config
- app.conf

```
# Remember not to use single quotes, only double quotes or no quotes, comments need to be on a separate line
appname = binance_futures
# zh, en
language = en
log = 1
debug = 0

[binance]
api_key = ""
api_secret = ""
# Local proxy (if not needed to be changed to "")
proxy_url = "http://127.0.0.1:7890"

[web]
# web port
port = 3333
# index path
index = zmkm
# jwt key
secret_key = 12321
# username
username = admin
# password
password = admin
# expires hours
expires = 24
# restart command
commend_start = pm2 restart binance_futures
# stop command
commend_stop = pm2 stop binance_futures
# log command
commend_log = pm2 log binance_futures

[notification]
# dingding, slack
channel = dingding

[dingding]
# token
dingding_token = ""
# trigger keywords
dingding_word = "报警"

[slack]
slack_token = ""
slack_channel_id = ""

[external]
# external links
links = [{"url": "url1", "title": "title1"}]
```
![交易配置](./img/en/config.png)

## important
- The network must be located outside the mainland (as the Binance interface cannot be accessed normally in mainland China). The proxy configuration for Binance API has been added (websocket has no proxy configuration due to component usage issues, and is only used to update the latest contract currency prices in the background). If there are available proxies, they can also be used normally
-Apply for api_key address: [Binance API Management Page]（ https://www.binance.com/cn/usercenter/settings/api-management )
- If your account already has contract positions, please be sure to exclude coins that you do not want to use this program for automatic trading on index page. Otherwise, all positions will be automatically closed according to the trading strategy rules by default
- !!! Please note that after modifying the app.cnf configuration, the program must be restarted, otherwise the configuration will not take effect!!!
-Please ensure that your account has sufficient USDT, otherwise placing an order will result in an error
- Do not exceed 20 notifications within 1 minute of DingTalk push, otherwise the IP address will be blocked for a period of time and the push will not be successful
-Adjusting too many parameters (such as using multiple combinations of functions under the same IP) may cause the Binance API request frequency to exceed the limit and disable the IP for a period of time

## how to use
> in https://github.com/sorry510/go_binance_futures/releases page download or use `golang` compile

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
> access address: http://ip:host/zmkm/index.html# The IP is the deployment server IP, and the port is the web. port in app.cnf
The login account password is the `web.username` and `web.password` in the `app.conf` file

### Trading Strategy
> Refer to the 'futuress/rate' folder


### description of the futures-trade list button (optional, used for restarting after modifying the configuration)
#### service restart
> will run `web.commend_start`

#### service stop
> will run `web.commend_stop`

#### open all coin
> open all futures coin

#### close all coin
> close all futures coin

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
> https://github.com/sorry510/go_binance_futures_ui
