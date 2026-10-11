# Stage 6-1 — Lead 后端路由与 JWT 鉴权

> 2026-10-11；离线开发/Mock 验证，**真实 Binance 与 Stage 7 实盘未授权**。

## 实际交付

- 新增 `controllers/lead_account.go` 和 `routers/router.go` 的 12 个 `/futures/lead/*` 路由；保持 Beego JWT 全局过滤器，并在 Controller 内再次校验 Authorization Bearer 与有效 JWT；写请求要求 JSON，严格拒绝未知字段。
- 状态/监控/持仓/订单/事件/白名单读取只返回 `account_id=lead`、固定 Gate 阻断原因。真实 Supervisor 在生产主程序中**没有自动 Run**。
- 真实 `/start`、`/reconcile` 均返回 HTTP 423（Locked）；先记录固定原因码审计请求，但绝不执行 Binance Submit/Cancel/WS。
- 旧 Stage 4 生产依赖断言随 Stage 6 只读路由接线进行了显式审计升级：允许生产导入 `service/leadaccount` 的管理用途，同时通过 Go AST 检查 Controller 不得调用 Supervisor Run、ExecuteOpen、Submit、Cancel 等真实写路径。

## 本阶段安全纪律

未修改 `app.conf`；没有执行 `sync db` 或生产数据库写入，不触发真实 Binance 下单/撤单/WS 启动；无自动 Git commit/push。历史用户未提交修改不回退。


**Stage 6 独立审计 F1/F2 补充**：生产入口新增精确白名单 AST 安全扫描，未经审计的 Supervisor 构造/Run/交易写入入口不能从其他生产模块直接引用；暂停接口优先设置进程内安全停止，缺少审计不拒绝停止，但响应标出持久性，未来 Stage 7 必须覆盖重启和多实例语义。参见 `币安合约自动带单-Stage6-审计修复记录.md`。
