# Phase V4-1：Batch Backtest

## 目标

把当前“选择一个币 → 开始回测”扩展为轻量批量回测，使同一个策略可以一次测试多个 Symbol。

这是 V4 最先实现的核心能力。

## 功能

- 在现有回测页面支持选择多个 Symbol 创建一组 Backtest Run。
- 继续复用现有单 Run Engine，不新建第二套批量回测引擎。
- 每个 Symbol 仍产生独立 Backtest Run，结果格式完全兼容现有详情页。
- 增加轻量 `batch_id` 或等价关联，用于把同一次批量测试的 Run 聚合起来。
- 支持查看批量进度：总数、运行中、成功、失败、取消。
- 单个 Symbol 失败不影响其它 Symbol。
- 支持整批取消，但已完成 Run 保留。
- 优先复用 Historical Market Repository 和当前 Memory LRU。

## UI

现有“历史回测”页面增加批量入口即可，不新建独立大型页面。

常用场景：

```text
Strategy: Wyckoff v3
Symbols: BTC / ETH / SOL / BNB / AVAX / UNI
Range: 2024-01-01 ~ 2026-01-01
→ 一次创建
```

现有“快捷测试最近 10 个币”继续保留。

## 聚合结果

批量列表只展示必要指标：

- Symbol
- Return / Net PnL
- Max Drawdown
- Profit Factor
- Trade Count
- LONG / SHORT Trade Count
- 状态

不计算综合评分。

## 验收 Gate

- 批量创建 N 个 Symbol 时产生 N 个正常 Backtest Run。
- 单个 Run 的结果必须与单独执行该 Symbol 时一致。
- 一个 Symbol 失败不会终止整批。
- 支持取消尚未完成的任务。
- 同时测试约 5～10 个币不会破坏现有 Historical Market 缓存和内存边界。

## 本阶段不做

- 不做参数网格搜索。
- 不自动生成 Strategy Template。
- 不自动挑选“最佳策略”。
- 不做分布式 Worker。
