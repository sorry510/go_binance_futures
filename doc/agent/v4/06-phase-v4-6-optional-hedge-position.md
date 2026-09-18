# Phase V4-6：Optional Hedge Position

## 1. 定位

本阶段是 **P2 / 可选阶段**。

只有 V4-1～V4-5 稳定后，且真实使用证明“同一 Symbol 同时持有 LONG 和 SHORT”确实有价值时才实施。

如果没有明确需求，可以保持取消，不影响 V4 完成。

## 2. 当前基础

项目已经使用 Binance Hedge Mode，底层 Order / Ownership 已明确包含 `positionSide`。

因此：

```text
BTCUSDT LONG
BTCUSDT SHORT
```

在交易所和底层模型上可以区分。

当前限制主要来自上层：

- Auto Strategy 对已有 managed Symbol 整体 block。
- Agent Risk 将同 Symbol 的任意现有 position/order 视为 duplicate。
- Position Policy 默认不允许 opposite-side open。

## 3. 首版能力

若实施，只放开：

```text
同 Symbol、不同 PositionSide
```

例如：

```text
BTC LONG  owner=auto_strategy
BTC SHORT owner=agent_trade
```

两边独立维护：

- Managed Qty。
- SourceRef。
- Stop / TP。
- Ledger。
- Guardian 状态。
- Proposal / Audit。

## 4. 默认策略

默认仍：

```text
allow_opposite_side_position = false
```

只有用户明确启用才允许。

## 5. 风险统计

不能简单把 LONG 和 SHORT 当成两个互不相关的仓位。

至少需要同时展示：

- Gross Long Exposure。
- Gross Short Exposure。
- Gross Exposure。
- Net Directional Exposure。

Risk Engine 的 Max Total Exposure 仍优先按 gross exposure 控制，避免通过对冲绕过风险上限。

## 6. Owner 隔离

- LONG owner 不能修改 SHORT owner。
- SHORT owner 不能取消 LONG protection。
- Reconcile 使用 `symbol + positionSide`。
- Ledger 必须按 positionSide 分离。

## 7. 验收 Gate

Testnet 至少验证：

```text
Open BTC LONG
→ Open BTC SHORT
→ 两侧 Stop/TP 独立
→ Close LONG
→ SHORT 保持不变
→ Close SHORT
```

还必须验证：

- Guardian 不串边。
- Ledger 不混 PnL。
- Owner 不越权。
- Gross exposure Risk 正确。

## 8. 本阶段仍然不做

- 不允许 same-side add。
- 不做 Position Leg。
- 不做自动 Hedge Strategy。
- 不让 AI 自动决定是否启用双开模式。
