# 06. Phase 5：事件驱动 AI 报警

## 目标

把现有 WS 快速波动/监听逻辑升级为统一的事件链：高速数据先由确定性代码筛选，只有形成有意义 Signal 后才调用 Agent。

## 目标链路

```text
Binance WS
  -> Event Bus
  -> Signal Engine
  -> Signal
  -> AlertAnalysis Skill（可选）
  -> Notification Service
```

关键约束：**禁止每个 WS Tick 调用 LLM**。

## Event Bus

第一版不需要引入 Kafka/NATS。进程内 typed channel/pub-sub 足够，先定义稳定事件类型：

```text
PriceTickEvent
KlineClosedEvent
LiquidationEvent
FundingRateEvent
PositionEvent
WsHealthEvent
```

事件必须带 `event_id`、`type`、`symbol`、`event_time`、`source`，方便去重和审计。

## Signal Engine

Signal Engine 使用纯 Go 规则，将高频 Event 聚合为低频业务 Signal。
第一批 Signal 建议：

```text
FastMoveSignal
VolumeSpikeSignal
OISpikeSignal
LiquidationSpikeSignal
FundingExtremeSignal
BreakoutSignal
BreakdownSignal
ShortSqueezeCandidate
LongSqueezeCandidate
```

Signal 至少包含：`signal_id`、`symbol`、`type`、`severity`、`window`、`metrics`、`evidence`、`created_at`。

## 与现有 WS 逻辑的迁移

FastMove 与 Liquidation 已完成最终切流：

1. Binance WS 只负责发布 `PriceTickEvent` / `LiquidationEvent`。
2. 阈值、窗口、聚合、去重和 Signal cooldown 统一由 `service/signal` 处理。
3. FastMove 与 Liquidation 的旧检测器、旧聚合器和直接通知实现已经删除。
4. `agent_alert_pipeline_enable=0` 时 Signal 继续监测，但这两类 Signal 不发送通知。
5. Pipeline 开启后，唯一通知出口为 `Alert Pipeline -> AgentAlert -> notifications`。

## AlertAnalysis Skill

不是每个 Signal 都必须调用 AI。先由规则判断：

- severity 低：直接模板通知或仅记录。
- severity 中高且满足 AI 开关：调用 `alert_analysis`。
- 同一 symbol/type 在 cooldown 内禁止重复 AI 分析。

Agent 可补充调用只读 Tool，判断异常是否得到 Funding/OI/强平/大盘环境确认。
## 报警输出

Alert 最终建议结构化：

```text
alert_id, signal_id, symbol, type, severity
summary
market_context
confirmed_by[]
risks[]
action: notify/record/ignore
cooldown_until
```

`action` 只决定是否通知/记录，不直接触发交易。

## 稳定性要求

- Event Bus 消费失败不能阻塞 Binance WS 回调。
- Signal Engine 必须有 bounded buffer / drop policy / metrics。
- 相同 signal 做幂等和冷却。
- AI 不可用时保留规则模板通知 fallback。
- WS 无数据健康报警属于确定性系统告警，不依赖 AI。
- Alert Agent 有独立并发上限和每分钟调用预算，行情极端时不能形成 LLM 风暴。

## 当前实现

Phase 5 第一版已经完成，并已完成 FastMove / Liquidation 最终切流：

- `agent/event`：进程内 typed Event Bus，bounded queue、非阻塞 Publish、drop policy 和统计指标。
- Binance 全市场 ticker 发布 `PriceTickEvent`；强平 WS 发布带稳定事件 ID 的 `LiquidationEvent`；WS 无数据报警额外发布 `WsHealthEvent`。
- `service/signal`：首批实际检测 `FastMoveSignal` 与 `LiquidationSpikeSignal`；其余 Signal 类型先固化契约，后续按业务需要增加检测器。
- `alert_analysis`：只消费已筛选 Signal，强制使用只读 Tool 补充市场上下文，只能返回 `notify/record/ignore`。
- `service/alertpipeline`：severity gate、symbol/type cooldown、AI 并发限制、每分钟预算、AI 失败 deterministic fallback、最近 Trace 和运行指标。
- Notification 持久化 `event_id/signal_id/task_id`，支持 Event → Signal → Agent Task → Notification 追踪。
- FastMove / Liquidation 的 legacy 检测和直接通知实现已删除；`agent_alert_pipeline_enable=0` 时只监测 Signal 不通知，开启后统一由新 Pipeline 处理。
- `agent_alert_analysis_enable=0` 或 AI 不可用时仍发送规则 fallback，不影响 WS 数据链。
- WS 无数据健康报警始终由确定性逻辑发送，不依赖 AI。

## Phase 5 验收

- [x] 正常 WS 高频流量不会触发同等数量的 LLM 请求。
- [x] 可从 Event 追踪到 Signal、Agent Task、Notification。
- [x] 关闭 AI Alert 后，基础确定性报警仍能工作。
- [x] 重复行情不会突破 cooldown 连续发送相同报警。
- [x] Agent/Notify 失败不会影响 WS 数据持续写入和价格刷新。