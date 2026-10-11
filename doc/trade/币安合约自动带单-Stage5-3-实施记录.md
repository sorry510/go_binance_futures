# 币安合约自动带单 — Stage 5-3 启动恢复、持久化 CAS 与权威对账

日期：2026-10-10 至 2026-10-11（按本地测试环境）
状态：**离线实现、生产尚未接线；Mock 测试通过。真实 Binance/Testnet 环境及 Gate 5 未验收**。

## 实现与验收

- 在 `service/futuresownership/service.go` 增加 `ReserveSubmission`：在 **任何 Broker HTTP 写入之前**按 account_id+ClientOrderID+PENDING 原子 CAS 为 SUBMITTED，跨实例仅一个请求取得提交资格；另一实例只能查单/对账，不能盲重发。通过 `Executor.Execute` 共同约束 Main/Lead，回归中不得改变 Main 结果。
- `OutstandingOrders` 不采用仅取最近 200/500 条的截断扫描，读取本账户全部未决订单并设置安全行数上限。唯一 ClientOrderID、Owner、Side/Intent 仍由既有 Ownership 持久化事务管理。
- `service/leadaccount/reconciler.go`：显式调用的 Lead 恢复器读取所有未决单、**包括 CLOSED** 的全部受控持仓历史、REST 全量持仓与挂单，通过同 ClientOrderID 的 Broker Lookup 累计成交幂等写回 Ownership；发现未知挂单、重复/不一致或账户归属不明立即阻断，绝不提交新订单。
- 启动流程先完成 DB/REST 恢复审计，再启动私有 WS；仅完整持久对账且得到**较新且无 WS 启动事件缺口**的镜像后，可经私有证明函数清除同一实例的待对账标记；不得清除亏损/身份熔断或设置 Stage 7 授权。
- 验证：`TestStage53DurableCASOnlyOneWorkerCanReserve`（32 并发）、`TestStage53RestartResolvesKnownFillButDoesNotEnableTrade`、`TestStage53MissingExchangeEvidenceStaysBlockedAcrossRestart`、`TestStage53IncludesClosedHistoryAndRejectsUnknownExchangeOrders`、`TestStage53OnlyNewerNoGapWSAndDurableProofClearsPending`。
- **限制**：数据库对账方法会在被明确调用时更新 Ownership；本轮只对 SQLite 内存测试库执行，未运行生产 MySQL 迁移。真实跨进程/故障切换及 Portfolio REST 返回尚需部署验收。

## 安全约束

未修改 `app.conf`、未执行 `sync db`、未访问真实 Lead API Key 或实际下单、未自动 commit/push。Main 原有行情与交易仍为独立代码路径；这份离线记录不构成 Stage 7 实盘授权。

**审计后说明**：所有持久恢复信息仍只回写受控 Ownership 订单/仓位，未新增独立的持久恢复审计流水。Stage 6/7 需在不破坏账户隔离的前提下设计操作审计与重启跨进程验证。
