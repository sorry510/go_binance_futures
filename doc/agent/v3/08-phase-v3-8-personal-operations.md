# Phase V3-8：个人运维与 V3 收尾

> 定位：P2。让系统适合长期个人运行，而不是再增加新的 Agent 产品层。

## 1. 目标

V3 最后一个阶段只解决长期自用会真正遇到的问题：

- 我现在的数据源是否正常？
- Agent / MCP / LLM 最近有没有持续失败？
- Scheduler 有没有停止？
- 历史行情和 Backtest 大表增长到什么程度？
- 交易链路有没有需要 Reconcile 的状态？
- 出问题时如何快速诊断、备份和恢复？

不再建设 Agent Studio。

## 2. 健康检查

优先复用现有 Observability、Market Source Status、Task、MCP 和配置数据，形成一个个人健康摘要：

- Binance REST / Futures WS 状态。
- Announcement WS 最后成功时间。
- Market Intelligence 各 Source 最后成功/失败时间。
- MCP Server 连接状态。
- LLM Model Gateway 最近错误。
- Scheduler 关键任务最后执行时间。
- Agent Task 最近失败率 / 超轮次情况。
- Managed Trade 是否存在 `execution_uncertain` / `reconcile_required`。
- DB Version。

可以是一个简单页面或诊断命令，不需要 NOC 大屏。

## 3. 数据增长控制

重点检查长期增长明显的表：

- `market_klines_*`
- `market_funding_rates`
- `agent_backtest_equity_points`
- `agent_backtest_events`
- Agent Task / Observation / Event 日志
- 通知 / 报警历史

原则：

- Historical Market Repository 是可复用资产，不默认定期删除 Kline。
- 用户主动删除 Backtest Run 时，应能明确级联删除该 Run 的 Trade/Event/Equity 数据。
- 对纯日志类数据提供简单保留天数或手动清理命令即可。
- 不做复杂冷热分层、对象存储和数据湖。

## 4. 备份与恢复

提供清楚的个人运维说明：

- 数据库备份/恢复命令。
- `conf` 中非 Secret 配置如何备份。
- Strategy Template 如何导出/导入。
- Skill / MCP 配置哪些来自 DB，哪些来自本地文件。
- Secret / OAuth Token 不放入普通导出包。

优先写清流程，不为了备份功能再开发一套平台。

## 5. 一键诊断

如果实现成本低，可增加类似：

```bash
./go_binance_futures doctor
```

只读检查：

- DB 可连接、Schema Version 正确。
- Binance 配置和连接可用。
- Historical Market 最近更新时间。
- LLM 至少一个候选可路由。
- MCP 状态摘要。
- Scheduler 状态。
- 未完成 Trade Execution。

诊断命令不自动修改数据库或交易状态。

## 6. V3 收尾清理

- 删除确认不再使用的旧代码、旧菜单和重复配置。
- README 与 `doc/agent/v3` 按最终实际能力同步。
- 全量检查 SQLite / MySQL / PostgreSQL 兼容边界（只检查项目实际支持部分）。
- 整理 build / `sync db` / frontend deploy 标准步骤。
- 最后一次全量 Test / Race / Build / Diff / Static Deploy Gate。

## 7. 验收 Gate

- 用户可以在几分钟内判断 Binance、Market Intelligence、MCP、LLM、Scheduler 和 Trade 是否健康。
- 删除一个 Backtest Run 不留下孤儿 Trade/Event/Equity。
- 关键大表能够查看行数/时间范围，日志类数据有明确清理办法。
- 有可执行的数据库备份/恢复说明。
- 诊断流程不会下单、修改仓位或隐式迁移 Schema。
- `go test -count=1 ./...`、关键 `-race`、前后端 build 全部通过。
- 前端 `dist` 与后端 `static` 一致。
- `doc/TODO.md` 保持不被 V3 自动修改。

## 8. 本阶段明确不做

- 不做 Agent Studio / Developer Portal。
- 不做 Skill Marketplace。
- 不做多租户运维平台。
- 不做 Prometheus/Grafana 等重型监控栈的强制集成。
- 不做自动数据库修复或自动交易恢复操作。
