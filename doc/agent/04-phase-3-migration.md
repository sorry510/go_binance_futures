# 04. Phase 3：迁移已有 AI 能力

## 目标

把已有 `market condition` 与 `strategy builder` 接入统一 Runtime，验证新架构能承载真实业务，同时保持现有 HTTP API 和前端契约。

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
### 3A 当前实施状态（已完成）

- `service/market/regime.go` 负责 Snapshot Builder、算法 fallback 和 MarketCondition 持久化。
- `agent/skills/marketregime` 只负责 LLM 分类与最终结果 Validator，不开放 Tool。
- `feature.UpdateMarketCondition*` 保持兼容入口和原有互斥锁/Task Store。
- 自动模式默认走统一 Runtime；Runtime 初始化、LLM、Decision 或 Validator 失败时回退本地算法。
- Market Regime 不再保留独立/旧 LLM 调用路径，所有 LLM 分类统一通过 Agent Runtime。
- 不增加 `[agent]` feature flag；`llm.LoadConfig()` 成功即自动启用 LLM 分类。
- 未配置 LLM，或 Runtime/LLM/Decision/Validator 失败时，直接使用确定性算法 fallback。

### Market Regime 验收

- 手动模式不调用 LLM。
- 自动模式下 Runtime 成功时结果 Schema 与当前 API 兼容。
- LLM/Tool/Validator 失败时能明确 fallback，且仍可更新 MarketCondition。
- 同一时间只允许一个市场环境更新任务，保留现有互斥语义。
- 旧 `/update-market-condition` 相关前端流程无需立即修改。

## 3B. Strategy Builder 迁移

### 当前实施状态（已完成）

- `agent/skills/strategybuilder` 持有 System Prompt、用户输入构造、历史 Repair 压缩、动态必需 Tool 判断和最终 Validator 适配。
- 最终响应统一使用 Runtime 协议 `{"action":"final","summary":"...","result":{...}}`，不再维护 Strategy Builder 私有 decision parser。
- `get_features` 已加入通用只读 Tool Registry，和 `get_test_strategy_results` 一样直接调用 Domain Service。
- Strategy Builder 开放 `get_market_condition`，读取 Phase 3A Market Regime 最新持久化到 `systemConfig.MarketCondition` 的结果，不在 Tool 内重复触发市场分析。
- Runtime 新增动态 RequiredTools：用户明确要求查询测试结果或合约数据时，对应 Tool 必须成功调用后才能 final；要求基于市场趋势/当前行情时，`get_market_condition` 也必须在本轮成功调用。
- 市场趋势型策略启用确定性语义校验：所有启用的 `long`/`short` 开仓规则必须显式引用运行时字符串变量 `MarketCondition`，整体覆盖 `"1"` 至 `"11"`；相似行情允许在同一规则中用 `&&`/`||` 分组，避免 11 个近似规则一比一展开。
- 复杂策略表达式增加可读性校验：短条件允许单行；长表达式或包含大量 `&&`/`||` 时必须使用多行 `let` 局部变量拆解行情、趋势、动量或结构条件，最后组合成布尔结果。
- AI 平仓规则增加质量校验：`close_long`/`close_short` 不能只依赖 ROI、持仓盈亏或时间阈值，必须结合指标、K 线结构或市场趋势确认；外层盈亏阈值只作为触发前置条件。
- Strategy Builder System Prompt 已压缩为协议、Schema、字段映射和关键策略规则，移除逐指标长篇解释，减少上下文占用。
- Runtime 新增 MessageHook、ValidationHook 和 Round 事件；旧 HTTP Task API 通过适配这些事件继续返回进度、失败候选 JSON 和 validationError。
- LLM retry、截断响应修复、Tool Result、协议 Repair、最终 Validator Repair、MaxRounds 全部由统一 Runtime 处理。
- `controllers/strategy_template_ai_task.go` 只保留 HTTP/Task Store/续聊兼容、Runtime 组装和导入状态管理，不再直接调用 `Client.Generate`。
- `controllers/strategy_template_ai_tools.go` 与旧 Strategy Agent contract test 已删除。
- “AI 生成成功”和“导入数据库”仍是两个授权边界；Runtime final 不会直接写入 `strategy_templates`。
- 现有 `conversationId` 继续兼容旧 taskId 语义并保存完整 Runtime transcript；独立 Conversation Store 按原计划在 Phase 6 实现。

### Strategy Builder 验收

- 多轮 Tool/Final/Repair/Retry 使用统一 Runner。
- 用户要求的必需 Tool 在 final 前必须成功执行。
- 策略 JSON 继续执行 framework schema、indicator 参数和 expr 编译/运行校验。
- 校验失败时前端仍能取得候选 JSON 与 `validationError`。
- 续聊保留前一轮 assistant、Tool Result、AGENT_FEEDBACK 和 import error 上下文。
- 原 `/strategy-templates/ai-generate*` HTTP 契约和独立 import 动作保持兼容。

## Phase 3 验证方式

不再维护运行时双轨开关。迁移完成的 Skill 只保留统一 Runtime 路径，不维护第二套 LLM 主循环。Market Regime 由确定性算法 fallback 保证可用性；Strategy Builder 生成失败则明确结束任务，不写数据库。

Market Regime 的验证重点是分类/置信度、fallback 命中率、耗时和 token 使用；Strategy Builder 迁移后验证 JSON 校验通过率、轮数、Tool 次数、耗时和 token 使用。

## Phase 3 验收

- 两项现有 AI 功能都使用同一 Runner。
- 不再各自维护 LLM/Tool/Retry 主循环。
- 旧 HTTP API 对前端保持兼容。
- 已迁移能力不再保留独立 LLM 主循环；非 AI fallback 属于业务层确定性兜底。
- 重复的旧 Agent Loop 与 Strategy Builder 私有 Tool executor 已删除。

## 删除旧代码的条件

满足以下条件后再清理：

1. 新路径稳定运行一个完整观察周期。
2. 没有未解决的输出契约差异。
3. Runtime 的错误率、平均耗时和 LLM 成本可观测。
4. 前端不依赖旧 Task Store 的私有字段。
5. 回滚只需要切配置，不需要重新部署旧代码。