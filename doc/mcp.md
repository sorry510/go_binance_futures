# MCP 服务说明

## 概述

本项目的 MCP 服务仅提供 **Streamable HTTP** 方式，不提供 stdio、本地命令或独立本地端口方式。

- MCP 地址：`http://<host>:<web.port>/mcp`
- 默认示例：`http://localhost:3333/mcp`
- 开关：`conf/app.conf` 中的 `mcp::mcp_enable`
- 鉴权：使用 `/login` 返回的 JWT，在 MCP HTTP 请求中添加 `Authorization: Bearer <token>`
- 工具总数：12

登录接口本身不会注册为 MCP 工具。MCP 客户端需要先通过普通 HTTP 登录接口获取 JWT。

MCP 使用显式白名单，不会自动暴露项目中的其他 HTTP 接口。当前仅保留合约交易对与市场强平订单、币种通知和行情监听三个分类。

## 启用方式

在本地真实配置中自行设置：

```ini
[mcp]
mcp_enable = 1
```

启动项目后，所有 MCP 初始化、工具发现和工具调用请求都必须携带登录 JWT。

## 统一参数

所有 MCP 工具使用相同的输入结构：

```json
{
  "path_params": {
    "id": "12",
    "flag": "1"
  },
  "query": {
    "page": 1,
    "limit": 20,
    "symbol": "BTCUSDT"
  },
  "body": {
    "enable": 1
  }
}
```

- `path_params`：替换路径中的 `:id`、`:flag`；对应工具存在路径参数时必填。
- `query`：对应 HTTP 接口的查询参数。
- `body`：对应 HTTP 接口的 JSON 请求体。
- 未使用的部分可以省略。

统一输出：

```json
{
  "status_code": 200,
  "body": {
    "code": 200,
    "data": {},
    "msg": "success"
  }
}
```

## 工具列表

### 合约交易对

| MCP 工具 | HTTP 方法 | HTTP 路径 | 说明 |
|---|---|---|---|
| `futures_symbols_list` | GET | `/features` | 查询合约交易对 |
| `futures_liquidation_orders_list` | GET | `/futures/liquidation-orders` | 获取市场强平订单 |

调用示例：

```json
{
  "name": "futures_symbols_list",
  "arguments": {
    "query": {
      "page": 1,
      "limit": 20,
      "symbol": "BTCUSDT"
    }
  }
}
```

获取 BTCUSDT 市场强平订单：

```json
{
  "name": "futures_liquidation_orders_list",
  "arguments": {
    "query": {
      "symbol": "BTCUSDT",
      "side": "SELL",
      "min_notional": 10000,
      "page": 1,
      "limit": 20
    }
  }
}
```

可用查询参数：`symbol`、`side`、`start_time`、`end_time`、`min_notional`、`page`、`limit`。

### 币种通知

| MCP 工具 | HTTP 方法 | HTTP 路径 | 说明 |
|---|---|---|---|
| `coin_notice_list` | GET | `/notice/coin` | 查询币种通知 |
| `coin_notice_create` | POST | `/notice/coin` | 新增币种通知 |
| `coin_notice_update` | PUT | `/notice/coin/:id` | 更新币种通知 |
| `coin_notice_delete` | DELETE | `/notice/coin/:id` | 删除币种通知 |
| `coin_notice_set_all_enable` | PUT | `/notice/coin/enable/:flag` | 批量开启或关闭币种通知 |

更新示例：

```json
{
  "name": "coin_notice_update",
  "arguments": {
    "path_params": {
      "id": "12"
    },
    "body": {
      "enable": 1
    }
  }
}
```

### 行情监听

| MCP 工具 | HTTP 方法 | HTTP 路径 | 说明 |
|---|---|---|---|
| `coin_listen_list` | GET | `/listen/coin` | 查询行情监听配置 |
| `coin_listen_create` | POST | `/listen/coin` | 新增行情监听配置 |
| `coin_listen_update` | PUT | `/listen/coin/:id` | 更新行情监听配置 |
| `coin_listen_delete` | DELETE | `/listen/coin/:id` | 删除行情监听配置 |
| `coin_listen_set_all_enable` | PUT | `/listen/coin/enable/:flag` | 批量开启或关闭行情监听 |

批量启停示例：

```json
{
  "name": "coin_listen_set_all_enable",
  "arguments": {
    "path_params": {
      "flag": "1"
    }
  }
}
```

## 已移除的 MCP 工具

以下功能仍可保留其原有 HTTP 接口，但不再通过 MCP 暴露：

- 合约预筛选、系统、通知、测试策略、现货交易对、抢购。
- 订单、合约账户、资金费率套利、策略模板。
- 合约交易对中除“查询合约交易对”和“获取市场强平订单”以外的操作。
- 行情监听中的 Keltner Channels 图表、合约资金费率配置与历史、策略规则测试。

## 安全说明

- `/mcp` 不在 JWT 白名单中，必须使用有效登录 Token。
- MCP 工具只允许访问代码白名单中预先注册的固定路径，不能传入任意 URL。
- MCP 内部调用会原样复用当前 MCP 请求的 `Authorization` 请求头。
- 新增、更新、删除和批量启停工具会改变数据，调用前应确认参数与目标环境。
- 不要在说明文档、日志或 MCP 对话中输出 Token、Binance Key、数据库密码等敏感信息。
