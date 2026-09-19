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
- Upper Wick Ratio。
- Retrace From High。
- LastClose local momentum。
- freshness。

## 4. 两阶段 Selector

### Stage A：全市场本地预筛

先做硬过滤：

- enabled。
- USDT perpetual / 支持的 futures type。
- 数据 freshness。
- 最低 QuoteVolume。
- 最近交易 cooldown。
- 异常极端涨跌过滤。

### Stage B：确定性排序

对通过候选计算综合 score / reason，按分数稳定排序。

可在已有 Prefilter 基础上补充：

- 流动性分位。
- TradeCount 活跃度。
- 24h range / volatility quality。
- 追高风险惩罚。
- 本地短周期 momentum（只在已有本地数据足够时使用）。

同分时使用 QuoteVolume、Symbol 做稳定 tie-break，不再随机。

## 5. 与 Auto Strategy 的整合

新增一个明确策略，例如：

```text
smart_local_v2
```

`StartTrade()` 使用 Candidate Service 返回 Top K，再交给现有 Line Strategy 判断 long/short。

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
```

可以复用现有 Scanner/Opportunity 调试入口，不新建大型策略实验平台。

## 8. Compatibility

- 旧 `TradeCoin1～6` 保留，避免现有配置失效。
- 新 V2 作为新的可选 selector。
- 实际测试确认后再决定是否作为推荐默认。

## 9. Gate

- 相同本地数据重复执行结果完全一致。
- 不出现随机抽样。
- disabled / stale / low-liquidity Symbol 不进入候选。
- Selector 本身不产生 Binance REST 请求。
- Top K 有明确 reason/risk。
- StartTrade 继续只把结果交给 Line Strategy，不绕过策略开仓条件。

## 10. 本阶段不做

- 不使用 LLM 选币。
- 不做 ML ranking。
- 不做回测优化。
- 不新增大规模历史指标计算。
