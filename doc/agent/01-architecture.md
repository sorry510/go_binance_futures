# 01. 目标架构

## 1. 目标形态

统一入口不等于只有一个业务 Agent，而是所有业务能力共享同一个运行时：

```text
Web/API ─┐
Scheduler ├─> Agent Manager -> Agent Runtime -> Skill
Event Bus ┘                         |
                                  Tool Registry
                                      |
                     Domain Services / Binance / DB / Scanner / Notify
```

外部 MCP 保持独立协议入口，但和 Agent Tool 复用 Domain Service：

```text
Agent Tool ─┐
            ├─> Domain Service -> Binance / DB / Notify
MCP Tool ───┘
```

## 2. Agent Runtime 职责

Runtime 只管理通用生命周期，不包含具体交易策略知识：

- 创建/恢复一次 Run。
- 加载 Skill 定义。
- 构造系统提示词和初始上下文。
- 调用 `llm.Client`。
- 解析 `tool`、`final`、`error` 等 Decision。
- 通过 Tool Registry 执行工具。
- 将 Tool Result 追加到会话上下文。
- 控制最大轮数、超时、取消、重试和上下文大小。
- 调用 Validator 校验最终结果。
- 记录 Task/Run 状态、Usage、耗时与错误。

Runtime 不直接 import `controllers`，也不直接拼 SQL。
## 3. 核心接口草案

```go
type Skill interface {
    Name() string
    SystemPrompt() string
    Tools() []string
    MaxRounds() int
    BuildInput(ctx context.Context, req AgentRequest) ([]llm.Message, error)
    ValidateFinal(ctx context.Context, raw json.RawMessage) (any, error)
}

type Tool interface {
    Name() string
    Description() string
    ReadOnly() bool
    Execute(ctx context.Context, arguments json.RawMessage) (any, error)
}

type Runner interface {
    Run(ctx context.Context, req AgentRequest) (*AgentResult, error)
}
```

接口名称后续可以调整，但职责必须保持分离。

## 4. Agent Decision 协议

第一版继续沿用现有策略生成已经验证过的简单 JSON 协议，不急于绑定某一家 Provider 的原生 Tool Calling：

```json
{"action":"tool","summary":"需要补充 15m K线","tool":"get_klines","arguments":{"symbol":"ONGUSDT","interval":"15m","limit":100}}
```

最终结果：

```json
{"action":"final","summary":"分析完成","result":{}}
```

优势：OpenAI、Anthropic、OpenAI Compatible、Claude/Codex SDK 都可以走同一 Runtime。未来可在 LLM Adapter 内支持原生 Tool Calling，但 Runtime 上层协议不变。
## 5. 推荐目录

```text
agent/
├── runtime/        # Runner、loop、decision、context、retry
├── skill/          # Skill 接口与注册表
├── tools/          # Tool 接口、Registry、具体 Tool Adapter
├── task/           # Task/Run 状态与 Store 接口
├── conversation/   # 多轮用户会话；与单次 Task 分离
├── validator/      # Schema 与业务校验
├── permission/     # Tool 权限和危险操作边界
├── scheduler/      # 周期任务触发器
└── event/          # Event Bus 与事件类型
```

业务实现优先放在已有或新建的 Domain Service 中，不把业务代码塞进 `agent/tools`。Tool 只是 Agent 到 Domain Service 的适配器。

## 6. 四个首批 Skill

### market_regime

复用 `feature/market_condition.go` 的数据计算与算法兜底，负责市场环境分类。

### strategy_builder

迁移 `strategy_template_ai_task.go` 的策略生成业务知识、工具约束和最终 JSON 校验。

### symbol_analysis

新增能力。组合实时行情、Kline、Funding、OI、强平、Taker、Depth、市场环境，输出结构化 TradingPlan。

### alert_analysis

只消费 Signal Engine 已筛选出的异常信号。LLM 负责综合解释、分级和生成通知内容，不消费每个 WS Tick。

## 7. 明确不做

第一版不引入 Multi-Agent、向量数据库、复杂 DAG、Agent 自我创建、无限轮次和 LLM Tick 级调用；不让 Agent 默认直接下单。