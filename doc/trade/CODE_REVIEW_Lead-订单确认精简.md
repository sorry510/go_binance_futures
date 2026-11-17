# CODE_REVIEW：Lead 订单确认对齐主合约（2026-11-17）

## 目标

个人使用，不另建 Lead 模拟调度/保护单状态机。复用 Main 的账户级订单和 WS 机制。独立 Lead API Key、Ownership、原 `clientOrderId` 幂等与 API 限额保留。

## 已修改

1. `LeadExecutionAdapter.ExecuteOpen` / `ExecuteManagedClose`：只有调用原 `futuresownership.Executor.Execute` 返回异常时才设置 `pendingReconcile`。Binance 正常接受（`NEW`、`FILLED` 等）不再额外全局锁定下一笔订单。对未知提交结果仍保留原始查询和禁止重复 POST。
2. `service/leadaccount/read_only_host.go`：在显式开启 Lead WS 观察器时使用 Main 已有的账户级 WS 本地持久化方法；不自动归属手工仓位，也不提交 Binance 订单。`feature/api/binance/account_client.go` 仅增加 Lead 客户端身份限定的包装方法。
3. 删除未接入生产的 `feature/lead_trade_scheduler.go`、`service/leadaccount/protection_plan.go`、`service/leadaccount/protection_recovery.go` 及其测试，撤回 `leadCycleOpenBlocked` 的轮次级全局阻断状态；原有共用 Runner 和现有 Lead 执行器保留。
4. 原有独立账户、Binance 规则、实际 Managed 仓位数量、交易所 API/WS 和原始 Order/Fill 账本仍复用。无额外审批、配置、DB Schema 或前端页面。

## 确认语义

- **下单成功**：Binance REST 正常返回并提供可用订单 ID / `clientOrderId`，表示交易所接受订单。
- **状态和成交**：`ORDER_TRADE_UPDATE` 的 `NEW`、`PARTIALLY_FILLED`、`FILLED` 等状态按原主合约写入；只有实际 `TRADE` WS 事件产生逐笔 Fill。**已接受 ≠ 已成交**，这是主合约已有的数据区分。
- **异常**：网络错误且无法确认时，使用原 `clientOrderId` 查单，不能因为本地没收到响应就新生成一笔重复交易。
- **保护单**：正常 STOP/TP ACK 不会互相阻断，若本地记录或币安状态未知仍按原订单状态检查。

## 本轮本地验收

- `TestLeadMainParityAcknowledgedStopAndTargetWithoutGlobalLock`：LONG/SHORT 两种方向，STOP 与 TP 依次正常提交，同一 ID 再调用不重复 POST。
- `TestLeadMainParityUnknownSubmitStillRequiresOrderLookup`：超时 + Lookup 不可用时保持查询阻断。
- `TestStage2UserDataMirrorIsolatedForSameSymbolAndOrderID`：Main 与 Lead 对同币同订单号分别写入状态和实际 WS Fill，不互相覆盖。
- 完整 Go 回归、`-race`、`go vet` 等按实际终端输出验证。

**现状：** WS 观察器的本地写入与正常确认逻辑已对齐 Main；Lead 生产自动策略尚未接入启动，TP/SL 自动创建/触发恢复未进行真实交易验收。`LeadDurableOpenGate`、`/futures/lead/start` 的生产锁定未改变。
