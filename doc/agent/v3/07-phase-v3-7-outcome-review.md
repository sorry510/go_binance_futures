# Phase V3-7：交易复盘与策略比较

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

按 Proposal / Symbol / Direction 查看：

- Entry / Exit / Stop / TP
- 实际 PnL、手续费、Funding（能获取时）
- Risk 计算值
- 来源分析 Task
- MarketCondition
- 执行异常 / Reconcile 情况

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
