# Code Review — V3-3 Historical Backtest Engine（历史回测引擎）

- **Phase**：V3-3 / Historical Backtest Engine
- **分支**：`feat/ai-agent-v3`，HEAD `55b6c90`（merge of `feat/ai-agent-v3`）
- **评审模式**：review-only（仅评审，未修改任何代码、未写内存、未追问）
- **评审结论**：**AUTOMATED PASS / 人工验收待定**（无阻塞级缺陷）
- **评审日期**：2026-09-09

---

## 0. 评审范围与验证手段

**范围锁定**：V3-3 实现由「未跟踪（untracked）的新文件」+「已跟踪文件的增量 diff」组成（前期 `git status | head -60` 因截断未显示 untracked 文件，已补全核对）。

新增文件（untracked，`??`）：
- `models/agent_backtest.go`、`models/historical_market.go`（21 张新表的 ORM 模型）
- `controllers/agent_backtest.go`、`controllers/agent_historical_market.go`
- `service/backtest/{types,dataset,engine,store,metrics,environment}.go` + `engine_test.go` + `store_test.go`
- `service/historicalmarket/{types,repository,binance_source}.go` + `repository_test.go`

已跟踪文件的增量 diff（vs HEAD）：
- `main.go`（`dbVersion 6→7` + 注册 21 张新表）
- `routers/router.go`（7 条新路由）
- `feature/api/binance/index.go`（`+GetHistoricalKlines`、`+GetHistoricalFundingRates`）
- `command/db_update_test.go`（v6/v7 迁移断言）、`models/agent_task_syncdb_test.go`（注册并断言新表）
- `doc/agent/v3/{03-phase-v3-3-backtest,README}.md`

**验证命令**（规避缓存与 git lock）：
```
export PATH=/usr/local/go/bin:$PATH
go build ./...
go vet ./...
go test -count=1 -race ./service/backtest/... ./service/historicalmarket/... \
  ./models/... ./controllers/... ./feature/api/binance/... ./command/...
go test -count=1 ./...
```

---

## 1. 构建与测试结果

| 项 | 结果 | 说明 |
|----|------|------|
| `go build ./...` | ✅ PASS | 无编译错误 |
| `go vet ./...` | ⚠️ 仅既有噪声 | 仅 `main.go:321/326` unreachable code（两个提前 `return` 退出的已禁用 goroutine）；**非 V3-3 引入**——V2-12/V3-1/V3-2 均为 `main.go` 同类噪声，行号因本 Phase 在 main.go 增加约 22 行而顺移 |
| `go test -count=1 -race`（受影响包） | ✅ 全绿 | service/backtest、service/historicalmarket、models、controllers、feature/api/binance、command 均 `ok` |
| `go test -count=1 ./...`（全量回归） | ✅ 无失败 | 仅 `ok` 与 `[no test files]`，无任何 FAIL |

**结论**：编译、静态检查、竞态测试、全量回归均通过，无回归。

---

## 2. 验收 Gate 逐条核对（规格 §验收 Gate，共 7 条）

### Gate 1 — LONG、SHORT、止盈、止损、无交易固定 Fixture
**状态：PASS**（已自动化覆盖：`TestBacktestLongUsesNextBarOpen` / `TestBacktestShort` / `TestBacktestTakeProfit` / `TestBacktestStopLossWinsWhenStopAndTargetBothTouched` / `TestBacktestNoTrade`）
- Long/Short 开平仓、止盈、止损均按预期产生 Trade。
- **同 Bar TP/SL 同时命中 → stop_loss 优先**（`engine.go:242` 先检查 StopLoss 并立即返回，`engine_test` 显式断言 `ExitReason=="stop_loss"`）。
- **保护平仓后同 Bar 不重开**（`engine.go:142` 的 `!closedProtectiveThisBar` 守卫）。

### Gate 2 — 手续费、Funding、滑点均有单元测试
**状态：PASS**（已自动化覆盖：`TestBacktestFeesAndSlippage` / `TestBacktestFundingLongPaysPositiveRate`）
- 滑点模型：`applySlippage`（engine.go:266）对 LONG 开仓/SHORT 平仓加价、其余减价，纯成本模型，对称性一致。测试断言 entry 100→100.1、exit 110→109.89。
- 手续费：开仓 `notional*FeeRate` + 平仓 `|qty|*fill*FeeRate`，计入 NetPnL。
- Funding 多空方向：LONG 正费率时 `payment=-notional*rate`（支出，FundingPnL<0），SHORT 反向收入（`engine.go:122-125`），测试断言 LONG 正费率时 `FundingPnL<0`。

### Gate 3 — 明确验证未来 Bar 不可见
**状态：PASS**（已自动化覆盖：`TestHistoricalEnvironmentCannotSeeFutureBars`）
- `environment.series/tickerStats` 用 `sort.Search(..., all[i].CloseTime > asOf)` 以 `asOf=bar.CloseTime` 为界，只返回 `CloseTime ≤ asOf` 的 Bar（`environment.go:87/103`）；指标计算（`addIndicators`）同样只基于可见窗口。测试断言 `series` 在 `first.CloseTime` 只返回 1 根、且 `NowPrice==100`（不泄漏后续 999 的 Bar）。

### Gate 4 — 同一输入内存 Dataset/Strategy/Engine 重放必须确定性
**状态：PASS**（已自动化覆盖：`TestBacktestReplayIsDeterministic`）
- 两次同输入 Run 的 JSON 序列化完全一致。确定性来自：`DatasetDataHash` 对 bars 按 key 排序后逐 bar 编码、环境只用 `bar.CloseTime`（非 `time.Now`）、`evaluateRules` 用按索引缓存的编译程序、无 map 遍历影响顺序。

### Gate 5 — 相同 Dataset Spec 在全局 Kline 被覆盖后：Dataset ID / Spec Hash 不变，Run data_hash 必须变化
**状态：PASS**（已自动化覆盖：`TestSameDatasetSpecUsesLatestCanonicalMarketData`）
- `DatasetSpecHash`（dataset.go:201）只哈希 Market/Symbol/ExecutionInterval/Intervals/Benchmarks/Start/End/Warmup，**不含 Bars/Funding** → 同 Spec 稳定。
- `DatasetDataHash`（dataset.go:212）哈希 `spec + 全量 bars（key 排序后逐 bar）+ funding` → 全局 Kline 内容变化则变。
- 测试：同 Spec 跑两次，中途 `repo.Import` 覆写一根 canonical Kline（last-write-wins），断言 `DatasetID/SpecHash` 不变、`DataHash` 必须变化。

### Gate 6 — 完整本地历史范围不得请求 Binance；内部缺口只请求缺失区间
**状态：PASS**（已自动化覆盖：`TestRepositoryReusesCompleteLocalRange` / `TestRepositoryFetchesOnlyInternalGap`）
- `Repository.LoadKlines`（repository.go:27）先查本地 `queryKlines`，再 `missingKlineRanges` 计算缺口，**仅当存在缺口且 `Source != nil` 时按 gap 逐个向 Binance 拉取并 Import**，重查后仍有缺口才报错。
- 测试：完整本地范围 → `source.calls==0`（不请求 Binance）；缺失中间一根 → 恰好 1 次 Source 调用且边界等于缺口 `b1` 的 open/close。

### Gate 7 — Backtest 不修改真实策略、旧模拟盘状态或真实 Binance 账户
**状态：PASS**（已自动化覆盖：`TestManagerPersistsDeterministicBacktestWithoutLegacyPaperTables`）
- `Manager.run`（store.go:76）只读 `StrategyTemplates` 快照，只写入 `agent_backtest_*` 与全局 `market_*` 缓存表；引擎从不调用真实下单、不写策略模板、不访问实时 Binance/WS/`symbols`/当前 MarketCondition（`engine.go` 全程只消费 `dataset.Bars` + `dataset.Funding`）。
- 测试断言：回测成功且 Dataset/Trades/Equity/Events/全局缓存均落库，**且不存在** `agent_backtest_bars`/`agent_backtest_funding` 旧表（Dataset 不拥有历史数据，符合"本阶段不做"与规格"Dataset 不再复制历史 Bar/Funding"）。

---

## 3. 安全专项核对

| 维度 | 结论 | 证据 |
|------|------|------|
| 未来函数防护 | ✅ | Gate 3 + `environment.go` `sort.Search(asOf)` 双重保证 |
| 确定性（同输入可重放） | ✅ | Gate 4 + 无 `time.Now`/随机源（runID 用 crypto/rand 但仅作 ID，不影响计算） |
| 与真实账户/策略隔离 | ✅ | Gate 7；引擎纯模拟，绝不写策略模板/模拟盘/Binance 账户 |
| 只读策略 | ✅ | `Manager.run` 仅 `o.Read(&template)` 取快照，零写入 |
| 外部导入鉴权 | ✅ | 全局 `JwtMiddleware`（`main.go:246`，`*` BeforeRouter）仅跳过 `excludeRoutes` 白名单；`/agents/historical-market/import`、`/agents/backtests*` **不在白名单**，需有效 JWT（与既有 `/agents/*` 一致） |
| SQL 注入防护 | ✅ | `KlineTable(interval)` 返回固定白名单表名（非用户输入）；`TestKlineTableWhitelist` 断言 `"1m;drop table x"` 被拒 |
| DB 迁移兼容（v6→v7 ORM-only） | ✅ | 见 §4 |
| 本阶段不直接下单 | ✅ | 规格"本阶段不做"明确禁止；引擎仅产出模拟 Trade/Equity，无下单路径 |
| 审计可追溯 | ✅ | `AgentBacktestEvent`（signal→order→fill→position）逐 Bar 记录；`MarketDataImportBatch` 记录所有导入/Binance 补缺来源 |

---

## 4. DB 迁移兼容（v6 → v7，ORM-only）

- `main.go`：`dbVersion 6→7`，`registerModels` 注册 21 张新表（`MarketKline1m…1mo`、`MarketFundingRate`、`MarketDataImportBatch`、`AgentBacktest*` 五张）。
- `command/db_update.go`：`UpdateDatabase` 对 `./command/sql/version/%d.sql` 做 `os.Stat`——**文件存在才执行**；现有仅 `1.sql`/`2.sql`/`3.sql`，故 v7 **无 SQL 文件、零破坏性 DDL**，纯靠 `SyncDatabase` 调用 `orm.RunSyncdb("default", false, false)`（additive，不删表）由 ORM 注册表创建新表。
- `command/db_update_test.go`：新增 `SyncDatabase(1)→(5)→(6)→(7)` 链路，断言 v6（4 张 MI 表）与 v7（16 张 kline/funding/import + 5 张 backtest 表）在「保留既有行」升级后均被创建且 `config.Version` 正确；`models/agent_task_syncdb_test.go` 同步注册并断言这 21 张表。两条测试均在 targeted race 运行中通过。
- **结论**：v6→v7 向后兼容、幂等、非破坏性，自动化覆盖到位。

---

## 5. 非阻塞建议（不阻塞 Gate，建议后续迭代）

1. **warmup 跳过用字符串匹配，存在脆弱点**：`engine.go:156` 用 `strings.Contains(envErr.Error(), "warmup") || "visible bars"` 决定是否跳过 warmup 期错误；但 `environment.series` 返回的 `"no execution bars visible at %d"` 不含这两个子串。正常执行区间（当前 Bar 必可见）不会触发该错误，但建议在 `environment.Build` 返回**类型化错误**（如 `ErrWarmup`/`ErrInsufficientBars`），引擎按类型判断，避免未来某次改动误判为致命错误。

2. **指标索引约定未被真实指标测试覆盖**：`engine_test.go` / `store_test.go` 的 fixture 均使用 `TechnologyJSON:"{}"`（无指标）。`environment.series` 将时序 Bar **反转成最新在前**（`environment.go:96-98`），指标（MA/EMA/MACD…）在反转序列上计算。若 `line` 指标与策略表达式的"当前值取索引"约定不一致，会导致回测与实盘偏离。建议补充一个**真实指标 fixture**（如 MA 金叉）并断言期望信号，以锁定该约定并验证回测保真度。

3. **`GetHistoricalKlines`/`GetHistoricalFundingRates` 缺独立单测**：这两个新函数仅通过 `repository_test.go` 的 `fixtureHistorySource`（非真实 API）间接验证。`binance_source.go` 对真实 Binance 分页/去重/范围过滤（`feature/api/binance/index.go` 中 `seen` 去重 + `CloseTime>endTime` 过滤）建议增加基于 mock client 的单测，确保分页边界与去重正确。

4. **Funding 缺口检测用固定 12h 阈值**：`fundingMissingRanges`（`repository.go:323`）以 12h 为最大可接受间隔；Binance Funding 通常 8h 一次，阈值合理，但非通用（若某标的 cadence >12h 会误判为完整）。建议将阈值作为参数或文档化。

5. **`DefaultManager` 为包级单例 + 进程内 `cancel` map**：`Cancel` 仅对**本进程**内的 run 生效，跨进程（多副本）无法取消。当前单实例假设下可接受，建议在文档/注释中明确该边界。

6. **`markInterrupted()` 重启安全**：进程启动时把 `queued/running` 标记为 `interrupted`（`store.go:159`）——多实例并发启动时，另一个实例真正在跑的 run 会被误标 interrupted。单实例场景下正确，建议备注。

---

## 6. 未验证项 / 人工验收待办

**自动化未覆盖（需人工/集成验证）：**
- 真实 Binance 历史 K 线/Funding API 端到端连通（`wss`/`api` 拉取、分页、last-write-wins 写入全局缓存）。
- 真实指标（MA/EMA/MACD/RSI…）在反转序列上的语义与实盘策略引擎的一致性（见建议 2）。
- 大数据量（百万级 Bar）回测的性能与内存（写入已用 `InsertMulti(500)` 分块；读取走 `open_time` 索引）。
- 前端 `go_binance_futrues_new_ui` 的回测 UI（创建/进度/结果/Equity Curve/Trades/Audit/分组指标/两次对比）——本仓 `static/` 仅构建产物，前端仓库需另行 review。

**人工验收清单：**
- [ ] 真机创建一次回测：`POST /agents/backtests`，确认状态机 `queued→running→succeeded` 且 Trades/Equity/Events 落库。
- [ ] 确认 `GET /agents/backtests/:id/trades|events|equity` 返回完整明细；`POST /agents/backtests/:id/cancel` 在运行中对本进程生效。
- [ ] 确认 `POST /agents/historical-market/import` 需 JWT，且导入后全局 `market_klines_*` 被 last-write-wins 覆盖。
- [ ] 用相同 Dataset Spec 跑两次，中途手动覆写一根 canonical K 线，确认 DatasetID/SpecHash 不变、DataHash 变化。
- [ ] 确认本地已完整的历史区间不再触发 Binance 请求（观察日志/源码路径，或单测已覆盖）。
- [ ] 确认回测结果不写入任何真实策略模板、模拟盘或真实 Binance 账户。

---

## 7. 最终结论

**AUTOMATED PASS / 人工验收待定。**

- 编译、静态检查（仅既有噪声）、竞态测试、全量回归全部通过。
- 7 项 Gate 验收均有对应自动化测试或代码证据支持，全部成立。
- 安全口径（未来函数防护、确定性、与真实账户/策略隔离、外部导入 JWT 鉴权、SQL 注入防护、DB v6→v7 ORM-only 非破坏性迁移、不直接下单、审计可追溯）全部满足。
- 无阻塞级缺陷；6 条非阻塞建议与人工/集成验收待办供后续迭代与联调参考。

> 本报告为纯评审产出，未改动任何源代码、未执行任何写操作、未写入项目/用户内存。
