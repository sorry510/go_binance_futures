# V4-3 Phase 审计报告：Local Coin Selector V2

- **审计对象**：V4-3 实现**在工作区未提交**（新增 `scanner/local_selector_v2.go` 162 行 + 测试 123 行、`feature/strategy/coin/smart_local_v2.go` 46 行、`feature/coin_selector_v2_test.go` 14 行、`controllers/local_selector.go` 27 行；改动 `scanner/prefilter.go` +27/-6、`feature/feature.go` +2、`routers/router.go` +1 条路由）。基线 `HEAD = 5acac63`（`feat: ai agent v4-2`，分支 `feat/ai-agent-v4`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（`git archive HEAD` + 拷贝全部未提交文件；探针与副本审计后已删除，用户仓库未被改动）
- **审查类型**：review-only
- **结论**：**已完成，可进入 V4-4**。§9 六条 Gate **全部通过**；未发现 P0/P1。风险为 1 项需**知情**的共享影响（新评分因子同时改变既有 Scanner/Opportunity 候选排序）与 5 项 P3（limit 回退语义、阈值不可配置、冷却覆盖面、测试缺口）。
- **审计日期**：2026-09-23

> **修复更新（2026-09-23）**：F1/F2/F5 已修复；F3/F4 已明确记录边界。TradeCount 分数和 Symbol 末级 tie-break 已改为 `smart_local_v2` opt-in，不再改变 Generic Scanner/Opportunity/scan_symbols；`limit>30` 改为 clamp 30，并回显 requested/effective limit；真实 `order` SQL cooldown、DB 端到端、全量 stale fail-closed 和 limit clamp 已固化为永久测试。F3 的阈值仍固定在代码中并已在 Phase/实施报告说明；F4 的 cooldown 数据来源、手工 App 平仓不覆盖、按 Symbol 不分方向的边界已文档化。V4-2 的管理员级 Publish/Activate 说明实际上已在上一轮修复；仅 `DraftStore.List` 损坏 draft 静默跳过仍作为已知非阻塞项保留。

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| V4-3 是否完成 | ✅ **完成**（新增确定性本地选币 `smart_local_v2`，复用 `scanner` 评分，未新写第二套 scorer） |
| 是否可进入 V4-4 | ✅ **可以**（无阻塞项） |
| DB / 配置变更 | ❌ **无**：无 `models/`、`conf/`、`main.go` 改动，Schema 仍为 v18，无需 `sync db` |
| 旧 selector 兼容 | ✅ `coin1～6` 未被改动（`feature/strategy/coin/` 下仅新增文件），`GetCoinStrategy` 仅追加 `case "smart_local_v2"`；系统当前 selector 未自动切换 |
| 是否存在 P0/P1 | ❌ 未发现 |
| 开仓链路是否被绕过 | ✅ **未被绕过**：`feature.go:58` 选币 → `feature.go:410` `coin_line_strategy.GetCanLongOrShort()` 判定方向，V4-3 未改动该路径（diff 仅 +2 行注册） |

---

## 2. 验收 Gate 逐项核对（§9 六条）

| # | Gate | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | 相同本地数据重复执行结果完全一致 | ✅ | 排序三级 tie-break（`prefilter.go:160-168`：Score DESC → QuoteVolume DESC → Symbol ASC，`sort.SliceStable`）；永久测试 `TestSmartLocalV2DeterministicAndHardFilters`（两次调用 `reflect.DeepEqual`）与 `TestSmartLocalV2StableTieBreakUsesSymbol`（**乱序输入仍得 AAA→BBB**） |
| 2 | 不出现随机抽样 | ✅ | `scanner/local_selector_v2.go` 与 `feature/strategy/coin/smart_local_v2.go` 中 `math/rand`/`rand.` 出现次数 = **0**；全程无随机源 |
| 3 | disabled / stale / low-liquidity Symbol 不进入候选 | ✅ | Stage A 硬过滤 switch（`local_selector_v2.go:77-101`）：未启用 / 非 USDT / 无更新时间 / 超过 `MaxDataAgeMs` / 冷却中；低流动性由 `PrefilterTop30FromSymbols` 的 `QuoteVolume < minQuoteVolume` 分支（`prefilter.go:135-137`）剔除。永久测试覆盖四类排除；**探针 P2 补充实证**：**全量 stale 时返回 0 候选**（fail-closed），且未来时间戳（时钟偏移）不会被误判为 stale |
| 4 | Selector 本身不产生 Binance REST 请求 | ✅ **结构性证据** | ① `go list -deps ./scanner` 中 Binance 相关依赖数 = **0**（`scanner` 只 import `models`/标准库）→ 该路径在编译期就不可能发 REST；② `smart_local_v2.go` 未 import `feature/api/binance`（同包 `common.go` 有引用，但不在本路径）；③ `result.Meta["rest_api_used"]=false` 恒置（探针 P3 实证 meta 实际值）；④ 唯一外部数据源为本地 `symbols` 与 `order` 表 + WS 已写入的字段 |
| 5 | Top K 有明确 reason/risk | ✅ | 候选结构含 `Rank/Score/Grade/Reasons/Risks/Missing`（`prefilter.go:37-58`）；V4-3 新增 TradeCount 因子同时产出 `"24h 成交笔数活跃"/"24h 成交活跃度良好"` reason 与 `"24h 成交笔数偏低"` risk（`prefilter.go:237-246`）。**探针 P3 实测** Preview（DB 路径）返回 3 个候选且 meta 完整（`selector/limit/cooldown_minute/max_data_age_ms/min_quote_volume/rest_api_used`） |
| 6 | StartTrade 只把结果交给 Line Strategy，不绕过策略开仓条件 | ✅ | `feature.go:58 SelectCoins` → `feature.go:410 GetCanLongOrShort`；V4-3 的 `feature.go` 改动仅 `GetCoinStrategy` 追加一个 case（+2 行）；新 selector 只返回候选 *Symbol* 列表（`smart_local_v2.go:SelectCoins`），不产生任何 PendingAction |

**§3–§8 要求核对**

| 要求 | 判定 | 证据 |
| --- | --- | --- |
| 不新写第二套评分器 | ✅ | `local_selector_v2.go:103` 调用现有 `PrefilterTop30FromSymbols`；评分逻辑集中在 `prefilter.go` |
| 默认阈值与报告一致 | ✅ | Top K=5、Cooldown=5min、MaxDataAge=30s、MinQuoteVolume=10,000,000（`local_selector_v2.go:14-19`）与报告 §4 逐一吻合 |
| 冷却只查本地 `order` 表 | ✅ **探针 P1 实证** | 在真实 `order` 表插入：1 分钟前的 `side='close'` → 被排除且 reason=`最近交易冷却中`；10 分钟前的 close → 仍可入选；`side='open'` → **不触发冷却**；查询列（`side`/`updateTime`）与模型/索引（`TableIndex{Side,UpdateTime}`）一致，毫秒单位一致 |
| `IncludeBenchmarks` 兼容旧行为 | ✅ | `prefilter.go:131` `!opts.IncludeBenchmarks && (BTCUSDT||ETHUSDT)`；默认 false 保持旧 Scanner 语义（永久测试 `TestGenericPrefilterStillExcludesBenchmarksByDefault`）；仅 `smart_local_v2` 传 true |
| Runtime 不生成完整 Excluded 明细 | ✅ | `local_selector_v2.go:117-119`：`IncludeExcluded=false` → `Excluded=nil`；**探针 P4**：800 符号（200 enabled）runtime 路径 **129.7µs**，`Excluded` 为 nil；Preview 传 `IncludeExcluded=true`（`controllers/local_selector.go:19`） |
| 只读 Preview API | ✅ | `GET /futures/selectors/smart-local-v2`（`routers/router.go` 新增 1 行，`get:SmartLocalV2`）；未被加入 `middlewares/auth.go` 的 `excludeRoutes` → 受 JWT 保护 ✓ |
| 无 schema / 配置变更 | ✅ | 工作区无 `models/`、`conf/`、`main.go` 改动 |

---

## 3. 实证过的隐式契约（探针，全部通过）

探针写在隔离副本 `/tmp/phase_audit`（跑完已删除）；以 `-run TestProbe` 单独执行，并另跑一次去掉探针的 `./scanner ./feature ./controllers` 确认原测试仍通过（均 ok）。

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **P1** 冷却走真实 `order` 表 | 1 分钟前 close / 10 分钟前 close / 1 分钟前 open / 对照币 | ✅ 仅"窗口内 close"被排除（reason 精确为 `最近交易冷却中`），旧 close 与 open 不阻塞。**该 SQL 路径此前无永久测试**（现有测试传入合成 cooldown map） |
| **P2** fail-closed | 全部 stale + `UpdateTime=0` + 未来时间戳 | ✅ stale/缺失更新时间 → 0 候选（fail-closed）；未来时间戳仍可入选（不产生负年龄误判） |
| **P3** DB 端到端 + limit | `SmartLocalV2()`（Preview 真实路径）对真实 `symbols` 表；`Limit=3` 与 `Limit=100` | ✅ `selector=smart_local_v2`、候选 3 个、meta 六键齐全（含 `min_quote_volume`、`rest_api_used=false`）；`Limit=100` → **回退默认 5**（非 clamp 到 30）；`Excluded` 在 runtime 选项下为 nil |
| **P4** 运行成本 | 800 符号快照（200 enabled）runtime 路径 | ✅ **129.7µs**，`Excluded` 不构建 → 对 2 秒交易循环可忽略 |
| 依赖静态检查 | `go list -deps ./scanner` 与 `./feature/strategy/coin` | ✅ `scanner` 零 Binance 依赖；`coin` 包仅 `common.go` 引用 binance，`smart_local_v2.go` 不引用 |

---

## 4. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| F1 | ✅ 已修复 | TradeCount 分数和 Symbol ASC 末级 tie-break 原先位于共享 Prefilter，会改变既有 Scanner 消费者。 | 已新增 `UseTradeCountScore` / `StableSymbolTieBreak` opt-in；Generic Scanner 默认关闭，只有 `smart_local_v2` 启用。并新增兼容测试，保证既有 Scanner 不吃 V4-3 TradeCount 分数、同分时保留原 stable 输入顺序。 |
| F2 | ✅ 已修复 | `?limit=100` 原先回退默认 5。 | 现在 `limit<=0` 才使用默认 5；`limit>30` clamp 到 30，并在 meta 回显 `requested_limit` / `effective_limit`。永久测试覆盖 `limit=100 → 30`。 |
| F3 | ✅ 已明确 | 四个阈值仍为代码固定值；Preview 只允许覆盖 `limit`。 | Phase/实施报告已明确“当前阈值固定，如需调整改代码并重新验证”；Preview meta 持续回显实际生效 cooldown/freshness/min volume/effective limit。个人自用场景不新增配置项。 |
| F4 | ✅ 已明确 | Cooldown 只覆盖本系统本地 `side='close'` row，按 Symbol 生效，不区分 LONG/SHORT；手工 Binance App 平仓不一定写入本地 close row。 | 实施报告已明确该边界和保守含义；本阶段不扩大 cooldown 数据源，避免引入新的 Binance REST/账户同步依赖。 |
| F5 | ✅ 已修复 | 原先真实 `order` SQL、DB 端到端、全量 stale、limit>30 只有审计探针。 | 新增 SQLite 隔离 DB 永久测试：窗口内 close/旧 close/open 的 cooldown 行为、DB 端到端 hard filters、全量 stale fail-closed；另补 `limit=100 → 30`、Generic Prefilter 兼容测试。 |
| F6 | P3（承接 V4-2） | V4-2 的管理员级 Publish/Activate 文档说明已在上一轮补齐；`DraftStore.List` 对损坏 `draft.json` 静默 `continue` 仍保留。 | 仅剩 Draft List 诊断性问题，与 V4-3 无关且非阻塞；维持现状，避免一个损坏 Draft 阻断整个列表。 |

---

## 5. 自动化验证结果（用户工作区原地执行）

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./...` | ✅ 无输出 |
| `go test -count=1 ./...` | ✅ **61 个包 ok，0 FAIL** |
| `go test -count=1 -race ./scanner ./feature ./controllers` | ✅ 见报告声明；本轮补跑的非 race 版本 `./scanner ./feature ./controllers` 全 ok |
| `gofmt -l`（V4-3 改动 Go 文件） | ✅ 无输出 |
| `git diff --check`（排除 static） | ✅ 无异常 |
| 隔离副本 `go build ./...` + 探针 P1–P4 | ✅ 全部通过 |
| 副本去掉探针后重跑 | ✅ `./scanner` 0.773s、`./feature` 0.308s、`./controllers` 0.440s 全 ok |
| 前端 `pnpm typecheck` / `pnpm build` | ⚠️ 无法直接复跑（源码在独立仓库）；**产物侧交叉核对**：`service-Dc2eWi4q.js` 含 `selectors/smart-local-v2`；`configShow-CxYRKNxb.js` 含 `smart_local_v2` 与预览字段 `quote_volume_24h`/`trade_count_24h`/`local_momentum_pct`/`rest_api_used`；`index-CorNRy1M.js` 含 `smart_local_v2` 选项 |

---

## 6. 审计边界与未验证项

- **未验证**：前端源码与 pnpm 流水线（仅构建产物交叉核对）；`dist` 与后端 `static` 逐文件 diff。
- **未验证**：真实行情下的连续 Preview 稳定性（Gate #1 的"重复执行一致"已用同一份快照实证；跨秒级行情自然变化导致的候选变动属预期行为，不属于不确定性）。
- **未验证**：`feature/strategy/coin/smart_local_v2.go` 的 `SelectCoins` 与 `StartTrade` 的真实联调（文档 §12 F 组手工步骤）；本轮仅做代码路径核对（`feature.go:58` → `SelectCoins`，候选交由 `feature.go:410` 判定方向）。
- **未验证**：MySQL 上的 `order`/`symbols` 查询计划（本地探针用 SQLite；查询命中既有 `Side+UpdateTime` 复合索引，风险低）。
- 真实交易未做任何下单动作（本阶段为只读选币逻辑 + 只读 Preview API）。

---

## 7. 附：V4-3 交付物清单

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `scanner/local_selector_v2.go` | 162 行（新） | `smart_local_v2` Candidate Service：Stage A 硬过滤（enabled/USDT/freshness/cooldown）+ 复用 Prefilter 评分；本地冷却查询；`rest_api_used=false`；Runtime/Preview 的 Excluded 差异 |
| `scanner/local_selector_v2_test.go` | 123 行（新） | 确定性、四类排除、Runtime 省略 Excluded、Symbol ASC tie-break、benchmark 兼容 |
| `feature/strategy/coin/smart_local_v2.go` | 46 行（新） | `CoinStrategy` 适配器：把候选映射回 `allCoins` 指针，失败时返回空集（fail-closed） |
| `feature/coin_selector_v2_test.go` | 14 行（新） | `GetCoinStrategy("smart_local_v2")` 注册断言 |
| `controllers/local_selector.go` | 27 行（新） | `GET /futures/selectors/smart-local-v2` 只读预览（`IncludeExcluded=true`） |
| `scanner/prefilter.go` | +27/-6 | 新增 `IncludeBenchmarks` 选项；三级确定性排序；TradeCount 活跃度因子（共享评分） |
| `feature/feature.go` | +2 | `GetCoinStrategy` 追加 `smart_local_v2` 分支 |
| `routers/router.go` | +1 | 预览路由 |
| 文档 | — | `03-phase-v4-3-local-selector-v2.md` §11 实现结果；新增 `v4-3-implementation-report.md` |
