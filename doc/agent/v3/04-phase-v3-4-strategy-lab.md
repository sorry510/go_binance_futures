# Phase V3-4：Strategy Lab

## 目标

把 Strategy Builder、Strategy Experiment、Backtest、模拟盘和正式策略统一成清晰的策略版本生命周期。

## 生命周期

```text
Candidate
   ↓
Validated
   ↓
Backtested
   ↓
Paper
   ↓
Active
   ↓
Retired
```

这是单用户项目，不建设审批人/审批流。进入 `Active` 只要求一次显式 Promote 操作，并记录版本与变更信息。

## 核心能力

- AI 或用户修改策略时永远产生新的 Candidate Version，不直接覆盖 Active。
- Candidate 先做表达式/Schema 校验，再进入 Backtest。
- Backtest 达标后可进入 Paper Trading。
- 用户可显式 Promote 某版本为 Active。
- Active Strategy 支持一键回滚到已验证的历史版本。
## Promote 条件

首版只做简单、可解释的门槛，例如：

- 回测完成且无数据完整性错误。
- Max Drawdown、Trade Count 等达到用户配置的最低要求。
- Paper 阶段达到最小样本数时显示结果，但不自动替用户 Promote。

Promote 不是权限审批，只是防止实验版本误覆盖正式版本的显式确认。

## UI

- Strategy Lab 展示所有版本和当前状态。
- 可查看 Backtest、Paper、版本差异、创建来源和 Active/Retired 状态。
- 提供 Promote、Rollback、Retire。
- AI 可以生成“为什么建议 Promote/不 Promote”的说明，但按钮行为由用户决定。

## 验收 Gate

- AI 修改 Active Strategy 时只能创建新 Candidate。
- Promote 后所有新任务能明确记录使用的 Strategy Version。
- Rollback 不删除历史版本，并能恢复到指定稳定版本。
- Strategy Experiment 和正式 Backtest 的结果类型明确区分。

## 本阶段不做

- 不做多人审批、审批角色、审批队列。
- 不做 AI 自动 Promote。
- 不做复杂策略商店或共享市场。
