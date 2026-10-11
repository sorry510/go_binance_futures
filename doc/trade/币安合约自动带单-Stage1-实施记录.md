# Binance 合约自动带单 Stage 1 — 多账户基础设施实施记录

> 日期：2026-10-09
>
> 状态：**Stage 1 离线基础实现完成**；后续账户级 Ownership/WS 持久化与带单真实下单均**未启用**。本阶段的测试全部使用测试 HTTP 服务器和已有 Fake/Mock，无 Binance 真实资金交易。

## 1. 本阶段完成内容

1. 新增 `AccountID`（`main`、`lead`）与 `AccountClient`：每个 AccountClient 独立保存 `*futures.Client`、signed mutex、账户仓位/挂单缓存、交易配置缓存和时间偏移。SDK 签名调用序列化避免内部状态竞态。
2. 新增 `NewLeadAccountClient`：明确使用 Lead API Key/Secret、生产 `https://fapi.binance.com`，通过现有 proxy HTTP Client 与 `binanceapiusage` 统一 IP API Budget 接入。**当前没有调用入口自动创建真实 Lead Client**，也不从 `app.conf` 读取 Lead Secret。
3. 新增独立账户的私有 API 封装：余额、仓位/新鲜仓位、挂单/新鲜挂单、普通与 Algo 订单提交/查询/撤单、杠杆与逐仓配置、listenKey 获取/续期。所有私有请求经自己的 SDK Client；行情、价格、历史 Kline 仍共用现有公共客户端和缓存。
4. 新账户状态缓存独立（独立 singleflight、generation/失效、完整读快照、`EnsureTradeConfigContext`）：Main 与 Lead 对同币杠杆设置、仓位/挂单均不共享缓存。Main 的旧入口/旧缓存**保持原路径不变**，降低本阶段回归风险；Stage 3 Main 运行时再迁移到统一注入式入口。
5. `futuresownership.BinanceOrderBroker` 支持可选 `Account *binance.AccountClient` 绑定。绑定时，普通和 Algo 下单、查询、撤单以及 Algo 已触发实际订单查询均使用对应账户；没有 Account 时按现有 Main 入口执行，以保持历史调用兼容。
6. 实现专用的 **只读 SAPI** 签名接口：`LeadTraderStatus` / `LeadTradingSymbols`；仅对 `AccountID=lead` 允许调用，签名请求直达 `api.binance.com`，身份、白名单 API 成功码 fail-closed；错误信息不包含 URL、查询签名、Secret、响应正文。
7. `NewLeadAccountClient` 带账户级 10s / 18 次保守发单配额（基于币安传统 Lead Key 20/10s 约束，预留 2 个请求空间）。达到预算在 HTTP 前以 `ErrBudgetDeferred` 拒绝；原系统 V4-5 全局 IP/weight 预算继续运行，公开行情不重复订阅。
8. Lead 私有 WS 本阶段只准备独立账户的 listenKey 获取/续期能力；**未创建第二条 User Data WS，也未改变单账户镜像数据表**，这些在 Stage 2/5 继续。

## 2. 实际变更文件

| 文件 | 内容 |
|---|---|
| `feature/api/binance/account_client.go`（新） | AccountID / AccountClient、新 Client 工厂、账户签名方法及私有 API |
| `feature/api/binance/account_state.go`（新） | 账户专属仓位/订单与交易配置缓存 |
| `feature/api/binance/lead_order_limiter.go`（新） | Lead Key 订单预发限流（18/10s） |
| `feature/api/binance/lead_readonly.go`（新） | SAPI 用户状态与交易白名单的签名查询 |
| `feature/api/binance/account_client_stage1_test.go`（新） | 双账户签名/仓位缓存/杠杆缓存/时间偏移/限流/SAPI Mock 测试 |
| `service/futuresownership/execution.go`（修改） | BinanceOrderBroker 可绑定账户，Algo 的二次订单查询保持相同绑定 |
| `service/futuresownership/account_broker_stage1_test.go`（新） | 双账户正常单 Submit/Lookup/Cancel、Algo Lookup 与触发后实际订单路由 |
| `doc/trade/币安合约自动带单-Stage实施方案.md`（修改） | 更新 Stage 1 状态 |
| 本文件（新） | 实施与限制记录 |

**未修改**：`feature/feature.go::StartTrade`、策略规则、选币逻辑、Ownership 模型、数据库、`conf/app.conf`、前端、调度任务。也未执行数据库 sync、真实 REST 下单或推送代码。

## 3. 验收记录

- `go test ./feature/api/binance ./service/futuresownership ./service/binanceapiusage ./binanceproxy`：**通过**。
- `go test -race ./feature/api/binance ./service/futuresownership ./service/binanceapiusage ./binanceproxy`：**通过**，macOS 链接器对部分测试对象报告 `LC_DYSYMTAB` warning，但进程 exit=0（变更后还会复测）。
- `go test ./feature ./controllers ./service/agenttrade -run '^$'`：**通过**（依赖编译验证，未运行这些模块的单元测试）。
- 阶段新增：Main/Lead 相同 symbol 的行情查询可复用，但每账户仓位缓存各自缓存正确结果；主账户缓存失效与 Lead 不交叉；交易配置对每账户独立提交。
- 测试：Headless HTTP Fixture 验证 Binance 签名头/查询签名、SAPI userStatus/leadSymbol 响应与主账户拒绝访问；限流 19 次请求前拒绝，另一个账户预算独立。
- 测试：绑定 Account 的 Broker 的 Submit/Lookup/Cancel 都携带对应 API Key，Algo 触发后查询到的实际订单也走对应 API Key。
- 测试：-1021 时间戳自修正不会改变另一个账户 Client 时间偏移。

## 4. 阶段边界与下一阶段前置条件

- **未将 Lead Broker 注入 `Executor` 真实交易**：当前 `futuresownership.Service` 与数据库无 `account_id`，即使 AccountClient 已能以绑定 Key 调用订单 API，也**不得**直接使用 Main Ownership Service 驱动 Lead 交易，这要等 Stage 2。Stage 1 测试只直接使用绑定 Broker + Mock HTTP。
- **Main 仍走现有全局函数包装**：保留 `binance.GetPositionContext`/`CreateOwnedOrder` 等原有 Main 语义；新增 AccountClient 是面向新账户的实例化适配层。待 Stage 3 将 main/lead 主交易循环都迁移到统一依赖注入，不允许临时换全局 Key。
- **SAPI 仅 Mock 测试**，缺少真实 Lead Portfolio Key 实测：身份、白名单、fapi 与 portfolio 的映射/权限仍归 Gate 0-LIVE。
- **安全配置与用户录入未实现**：Stage 6 完整凭证安全存储及前端入口；Stage 1 只提供传入凭证的客户端工厂，不写 DB、不让用户把 Secret 发在聊天里。
- **未实现**独立 User Data WS、账户维度持仓/挂单镜像和多账户 Ownership；分别交 Stage 2、5。
- **后续注意**：SDK 全局测试网 WS 选择、lead 与 main 并行 WS generation 隔离、Binance per-key 响应限速统计仍需 Stage 5 完整审计；当前本阶段并未开启 Lead 私有 WS。
- **只有 Stage 2** 实现 `account_id` 持久化、仓位/挂单/所有 Owner 作用域隔离、正确迁移后，才能考虑把 Lead Broker 接入 Executor；Stage 3/4 继续独立启停与完整开平仓逻辑；Stage 7 在明确授权后才做真实带单订单。

## 5. Stage 1 Gate 结论

**离线 Gate 1：通过（新账户基础 API、签名缓存/Broker 路由与单测完成）**；**真实 Lead Key Gate：尚未验证，实盘阻塞**。Stage 2 可以在模拟环境下继续开发，但不得依据此记录开启真实带单交易。

