# AI 与 Agent 功能

[返回中文 README](../../../README.md) · [English](../../../README.EN.md)

## AI 功能

### 对话

`AI → 对话` 是统一的人机交互入口。可显式选择允许在对话使用的 Skill；当使用单币分析时，输入框旁可直接从系统真实合约列表选择 Symbol，发送请求时后端优先使用该 Symbol，因此中文币名、别名或其它自然语言不会再导致币种识别错误。未选择 Symbol 时仍可进行普通对话，只有明确的续问语义才会复用上一轮币种。

### 单币分析

入口：`AI → 市场分析 → 单币分析`。

从真实 USDT 永续合约列表选择一个 Symbol，并可补充希望 AI 重点判断的方向。系统启动 `symbol_analysis` Skill，读取行情、市场环境和最近一次成功分析，输出结构化 `TradingPlan`。历史列表展示分析状态、方向、置信度、市场环境、当时价格、当前价格、后续涨跌和总结。

### 机会

`AI → 市场分析 → 机会发现` 是 V3-6 机会自动监控（Opportunity Watch）的用户入口。系统可以从 FastMove、强平 Signal、MarketEvent/Binance Announcement 和定时 `market_scan` 主动发现候选，并在去重/冷却后调用现有 `symbol_analysis` 生成完整 `TradingPlanV1`。机会默认 60 分钟过期，同一 Symbol + 同类来源默认 30 分钟内只分析一次。

- 机会自动监控（Opportunity Watch）默认关闭；在 `AI → AI 设置 → AI 配置` 中启用，可调整市场扫描周期和最低机会置信度。置信度输入为 **1%～100%**（保留一位小数），修改后自动保存并从服务端回读验证；接口存储为 0～1 的小数。后端会拒绝空请求、格式错误或低于 1%／高于 100% 的置信度，避免错误配置被当成保存成功。
- `neutral`、`partial`、`data_missing`、`failed` 或已过期机会只能查看，不能创建真实交易 Proposal。
- 满足条件的机会所涉及的原有受控交易流程仍需经过 Risk、人工批准和 V3-5 Ownership-safe Execution；当前「合约仓位」页面已不提供 Proposal 创建/审批/执行入口，Opportunity 本身没有 Binance Submit 路径。
- 打开机会详情可查看原 `symbol_analysis` 的完整分析，也可跳转单币分析重新分析；完整 AI 结果继续保存在 Agent Task，不复制到 Opportunity 表。

### 模型配置

入口：`AI → AI 设置 → 模型`。

LLM 配置保存在数据库中。页面支持 Provider 模板、自定义 Provider、配置名称、API 地址、API Key、模型名称、请求超时和 Temperature，支持连接测试和切换当前模型。Agent 新任务直接读取当前配置，无需重启服务；API Key 不会由接口回传明文。

### Skill 管理

Skill 注册与治理配置保存在数据库中。Native 与 Portable Skill 分为两个 Tab，支持搜索、分页、新增、编辑、启停、删除，并提供独立的 **对话可用** 开关。`enabled` 控制 Skill 能否运行，`chat_enabled` 只控制是否出现在对话入口，两者互不替代。Portable Skill 的 `allowed-tools` 只是权限申请，不能自行获得高风险或交易权限。

### MCP 管理

系统可以作为 MCP Client 连接第三方标准 HTTP MCP Server，并将远端 Tool、Resource、Prompt 纳入统一 Runtime、Permission、Trace 与 Context。支持连接测试、Catalog 刷新、认证/OAuth 和按 Skill 授权。外部 MCP 中识别出的真实交易能力会被强制归类为高风险交易能力并保持禁用，不能绕过本项目的受控执行链路。

### Memory 管理

长期 Memory 与 Conversation/Task 分离持久化。页面可查看内容、来源、Scope、TTL、状态和更新时间，用于长期上下文维护；Memory 不会获得额外 Tool 或交易权限。

### 业务 Workflow

业务 Workflow 复用现有 Agent Runtime，不创建第二套 Agent。当前包括：

- `market_scan`：确定性 Scanner 先筛选候选，再由 Agent 分析少量币种并输出机会集合。
- `strategy_review`：结合策略模板、已有测试结果、手续费后收益和当前市场环境进行复盘，只提出建议，不直接修改正式模板。
- `strategy_experiment`：界面名称为「策略草案验证」。Agent 提议候选策略，系统做确定性表达式校验和固定场景测试，再由 Agent 总结；**不等同于真实历史单仓回测**，结果不会覆盖正式策略。
- `alert_triage`：AI 开启时可把同一币种短时间内的多个相关 Signal 归并为 Incident，降低重复通知；AI 关闭或失败时保持确定性 fallback。
- `daily_market_brief`：聚合市场环境、Scanner 和重要 Signal，生成固定结构的市场摘要；Scheduler 默认关闭。

### 受控交易

**页面变更：**「合约仓位」目前只保留仓位/订单归属卡片与手动修复弹窗；Trade Proposal 创建、审批、执行界面已从该页面移除。下述流程仍是后端现有能力的说明，不代表此页面仍提供操作入口。

受控交易不是“让 LLM 直接下单”。真实执行固定经过：

`成功的 symbol_analysis → Trade Proposal → 确定性 Risk Engine → 人工批准 → 执行前再次 Risk → Execution Service → Binance → Audit`。

- 默认真实 AI 执行关闭，允许交易的 Symbol 白名单默认为空。
- Proposal 只能从成功的结构化单币分析创建；LLM 不决定最终 quantity。
- Risk Engine 检查白名单、方向、当前 MarketCondition 及漂移、价格 freshness、Entry Zone、Stop Loss、滑点、已有仓位/挂单、重复订单、cooldown、杠杆、单笔风险、名义金额、总 Exposure 和 Kill Switch。
- 最终 quantity 根据最大风险、止损距离、最大名义金额和交易对 StepSize 计算。
- 即使已经人工批准，真实执行前仍会重新运行 Risk；Broker 提交前还会再次检查 Kill Switch。
- 使用唯一 `client_order_id` 保证幂等。网络结果不确定时不会自动重复下单，只允许按 `client_order_id` 对账。
- 所有 Proposal、Risk、批准/拒绝、执行和对账动作都会写入审计。
- 受控交易开仓成交后由 V3-5 Ownership Lifecycle 为 managed position 创建必需的 `STOP_MARKET` 止损，并在 TradingPlan 提供有效目标时创建可选的 `TAKE_PROFIT_MARKET`；部分人工减仓后会按剩余 managed quantity 对账并修复保护单。

### AI 配置

所有 AI 相关运行配置集中在 `AI → AI 设置 → AI 配置`（第一个、默认标签），包括：

- AI 报警 Pipeline、AI 分析开关、最小严重级别、cooldown、并发与每分钟上限。
- 机会自动监控开关、市场机会扫描周期与最低机会置信度（1%～100%；编辑后通过接口回读确认保存）。
- 市场环境自动更新和每日市场摘要 Scheduler。
- Agent 每分钟/每小时启动额度、全局 Token 与 Tool 调用预算。
- 受控交易 Kill Switch、Symbol 白名单、最大单笔风险、最大名义金额、总 Exposure、最大杠杆、价格 freshness、滑点、cooldown 和 Proposal TTL。

AI 设置还有「模型」「Skill」「MCP」「Memory」标签。以上 AI 运行配置已从“配置中心”迁出；`配置中心` 只保留非 AI 的交易、行情监听、提醒和系统运行配置。

### 任务中心、系统看板与报警历史

- **任务中心**：查看 Agent 治理状态、Runtime 直接交易权限与受控执行开关、Scheduler、Task 历史、模型和 Token 等运行信息。
- **系统看板**：原“AI → 可观测性”已提升为最上方顶级菜单。顶部只显示 Database、Binance REST、统一 WS 行情、LLM、Agent 和 Trade Safety 六项健康摘要；WS 展开后分别查看 U 本位合约（symbols）、现货（spot_symbols）和币本位合约（delivery_symbols）的行情新鲜度，未启动的币本位 WS 不计入异常。公告、市场情报、MCP、调度器详情按需展开；保留 Agent 任务数、成功率、Token、LLM 错误和调用追踪，以及 Skill/模型/Tool 统计。Binance REST 常显限额预算、429/418，详细排行按需展开，前端每 1 秒无 loading 静默刷新一次本地 REST 快照（请求不重叠、失败保留上次数据）、每 60 秒刷新一次系统健康；切换 Agent 时间窗口不重新请求健康和 REST 状态。Lead 状态在合约带单页面查看，变更历史已从看板移除。
- **报警链路历史**：查看 Signal 从 Event Bus、Signal Engine、AI 分析/归并到 Notification 或 fallback 的完整处理历史。
