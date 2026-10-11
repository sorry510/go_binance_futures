# 币安合约自动带单 — Stage 5-1 Lead 私有 User Data WS 生命周期

日期：2026-10-10 至 2026-10-11（按本地测试环境）
状态：**离线实现、生产尚未接线；Mock 测试通过。真实 Binance/Testnet 环境及 Gate 5 未验收**。

## 实现与验收

- 新增 `feature/api/binance/lead_stream_client.go` 的 Lead 专属 ListenKey 清理与复用现有代理的 WS Dial；`service/leadaccount/stream.go` 通过显式 `Run(ctx)` 创建私有连接，`Stop()` 取消及有界清理；不能在模块导入/主启动时自动连接。
- 每次连接生成独立 generation，旧 WS 事件不得更新新镜像；ListenKey 过期/keepalive 错误/WS 异常触发断线重建，Lead 的连接管理与 Main 完全分开。
- **竞态保护**：先建立 WS，再读完整 REST 余额/持仓/挂单；在 REST 启动窗口收到任一 WS 事件时将该代镜像视为不可信并重新拉快照，不把有缺口数据提升为可交易证据。
- 验证：`TestStage51MirrorRequiresCompleteLeadSnapshotAndGeneration`、`TestStage51StreamLifecycleReconnectKeepaliveAndStop`、`TestStage51ClientRejectsMainAccount`、`TestStage56PreSnapshotWSMessageForcesResync`。
- **限制**：尚未连真实 Lead ListenKey/实际交易所 WS；Stage 5 实现为显式 opt-in，生产主交易循环没有自动启用。

## 安全约束

未修改 `app.conf`、未执行 `sync db`、未访问真实 Lead API Key 或实际下单、未自动 commit/push。Main 原有行情与交易仍为独立代码路径；这份离线记录不构成 Stage 7 实盘授权。
