# V3-2 Market Intelligence Implementation Report

## 状态

V3-2 已完成。实现统一 Market Intelligence 数据面，并在不建立第二套 Agent Runtime 的前提下，将新闻/公告/Alpha、资金面与本地 Signal 汇入可去重、可追踪、可 Replay 的 `MarketEvent / MarketFact`。

## 核心架构

```text
Binance Announcement WS ─┐
External News Provider/API ├─→ canonical Ingest ─→ MarketEvent
Local Signal Engine ──────┘

Funding / OI / Taker / Depth / Liquidation
                ↓
        deterministic features
                ↓
             MarketFact

MarketEvent + MarketFact
        ↓
Snapshot / Timeline / Replay
        ↓
get_market_intelligence
        ↓
Symbol Shared Context
        ↓
Technical + Flow + News → Supervisor
```
## 数据模型

新增 ORM 表：

- `agent_market_events`：canonical event，`event_key` 唯一。
- `agent_market_event_sources`：一个 canonical event 的多来源记录，`source_key` 唯一。
- `agent_market_facts`：点时确定性事实，`fact_key` 唯一。
- `agent_market_source_status`：Provider 健康状态。

事件和事实都保存 `event_time` 与 `observed_at`。canonical Event 的 `observed_at` 永远保留最早观测时间；晚到的第二来源只更新 source 列表和 `updated_at`，不会把事件伪装成刚发生。

实时 freshness 按当前 as-of 计算；历史 Timeline 将 `end_time` 固定为 as-of，并增加 `observed_at <= end_time` 条件，因此 Replay 不会读取系统当时尚未获得的信息。

当前 freshness TTL：

- Depth / Taker / OI：5 分钟。
- Funding：15 分钟。
- Liquidation / local Signal：1 小时。
- Announcement / Alpha / News：24 小时。
- 其它类型：6 小时。

## Binance Announcement

新增官方 Announcement WebSocket consumer：

- endpoint：`wss://api.binance.com/sapi/wss`
- topic：`com_announcement_en`
- 复用现有 Binance API Key / Secret。
- 复用现有 `binance::proxy_url` 以及 SOCKS5/SOCKS5H/HTTP proxy pool。
- HMAC-SHA256 签名、30 秒 PING、5 秒断线重连。
- `publishDate → event_time`，本地接收时间 → `observed_at`。
Announcement payload 保存 catalog/title/body 原始内容；明确包含 Alpha listing/launch 语义的公告归一化为 `alpha_listing`。标题中的资产符号会转换为对应 USDT symbol，用于按币种查询。

连接、签名、代理、解析或 Ingest 错误只写 `agent_market_source_status`，不会终止价格 WS、Signal Engine、Alert Pipeline 或交易循环。

## 本地 Market Data / Signal

`symbolanalysis.Build()` 继续复用现有 Binance 请求和本地数据，不新开第二套抓取：

- Funding → `MarketFact(funding)`
- Open Interest → `MarketFact(open_interest)`
- Taker Ratio → `MarketFact(taker)`
- Depth → `MarketFact(depth)`
- Liquidation aggregate → `MarketFact(liquidation)`

这些 Fact 按确定性 key 去重，因此重复分析同一分钟数据不会制造无意义副本。

FastMove / Liquidation Signal 保留原 Alert Pipeline，同时通过异步镜像写入 `MarketEvent(signal)`。Market Intelligence 写入失败只记录 warning，不改变报警发送语义。

## Tool / API

新增只读 Native Tool：`get_market_intelligence`。

支持两种模式：

- realtime：`symbol + window_minutes + limit`
- replay：`symbol + start_time + end_time + limit`

新增 HTTP API：

- `GET /agents/market-intelligence`
- `POST /agents/market-intelligence/events`

POST Ingest 每次最多 200 条事件，外部 Crypto News/MCP/HTTP collector 统一通过该 canonical boundary 接入。另保留 `Provider` + `SyncProviders()` 接口，用于后续增加受控 Provider，而不改变 Agent 数据契约。
## Multi-Agent Team

V3-1 Team 已从两个 Analyst 扩展为三个 Analyst：

`Technical + Flow + News → Supervisor`

新增内部 Skill：`symbol_team_news` / `news_analyst`。

约束：

- Tools 为空，不允许自主联网或调用数据源。
- `ChatDefault=0`，不在用户 `/` Skill 菜单中暴露。
- 只允许消费 Team 已一次采集的 `shared_context.market_intelligence`。
- 输出 Typed `NewsAnalysisV1`：bias、impact、summary、confidence、data_missing、evidence。
- stale event 不能描述为新催化剂。
- 没有 relevant fresh event 时输出 `neutral / none`，Evidence 可以为空，禁止编造。

Supervisor input/output contract 升级到 V3-2，Evidence role 允许 `technical_analyst / flow_analyst / news_analyst`。任一 Analyst 失败仍遵循 partial/data_missing，不降低 V3-1 的 bounded Team 安全边界。

## 数据库

- `dbVersion: 5 → 6`
- Version 6 为纯 ORM Schema 升级，没有 `command/sql/version/6.sql`。
- 实际执行 `./go_binance_futures sync db` 成功完成 5 → 6。
- 最终连续两次再次执行 `sync db` 均返回 Version 6 已是最新，幂等通过。

## 验收与回归
最终通过：

```bash
go test -count=1 ./...
go test -race ./service/marketintelligence ./service/symbolanalysis \
  ./agent/skills/symbolteam ./agent/team ./agent/tools/domain \
  ./service/alertpipeline ./agent/app
go test -count=1 -run \
  'TestSymbolAnalysisTeamFixedFixtureStabilityMatchesSingleAgentBaseline|TestTeamFixtureReplayIsDeterministic' \
  -v ./agent/team
go build -o go_binance_futures .
./go_binance_futures sync db
```

核心 Gate 测试覆盖：

- Announcement 重复 Ingest 的 canonical dedupe + source merge。
- `event_time / observed_at / freshness` 独立语义。
- Replay 以历史 as-of 计算 freshness，并排除 `observed_at > end_time` 的未来信息。
- Provider failure isolation 与 source health。
- Binance Announcement payload 的 publishDate、Alpha 分类和 symbol 提取。
- Market Intelligence Tool 与 symbol shared context。
- News Analyst Typed validation、无 Tool、无 Chat 暴露。
- 三 Analyst Team、partial child failure、token budget、Replay determinism 和 V3-1 stability baseline。

Race Test 仅出现已有的 macOS `LC_DYSYMTAB` linker warning，没有 data race。

## 边界确认

- Market Intelligence / News Analyst 不创建 Proposal、不触发真实下单。
- 未进入 V3-3 Backtest。
- 未新增复杂知识图谱。
- 未修改 `doc/TODO.md`。
- V3-2 没有新增专门前端页面；现有前端未提交工作保持原样，没有被本阶段覆盖。
