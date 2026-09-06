# Phase V3-3：Historical Backtest Engine

## 目标

实现正式历史回测，不再把固定场景 Strategy Experiment 当作历史回测。

## 核心能力

- 基于历史 K 线和必要市场数据驱动策略逐步执行。
- 支持 indicator warmup，严格禁止未来函数。
- 支持 LONG / SHORT、开仓、平仓、止盈、止损。
- 计入手续费、Funding 和可配置滑点。
- 记录每次信号、订单、成交、仓位和资金曲线。
- 回测结果与 `StrategyVersion + DatasetVersion + EngineVersion` 绑定，可完全复现。

## 指标

至少输出 Net PnL、Max Drawdown、Win Rate、Profit Factor、Sharpe、Sortino、Trade Count、Fees、Funding、Average Holding Time，并按 LONG/SHORT 和 MarketCondition 分组。
## 数据与执行原则

- Dataset 必须有明确时间范围、Symbol、Interval 和版本 Hash。
- 回测运行时只读取当时可见数据；新闻/MarketEvent 后续接入时必须使用 event_time/observed_at 语义避免未来数据泄漏。
- Backtest Engine 不调用 LLM 决定每根 K 线交易动作；策略执行必须确定性。
- AI 只用于生成候选策略、解释结果和比较不同版本。

## UI

- 新增回测任务、进度、结果详情和 Equity Curve。
- 支持选择 Strategy Version、Symbol、时间范围和关键回测参数。
- 支持结果对比，但首版不做复杂参数寻优。

## 验收 Gate

- 同一 Dataset + Strategy + Engine Version 重跑结果一致。
- 手续费、Funding、滑点和止损均有单元测试。
- 有未来函数检测或明确的时间访问边界测试。
- 至少用固定历史 Fixture 验证 LONG、SHORT、止盈、止损、无交易五种场景。
- 回测不会修改真实策略、模拟盘状态或真实 Binance 账户。

## 本阶段不做

- 不做遗传算法/大规模参数优化。
- 不让 LLM 在回测循环里逐 K 线做决策。
- 不做策略自动上线。
