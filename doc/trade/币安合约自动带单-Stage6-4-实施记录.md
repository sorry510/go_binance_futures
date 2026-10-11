# Stage 6-4 — 持仓、订单与审计安全读取

> 2026-10-11；离线开发/Mock 验证，**真实 Binance 与 Stage 7 实盘未授权**。

## 实际交付

- `GET /futures/lead/positions`、`orders` 严格通过 `futuresownership.BindAccount(LeadAccountID)` 读取本地 managed 持仓（含 CLOSED）和订单，限制订单条数，不读取 Main 所有权；前端标注数据源为 **ownership_database（非实时 Binance 证明）**。
- 新增 `service/leadaccount/audit.go`：`BINANCE_LEAD_AUDIT_FILE` 指向私有文件，记录仅允许白名单 action、reason code、时间与 account_id=lead 的本地持久运维请求，绝不存密钥/原始 HTTP 错误。单次读取最多 200，超过 8 MiB 的异常审计文件拒绝读取以保护服务。
- `GET /futures/lead/events` 展示已记录的用户操作意图，而不是伪造的订单/恢复日志。Stage 5 持久权威恢复流水目前尚未独立建立，`/reconcile` 故意保持未授权。
- 实时报价、真实实际仓位、unmanaged 与 managed 数量对比、PnL 均需未来真实私有 WS/Portfolio 验证；此阶段不捏造真实余额或数量。

## 本阶段安全纪律

未修改 `app.conf`；没有执行 `sync db` 或生产数据库写入，不触发真实 Binance 下单/撤单/WS 启动；无自动 Git commit/push。历史用户未提交修改不回退。
