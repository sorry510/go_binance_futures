# Phase V3-1：Multi-Agent Team

## 目标

在现有 Agent Runtime 之上增加轻量 Team 协作，让不同专业 Agent 分工分析同一个市场问题，再由 Supervisor 汇总。

## 首批角色

- Technical Analyst：多周期趋势、结构、关键价位、波动。
- Flow Analyst：Funding、OI、Taker、Depth、Liquidation。
- News Analyst：新闻、公告、Alpha 等外部事件；V3-2 完成前允许数据缺失。
- Strategy Reviewer：历史策略表现、当前策略适配性。
- Supervisor：只汇总 Typed Output，不直接获得额外交易权限。

## 协作模型

`Team Run → Child Agent Tasks → Typed Results → Supervisor Result`

每个子 Agent 都继续使用现有 Task、Context、Tool、MCP、Permission、Model Gateway 和 Observability。
## 关键约束

- Team 不是多个 Agent 自由互聊；每个节点输入/输出必须有固定 Schema。
- Supervisor 不能调用子 Agent 未授权的高风险 Tool。
- Child Task 失败时必须显式标记 `data_missing` / `partial`，不能伪造结论。
- Team 有统一最大并发、Token 和 Tool Budget，避免成本失控。
- 同一个事实只采集一次时优先共享结构化 Context，避免重复请求 Binance/MCP。

## UI

- 在 Task/Observability 中展示 Team Run、子 Task、角色、耗时、Token 和最终汇总。
- 首版 Team 配置由后端定义，Web 只查看；可编辑 Team 放到 V3-9。

## 验收 Gate

- 至少一个 `symbol_analysis_team` 可稳定运行 Technical + Flow + Supervisor。
- 单个子 Agent 超时/失败不会导致其它结果丢失。
- Team 输出可 Replay，且能追踪到每个 Evidence 和 Child Task。
- 与原单 Agent `symbol_analysis` 做固定样本对比，不能显著降低稳定性。

## 本阶段不做

- 不做群聊式 Agent-to-Agent conversation。
- 不做 Agent 自主创建新 Agent。
- 不做多用户权限、审批人或组织结构。
