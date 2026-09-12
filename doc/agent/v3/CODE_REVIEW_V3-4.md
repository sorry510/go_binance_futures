# V3-4 Code Review：Adaptive Resolution Backtest

- **Phase**：V3-4 / Adaptive Resolution Backtest（1m → 1s → trades 自适应精度回放）
- **审查基线**：分支 `feat/ai-agent-v3`，HEAD `cbbbee7`（Merge PR #49），工作区含 V3-4 全部改动（新增 9 个 Go 文件约 2,390 行 + 修改 14 个文件约 940 行）
- **审查类型**：**Code Review（review-only）** —— 仅审查，未修改任何代码
- **审查结论**：**AUTOMATED PASS / 人工验收待定** —— 未发现阻塞级（P0）缺陷；V3-4A / V3-4B 两套 Gate 在代码与自动化测试层面均可验证通过；提出 3 项 P1、6 项 P2/P3 改进建议
- **审查日期**：2026-09-11

---

## 1. 审查范围与方法

1. 以 `doc/agent/v3/04-phase-v3-4-adaptive-resolution-backtest.md`（20KB，含 §10 V3-4A Gate、§11 V3-4B Gate、§9 测试矩阵、§14 实现结果）为唯一需求基线。
2. 逐文件通读新增实现：
   - 数据层：`service/historicalmarket/{public_data,sparse_repository,resolution_provider}.go`
   - 解析层：`service/backtest/{intrabar,adaptive_strategy}.go`
   - 引擎层：`service/backtest/adaptive_engine.go`
   - 并对 `service/backtest/{engine,environment,types,store}.go`、`models/{historical_market,agent_backtest}.go`、`main.go`、`controllers/agent_backtest.go`、`go.mod` 做全量 diff 审查。
3. 用全仓检索交叉验证四条硬约束：**是否引入未来函数**、**是否全量秒级入库**、**高精度 evidence 是否进入 DataHash/Audit**、**Standard 1m 是否与 V3-3 基线等价**。
4. 逆向核对依赖库实现（Beego ORM v2.1.0 索引命名）以判定 `FORCE INDEX (market)` 的 MySQL 兼容性。
5. 运行 `go build ./...`、`go vet ./...`、受影响包 `-race` 测试、全量 `go test -count=1 ./...`。
6. 只读核对独立前端仓库 `go_binance_futrues_new_ui` 是否同步了 V3-4 的 API 契约与 UI 要求。

---

## 2. 自动化验证结果

| 项目                                                                          | 结果                                                        |
| --------------------------------------------------------------------------- | --------------------------------------------------------- |
| `go build ./...`                                                            | ✅ PASS                                                   |
| `go vet ./...`                                                              | ⚠️ 仅既有噪声 `main.go:324/329 unreachable code`（历史禁用 goroutine 遗留，非本 Phase 引入） |
| `go test -count=1 -race ./service/backtest/... ./service/historicalmarket/...` | ✅ 全绿（backtest 2.749s / historicalmarket 2.524s）           |
| `go test -count=1 ./...`                                                    | ✅ 55 个包 ok，无 FAIL                                        |

> 说明：链接期 `ld: warning ... malformed LC_DYSYMTAB` 为 macOS 工具链噪声，不影响测试结果。

---

## 3. 变更清单（关键文件）

| 文件                                                | 变更   | 作用                                                                                     |
| ------------------------------------------------- | ---- | -------------------------------------------------------------------------------------- |
| `service/historicalmarket/public_data.go`         | 新增 629L | Go 原生 Binance Public Data Client：URL Builder / HTTP 下载 / retry / CHECKSUM SHA256 / ZIP+CSV 流式解析 / singleflight / ctx cancel |
| `service/historicalmarket/sparse_repository.go`   | 新增 215L | sparse `market_klines_1s` / `market_trades` 读写（事务 + 分块 + cancel 回滚）                        |
| `service/historicalmarket/resolution_provider.go` | 新增 383L | `ResolutionProvider`：本地 sparse → archive → trades 聚合 fallback；evidence hash 与统计            |
| `service/backtest/intrabar.go`                    | 新增 227L | `PriceEvent` / `IntrabarResolver` / `MergeResolutionEvidenceHash`                        |
| `service/backtest/adaptive_strategy.go`           | 新增 30L  | `DetectROICandidate`：仅用 1m High/Low 推导 ROI 候选区间                                        |
| `service/backtest/adaptive_engine.go`             | 新增 298L | Adaptive 运行态：秒级 ROI 重放、trade 重放、Funding as-of、evidence 累积、统计                              |
| `models/historical_market.go`                     | 修改   | `MarketKline1s`（复用 1m schema）+ `MarketTrade`（唯一键 + trade_time 索引）                         |
| `models/agent_backtest.go`                        | 修改   | Run 增 `resolution_mode/model/stats_json`；Trade 增 `entry/exit_resolution`                 |
| `service/backtest/types.go`                       | 修改   | `StandardEngineVersion=v6` / `AdaptiveEngineVersion=v8`、`ResolutionStats`、`NormalizeResolutionMode` |
| `service/backtest/engine.go`                      | 修改   | `RunWithResolution`；Adaptive 分支接入；funding 计提闭包化                                            |
| `service/backtest/environment.go`                 | 修改   | `BuildIntrabar` + `visibleBars` overlay + `buildIntrabarOverlays` 高周期 partial 重建        |
| `service/backtest/store.go`                       | 修改   | 持久化 resolution 元数据；新增 `TradesPage` / `EventsPage` 分页                                  |
| `controllers/agent_backtest.go`                   | 修改   | trades/events 改为分页响应                                                                     |
| `main.go`                                         | 修改   | `dbVersion 10 → 11`；注册两张新表                                                              |
| `go.mod`                                          | 修改   | `golang.org/x/sync` 由 indirect 提升为直接依赖（singleflight）                                    |

---

## 4. V3-4A Gate 逐条核对（规格 §10）

| # | Gate                                                | 实现证据                                                                                                                                                                 | 状态     |
| - | --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1 | Go 原生 Client 能下载、验证、解析 USD-M 1s/trades archive      | `public_data.go`：`ArchiveURL`（um/daily/klines/trades）、`downloadFile`+retry、`parsePublicDataChecksum`、`zip.NewReader`+`csv.Reader`；8 个 httptest 单测覆盖 URL/校验/解析/404/5xx/singleflight/cancel                     | ✅      |
| 2 | 普通无歧义回测不会下载 1s/trades                              | `adaptive_engine.go:86` `DetectROICandidate.Required=false` 直接返回；`TestAdaptiveV8MatchesStandardV6WhenNoIntrabarResolutionIsNeeded` 断言 `ResolutionStats{}` 零值  | ✅      |
| 3 | 多事件 1m Bar 可被 1s 正确排序                               | `intrabar.go:85-118` 按 `second.OpenTime` 顺序扫描，首个单命中秒即确定；`TestIntrabarResolverUsesSecondsForMinuteConflict`                                                  | ⚠️ 见 I-1 |
| 4 | 同秒冲突可被 trades 正确排序                                 | `intrabar.go:127-140` 按 trade 顺序扫描 + `sortPriceEvents`；`TestIntrabarResolverUsesTradesForSameSecondConflict`、`TestAdaptiveV8TradeReplayUsesTradeIDForSameTimestampCrossing` | ✅      |
| 5 | 所有高精度数据 lazy + sparse，不做全历史秒级入库                   | `StoreSparseSecondBars/StoreSparseTrades` 只写传入 slice；无批量回填入口；`TestSparseSecondRepositoryStoresOnlyRequestedSlice`                                                  | ✅      |
| 6 | 不引入未来函数                                             | `intrabar.go:58` 与 `:92` 用 `KnownAt` 过滤；`resolveWithTrades` 用 `TradeTime < KnownAt` 跳过；`environment.go` overlay 检查 `OpenTime/CloseTime > asOf` 即丢弃；3 个专项测试      | ✅      |
| 7 | Standard 1m 模式结果与 V3-3 基线不变                          | `engine.go` 标准路径执行顺序未变（`advanceFunding` 仍位于 pending 执行之后、Build 之前）；`TestV34AStandardOneMinuteBaselineKeepsV6Semantics` 强断言引擎版本/开平仓时点/成交价/ExitReason/FundingPnL/MarketCondition/funding 事件数 | ✅      |
| 8 | `go test ./...`、相关 race、build、SQLite/MySQL schema Gate | build ✅、race ✅、全量 ✅、SQLite sync ✅；**MySQL 未执行**                                                                                                                     | ⚠️ 见 §8 |

**关键佐证（标准模式零回归）**：`engine.go:141` 的 Adaptive 块受 `mode == ResolutionModeAdaptive` 保护，标准模式下完全跳过；`advanceFunding(bar.CloseTime, bar.Close)`（:163）所处位置与被替换的原 funding 循环一致，且 `fundingIndex` 单调推进保证幂等，故标准模式语义与 V3-3 逐位等价。

---

## 5. V3-4B Gate 逐条核对（规格 §11）

| # | Gate                                                     | 实现证据                                                                                                                                      | 状态 |
| - | -------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | -- |
| 1 | ROI Gate + close strategy 可在候选分钟按秒重放                    | `adaptive_engine.go:101-161`：逐秒重建 partial bar → `BuildIntrabar` → 重算 ROI → `closeGateReason` → 原 `evaluateRules`；`TestAdaptiveV8ClosesOnIntraminuteROIGateWhenRuleBecomesTrue` | ✅  |
| 2 | 1m High/Low 只用于候选筛选，不直接决定 close rule 结果              | `DetectROICandidate` 仅接收/使用 `Bar.High`、`Bar.Low`（`adaptive_strategy.go:16-29`）；最终判定走秒级 Env                                                            | ✅  |
| 3 | 秒级 Env 与 Live 时间语义有明确测试约束                          | `BuildIntrabar` 复刻 Live 动态 Kline；`TestAdaptiveIntrabarEnvironmentUsesOnlyObservedPartialKline` 断言 `NowPrice=partial.Close`、未来 MarketCondition 不可见、未来 1m close 未泄漏              | ✅  |
| 4 | 同秒多事件继续下钻 trade，不猜 High/Low 顺序                          | `evaluateTradeROIClose` 逐 trade 重建 partial 并评估；`TestAdaptiveV8TradeReplayIgnoresFutureTrade`                                                 | ✅  |
| 5 | high-resolution evidence 进入 DataHash 和 Audit               | `adaptive_engine.go:61` `MergeResolutionEvidenceHash`；`engine.go:148` 写 `intrabar_resolution` 审计事件；`TestAdaptiveV8...` 断言 `DataHash` 已变化且事件存在              | ✅  |
| 6 | 无高精度候选时性能应接近 V3-3 1m Engine                            | 候选判定为纯算术，不触碰 Provider/DB/网络                                                                                                              | ✅  |
| 7 | 不为了 Adaptive 模式预下载整段 1s/trades                          | `repository.go`（prefetch 路径）**未改动**；下载仅发生在 `SecondBars`/`Trades` 调用点                                                                        | ✅  |

---

## 6. 安全属性专项

### 6.1 未来函数防护（本 Phase 最高风险项）

四层防护，全部有代码与测试对应：

1. **事件层**：`KnownAt > minute.CloseTime` 直接剔除（`intrabar.go:58`）；秒级 `KnownAt > second.CloseTime` 跳过（:92）；trade 级 `TradeTime < event.KnownAt` 跳过（:130）。
2. **数据可见性层**：`visibleBars` 用 `sort.Search(CloseTime > asOf)` 取 completed-only，overlay 若 `CloseTime > asOf` 即丢弃。
3. **高周期重建层**：`buildIntrabarOverlays` 只聚合 `OpenTime < partialMinute.OpenTime` 的已完成 1m bars，再拼接 partial；`partial.CloseTime` 恒等于 `asOf`。
4. **Funding 层**：`advanceFunding(until, ...)` 只计提 `FundingTime <= until`；同秒内先 trade 重放再 1s 评估（`adaptive_engine.go:117-125`），并有注释说明动机；`TestAdaptiveV8TradeReplayDoesNotSeeLaterFundingInSameSecond` 精确验证。

### 6.2 lazy + sparse

- 无任何"全区间下载/入库"入口；`SecondBars`/`Trades` 均以单分钟/单秒为请求粒度。
- 入库走事务 + 分块，`ctx.Err()` 检查点位于分块边界与 CSV 行扫描（每 1024 行），cancel 时 `Rollback` 不留半截数据（`TestStoreSparseSecondBarsRollsBackOnCancelBetweenChunks`、`TestStoreSparseTradesRollsBackOnCancelBetweenChunks`）。
- 预下载路径未受影响（`repository.go` 未改动）✓ 符合规格 §7「不允许因此全范围下载 1s/trades」。

### 6.3 DataHash / Evidence 合并

- `MergeResolutionEvidenceHash`：evidence hash 去重 + 排序后与 base 拼接再 SHA256 → **与 evidence 顺序/重复无关，确定性**（`TestMergeResolutionEvidenceHashIsStableAndEvidenceSensitive`）。
- `EvidenceHash` 基于行数据（含 `SourceRef`，其携带 `#sha256=<archive hash>`）→ archive 校验和变化会导致 hash 变化，满足测试矩阵「Archive checksum 改变：最终 DataHash 必须改变」。
- 未消费任何高精度数据的 Adaptive Run，DataHash 与标准模式**完全相同**（`TestAdaptiveV8MatchesStandardV6...` 断言）✓。

### 6.4 成交与成本一致性

- Adaptive 平仓复用 `closePosition`（fee/slippage/PnL 单一实现），未复制第二套计算 ✓（规格 §6.4 要求）。
- 秒级成交价取 `second.Close`，trade 级取 `trade.Price`，随后统一走 `applySlippage` ✓。
- Limit 排队/部分成交深度**未模拟** —— 与规格 §5.4、§12 一致（首版简化模型，已在文档声明需写入 Run metadata；见 I-10 注记）。

---

## 7. 测试矩阵覆盖核对（规格 §9）

| 矩阵项                              | 覆盖测试                                                                                    | 状态 |
| ------------------------------- | -------------------------------------------------------------------------------------- | -- |
| 1m 无歧义：0 次 1s/trade 读取            | `TestAdaptiveV8MatchesStandardV6WhenNoIntrabarResolutionIsNeeded`                       | ✅  |
| 1m 同时触达 TP + SL：1s 决定先后         | `TestIntrabarResolverUsesSecondsForMinuteConflict`                                      | ✅  |
| 1s 同时触达两个事件：trades 决定先后         | `TestIntrabarResolverUsesTradesForSameSecondConflict`                                   | ✅  |
| trade_time 相同：trade_id 稳定 tie-break | `TestPublicDataParseTradesOrdersByTimeThenID`、`TestAdaptiveV8TradeReplayUsesTradeIDForSameTimestampCrossing` | ✅  |
| 1s archive 缺失：trades 聚合目标分钟 1s   | `TestResolutionProviderSecondBarsFallsBackToTrades`                                     | ✅  |
| CHECKSUM 错误：fail closed，不导入      | `TestPublicDataChecksumMismatchFailsClosed`                                             | ✅  |
| Archive 404：进入 fallback，不无限重试    | `TestPublicData404DoesNotRetry`                                                         | ✅  |
| 同一 `{kind,symbol,date}` 并发只下载一次 | `TestPublicDataSingleflightDeduplicatesArchiveDownload`                                 | ✅  |
| Cache hit：重复回测不再访问公网            | `TestResolutionProviderSecondBarsUsesCompleteSparseCache`、`...DownloadsVerifiedDailyKlines` 二次命中 | ✅  |
| Cancel：不写半截数据                    | `TestPublicDataDownloadHonorsContextCancel`、两个 `RollsBackOnCancelBetweenChunks`            | ✅  |
| Archive checksum 改变：DataHash 改变   | `TestMergeResolutionEvidenceHashIsStableAndEvidenceSensitive`                           | ✅  |
| 未来 1s/trade 不可被当前秒读取             | `TestAdaptiveV8TradeReplayIgnoresFutureTrade`、`TestIntrabarResolverIgnoresFutureEvent`   | ✅  |
| ROI Gate 触线但 rule=false：不得平仓      | `TestAdaptiveV8ROIGateDoesNotForceCloseWhenRuleIsFalse`、`TestBacktestROIGateDoesNotForceCloseWhenRuleIsFalse` | ✅  |
| ROI Gate 触线且 rule=true：按首次成立时刻   | `TestAdaptiveV8ClosesOnIntraminuteROIGateWhenRuleBecomesTrue`（断言 ExitTime=秒边界）            | ✅  |
| MarketCondition 只用当前时间之前的值       | `TestHistoricalEnvironmentUsesLatestVisibleMarketCondition` + `TestAdaptiveIntrabarEnvironmentUsesOnlyObservedPartialKline` | ✅  |
| Funding 不重复计提                    | `TestV34AStandardOneMinuteBaselineKeepsV6Semantics`（断言 funding 事件恰 1 次）、`TestAdaptiveFundingAfterIntraminuteCloseIsNotAppliedEarly` | ✅  |
| LONG / SHORT 两侧触发价分别测试            | `TestDetectROICandidateLongAndShort`                                                    | ✅  |
| 高精度 PnL 与统一 fee/slippage 逻辑一致     | 代码复用 `closePosition`；无并行实现                                                              | ✅  |
| 网络集成测试不进入普通 `go test`             | 全部使用 `httptest`，无公网依赖                                                                   | ✅  |

**覆盖度结论：测试矩阵 19 项均有对应实现证据，断言质量高（非空断言）。** 缺口见 I-2、I-5。

---

## 8. 数据库迁移 10 → 11

- `main.go:37` `dbVersion = 11`；`registerModels()` 新增 `MarketKline1s`、`MarketTrade`。
- `market_klines_1s` 通过 `type MarketKline1s MarketKline1m` 复用列定义，并单独提供 `TableName()` / `TableUnique()`（Go 新类型不继承方法，此处正确补齐）。
- `command/db_update_test.go` 新增断言：`config.Version == 11`、两张表存在、`market_trades(market,symbol,trade_time)` 索引存在、`agent_backtest_runs` 三个新列存在、`agent_backtest_trades` 两个新列存在、二次 sync 幂等 ✅。
- 迁移为**纯新增**（两张新表 + 追加列），不改动既有列语义；回退到 v10 时新表成为孤儿表，不阻塞旧版本。
- **`FORCE INDEX (market)` MySQL 兼容性已排除风险**：Beego ORM v2.1.0（`client/orm/models.go:502`）对 `TableUnique()` 生成**匿名 `UNIQUE (market, symbol, ...)`** 内联约束，MySQL 对匿名唯一索引以首列名命名，故 `market_klines_1s` 与既有 `market_klines_*` 均具备名为 `market` 的索引 ✓。
- **但 MySQL 环境的实际 sync 与查询执行仍未验证**（见 §11）。

---

## 9. 文档 ↔ 实现一致性

| 规格条目                                        | 实现                                        | 判定       |
| ------------------------------------------- | ----------------------------------------- | -------- |
| §5.6 建议 V3-4A 使用 `backtest_engine_v7`       | 未产出 v7，Adaptive 统一为 `v8`                  | ⚠️ I-4  |
| §6.5 建议 V3-4B 使用 `backtest_engine_v8`       | ✅ `AdaptiveEngineVersion = v8`             | ✅       |
| §5.6 示例 `adaptive_intrabar_v1`               | 实现为 `adaptive_intrabar_v2`（规格用"例如"措辞，非硬约束） | ✅       |
| §5.2 `ResolutionProvider` 四方法（含 MarkPriceSeconds） | 实现为 `MinuteBars/SecondBars/Trades/Stats/Close`，**无 `MarkPriceSeconds`**（规格标注"可选/liquidation 专用"） | ✅ 可接受  |
| §6.2 要求先审计 Live `InitParseEnv` 时间语义（V3-4B-0） | 实现以 `BuildIntrabar` 复刻 Live 动态 Kline，并有测试约束；**未见独立的 V3-4B-0 审计记录** | ⚠️ I-1 相关 |
| §5.4 Limit 简化模型"必须写入 Run metadata"          | `ResolutionModel` 已记录（`adaptive_intrabar_v2`），但**未记录 Limit 排队/部分成交假设** | ⚠️ I-10 |
| §7 UI 增加精度模式 + 统计 + badge + evidence 展开    | 前端仓库 `backtest.vue`（+274 行）已实现模式选择/统计/badge；本仓提供 API | ✅       |
| §7 现有"获取历史数据"不得全量下载 1s/trades            | `repository.go` 未改动                        | ✅       |
| §14 实现结果描述                                  | 与代码一致（唯一遗漏见 I-1/I-3）                      | ⚠️       |

---

## 10. 发现的问题

### P1（建议人工验收前处理）

**I-1｜`IntrabarResolver` / `PriceEvent` 在生产链路零调用（A 阶段"接入引擎"未落地）**

- 证据：`intrabar.go` 中仅 `MergeResolutionEvidenceHash` 被生产代码引用（`adaptive_engine.go:61`）；`IntrabarResolver.Resolve`、`resolveWithTrades`、`eventTouched`、`eventSatisfied`、`sortPriceEvents` 仅出现在 `intrabar_test.go`。
- 规格 §2 Level 2 / §5.3–§5.4 / §8 V3-4A-5 均要求把 Hard TP/SL/Limit 等确定性事件抽象为 `PriceEvent` 并接入引擎。实现中 Adaptive 引擎只做 **ROI Gate 的秒级重算**，未消费 `IntrabarResolver`。
- 客观背景：V3-3 回测模型不存在"硬止盈止损单"对象（规格 §3 明确指出 ROI Gate 不是触价即成交），引擎侧确实缺少可接入的真实事件源，A 阶段能力更像被 B 阶段合并取代；文档 §14 亦只描述 ROI 路径。
- 影响：A Gate 第 3 条（多事件 1m Bar 被 1s 正确排序）仅在单元测试层面成立，**引擎层面未生效**；`IntrabarResolver` 属"已实现未启用"能力。
- 建议：在文档中明确"V3-4A IntrabarResolver 作为独立能力保留、首版未接入引擎"，或将 §5/§8 的 A 阶段描述收敛为实际实现范围，消除歧义。

**I-2｜`ResolutionStats.Unresolved` 恒为 0（字段未填充）**

- 证据：`types.go:219` 定义该字段，前端 `BacktestResolutionStats.unresolved` 已消费，规格 §5.6 要求记录"unresolved / conservative fallback 次数"，但全仓**无任何 `Unresolved++`**。
- 建议：在 `evaluateROIClose` 返回未 resolved（秒级扫描完仍未命中，或 trade 重放未找到 crossing）时递增；若暂不使用，应在文档标注为保留字段。

**I-3｜`/trades`、`/events` 响应改为分页对象（破坏性 API 变更，文档未记录）**

- 证据：`controllers/agent_backtest.go` 由 `data: [...]` 改为 `data: {list, total, page, limit}`，默认 limit 由 5000/10000 降为 20/50。
- 前端已同步适配（`src/api/backtest.ts` 增加 `params` 与 `BacktestResolutionStats`；`backtest.vue:385/415` 读取 `res.data.list/total`）✓
- 但规格 §7 只要求"结果页增加统计展示"，未要求分页化；文档 §14 也未记录 → 属**范围外改动 + 文档↔实现不一致**，对未同步的第三方调用方是破坏性变更。
- 建议：在文档 §7/§14 补充说明该 API 变更与分页语义。

### P2 / P3

| #    | 级别 | 问题                                                                                                                                    | 建议                                                              |
| ---- | -- | ------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| I-4  | P2 | 文档 §5.6 建议 A 用 v7、§6.5 建议 B 用 v8，实现统一 v8，文档内部不一致                                                                                     | 统一为 v8 并注明 A/B 合并                                               |
| I-5  | P2 | `DetectROICandidate` 未覆盖"阈值关闭（`StopLossPct/TakeProfitPct <= 0`）"场景；`closeGateReason` 用 `DisabledFuturesROIThreshold=1e6` 兜底（有测试固化），二者当前行为等价（仅 ROI 达 1e6% 时理论差异，实际不可达） | 补一条对齐测试，防止未来语义漂移                                                |
| I-6  | P2 | `secondRangeComplete` 要求请求区间**每一秒**都有 1s bar；若 Binance 1s archive 对无成交秒不产出记录（稀疏），该判定永不成立 → 每次下钻都退回 trades 聚合并下载 trades 大文件（正确性不受损，但"1s 优先"可能退化为"trades 优先"） | 公网实测一次，记录 1s archive 稀疏性与 `ArchiveDownloads` 实际次数；必要时放宽完整性判定条件 |
| I-7  | P3 | `FetchArchive` 用 `singleflight.DoChan`，合并请求共享**首个调用者的 ctx**；首位取消会导致其余等待者一并失败（singleflight 既有语义，`.part` 已清理，不会污染缓存）                | 在注释中显式说明该语义                                                     |
| I-8  | P3 | `cachedArchive` 命中时只 `os.Stat` 校验存在性，不重算 SHA256                                                                                       | 说明"内存缓存仅在单次 Run 内有效、Run 结束即清理"即可；如需更强保证可在命中时校验                    |
| I-9  | P3 | `validateSparseTrade` 与 `sparseTradeRangeCached` 只校验 `ArchiveSHA256` 长度为 64，未校验合法 hex                                                 | 增加 `hex.DecodeString` 校验                                         |
| I-10 | P3 | 规格 §5.4 要求"Limit 触达即成交的简化假设必须写入 Run metadata"，实现仅在 `ResolutionModel` 记录模型名，未显式记录该假设                                                        | 在 Run metadata 中增加 assumptions 描述                               |

---

## 11. 未验证项 / 审查边界

1. **MySQL schema / sync Gate 未执行** —— 文档 §14 已声明。`FORCE INDEX (market)` 的兼容性已通过 Beego 源码推导排除风险，但 MySQL 实例上的实际建表、索引与查询仍需 `./go_binance_futures sync db` 后实测。
2. **真实 Binance Public Data 公网行为未验证** —— 全部测试基于 `httptest` 模拟（符合规格"普通 go test 不依赖公网"）。1s archive 的稀疏性、trades 归档体量、CHECKSUM 文件真实格式均未实测（见 I-6）。
3. **前端构建与端到端联调未验证** —— 本仓只做后端 review；前端仓库 `go_binance_futrues_new_ui` 工作区改动已就位（`api/backtest.ts`、`backtest.vue`、`locales/*.yaml`），未执行 `pnpm build`，也未做接口联调。
4. **`Unresolved` 统计无实际数据**（见 I-2）。
5. **长区间 Adaptive 性能未验证** —— 无 drill-down 占比、单次 Run 耗时、下载总量的实测数据。
6. **未验证 1w/1M 等高周期 overlay 在边界日期（周/月切换）的对齐正确性** —— `intervalWindowStart` 逻辑已人工推演正确（周一为周起点），但无测试覆盖。

---

## 12. 人工验收待办

1. **标准基线回归**：同一 dataset 分别以 `standard_1m` / `adaptive` 运行，确认无候选分钟时 `Trades`、`Metrics`、`DataHash` 完全一致。
2. **MySQL sync Gate**：执行 `./go_binance_futures sync db`，确认 `market_klines_1s`、`market_trades` 建表成功、`market` 索引存在、二次执行幂等；并在 MySQL 上跑一次含 drill-down 的回测，确认 `FORCE INDEX (market)` 不报错。
3. **真实公网 drill-down**：对一段含大波动分钟的历史区间跑 Adaptive，记录 `second_drilldown_minutes`、`trade_drilldown_seconds`、`archive_downloads`、`download_bytes`，核实 1s 优先路径是否生效（I-6）。
4. **取消与恢复**：运行中取消 Adaptive Run，确认无残留 `.part`/半截 sparse 数据；重启后重复回测命中本地 sparse，不重复下载。
5. **未来函数抽查**：抽查一笔 Adaptive 平仓的 `intrabar_resolution` 审计事件，确认 `hit_time`、`evidence`（URL+SHA256）与实际 1s/trade 数据吻合，且不使用该时刻之后的数据。
6. **API 契约**：确认前端 trades/events 分页展示、模式选择、resolution badge、统计面板均正常（I-3）。
7. **Funding 边界**：构造 funding 恰好落在平仓秒的场景，确认 funding 不被提前或重复计提。
8. **UI 范围**：确认"获取历史数据"不会因 Adaptive 而下载 1s/trades。

---

## 13. 结论

**AUTOMATED PASS / 人工验收待定。**

V3-4 的核心设计目标"以 1m 为主时间轴、仅在歧义或 ROI Gate 可能触发时按需下钻 1s/trades"在实现中**完整落地且可验证**：

- **标准模式零回归**：`standard_1m` 代码路径与 V3-3 逐位等价，并有强断言测试固化（引擎版本 v6、开平仓时点、成交价、FundingPnL、funding 事件计数、JSON 全等）。
- **未来函数防护严密**：事件/数据可见性/高周期重建/Funding 四层均有代码约束与专项测试，"未来 1s/trade 不可被当前秒读取"成立。
- **lazy + sparse 贯彻到位**：无全区间下载或入库入口，预下载路径未改动，cancel 回滚有测试保证。
- **证据链完整**：高精度 evidence 经排序去重后合并进 DataHash，`intrabar_resolution` 审计事件含分辨率、ROI、gate_reason 与 evidence，Trade 记录 entry/exit resolution。
- **工程质量高**：新增约 2,390 行实现配 60+ 测试用例，断言具体（非空断言）；`go build`、`go vet`（仅历史噪声）、受影响的 `-race` 测试与全量 55 包测试全部通过。

**无阻塞级（P0）缺陷。** 三项 P1 中，I-1（IntrabarResolver 未接入）与 I-3（分页 API 变更未记录）本质是**文档↔实现一致性**问题而非运行风险，I-2（Unresolved 未填充）是一处**统计字段缺失**。建议在人工验收前先澄清 I-1 的文档表述、补齐 I-2 的计数或标注保留、并在文档中记录 I-3 的 API 变更。

**放行建议**：可进入人工验收。待 §12 的 8 项用例通过、且在 MySQL 环境完成一次 sync Gate 与一次真实公网 drill-down 验证后，V3-4 可判定为交付完成。


---

## 14. 复核处理记录（2026-09-11）

本节为原 review 之后的修复/判定记录，不改写前述审查结论。

- **I-1**：不改生产逻辑。已在 Phase 文档明确 `IntrabarResolver/PriceEvent` 是硬 TP/SL/Limit 的预留能力；当前生产 Adaptive 只接入 ROI Gate / close strategy 高精度重放。
- **I-2**：不按原建议递增。当前只有 strict policy；高精度数据缺失直接使 Run 失败，rule=false 是确定性“不平仓”，均不应计为 unresolved。Phase 文档已明确成功 Run 当前该字段保留为 0。
- **I-3**：已在 Phase 文档补充 trades/events 分页 API 契约：`{list,total,page,limit}`，默认 20/50。
- **I-4**：已统一文档为 Adaptive A/B 最终共用 `backtest_engine_v8`，Standard 保持 V6。
- **I-5**：已补 `StopLossPct/TakeProfitPct <= 0` 不触发 Adaptive candidate 的测试。
- **I-6**：已真实公网复核。USD-M Futures `BTCUSDT` 的 `1m` daily archive 可用，但抽查多个日期 `1s` daily archive 均为 404；当前 Provider 会 fallback 到 daily trades 并聚合目标分钟 1s。PublicDataClient 新增单次 Run 内 404 negative cache，避免同一不存在 archive 被重复请求。
- **I-7**：已补 singleflight 首调用者 ctx 共享语义注释；取消不会污染缓存。
- **I-8**：不改。archive cache 只在单次 PublicDataClient/Run 生命周期内有效，且首次写入前已经 SHA256 fail-closed 校验。
- **I-9**：已增加 archive SHA256 的合法 hex 校验，并补测试。
- **I-10**：不新增 schema。当前生产模型无独立 Limit 挂单/部分成交对象；Phase 文档明确未来真正接入 Limit 时再要求显式 Run metadata。

另外修复了 review 后人工测试发现的两项回归：Standard 1m `visibleBars` O(n²) 全历史复制导致长回测从秒级退化到 20+ 分钟，以及详情 Drawer 重建后 ECharts 仍绑定旧 DOM 导致 Chart 空白。 后续 Adaptive 实测又修复了两类 ZIP 重复解析：sparse trades 父范围现可覆盖秒级子范围；同一 Run 内 daily 1s/trades archive 改为前向流式 scanner，不同候选分钟不再从 ZIP 起点重复解压。

后续 Adaptive 实测又发现并修复两项性能问题：① 分钟级 verified trades 的 `range` coverage 未被秒级子请求复用，导致每个候选秒反复从 daily ZIP 起点解压/扫描；现改为父范围可覆盖子范围，并有 0-network / `TradeCacheHits` 集成测试。② `BuildIntrabar` 的 overlay/高周期构造会复制或扫描从回测起点到当前的完整历史；现改为二分定位当前窗口，指标输入只复制最后 200 根。真实 BTCUSDT 同策略复现区间从 10.24s 降至 4.94s，结果/DataHash 不变。
