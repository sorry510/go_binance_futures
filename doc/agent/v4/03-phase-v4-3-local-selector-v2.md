# Phase V4-3：Local Coin Selector V2

## 1. 目标

替换目前主要依赖随机抽样、24h 涨跌幅排序和成交额 Top N 的本地选币策略，让自动交易候选选择变成：

**确定性、可解释、多因子、只使用本地数据、不新增 Binance REST 压力。**

## 2. 当前问题

现有 `TradeCoin1～6` 主要是：

- 最近交易 cooldown。
- Enable 过滤。
- 24h PercentChange 排序。
- QuoteVolume Top 200。
- 在候选中随机抽 2～5 个。

这会导致：

- 同一时刻重复运行结果不稳定。
- 随机性掩盖候选质量。
- 没有充分利用已经存在的本地行情指标。
- 选币逻辑与 `scanner.PrefilterTop30` 重复演进。

## 3. 复用 Scanner

V4-3 不再新写一套评分器。

以已有 `scanner.PrefilterTop30` 为基础统一 Candidate Service，继续使用本地 `symbols` / WS 数据：

- QuoteVolume 24h。
- TradeCount 24h。
- PercentChange 24h。
- 24h High / Low / Open / Close。
- Center Offset。
- 上涨方向：Upper Wick Ratio / Retrace From High。
- 下跌方向：Lower Wick Ratio / Rebound From Low。
- LastClose local momentum（方向对称）。
- freshness。

## 4. 两阶段 Selector

### Stage A：全市场本地预筛

先做硬过滤：

- enabled。
- USDT perpetual / 支持的 futures type。
- 数据 freshness。
- 最低 QuoteVolume：5,000,000 USDT / 24h。
- 最近交易 cooldown。
- 异常极端涨跌过滤。

### Stage B：确定性排序

对通过候选计算综合 score / reason，按分数稳定排序。

在已有 Prefilter 基础上，`smart_local_v2` 通过 opt-in 选项启用额外候选质量因子：

- 流动性分位。
- TradeCount 活跃度。
- 24h range / volatility quality。
- 极端涨跌变化与方向性反转风险惩罚（上涨看长上影/冲高回落，下跌看长下影/低位反弹）。
- 本地短周期 momentum（上涨/下跌方向对称处理，只在已有本地数据足够时使用）。

`smart_local_v2` 同分时使用 QuoteVolume、Symbol 做稳定 tie-break，不再随机。Generic Scanner 默认不启用 V4-3 的 TradeCount 加分和 Symbol 末级 tie-break，因此不会改变既有 market_scan / Opportunity / scan_symbols 的评分与同分顺序。

## 5. 与 Auto Strategy 的整合

新增一个明确策略，例如：

```text
smart_local_v2
```

真实交易 `StartTrade()` 与测试交易 `NoticeAllSymbolByStrategy()` 现在共用同一个 `selectConfiguredCoins(...)` 入口。`smart_local_v2` 使用同一个 Candidate Service 构建 **Top60 候选池**，再由进程内 Round-Robin 每轮取 5 个交给现有 Line Strategy 判断 long/short。真实与测试分别维护独立 Round-Robin 状态，互不推进游标；只要某个 Symbol 持续留在 Top60，就会持续获得轮询机会。

职责保持：

```text
Coin Selector → 选“看哪些币”
Line Strategy → 判断“是否开 LONG/SHORT”
```

不能把两层混成一个策略。

## 6. API 原则

V4-3 **不得为了选币对每个 Symbol 调 REST**。

数据来源优先：

```text
Futures WS
→ local symbols
→ 已有本地行情表
```

如果某指标本地缺失，就标记 missing 或降低评分，不允许扫描几百个币时逐币 REST 补数据。

## 7. UI / Debug

增加轻量“候选解释”能力：

```text
Rank
Symbol
Score
QuoteVolume
24h Change
Momentum
Reasons
Risks
Excluded Reason
Top60 Pool Size
Round-Robin Batch Size
Next Batch
```

可以复用现有 Scanner/Opportunity 调试入口，不新建大型策略实验平台。

## 8. Compatibility

- 旧 `TradeCoin1～6` 保留，避免现有配置失效。
- 新 V2 作为新的可选 selector。
- 实际测试确认后再决定是否作为推荐默认。

## 9. Gate

- 同一份本地数据下 Top60 候选池完全一致；每轮 Batch 由 Round-Robin 状态决定，同一调用序列下可预测且确定。
- 不出现随机抽样。
- disabled / stale / low-liquidity Symbol 不进入候选。
- Selector 本身不产生 Binance REST 请求。
- Top60 候选池有明确 reason/risk。
- 每轮只返回 5 个给 Line Strategy；稳定池下 12 轮覆盖全部 60 个。
- Preview 只能查看下一批，不能推进真实或测试 Round-Robin 状态。
- 真实/测试交易共用同一评分、Top60 与 Batch=5 算法；测试始终跟随当前 `FutureStrategyCoin`，不再按 ID 顺序轮询所有 Enable 币。若配置旧 `coin1～6`，测试也会继承对应旧 selector 的随机/窄覆盖行为，这是“测试与真实一致”的兼容语义。
- StartTrade / TestTrade 都继续只把结果交给 Line Strategy，不绕过策略开仓条件。

## 10. 本阶段不做

- 不使用 LLM 选币。
- 不做 ML ranking。
- 不做回测优化。
- 不新增大规模历史指标计算。

## 11. 实现结果

**V4-3 已完成（2026-09-23）。**

新增 selector：`smart_local_v2`。旧 `coin1～6` 完整保留，不改变现有配置语义；只有显式选择 `smart_local_v2` 才启用新 Candidate Service。V4-3 的 TradeCount 加分和 Symbol tie-break 也只由该 selector opt-in，不改变现有 Generic Scanner/Opportunity 的评分行为。

实现、默认阈值、Preview API、自动测试与人工测试步骤见 [v4-3-implementation-report.md](./v4-3-implementation-report.md)。当前 Pool Size=60、Batch Size=5、Cooldown=5m、Freshness=30s、Min QuoteVolume=5M USDT 为代码固定阈值；24h Change 对上涨/下跌对称处理。共享 Generic Scanner 的默认上限仍为 Top30，只有 `smart_local_v2` 显式放宽到 60。
