# Phase V3-6：Opportunity Watch 与人工确认执行

> 状态：✅ 已完成。定位：P1。系统主动发现、分析和提醒机会，但真实交易仍必须由用户确认。

## 1. 边界

V3-6 不创建第二套 Scanner、Agent Runtime、Risk 或下单系统，而是把已有能力串成可追踪的机会闭环：

```text
Signal / MarketEvent / Market Scan
          ↓
Opportunity Watch（预筛选 / 去重 / 冷却）
          ↓
symbol_analysis → TradingPlanV1
          ↓
AgentOpportunity
          ↓
Web + 高价值通知
          ↓
用户查看 / 重新分析 / 创建 Proposal
          ↓
现有 Risk → 人工批准 → V3-5 Ownership-safe Execution
```

自动化边界固定在“发现 + 分析 + 提醒”。Opportunity 不会自动批准或执行真实交易。

## 2. V3-6A：Opportunity Foundation

新增 `agent_opportunities`，数据库版本由 **v12 → v13**。完整 AI 结果继续保存在 `AgentTask`，Opportunity 只保存列表和生命周期所需字段：

- `opportunity_id`
- `symbol` / `direction`
- `source_type` / `source_id`
- `analysis_task_id`
- `summary` / `confidence` / `market_condition`
- `analysis_status` / `analysis_error`
- `status` / `expires_at` / `reviewed_at`
- `created_at` / `updated_at`
### 去重与生命周期

- `(source_type, source_id)` 使用数据库唯一约束，保证同一 Signal/Event/Scan candidate 重放不会重复创建。
- 冷却按 `symbol + source_type`，默认 30 分钟；不同来源不会互相错误 suppression。
- Opportunity 默认 TTL 为 60 分钟。
- 用户状态：`new / reviewed / expired`。
- 分析状态：`pending / succeeded / partial / data_missing / failed`。

只有以下条件全部满足时 `CanCreateProposal()` 才返回 true：

```text
analysis_status == succeeded
AND direction in (long, short)
AND opportunity 未过期
```

`neutral / partial / data_missing / failed / expired` 全部 fail-closed。

## 3. V3-6B：自动分析 Pipeline

Opportunity Pipeline 使用有界队列和固定 Worker，Trigger 进入后：

1. 校验 USDT Futures Symbol 与 Source。
2. 检查 source replay。
3. 检查 Symbol + SourceType cooldown。
4. 创建 `pending` Opportunity。
5. 启动现有 `symbol_analysis`，记录 `analysis_task_id`。
6. 等待 Agent Task 终态并解析 `TradingPlanV1`。
7. 更新方向、Confidence、MarketCondition、摘要与分析状态。
8. 高价值机会达到配置的最低 Confidence 后才发送通知。

Agent 启动失败、超时、Tool/LLM 失败只会把 Opportunity 标成 failed，不会阻塞行情、Scanner 或报警主链路。
## 4. V3-6C：Trigger Integration

首版实际接入三类来源：

- `fast_move`：复用现有 Signal Engine。
- `liquidation_spike`：复用现有强平 Signal。
- `market_event`：新创建且能解析出 Symbol 的 MarketEvent / Binance Announcement。
- `market_scan`：复用现有 Agent Scheduler 和 `market_scan` Skill。

Signal 路径采用非阻塞 fan-out：

```text
Signal Engine
  ├─ Alert Pipeline（原逻辑）
  └─ Opportunity Pipeline.Emit（非阻塞）
```

Market Intelligence 中的 `signal` 类型 Event 不会再次触发 Opportunity，避免同一个 FastMove 经过持久化后被重复分析。

定时 Market Scan 使用现有 Scheduler Job `opportunity_market_scan`。先由 Scanner/market_scan 排名，再只把达到 `AgentOpportunityMinConfidence` 的候选送进 `symbol_analysis`，不会把整个候选集全部深度分析。

## 5. 配置

AI 配置页仅增加三个字段：

- `AgentOpportunityWatchEnable`：总开关，默认关闭。
- `AgentOpportunityScanIntervalMin`：Market Scan 周期，默认 60 分钟。
- `AgentOpportunityMinConfidence`：Market Scan 深入分析和主动通知的最低 Confidence，默认 0.7。

Cooldown（30 分钟）和 Opportunity TTL（60 分钟）使用代码默认值，不继续膨胀配置项。
## 6. V3-6D：UI 与通知

新增 `AI → 机会` 页面：

- 按 Symbol、机会状态、分析状态、来源筛选。
- 展示方向、Confidence、来源、MarketCondition、摘要、发现时间和状态。
- 打开详情时自动标记 `reviewed`。
- 详情读取原 `AgentTask` 的完整 `TradingPlanV1`，不复制分析 JSON。
- 支持跳转到现有单币分析页面重新分析并预填 Symbol。
- 已存在 Trade Proposal 时显示 Proposal ID 并跳转“仓位与受控交易”。

高价值 Opportunity 复用现有通知通道。通知阈值只控制主动提醒，不替代 Proposal/Risk 的交易安全规则。

## 7. V3-6E：Controlled Trade Bridge

Opportunity 不直接调用 Binance，也不自动创建执行订单。唯一桥接路径为：

```text
AgentOpportunity
  ↓ CanCreateProposal()
analysis_task_id
  ↓
agenttrade.CreateFromTask()
  ↓
Deterministic Risk
  ↓
用户 Approve
  ↓
V3-5 Ownership-safe Execution
```

`CreateFromTask()` 继续只接受成功的 `symbol_analysis`，并保持 task 级幂等；Market Scan、Alert Analysis、Team Analysis 都不能直接成为真实交易 Proposal。

## 8. API

- `GET /agents/opportunities`
- `GET /agents/opportunities/:opportunityId`
- `POST /agents/opportunities/:opportunityId/review`
- `POST /agents/opportunities/:opportunityId/proposal`
## 9. 验收 Gate

必须满足：

- 同一 source 重放只产生一条 Opportunity。
- 同 Symbol + 同类来源在 cooldown 内不重复启动 Agent Task。
- 不同来源不会被错误合并。
- Opportunity 可追踪到原始 source 和 `analysis_task_id`。
- `neutral / partial / data_missing / failed / expired` 不能创建 Proposal。
- Opportunity Pipeline 失败不会影响 Alert Pipeline、Market Intelligence 或 Scanner。
- 未经用户批准时，Opportunity 自身没有 Binance Submit 路径。
- Opportunity 创建 Proposal 后仍必须经过现有 Risk、人工批准和 V3-5 Ownership Executor。
- 中英文 UI、通知、README 和本阶段文档同步。
- `go test ./...`、race、vet、build、前端 typecheck/build、diff check 全部通过。

## 10. 本阶段明确不做

- 不做 Auto Trading Mode。
- 不做 LLM 自我批准或自动 Execute Proposal。
- 不做自动加仓、反手和复杂动态 TP。
- 不做新的 Scanner、Agent Runtime、Risk Engine 或订单系统。
- 不做机器学习 Opportunity Ranking。
- 不做企业级队列、审批、RBAC 或 SLA。

升级数据库仅使用：

```bash
./go_binance_futures sync db
```

不在服务启动时自动迁移数据库。
