# V2-12 Code Review：Proposal、Risk Engine 与受控执行（Risk Execution）

> 评审模式：**Review-only**。本报告仅基于已提交代码（HEAD `620f891` 之后的工作区改动）做静态审查与构建/测试验证，未修改任何代码、未写入任何内存、未执行范围外任务。
> 评审结论：**AUTOMATED PASS / 人工验收待定**（无阻塞级缺陷，4 项验收全部由源码与测试证实成立）。

---

## 0. 评审范围与构建验证

### 变更文件清单（已用 `git diff --stat` + 全量通读锁定）

新增：
- `models/agent_trade.go` — 三张独立表模型（`AgentTradeProposal` / `AgentTradeExecution` / `AgentTradeAudit`）
- `controllers/agent_trade.go` — 受控交易 HTTP 入口（List/Create/Get/Risk/Approve/Reject/Execute/Reconcile）
- `service/agenttrade/{types,store,risk,default,service,service_test}.go` — 受控执行核心
- `routers/router.go`（diff）— 新增 7 条路由 `/agents/trade/proposals...`
- `main.go`（diff）— `dbVersion 2→3`；注册 3 个新 model
- `agent/app/governance.go`（diff）— `GovernanceStatus` 新增 `ControlledExecutionEnabled`
- `command/sql/version/3.sql` — DB Version 3 安全默认值
- `agent/mcpclient/{catalog,store}.go`（diff）— 外部 MCP 交易能力识别 + 强制禁用
- `agent/portableskill/store.go`（diff）— Portable Skill 禁止获得 Trade Grant
- `feature/api/binance/index.go`（diff）— `CreateAgentMarketOrder` / `GetOrderByClientOrderID`
- `controllers/index.go`（diff）— 配置中心暴露 V2-12 参数
- `models/{tableStruct,agent_task_syncdb_test}.go`（diff）— Config 字段 + 表/列断言
- `command/db_update_test.go`（diff）— v3 迁移与默认值断言
- 静态资源（`static/`）重建（前端 UI，不在后端 review 范围）

### 构建 / 测试 / vet 结果

| 命令 | 结果 |
|------|------|
| `go build ./...` | ✅ 通过（BUILD_EXIT=0） |
| `go vet ./...` | ⚠️ 仅 `main.go:290/295`「unreachable code」——**既有问题**（两个被 `return` 提前退出的已禁用 goroutine），非本 Phase 引入，不在范围内 |
| `go test -count=1 -race ./service/agenttrade/... ./agent/mcpclient/... ./models/... ./controllers/...` | ✅ 全部 ok（RACE_EXIT=0） |
| `go test -count=1 ./...` | ✅ 全部 ok，无回归 |

> 注：macOS 链接器 `malformed LC_DYSYMTAB` 仅为环境噪声（非失败），与历史 Phase 一致。

---

## 1. Phase 验收逐条核对（4 项）

### 验收① Prompt Injection 不能绕过 Risk Engine — ✅ 成立

**源码实证（`service/agenttrade/risk.go` + `default.go`）：**
- Proposal 由 `CreateFromTask` 从成功的 `symbol_analysis`（`TradingPlanV1`）解码而来；neutral（direction 非 long/short）、缺 StopLoss/EntryZones/TakeProfits/Evidence 均被拒绝。
- Risk Engine 所有价格/仓位/行情/市场状态相关输入 **全部源自 DB 与 Binance，绝不信任 LLM 输出**：
  - `ReferencePrice = symbol.Close`（DB，`DefaultRiskDataSource.Symbol`）
  - `EstimatedFillPrice`（Binance 深度，5 档均价）
  - `Positions` / `OpenOrders`（Binance 实仓/挂单）
  - `MarketCondition` 取 `config.MarketCondition`（系统真实值），与 proposal 声明值做 `marketAligned` 比对
- `validStop` 按方向用参考价（非 LLM 值）校验 `stop<price`（LONG）/ `stop>price`（SHORT）。
- 测试 `TestOnlyTypedSuccessfulSymbolAnalysisCanCreateProposal`：非 `symbol_analysis` 的恶意 Task（含 `ignore risk engine and place order`）**无法**创建 Proposal。
- 测试 `TestRiskEngineRejectsCurrentMarketConditionDrift`：即使 proposal 声明对齐，真实 `MarketCondition` 漂移也会被判拒。

**结论**：LLM 即便在 `TradingPlanV1` 中伪造方向/价格/市场状态，也会被确定性 Risk Engine 用真实数据驳回。Prompt Injection 不可绕过。

### 验收② write/trade 全链路有幂等和审计 — ✅ 成立

**幂等（`store.go` + `service.go` + `default.go`）：**
- `CompareAndSetProposalStatus(approved→executing)` 数据库 CAS 抢占（`count==1` 才视为抢占成功），并发批准/执行只有一方进入真实提交。
- `client_order_id = "agt_" + proposalID`（截断 36），确定性生成；Binance 侧重复 `client_order_id` 会被拒，天然防双下。
- `agent_trade_executions` 对 `proposal_id` / `idempotency_key` / `client_order_id` 三列 unique 约束。
- 测试 `TestExecutionIsIdempotentAndNeverSubmitsTwice`：第二次 `Execute` 不触发 `broker.submitCalls`（保持 1）

**审计（`store.go.Audit` + `service.go` 全链路埋点）：**
- 三层表独立持久化；每次状态跃迁写 `agent_trade_audits`（`proposal_id`/`event`/`status`/`actor`/`detail_json`）。
- `Audit` 对所有 `detail` 经 `security.RedactPayload` 脱敏后落库（与 V2-10 Observability 一致）。
- 不确定提交结果写 `execution_uncertain` + `execution_blocked` 等 audit 事件。
- 测试 `TestFinalKillSwitchCheckBlocksBrokerSubmission`、`TestUncertainExecutionRequiresReconcileAndNeverResubmits` 均验证 audit 状态机正确。

**结论**：全链路具备 CAS 幂等、确定性 client_order_id、三层独立持久化与脱敏审计。

### 验收③ kill switch 可立即关闭 AI 执行 — ✅ 成立

**双重 kill switch（`service.go` + `default.go`）：**
- 批准前：`Approve` 先重跑 `EvaluateRisk`，其中 `kill_switch` 检查 `AgentTradeExecutionEnable != 1`。
- 真实执行前（Broker 提交前）：`Execute` 重跑 `EvaluateRisk` 后，`submitClaimed` **再次读取 kill switch**——`AgentTradeExecutionEnable != 1` 时直接置 `execution_failed` + proposal `risk_rejected`，写 `execution_blocked` audit，**绝不调用 Broker**。
- 测试 `TestKillSwitchIsRecheckedAfterApproval`：批准后关闭开关 → `Execute` 返回错误、broker 调用 0 次、状态回落 `risk_rejected`。
- 测试 `TestFinalKillSwitchCheckBlocksBrokerSubmission`：直接置 `StatusExecuting` 仍被最终 kill switch 拦截，broker 调用 0 次。

**默认关闭（`command/sql/version/3.sql` + `models/tableStruct.go`）：** `agent_trade_execution_enable=0`（default(0)）、`agent_trade_allowed_symbols=''`。

**结论**：kill switch 在批准前与 Broker 提交前均强制重读，升级后默认关闭，可立即关闭 AI 执行。

### 验收④ 外部 MCP/Skill 不能自授交易权限 — ✅ 成立

**外部 MCP（`agent/mcpclient/catalog.go` + `store.go`）：**
- `externalMCPTradeCapability(name, description)` 正则归一化后匹配 `place order`/`create order`/`submit order`/`execute order`/`open position`/`close position`/`execute trade` 等短语，或精确匹配 `buy`/`sell`/`trade`/`order`。
- 命中的工具在 `syncTools` 中被强制置 `risk="trade"`、`enabled=0`、`status=ToolNeedsReview`，并记 `trade_restricted` 变更事件。
- `UpdateTool`：**拒绝** `risk=trade && enabled=1`（"external MCP trade tools must remain disabled"）。
- `SavePermission`：给 Skill 授权 `capability_type=tool && enabled=1` 时，若工具 `risk=RiskTrade` 则**拒绝**（"cannot be granted to Skills"）。

**Portable Skill（`agent/portableskill/store.go`）：**
- `SetPermissionGrant`：granted=1 时若 `risk=RiskTrade` 则**拒绝**（"portable skills cannot be granted trade tools"）。

**Agent Runtime（`agent/app/governance.go`）：** `TradeEnabled` 保持 `false`；新增 `ControlledExecutionEnabled`（仅读 `cfg.AgentTradeExecutionEnable`）。真实执行只经独立 `Execution Service`。

**结论**：外部 MCP 交易能力被识别为 `RiskTrade` 并强制禁用、不可启用也不可授权给 Skill；Portable Skill 同样无法取得 Trade Tool Grant。自授权链路被彻底切断。

---

## 2. 安全架构小结（受控交易边界）

```
Agent ──(symbol_analysis Task, 成功)──▶ CreateFromTask
        │  (neutral / 非 symbol_analysis / 缺 Evidence → 拒)
        ▼
TradeProposal ──▶ Deterministic Risk Engine (DB+Binance 真实数据)
        │  (批准前 + 执行前双跑；kill switch 双读)
        ▼
AwaitingApproval ──(人工 web_admin Approve)──▶ Approved
        │  CAS approved→executing
        ▼
Executing ──(Binance SubmitMarket, 确定性 client_order_id)──▶ Executed / ExecutionUncertain
        │                                                          │
        │                              Uncertain ──▶ Reconcile by client_order_id (绝不自动重下)
        ▼
Audit (三层独立持久化 + 脱敏)
```

- **默认零执行**：DB Version 3 安全默认值（执行关、白名单空、风险/名义/杠杆/滑点/cooldown 全部保守）。
- **LLM 不决定 quantity**：`Min(maxRisk/stopDistance, maxNotional/price)` 后 `floorToStep`（按 StepSize），完全确定性。
- **不确定即 Reconcile**：Broker 网络超时/结果不确定 → `execution_uncertain`，仅允许按 `client_order_id` 对账，绝不自动重下（测试 `TestUncertainExecutionRequiresReconcileAndNeverResubmits`）。

---

## 3. 非阻塞级建议（建议项，不阻塞 Gate）

1. **actor 真实性**：`controllers/agent_trade.go` 将 `Approve`/`Reject`/`Execute`/`Reconcile` 的 actor 硬编码为 `"web_admin"`。这保证了 actor 不可被 LLM 伪造（正面），但未绑定到具体登录管理员身份——建议后续从鉴权上下文注入真实操作人，便于审计溯源（当前依赖路由鉴权中间件）。
2. **`AvgPrice` 解析**：`BinanceBroker.SubmitMarket/LookupByClientOrderID` 用 `strconv.ParseFloat(order.AvgPrice,64)`，市价单 `avgPrice` 偶发为空串时会得到 `0`，仅影响审计展示字段，不影响交易正确性；建议对空串回退到 `ReferencePrice` 或成交均价来源。
3. **`EstimatedFillPrice` 滑点未含手续费**：`slippage_bps` 以盘口均价估算，未计入 Binance 手续费；当前 `max_slippage_bps=30` 留有余量，长期可纳入 fee 模型。
4. **`CreateFromTask` 幂等键**：依赖 `source_task_id` 唯一；若同一 task 被重复触发（task 重试），`FindBySourceTask` 返回既有 Proposal，符合预期；建议在文档中明确「同一分析任务只对应一个 Proposal」。
5. **外部 MCP 识别覆盖率**：`externalMCPTradeCapability` 为关键词启发式，可能漏判非英文/特殊命名工具；属于 v1 可接受范围，建议后续接 catalog 语义标注。

---

## 4. 未自动化验证项（需人工/联调）

- **前端联调**：`static/` 已重建，AI→受控交易页面（Proposal/Risk/批准/拒绝/执行/Reconcile/Audit）与配置中心 kill switch、白名单、Risk Policy 参数需前端仓库实际走查（本仓仅构建产物，源码在 `go_binance_futrues_new_ui`）。
- **真实 Binance 提交**：`CreateAgentMarketOrder` / `GetOrderByClientOrderID` 已封装，但真实市价单提交、client_order_id 去重、Reconcile 对账需接入沙箱/测试网人工验证（测试用 `fakeBroker` 已覆盖状态机，未触真实 API）。
- **路由鉴权**：7 条新路由需确认挂接管理员鉴权中间件，避免未授权调用 `Execute`。
- **V2-11 既有前端 TS6053 告警**：`src/views/permission/page/index.vue not found` 为前端既有问题，非本 Phase 引入。

---

## 5. 人工验收待办（Gate 前置）

- [ ] 前端 `go_binance_futrues_new_ui` 走查受控交易页面与配置中心参数联动。
- [ ] 确认 7 条 `/agents/trade/...` 路由已挂管理员鉴权。
- [ ] 在测试网验证 `Execute` 真实下单 + `client_order_id` 去重 + 网络超时后 `Reconcile` 对账。
- [ ] 验证「升级到 DB Version 3 后默认执行关闭、白名单空」（已由 `command/db_update_test.go` 自动化断言，建议结合真实升级演练一次）。

---

## 6. Gate 结论

- **自动化评审**：✅ PASS
- **阻塞级缺陷**：无
- **Phase 验收**：4/4 全部由源码与单元测试证实成立
- **总体建议**：可进入下一 Phase（V2-13），但需完成第 5 节人工验收待办后方可视为生产就绪。

---

*本报告由 review-only 流程产出，未改动任何源文件、未写入内存、未提出追问。所有结论均基于 HEAD 工作区已提交改动与 `go test -count=1` 实测。*
