# Phase V3-8：Outcome Attribution 与 Continuous Eval

## 目标

把“任务成功”升级为“交易判断是否真的有价值”，让每个结果都能追溯到 Signal、Evidence、Agent、模型、策略、Risk 和 Execution。

## Attribution

每个 Opportunity / Proposal / Execution 最终关联：

- Market Event / Signal
- Evidence
- Team Run / Child Agent
- Skill / Prompt / Model Version
- Strategy Version
- MarketCondition
- Risk Decision
- Execution Result
- PnL / Fees / Funding / Slippage
- 最大有利/不利变动（MFE / MAE）

即使 Shadow 模式没有真实订单，也要生成可比较的虚拟 Outcome。
## Eval 维度

- 单币分析方向准确率和计划失效率。
- 各 Agent 角色的 Evidence 命中率与贡献。
- 不同 Model / Prompt / Skill Version 的结果差异。
- Strategy Version 在不同 MarketCondition 下的表现。
- Signal → Opportunity → Proposal → Execution 的漏斗质量。
- Shadow / Assisted / Auto 的结果对比。

## Drift

当某个 Strategy、Skill、Model 或 Signal 的滚动表现明显恶化时，产生 Drift Warning；首版只报警和展示，不自动修改模型或策略。

## UI

- Outcome Dashboard 展示 PnL、胜率、MFE/MAE、费用、按 MarketCondition/Strategy/Model 分组结果。
- 能从一笔交易回看完整 Evidence、Agent 输出、Risk 和 Execution。

## 验收 Gate

- 任意真实/Shadow 交易都能追溯到产生它的版本和证据。
- 同一指标计算有 deterministic 测试。
- Drift 只产生提示，不会自动 Promote/Retire Strategy。
- Eval 失败不会影响行情和交易主循环。

## 本阶段不做

- 不做在线强化学习自动改策略。
- 不让 LLM 根据短期盈亏自动修改生产配置。
