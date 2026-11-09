# CODE_REVIEW_01-0_01-3 第三轮复审（R3）整改记录

时间：2026-11-08（北京时间），分支 `feat/trade-upgrade`，依据 `CODE_REVIEW_01-0_01-3-rereview2.md`。保留原审计文件原始结论。**仍未达到生产 T0 Gate，不进行真实订单测试。**

## R3-N1～N9 逐项处理

| 编号 | 修复、限制和实测 | 状态 |
| --- | --- | --- |
| **R3-N1 P1** | `reconcileClosedCycleFills` 已移到 closed 周期的零仓位证明门禁**之前**：即使 `FreshPositionSide` 获取明确零 row 失败，也只读补 `userTrades`、将原 `rest_verified=0` Fill 核实为 1；该周期仍 `incomplete`、不触发收益/风控。独立新鲜 REST 证明归零成功后才调用 `settleVerifiedFlat`。永久测试 `TestStageR3MissingExactZeroBackfillsFillsWithoutSettling` 覆盖两次恢复路径。 | **已修复** |
| **R3-N2 P1/P2** | 原 `main.go` 每分钟自动调用 AgentTrade 真实重挂的代码**移除**，新增**独立、默认关闭**的 opt-in 任务；只有明确设置运行环境变量 `BINANCE_AGENT_PROTECTION_AUTO_REPAIR=enabled` 才运行。内部仍要求同样 opt-in；所有新订单必须属于 `executed/protection_failed`、`RiskStatus=pass`、`ApprovedAt>0`、`ExecutedAt>0`、`RejectedAt=0`、`ExpiresAt>now` 的原始提案，且与持仓 SourceRef、symbol、side 绑定，存在可信 StopLoss，缺任一条件**不下单**。Rejected/risk_failed/expired/unapproved/never-executed 各自永久反例测试。 | **代码已加固，默认禁用；实盘授权仍未通过** |
| **R3-N3 P2** | `RepairActiveManagedProtections` 对无提案、不可核实、任一 TP/SL 失败的周期收集 error 并继续其它周期；仅在 context 取消时停止，最终返回 `errors.Join`，逐周期记录含“报警”的 Warning。永久测试验证前一个周期缺提案、后一个周期仍生成真实身份与数量匹配的 SL/TP。 | **已修复** |
| **R3-N4 P2** | 独立重挂任务使用 `context.WithTimeout(30s)` 并与原 Ownership 每分钟对账分离；先使用同一账户 `ReconcileAll` 重新核对**全部 Owner**，任一 `OrdersUnresolved>0`、返回 error、或没有 AgentTrade Owner summary，都拒绝自动重挂。原每分钟 Ownership REST 对账补 45s deadline，过期不启动原 Notice 保护恢复。永久测试验证其他 Owner 未核实也禁止自动恢复。 | **已修复逻辑门禁；真实交易所阻塞/网络仍需独立验证** |
| **R3-N5 P3** | 独立检查 StopLoss **和**已在提案中批准的第一档 TakeProfit；原 Stop 已足量时，TP 若取消或缺失，只重挂 TP、不重新提交 Stop；TP submit 错误不再被 `TakeProfitError` 静默吞掉。检查新单实际 ExchangeOrderID、PositionID、remainingQty 与订单状态。永久测试涵盖 TP-only 恢复、失败明确报错。 | **已修复** |
| **R3-N6 P3** | 没有 `ExactPositionSideSource` 的自定义/包装账户数据源禁止按批量快照进行 destructive **shrink/close/settle**（包括批量含明确零或较小正数）；缺失方向的 Binance source 失败时同样拒绝。缺失键按排序+**每分钟轮转**选择最多 6 次定向查询，避免后面的币种永久饿死；Warning 输出尚未核实的数量。永久测试涵盖无接口但明确 0/0.25、8 个币分两轮覆盖。 | **已修复** |
| **R3-N7 P3** | 独立的 opt-in 开关，单次最多自动处理 **2 个仓位周期**，通过各自 ClientOrderID 的既有幂等下单链。每次出单前必须成功写 `agent_trade_audits` 的 `protection_rearm_attempt`（position_id、account_id、intent、remainingQty、trigger、reason），确认真实交易所 order ID 后写 success；失败写 error/报警。默认关闭不会自动提交真实订单。审计记录可供后续仪表板，**本轮未增加新看板**。 | **主要治理代码已实现；全 Owner 和仪表板 Gate 未关闭** |
| **R3-N8 P3** | Lead `stream.go` 原动态只读测试、导入白名单及字符串警戒继续保留；新增 **完整 AST** 检查，`go_binance_futures/feature/api/binance` 只允许引用 `AccountID`、`AccountClient`、`LeadAccountID` 这三个只读身份/类型符号，禁止调用该包其他导出函数，并加负例变异测试（即使原 import 未改也必须被拒绝）。**这不是跨所有 helper 的完整静态调用图证明**，仍以 Lead 只读运行链的动态验证为最终 Gate。 | **守卫加固；不能宣称形式化证明** |
| **R3-N9 P3** | 01-0～01-3 的迁移证据使用本地明确命令与当次执行时间表述，不再以「当前工作区」否定旧审计的当时快照；R3-N1 的永久测试采用有 `FreshPositionSide` 但明确零缺失的生产型 source，而非只使用不实现该接口的 fake。 | **已修正** |

## 离线测试与风险界线

- 核心新增永久测试：`TestStageR3MissingExactZeroBackfillsFillsWithoutSettling`、`TestStageR3NoExactSideProviderCannotCloseManagedPosition`、`TestStageR3NonExactSnapshotCannotShrinkEvenWithPresentKey`、`TestStageR3ExactSnapshotRateBudgetRotatesAcrossEightSymbols`、`TestStageR3ProtectionRepairDefaultDisabled`、`TestStageR3InvalidProposalCannotRepair`、`TestStageR3FailedFirstCycleDoesNotBlockValidSecondCycle`、`TestStageR3MissingTPOnlyRearmsTPAndAudits`、`TestStageR3ProtectionRepairRequiresAllOwnerReconcileSuccess`；Lead AST 负例已整合到 `TestStageReviewLeadReadOnlyStreamCannotPersistTradeFills`。
- 受影响 Go 业务包完整测试、Race、Vet、主程序编译、补丁空白检查都需再次跑齐并如实记录最终 Gate。
- **DB Schema 仍为 v23，不新增表，不要求本轮对本地或生产 MySQL 再执行升级**。只用现有 `agent_trade_audits` 写恢复审计。运行中的旧二进制**没有被替换**，新 opt-in 机制并未在服务器启用。
- 没有写 `go_binance`、没有重启服务、没有主动请求真实 Binance、没有修改 `conf/app.conf`，没有自动 commit/push。

## 实盘 Gate 保留

1. 尚无真实多次部分成交的 WS 与 `userTrades` 逐 tradeId 对照；REST 手续费来源虽然已保护，不能视为实盘核验。
2. `BINANCE_AGENT_PROTECTION_AUTO_REPAIR=enabled` 只在另行批准、可人工干预、明确审计/限额以后才允许操作员设置；**本轮绝不启用**。现有 AgentTrade 提案如果已经过期，新的自动恢复会拒绝，需要人工重新验证并建立受控恢复流程，不能绕过过期门禁。
3. Notice 使用既有 `RepairNoticeAutoOrderProtections`；其它 Owner/Lead 均没有因此自动获得通用重挂能力。真实 Binance Algo 保护单的 order ID、触发价格、止损/止盈及仓位数量必须分别实测。
4. Binance symbol-scoped `PositionRisk` 对已归零方向是否返回明确行仍未获实盘证据。若不返回：Fill 可自动补确认，但 CLOSED 周期保持 `incomplete`、收益风控不触发。这是刻意的 fail-closed，而不是“已实盘成功”。
5. 未实现前端 `incomplete/settled` 可视化、自动重挂看板和全链 DingTalk 主动推送；01-4/01-5 未实施，T0 仍禁止启动。
