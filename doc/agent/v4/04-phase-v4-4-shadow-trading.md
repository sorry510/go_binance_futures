# Phase V4-4：Shadow Trading

## 目标

让候选 Strategy Template 在真实实时行情上运行，但**不向 Binance 提交订单**。

用于验证历史回测与真实前向信号之间是否存在明显差异。

## 执行原则

Shadow 复用正式策略判断逻辑和实时行情，不复用真实下单动作。

记录虚拟生命周期：

```text
Signal
  ↓
Virtual Entry
  ↓
Virtual Position
  ↓
Strategy Exit / TP / SL
  ↓
Virtual Close
```

## 保存内容

首版只保存必要字段：

- Strategy Template / Snapshot Hash
- Symbol
- LONG / SHORT
- Signal Time
- Virtual Entry Price / Time
- Virtual Exit Price / Time
- Exit Reason
- Gross / Net PnL
- Fee/Funding 模拟值
- Holding Time
- MarketCondition

## UI

在现有交易复盘中增加 Shadow Tab，或复用模拟盘页面能力。

用户可以：

- 启动/停止某个 Strategy + Symbol 的 Shadow。
- 查看当前虚拟仓位。
- 查看已结束 Trade。
- 查看累计结果。

## 安全边界

- Shadow 永远不能调用真实下单 Tool。
- Shadow 不创建 Ownership。
- Shadow 不修改真实仓位。
- Shadow 与现有模拟盘若逻辑可复用则优先复用，不建设重复系统。

## 验收 Gate

- Shadow 开启后实时产生信号和虚拟 Trade。
- 重启后不会错误地产生真实订单。
- 相同策略的关键开平仓判断与正式策略逻辑一致。
- 可以明确区分 Shadow、Testnet 和 Live 数据。

## 本阶段不做

- 不模拟 order book 撮合。
- 不追求高频级滑点模型。
- 不自动从 Shadow 晋级 Testnet。
