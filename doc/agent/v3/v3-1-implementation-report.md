# V3-1 Multi-Agent Team Implementation Report

## 状态

V3-1 已完成。实现了一个后端固定定义的 `symbol_analysis_team`，采用：

`Team Run → Technical/Flow Child Tasks → Typed Results → Supervisor Result`

首版默认运行 Technical Analyst、Flow Analyst 和 Supervisor。News Analyst 依赖的新闻/公告/Alpha 统一数据留到 V3-2；Strategy Reviewer 暂不加入默认 Team Run，避免在 V3-1 扩大数据采集与业务范围。

## 核心实现

- 新增 `agent/team` Team Coordinator，不创建第二套 Agent Runtime。
- 新增 `agent/skills/symbolteam` Typed Skill：Technical、Flow、Supervisor。
- Technical / Flow 并发执行，均继续通过现有 Manager、Task、Model Gateway、Permission、Runtime 和 Observability。
- Supervisor 只接收 Typed Child Result，`Tools()` 为空，不获得额外 Tool 权限。
- `get_symbol_analysis_context` 由 Team 共享 Tool Runtime 只采集一次，再把结构化上下文传给 Technical / Flow。
- Child 失败或数据不完整时保留其它成功结果，并输出 `partial` / `data_missing`。
- Team 总并发、Token 和 Tool Budget 均有明确上限。
## Task / Observability

`agent_tasks` 新增以下关联字段，用于父子 Task 和 Team lineage：

- `parent_task_id`
- `team_run_id`
- `team_name`
- `team_role`

Task Center 可查看 Team Run、子 Task、角色、状态、耗时、Token 和父子关联。Observability Trace 会通过 Task lineage 补充 Team Run / Role 信息。

最终 Team Typed Result 不包含调度时序相关的 Child Task ID，保证 Replay 结果确定性；Child Task ID 仍完整保存在 Task lineage 中，可用于审计和追踪。

## 数据库

- 数据库版本：`3 → 4`。
- Schema 继续由 ORM Model + `orm.RunSyncdb` 管理。
- Version 4 仅包含 ORM Schema 变更，不需要单独的 SQL 迁移文件；`sync db` 会在版本 SQL 不存在时直接跳过数据迁移步骤。
- 实际 MySQL 仅通过 `./go_binance_futures sync db` 升级。
- 第二次执行 `sync db` 返回 Version 4 已是最新，幂等检查通过。
## 验收 Gate

已通过：

```bash
pnpm typecheck
pnpm build
go test ./...
go test -race ./agent/... ./service/agenttrade ./service/alertpipeline
go test -run 'TestSymbolAnalysisTeamFixedFixtureStabilityMatchesSingleAgentBaseline|TestTeamFixtureReplayIsDeterministic' -v ./agent/team
./go_binance_futures sync db
```

固定样本稳定性：Team 连续 20 次成功 `20/20`，原 `symbol_analysis` Replay 连续 20 次成功 `20/20`；Team Replay Typed Result 一致。

Race Test 仅有 macOS linker 的既有 `LC_DYSYMTAB` warning，没有 data race。

前端最终 `dist` 已按约定同步到后端 `static`，rsync dry-run 无差异。

## 边界确认

- 未实现群聊式 Agent-to-Agent conversation。
- 未允许 Agent 自主创建 Agent。
- 未增加新的真实交易权限或绕过 deterministic Risk Engine。
- Team 结果不接入 V2-12 Proposal 创建入口；即使 Team 状态为 `succeeded`，`agenttrade` 仍只接受 `symbol_analysis` Task。
- 未修改 `doc/TODO.md`。
- 未进入 V3-2。

- Skill 管理增加 `team` 类型并展示 `symbol_analysis_team`；对话 `/` 菜单可直接选择“多智能体单币分析”。
- Chat 启动 Team 时父 Task 绑定 Conversation，完成后以 Markdown 写回 Supervisor 汇总。
