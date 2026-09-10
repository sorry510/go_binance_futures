# V3-3 文档 ↔ 实现一致性核查（更新版）

- **Phase**：V3-3 / Historical Backtest Engine
- **核查基线**：HEAD `41e05b4`（`feat: ai agent v3-3 update`，含 `2846df0`/`39cb85d` 两个 fix 提交），工作区干净（无未提交/未跟踪改动）
- **核查类型**：**文档描述 ↔ 代码实现 一致性/完整性核查**
- **核查模式**：review-only（仅检查，未修改任何代码、未提问）
- **核查结论**：**规格文档描述的功能已全部实现，未发现缺失项**（AUTOMATED PASS / 人工验收待定）
- **核查日期**：2026-09-10

> 说明：本文件上一版针对旧规格（Price% 止盈止损、无 prefetch/backfill/delete）。本次规格与实现均已更新，本版为**当前基线的一致性核查**，覆盖新增/变更条目。

---

## 1. 核查方法

1. 以 `doc/agent/v3/03-phase-v3-3-backtest.md`（当前版）为唯一需求基线，逐条拆解为可验证项。
2. 在实现代码中定位对应证据（模型 / 服务 / 引擎 / 控制器 / 路由 / 测试）。
3. 运行 `go build ./...`、`go vet ./...`、`go test -count=1 -race <受影响包>`、`go test -count=1 ./...` 验证代码健康。
4. 对前端 UI 条目，核对独立前端仓库 `go_binance_futrues_new_ui` 的页面与 API 层（本仓 `static/` 仅为构建产物）。

---

## 2. 规格 ↔ 实现 逐条对照

### 2.1 核心能力（规格 L11-16）

| # | 规格要求                                             | 实现证据                                                                                                                                                       | 状态 |
| - | ------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | -- |
| 1 | 历史 K 线 + Funding 驱动策略逐 Bar 确定性执行                 | `service/backtest/engine.go` `RunWithProgress` 主循环                                                                                                         | ✅  |
| 2 | 支持 indicator warmup，严格禁止未来函数                     | `DefaultWarmupBars=200`；`environment.series/tickerStats` 用 `sort.Search(asOf)` 只取 `CloseTime ≤ asOf`；`TestHistoricalEnvironmentCannotSeeFutureBars`        | ✅  |
| 3 | LONG / SHORT、策略平仓、止盈、止损                          | `engine.go` 开平仓分支 + `closeGateReason` ROI 门槛                                                                                                               | ✅  |
| 4 | 双边手续费、Funding、可配置滑点                              | 开仓 `notional*FeeRate`、平仓 `\|qty\|*fill*FeeRate`；`applySlippage`；Funding 多空方向；`TestBacktestFeesAndSlippage`/`TestBacktestFundingLongPaysPositiveRate`       | ✅  |
| 5 | 记录 signal/order/fill/position、Trade、Equity Curve | `AgentBacktestEvent`(type: signal/order/fill/position)、`AgentBacktestTrade`、`AgentBacktestEquityPoint`                                                     | ✅  |
| 6 | 输出 11 项指标并按 **LONG/SHORT** 分组                    | `metrics.go` 计算 NetPnL/Return/MaxDD/WinRate/ProfitFactor/Sharpe/Sortino/TradeCount/Fees/Funding/AvgHolding；`Metrics.BySide`（`ByMarketCondition` 已移除，与规格一致） | ✅  |


### 2.2 Historical Market Repository（规格 L20-26）

| #  | 规格要求                                                                | 实现证据                                                                                                                                                          | 状态 |
| -- | ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | -- |
| 7  | Kline 按 interval 物理分表 15 张（1m…1mo，含 `1M`→`1mo`）                     | `models/historical_market.go` + `types.go` `intervalTables` 白名单                                                                                               | ✅  |
| 8  | 上层只能通过 `service/historicalmarket` 访问，不得自行拼动态表名                      | 全仓 grep `market_klines_` 仅出现在 `service/historicalmarket`、模型 `TableName` 与测试断言中                                                                                | ✅  |
| 9  | 唯一键 `(market,symbol,open_time)` + last-write-wins UPSERT            | `TableUnique` 三字段；SQLite `ON CONFLICT DO UPDATE` / MySQL `ON DUPLICATE KEY UPDATE`；`TestRepositoryLastWriteWins`                                              | ✅  |
| 10 | Funding 存 `market_funding_rates`，唯一键 `(market,symbol,funding_time)` | 模型 `TableUnique` + `buildFundingUpsert`                                                                                                                       | ✅  |
| 11 | `market_data_import_batches` 记录补缺/外部导入审计                            | `models.MarketDataImportBatch` + `Repository.Import` 写入批次                                                                                                     | ✅  |
| 12 | 本地优先；完整不调 Binance；缺数据只请求缺口                                          | `Repository.LoadKlines` 先查本地→`missingKlineRanges`→仅按 gap 拉取→回写；`TestRepositoryReusesCompleteLocalRange`(0 次调用)、`TestRepositoryFetchesOnlyInternalGap`(精确 1 次) | ✅  |
| 13 | 外部 Kline/Funding canonical import API                               | `POST /agents/historical-market/import` + `controllers/agent_historical_market.go`                                                                            | ✅  |

### 2.3 Dataset 与可追踪性（规格 L30-36）

| #  | 规格要求                                               | 实现证据                                                                                                                                                                           | 状态 |
| -- | -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -- |
| 14 | Dataset 不复制历史 Bar/Funding，只存查询规格                   | `DatasetSpecHash` 仅含 Market/Symbol/Interval/Intervals/Start/End/Warmup；`TestManagerPersistsDeterministicBacktestWithoutLegacyPaperTables` 断言无 `agent_backtest_bars/funding` 旧表 | ✅  |
| 15 | `dataset_spec_hash` 相同规格复用同一 Dataset ID            | `saveDataset` 按 `dataset_spec_hash` 查重复用                                                                                                                                       | ✅  |
| 16 | `data_hash` 为本次 Run 实际读取内容指纹                       | `DatasetDataHash` 哈希 spec + 全量 bars（key 排序后逐 bar）+ funding                                                                                                                     | ✅  |
| 17 | 同 Dataset 重跑允许结果不同，以最新 canonical 为准                | Dataset 不缓存数据，每次 Run 重新从全局仓库读取                                                                                                                                                 | ✅  |
| 18 | 可用 `data_hash` 判断是否由历史数据变化导致                       | `TestSameDatasetSpecUsesLatestCanonicalMarketData`                                                                                                                             | ✅  |
| 19 | Strategy 快照绑 `StrategyVersion`，执行绑 `EngineVersion` | `AgentBacktestRun.StrategyVersion`/`EngineVersion`；`EngineVersion="backtest_engine_v1"`                                                                                        | ✅  |


### 2.4 执行与时间语义（规格 L40-51）— 本轮主要变更区

| #  | 规格要求                                                                                   | 实现证据                                                                                                                                                                                    | 状态 |
| -- | -------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -- |
| 20 | 回放主轴固定 `1m`，API/UI 不可选 Execution Interval                                              | `types.go:12 ReplayInterval="1m"`；`StartRequest` 已无该字段；`store.go:77` 强制 `ExecutionInterval: ReplayInterval`；`store_test.go:92` 断言；前端 `StartBacktestRequest` 亦无该字段                       | ✅  |
| 21 | Engine 运行期不访问 Binance/WS/`symbols`/当前 MarketCondition                                  | 引擎全程只消费 `dataset.Bars`/`dataset.Funding`/`dataset.MarketConditions`                                                                                                                     | ✅  |
| 22 | 每个 Bar 只允许 `close_time ≤ 当前 Bar close_time` 数据                                         | `environment.go` `sort.Search(asOf)`（`series`/`tickerStats`/`marketConditionAt` 均按此界）                                                                                                   | ✅  |
| 23 | Bar close 出 signal，最早下一根 Bar open 成交                                                   | signal 记于 `bar.CloseTime`，`pending` 在下一次迭代以 `bar.Open` 成交；`TestBacktestLongUsesNextBarOpen`                                                                                             | ✅  |
| 24 | **单仓位模式**：同时最多一个 Position，有仓位只评估 `close_long`/`close_short`                            | `engine.go:128`（有仓位→只评估 close）与 `:142`（无仓位→才评估 open）互斥分支                                                                                                                                | ✅  |
| 25 | 先平后才能再开；不支持加仓/金字塔/并行持仓/多空同持/反手                                                         | 同上互斥 + 无 add/reduce 分支                                                                                                                                                                  | ✅  |
| 26 | 平仓后下一根 Bar 收盘可出新开仓 signal、再下一根 open 成交                                                 | 平仓于 Bar N+1 open → `position=nil` → 同 Bar N+1 close 可出 open signal → Bar N+2 open 成交                                                                                                    | ✅  |
| 27 | SL/TP 为**杠杆后持仓 ROI 门槛**，公式 `unrealizedPnL/(\|qty\|*mark)*leverage*100`，手续费/Funding 不参与 | `utils.FuturesLeveragedROI`（注释明确与线上 futures loop 同口径、排除费用）；`environment.go` `env["ROI"]=grossROI(...)`                                                                                  | ✅  |
| 28 | 未越门槛不评估平仓规则；越门槛且规则 true 才平；`0`=关闭（等价 1,000,000%）                                       | `closeGateReason`：`0/负`→`utils.DisabledFuturesROIThreshold`；未越门槛返回 `""`→`engine.go:131` 不评估；`ok` 才产生 pending；`utils/futures_roi.go:5 = 1_000_000.0` + `TestDisabledFuturesROIThreshold` | ✅  |
| 29 | 不用 Bar High/Low 模拟瞬时触发，Bar close 算 ROI、下一根 open 成交                                     | 旧 `protectiveExit`（High/Low 触发）已移除；`TestBacktestROIGateDoesNotForceCloseWhenRuleIsFalse`                                                                                                | ✅  |
| 30 | 末 Bar 有仓位以 `end_of_data` 确定性平仓                                                         | `engine.go:174-194`                                                                                                                                                                     | ✅  |
| 31 | 不用实时 MarketCondition；含 MC 的策略读 `market_condition_histories` 可见最近值；覆盖不足拒绝回测             | `marketConditionAt`（`sort.Search` 取 `Time ≤ asOf` 最近一条，不读未来）；`store.go:69-73` 先 `ValidateMarketConditionCoverage`；`models/market_condition_history.go`                                  | ✅  |


### 2.5 API / UI（规格 L55-65）

| #  | 规格要求                                                                                               | 实现证据                                                                                                                                                      | 状态                         |
| -- | -------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------- |
| 32 | `GET/POST /agents/backtests`                                                                       | router L46 + `List`/`Start`                                                                                                                               | ✅                          |
| 33 | `GET /agents/backtests/:id`                                                                        | router L47 + `Get`                                                                                                                                        | ✅                          |
| 34 | `DELETE /agents/backtests/:id`（删 Run+子表；无其它引用时删 Manifest；**不删全局缓存**）                               | router L47 `delete:Delete`；`store.go:199-272` 事务删除 equity/events/trades→run→`refs==0` 时删 manifest；无 `market_*` 删除操作                                       | ✅                          |
| 35 | `POST /agents/backtests/prefetch`（目标 Symbol 1m + 指标周期 Kline + Funding，含 warmup，**不再获取 benchmark**） | `prefetch.go` `PrefetchPlan`/`Prefetch`；无 benchmark 循环                                                                                                    | ✅                          |
| 36 | `GET /agents/backtests/prefetch/:jobId`                                                            | router L43 + `PrefetchStatus`                                                                                                                             | ✅                          |
| 37 | `POST /agents/backtests/market-condition/backfill`（BTC/ETH 1h 自上线至今 + 逐小时确定性推断）                    | `market_condition_backfill.go`：从各自最早 K 线加载 1h 至最近收盘小时；`inferHistoricalMarketConditions` 按 CloseTime 对齐 + 24h 滚动涨跌/波动推断                                    | ✅                          |
| 38 | `GET .../backfill/:jobId`（进度；同小时已有记录直接跳过不覆盖）                                                       | `manager` 进度字段；`occupied` 按小时桶 `skipped++` 不覆盖；`InsertMulti(500)`                                                                                         | ✅                          |
| 39 | `POST /agents/backtests/:id/cancel`                                                                | router L48 + `Cancel`                                                                                                                                     | ✅                          |
| 40 | `GET /agents/backtests/:id/trades\|events\|equity`                                                 | router L49-51                                                                                                                                             | ✅                          |
| 41 | `POST /agents/historical-market/import`                                                            | router L41                                                                                                                                                | ✅                          |
| 42 | Web UI：创建任务、手动"获取历史数据"、进度、结果详情、Equity Curve、Trades、Audit Events、按 Side 分组、两次对比                     | 前端仓库存在 `src/views/ai/backtest.vue` + `src/api/backtest.ts`（含 prefetch/backfill/delete/trades/events/equity 全部接口，且请求体无 `execution_interval`、指标仅 `by_side`） | ✅（API 层对齐；页面交互细节需前端仓库人工确认） |

### 2.6 验收 Gate（规格 L69-75）

| #  | Gate                                                             | 覆盖测试                                                                                                                                                                                                                                                  | 状态 |
| -- | ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -- |
| 43 | LONG/SHORT/**止盈止损 ROI Gate**/无交易；验证"越过门槛才评估、规则 false 不强平、0 表示关闭" | `TestBacktestLongUsesNextBarOpen`、`TestBacktestShort`、`TestBacktestTakeProfit`(ROI)、`TestBacktestStopLoss...`(ROI)、`TestBacktestROIGateDoesNotForceCloseWhenRuleIsFalse`、`TestCloseGateZeroThresholdsMatchLiveDisabledDefaults`、`TestBacktestNoTrade` | ✅  |
| 44 | 手续费 / Funding / 滑点单测                                             | `TestBacktestFeesAndSlippage`、`TestBacktestFundingLongPaysPositiveRate`                                                                                                                                                                               | ✅  |
| 45 | 明确验证未来 Bar 不可见                                                   | `TestHistoricalEnvironmentCannotSeeFutureBars`                                                                                                                                                                                                        | ✅  |
| 46 | 同输入重放确定性                                                         | `TestBacktestReplayIsDeterministic`                                                                                                                                                                                                                   | ✅  |
| 47 | 同 Spec 覆写后 Dataset ID/SpecHash 不变、`data_hash` 变                  | `TestSameDatasetSpecUsesLatestCanonicalMarketData`                                                                                                                                                                                                    | ✅  |
| 48 | 完整本地不请求 Binance、只补缺口                                             | `TestRepositoryReusesCompleteLocalRange`、`TestRepositoryFetchesOnlyInternalGap`                                                                                                                                                                       | ✅  |
| 49 | 不修改真实策略 / 模拟盘 / Binance 账户                                       | `Manager.run` 只读模板快照，仅写 `agent_backtest_*` 与全局 `market_*`；无下单路径                                                                                                                                                                                       | ✅  |

### 2.7 本阶段不做（规格 L79-82）

| #  | 约束                               | 核查                                     | 状态 |
| -- | -------------------------------- | -------------------------------------- | -- |
| 50 | 不做遗传算法/参数优化                      | 无相关代码                                  | ✅  |
| 51 | 不让 LLM 在回测循环逐 K 线决策              | 规则用 `expr` 预编译缓存，循环内无 LLM 调用           | ✅  |
| 52 | 不做 Candidate/Active/Promote 生命周期 | 无相关代码                                  | ✅  |
| 53 | 不做 Kline Revision/Watermark      | 历史采用 latest canonical value（UPSERT 覆盖） | ✅  |

---

## 3. 构建与测试验证

| 项                                                                                                       | 结果                                                                       |
| ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| `go build ./...`                                                                                        | ✅ PASS                                                                   |
| `go vet ./...`                                                                                          | ⚠️ 仅既有噪声 `main.go:322/327` unreachable code（禁用 goroutine 遗留，非本 Phase 引入） |
| `go test -count=1 -race`（backtest / historicalmarket / models / controllers / utils / market / command） | ✅ 全绿                                                                     |
| `go test -count=1 ./...`                                                                                | ✅ 无失败                                                                    |

---

## 4. 上一版评审建议的闭环情况

| 上版建议                                      | 本次状态                                                                                                 |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| ① warmup 跳过改用类型化错误（原字符串匹配脆弱）              | ✅ **已修复**：`ErrInsufficientHistoricalBars` + `errors.Is`（`engine.go:156`、`environment.go:42/55`）      |
| ② 补真实指标 fixture 锁定索引约定                    | ✅ **已实现**：`TestHistoricalEnvironmentIndicatorOrderMatchesLive` 断言 newest→oldest 且 `ma3.Data[0]==103` |
| ③ `GetHistoricalKlines/FundingRates` 独立单测 | ⚠️ 仍为间接覆盖（`prefetch_test.go` 用 fixture source）；真实 API 分页/去重建议实机验证                                    |
| ④ Funding 缺口 12h 阈值硬编码                    | ⚠️ 保留（当前 Binance 8h cadence 下合理）                                                                     |
| ⑤⑥ 单例 cancel / `markInterrupted` 多实例边界    | ⚠️ 保留（单实例假设成立）                                                                                       |

---

## 5. 观察与非阻塞建议（不影响"已实现"结论）

1. **`BenchmarkSymbolsJSON` 已固定写 `[]`**：benchmark 能力已彻底移除（environment 不再使用），仅保留 DB 列以兼容历史数据。建议后续大版本迁移中评估是否清理该列（当前无害）。
2. **真实 Binance 历史 API 未做单测**：`GetHistoricalKlines/GetHistoricalFundingRates` 的分页、去重、范围过滤建议以 mock client 补单测，或在人工验收时用真实环境验证。
3. **前端页面交互细节未逐项核对**：本仓仅能确认前端 API 层与后端接口对齐；"手动获取历史数据""两次结果对比"等具体交互需在前端仓库人工确认。

---

## 6. 需人工 / 集成验收项

- [x] 真机跑一次完整回测（创建→进度→完成），校验结果明细与 Equity Curve。
- [x] 真机验证 `POST /agents/backtests/prefetch` 与 `market-condition/backfill`（真实 Binance 拉取、缺口补齐、已存在小时跳过）。
- [x] 验证 `DELETE /agents/backtests/:id` 后全局 `market_klines_*` / `market_funding_rates` 未被删除。
- [x] 验证含 `MarketCondition` 的策略在历史覆盖不足时被拒绝并给出"先补充历史数据"提示。
- [x] 前端仓库逐项确认回测页交互（创建、手动取数、进度、结果、分组指标、两次对比）。

---

## 7. 结论

**规格 `doc/agent/v3/03-phase-v3-3-backtest.md` 中描述的能力、数据模型、时间语义、API 与验收 Gate 均已实现，未发现缺失项；构建、静态检查与全量测试均通过。**

上版评审提出的两条代码级建议（类型化 warmup 错误、指标顺序 fixture）本轮已闭环。剩余为观察项与人工/集成验证项，不构成阻塞。

> 本报告为纯核查产出，未改动任何源代码、未写入项目/用户内存。
