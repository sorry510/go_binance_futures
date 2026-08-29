# 04. Phase 3：迁移已有 AI 能力

## 目标

把已有 `market condition` 与 `strategy builder` 接入统一 Runtime，验证新架构能承载真实业务；迁移期间保持旧 API 和旧行为可回退。

## 迁移原则

采用 Strangler Pattern：先在旁边建立新路径，验证一致后再切入口，不直接删除旧代码。

```text
旧 Controller/API ──┬─> legacy implementation
                    └─> new Agent Runtime（灰度/影子）
```

稳定后再变为：

```text
旧 Controller/API -> Agent Runtime -> Skill
```

Controller 的 HTTP 契约尽量保持不变，前端无需和后端重构同步发生。

## 3A. Market Regime 迁移

现有 `feature/market_condition.go` 不应整体塞入 Skill。建议先拆成三部分：

1. Market Snapshot Builder：收集/计算确定性市场统计。
2. Algorithm Fallback：保留现有本地算法。
3. MarketRegime Skill：只负责 LLM 分类和最终结果校验。

LLM 不可用或 Runtime 失败时仍调用 Algorithm Fallback，保证市场环境更新不是单点依赖 AI。
### Market Regime 验收

- 手动模式不调用 LLM。
- 自动模式下 Runtime 成功时结果 Schema 与当前 API 兼容。
- LLM/Tool/Validator 失败时能明确 fallback，且仍可更新 MarketCondition。
- 同一时间只允许一个市场环境更新任务，保留现有互斥语义。
- 旧 `/update-market-condition` 相关前端流程无需立即修改。

## 3B. Strategy Builder 迁移

这是验证 Runtime 的关键业务，因为现有实现已经覆盖多轮、Tool、Repair、Progress 和续聊。

迁移时按职责拆分：

- System Prompt/策略契约 -> `strategy_builder` Skill。
- `requiredStrategyTemplateAITools` -> Skill 的动态 Tool Requirement/Policy。
- `executeStrategyTemplateAITool` -> 通用 Tool Registry。
- `parseStrategyTemplateAIAgentDecision` -> Runtime Decision Parser。
- LLM retry/round loop -> Runtime。
- `validateGeneratedStrategyTemplateJSON` -> Skill Validator。
- Task progress/event -> 通用 Task Manager。
- 策略导入动作仍保留在业务 Controller/Service，不自动并入 AI final。

重要：迁移后仍保留“生成”和“导入”为两个授权边界，Agent 成功生成 JSON 不等于写数据库。
## Phase 3 双轨验证

建议为两个 Skill 增加 feature flag，例如：

```text
agent.runtime_enable
agent.market_regime_enable
agent.strategy_builder_enable
```

灰度顺序：开发环境 -> 手动触发 -> 影子比较 -> 少量真实请求 -> 全量切换。

影子比较只记录差异，不同时执行两个会产生写副作用的路径。Market Regime 可比较分类/置信度；Strategy Builder 可比较 JSON 是否通过校验、轮数、Tool 次数、耗时和 token 使用。

## Phase 3 验收

- 两项现有 AI 功能都使用同一 Runner。
- 不再各自维护 LLM/Tool/Retry 主循环。
- 旧 HTTP API 对前端保持兼容。
- 新 Runtime 关闭时有明确 legacy/fallback 路径。
- 关键行为回归测试通过后，才允许删除重复的旧 Agent Loop 代码。

## 删除旧代码的条件

满足以下条件后再清理：

1. 新路径稳定运行一个完整观察周期。
2. 没有未解决的输出契约差异。
3. Runtime 的错误率、平均耗时和 LLM 成本可观测。
4. 前端不依赖旧 Task Store 的私有字段。
5. 回滚只需要切配置，不需要重新部署旧代码。