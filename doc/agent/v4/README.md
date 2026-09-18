# AI Agent V4 开发计划（精简个人版）

## 1. V4 定位

V4 不继续扩展 Agent 平台能力，而是解决一个核心问题：**怎样判断一个新策略真的比旧策略更稳定，而不是只对某个币、某段历史过拟合。**

V4 基于已经完成的 Historical Backtest、Outcome Review、Controlled Trade、Ownership、Task、Observability 和 Scheduler，只补策略验证闭环，不重做已有系统。

最终闭环：

```text
新 Strategy Template
    ↓
批量回测
    ↓
稳健性检查
    ↓
Shadow 前向验证
    ↓
Testnet 验证
    ↓
用户决定是否用于真实交易
    ↓
结果复盘 → AI 辅助下一次策略改进
```

## 2. 明确不做

- 不做多 Agent 自主研究集群。
- 不做遗传算法、贝叶斯优化或大规模参数搜索。
- 不让 LLM 自动修改正式策略。
- 不让 AI 自动批准或进入真实交易。
- 不做复杂 Strategy Registry / Candidate / Approval 平台。
- 不做机构级 Portfolio、VaR、风险预算。
- 不新建第二套回测、复盘、任务或监控系统。
- 不为了 V4 引入向量数据库、知识图谱或新的基础设施。

## 3. V4 核心原则

- 旧策略不原地修改；新想法继续创建新的 Strategy Template。
- 所有指标由 Go/SQL 确定性计算，LLM 只负责解释和提出下一步研究建议。
- 优先复用现有 Backtest Run、Trade、Equity、Outcome Review 和 Task。
- 回测结果不能只看收益，至少同时看回撤、交易次数、Profit Factor 和跨区间稳定性。
- Shadow/Testnet 只验证候选策略，不改变现有真实交易安全边界。
- 个人项目保持人工最终决策，不增加审批流。

## 4. Phase

| Phase | 目标 |
| --- | --- |
| [V4-0](./00-phase-v4-0-baseline.md) | 冻结 V3/V4 起点和回归基线 |
| [V4-1](./01-phase-v4-1-batch-backtest.md) | 一次对多个币执行同一策略回测 |
| [V4-2](./02-phase-v4-2-robustness.md) | 自动计算最必要的稳健性指标 |
| [V4-3](./03-phase-v4-3-time-validation.md) | 多时间窗口验证，降低单一区间过拟合 |
| [V4-4](./04-phase-v4-4-shadow-trading.md) | 实时 Shadow Trading，不下真实订单 |
| [V4-5](./05-phase-v4-5-testnet-validation.md) | 候选策略 Testnet 前向验证 |
| [V4-6](./06-phase-v4-6-ai-strategy-review.md) | AI 基于真实测试结果辅助策略迭代 |
| [V4-7](./07-phase-v4-7-result-review.md) | 统一比较 Backtest / Shadow / Testnet / Live |
| [V4-8](./08-phase-v4-8-finalization.md) | V4 收尾、文档、测试和长期运行检查 |

## 5. V4 最重要的输出

V4 完成后，对一个 Strategy Template，系统应能回答：

1. 在最近测试的多个币上是否稳定。
2. 是否只在某一个时间段表现好。
3. 交易频率是否因为不断优化而明显下降。
4. 收益是否过度依赖少数几笔交易。
5. Shadow/Testnet 与历史回测是否出现明显偏差。
6. AI 下一次建议修改什么，以及为什么。

不计算“综合评分”，避免用一个数字掩盖不同指标的 trade-off。

## 6. 开发顺序

```text
V4-0 Baseline
   ↓
V4-1 Batch Backtest
   ↓
V4-2 Robustness
   ↓
V4-3 Time Validation
   ↓
V4-4 Shadow Trading
   ↓
V4-5 Testnet Validation
   ↓
V4-6 AI Strategy Review
   ↓
V4-7 Result Review
   ↓
V4-8 Finalization
```

V4-1～V4-3 是第一优先级。只有离线验证稳定后，再进入 V4-4/V4-5。

## 7. Definition of Done

- 同一策略可以方便地批量测试多个 Symbol。
- 系统能自动指出交易频率下降、跨币种不稳定、跨时间窗口不稳定和收益过度集中。
- 候选策略可以在不下单的情况下进行实时 Shadow 验证。
- Testnet 可以复用现有真实执行路径验证前向表现。
- AI 可以读取结构化测试结果，给出有依据的改进建议，但不能自动修改策略或自动实盘。
- Backtest、Shadow、Testnet、Live 的结果可以分开查看并进行简单比较。
- V4 不破坏现有 V3 Backtest、Ownership、受控交易和运维能力。
