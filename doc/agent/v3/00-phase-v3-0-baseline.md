# Phase V3-0：Baseline 与 Benchmark

## 目标

在进入 V3 前冻结 V2 的稳定行为，保证后续 Multi-Agent、Backtest、Risk 和 Execution 改造不会破坏现有能力。

## 范围

- 固化核心 Native Skill、Portable Skill、MCP、Workflow、Chat、Memory 的 Replay/Eval Fixture。
- 固化 `symbol_analysis`、`alert_analysis`、`market_scan`、`strategy_review`、受控交易的关键输出契约。
- 保存当前模型路由、Tool 权限、Risk、Execution 状态机的回归样本。
- 建立当前性能基线：成功率、P95 延迟、Token、Tool 调用数、任务轮次、Signal→Notification、Proposal→Execution。
- 明确 V3 新表和字段继续使用 `sync db` 管理，正常启动不迁移数据库。
## 验收 Gate

- `go test ./...` 通过。
- Agent 核心模块 race test 通过。
- V2 Replay/Eval Core Gate 全部通过。
- 保存一份可比较的 Benchmark 结果。
- 任何 V3 改造都不能让 V2 的受控交易绕过 deterministic Risk Engine。

## 本阶段不做

- 不新增新业务 Skill。
- 不修改真实交易行为。
- 不新增自动交易模式。
- 不重构现有 Runtime API，只补测试和基线。

## 完成状态

已完成。验收与 Benchmark 结果见 [v3-0-baseline-report.md](./v3-0-baseline-report.md)，机器可比较快照见 [v3-0-benchmark.json](./v3-0-benchmark.json)。
