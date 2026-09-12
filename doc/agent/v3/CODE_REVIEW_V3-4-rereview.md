# V3-4 复评报告（2026-09-12 代码更新后重审）

- **Phase**：V3-4 / Adaptive Resolution Backtest
- **审查基线**：分支 `feat/ai-agent-v3`，HEAD `cd950cf`（`feat: ai agent v3-4`），工作区含 13 个文件的未提交改动（`+841 / -85`，`static/` 前端构建产物不计入）
- **审查类型**：**review-only** —— 仅审查，未修改任何代码、未写入任何项目/用户记忆
- **审查结论**：**AUTOMATED PASS / 人工验收待定** —— 上一轮 `CODE_REVIEW_V3-4-public-data-loops.md` 的 3 项缺陷**已全部修复且有测试覆盖**；本轮**未发现 P0/P1**；新增 **2 项 P2 正确性边界**（ROI 阈值字面量提取不全、scanner 前向复用边界）与 5 项 P3
- **审查日期**：2026-09-12

---

## 1. 本轮改动范围

| 文件 | 变更 | 本轮作用 |
| --- | --- | --- |
| `service/historicalmarket/public_data.go` | 768 → 807 | 单 Run 前向流式 CSV scanner（按 kind 各持一个，回退时重开）；`TempRootDir` 配置与 `CacheDir` 互斥；`activity` 回调 |
| `service/historicalmarket/resolution_provider.go` | +46 | 父 `range` coverage 覆盖秒级子请求；读 `public_data_cache_dir`（默认 `./cache/tmp`）；`SetActivity` 转发 |
| `service/historicalmarket/public_data_test.go` | +205 | 前向复用 / 回退重开 / 重开失败清 slot / TempRoot 归属 / 配置互斥 |
| `service/historicalmarket/resolution_provider_test.go` | +54 | 父 range 覆盖子秒请求、0 网络请求集成用例 |
| `service/backtest/adaptive_engine.go` | +258 | ROI-only 平仓规则可达性证明（分钟级 + 秒级剪枝）；trade 重放跳过无谓 `BuildIntrabar` |
| `service/backtest/environment.go` | +68 | `series`/`tickerStats` 二分定位窗口，指标序列只复制 warmup 200 根；overlay 组件二分切片 |
| `service/backtest/engine.go` | +3 | `ResolutionActivity` 字段透传 |
| `service/backtest/store.go` | +17 | ResolutionActivity 1s 节流回调 |
| `service/backtest/engine_test.go` | +39 | 剪枝生效（0 次高精度抓取）+ profile 拒绝动态输入 |
| `.gitignore` / `conf/app.conf.example` / `doc/...04-phase...` | — | `/cache/tmp/`、`public_data_cache_dir`、文档记录 |

---

## 2. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./service/backtest/... ./service/historicalmarket/...` | ✅ 无输出（`main.go` 既有 unreachable code 噪声未在本轮范围内复查，非本轮引入） |
| `go test -count=1 -race ./service/backtest/... ./service/historicalmarket/...` | ✅ backtest 4.314s / historicalmarket 4.573s，无 race |
| `go test -count=1 ./...` | ✅ **55 个包 ok，0 FAIL** |
| `gofmt -l service/backtest service/historicalmarket` | ✅ 无输出 |
| `git diff --check`（排除 `static/`） | ✅ 无空白错误 |
| 前端产物交叉验证 | ✅ `static/static/js/backtest-C6MeXbsa.js` 同时含 `downloading_intrabar_archive` / `parsing_intrabar_archive` / `resolving_intrabar_data`，说明前端已按新阶段名重建 |

> `ld: warning ... malformed LC_DYSYMTAB` 为 macOS 工具链噪声，不影响结果。

---

## 3. 上一轮结论的逐项复核（已修复）

| 上轮缺陷 | 状态 | 证据 |
| --- | --- | --- |
| 缺陷 1（P2）：`openPublicDataCSVScanner` 失败后 slot 未清空，后续复用已关闭 scanner | ✅ 已修复 | `readPublicDataCSVRange` 在重开前 `closePublicDataCSVScanner(scanner); *scannerSlot = nil`，失败时不回写 slot；测试 `TestPublicDataScannerOpenFailureClearsClosedSlot`（删除 archive 后重开必须报错且 slot 为 nil） |
| 缺陷 2（P2）：`scanRange` 首行 visitor 返回 `errStopCSV` 被静默忽略 | ✅ 已修复 | `scanRange` 首行分支现为 `if errors.Is(visitErr, errStopCSV) { return nil }`，与后续行语义一致 |
| 缺陷 3（P3）：缺少"范围回退 → scanner 重开"测试 | ✅ 已补 | `TestPublicDataTradeScannerReopensOnBackwardRange`（断言 `client.tradeScanner != oldScanner` 且结果正确）+ 两个前向复用用例 |
| 上上轮：404 negative cache / SHA256 合法校验 / 父 range 覆盖 | ✅ 保持 | 本轮 `sourceRefRange` 替换 tag 包含判定，语义更严格（`coverageStart <= start && coverageEnd >= end`），并新增集成测试 |

**死循环结论维持**：`scanRange` 仍是唯一事件循环，两条 `continue` 均伴随 `reader.Read()` 或 `pending` 消费推进；新增的 `pending`/`eof` 状态机没有引入自循环路径。

---

## 4. 本轮新发现

### F1（P2，正确性边界）ROI 阈值字面量提取不完整，指数/下划线写法会得到错误阈值

`service/backtest/adaptive_engine.go:47`

```go
roiComparePattern = regexp.MustCompile(`(?:\bROI\s*(?:<=|>=|==|!=|<|>)\s*(-?\d+(?:\.\d+)?)|(-?\d+(?:\.\d+)?)\s*(?:<=|>=|==|!=|<|>)\s*\bROI\b)`)
```

数字部分 `-?\d+(?:\.\d+)?` **不覆盖**科学计数法与数字分隔符。`ROI >= 1.5e3` 会在 `1.5` 处截断并 `ParseFloat("1.5") = 1.5`，而 `roiCount == len(matches)` 仍成立 → 规则被判定为 eligible，但阈值集合缺失真实断点。

**可导致错误剪枝的场景**：`ROI >= 1.5e3 && ROI <= 1.6e3`，1m ROI 区间上界 > 1600。探针集合为 `{min, max, 1.5, 1.6, 各中点}`，真实可行区间 `[1500,1600]` 内没有任何探针 → `ruleCanPassForROIRange` 返回 false → `evaluateROIClose` 直接返回未解析、不下钻 1s/trades → **漏掉一次本可成交的平仓，结果静默改变**（无报错、无日志、无 evidence）。

- 触发概率取决于策略生成器是否产出指数/下划线字面量（expr 支持 `1e3`）；普通小数写法不受影响。
- 建议：把数字符号扩为 `-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?`；更稳妥的做法是"提取后回填校验"——把解析出的阈值与代码中的实际数字串做一次一致性断言，不一致即返回 ineligible（保守走全路径）。含 `_` 分隔符同理（`1_000` 会得到 1）。
- 补充测试建议（当前 profile 测试只覆盖 `ROI * 2 >= 5` 这类非比较式）：断言 `ROI >= 1.5e3 && ROI <= 1.6e3` 要么 ineligible、要么 `Thresholds == [1500,1600]`。

### F2（P2，正确性边界）scanner 前向复用判据用严格 `<`，`start == lastTime` 会漏掉该记录

`service/historicalmarket/public_data.go`（`readPublicDataCSVRange` 重开条件）

```go
if scanner == nil || scanner.archiveURL != archive.URL || (start > 0 && scanner.lastTime > 0 && start < scanner.lastTime) {
```

`lastTime` 是"已消费记录的最大时间戳"（pending 前瞻记录刻意不推进）。当本次请求的 `start` **恰好等于**上轮最后一条已消费记录的时间时，`start < lastTime` 为假 → 不重开；而该记录已被 scanner 越过，pending 记录时间又 `> end` → 本次返回空，**漏掉 `start` 时刻的那条记录**。

**真实可达路径**：`SecondBars(minute)` 在 1s archive 不可用时内部先调 `Trades(minute)`（整分钟解析，`lastTime` 推进到分钟内最大成交时间），随后 `evaluateROIClose` 对每个秒再调 `Trades(second)`；对**分钟最后一秒**，若该分钟最大成交时间恰好落在这一秒的首毫秒，则 `start == lastTime` 命中。

- 缓解因素：`Trades(minute)` 同事务把行写入 sparse（父 range coverage），秒级请求通常先命中 sparse cache（`sparseTradeRangeCached` 返回 true）而不触达 scanner；仅在冷缓存/落库被跳过时暴露。
- 建议：改为 `start <= scanner.lastTime` 触发重开。对单调前进的分钟级请求无额外成本（`start = prev.end + 1 > lastTime`），只修复等值边界。

### F3（P3，健壮性回归）分钟级证明引入 `BuildIntrabar` 硬失败路径

`service/backtest/adaptive_engine.go:236-248`

新增证明用**整根 1m bar** 作为 partialMinute 调 `BuildIntrabar`；`buildIntrabarOverlays` 对 `Open <= 0 || Close <= 0 || High < Low` 返回 `invalid intrabar partial minute`。该错误不是 `ErrInsufficientHistoricalBars`，会被 `return adaptiveCloseDecision{}, err` 直接上抛终止整个 Run。

而更新前只在秒级 partial 上构建环境，同样的脏 bar（`High/Low` 合法但 `Open/Close == 0`，`DetectROICandidate` 不拦截）不会导致整体失败。建议：把该类错误视同 `ErrInsufficientHistoricalBars` 处理（回退保守全路径），保持"优化只影响性能、不改失败语义"。

### F4（P3，数据一致性假设）trade 重放价格未校验落在 1s K 线 `[Low, High]` 内

`service/backtest/adaptive_engine.go:278-285` 的 `tradeCouldResolve` 证明基于 `second.Low/High` 推导的 ROI 区间；但重放循环（:400）只校验 `TradeTime` 范围与 `Price > 0`。若 1s K 线与 trades archive 在同一秒存在微小不一致（trade 价格越界），剪枝会误剪掉一次可成交平仓。建议：重放循环内对越界 trade 保守回退（不剪枝），或至少计数并落诊断字段。

### F5（P3，可复现性）Adaptive 行为已变，`AdaptiveEngineVersion` 仍为 `backtest_engine_v8`

`service/backtest/types.go:12`。本轮剪枝改变了"抓取哪些高精度数据"的行为（虽已论证结果不变，但稀疏落库内容、`ResolutionStats`、evidence 集合会变）。若 v8 已有落库 Run（`agent_backtest_run.engine_version`），新旧 Run 无法区分。若 v8 尚未发布可维持；否则建议 bump 到 v9（本轮无 schema 变更，DB version 不用动）。

### F6（P3，可观测性）高精度阶段名会被引擎进度回调覆写

`service/backtest/store.go:134-148`：`ResolutionActivity` 用 `updateProgress(runID, stage, lastProgress)` 写入 `downloading/parsing_intrabar_archive`，但紧随其后的 bar 级进度回调一旦数值上升就把 `stage` 覆写回 `running_backtest`，前端只能在两次 bar 进度之间看到高精度阶段。节流按"同 stage 且 <1s"生效，交替 stage 互不节流，量级仍可控（每候选分钟至多数次 DB 写）。若要求阶段稳定展示，建议引擎进度回调不要覆盖非 `running_backtest` 的 stage。

### F7（P3，文档）04-phase 文档重复与 benchmark 口径

- forward scanner 在 diff 中被记录两次（增量 bullet + 新章节 bullet），建议去重。
- 两组 benchmark 是不同区间："3 个 1s 分钟 / 104 个 trade 秒"（首次问题区间）与 "55 分钟 / 2,859 秒"（2026-01-01~01-15 全量），建议标注区间避免误读。

---

## 5. 正向确认（本轮设计的正确性论证要点）

1. **ROI-only 证明是完备的**：规则的阈值集合 + `closeGateReason` 的 SL/TP 断点构成全部真值变化点；探针覆盖所有端点/阈值点/相邻中点，因此对"分段常量"函数是完备采样。`closeGateReason` 的断点恰为 `±StopLossPct/TakeProfitPct`（禁用时 `DisabledFuturesROIThreshold = 1e6`，不会在真实区间内产生断点），与代码里"仅在 `>0` 时补充断点"一致。
2. **profile 的保守性足够**：`[0]` 索引、`NowTime/NowPrice/NowSymbol*`、`NetROI/Fee/NetProfit/Position(s)`、`IsAsc(/IsDesc(/KdjSimple(` 一律拒绝；ROI 出现次数必须等于"标准比较式"匹配数，非比较式（`ROI*2>=5`、`(ROI+1)>2`、`ROI>ma.Data[1]`）全部回退。
3. **stableEnv 复用的索引对齐成立**：我逐项验证了 1m 与高周期序列在 `stableEnv`（overlay=整根 1m bar、asOf=bar.CloseTime）与逐秒 env（overlay=partial、asOf=second.CloseTime）两种构建下：`replacesLast` 与 `canonicalLimit--` 的组合使两者长度相同、`index ≥ 1` 元素逐一相同；`Data/Close[1+]`、`kline_*[1+]` 与指标序列均无泄漏。
4. **trade 重放跳过 `BuildIntrabar` 是等价的**：`grossROI(position, trade.Price, leverage)` 与 `BuildIntrabar` 产生的 `env["ROI"]`（`= grossROI(position, current.Close)`，且 `current.Close == trade.Price`）定义一致；外层 gate 判定顺序未变。
5. **funding 语义未偏移**：`advanceFunding` 优先使用 `fund.MarkPrice`，仅当 `MarkPrice <= 0` 才用 fallbackMark；因此跳过 trade 重放（改用 `second.Close` 作 fallback）在正常数据下**完全等价**，仅在 funding 缺 MarkPrice 时有微小差异。
6. **`build()` 实际未使用 `cash` 参数**，故 `stableEnv` 一次构建、分钟内复用不存在 cash 陈旧问题。
7. **并发/锁**：`parseMu`（scanner）与 `mu`（archives/unavailable/activity）不嵌套，无锁序环；`Close()` 先锁 `parseMu` 关闭 scanner，再删目录。
8. **前向 scanner 内存/推进性**：`pending` 仅保留一条 `append([]string(nil), record...)` 拷贝（`csv.Reader.ReuseRecord=true` 必须拷贝），循环每轮必推进或返回，无死循环。

---

## 6. 未验证项 / 超范围

- 独立前端仓库 `go_binance_futrues_new_ui` 源码（本仓仅构建产物；已确认产物含三个新阶段名与更新后的 backtest chunk）。
- 真实 MySQL 环境索引行为、真实 Binance Public Data 全量 benchmark（本轮未联网复跑，引用文档数据）。
- `main.go` 既有 vet 噪声（非本轮引入，未复查）。
- 剪枝前后"结果完全一致"的**差分测试**缺失：当前只有"剪枝后不再抓取 + 结果符合预期"的断言，没有对同一策略分别走剪枝/保守路径比较 Trades/DataHash。建议补一条（用语义等价但含 `NowPrice` 触发 ineligible 的规则作对照）。

---

## 7. 人工验收待办

1. 跑一次真实 Adaptive Run（含 1s 与 trades 下钻），确认 `ResolutionStats` 与 `DataHash` 在同输入下稳定。
2. 若采纳 F1/F2：补 F1 的阈值断言测试、把复用判据改为 `<=` 并补 `start == lastTime` 用例。
3. 明确 `AdaptiveEngineVersion` 发布口径（是否 bump 到 v9）。
4. 若在意前端阶段展示（F6），确认 stage 覆写行为符合预期。
5. 按 F7 清理文档重复与 benchmark 口径。

---

## 8. 结论

本轮更新的**性能与可观测性改造方向正确、实现自洽**：单 Run 前向流式 scanner、父 range coverage、二分窗口定位、ResolutionActivity 阶段回调均在自动化测试下通过，死循环风险维持排除；ROI-only 证明的数学构造经逐点核对是完备的，且对动态输入保持保守回退。

**未发现 P0/P1 缺陷。** 两项 P2 均为"边界条件下静默改变结果"的类型（F1 阈值提取、F2 scanner 等值边界），触发条件窄且在正常数据/常规策略写法下不出现，但一旦命中不会报错、只体现在回测结果差异上，建议在人工验收前处理或至少补测试固化预期。其余 P3 为健壮性、版本口径、可观测性与文档问题。

**AUTOMATED PASS / 人工验收待定**（沿用本轮结论）。
