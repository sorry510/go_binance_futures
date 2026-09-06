# Phase V3-5：Portfolio Risk Engine

## 目标

把 V2 的单 Proposal Risk 扩展为账户级组合风险，解决多个合法 Proposal 同时出现时的总风险失控问题。

## 核心检查

- Account Equity / Available Balance。
- Portfolio Total Risk Budget。
- Gross / Net Exposure。
- LONG / SHORT 方向暴露。
- 单 Symbol 最大暴露。
- 最大同时持仓数量。
- Daily Loss Limit。
- Account Drawdown Limit。
- 连续亏损后的自动暂停。
- 多个 Proposal 并发时的 Risk Reservation。

Portfolio Risk 仍是确定性 Go Service，不询问 LLM“是否安全”。
## Risk Reservation

当多个 Proposal 同时通过单笔 Risk 时，先为正在准备执行的 Proposal 预留风险额度，防止并发请求都按同一份可用额度计算。

示例：总 Risk Budget 为 20 USDT，已有 18 USDT，新的 Proposal 需要 5 USDT，则直接 REJECT，并返回当前风险、申请风险和上限。

## 简化原则

- 单用户项目不做风险审批角色。
- 不做“管理员 override”权限体系。
- 普通 UI 不提供一键忽略 Portfolio Risk 的按钮。
- Risk Policy 可在 AI 配置中直接维护，并记录配置变更。

## 验收 Gate

- 并发 Proposal 不会超卖同一份 Risk Budget。
- Daily Loss / Drawdown / 连续亏损触发后，新 Proposal 被确定性阻止。
- 单 Proposal Risk PASS 但 Portfolio Risk FAIL 时不能进入执行。
- 风控失败返回可读的原因和当前数值，不只返回布尔值。
- Fake Account Fixture 覆盖余额不足、方向集中、Risk 超限和并发 Reservation。

## 本阶段不做

- 不做复杂 VaR/蒙特卡洛模型。
- 不做用户/角色级风险权限。
- 首版不实现基于统计相关性的动态风险权重；相关性暴露可后续按简单分组增强。
