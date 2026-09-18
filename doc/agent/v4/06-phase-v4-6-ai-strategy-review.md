# Phase V4-6：AI Strategy Review

## 目标

让 AI 真正利用 V4 的测试结果辅助策略迭代，而不是只根据单次回测结果继续“猜参数”。

## 输入

AI Review 只读取结构化结果：

- Strategy Template 内容。
- 被比较 Strategy Template。
- Batch Backtest 聚合。
- Robustness 指标。
- Time Validation 结果。
- Shadow 结果。
- Testnet 结果（如果存在）。

## 输出

统一输出简单结构：

- 当前策略的主要优点。
- 当前策略的主要问题。
- 与上一版本相比发生了什么变化。
- 是否出现交易频率明显下降。
- 哪些 Symbol / 时间窗口表现最差。
- 下一轮只建议验证 1～3 个核心假设。
- 每个假设说明预期改善的指标。

例如：

```text
假设：当前趋势过滤过严。
证据：BTC/ETH Return 提升，但 Trades/Week 比上一版下降 52%，AVAX/UNI 基本无交易。
下一步：只放宽趋势过滤，不同时修改 TP/SL。
```

## 约束

- AI 不计算最终财务指标，只读取确定性结果。
- AI 不自动修改 Strategy Template。
- AI 不自动创建大量参数组合。
- AI 不自动启动真实交易。
- AI 必须区分事实指标和建议。

## UI

优先作为现有策略比较或回测详情中的“AI 分析”操作，不新建 Agent Studio。

## 验收 Gate

- AI 输入包含明确 Strategy Snapshot 和测试范围。
- 输出能引用具体指标，不只输出泛泛建议。
- 缺少 Shadow/Testnet 时明确说明缺失，不补造数据。
- Review 结果可保存到现有 Task/Conversation，便于后续追踪。

## 本阶段不做

- 不做自主策略生成循环。
- 不做 AI 自动提交代码。
- 不做 AI 自动部署策略。
