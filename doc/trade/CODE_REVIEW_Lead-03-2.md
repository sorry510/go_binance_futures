# CODE_REVIEW_Lead-03-2：共用策略 Runner 与现有 Lead 执行器接线

> **状态：Lead-03-2 的桥接实现及本地隔离测试完成；Lead-03 整体尚未完成。**
> 以 `CODE_REVIEW_Lead-03-1.md` 的独立 Lead 账户读取、真实仓位和 Ownership 校验为基础。
> **无独立模拟预览、无审批流程、无额外配置、无新业务表。**

## 1. 目标和代码

1. `service/leadaccount/strategy_execution.go`：在原有 `LeadExecutionAdapter` 上添加与主合约策略规则相匹配的参数转换、`PreflightStrategyOpen`、`ExecuteStrategyOpen` 与 `BoundTo`。没有新 Broker：`ExecuteStrategyOpen` 始终委托已有的 `ExecuteOpen`，二次进行持久化开仓权限、身份、资金、账户/WS、交易规则及 Ownership 校验。`BoundTo` 要求严格相同的 Lead Client 指针，不允许同类型但不同凭证的账户混用。
2. `feature/lead_trade_execution.go`：把现有 `accountTradeRunner` 的 `PreflightOpen`、`SubmitOpen`、`SubmitClose` 与 Lead 已有执行适配器对接；同时复用既有选币、DSL、平仓策略及 `lead_trade_io.go` 的 Lead 专属读取钩子。**没有新调度器**，也未解除 `accountTradeRunner.validate` 对 Lead 的 test-only Mock Permit 要求。
3. `feature/api/binance/lead_symbol_cache.go`：仅增加 `BoundTo` 身份校验。白名单必须与执行器、仓位读取使用相同的 Lead Client；过期、禁用或凭证已轮换时拒绝新开仓。
4. 完成准确的一次性开仓关联：预检查使用已取整的参考价格，MARKET 真正下单价为 `0`；LIMIT 下单价必须与参考价格相同；`Quantity` / `PositionSide` / `OrderType` / `Side` 均必须精确匹配；一次提交后票据立即失效。任何新的失败或取消的预检查都会清除旧票据，避免重复 POST。
5. 每个内部钩子均设置有界上下文超时（30 秒）。当 Binance 提交结果不确定时，保持 Ledger 的未知状态及现有恢复约束，**不在桥接层自动重发**。
6. 已有 Managed 退出**不依赖 `AllowNewOpens` 或白名单**；退出只向 `ExecuteManagedClose` 提交相反方向的 MARKET 请求，最终平仓数量仍由绑定的 Lead 执行器再次核对真实 Binance 仓位与 Managed Qty。绝不借用 Main 的撤单函数。
7. 仅在交易所响应 ID、Lead ClientOrderID 前缀、接受状态均可信时，把回执转换成原有 `futures.CreateOrderResponse`，并记入本轮待处理仓位槽位；这不等于 Fill 已成交。为避免误报，Lead 下单“成功”的通知仍等待以真实 Fill 为依据的后续阶段。

## 2. 不变量和审查范围

- **生产订单仍不可达**：Stage 3 Lead runner 的 Mock Permit 只能在 `_test.go` 构造，且 `LeadDurableOpenGate()` 始终拒绝实盘；本轮不启动后台任务、不增加 HTTP 接口、不启用 `/futures/lead/start`。
- `feature/account_trade_cycle_stage4_6_test.go` 的生产 AST 审查只放行 `feature/lead_trade_execution.go` 的两个已审查委托调用，以及 `service/leadaccount/strategy_execution.go` 对原 `ExecuteOpen` 的**唯一委托**；其他生产代码调用仍拒绝。
- `service/leadaccount/stage46_integration_test.go` 更新了公开方法的**精确白名单**，仅加入 `BoundTo`、`PreflightStrategyOpen`、`ExecuteStrategyOpen`，没有添加 Enable、Resume、无条件取消订单或 ResetPending。
- Main Runner / Main Binance Client 及 `app.conf` 不变；执行器从未回退 Main。
- `EnsureConfig` 不用 Main 的可写全局 API 去修改 Lead 的保证金/杠杆。若 Lead 当前设置与策略参数不符，由真实风控校验拒绝开仓，不能私自改变配置。

## 3. 已通过的本地隔离测试

新增 `feature/lead_trade_execution_test.go`：

- MARKET LONG、LIMIT SHORT：预检查价格与真实请求的精确对应、双向的 `OrderRequest` 构造。
- 请求数量/价格/方向/类型不匹配、上一次票据重用、预检查失败/取消、超时后重复提交：全部拒绝。
- Binance 错误回执、错误账户 ID、拒绝状态、无可信 ID：不能记录为成功。
- 关闭新开仓和 Lead 白名单失效，Managed CLOSE_SHORT 仍到达有权限的执行端口；手动 Owner 和零数量不能平仓。
- 用真正的 `runAccountTradeCycle` 和**测试专用 Mock Permit**，验证 LONG/SHORT 同轮次发送两次独立的受控请求；禁用新开仓后两个 Managed 方向均继续进入平仓钩子。
- 真实 `NewLeadExecutionAdapter` 在未经 Stage 7 授权时仍拒绝开仓和平仓，无 HTTP/真实交易所下单。

新增 `service/leadaccount/strategy_execution_test.go`：

- 策略参数 → `RiskOpenOrder` 的精确转换，拒绝错误侧、无效数量/价格、未知类型、非法杠杆/USDT/保证金。
- 严格相同账户实例的 `BoundTo`，拒绝 Main 和其他 Lead 凭证。
- 实际 `ExecuteStrategyOpen` 仍受原始 `ExecuteOpen` 持久门禁阻止；不接受伪造 Owner、Symbol。

### 最终命令结果（2026-11-17，本地）

| 检查 | 实际结果 |
| --- | --- |
| `./scripts/test-go.sh -p=2 -timeout=120s` | **EXIT=0**，完整 Go 包回归 |
| `go test -race ./feature ./service/leadaccount -run '^TestLead03\|^TestStage46\|^TestStage6ReviewProductionLeadIngressImportAllowlist' -count=1 -p=2` | **EXIT=0**，无数据竞争；macOS 链接器有非阻断 `LC_DYSYMTAB` 警告 |
| `go vet ./feature ./feature/api/binance ./service/leadaccount ./service/futuresownership` | **EXIT=0** |
| `git diff --check` 与本轮 Go 文件 `gofmt -l` | **通过**，无未格式化文件 |
| 前端 | 本轮未修改 Vue，也无需重构前端/重新生成 `static` |
| 交易所和数据库 | **未发送任何真实订单；未操作生产数据库；未修改 `app.conf`** |

## 4. 未完成项目（不能标记完整 Lead-03 结束）

- **Lead-03-3：真实成交、保护单和恢复闭环**。必须在 Fake/Mock 上跑完整 Fill、部分成交、TP/SL、重复事件、保护单恢复、未知提交 Lookup、重启和 WS 重连后再验收。
- **Lead-03-4：生产运行接线、币种级异常隔离、API Weight**。当前只完成了执行器与共享策略的离线桥接；实际 Lead 定时调度仍未建立，不允许因完成测试便开启真实交易。
- **03-1 保守限制**：任意未决 Managed 订单目前会使该轮 Ownership 检查安全中断；需要在安全对账前提下允许其他币种已有 Managed 仓位继续退出。
- **Gate 未解除**：`/start`、`LeadDurableOpenGate`、Stage 7 实盘权限及 Mock-only 验证都保持原样；没有真实 Binance Lead 主网验证或成交证据。
- Main + Lead 同时运行的 1m/5m HTTP 权重、429/418 退避以及订单频率 P95 仍是 Lead-04 的实测内容。

**阶段结论：03-2 共享执行器桥接与自动化测试完成；Lead-03 完整订单闭环未完成。**
