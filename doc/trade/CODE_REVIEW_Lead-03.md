# CODE_REVIEW_Lead-03：合约带单共享交易闭环

> **Lead-03 本地代码接线与自动化测试完成；真实 Binance 资金交易仍由硬锁阻止，生产启用属于 Lead-04。**
> 个人使用：不新增第二个定时器、模拟预览、影子订单表或独立保护单状态机。

## 一、交易链路

1. `main.go` 原每 2 秒调度 `feature.StartTrade`，本轮保留同一个入口；`runLeadOnSharedTradeTick` 直接在该入口调用，不增加第二个调度器。
2. `feature/lead_trade_runtime.go` 先验证既有 Lead WS 观察器、`LeadDurableOpenGate`。该 Gate **目前恒拒绝真实交易**，默认没有 Lead 私有 REST 扫描或订单提交。
3. `CurrentLeadTradeSession` 复用 WS 观察器相同 Lead Client、独立白名单及执行器。缓存每个 Host 一份，SAPI 刷新不持有观察器生命周期锁，旧凭证轮换后不可复用。
4. 通过现有 `runAccountTradeCycle` 共享 Main 的选币、指标 DSL、LONG/SHORT、CLOSE_LONG/CLOSE_SHORT、收益率 TP/SL 与策略退出规则。Lead 私有余额、挂单、Order/Fill、Managed 和客户端凭证单独按 account_id 隔离。
5. `lead_trade_io.go`：BTC 的未确认订单只阻止 BTC 重复开仓/危险退出，不会掩盖已核实的 ETH Managed 退出；跨账户/非法身份直接拒绝。
6. `strategy_timeout.go`：复用 Main 的超时开仓单判定，仅对 Lead + auto_strategy 的确实超时且尚有剩余数量的订单，调用原 Ownership Reconcile/Cancel；不撤用户手动订单。
7. 正常 Binance REST 回执为**交易所已接单**；`ORDER_TRADE_UPDATE` 的真实状态、`TRADE` WS 逐笔 Fill 通过原 Main 写入器按 `account_id=lead` 入库。未知提交按原 `clientOrderId` 查单，不重复 POST。

## 二、TP/SL 与 Main 一致

Main 自动策略的 TP/SL 主要是在共享循环里依据已配置收益率判断并发送 CLOSE_* 平仓请求，并非每次开仓都必须另建两张交易所 Algo 条件单。Lead 复用同一套 `evaluateSharedTradeExit` 和 `runAccountTradeCycle`；不恢复此前已删除的独立保护单规划器。

## 三、本地测试

- `TestLead032FullSharedCycleUsesBoundAccountExecutorBothDirections`：共用循环正确产生 LONG/SHORT。
- `TestStage46SharedExitSignalsBothSidesEvenWhenLeadPaused`、`TestLead032FullSharedCyclePausedStillClosesBothManagedLegs`：TP、SL、AutoStop、暂停新开仓后的 Managed 退出。
- `TestLead03IndependentPendingDoesNotStopUnrelatedManagedExit`：BTC 未决订单隔离，ETH 可继续退出，BTC 不重复开仓。
- `TestLead03TimeoutOnlyCancelsIncompleteAutoOpen`、`TestLead03TimeoutCannotMutateWithoutLiveAuthorization`：超时撤单筛选和独立账户授权。
- `TestLeadMainParityAcknowledgedStopAndTargetWithoutGlobalLock`：成功订单不产生全局待对账锁，同一个 ID 不重复下单。
- `TestLeadMainParityUnknownSubmitStillRequiresOrderLookup`：提交不确定时查原 ID。
- `TestStage2UserDataMirrorIsolatedForSameSymbolAndOrderID`：Main/Lead WS 同币同交易所订单 ID 也不能交叉记录。
- 原 Stage 6/4-6 AST 审查：仅新增受审核的 Lead Gate、共享 Tick 和观察器会话连接；禁止控制器解锁、异常写入入口。

**2026-11-17 本地执行结果：**
- `./scripts/test-go.sh -p=2 -timeout=120s`：完整 Go 回归 EXIT=0。
- `go test -race ./feature ./service/leadaccount ./feature/api/binance -run '^TestLead03|^TestStage46|^TestStage6ReviewProductionLeadIngressImportAllowlist|^TestLeadMainParity' -count=1 -p=2`：EXIT=0，无数据竞争；macOS 链接器的 LC_DYSYMTAB 警告不影响测试。
- `go vet ./feature ./service/leadaccount ./feature/api/binance ./service/futuresownership`：EXIT=0；`git diff --check` 和 Go 格式检查通过。

上述测试均为本地 Fake、SQLite 和 Mock，没有向 Binance 主网发送订单，也没有更改生产 MySQL、`conf/app.conf` 或已运行服务。

## 四、Lead-04 必须验收

真实 Lead Portfolio/权限、独立签名 REST/WS、Hedge 与资金、部分成交及触发 TP/SL 后的真实 Fill、Main+Lead 两账户 2 秒轮次的 API Weight/429/418、重启恢复。最终只有用户明确授权后才能解除目前永久的 `LeadDurableOpenGate` 和 Stage7 生产许可；代码编译/单测成功**不等于**可以实盘。

**结论：Lead-03 代码复用与本地自动化测试完成，生产真实交易验收在 Lead-04。**
