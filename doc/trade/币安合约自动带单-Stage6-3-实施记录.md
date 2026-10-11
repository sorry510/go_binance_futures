# Stage 6-3 — Lead 独立风控参数与暂停

> 2026-10-11；离线开发/Mock 验证，**真实 Binance 与 Stage 7 实盘未授权**。

## 实际交付

- 新增 `service/leadaccount/management.go`：`LeadSettings` 与 `LeadConfigStore`，通过 `BINANCE_LEAD_CONFIG_FILE` 指向仓库外私有目录（0700），文件模式 0600，原子写入；不修改 app.conf，也不写生产数据库。
- 客户端只能配置正当风控数值：单笔/总名义额、日亏损熔断、回撤、最大持仓/亏损仓位；账户固定 `lead`。任何 `enabled=true`、`allow_new_opens=true`、负值/NaN/无效额度一律拒绝。
- `POST /futures/lead/pause` 幂等地将安全设置写为 disabled/禁止新开仓，不能恢复开仓。Main 策略配置仍为共享只读说明，没有复制第二套自定义策略编辑器。
- 配置文件若缺失/路径不安全，默认禁用并返回安全的不可用状态；需要用户在将来部署时显式提供文件路径，不擅自初始化用户服务器文件。

## 本阶段安全纪律

未修改 `app.conf`；没有执行 `sync db` 或生产数据库写入，不触发真实 Binance 下单/撤单/WS 启动；无自动 Git commit/push。历史用户未提交修改不回退。
