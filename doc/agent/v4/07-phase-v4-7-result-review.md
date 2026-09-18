# Phase V4-7：Unified Result Review

## 目标

在现有 V3-7 Outcome Review 上补齐 V4 新结果，让用户能快速看到“历史表现”和“前向表现”是否一致。

不重写复盘系统。

## 四类结果继续分开

```text
Backtest
Shadow
Testnet
Live
```

不把四者合并成一个 PnL。

## 比较维度

按 Strategy Template / Symbol 查看：

- Return / Net PnL
- Max Drawdown
- Profit Factor
- Trade Count
- Trades / Week
- Win Rate
- LONG / SHORT
- Average Holding Time

Backtest 额外展示：

- 跨 Symbol 稳定性。
- Time Validation。

Shadow/Testnet 额外展示：

- 前向运行天数。
- 实际 Trade Count。
- 与最近对应 Backtest 的明显偏差。

## 差异展示

允许简单显示：

```text
Backtest Trades/Week: 2.1
Shadow Trades/Week:   0.8

Backtest PF: 1.46
Shadow PF:   1.05
```

系统只描述差异，不自动给“推荐上线”结论。

## 验收 Gate

- V3 原有 Backtest/Paper/Live 复盘不回归。
- Shadow/Testnet 使用独立数据来源和明确环境标识。
- 同一策略不同 Snapshot 不错误合并。
- 查询保持轻量，不扫描大规模原始 Kline。
- 所有展示指标能下钻到对应 Run/Trade。

## 本阶段不做

- 不做 BI 平台。
- 不做综合评分。
- 不做策略排行榜。
- 不做自动上线决策。
