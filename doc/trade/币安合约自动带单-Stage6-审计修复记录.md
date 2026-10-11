# 币安合约自动带单 — Stage 6 独立审计修复记录

日期：2026-10-11  
分支：`feat/lead-trading`  
来源：`doc/trade/CODE_REVIEW_Stage6.md`（保留原始 review-only 审计报告，不改写审计结论）

## 总结

审计发现 **0 项 P0/P1**；本轮针对 F1～F3 做预防性安全收口，F4 已由之前的前端同步操作解决，F5 延期至真实环境 Gate。

**本轮始终禁止真实 Lead Submit/Cancel**，未启动 Stage 5 私有 WS Supervisor，也未改动 `app.conf`、业务 DB、策略模板、正在监听 3333 的服务，更未自动 Git commit/push。

## 逐项审计处理

| 问题 | 修复与证据 | 边界 |
|---|---|---|
| **F1 P3：生产激活路径守卫不完整** | 新增 `feature/lead_management_stage6_review_test.go`：扫描仓库全部生产 Go 文件（剔除测试与 Lead 实现包）。**只有** `controllers/lead_account.go` 能导入 `service/leadaccount`，且该管理入口调用的包导出符号需在精确白名单内；控制器禁止 `Run`、`NewLeadSupervisor`、`NewLeadUserStream`、`ExecuteOpen`、`Cancel`、`Submit`、`Reconcile` 等执行/激活方法。单测构造别名导入、其他包偷偷导入、点导入、Supervisor 构造和 `Run` 的反例并验证被拒绝。保留 Stage 4/5 旧 AST 安全测试。 | 本守卫是代码层安全网，不是 Stage 7 正式授权；如未来生产只读 Supervisor 接线，需先审计后修改白名单与测试 |
| **F2 P3：Pause 被审计/配置不可用阻断** | `POST /futures/lead/pause` 经 JWT 后 **先**执行 `leadaccount.LatchLeadEmergencyPause()`（进程内单向原子闸），然后分别尽力写入私有 Lead 配置和固定码审计。执行器 `checkOpenLocked` 新增闸门判断，保证 Lead 新开仓不绕过该闸；不会影响 Main，也不放开被禁止的 Broker。配置可持久化则返回 200；持久化失败返回 **202**，并明确返回 `pause_persisted=false`、`pause_in_memory=true` 及原因码。审计失败只固定脱敏日志+响应 `audit_recorded=false`，不阻止安全暂停。 | **内存暂停不能跨进程/重启保证**，HTTP 202 不表示持久暂停；生产仍未启用真实 Lead Runner。Stage 7 必须同时检查内存闸与持久停止状态，且经强鉴权人工 Gate 才能恢复 |
| **F3 P3：看板误报“告警丢弃=0”** | `LeadUsageMetrics` 增加 `alert_queue_present`；仅有真正 `LeadAlertQueue` 实例时才为 true。`/futures/lead/metrics` 同时显式返回顶层 `alert_queue_present=false`、`alert_queue_source=supervisor_not_started`。前端系统看板和 Lead 账户页面仅在 `alert_queue_present=true` 时显示真实丢弃数量，否则显示 `-`。 | 当前生产 Supervisor 未运行，仍无实时队列、WS 和账户 PnL；不把未观测数据视为 0 |
| **F4 P3：前端构建产物未同步** | 用户前一轮指出后已完成 `pnpm build` 与 `dist → backend/static` 同步；逐文件校验一致（100 个文件），新 Lead JS 资源存在，`platform-config.json` 保留，旧哈希资源已清理。 | 本轮前端修改后需再次 build/sync，且不重启既有 3333 服务 |
| **F5 历史 P3** | 保留并整理 Stage 4/5 遗留风险，明确“运维请求审计（当前文件）”与“订单恢复证据审计流水（尚未完成）”不是一回事。 | 后续 Stage 7 前需验证真实 Portfolio、Algo/动态价格限制、主账户在线回归、MySQL 8 持久恢复、受控最小额授权 |

## 新增回归测试

- `feature/lead_management_stage6_review_test.go`：白名单扫描及反例自测。
- `controllers/lead_account_stage6_review_test.go`：无审计/无配置暂停返回 202 + 内存阻断、有配置无审计仍可持久暂停、完整配置暂停被审计、指标缺少队列不会声称真实观测。
- 原有 Stage 6 JWT/423/跨账号设置限制和 Stage 4/5 主要路径仍需完整 Race 回归。

## 审计修复后最终验证（2026-10-11）

| 检查 | 实际结果 |
|---|---|
| `go test -race -count=1`（`leadaccount`、`feature/api/binance`、`futuresownership`、`feature`、`strategy/line`、`scanner`、`backtest`、`command`、`controllers`、`middlewares`、`routers`） | **10 个有测试包全部 PASS；routers 无测试文件，RACE_EXIT=0** |
| `go vet` 同一模块集 | **PASS，VET_EXIT=0** |
| 后端生产 `go build` | **PASS，BUILD_EXIT=0**，临时二进制已删除 |
| 前端 `pnpm typecheck` | **PASS（tsc/vue-tsc）** |
| 定向 ESLint、Prettier、`pnpm build` | **PASS；新 Lead/看板 JS 已编译** |
| 前后端 `git diff --check` | **PASS** |
| 最终 `dist → backend/static` | **已同步；100/100 文件一致，`index.html` 与 `platform-config.json` 校验一致；旧哈希资源已删除。** |
| 本地 3333 端口 | 已有 `go_binance_futures` 进程监听；本次没有启停/重启进程，不将该进程视为已更新后端代码 |
| 生产 Binance、MySQL、Stage 7 | **未触发真实 Lead Submit/Cancel、没有生产 DB 写入或 Schema 更新，也没有提供真实环境 Gate 结论** |

沿用已有约定排除与本阶段无关的 `TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals`，macOS 编译会出现非致命 `LC_DYSYMTAB` linker 警告，但全部验证退出码为 0。

静态同步前已将旧资源保存在临时目录 `/tmp/stage6-review-static-backup.P73hVE`，仅作本机回滚参考；Git 工作区中的静态资源变更均未自动提交。

**Gate 6 结论：审计 F1～F3 的离线修复与回归 PASS；F4 前端静态同步已完成。Gate 5/6/7 真实环境验收未通过，真实 Lead Broker 仍未授权。**

## 后续真实环境的责任边界

当前真实 Lead 账户 **disabled**；Stage 5 真正 WS/ListenKey/重启对账未完成线上 Gate。强鉴权生产管理页面与 Stage 7 真实下单资格仍需独立审计和人工批准。任何 `open_ready`、`pause_in_memory` 或离线 Mock PASS 均不构成实盘许可。
