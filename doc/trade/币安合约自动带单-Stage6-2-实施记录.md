# Stage 6-2 — 安全凭证配置与只读验证

> 2026-10-11；离线开发/Mock 验证，**真实 Binance 与 Stage 7 实盘未授权**。

## 实际交付

- 复用原 Stage 4-1 的 AES-GCM `EncryptedFileStore`、独立环境变量 `BINANCE_LEAD_CREDENTIAL_KEY` 与 `BINANCE_LEAD_CREDENTIAL_FILE`，并未将 Lead API Secret 写入 app.conf/数据库。
- `PUT /futures/lead/credentials` 经过 JWT + 私有审计目录条件，支持密钥保存/轮换；Manager 自动使旧只读验证结果失效。`POST /futures/lead/verify` 仅用户明确点击后调用只读 Binance Reader，不自动启动程序。
- 状态 API 只回显固定尾号和受控验证摘要，不返回 API Secret、签名 URL。未配置环境变量时 API fail-closed。
- `DELETE /futures/lead/credentials` 为固定 HTTP 423：现阶段无法从只读/离线状态证明不存在任何未决/人工仓位或保护性挂单，绝不冒险删除。

## 本阶段安全纪律

未修改 `app.conf`；没有执行 `sync db` 或生产数据库写入，不触发真实 Binance 下单/撤单/WS 启动；无自动 Git commit/push。历史用户未提交修改不回退。
