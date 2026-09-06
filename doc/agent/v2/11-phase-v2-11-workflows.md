# Phase V2-11：业务 Workflow 与新 Skill

## 状态

P1 ✅

## 前提

Runtime/Context/Tool/Eval/MCP/Standard Skill 基础稳定后，再扩展多步业务能力。

## market_scan

确定性 Scanner 先扫描全市场并筛出候选，只把前 10 个以内的结构化候选交给 Agent 排序和解释；LLM 不扫描全市场 Tick。输出固定 `opportunity_set_v1`。

## strategy_review

读取 Strategy Template、匹配完整 strategy/technology 快照的测试结果、手续费后净收益和 MarketCondition，输出适用环境、失败模式和修改 Proposal。输出固定 `strategy_review_v1`，不直接修改正式模板。

## strategy_experiment

流程固定为 Agent 提议候选 -> Native Go Validator/确定性测试 -> Agent 归纳。候选技术指标与策略规则使用固定 synthetic scenarios 编译/执行；最终结果必须原样保留候选 JSON 和 deterministic test report。输出 `strategy_experiment_proposal_v1` / `strategy_experiment_result_v1`，不会自动覆盖正式策略。

## alert_triage

AI 报警开启时，同一 symbol 的 Signal 进入短时间 Incident 聚合窗口。单条继续使用 `alert_analysis`；窗口内多条 FastMove、Liquidation、OI、Funding 等 Signal 使用 `alert_triage` 归并判断并只产生 Incident 级通知。AI 关闭、预算不足或 triage 失败时保持确定性 fallback，不吞报警。输出固定 `incident_set_v1`。

## daily_market_brief

Scheduler 聚合 MarketCondition、Scanner 与最近重要 Signal，输出固定 `daily_market_brief_v1`。独立调度开关默认关闭，默认周期 1440 分钟，避免升级后自动增加 LLM 消耗。

## Workflow Runtime

五类 Workflow 统一通过 `service/workflow` 编排，父 Run 持久化到 `agent_workflow_runs`，保存阶段、状态、结果与所有 Child Task ID。Agent 子步骤继续复用既有 Manager、Model Router、Runtime、Task、Context、Memory、Tool Runtime、MCP、Permission 与 Observability，不建立第二套 Agent 执行体系。

## MCP/Portable Skill 的角色

新业务 Workflow 可以调用已授权 MCP Tool，也可以由 Portable Skill 提供流程知识；关键确定性计算、Validator 和风控仍保留 Native Go 实现。外部能力失败时必须 fallback 或形成可查询的明确 failure。

## API / UI

- `POST /agents/workflows`
- `GET /agents/workflows`
- `GET /agents/workflows/:id`
- 前端：AI → 业务 Workflow
- Dashboard：Daily Market Brief Scheduler 开关与周期

## 验收

- [x] 新 Skill 不复制 Runtime/Task/Tool 基础设施。
- [x] 大规模计算由确定性 Service 完成。
- [x] 每个输出有版本 Schema 和 Eval。
- [x] 外部 MCP/Skill 故障有 fallback 或明确 failure。
- [x] `go test ./...` 通过。
- [x] V2-11 相关 race test 通过。
- [x] Replay/Eval Core Gate 通过。
- [x] 前端 typecheck/build 通过并已同步到 backend/static。
