# Phase V3-2：Market Intelligence

## 目标

把新闻、公告、Alpha、Funding、OI、爆仓和本地 Signal 统一成可追踪、可去重、可按时间回放的 MarketFact / MarketEvent。

## 数据模型

建议至少包含：

- `type` / `category`
- `symbols`
- `event_time`：事件真实发生时间
- `observed_at`：系统获取时间
- `source` / `source_ref`
- `headline` / `summary`
- `severity`
- `confidence`
- `freshness`
- `raw_ref` 或原始 Payload 引用

事件时间和观测时间必须分开，防止旧新闻被误认为实时事件。
## 数据源

首版优先：

- Binance Announcement / Alpha 上新等官方事件。
- 可靠实时 Crypto News MCP/HTTP Source。
- 本地 Funding、OI、Taker、Depth、Liquidation、FastMove Signal。

外部新闻不要求一次接很多 Provider，先把统一接口、去重、时间语义和 Replay 做对。

## 使用方式

- Agent 通过统一 Tool 查询 Market Intelligence，不直接依赖每个 Provider 的原始格式。
- 同一事件多来源重复出现时合并 Source，不重复制造多个交易事件。
- 保存足够历史用于后续回测、Replay 和 Outcome Attribution。

## 验收 Gate

- 同一 Binance 公告重复抓取不会重复生成事件。
- 能按 Symbol + 时间窗口查询统一事件流。
- Event 必须能区分 event_time / observed_at / stale。
- News Analyst 能消费统一结构并给出 Evidence。
- Provider 故障只造成 data_missing，不阻塞行情与交易主循环。

## 本阶段不做

- 不让新闻 Agent 直接触发真实下单。
- 不建设复杂 NLP 知识图谱。
- 不为了“更多新闻源”牺牲来源可信度和时间准确性。
