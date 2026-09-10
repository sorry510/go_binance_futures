# AI Agent V3 开发计划（个人版）

## 1. 项目定位

V3 面向 **单用户、个人自用的 Binance 合约辅助交易系统**。目标不是建设大型量化平台，而是让已有 Agent、行情、回测和受控交易能力变得更可靠、更容易验证、更适合长期自己使用。

V2 已经完成 Runtime、Task/Conversation、Tool Runtime、MCP、Portable Skill、Model Gateway、Memory、Observability、Workflow 和受控交易；V3-1～V3-3 又完成 Multi-Agent、Market Intelligence 和正式 Historical Backtest。因此后续阶段只补真正缺失的交易闭环，不再重复建设平台能力。

### 明确不做

- 不做企业级 RBAC、审批流、组织/租户、多用户协作。
- 不做 Strategy Version / Candidate / Promote / Retired 生命周期。
- 不做机构级 Portfolio Optimizer、VaR、相关性矩阵或复杂 Risk Reservation。
- 不做 Agent Studio / Developer Portal；继续使用现有 Skill、MCP、Workflow 管理页面。
- 不让 LLM 自动修改正式策略、自动突破 Risk、自动批准真实订单。

## 2. 策略使用约定

本项目采用非常简单的策略管理原则：**已有策略不原地修改，新想法直接创建新的 Strategy Template。**

```text
策略 A（保留不变）
   ↓ 新想法
新建策略 B
   ↓
Backtest / 模拟盘
   ↓
用户决定哪个策略绑定到 Symbol
```

V3-3 的 Backtest Run 已保存完整 `technology_json`、`strategy_json` 和策略指纹；模拟盘结果也保存完整 Technology / Strategy Snapshot，因此历史结果本身已经可复现，不再额外维护策略版本系统。

## 3. 已完成基线

| Phase | 状态 | 已有能力 |
| --- | --- | --- |
| [V3-0](./00-phase-v3-0-baseline.md) | ✅ | 冻结 V2 行为、Benchmark、Replay/Eval 基线 |
| [V3-1](./01-phase-v3-1-multi-agent.md) | ✅ | Multi-Agent Team：Technical / Flow / News / Supervisor 协作 |
| [V3-2](./02-phase-v3-2-market-intelligence.md) | ✅ | Market Intelligence：统一 Event/Fact、公告/新闻/资金面来源 |
| [V3-3](./03-phase-v3-3-backtest.md) | ✅ | Historical Market Repository + 确定性历史回测 |

> 原 V3-4 Strategy Lab 在规划复审后取消：个人使用方式是“新策略新建模板”，而 Backtest / Paper 已保存完整策略快照，额外版本生命周期属于重复抽象，因此删除该 Phase 文档并保留编号空缺。

## 4. 后续 Phase

| Phase | 优先级 | 目标 |
| --- | --- | --- |
| [V3-5](./05-phase-v3-5-trade-safety.md) | P0 | 真实交易安全：先统一仓位/订单 Ownership，确保“谁创建、谁管理”，再补 Agent Stop、整仓平仓与重启恢复 |
| [V3-6](./06-phase-v3-6-opportunity-watch.md) | P1 | Opportunity Watch：自动发现和分析机会，但真实执行继续由用户确认 |
| [V3-7](./07-phase-v3-7-outcome-review.md) | P1 | 交易复盘与策略比较：统一查看 Backtest、模拟盘和真实交易表现 |
| [V3-8](./08-phase-v3-8-personal-operations.md) | P2 | 个人运维与 V3 收尾：健康检查、数据增长控制、备份说明和最终清理 |

V3 到 V3-8 结束，不再规划 Agent Studio。

## 5. 新的交易闭环

```text
Market Data / Event / News
          ↓
Market Intelligence / Scanner
          ↓
Single Agent / Multi-Agent / Workflow
          ↓
Opportunity / Trading Plan
          ↓
V2 Controlled Trade Proposal
          ↓
现有 Deterministic Risk
          ↓
用户确认
          ↓
V3-5 Ownership-Safe Execution + Managed Position
          ↓
Binance
          ↓
V3-7 Outcome Review
```

自动化重点放在“发现、分析、通知和准备 Proposal”，而不是自动替用户批准真实交易。

## 6. 开发顺序

```text
V3-3 Historical Backtest ✅
        ↓
V3-5 Trade Ownership & Safety
        ↓
V3-6 Opportunity Watch
        ↓
V3-7 Outcome Review
        ↓
V3-8 Personal Operations / Finalization
```

前一 Phase 未通过 Gate，不进入下一 Phase。V3-5 未完成前，不扩大真实交易自动化范围。

## 7. 统一设计原则

- 优先复用现有代码和表；能扩展现有 Service 就不新增第二套系统。
- LLM 做分析、解释和候选建议；风险、下单、仓位、保护和平仓由确定性 Go 逻辑完成。
- 系统只管理自己创建的真实仓位；用户手工仓位、其他机器人仓位一律视为外部资产，不主动修改。
- 新策略永远创建新的 Strategy Template，不原地覆盖旧模板。
- Risk 配置保持少而明确，失败时返回具体数值和原因。
- 自动测试只使用 Fake Broker / Replay / Testnet，不调用生产账户下单。
- 不为了“架构完整”增加用户实际不会使用的状态、表、审批或页面。

## 8. V3 Definition of Done

- Multi-Agent 和 Market Intelligence 可以稳定给出可追踪的分析结果。
- Historical Backtest 可以使用统一历史行情仓库复现策略表现。
- 真实交易只管理本系统自己的订单/仓位，并具备 Stop、可选 TP、确定性平仓和重启恢复能力。
- 自动发现机会后可以通知用户并快速进入现有受控交易流程，但不会绕过用户确认。
- Backtest、模拟盘和真实交易可以按策略/Symbol/行情环境做实用复盘。
- 系统长期运行时可以快速检查数据源、Agent、Scheduler、DB 和交易链路健康状态，并控制大表增长。
