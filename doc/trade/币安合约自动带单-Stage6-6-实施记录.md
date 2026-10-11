# Stage 6-6 — 观测、跨模块测试与 Gate 6 离线验收

> 2026-10-11；离线开发/Mock 验证，**真实 Binance 与 Stage 7 实盘未授权**。

## 实际交付

- Lead-only `/metrics` 复用 Stage 5 `LeadUsageSnapshot`：可独立查看已有的 5 分钟 REST 源使用情况；由于生产 Supervisor 未启动，WS 状态明确 `runtime_connected=false`。页面不宣称实时 WS/手续费/PnL。
- 后端新增 `controllers/lead_account_stage6_test.go`、`service/leadaccount/management_stage6_test.go`：JWT 匿名拒绝、Start/Reconcile 423、配置不能启用 real Lead、审计白名单、配置文件 0600、Lead-only Ownership 与只读状态无 Secret。
- Stage 4/5 生产安全回归更新：生产现在允许 **只读管理 API** 导入 Lead 模块，因此原零依赖断言不再适用；新的源代码 AST 守卫禁止 Lead Controller 直接调用 `Run/Submit/Cancel/ExecuteOpen`，并固定新路由数，防止后续未审计扩权。
- Gate 6 仅按**离线 UI/API/Mock** 条件验收；凭证实际只读联调、完整恢复审计流水、实时 managed/actual/PNL 合并、真实私有 WS、高负载、MySQL 8 迁移及安全上线均未做，不能当作 Stage 7 真实交易授权。

## 本阶段安全纪律

未修改 `app.conf`；没有执行 `sync db` 或生产数据库写入，不触发真实 Binance 下单/撤单/WS 启动；无自动 Git commit/push。历史用户未提交修改不回退。


## 系统看板接入（离线 UI）

`src/views/ai/observability.vue` 系统看板新增“合约带单”只读摘要，展示隔离账户就绪状态、私有 WS 状态、五分钟 REST 次数/权重及告警丢弃数，提供进入 `/futures/lead` 的链接。请求后台未部署时显示 `offline` / `-`，**不将失败请求当作健康状态**。既有 Binance API 总览的 5 秒静默刷新机制没有更改，Lead 摘要仅随页面主动刷新读取，不额外高频调用 Binance REST。


## 最终测试结果（2026-10-11）

| 检查 | 实际结果 |
|---|---|
| `go test -count=1 -race`：Lead、Binance Client、Ownership、Feature、Line Strategy、Scanner、Backtest、Command、Controllers、Middlewares（另检查 Routers 包） | **10 个有测试的 Go 包全部通过，Routers 无测试文件，RACE_EXIT=0** |
| `go vet`（上述后端包） | **通过，VET_EXIT=0** |
| `go build` 后端根包 | **通过，BUILD_EXIT=0**；临时编译文件已删除 |
| `pnpm typecheck`（tsc + vue-tsc） | **通过** |
| `pnpm exec eslint`（新 Lead 页面/API、路由和系统看板） | **通过** |
| `pnpm build` | **通过，构建独立 leadTrading 组件** |
| 前后端 `git diff --check` | **通过** |
| 3333 端口 | 已有其他 `go_binance_futures` 服务监听，本轮**未启动/停止/重启**，没有修改其正在服务的 `static/` 文件 |

本次保留现有未提交文件，不自动 commit/push；仅跳过与带单无关的 `TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals` 旧静态模板测试。Mac `LC_DYSYMTAB` 非致命链接器警告仍存在，但所有测试退出码均为 0。

**Gate 6 结论：UI/API 离线验收通过；生产部署与真实 Lead Gate 5/6/7 仍未通过。** 后续上线/真数据展示需先完成授权与真实环境核验，不能仅依据本次页面构建自动开启实盘。


**Stage 6 独立审计 F3/F4**：当 Supervisor 告警队列不存在时，看板/Lead 页面显示 `-`，避免无观测被误报为“告警丢失 0”；前端构建文件已在后续用户明确要求下同步到后端 `static/`。本轮审计前端再修改后仍需重新构建与同步，且不停止已有后台服务。


**2026-10-11 Stage 6 审计修复后再次同步**：根据 `CODE_REVIEW_Stage6.md` F3，已将“未连接告警队列”从误导性的丢弃 0 修正为 `-`，前端 Typecheck/ESLint/Build 通过，产物已再次从 `dist` 同步至后端 `static/`。两端各 100 个文件且逐文件完全一致，`platform-config.json` 保留，未重启已运行的 3333 服务。具体见 [Stage 6 审计修复记录](币安合约自动带单-Stage6-审计修复记录.md)。
