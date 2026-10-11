# 币安合约自动带单 — Stage 5-5 Lead 故障通知、状态回调与非阻塞处理

日期：2026-10-10 至 2026-10-11（按本地测试环境）
状态：**离线实现、生产尚未接线；Mock 测试通过。真实 Binance/Testnet 环境及 Gate 5 未验收**。

## 实现与验收

- `service/leadaccount/alerts.go`：有界非阻塞队列和单个有生命周期的 Worker，队列背压丢弃会计数但**绝不改变风险阻断**，发送失败同样不自动恢复开仓；事件只接受固定 Lead AccountID/故障码，不传播原始 HTTP 内容。
- `SendLeadAlertToConfiguredChannel` 复用现有 DingDing/Slack `AgentAlert` 接口，标识 `Module=lead_trading`；只有显式 Run 才会调用配置的通知服务，应用启动并未添加新的后台发送进程。
- 对每次连续 429/418 故障分别最多通知 3 次、至少相隔 2 分钟；下一次有效恢复后的故障可重新开启次数预算；WS/身份/保护单等原因通过同一状态机记录。
- `service/leadaccount/supervisor.go` 将显式启动恢复、私有 WS、Guard、HTTP 429/418 回调和通知队列整合；context 取消时停止本账户 Worker/ListenKey，不停止 Main。
- 验证：`TestStage55RateLimitThreeAlertsAtTwoMinuteSpacing`、`TestStage55BoundedQueueNeverBlocksRiskAndWorkerStops`、`TestStage55QueueConcurrencyAndSanitization`、`TestStage56SupervisorRecoveryStartsBeforeWSAndStopsAllWorkers`。
- **限制**：真实 DingDing/Slack 网络回调的时延、超时约束还需要生产级集成检查；目前只有 Fake Sender 的 Worker 生命周期被验证。

## 安全约束

未修改 `app.conf`、未执行 `sync db`、未访问真实 Lead API Key 或实际下单、未自动 commit/push。Main 原有行情与交易仍为独立代码路径；这份离线记录不构成 Stage 7 实盘授权。

**Stage 5 审计 F2 修复**：`LeadAlertQueue.Snapshot()` 输出当前队列深度、容量、累计丢弃数、严重级丢弃数；`LeadUsageSnapshotWithAlerts` 与 `LeadSupervisor.Status()` 暴露同样的 Lead 观测字段。首次发生 Critical/其他级别背压时记录不含敏感数据的固定日志；通知仍为有界非阻塞、失败不放行。**生产 System Dashboard 还没有新增 Lead 区块或路由**，将在 Stage 6 鉴权界面接线。
