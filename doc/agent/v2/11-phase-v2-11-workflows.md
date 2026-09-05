# Phase V2-11：业务 Workflow 与新 Skill

## 前提

Runtime/Context/Tool/Eval/MCP/Standard Skill 基础稳定后，再扩展多步业务能力。

## market_scan

确定性 Scanner 筛候选，Agent 分析少量候选并排序，输出版本化 Opportunity Set；不让 LLM 扫全市场 Tick。

## strategy_review

读取 Strategy Template、测试结果、手续费净收益和 MarketCondition，输出策略适用环境、失败模式和修改 Proposal；不直接修改正式模板。

## strategy_experiment

Agent 提议候选 -> Validator -> 确定性测试引擎 -> Agent 归纳。实验结果版本化，不自动覆盖正式策略。

## alert_triage

把 FastMove、Liquidation、OI、Funding 等同时间窗口 Signal 聚合成 Incident，再由 Agent 判断是否属于同一市场事件，降低重复报警。

## daily_market_brief

Scheduler 聚合 MarketCondition、Scanner、重要 Signal 和变化，输出固定 Schema 市场摘要。

## MCP/Portable Skill 的角色

新业务 Workflow 可以调用已授权 MCP Tool，也可以由 Portable Skill 提供流程知识；关键交易 Validator/风控仍保留 Native Go 实现。

## 验收

- [ ] 新 Skill 不复制 Runtime/Task/Tool 基础设施。
- [ ] 大规模计算由确定性 Service 完成。
- [ ] 每个输出有版本 Schema 和 Eval。
- [ ] 外部 MCP/Skill 故障有 fallback 或明确 failure。
