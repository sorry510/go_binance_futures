# go_binance_futures API 接口文档

本文档汇总所有 HTTP 接口，基于 `routers/router.go` 及对应 controller 整理。

所有需要鉴权的接口请在 Header 中携带：`Authorization: Bearer <token>`

---

## 通用响应格式

```json
{
  "code": 200,       // 200=成功，400=业务错误，500=服务端错误
  "data": {},        // 响应数据，失败时为 null
  "msg": "success"   // 描述信息
}
```

---

## 1. 认证

### POST `/login`
用户登录，获取 JWT Token。

**Request Body**
```json
{
  "username": "admin",
  "password": "123456"
}
```

**Response**
```json
{
  "code": 200,
  "data": { "token": "Bearer eyJhbGci..." }
}
```

---

## 2. 服务配置

### GET `/service/config`
获取系统运行配置，包括合约/现货/交割开关、策略参数、WS 配置等。

**Response `data` 字段（部分）**

| 字段 | 类型 | 说明 |
|---|---|---|
| `tradeFutureEnable` | int | 合约交易总开关 (0/1) |
| `tradeSpotEnable` | int | 现货交易总开关 |
| `tradeDeliveryEnable` | int | 交割合约总开关 |
| `noticeCoinEnable` | int | 价格预警开关 |
| `listenCoinEnable` | int | 监听币种开关 |
| `listenFundingRate` | int | 资金费率监听开关 |
| `marketCondition` | string | 市场状态 (bull/bear/震荡) |
| `marketConditionIsAuto` | int | 市场状态自动判断 (0/1) |
| `wsFuturesEnable` | int | 合约 WS 推送开关 |
| `coinMaxCount` | int | 同时开仓最大数量 |
| `lossMaxCount` | int | 最大亏损次数 |

### PUT `/service/config`
修改系统运行配置，Body 结构同 GET 响应的 `data`。

### POST `/test-pusher`
测试通知推送渠道是否正常。

### POST `/update-market-condition`
手动触发更新市场状态。

**Response**
```json
{
  "code": 200,
  "data": { "marketCondition": "bull" }
}
```

---

## 3. 通知渠道配置

### GET `/notify-config`
获取所有通知渠道配置列表。

### POST `/notify-config`
新增通知渠道（默认 `enable=1`）。

**Request Body**
```json
{
  "type": "dingding",   // 渠道类型: dingding / slack 等
  "token": "...",
  "enable": 1
}
```

### PUT `/notify-config/:id`
更新指定通知渠道配置。

### DELETE `/notify-config/:id`
删除指定通知渠道配置。

---

## 4. 合约交易对管理（Features）

### GET `/features`
分页查询合约交易对列表。

**Query 参数**

| 参数 | 说明 |
|---|---|
| `symbol` | 模糊搜索交易对名称 |
| `symbol_type` | 交易对类型筛选 |
| `enable` | 是否开启 (0/1) |
| `margin_type` | 仓位类型 (ISOLATED/CROSSED) |
| `pin` | 是否置顶 |
| `sort` | 排序字段，支持 `percent_change+/-`、`quote_volume+/-` |
| `page` | 页码，默认 1 |
| `limit` | 每页数量，默认 20 |

**Response `data`**
```json
{
  "total": 100,
  "list": [{ "id": 1, "symbol": "BTCUSDT", "enable": 1, ... }]
}
```

### POST `/features`
新增合约交易对（默认 leverage=3，marginType=CROSSED，usdt=10）。

### GET `/features/:id`
获取单个合约交易对详情。

### PUT `/features/:id`
更新合约交易对信息。

### DELETE `/features/:id`
删除合约交易对。

### PUT `/features/enable/:flag`
批量修改所有合约交易对的开启状态，`:flag` 为 0 或 1。

### PUT `/features/batch`
批量修改合约交易对参数。

**Request Body**
```json
{
  "usdt": "50",
  "profit": "2",
  "loss": "1.5",
  "leverage": "5",
  "marginType": "CROSSED",
  "strategyType": "trend",
  "strategyTemplateId": 3
}
```

### POST `/features/strategy-rule/test/:id`
测试指定合约交易对的策略规则是否满足条件。

**Response**
```json
{
  "code": 200,
  "data": { "pass": true, "type": "close_long" }
}
```

### GET `/features-options`
获取所有合约交易对 symbol 列表（用于下拉选项）。

---

## 5. 策略测试结果

### GET `/test-strategy-results`
分页查询策略测试下单/平仓记录。

**Query 参数**

| 参数 | 说明 |
|---|---|
| `symbol` | 模糊搜索 |
| `position_side` | LONG / SHORT |
| `type` | open（未平仓）/ close（已平仓）|
| `start_time` | 开始时间 |
| `end_time` | 结束时间 |
| `page` / `limit` | 分页 |

### DELETE `/test-strategy-results`
删除全部测试策略结果。

### GET `/test-strategy-results/:id`
获取单条测试策略结果详情。

### DELETE `/test-strategy-results/:id`
删除单条测试策略结果。

### POST `/test-strategy-results/test/:symbol`
测试指定 symbol 未平仓记录对应的平仓策略是否满足条件。

---

## 6. 现货交易对管理（Spots）

### GET `/spots`
查询现货交易对列表。

**Query 参数**

| 参数 | 说明 |
|---|---|
| `symbol` | 模糊搜索 |
| `symbol_type` | 类型筛选 |
| `enable` | 0/1 |
| `pin` | 是否置顶 |
| `sort` | `percent_change+/-` |

### POST `/spots`
新增现货交易对（默认 usdt=10，profit=100，loss=100）。

### PUT `/spots/:id`
更新现货交易对信息。

### DELETE `/spots/:id`
删除现货交易对。

### PUT `/spots/enable/:flag`
批量修改所有现货交易对开启状态。

### PUT `/spots/batch`
批量修改现货交易对参数（同 `/features/batch`）。

---

## 7. 新币抢购（Rush）

### GET `/rush`
查询新币抢购列表。

### POST `/rush`
新增新币抢购配置（默认 leverage=3，usdt=10，enable=0）。

### PUT `/rush/:id`
更新新币抢购配置。

### DELETE `/rush/:id`
删除新币抢购配置。

### PUT `/rush/enable/:flag`
批量修改所有抢购配置开启状态。

---

## 8. 价格预警（Notice Coin）

### GET `/notice/coin`
查询价格预警列表。

**Query 参数**

| 参数 | 说明 |
|---|---|
| `type` | 1=现货，2=合约 |

### POST `/notice/coin`
新增价格预警配置。

**Request Body（主要字段）**
```json
{
  "symbol": "BTCUSDT",
  "noticePrice": "60000",
  "side": "buy",
  "type": 2,
  "usdt": "50",
  "leverage": 3
}
```

### PUT `/notice/coin/:id`
更新价格预警配置。

### DELETE `/notice/coin/:id`
删除价格预警配置。

### PUT `/notice/coin/enable/:flag`
批量修改所有价格预警开启状态。

---

## 9. 监听币种（Listen Coin）

### GET `/listen/coin`
查询监听币种列表。

**Query 参数**

| 参数 | 说明 |
|---|---|
| `type` | 1=现货，2=合约 |
| `symbol` | 模糊搜索 |
| `listen_type` | 监听类型，如 `kline_base` |
| `enable` | 0/1 |

当 `type=1` 或 `type=2` 时，响应中会附加当前 `close` 价格。

### POST `/listen/coin`
新增监听币种（默认 klineInterval=1m，changePercent=1.1%，noticeLimitMin=5）。

### PUT `/listen/coin/:id`
更新监听币种配置。

### DELETE `/listen/coin/:id`
删除监听币种。

### PUT `/listen/coin/enable/:flag`
批量修改所有监听币种开启状态。

### GET `/listen/coin/kc-chart/:id`
获取 Keltner Channel 图表数据（150条K线，period=50）。

**Response**
```json
{
  "code": 200,
  "data": {
    "upper1": [...], "ma1": [...], "lower1": [...],
    "upper2": [...], "lower2": [...],
    "close1": [...], "high1": [...], "low1": [...]
  }
}
```

### POST `/listen/strategy-rule/test/:id`
测试指定监听币种的策略规则。

### GET `/listen/funding-rates`
查询合约资金费率列表。

**Query 参数**

| 参数 | 说明 |
|---|---|
| `symbol` | 模糊搜索 |
| `sort` | `+` 费率从高到低 / `-` 从低到高 |

### PUT `/listen/funding-rates/:id`
更新资金费率配置。

### GET `/listen/funding-rate/history`
获取资金费率历史（最近200条）。

**Query 参数**

| 参数 | 说明 |
|---|---|
| `symbol` | 交易对，必填 |

---

## 10. 订单记录（Orders）

### GET `/orders`
分页查询本地订单记录（side=open 的记录）。

**Query 参数**

| 参数 | 说明 |
|---|---|
| `symbol` | 模糊搜索 |
| `position_side` | LONG / SHORT |
| `type` | open（未平仓）/ close（已平仓）|
| `start_time` / `end_time` | 时间范围 |
| `page` / `limit` | 分页，limit 默认 100 |

### DELETE `/orders`
删除全部订单记录。

### DELETE `/orders/:id`
删除指定订单记录。

---

## 11. 配置文件

### GET `/config`
读取服务器上 `conf/app.conf` 文件内容。

**Response**
```json
{
  "code": 200,
  "data": { "content": "[app]\nappname = ..." }
}
```

### PUT `/config`
保存编辑后的配置文件；内容未变化时不重复写盘。内容变化时进行 INI 格式检查、原子替换，响应中返回 changed 和 restart_required。保存本身不会触发重启。

### POST `/config/restart`
手动重启以应用已保存的配置，仅在 restart_required=true 时执行。Go 进程通过自身二进制 re-exec 实现，不调用 PM2 命令；接口受 JWT 认证。返回的是「重启已发起」，不是已完成的保证。

**Request Body**
```json
{
  "code": "[app]\nappname = ..."
}
```

---

## 12. 合约账户（Futures Account）

### GET `/futures/account`
获取 Binance 合约账户资产信息（过滤余额为0的资产）。

### GET `/futures/positions`
获取 Binance 合约当前持仓（过滤空仓位）。

### GET `/futures/open-orders`
获取 Binance 合约当前所有挂单。

### GET `/futures/local/positions`
获取本地数据库缓存的合约持仓，附加未实现盈亏计算。

### PUT `/futures/local/positions/:id`
修正本地缓存的合约持仓数据。

### DELETE `/futures/local/positions/:id`
删除本地缓存的合约持仓记录。

### GET `/futures/local/open-orders`
获取本地数据库缓存的挂单（状态为 NEW 或 PARTIALLY_FILLED）。

### GET `/futures/market-notice-logs`
查询合约市场快速波动通知日志。

**Query 参数**

| 参数 | 说明 |
|---|---|
| `symbol` | 模糊搜索 |
| `window` | 时间窗口 |
| `direction` | 方向 |
| `start_time` / `end_time` | 时间戳（秒或毫秒均支持）|
| `page` / `limit` | 分页，默认 20 |

---

## 13. 策略模板（Strategy Templates）

### GET `/strategy-templates`
查询策略模板列表。

**Query 参数**

| 参数 | 说明 |
|---|---|
| `name` | 按类型筛选 |

### POST `/strategy-templates`
新增策略模板。

### PUT `/strategy-templates/:id`
更新策略模板。

### DELETE `/strategy-templates/:id`
删除策略模板。

### POST `/strategy-templates/test/:symbol`
用策略模板测试指定 symbol 的策略规则。

---

## 14. 系统日志

### GET `/system/logs`
读取 Go 内置应用日志。需 JWT 认证，支持日志级别、关键字和显示条数筛选。
原 `/start`、`/stop`、`/pull`、`/pm2-log` 已删除。服务进程管理仍通过 PM2 在系统外进行。
