# Code Review — V3-2 Market Intelligence（统一市场情报）

- **Phase**：V3-2 / Market Intelligence（统一市场情报）
- **分支**：`feat/ai-agent-v3`，HEAD `c3b211d`（v1.0.9）
- **评审模式**：review-only（仅评审，未修改任何代码、未写内存、未追问）
- **评审结论**：**AUTOMATED PASS / 人工验收待定**（无阻塞级缺陷）
- **评审日期**：2026-09-04

---

## 0. 评审范围与验证手段

**范围锁定**（git diff --stat / git status — 仅后端 Go 代码，前端 `static/` 构建产物不在评审范围）：

新增文件：
- `models/agent_market_intelligence.go`（4 张表模型）
- `controllers/agent_market_intelligence.go`（HTTP 入口：Snapshot / Timeline / Ingest）
- `service/marketintelligence/{service,snapshot,types,binance_announcement,provider,service_test}.go`（核心服务 + 9 个测试）
- `service/symbolanalysis/market_intelligence.go`（Funding/OI/Taker/Depth/Liquidation 归一化为 Fact）

修改文件：
- `agent/app/{default,skill_catalog,skill_catalog_test}.go`（注册 `symbolteam.News()`、并发 2→3）
- `agent/skills/symbolteam/{skills,skills_test,types}.go`（新增 News skill + 校验）
- `agent/team/{runner,runner_test}.go`（并发 2→3，specs 加 News）
- `agent/tools/domain/register.go`（只读工具 `get_market_intelligence`）
- `binanceproxy/proxy.go`（复用代理池的 WebSocket Dialer）
- `main.go`（`dbVersion 5→6`、注册 4 表、启动 Binance 公告 WS、tradeSecret/tradeProxyURL）
- `routers/router.go`（新增两条路由）
- `service/alertpipeline/default.go`（异步 `IngestSignal` 镜像本地 Signal）
- `service/symbolanalysis/context.go`（`Context.MarketIntelligence` + `BuildMarketIntelligence`）
- `models/agent_task_syncdb_test.go`（注册并断言 4 张新表）
- `doc/agent/v3/{02-phase-v3-2-market-intelligence,README}.md`

**验证命令**（规避缓存与 git lock）：
```
export PATH=/usr/local/go/bin:$PATH
go build ./...
go vet ./...
go test -count=1 -race ./service/marketintelligence/... ./agent/skills/symbolteam/... \
  ./agent/team/... ./models/... ./controllers/... ./service/alertpipeline/... \
  ./service/symbolanalysis/... ./agent/app/... ./agent/tools/... ./binanceproxy/...
go test -count=1 ./...
```

---

## 1. 构建与测试结果

| 项 | 结果 | 说明 |
|----|------|------|
| `go build ./...` | ✅ PASS | BUILD_EXIT=0，无编译错误 |
| `go vet ./...` | ⚠️ 仅既有噪声 | 仅 `main.go:299/304` unreachable code（两个提前 `return` 退出的已禁用 goroutine）；**非 V3-2 引入**，V2-12/V3-1 已确认并排除 |
| `go test -count=1 -race`（受影响包） | ✅ 全绿 | marketintelligence / symbolteam / team / models / controllers / alertpipeline / symbolanalysis / agent/app / tools/domain / binanceproxy 均 `ok` |
| `go test -count=1 ./...`（全量回归） | ✅ 无失败 | 仅 `ok` 与 `[no test files]`，无任何 FAIL |

**结论**：编译、静态检查、竞态测试、全量回归均通过，无回归。

---

## 2. 6 项 Gate 验收逐条核对

### Gate 1 — 同一 Binance 公告重复 Ingest 不重复生成 canonical event，并可合并多个 source
**状态：PASS（已自动化覆盖：TestIngestEventDeduplicatesAnnouncementAndMergesSources / TestBinanceAnnouncementAndExternalNewsMergeIntoOneCanonicalEvent）**

- `IngestEvent` 以 `event_key`（SHA256 确定性，含 1 分钟桶 + 归一化 headline）唯一查表；`ErrNoRows` 才 insert，冲突后 reread 视为成功（幂等）。
- 已存在 canonical event 时（service.go:71-87）：**仅当 `input.ObservedAt < row.ObservedAt` 更新 `ObservedAt` 与重算 `Freshness`；仅当 `input.ObservedAt > row.UpdatedAt` 更新 `UpdatedAt`**。绝不回写 `EventTime`/`Headline`/`Summary`/`Source`/`Severity`/`Confidence`/`SymbolsJSON` —— **canonical event 不可变（append-only）**。
- 多来源合并走 `upsertEventSource`（service.go:96-122）：以 `source_key = hash(event_key|source|sourceRef|rawRef)` upsert `agent_market_event_sources`，不触碰 canonical 行。
- `TestParseBinanceAnnouncement...` 系列验证同一公告经 WS 与外部新闻两次 Ingest 合并为同一 canonical event。

### Gate 2 — 能按 Symbol + 时间窗口查询统一 Event/Fact Timeline
**状态：PASS（已自动化覆盖：TestFactQueryBySymbolAndWindow / TestTimelineUsesEndTimeAsReplayFreshnessReference / TestTimelineExcludesEventsNotObservedByReplayEnd）**

- `ListEvents`（service.go:158+）：支持 `symbols_json__icontains`、`event_type`、`category`、`event_time__gte/lte`、`observed_at__lte`，按 `-event_time,-observed_at` 排序。
- `ListFacts`（同包）：按 `symbol`/`fact_type`/`category`/时间窗过滤。
- 控制器 `Get`（controllers/agent_market_intelligence.go）：`start_time/end_time` 任一 >0 → `Timeline`，否则 `window_minutes`（1..10080）→ `Snapshot`。

### Gate 3 — event_time / observed_at / freshness 明确区分，Replay 排除当时尚未 observed 的数据
**状态：PASS（已自动化覆盖：TestEventTimeObservedAtAndFreshnessRemainDistinct / TestTimelineUsesEndTimeAsReplayFreshnessReference / TestTimelineExcludesEventsNotObservedByReplayEnd / TestParseBinanceAnnouncementPreservesEventAndObservedTime）**

- 三字段在 insert/update 路径严格分列；`freshness(type, eventTime, observedAt)` 按 TTL 表（depth/taker/oi=5m、funding=15m、liquidation/signal=1h、announcement/alpha/news=24h、default=6h）计算 fresh/stale/unknown。
- `Timeline` 双过滤（snapshot.go:48-69）：`replay.Now = func() time.Time { return asOf }`（asOf = end_time UTC），且 `ListOptions{ObservedBefore: endTime}` 同时约束 `event_time ≤ endTime` 与 `observed_at ≤ endTime`，**杜绝 Replay 泄漏「当时尚未观测」的未来信息**。`startTime > endTime` 被拒。

### Gate 4 — News Analyst 能消费统一 Market Intelligence 并输出 Typed Evidence
**状态：PASS（已自动化覆盖：TestNewsValidatorUsesMarketIntelligenceEvidenceAndAllowsNoFreshCatalyst）**

- `symbolteam.News()` 仅返回 `Definition{kind: kindNews}`，**无 tools、无 Chat、不联网**，`newsPrompt` 明确「Use only shared_context.market_intelligence」「never invent missing news or sources」。
- `validateNews`（skills.go:262-284）：`Bias ∈ {bullish|bearish|mixed|neutral}`、`Impact ∈ {high|medium|low|none}`、`Confidence 0..1`、`Summary` 必填；**每条 evidence 必须 `Source == "market_intelligence"` 且 `Finding` 非空**，否则拒绝（拒绝 `twitter_guess` 等非可信来源）。
- evidence 允许为空（无新鲜催化剂 → `bias=neutral/impact=none`，保持空 evidence 而非编造），符合「不编造」。

### Gate 5 — Provider/Announcement 故障只造成 source_status=error / data_missing，不阻塞行情与交易主循环
**状态：PASS（已自动化覆盖：TestProviderFailureIsDataMissingAndDoesNotBlockOtherProviders）**

- `provider.SyncProviders`：逐 provider 隔离，单 provider 失败仅 `RecordSourceFailure` 并返回带 `Error` 的 `ProviderSyncResult`，**不阻断其余 provider**（service.go provider.go）。
- `RunBinanceAnnouncementStream`（binance_announcement.go）：独立 goroutine；缺 `api_key/secret` → `RecordSourceFailure` 返回；循环 5s 重连；连接/解析错误仅记 source 健康，**不向上抛出、不阻塞调用方**。
- `Snapshot` 将 `source_status=error` 汇总为 `DataMissing`（snapshot.go:36-44），供 Team 作为 `data_missing` 显式呈现，而非中断主循环。
- `service/alertpipeline/default.go` 的 `IngestSignal` 走 `go persistMarketIntelligenceSignal(...)`（2s 超时），异步、故障隔离。

### Gate 6 — V3-1 Team Replay/Stability 基线保持通过
**状态：PASS**

- `agent/team/runner.go`：`MaxConcurrency 2→3`（Technical+Flow+News 并行）；`SkillSourceVersion "v3-2"`；`specs` 加 `RoleNews/NewsSkillName`；`Start`/`StartWithOptions` 入口保留，向后兼容。
- `agent/team/runner_test.go`：子任务 3→4、evidence 2→3、budget links 2→3，固定样本断言保持。
- 受影响包（`agent/team`、`agent/skills/symbolteam`、`agent/app`）race 测试全绿，无并发回归。

---

## 3. 安全专项核对（与 V2-12 / V3-1 一致口径）

| 维度 | 结论 | 证据 |
|------|------|------|
| 去重不复制交易事件 | ✅ | canonical event 不可变（Gate 1）；事实 `IngestFact` 同样 `fact_key` 幂等、append-only、不跨 fact 合并数据 |
| 时间语义不泄漏未来信息 | ✅ | Gate 3 双过滤；`freshness` 始终由 canonical `EventTime` 重算 |
| Provider 故障隔离不阻塞主循环 | ✅ | Gate 5；WS/provider 失败仅记 source 健康 |
| News 不自授工具、不编造证据 | ✅ | `newsPrompt` 禁工具/禁编造；`validateNews` 锁 `market_intelligence` 来源 |
| Team 仅一次共享上下文 | ✅ | `context.go` 中 `MarketIntelligence` 由 `BuildMarketIntelligence` 一次性构建挂入 `shared_context`；Team 仍只调一次 `get_symbol_analysis_context` |
| 只读工具 | ✅ | `get_market_intelligence`：`ToolRisk=permission.RiskRead`、`additionalProperties:false`（`strictDecode` 强制）、`required:["symbol"]`、`window_minutes≤10080`、`limit≤200`、5s 超时/192KB 上限；仅经 `GetMarketIntelligence`/`GetMarketIntelligenceTimeline` 读，无写入/Ingest 路径暴露给 Agent |
| DB 迁移兼容（v5→v6 ORM-only） | ✅ | 见 §4 |
| 本阶段不直接下单 | ✅ | 规格「本阶段不做」明确禁止 News/MI 事件触发真实下单；无新增交易/下单代码路径 |

---

## 4. DB 迁移兼容（v5 → v6，ORM-only）

- `main.go`：`dbVersion 5→6`，`registerModels` 注册 `AgentMarketEvent/Source/Fact/SourceStatus` 4 表；仅本仓新增 `tradeSecret`/`tradeProxyURL` 配置项。
- `command/db_update.go`：`UpdateDatabase` 在版本循环中对 `./command/sql/version/%d.sql` 做 `os.Stat`——**文件存在才执行**；现有仅 `1.sql`/`2.sql`/`3.sql`，故 v6 **无 SQL 文件、零破坏性 DDL**，纯靠 `SyncDatabase` 调用 `orm.RunSyncdb("default", false, false)`（additive，不删表）由 ORM 注册表创建 4 张新表。
- `models/agent_task_syncdb_test.go`：`orm.RegisterModel` 已纳入 4 张新表，断言循环确认 `agent_market_events/agent_market_event_sources/agent_market_facts/agent_market_source_status` 在**保留既有行**的既有 DB 上升级后均被创建；该测试包含在 `models` 包 race 测试中且通过。
- **结论**：v5→v6 向后兼容、幂等、非破坏性，自动化覆盖到位。

---

## 5. 非阻塞建议（不阻塞 Gate，建议后续迭代）

1. **`main.go:299/304` unreachable code**：两个提前 `return` 的已禁用 goroutine 残留 `unreachable code`，历史既有问题（V2-12/V3-1 已标注）。建议单独清理，勿混入本 Phase。
2. **`parseBinanceAnnouncement` 币种抽取正则可观测性**：`parenthesizedAsset`/`usdtPairAsset` 依赖标题格式，建议补一条「未识别到币种 → `RecordSourceFailure`/记 data_missing」的显式分支，避免静默漏掉公告。
3. **`IngestEvents` 批量失败部分成功**：控制器循环 `IngestEvent`，单条失败当前直接返回错误（未继续后续条）。建议改为「收集 per-item 错误并返回部分成功结果」，与 `SyncProviders` 的隔离风格一致。
4. **`Snapshot` 未传 `DataMissing` 到 `Timeline`**：`Timeline` 返回空 `Sources/DataMissing`（snapshot.go:68），Replay 场景不反映 source 健康。若 Replay 也需呈现 data_missing，建议从 `SourceStatuses` 注入。
5. **`RawJSON/DataJSON` 防泄露**：模型已 `json:"-"`，API 不序列化原始负载；建议补充一条序列化层测试，防止未来误加 `json` tag 导致原始数据外泄。

---

## 6. 未验证项 / 人工验收待办

**自动化未覆盖（需人工/集成验证）：**
- 真实 Binance 公告 WS 端到端连通（`wss://api.binance.com/sapi/wss`，HMAC-SHA256 签名 + 代理 + 5s 重连）：单元测试用 mock，未连真实环境。
- 外部 Crypto News Provider 真实 HTTP 拉取与归一化质量：依赖具体 Provider 实现，需联调。
- `symbol_analysis` 已算特征实时归一化为 Fact 的端到端时机与去重稳定性（1 分钟桶 `DedupKey`）。
- 前端 `go_binance_futrues_new_ui` 对 `/agents/market-intelligence` 的展示（本仓 `static/` 仅构建产物，前端仓库需另行 review）。

**人工验收清单：**
- [ ] 真机运行 Binance 公告 WS，确认公告进入 `agent_market_events` 且多来源合并。
- [ ] 用 `curl` 调 `POST /agents/market-intelligence/events` 注入一条外部新闻，确认与 Binance 公告按 `event_key` 合并为同一 canonical event。
- [ ] 调 `GET /agents/market-intelligence?symbol=BTCUSDT&start_time=...&end_time=...` 验证 Replay 双过滤（不应返回 `observed_at > end_time` 的事件）。
- [ ] 确认 Provider 故意返回 500 时，`source_status=error` 且行情/交易主循环不受影响。
- [ ] 确认 News Analyst 在「无新鲜催化剂」场景下输出 `neutral/none` 且 evidence 为空，不编造。
- [ ] 确认 DB 从 v5 升级到 v6 后既有数据完好、4 张新表存在。

---

## 7. 最终结论

**AUTOMATED PASS / 人工验收待定。**

- 编译、静态检查（仅既有噪声）、竞态测试、全量回归全部通过。
- 6 项 Gate 验收均有对应自动化测试或代码证据支持，全部成立。
- 安全口径（去重/时间语义/故障隔离/News 安全/Team 只读/只读工具/DB 兼容/不直接下单）全部满足。
- 无阻塞级缺陷；5 条非阻塞建议与 6 项人工验收待办供后续迭代与联调参考。

> 本报告为纯评审产出，未改动任何源代码、未执行任何写操作、未写入项目/用户内存。
