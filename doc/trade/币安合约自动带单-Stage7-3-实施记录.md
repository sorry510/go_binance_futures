# 币安合约自动带单 Stage 7-3 — 持久化安全门禁、恢复审计与价格过滤器

> 日期：2026-10-11，分支 `feat/lead-trading`。**仅离线代码和 Mock 验收，不允许真实 Lead 交易。**

## 实施及实际证据

- 在 `service/leadaccount/management.go` 新增 `LeadDurableOpenGate`：每次未来生产开仓预检重新读取独立 Lead 安全配置，内存 Pause/文件缺失或配置不允许均失败。Stage 7-1～7-4 **根本不提供**开启真实下单的成功分支，仍必须人工 Gate 7 才可能增加授权。
- 执行器 `checkOpenLocked` 的生产路径连接以上持久安全闸（Mock 证据钩子除外），避免未来仅修改 Controller `/start` 就绕过配置禁用。
- 现有 `LeadAuditStore` 新增固定动作 `recovery_inspection` 和固定理由码：显式 opt-in 的只读观察器可把只读所有权异常检查结果写入**独立本地运维审计文件**；未配置则 `audit_recorded=false`，但仍然只观察，不解锁。该文件记录不包含 API Secret、HTTP 原始内容或签名 URL。**这并不替代真实交易所订单恢复写回后的完整证据链审计**，仍是后续 Gate 的条件。
- `ValidateLeadOrderRules` 对 `PERCENT_PRICE`、`PERCENT_PRICE_BY_SIDE` 按缺乏权威动态标记价依据 **拒绝开仓**；`ValidateLeadCloseRules` 对需要动态触发/限价证据的 LIMIT/STOP/TAKE_PROFIT 条件单也保守拒绝。不能用用户传入请求价或静态 exchangeInfo 推断交易所标记价格。
- 自动化：`TestStage73DurablePauseFileBlocksAcrossInstancesAndAudit`（通过新 Store 实例模拟重启后的禁用）、`TestStage73DynamicPriceNeedsAuthoritativeReference`。Stage 4/5 mock 订单保护测试及 Race 回归通过。
- **剩余真实阻断项**：跨进程真正 MySQL 8 的长期停机/重启检查、订单/Algo 原始事件持久审计、真实 markPrice 动态过滤器、STOP/TP 的真实 Binance Algo 支持、人工部分减仓/手工挂单冲突。以上在解决前不允许实盘解锁。

## 工作边界

没有修改 `app.conf`，没有运行 `sync db`、修改真实业务库或触发任何真实 Lead Submit/Cancel/REST/WS 测试；没有自动提交/推送 Git，不回退未提交的用户修改。
