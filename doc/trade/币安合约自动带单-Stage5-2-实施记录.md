# 币安合约自动带单 — Stage 5-2 Lead WS 账户镜像与乱序数据保护

日期：2026-10-10 至 2026-10-11（按本地测试环境）
状态：**离线实现、生产尚未接线；Mock 测试通过。真实 Binance/Testnet 环境及 Gate 5 未验收**。

## 实现与验收

- `service/leadaccount/mirror.go` 增加独立 Lead 内存镜像：REST 完整 Seed 才 ready，隔离 USDT/其他资产、双向仓位与原始订单的累计成交。所有 WS 事件只更新本地只读投影，不自动认领订单或改写生产 Ownership 表。
- `ORDER_TRADE_UPDATE` 按 OrderID/交易事件时间/累计成交量严格单调校验，终态之后拒绝旧事件导致挂单重新出现；`ACCOUNT_UPDATE` 逐腿与逐资产校验及时间水位，不允许旧数据回滚新持仓。
- `ALGO_UPDATE`/不完整数据不能冒充最终实际子订单，须由 Stage 4-4 只读 Algo 恢复与 Stage 5-3 持久化对账佐证。
- 验证：`TestStage52MirrorMonotonicOrderFillsAndTerminalWatermark`、`TestStage52InvalidBalanceUpdateDoesNotPartiallyChangeMirror`、generation race 测试。
- **限制**：WS 镜像绝不单独授权新开仓；日 PnL 与真实资产身份验证仍须完整 REST/Portfolio 证据。

## 安全约束

未修改 `app.conf`、未执行 `sync db`、未访问真实 Lead API Key 或实际下单、未自动 commit/push。Main 原有行情与交易仍为独立代码路径；这份离线记录不构成 Stage 7 实盘授权。
