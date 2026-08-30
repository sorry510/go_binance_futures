# 00. Phase 0 基线

本文记录 Agent 重构前的 AI 行为、入口和兼容边界。Phase 3 完成前，下列旧 API 与业务入口不得删除或改变响应契约。

## 1. 市场环境 AI 调用链

### 定时入口

```text
main.go
 -> loopRun(60m)
 -> feature.UpdateMarketCondition
 -> UpdateMarketConditionWithProgress
 -> service/market.LoadRegimeSymbols (symbols 表 USDT 合约)
 -> service/market.CalculateAlgorithmCondition (本地 fallback)
 -> service/market.BuildRegimeSnapshot
 -> market_regime Skill
 -> Agent Runtime / Final Validator
 -> config.market_condition 持久化
```

LLM 不可用或解析失败时继续使用本地加权算法结果。

### 手动入口

```text
POST /update-market-condition
 -> IndexController.UpdateMarketCondition
 -> startMarketConditionUpdateTask
 -> goroutine runMarketConditionUpdateTask
 -> feature.UpdateMarketConditionWithProgress

GET /update-market-condition/:taskId
 -> 返回任务进度/结果
```

## 2. 策略生成 Agent 调用链

```text
POST /strategy-templates/ai-generate
 -> StrategyTemplateController.StartAIGeneration
 -> startStrategyTemplateAITask
 -> goroutine runStrategyTemplateAITask
 -> llm.NewFromConfig
 -> 最多 10 轮 tool/final 决策
 -> executeStrategyTemplateAITool
    -> get_features / get_test_strategy_results
 -> validateGeneratedStrategyTemplateJSON
 -> succeeded / failed

GET /strategy-templates/ai-generate/:taskId
 -> 返回任务状态、进度、事件、JSON、错误

POST /strategy-templates/ai-generate/:taskId/import
 -> 再次校验并写入 strategy_templates
 -> 标记 Imported，结束该对话上下文
```

当前策略 Agent 已具备多轮、Tool Result、Repair、重试、最终 JSON 校验，是 Phase 1 Runtime 的主要行为参考。

## 3. 旧 Task Store 行为

- `marketConditionTaskStore`：内存 Map，单个 active task，完成任务保留 30 分钟。
- `strategyTemplateAITaskStore`：内存 Map，任务/续聊上下文保留 30 分钟。
- 服务重启后任务丢失属于当前既有行为；Phase 0-1 不修改为 DB。
- 旧 Task 状态和前端轮询接口保持原样，统一 Task Store 在 Phase 3 再接入旧入口。

## 4. Tool 权限基线

Agent Tool 统一分三档：

| 等级 | 含义 | 典型操作 | Phase 1 默认 |
| --- | --- | --- | --- |
| `read` | 只读行情/数据库查询 | Kline、Funding、OI、强平、symbols、测试结果 | 允许 |
| `write` | 修改业务配置但不直接交易 | 新建提醒、保存策略、更新监听配置 | 禁止 |
| `trade` | 影响真实账户资金或仓位 | 开仓、平仓、调杠杆、撤单 | 禁止 |

Phase 1 Runtime 默认使用 `read` 上限。未来 Skill 即使声明了 Tool，也必须同时通过全局 Permission Policy。

## 5. MCP 边界

现有 `mcpserver` 是外部协议入口，目前通过 HTTP 调用项目已有 API。Phase 0-1 不改 MCP 行为。

后续 Phase 2 的目标是把公共业务逻辑下沉到 Domain Service：

```text
Agent Tool -> Domain Service
MCP Tool   -> Domain Service
```

不让内部 Agent 通过 `127.0.0.1` MCP/HTTP 绕一圈调用自己。

## 6. 回归测试基线

Phase 0 新增/确认的关键覆盖：

- 市场环境手动模式 Task progress/completion 和 30 分钟 cleanup。
- 市场环境 LLM JSON 输出契约及非法 condition/confidence/reason 拒绝。
- 本地市场环境 fallback 对强多/强空的分类契约。
- 策略 Agent `tool` / `final` Decision 协议。
- 策略 Tool 失败必须编码为 `TOOL_RESULT` 且 `ok=false`。
- 用户明确请求行情/测试结果时的必需 Tool 识别。
- 不完整策略 JSON 必须被最终 Validator 拒绝。
