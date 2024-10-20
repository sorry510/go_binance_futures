<p align="center">
    <a href="./README.md">English </a>
    ·
    <a href="./README.CN.md">简体中文</a>
</p>

# 币安交易机器人

# 免责申明
>！！！本项目不构成任何投资建议，投资者应独立决策并自行承担风险！！！

# 实时推送
> dingding, slack

- 钉钉
![钉钉推送1](./img/zh/dingding_future1.jpg)

- slack
![slack](./img/zh/listen_slack.jpg)

# 功能

# Features <!-- omit in toc -->

- [合约交易](#合约交易)
- [合约订单](#合约订单)
- [新币抢购](#新币抢购)
- [币种通知](#币种通知)
    - [现货通知](#现货通知)
    - [合约通知](合约通知)
- [行情监听](#行情监听)
    - [现货监听](#现货监听)
    - [合约监听](合约监听)
- [资金费率](#资金费率)
- [系统设置](#系统设置)

## 合约交易
- 每个币种的单独配置
![交易币种](./img/zh/coins.jpg)
![钉钉推送1](./img/zh/dingding_future1.jpg)

## 合约订单
- 合约自动交易的订单历史(收益是根据下单预估的，没有查询币安的接口，与实际收益会有稍微不同)
![交易订单](./img/zh/order.jpg)

## 新币抢购
- 币币抢买
- 币币挖矿抢卖
- 合约抢买做多
- 合约抢买做空
![新币抢购](./img/zh/rush.jpg)

## 币种通知
### 现货通知
- 达到预设价格报警通知
- 自动买入或卖出
![现货通知](./img/zh/spot_notice.jpg)

### 合约通知
- 达到预设价格报警通知
- 自动买入并自动挂止盈止损单
![合约通知](./img/zh/feature_notice.jpg)

## 行情监听

### 现货监听
- k线变化监听
![现货监听](./img/zh/listen_spot.jpg)

### 合约监听
- k线变化监听
- 肯纳特通道信号监听
![合约监听](./img/zh/listen_feature.jpg)
![合约监听chart1](./img/zh/listen_chat_kc.jpg)
![合约监听通知1](./img/zh/dingding_listen1.jpg)

## 资金费率
- 资金费率查询和历史记录
- 资金费率变化监听
![资金费率](./img/zh/fundingrate.jpg)
![资金费率历史](./img/zh/fundingrate_history.jpg)

## 系统设置
- 交易的基本相关的配置(conf)
![交易配置](./img/zh/config.jpg)

## 使用注意事项
- 网络必须处于大陆之外(因为币安接口大陆正常无法访问), 已添加币安 api 的代理配置(websocket 因为使用组件问题，暂无代理配置， websocket 只是用于后台更新合约币种最新价格)，如果有可用代理也可以正常使用
- 申请api_key地址: [币安API管理页面](https://www.binance.com/zh/usercenter/settings/api-management)
- 如果你的账号本身已经有合约仓位，请一定要在 `app.conf` 文件中配置 `excludeSymbols`, 排出掉你不想使用本程序自动交易的币，否则默认所有的仓位都会根据交易策略规则自动平仓
- !!!注意修改app.conf配置后必须重新启动程序，否则配置不会生效!!!
- 请保证账户有足够的 USDT，否则下单会报错
- 钉钉推送 1min 中内不要超过 20 条，否则会被封 ip 一段时间，无法推送成功
- 调整过大的参数(例如同一个 ip 下使用多种组合功能)可能会造成币安 api 请求频率超出限制，会禁用一段时间 ip

## 如何使用
> 在 https://github.com/sorry510/go_binance_futures/releases 页面下载最新对应操作系统的发布版解压后配置运行或者使用`golang`自行编译

### 修改配置文件
> 配置说明请参考 `app.conf.example` 中每一项的说明，复制修改文件名为 `app.conf`

```
cp conf/app.conf.example conf/app.conf
```

### 策略配置参数说明
> 自动买卖的策略需要看代码自行分析，有好的思路可以提供建议方案我来实现

```
[trade]
# 是否开启合约交易的总开关
future_enable = 1
# 止盈百分比
profit = 100
# 止损百分比
loss = 100
# 购买策略(目前可以写 line1, line2, line3, line4, line5, line6, line7) 数字越大，策略越新
strategy_trade = line0
# 选币策略(目前可以写 coin1, coin2, coin3, coin4, line5)
strategy_coin = coin5
```

### 程序运行
> !!!注意修改app.conf配置后必须重新启动程序，否则配置不会生效!!!

```
./go_binance_futures
```

### web 界面说明
>访问地址: http://ip:host/zmkm/index.html # ip 为部署服务器ip，port 为 app.conf 中 web.port
登录的账号密码为 app.conf 文件中的  web.username 和 web.password

### 交易策略
> 参考 `feature/strategy` 文件夹

### 交易列表按钮说明(非必需，用来修改配置后的重新启动)
#### 重启所有服务
> 对应的是 app.conf 中 web.commend_start 下的命令，需要自行配置

#### 停止合约服务
> 对应的是 app.conf 中 web.commend_stop 下的命令，需要自行配置

#### 开启所有
> 开启所有币种

#### 停用所有
> 停用所有币种

### 新币抢购配置说明

#### 币币抢买功能配置例子

| 币种  |  买卖类型 | 类型  | usdt  | 数量精度  | 开启  |
| ------------ | ------------ | ------------ | ------------ | ------------ | ------------ |
| ABCUSDT(切记带着USDT后缀)   | 买币  | 币币  | 10  |0.1(手动设定会减少一次api请求，不知道时设置为0会在上线时查询接口自动获取)   | 开启   |

#### 币币挖矿抢卖功能配置例子
> ps: 如果挖矿的总价值小于5usdt，不能进行交易

| 币种  |  买卖类型 | 类型  | 数量精度  | 数量 | 开启  |
| ------------ | ------------ | ------------ | ------------ | ------------ | ------------ |
| ABCUSDT(切记带着USDT后缀)   | 卖币  | 币币  | 0.1(手动设定会减少一次api请求，不知道时设置为0会在上线时查询接口自动获取)   | 80(挖矿所得数量) |开启   |

#### 合约抢买做多配置例子

| 币种  |  买卖类型 | 类型  |模式| usdt|  倍率| 数量精度  |  开启  |
| ------------ | ------------ | ------------ | ------------ | ------------ | ------------ |------------ | ------------ |
| ABCUSDT(切记带着USDT后缀)   | 买币  | 合约  | 逐仓或全仓| 10|3 |0.1(手动设定会减少一次api请求，不知道时设置为0会在上线时查询接口自动获取)  |开启   |

#### 合约抢买做空配置例子
| 币种  |  买卖类型 | 类型  |模式| usdt|  倍率| 数量精度  |  开启  |
| ------------ | ------------ | ------------ | ------------ | ------------ | ------------ |------------ | ------------ |
| ABCUSDT(切记带着USDT后缀)   | 卖币  | 合约  | 逐仓或全仓| 10|3 |0.1(手动设定会减少一次api请求，不知道时设置为0会在上线时查询接口自动获取)  |开启   |


## 开发
>安装最新版 golang

## 配置文件

```
cp ./conf/app.conf.example app.conf
```

### 安装 bee
> 记得将`GOPATH/bin`添加到环境变量`PATH`，否则 `bee` 命令无法全局使用
> 使用 `go env GOPATH` 查看 `GOPATH` 路径

```
go install github.com/beego/bee/v2@latest
```

### 安装依赖
> 进入项目根目录下执行

```
go mod tidy
```

### 启动
> http://localhost:3333/zmkm/index.html

```
bee run
```

### 打包

#### 打包到`windows`平台
> 其它平台需要参考 bee 文档
> 此项目的 github 的 workflows 实现了 linux amd64 和 window amd64 下的编译打包

```
bee pack -be GOOS=windows
```

## web ui 开发
> https://github.com/sorry510/go_binance_futures_ui

### TODO

- [X] 完成新币抢购功能
- [X] 完成挖矿新币抛售功能
- [X] 添加独立的币种配置收益率
- [X] 添加一键修改所有币种的配置
- [X] 系统首页显示(那些服务开启和关闭)
- [X] 监听币种的价格突变情况，报警通知
- [X] 学习蜡烛图结合其它数据，报警通知
- [ ] 添加新的自动交易策略
- [ ] 添加定时自动交易(现货买入和一倍合约等价格对冲，吃资金费用)
- [ ] 监控资金流入流出，报警通知