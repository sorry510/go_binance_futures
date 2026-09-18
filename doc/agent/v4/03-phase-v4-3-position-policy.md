# Phase V4-3：Unified Position Policy

## 1. 目标

把当前散落在 Auto Strategy、Agent Risk 和 Ownership 中的仓位冲突规则整理为一个明确、可测试的 Position Policy。

本阶段的目标是**统一规则，不是增加交易自由度**。

## 2. Policy 输入

至少包含：

- Symbol。
- Requested PositionSide。
- Owner。
- Intent。
- Existing Account Position。
- Existing Managed Position。
- Existing Managed/Open Order。
- Existing Owner / SourceRef。

## 3. 首版规则

默认保持 V3 保守行为：

```text
same-side add              = deny
opposite-side open         = deny
multiple managed owner     = deny
merge manual position      = deny
claim unmanaged position   = deny
```

也就是说，本阶段完成后真实行为原则上不变。

## 4. 统一回答的问题

例如：

```text
BTC LONG 已存在，能否再开 LONG？
BTC LONG 已存在，能否开 SHORT？
LONG 属于 auto_strategy，agent_trade 能否再开 LONG？
账户有 manual LONG，系统能否认领？
存在 pending LONG open order 时是否允许其它 owner 下单？
```

所有入口得到同样结论。

## 5. 与 Risk Engine 的边界

Position Policy 只负责 position conflict / ownership conflict。

Risk Engine 继续负责：

- kill switch。
- allowlist。
- MarketCondition。
- price freshness。
- slippage。
- max risk。
- notional。
- total exposure。
- leverage。
- cooldown。

不重写 V2 Risk Engine。

## 6. 与 Ownership 的边界

Ownership 仍负责“谁可以修改哪一部分真实仓位”。

Position Policy 负责“一个新的 mutation 是否允许进入 Ownership/Executor”。

## 7. 验收 Gate

- Auto Strategy、Agent Trade 和其它真实 Futures owner 使用一致的冲突语义。
- 默认行为与 V3 保持兼容。
- manual/unmanaged 永远不会被自动认领。
- Policy 决策具有明确 reason code，便于日志和 UI 展示。
- 并发真实开仓仍不能绕过 Ownership slot safety。

## 8. 本阶段不做

- 不开启 Hedge 双开。
- 不开启 same-side add。
- 不做 Portfolio Risk。
- 不把全部 Risk 逻辑塞入 Position Policy。
