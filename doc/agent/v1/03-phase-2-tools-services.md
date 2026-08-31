# 03. Phase 2：Tool Registry 与共享 Service 层

## 目标

让 Agent、MCP、Controller 共享同一份领域能力，避免出现三套查询/写入实现。

## 当前问题

目前 Tool 能力主要有两种：

- `strategy_template_ai_tools.go` 直接查询 ORM。
- `mcpserver/api_tools.go` 通过 HTTP 调本项目 Controller。

这两种方式都不适合作为长期统一内核。建议逐步抽 Domain Service，并让不同入口做薄适配。

## Service 分层建议

第一批只抽 Agent/MCP 真实需要复用的服务：

```text
service/market      行情快照、Kline、市场环境读取
service/symbol      合约基础信息、本地行情信息
service/liquidation 强平查询/聚合
service/strategy    策略模板查询、生成结果校验、测试结果查询
service/alert       监听规则、报警记录
service/notify      统一通知发送入口
```

无需为了“架构漂亮”一次性重构全部 `feature/`。按 Tool 使用需求逐个抽取。
## Tool Registry 第一批能力

建议优先只开放只读 Tool：

```text
get_symbol_snapshot
get_klines
get_funding_rate
get_liquidations
get_market_condition
scan_symbols
get_test_strategy_results
get_strategy_template
```

第二批再加入低风险写 Tool：

```text
create_strategy_template
create_alert_rule
update_alert_rule
send_notification
```

交易 Tool 暂不注册给通用 Agent：

```text
open_long/open_short/close_position/set_leverage
```

未来如需要自动交易，应先经过独立 `TradeProposal -> Risk Engine -> Executor` 流程。

## Tool 元数据

每个 Tool 至少声明：名称、描述、输入 Schema、输出 Schema、RiskLevel、Timeout、MaxResultBytes、是否幂等。

RiskLevel 建议：`read`、`write`、`trade`。Skill 还需声明允许调用的 Tool 白名单，Runtime 同时检查 Skill 白名单和全局 Permission Policy。
## MCP 与 Agent 的最终关系

目标：

```text
Agent Tool Adapter -> Domain Service
MCP Tool Adapter   -> Domain Service
HTTP Controller    -> Domain Service
```

不建议：

```text
Agent -> MCP HTTP -> Controller -> Service
```

内部 Agent 走 Go 函数调用，减少认证、序列化、网络和错误转换层。

## Phase 2 实施顺序

1. 为现有策略 AI 的 `get_features` 和 `get_test_strategy_results` 抽共享查询 Service。
2. Agent Tool 适配到 Service，并保持原 Tool 返回结构兼容。
3. MCP 中相关只读 Tool 逐步改为调用 Service；外部 MCP 协议不变。
4. 再增加 Kline、Funding、Liquidation、MarketCondition Tool。
5. 给每个 Tool 增加输入校验、超时、结果大小限制和日志字段。
6. 最后才考虑写操作 Tool。

## 实施状态（2026-08-29）

Phase 2 已完成第一批只读能力落地：

- 新增 `service/symbol`、`service/market`、`service/liquidation`、`service/strategy`。
- `FeatureController.Get`、`TestStrategyResultController.Get`、`StrategyTemplateController.Get`、`FuturesLiquidationOrderController.Get` 已改为薄适配并复用 Service。
- 旧 `strategy_template_ai_tools.go` 的 `get_features` / `get_test_strategy_results` 已改为复用 Service，Tool 名称和返回结构保持兼容。
- 新增 8 个通用只读 Agent Tool：`get_symbol_snapshot`、`get_klines`、`get_funding_rate`、`get_liquidations`、`get_market_condition`、`scan_symbols`、`get_test_strategy_results`、`get_strategy_template`。
- Tool 元数据已包含输入/输出 Schema、RiskLevel、Timeout、MaxResultBytes、Idempotent。Runtime 会应用单 Tool 超时和结果大小限制，并记录 task/skill/tool/duration/status Event。
- MCP 的 `futures_symbols_list` 和 `futures_liquidation_orders_list` 已改为进程内调用 Service；外部 MCP 名称、输入和响应 envelope 不变，Authorization 仍要求存在。
- MCP 的通知/监听写操作仍保留原 HTTP 镜像路径；写 Tool 和交易 Tool 按计划继续延后。

## Phase 2 验收

- 同一业务查询只有一份核心实现。
- Tool 单测可使用 Fake Service，不需要启动 Beego HTTP。
- MCP 现有客户端接口不破坏。
- Agent 只读 Tool 不需要项目自身 Authorization Header。
- Tool 日志记录 task_id、skill、tool、duration、status，不记录 API Key/Token。