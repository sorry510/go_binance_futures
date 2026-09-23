# V4-0 Baseline Report

> 状态：V4-0 基线冻结报告  
> 日期：2026-09-19  
> 范围：Chat / Conversation / Skill / Local Selector / Binance API  
> 原则：只记录和验证现状，不改变业务行为。

## 1. 结论

V4-0 不需要新增业务代码。V2/V3 已经提供了后续 V4 所需的大部分底层能力，V4 应在这些能力之上演进，而不是重写 Runtime、Skill、MCP 或交易执行链。

当前基线事实：

- Chat 已有 Conversation 持久化、历史、标题、删除和单 Skill 执行，但没有 attached skills / Auto router / model preference。
- Portable Skill 已有标准 SKILL.md parser、版本、Package Hash、权限请求和安全导入；V4-2 只补 Web Draft / Editor / Publish。
- TradeCoin1～6 仍大量依赖随机与简单 24h 指标；项目同时已有完全本地、确定性的 scanner.PrefilterTop30。
- Binance REST 存在账户快照重复读取等潜在放大点；V4-4 先观测，V4-5 再治理。

详细 REST inventory 见 [v4-0-binance-api-inventory.md](./v4-0-binance-api-inventory.md)。

## 2. Chat / Conversation 基线

现有入口包括 Conversation 列表/创建/改标题/删除，以及消息读取/发送和 chat-capable Skill 列表。

核心实现文件：

- agent/conversation/chat.go
- agent/app/chat.go
- controllers/agent_chat.go
- routers/router.go
已确认行为：

- Chat Conversation 固定使用 skill=chat 区分普通 Skill Conversation。
- 默认标题为“新对话”，第一条用户消息会自动生成标题。
- Message 以 sequence 保存并按序注入 Conversation History。
- 持久化消息经过敏感文本 Redact。
- 当前一条消息只有一个 effectiveSkill；未指定时默认 general_chat。
- 同一 Conversation 已有 running task 时拒绝新任务。
- Skill 必须 Runtime 已注册、支持 Chat、Enabled 且 chat_enabled=1。
- Symbol Analysis Team 继续走现有 Team Runtime。

### 2.1 删除聊天已经存在

ORMStore.DeleteChat() 当前语义：

- 只允许删除 skill=chat 的 Conversation。
- queued/running/waiting_llm/waiting_tool/validating 状态存在时拒绝删除。
- Transaction 内删除 agent_conversation_messages 和 agent_conversations。
- 不删除 agent_tasks / Trace / Observation 等历史审计。

因此 V4-1 的删除聊天主要是前端补齐，不重新设计后端语义。

### 2.2 V4-1 缺口

当前 Conversation 缺少：

- attached skills。
- Skill 顺序/启用关系。
- Auto 多 Skill Router。
- Conversation model preference。
- message-level model override。
- 前端多 Skill与模型切换。
## 3. Portable Skill 基线

当前 SKILL.md frontmatter 支持：name、description、license、compatibility、metadata、allowed-tools。

当前约束：

- name 为 1～64 字符，只允许小写字母、数字和单连字符。
- name 必须和 package parent directory 一致。
- description 为 1～1024 字符。
- compatibility 存在时为 1～500 字符。
- 未知 top-level 字段直接失败。
- 历史 version / trusted top-level 字段会提示迁移到 metadata。

Package 安全边界已经具备：

- ZIP 或单文件 SKILL.md 导入。
- path traversal / absolute path / symlink / special file 拦截。
- Markdown 本地引用必须留在 package root 且真实存在。
- Package Hash 去重和 immutable AgentSkillVersion。
- failed install cleanup。
- scripts 可保存/索引，但标记 script_execution_disabled，不执行。

Importer 固定限制：archive <= 16 MiB；unpacked <= 32 MiB；single file <= 4 MiB；file count <= 256。

V4-2 必须复用当前 parser/importer/store，只增加 Web Draft → Validate → Publish 工作流。

## 4. Local Selector 基线

| Selector | 当前规则 |
| --- | --- |
| TradeCoin1 | 最近5m订单排除；涨跌幅两端各6个中随机2+2 |
| TradeCoin2 | 最近5m订单排除；全候选随机3 |
| TradeCoin3 | 最近10m订单排除；全候选随机2 |
| TradeCoin4 | 最近10m订单排除；全候选随机3 |
| TradeCoin5 | 最近5m本地订单排除；全候选随机5 |
| TradeCoin6 | 最近5m本地订单排除；QuoteVolume Top 200 中随机5 |

关键差异：

- TradeCoin1～4 cooldown 使用 binance.GetOrders(StartTime=...)，会产生 Futures REST。
- TradeCoin5～6 已使用本地 order 表完成 cooldown。
- 所有 selector 都要求 Enable=1。
- 随机性导致同一份市场状态重复运行结果不稳定。

现有 scanner.PrefilterTop30 已经提供完全本地的确定性评分，数据源是 local_db.symbols，当前包含：QuoteVolume、TradeCount、24h Change、High/Low/Open/Close、Center Offset、Upper Wick、Retrace、LastClose momentum、freshness、reasons/risks。

scanner 明确标记 rest_api_used=false。V4-3 应把它演进成统一 Candidate Service，而不是新写另一套评分器。

当前 scanner 已知缺失：15m/1h 转强状态、盘口点差、Binance 风险公告。V4-3 不允许为了补这些信息对全市场逐币 REST。

## 5. Binance API 调度基线

主程序主要周期：

- StartTrade：2s。
- Ownership ReconcileAll：1m。
- UpdateOrderStatus：30m。
- NoticeAllSymbolByStrategy：1.5s。
- CheckTestResults：1.5s。
- Spot/Futures TryRush：100ms。
- Spot/Futures NoticeAndAutoOrder：2s。
- Spot/Futures ListenCoin：2s。
- Funding Rate update/listen：120s。
- ExchangeInfo/precision refresh：12h。
- User Data full REST refresh：启动时 + 每30m（启用 WS 时）。
- ListenKey keepalive：20m。
这些是 scheduler 周期，不等于每次都发 REST；V4-4 必须采集真实 endpoint count。

### 5.1 User Data WS

ws::futures_user_data=1 时：启动时 REST 拉 Positions/OpenOrders → 写本地表 → User Data WS 增量更新 → 每30m完整 REST refresh。

此时 GetTransformPositions()/getTransformOpenOrders() 优先读取本地表，是当前最重要的降压机制。V4-5 只能补 freshness/fallback，不能推翻它。

### 5.2 已确认热点

热点 A：StartTrade 每轮先读取 positions/open orders，但每个候选开仓在 ensureAccountOpenSlotAvailable() 中再次读取相同数据。WS 关闭时会直接放大 REST；无 Symbol GetOpenOrder() 当前代码明确注释权重40。

热点 B：ReconcileAll 每分钟按 5 个 owner 逐个执行；每个存在 active managed position 的 owner 都可能独立 GetPosition(all)，可共享 snapshot。

热点 C：AgentTrade Risk 当前分别读取 GetPosition(all)、GetOpenOrder(all)、GetDepthAvgPrice(symbol)，连续 Proposal 有复用空间。

热点 D：TradeCoin1～4 cooldown 使用 GetOrders，而 TradeCoin5/6 已证明可以完全本地替代。

热点 E：Line Strategy / Symbol Analysis 中存在大量按 symbol/interval 的 Kline、Depth 调用；V4-4 应先实测重复率，再决定 short TTL / singleflight / local data。

### 5.3 已有保护

- Futures User Data WS + 本地 Position/OpenOrder 镜像。
- Historical REST 专用 throttling。
- signed recvWindow。
- -1021 midpoint server time sync + exactly one retry。
- Trading mutation deterministic client order ID。
- uncertain submit result reconcile，不 blind retry。
- 全市场行情 WS 减少 ticker REST。
- ExchangeInfo 主刷新为 12h。
## 6. 后续 Phase 不得破坏的基线

Chat：Conversation History 顺序不变；running task 互斥不变；DeleteChat 不删 Task/Audit；多 Skill 不自动扩大权限。

Skill：parser 安全规则不降低；published version immutable；scripts 不执行；Draft 不直接进入 Runtime。

Selector：新 selector 不让旧配置失效；Candidate Selector 不绕过 Line Strategy；不新增全市场逐币 REST。

Binance API：Trading Mutation 不普通 retry；Ownership uncertain 继续 reconcile；优化不能通过减少安全校验换权重；User Data WS 本地数据必须有 freshness/fallback。

## 7. Gate Checklist

- [x] `go test ./...`：PASS。
- [x] `go test -race ./agent/conversation ./agent/portableskill ./agent/app ./scanner`：PASS。
- [x] `go build ./...`：PASS。
- [x] frontend `pnpm typecheck`：PASS。
- [x] frontend `pnpm build`：PASS。
- [x] `git diff --check -- doc/agent/v4`：PASS。

补充说明：race 构建阶段 macOS linker 输出 `LC_DYSYMTAB` warning，但相关测试全部通过；前端构建提示 `caniuse-lite` 数据较旧，但构建成功。两者均为工具链/依赖提示，不是 V4-0 功能失败。

## 8. V4-0 完成条件

- 本报告落盘。
- Binance API inventory 落盘。
- Gate 通过，或环境性失败被明确记录。
- 不修改业务行为。
- 不修改 app.conf。
- 不写生产数据库。
- 测试结束后不留下进程。
