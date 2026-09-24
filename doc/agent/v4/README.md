# AI Agent V4 开发计划：Chat Workspace / Skill Studio / Smart Selector / API Budget

## 1. V4 定位

V4 不再做回测、仓位生命周期或新的多 Agent 基础设施。V1～V3 已经建立 Runtime、Task/Conversation、Tool、MCP、Portable Skill、Model Gateway、Memory、Workflow、Market Intelligence、Opportunity、受控交易与系统看板。

V4 聚焦四个实际缺口：

1. AI 聊天体验仍然偏“单 Skill / 固定模型”，需要支持一个对话挂载多个 Skill、删除聊天、随时切换模型。
2. Portable Skill 虽然已经支持标准 Agent Skills 导入，但缺少 Web 端原生编写和版本发布工作流。
3. 本地选币仍然大量依赖随机、24h 涨跌幅和成交额排序，没有充分利用已有本地行情与 scanner 能力。
4. Binance REST 调用目前缺少统一的权重观测、预算和请求协调；多币同时开单时存在重复账户查询和高权重 API 放大问题。

V4 的原则是：**先复用现有能力，再补用户真正能感知的入口和系统级约束。**

Skill Studio 的发布/激活属于管理员级 Agent 配置变更，因为 `SKILL.md` 会进入 Agent System Prompt。当前项目是单用户自用部署；如果未来引入多用户，必须为 Skill Publish / Activate 增加管理员角色权限。

## 2. 当前已经存在、V4 不重复建设的能力

- Chat Conversation 持久化、历史上下文、标题。
- 后端已经支持 Chat Conversation 删除；V4-1 主要补齐前端和一致性行为。
- Native / Portable Skill、Chat-enabled Skill。
- Agent Skills 标准 `SKILL.md` 解析、校验、ZIP/SKILL.md 导入、版本、激活与 Permission。
- Model Gateway、多 Provider、Fallback、健康状态。
- MCP Client、远端 Tool/Resource/Prompt。
- Scanner 本地预筛选：已有成交额、24h 变化、中枢偏离、上影、冲高回落、本地 momentum 等确定性评分。
- Futures User Data WS、本地 position/order 镜像。
- Historical REST 已纳入全局 Binance API Budget 的 P3 低优先级调度，并保留历史数据获取的独立安全约束。

## 3. 新 V4 Phase

| Phase | 优先级 | 目标 |
| --- | --- | --- |
| [V4-0](./00-phase-v4-0-baseline.md) | P0 | 冻结当前 Chat / Skill / Selector / Binance API 行为基线 |
| [V4-1](./01-phase-v4-1-chat-workspace.md) | P0 ✅ | Chat 支持多个 Skill、删除聊天、对话内切换模型 |
| [V4-2](./02-phase-v4-2-skill-studio.md) | P0 ✅ | Web 端创建、编辑、校验和发布标准 Agent Skills |
| [V4-3](./03-phase-v4-3-local-selector-v2.md) | P1 ✅ | 重构本地选币：确定性、多因子、复用 scanner，不新增 REST 压力 |
| [V4-4](./04-phase-v4-4-binance-api-observability.md) | P0 ✅ | 统一观测 Binance API endpoint、权重、延迟、429/418 和 Order Count |
| [V4-5](./05-phase-v4-5-binance-api-budget.md) | P0 ✅ | 全局 API Budget / 请求合并 / 快照复用 / 优先级调度，解决多单 API 超限 |
| [V4-6](./06-phase-v4-6-finalization.md) | P2 🚧 | V4 收尾、文档、性能和长期运行验证 |

## 4. 推荐开发顺序

```text
V4-0 Baseline
  ↓
V4-1 Chat Workspace ✅
  ↓
V4-2 Skill Studio ✅
  ↓
V4-3 Local Selector V2 ✅
  ↓
V4-4 Binance API Observability ✅
  ↓
V4-5 Binance API Budget & Optimization ✅
  ↓
V4-6 Finalization 🚧
```

V4-4 / V4-5 已完成 Binance API 观测、预算和请求优化；V4-6 Finalization 的代码、文档和自动化 Gate 已完成，长期运行观察仍待实际部署验证。

## 5. V4 明确不做

- 不做新的 Backtest 功能。
- 不做自动策略参数优化。
- 不做新的 Multi-Agent 架构。
- 不重写现有 Model Gateway。
- 不重写 MCP Client。
- Skill Web 编辑器不允许直接修改已激活版本文件；采用 Draft → Validate → Publish 新版本。
- Skill scripts 首版仍不执行。
- 本地选币不调用 LLM，不对每个 Symbol 新增 REST 请求。
- API 超限修复不使用简单的全局固定 sleep 作为主要方案。

## 6. V4 Definition of Done

- 一个 Chat Conversation 可挂载多个 Skill；支持 `Auto` 从已挂载 Skill 中选择适合当前消息的能力，也支持用户显式指定 Skill。
- Conversation 支持 Web 删除；正在运行的任务不能被不安全删除。
- 用户可在 Chat 中选择启用的模型，并在后续消息中切换；每个 Task 记录实际 provider/model。
- 用户可在 Web 创建、编辑、校验、预览和发布标准 `SKILL.md` Skill Package。
- 本地自动选币不再依赖随机抽样作为主策略，并能给出确定性候选和理由。
- 系统可以看到 Binance API 当前实际使用量、最重 endpoint、429/418、Order Count 和延迟。
- 多币同时开单时不会因为重复 Position/OpenOrder 等安全检查放大 REST 权重。
- 非关键后台请求会在 API Budget 紧张时主动降级/延后，真实交易关键请求优先。
