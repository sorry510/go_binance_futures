# Phase V3-8：系统看板、个人运维与 V3 收尾

> 状态：✅ 已完成
>
> 定位：P2。V3 最后一个阶段不再增加新的 Agent/交易产品层，只补长期个人运行真正需要的健康诊断、日志清理和最终收尾。

## 1. 最终范围

V3-8 只实现四部分：

1. 复用现有 Observability，升级为顶级菜单 **系统看板**。
2. 提供只读命令 `./go_binance_futures doctor`。
3. 提供白名单日志清理命令 `./go_binance_futures cleanup logs --before-days N`。
4. 完成 V3 最终代码、文档、测试和前端静态资源收尾。

明确取消原计划中的数据规模/大表统计、备份中心以及备份/恢复文档。

## 2. 系统看板

原 `AI → 可观测性` 页面移动为最顶部的顶级菜单 `系统看板 / System Dashboard`，继续复用原有 Trace、Skill/Model/Tool/MCP 指标和变更历史。

页面新增只读 System Health 摘要：

- Database：连接状态、当前 Schema Version 与二进制要求版本。
- Binance REST：使用当前 testnet/mainnet 和代理配置检查 Futures `/fapi/v1/time`。
- Futures WS：根据 `symbols.updateTime` 判断行情更新时间是否新鲜。
- Announcement WS / Market Intelligence：复用 `agent_market_source_status` 的成功/失败状态。
- MCP：展示已启用 Server 的健康状态、最近成功和最近错误。
- LLM Gateway：检查至少存在一个启用/Router Candidate，并结合最近 24h `llm_call` 观察记录提示异常。
- Scheduler：Web 页面使用当前进程内 Scheduler 状态；独立 CLI 无法读取服务器进程内状态，因此改用配置 + 最近 Agent Task 时间做 freshness 诊断，不伪造实时状态。
- Agent Runtime：统计最近 24h Task、Failed 和达到最大轮次的失败。
- Trade Safety：检查 `execution_uncertain`、`protection_failed` 以及 Managed Position/Order 的 `reconcile_required`。

Health API：

```text
GET /system/health
```

该接口和页面只读，不会下单、Reconcile、修改配置或迁移数据库。

## 3. doctor 只读诊断

服务器/Web 页面异常时可以直接执行：

```bash
./go_binance_futures doctor
```

输出 Database、Binance REST、Futures WS、Announcement、Market Intelligence、MCP、LLM、Scheduler、Agent Runtime、Trade Safety 和 Overall 状态。

`doctor` 在 CLI 初始化阶段只注册数据库连接，不启动 Web、WebSocket、Scheduler、Opportunity Pipeline、Ownership Reconcile 或交易循环，也不会隐式执行 `sync db`。

退出码约定：`Overall=error` 时退出码为 `1`，方便 shell/监控脚本识别硬故障；`healthy`、`warning`、`disabled` 状态下退出码为 `0`，具体健康状态以输出内容为准。
## 4. cleanup logs

唯一新增的日志清理命令：

```bash
./go_binance_futures cleanup logs --before-days 90
```

`--before-days N` 表示删除当前时间往前 N 天以前的纯日志/运行记录。命令使用固定白名单，并在一个数据库事务中执行：

| 表 | 时间字段 | 用途 |
| --- | --- | --- |
| `agent_observations` | `created_at` | Agent/LLM/Tool 可观测性 Trace |
| `agent_change_events` | `created_at` | Skill/MCP 等变更历史 |
| `agent_task_events` | `event_time` | Agent Task 过程事件 |
| `agent_alert_pipeline_traces` | `created_at` | 报警链路历史 |
| `notifications` | `create_time` | Web 通知历史 |

命令不会根据表名模糊匹配，也不会自动扩大删除范围。`N <= 0` 会直接拒绝执行。

明确不会删除：Historical Market/Kline/Funding、Backtest Run/Trade/Event/Equity、Strategy Template、Paper Trade、真实 Order/Position/Ownership、Agent Trade Proposal/Execution/Audit、Opportunity、Conversation、Memory、Skill/MCP 配置、Workflow 定义以及 Scheduler 配置。

其中 `agent_trade_audits` 属于真实交易生命周期审计，即使名称包含 Audit，也永远不属于该清理命令。
## 5. V3 最终收尾

- 数据库版本使用 `appversion.DatabaseSchemaVersion` 单一常量，主程序、`sync db`、`doctor` 共用同一版本来源。
- V3-8 不新增数据库表或字段；当前二进制要求 Schema Version 为 **v14**。
- Backtest Run 的异步删除逻辑继续由既有测试覆盖 Trade/Event/Equity 等 Run 私有数据清理，不新增第二套删除机制。
- 清理确认无用的旧菜单/重复文案时保持保守，只删除能够证明已经废弃的内容。
- `doc/TODO.md` 不由 V3 自动修改。
- 前端继续执行 `pnpm build`，并将 `dist` 内容同步到后端 `static`。

## 6. 验收 Gate

- 系统看板位于顶级菜单最上方，原 AI 菜单不再出现“可观测性”。
- 用户可以快速判断 Database、Binance、Market Intelligence、MCP、LLM、Scheduler、Agent Runtime 和 Trade Safety 状态。
- `doctor` 可独立运行且完全只读。
- `cleanup logs --before-days N` 只删除固定白名单；Fixture 必须证明真实交易 Audit 不会被删除。
- `go test -count=1 ./...`、`go vet ./...`、关键 `-race`、前端 `vue-tsc --noEmit` 和 `pnpm build` 全部通过。
- 前端 `dist` 与后端 `static` 一致。

## 7. 本阶段明确不做

- 不做 Kline/Backtest 表行数、容量、磁盘增长等数据统计页面。
- 不做备份中心、自动备份、数据库备份/恢复流程或相关 V3 文档。
- 不做 Agent Studio / Developer Portal / Skill Marketplace。
- 不做多租户运维平台或 Prometheus/Grafana 强制集成。
- 不做自动数据库修复、自动 Trade Reconcile 或自动恢复下单。

## 8. 最终实现与 Gate

最终实现：顶级“系统看板”复用原 Observability；新增 `/system/health`、只读 `doctor` 和固定白名单 `cleanup logs`。V3-8 没有新增数据库 Schema，当前要求版本保持 v14。

实际验证：

- `./go_binance_futures doctor`：已在当前 MySQL 配置实跑；DB/Binance REST/Futures WS/Announcement/Market Intelligence/LLM/Trade Safety 正常返回真实状态。
- `./go_binance_futures cleanup logs --before-days 0`：实跑确认拒绝执行且退出非 0，没有删除数据。
- Log Cleanup SQLite Fixture：验证五类白名单旧记录删除，且 `agent_trade_audits` 保留。
- System Health SQLite Fixture：验证 Database、Binance REST、Futures WS、MCP、LLM、Scheduler 和 Trade Safety。
- `go test -count=1 ./...`：通过。
- `go vet ./...`：通过。
- `go test -race ./service/systemhealth ./service/logcleanup ./command ./controllers ./routers`：通过；仅有 macOS linker `LC_DYSYMTAB` 非阻塞 warning。
- 前端 `pnpm exec vue-tsc --noEmit`：通过。
- 前端 `pnpm build`：通过。
- `dist` 与后端 `static`：92 个文件，逐文件一致。
