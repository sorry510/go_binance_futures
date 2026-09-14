# Phase V3-7：交易复盘与策略比较

> 状态：✅ 已完成。个人自用轻量实现，优先复用现有表和确定性 SQL/Go 聚合。
>
> 定位：P1。解决个人使用最实际的问题：哪些策略、哪些行情和哪些分析真的有效。

## 1. 目标

不建设大型 Attribution / Continuous Eval 平台，而是把现有数据整理成用户能直接用于决策的复盘页面。

已有数据已经非常丰富：

- Backtest Run：完整 Technology / Strategy Snapshot、Trades、Events、Equity、Metrics、DataHash。
- 模拟盘：`test_strategy_results` 保存完整 Technology / Strategy、Snapshot Hash、手续费和开平仓结果。
- Controlled Trade：Proposal、Risk、Execution、Audit、来源 Task。
- Agent：Task、Team Child Task、Model、Tool、Evidence、Observability。
- Market：MarketCondition、MarketEvent / Fact、Historical Market Repository。

本阶段优先做“查询和聚合”，不是再复制一份数据。

## 2. 三类结果分开看

### Historical Backtest

按 Strategy Template / Symbol / interval 比较：

- Return / Net PnL
- Max Drawdown
- Win Rate
- Profit Factor
- Fees / Funding
- Trade Count
- LONG / SHORT
- MarketCondition 分组

### 模拟盘

按 `strategy_template_id` 或 `strategy_snapshot_hash` 汇总：

- 已平仓数量 / 未平仓数量
- 净收益
- 胜率
- 手续费
- 平均持仓时间
- Symbol 分布

### 真实受控交易

最终个人版按确定性数据边界收窄：以 Proposal 为入口查看 Risk、Execution、Managed Position / Order 和 Audit 链路，并保留来源 Task、Symbol、Direction、MarketCondition 与执行异常信息。

- 页面展示 Proposal / 已执行 / Managed Position / Managed Order 统计。
- 最近 Proposal 可以在复盘页直接下钻到 Proposal → Risk → Execution → Managed Position → Audit。
- Entry / Stop / TP 等计划字段可在 Proposal 详情中查看。
- 实际 PnL、手续费、Funding 只有在能够可靠归因到受控仓位时才展示；当前版本无法确定性归因，因此明确不计算、不估算。

不要把 Backtest、Paper、Live 混成一个收益率指标。

## 3. 策略比较

用户选择两个或多个 Strategy Template 时，可以比较：

```text
策略 A              策略 B
Backtest Return      Backtest Return
Max Drawdown         Max Drawdown
Trade Count          Trade Count
Paper Net PnL        Paper Net PnL
Paper Win Rate       Paper Win Rate
```

由于用户约定“不修改旧策略，只创建新模板”，Strategy Template ID 本身就是主要比较维度，不需要 Strategy Version。

## 4. 实用归因

只保留能帮助交易决策的维度：

- Symbol
- Strategy Template / Snapshot Hash
- LONG / SHORT
- MarketCondition
- Signal / MarketEvent 来源
- 单 Agent / Team Analysis
- 持仓时间
- Fee / Funding
- MFE / MAE（如果可以从已有 Kline 确定性计算）

Model / Prompt / Skill 的调用成本和失败率继续放在现有 Observability，不在这里重新做模型排行榜。

## 5. UI

建议新增或整理成“交易复盘”页面：

- 顶部时间范围。
- Backtest / 模拟盘 / 真实交易三个 Tab。
- Strategy / Symbol / MarketCondition 筛选。
- 常用 Metrics 卡片和简单趋势图。
- 策略对比入口。
- 从真实交易可以跳回来源 Task、Risk 和 Execution Audit。

以“看得懂、能比较”为优先，不追求 BI 平台。

## 6. 数据计算原则

- 所有 PnL / Fee / Funding 聚合使用确定性 Go/SQL 逻辑。
- 不让 LLM 计算财务指标。
- Snapshot Hash 相同的模拟盘结果可以聚合；Hash 不同不能强行合并。
- Backtest 使用 Run 自己保存的 Snapshot / DataHash，不读取当前 Strategy Template 内容替代历史快照。
- 多 Run 的 `ReturnPct = ΣNetPnL / ΣInitialEquity × 100%`，表示按初始资金加权的 Run 收益口径，不等同于把多个 Run 串成一条组合权益曲线。
- 多 Run 最大回撤取匹配 Run 的最大值；按 LONG/SHORT 或 MarketCondition 等交易子集筛选时无法确定性重建子集权益曲线，因此返回“不可用”而不是估算。

## 7. 验收 Gate

- Backtest 汇总与 `agent_backtest_runs/trades` 可逐项对账。
- 模拟盘汇总与 `test_strategy_results` 可逐项对账。
- 真实交易能追踪到 Proposal → Risk → Execution → Audit。
- 策略 A/B 比较不会因为模板后来新增而改变旧 Run 的历史结果。
- 聚合计算有固定 Fixture 测试。
- 页面查询不能明显影响行情和交易主循环。

## 8. 本阶段明确不做

- 不做在线强化学习。
- 不做自动修改策略。
- 不做自动“淘汰/晋级”策略。
- 不做复杂 Model/Prompt Attribution 平台。
- 不做机构级 BI/Data Warehouse。

## 9. 最终实现与验收记录（个人版）

- 新增 `service/outcomereview` 和 `/agents/outcomes/backtest|paper|live` 只读 API。
- Backtest 使用 SQL 聚合交易结果，支持时间、策略、Symbol、方向筛选，并按 Symbol / LONG-SHORT / MarketCondition 分组。
- 模拟盘直接复用 `CalculateTestResultReviewStats`，保持 Snapshot Hash 隔离和“未平仓不污染已实现收益”的既有规则。
- Live 统计 Proposal、已执行交易、Managed Position 和 Managed Order，并返回最近 20 条 Proposal 供页面下钻；详情复用既有只读接口展示 Risk、Execution、Managed Position 和 Audit。当前本地数据无法确定性归因完整手续费/Funding，因此不伪造 Live Net PnL。
- 前端“交易复盘”放在“合约交易”菜单，Backtest / 模拟盘 / Live 三个 Tab 分开显示；策略比较最多选择 3 个模板，不计算综合评分。
- 未新增数据库表，数据库版本保持 v13，不需要额外 `sync db`。
- Backtest 汇总采用数据库聚合，避免复盘查询把大量历史 Trade 全部加载到 Go 内存。
- Live 当前受控仓位统计与 Ownership 口径一致，仅将 `managed_qty > 0` 且未关闭的仓位计为 Open Position。

### 自动化验证

- `go test ./...`：通过。
- `go vet ./...`：通过。
- `go test -race ./service/outcomereview ./controllers ./routers`：通过。
- Outcome Review SQLite 固定 Fixture：通过。
- 前端 `vue-tsc --noEmit`：通过。
- 前端 `pnpm build`：通过，`dist` 已同步到后端 `static`。
