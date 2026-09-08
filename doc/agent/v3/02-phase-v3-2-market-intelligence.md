# Phase V3-2：Market Intelligence

## 状态

✅ 已完成。V3-2 建立统一、可持久化、可去重、可 Replay 的 `MarketFact / MarketEvent`，并将 News Analyst 正式加入 `symbol_analysis_team`。

## 目标

把新闻、公告、Alpha、Funding、OI、Taker、Depth、爆仓和本地 Signal 统一成可追踪、可去重、可按时间回放的 Market Intelligence。

核心时间语义：

- `event_time`：事件真实发生或数据对应的市场时间。
- `observed_at`：本系统第一次看到该事件/事实的时间。
- Replay 必须同时满足 `event_time <= replay_end` 与 `observed_at <= replay_end`，防止未来信息泄漏。
- `freshness` 在实时查询时以当前时间计算，在 Replay 时以 `end_time` 作为 as-of 计算。

## 数据模型

V3-2 新增四张表：

- `agent_market_events`：canonical 离散事件。
- `agent_market_event_sources`：同一事件的多个来源及各自观测时间。
- `agent_market_facts`：确定性的点时市场事实。
- `agent_market_source_status`：Provider 健康状态和最近错误。
统一字段包含：`type/category/symbols/event_time/observed_at/source/source_ref/headline/summary/severity/confidence/freshness/raw_ref`。

## 数据源

首版已接入：

- Binance 官方 Announcement WebSocket：`wss://api.binance.com/sapi/wss`，topic=`com_announcement_en`。
- Binance Announcement 中明确属于 Alpha listing/launch 的事件归一化为 `alpha_listing`。
- 本地 FastMove / Liquidation 等 deterministic Signal 异步镜像为 `MarketEvent`。
- 现有单币分析已经计算出的 Funding、Open Interest、Taker、Depth、Liquidation 特征归一化为 `MarketFact`，不重复放大 Binance API 请求。
- 外部 Crypto News 使用统一 `Provider` 接口或 `POST /agents/market-intelligence/events` Ingest 边界接入；Provider 故障只记录 `data_missing/source_status`。

Binance Announcement 复用现有 `binance::api_key`、`binance::api_secret`、`binance::proxy_url`，包含签名、PING、断线重连；连接或解析错误不阻塞行情、报警或交易主循环。

## 使用方式

- Agent 统一通过只读 Tool `get_market_intelligence` 查询，不直接消费 Provider 原始格式。
- `GET /agents/market-intelligence` 支持实时 Snapshot 与指定 `start_time/end_time` 的 Timeline Replay。
- `POST /agents/market-intelligence/events` 提供最多 200 条事件的 canonical Ingest。
- 同一事件重复抓取时由 `event_key` 去重，多来源写入 `agent_market_event_sources`，不会复制交易事件。
- `symbol_analysis` 的共享上下文增加 `market_intelligence`，Team 仍只执行一次 `get_symbol_analysis_context`。
## Multi-Agent 扩展

V3-1 的 Team 从：

`Technical + Flow → Supervisor`

扩展为：

`Technical + Flow + News → Supervisor`

`symbol_team_news` 是内部 Native Skill：无 Tool、无 Chat 入口、不联网，只消费 `shared_context.market_intelligence`。它必须区分 fresh/stale，不能把旧事件描述成新催化剂；无相关 fresh event 时允许 `bias=neutral / impact=none`，不得编造 Evidence。

Supervisor 现在接收三类 Typed Result；任一 Analyst 失败时仍按原 bounded Team 语义输出 partial/data_missing，不允许由其它 Agent 猜测缺失证据。

## 数据库

- 数据库版本：`5 → 6`。
- Version 6 只有 ORM Schema 变更，没有 `command/sql/version/6.sql`。
- 实际数据库只通过 `./go_binance_futures sync db` 升级。
- Version 6 连续两次 `sync db` 幂等验证通过。

## 验收 Gate

- 同一 Binance 公告重复 Ingest 不重复生成 canonical event，并可合并多个 source。
- 能按 Symbol + 时间窗口查询统一 Event/Fact Timeline。
- `event_time / observed_at / freshness` 明确区分，Replay 排除当时尚未 observed 的数据。
- News Analyst 能消费统一 Market Intelligence 并输出 Typed Evidence。
- Provider/Announcement 故障只造成 `source_status=error` / `data_missing`，不阻塞行情与交易主循环。
- V3-1 Team Replay/Stability 基线保持通过。

## 本阶段不做

- 不让 News Agent 或任何 Market Intelligence 事件直接触发真实下单。
- 不建设复杂 NLP 知识图谱。
- 不为了增加新闻源数量绕过 canonical Ingest、来源可信度或时间准确性。
- 不进入 V3-3 Historical Backtest Engine。