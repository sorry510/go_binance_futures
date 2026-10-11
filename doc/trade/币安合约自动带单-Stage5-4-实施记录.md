# 币安合约自动带单 — Stage 5-4 Binance REST 预算、WS 效率与账户级限速

日期：2026-10-10 至 2026-10-11（按本地测试环境）
状态：**离线实现、生产尚未接线；Mock 测试通过。真实 Binance/Testnet 环境及 Gate 5 未验收**。

## 实现与验收

- 完全复用项目已上线的 `service/binanceapiusage` 全局预算、源归属和 429/418 降级；Lead Client 的请求继续由 `lead_trading`/`lead_trading_sapi` 记录，不产生第二套全局 IP 预算。
- `service/leadaccount/metrics.go` 合并现有 Collector 中 Lead 专属五分钟请求量、估算权重、429/418，并展示独立 WS 事件数/REST 完整快照次数；**不会把 WS 消息数虚报为省下的 REST 请求数**。
- `feature/api/binance/lead_order_limiter.go` per-Key 单 10s 最多 18 次受控订单相关请求，普通开仓至多 16 次，至少保留 2 个通道名额给 CLOSE/Cancel；允许账户内安全退出意图标识，但外层共享 IP 预算依然生效。不能利用 Lead 高优先级绕过交易所全局限流。
- `feature/api/binance/lead_http_fault.go` 将 429/418 状态码直接反馈给 Lead 独立 Guard（不含 Secret/签名 URL/错误 Body）；避免 transport 内回调反向获取执行器互斥锁造成自锁。
- 验证：`TestStage54LeadPerKeyReservesCloseAndCancelSlots`、`TestStage54AccountSourceUsesSharedGlobalBudgetCollector`、`TestStage55LeadHTTP429418RoutesOnlyBoundedCodes`。
- **限制**：没有实际在线流量 A/B 评估，API 使用节省无法在本阶段作数值承诺；真实 High-load/IP 预算仍需 Testnet 验证。

## 安全约束

未修改 `app.conf`、未执行 `sync db`、未访问真实 Lead API Key 或实际下单、未自动 commit/push。Main 原有行情与交易仍为独立代码路径；这份离线记录不构成 Stage 7 实盘授权。

**Stage 5 审计 F3 修复**：Lead 的 `LeadRuleSource.exchangeInfo` 使用 **5 分钟成功快照 TTL + singleflight**，新开仓风控及受控退出共用同一公共交易规则缓存；仓位模式/杠杆档位保持独立新鲜读取。缓存到期后如果公共规则读取失败则严格阻断，不退回过期数据。无真实 API 节省的实测宣称。
