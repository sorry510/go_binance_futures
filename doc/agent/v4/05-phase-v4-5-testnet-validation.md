# Phase V4-5：Testnet Validation

## 目标

把已经完成的 Binance Futures Testnet 执行能力正式纳入策略前向验证流程。

V4-5 不新建交易执行系统，只复用 V3-5 Ownership-safe Execution。

## 功能

- 用户手动选择一个已测试 Strategy Template + Symbol 进入 Testnet 验证。
- Testnet 使用真实策略信号和真实订单生命周期。
- 所有 Testnet Order/Position 必须保持明确 owner。
- 继续验证开多、平多、开空、平空、TP、SL、重启恢复。
- 记录与 Shadow/Backtest 可比较的 Trade 结果。
- 明确标记环境为 `testnet`，不得与 Live 聚合。

## 关注指标

只保留：

- Trade Count
- Net PnL
- Win Rate
- Profit Factor
- Max Drawdown（能够确定性计算时）
- Average Holding Time
- 实际执行异常数量
