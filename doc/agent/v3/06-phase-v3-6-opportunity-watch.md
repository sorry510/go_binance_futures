# Phase V3-6：Opportunity Watch 与人工确认执行

> 定位：P1。让系统主动帮用户“找机会”，但不把系统变成无人值守的自动交易平台。

## 1. 已有能力

进入本阶段前已经存在：

- Scanner / FastMove / Liquidation 等本地 Signal。
- V3-2 MarketEvent / MarketFact 和 Binance Announcement / 外部数据源。
- `symbol_analysis`、`symbol_analysis_team`。
- Market Regime、Daily Market Brief、业务 Workflow。
- Agent Task / Conversation / Event / Observability。
- V2 Controlled Trade Proposal + Risk + 用户确认。

因此本阶段不再创建第二套 Agent Runtime，也不重新实现 Scanner。

## 2. 目标流程

```text
Signal / MarketEvent / Scanner
          ↓
轻量预筛选 / 去重 / 冷却
          ↓
Symbol Analysis 或 Team Analysis
          ↓
Opportunity
          ↓
Web + 通知
          ↓
用户查看
          ↓
可选：创建现有 Controlled Trade Proposal
          ↓
Risk → 用户确认 → V3-5 Safe Execution
```

自动化到“发现 + 分析 + 提醒”为止；真实订单仍然必须走现有确认链路。

## 3. Opportunity 最小数据

如果现有 Task/Workflow 结果不足以支持去重和列表展示，再新增一个轻量 Opportunity 记录，字段只需要：

- `opportunity_id`
- `symbol`
- `direction`（long / short / neutral）
- `source_type` / `source_id`
- `analysis_task_id`
- `summary`
- `confidence`
- `market_condition`
- `status`（new / reviewed / expired）
- `expires_at`
- `created_at`

不要加入版本审批、复杂 Stage、Team DAG 等与用户操作无关的字段。

## 4. 自动触发策略

首版只支持少量清楚的触发来源：

- 快速波动达到现有阈值。
- 重要 Market Event / Announcement。
- 用户配置的定时 Market Scan。
- 已有 Workflow 明确输出候选 Symbol。

同一 Symbol + 同类来源在冷却窗口内只触发一次分析，避免 Agent 成本失控。

## 5. 分析失败处理

- Agent 超轮次、Tool 失败、数据缺失时保存失败原因。
- 不因为分析失败阻塞行情采集或通知主循环。
- `partial` / `data_missing` 只能展示和提醒，不能创建真实 Trade Proposal。
- neutral 结果不创建交易 Proposal。

## 6. UI / 通知

建议在现有 AI 菜单增加一个轻量“机会”页面，或复用任务中心增加 Opportunity 视图：

- 当前机会列表。
- Symbol、方向、来源、时间、MarketCondition、摘要。
- 查看完整分析。
- 一键跳到单币分析 / Team 分析。
- 满足现有 `symbol_analysis` Proposal 契约时，允许用户点击“创建受控交易 Proposal”。

通知只发高价值 Opportunity，避免每次 Scanner 命中都推送。

## 7. 为什么不做 Shadow / Assisted / Auto 三套模式

个人使用时，三套执行模式会增加大量状态和配置，却没有实际收益。本项目统一采用：

> **自动发现和分析，真实执行人工确认。**

如果未来长期使用后确实需要全自动真实交易，再单独评估，不预先为未知需求建设 Auto Policy 平台。

## 8. 验收 Gate

- 自动扫描能稳定生成可读 Opportunity，不重复轰炸同一 Symbol。
- Opportunity 能追踪回原始 Signal/Event 和分析 Task。
- neutral / partial / failed 分析不能进入真实交易。
- 没有用户确认时，Opportunity Pipeline 的 Binance Submit 次数始终为 0。
- Opportunity 过期后不能再创建交易 Proposal。
- Agent 失败不会影响 Market Intelligence / Scanner 主链路。

## 9. 本阶段明确不做

- 不做 Auto Trading Mode。
- 不做 LLM 自我批准。
- 不做复杂 Opportunity Ranking 模型或机器学习排序。
- 不做企业级队列、审批和 SLA。
