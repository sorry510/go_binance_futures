# CODE_REVIEW_Lead-03-1：Lead 账户读取与 Managed 归属校验

> **阶段状态：Lead-03-1 已完成；整个 Lead-03 尚未完成。** 基于四阶段方案，仍不允许生产真实交易。

## 一、已完成

新增 `feature/lead_trade_io.go`，不添加新的业务配置、页面、模拟预览、调度器或数据库表：

- `newLeadTradeIO`：仅允许 `LeadAccountID` 的专属 `AccountClient`，绑定独立 `futuresownership.BindAccount(lead)`；绝不回退 Main API Key。
- `bindReadHooks`：为现有 `accountTradeRunner` 绑定 `ReadPositions`、`ReadOpenOrders`、`SyncPositions`。**原有 Mock-only 校验未解除**，此时没有生产可执行的 Lead 自动交易循环。
- `readPositions`：真实 Lead 账户 `GetPositionFreshContext` 的只读适配。严格拒绝 BOTH、仓位方向与数量正负不符、重复仓位、非法数量/杠杆/开仓价/标记价及 NaN。
- `readOrders`：真实 Lead `GetOpenOrderFreshContext` 的只读适配。严格拒绝重复、无 ClientOrderID、无合法方向、非法数量与超量成交。
- `managedPositions`：Lead 账户范围查询未结订单与 `auto_strategy` Managed 仓位。发现任何 Pending/未知订单，当前轮次先安全失败，等待可靠对账。
- `verifyLeadManagedSnapshot`：只有账户为 lead、owner 为 auto_strategy、状态 active、开仓成本和数量合法的仓位才可进入共用平仓策略；手动加仓不会计入 Managed 数量，账户数量不足、需恢复状态和重复归属均拒绝。

## 二、本地测试证据

新增 `feature/lead_trade_io_test.go` 验证：
- Main 和 nil 客户端不得绑定 Lead；无验证不能修改共享 Runner 的读取钩子。
- Hedge LONG/SHORT、方向、数量、重复仓位、无效价格、异常杠杆均安全失败。
- 挂单 ClientOrderID、数量、成交数量不一致时拒绝。
- Managed 仓位不归属手动加仓，数量不足、跨账户/需恢复状态/未结订单拒绝。
- HTTP Mock 验证 Lead 私有读取只发起签名 GET，使用 Lead Key，单轮分别读取一次仓位与挂单端点。

实际运行：
- `go test ./feature -run '^TestLead03' -count=1 -p=2`：通过。
- `./scripts/test-go.sh -p=2 -timeout=120s`：**EXIT=0**，Go 全部应用包通过。
- `go test -race ./feature -run '^TestLead03|^TestStage46' -count=1 -p=2`：**EXIT=0**，无 Data Race（macOS 链接警告非阻断）。
- `go vet ./feature ./service/leadaccount ./service/futuresownership ./feature/api/binance`：**EXIT=0**。
- `git diff --check`：通过。

所有 Binance HTTP 均为本地 Fake/Mock，没有向交易所发送任何真实交易订单。

## 三、未完成与下一轮目标

1. **Lead-03-2 执行器接线**：共享 Runner 的开仓前检查、SubmitOpen、SubmitClose 尚未绑定现有 `LeadExecutionAdapter` 的真实执行路径。需要账户权限、WS、Portfolio、风险额度及资金最终验证，不能靠删除现有门禁解锁。
2. **Lead-03-3 保护单/成交/恢复**：TP/SL、Fill、部分成交、未知提交 Lookup、重启与 WS 重连恢复未完成真实端到端验收。
3. **Lead-03-4 调度与限额**：Managed 退出优先、新开仓开关、币种隔离的 Pending 恢复、REST Weight/429/418 需在实际执行循环下验证。目前任一未结订单导致整个轮次安全中断，仍需改为不会长期阻止其他币种安全退出的逻辑。
4. **生产仍锁定**：`LeadDurableOpenGate`、`/start`、Stage 7 权限及原 Runner 生产 Mock-only 限制未改变；没有新增生产启动任务。真实 Binance Lead Portfolio 和 Testnet/主网交易均尚未验收。

**结论：Lead-03-1 已完成；Lead-03 全阶段仍为进行中。** 下一轮继续接通既有执行器及真实订单闭环，不开发独立模拟预览或多余审批、配置层。
