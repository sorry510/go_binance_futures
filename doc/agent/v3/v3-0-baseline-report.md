# V3-0 Baseline Report

## 状态

V3-0 已完成。该阶段只增加测试、Benchmark 与基线文档，没有修改 Runtime API、真实交易行为、数据库结构或前端。

## 冻结契约

- `symbol_analysis` → `trading_plan_v1`
- `alert_analysis` → `alert_v1`
- `market_scan` → `opportunity_set_v1`
- `strategy_review` → `strategy_review_v1`
- 受控交易状态：`risk_rejected`、`awaiting_approval`、`approved`、`rejected`、`executing`、`executed`、`execution_failed`、`execution_uncertain`、`expired`
- Risk 状态：`pass` / `fail`

## Replay / Eval 基线

现有 V2 Core Eval 继续作为主 Gate，并覆盖 Native Skill 与 Workflow。Portable Skill、MCP failure recovery、permission escalation、router identity 等继续由现有 Eval/Tool Runtime/Model Gateway 测试冻结。

V3-0 新增显式关键输出契约 Replay，防止后续 Phase 在无意中改变交易分析和工作流结果类型。

## Benchmark

机器：Darwin arm64 / Apple M1 Pro。详细机器可比较数据保存于 `v3-0-benchmark.json`。

- 4 个核心 Replay：中位数约 `636093 ns/op`。
- 单次 Replay 聚合：成功率 `100%`，P95 `449 us`，Tool 调用 `6`，总轮次 `6`，平均轮次 `1.5`。
- Signal → Notification（AI 关闭、确定性 fallback）：约 `71019 ns/op`。
- Proposal → Execution（Fake Broker）：约 `1445684 ns/op`。

说明：Replay Fixture 使用 scripted LLM，当前 Fixture 未记录 provider token usage，因此该基线中的 Token 为 `0`。这表示 Fixture 成本身份，不代表生产模型实际 Token 消耗；后续对比必须使用相同采集口径。

## Gate 结果

执行并通过：

```bash
go test ./...
go test -race ./agent/... ./service/agenttrade ./service/alertpipeline
go test -run TestV3BaselineReplayMetrics -v ./agent/replay
go test -run '^$' -bench BenchmarkV3BaselineCoreReplay -benchmem ./agent/replay
go test -run '^$' -bench BenchmarkV3BaselineSignalToNotification -benchmem ./service/alertpipeline
go test -run '^$' -bench BenchmarkV3BaselineProposalToExecution -benchmem ./service/agenttrade
```

Race Test 在 macOS 链接阶段存在 `LC_DYSYMTAB` warning，但所有包均通过，未发现 data race。

## 数据库约束

V3 后续新增表/字段继续通过 ORM Model + `./go_binance_futures sync db` 管理。正常服务启动不得隐式迁移数据库。本阶段没有新增数据库字段，也没有直接修改实际数据库。

## V3-0 边界确认

- 未新增业务 Skill。
- 未修改真实交易行为。
- 未新增自动交易模式。
- 未重构 Runtime API。
- 未修改 `doc/TODO.md`。
- 未修改前端。
