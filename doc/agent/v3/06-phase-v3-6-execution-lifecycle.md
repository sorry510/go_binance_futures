# Phase V3-6：Execution / Position Lifecycle

## 目标

把 V2 的“受控 MARKET 开仓”升级为完整交易生命周期，确保真实仓位始终有明确的保护、状态和恢复能力。

## Execution Plan

一次真实交易至少包含：

- Entry Order
- Protective Stop Loss
- Take Profit Plan
- Position State
- Reconcile State

开仓成功但保护单未建立时，不能直接标记为普通成功状态。

## 核心能力

- MARKET / LIMIT Entry。
- Protective Stop 创建、查询、更新、撤销。
- Take Profit 支持单次或分批。
- 部分成交和剩余订单处理。
- reduceOnly 平仓。
- 主动减仓、加仓和平仓。
- 订单/仓位与 Binance 状态 Reconcile。
- 程序重启后恢复未完成 Execution。
## 安全状态

建议明确区分：`entry_pending`、`entry_partial`、`protecting`、`protected`、`closing`、`closed`、`reconcile_required`、`emergency`。

如果 Entry 已成交但 Stop 创建失败，进入 `emergency`，按固定策略重试保护或安全平仓，不能由 LLM 临场决定。

## 幂等

- 每个 Entry/Stop/TP 都有独立且可复现的 client order id。
- 网络超时先查询订单状态，不自动重复提交。
- Position Manager 以交易所状态为最终外部事实，本地状态可通过 Reconcile 修复。

## 验收 Gate

- Fake Broker 覆盖完整成交、部分成交、Stop 失败、TP 部分成交、网络超时和重启恢复。
- 开仓后不存在“无保护但显示成功”的状态。
- 重复 Execute/Recover 不会制造重复订单。
- 所有真实动作继续经过 Trade Risk + Portfolio Risk。

## 本阶段不做

- 不让 LLM 直接决定撤单、加仓或紧急平仓。
- 不做复杂网格/高频订单管理。
- 自动测试不调用生产 Binance 下单。
