# Phase V4-2：Robustness Check

## 目标

在批量回测结果上增加少量但真正有价值的稳健性检查，优先发现明显过拟合。

## 首版只检查四件事

### 1. 交易频率

计算：

- 总 Trade Count
- Trades / Week
- LONG / SHORT Trades / Week

当新策略相对被比较策略的交易频率明显下降时，在比较结果中提示。

系统只展示变化，不自动判断“新策略更好”。

### 2. 跨 Symbol 稳定性

统计一批 Symbol 中：

- 盈利 Run 数 / 亏损 Run 数
- Profit Factor > 1 的 Run 数
- 最大和最小 Return
- 最大 Max Drawdown
- Trade Count 分布

避免只看 BTC/ETH。

### 3. 收益集中度

计算：

- 最大单笔盈利占总正收益比例
- Top 3 盈利 Trade 占总正收益比例

用于发现“整个策略靠极少数交易撑起来”的情况。

### 4. 结果退化

比较两个 Strategy Template 的同 Symbol、同时间范围结果：

- Return 变化
- Max Drawdown 变化
- Profit Factor 变化
- Trade Count / Trades per Week 变化

## UI

优先扩展现有 Backtest Compare / Outcome Review。

只显示指标和明确提示，例如：

```text
交易频率：-43%
跨币种：6 个 Symbol 中 2 个盈利
Top 3 Trade：贡献总正收益的 71%
```

不输出 0～100 分的“稳定性评分”。

## 验收 Gate

- 所有指标可由现有 Run/Trade 确定性重算。
- 相同输入重复计算结果一致。
- 指标缺少有效样本时返回不可用，不伪造结论。
- 不把不同资金规模或不同 Snapshot 的结果错误合并。

## 本阶段不做

- 不做 Monte Carlo。
- 不做复杂统计显著性检验。
- 不做自动淘汰策略。
- 不做大规模参数敏感性扫描。
