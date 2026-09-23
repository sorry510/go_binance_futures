# V4-3 复评报告（Local Coin Selector V2，大改后重新审查）

- **复审对象**：V4-3 **当前工作区/索引**版本（相对首轮审计时点的大幅重构）。基线 `HEAD = 5acac63`（v4-2）；V4-3 全部改动仍未提交（已 `git add`）
- **首轮报告**：`doc/agent/v4/CODE_REVIEW_V4-3.md`（2026-09-23 16:25，针对"池 30 + 单批 Top5、无轮转"的旧实现）
- **复审方式**：原地只读审计 + 隔离副本写临时探针（`git archive HEAD` + 拷贝全部未提交文件；探针与副本已删除，用户仓库未被改动）
- **结论**：**已完成，可进入 V4-4**。§9 Gate **九条全部通过**（含 4 条新增 Gate）；首轮 5 项发现中 **4 项已彻底解决**；本轮新增 **1 项 P2（需知情的行为变更）+ 5 项 P3**，无 P0/P1。
- **复审日期**：2026-09-23（18:35）

> **处理更新（2026-09-23）**：R1 按产品语义保留——测试交易必须跟随当前 `FutureStrategyCoin`，这是用户明确要求的“测试与真实一致”；文档已明确旧 `coin1～6` 也会继承其随机/窄覆盖行为。R2 已修正文档 Gate；R3/R4 已增加代码注释，明确 Preview-only 字段与直接 `SmartLocalV2.SelectCoins` = trade scope；R6 已把 PR-1 的四组精确分数差固化为永久测试。R5 继续按单实例自用设计保留。R7 中 V4-2 的管理员级 Publish/Activate 文档实际上已在此前 review 修复，仅损坏 Draft 静默跳过仍保留为非阻塞诊断项。

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| V4-3 是否完成 | ✅ **完成**，且实现质量较首轮显著提升（评分因子全部 opt-in 化、双模式隔离、轮转有界且可预览） |
| 是否可进入 V4-4 | ✅ **可以**（无阻塞项） |
| 首轮发现处理 | ✅ F1（共享评分影响面）**已彻底解决**、F2（limit 回退语义）**已修**、F4（冷却数据源覆盖面）**已文档化并测试化**、F5（测试缺口）**已补 7 个永久测试**；⚠️ F3（阈值不可配置）保持现状（文档已说明阈值固定） |
| 是否存在 P0/P1 | ❌ 未发现 |
| 关键新机制 | Top60 候选池 + **进程内 Round-Robin 每轮 5 个**；真实/测试**独立作用域**；Preview 走只读 `Peek` 不推进游标；评分新因子（TradeCount / 对称涨跌 / Symbol tie-break）全部经 `PrefilterOptions` **opt-in**，Generic Scanner 行为不变 |
| DB / 配置变更 | ❌ 无（Schema 仍 v18，无需 `sync db`） |

---

## 2. 与首轮审计的差异总览

| 文件 | 首轮 | 本轮 | 关键变化 |
| --- | --- | --- | --- |
| `scanner/local_selector_v2.go` | 162 | **239** | 池上限 30→**60**、`MinQuoteVolume` 1000 万→**500 万**、新增 `Mode`（trade/test）与 mode 专用冷却源、limit 语义由"回退 5"改为"clamp 到 60"、meta 增加 `requested_limit/effective_limit/pool_limit/pool_size/batch_size/mode` |
| `scanner/prefilter.go` | +27/-6 | **+159/-58** | 新增 `MaxLimit / IncludeBenchmarks / UseTradeCountScore / StableSymbolTieBreak / SymmetricChangeScoring` 五个选项；新增 `lowerWickRatio / reboundFromLow` 指标与对称涨跌评分；默认值全部保持旧行为 |
| `scanner/local_selector_v2_round_robin.go` | — | **176（新）** | 按 scope 隔离的轮转调度（`Next`/`Peek`）、公平性（新进场不饿死未服务者）、`sync.Mutex` 保护、池内失效项自动清理 |
| `scanner/local_selector_v2_db_test.go` | — | **215（新）** | 真实 `order` 表冷却、`test_strategy_results` 冷却、DB 端到端 + fail-closed、**真实/测试冷却源互不干扰** |
| `scanner/local_selector_v2_test.go` | 123 | **273** | 补 Generic 非回归（TradeCount/tie-break 不生效）、limit clamp 60、Generic 仍 30、5M 阈值、对称评分与对称极端过滤 |
| `scanner/local_selector_v2_round_robin_test.go` | — | **131（新）** | 12 轮覆盖 Top60 + 回绕、新进场公平性、**Peek 不推进**、scope 独立 |
| `feature/coin_selection.go` | — | **17（新）** | 统一入口 `selectConfiguredCoins(config, allCoins, mode)`：smart 走 mode 感知路径，其余回落旧 selector |
| `feature/strategy/coin/smart_local_v2.go` | 46 | **59** | 增加 `SelectSmartLocalV2CoinsForMode`；运行时对候选池取一批（`Next...For(mode)`） |
| `controllers/local_selector.go` | 27 | **32** | 支持 `mode=trade|test`；响应填充 `NextBatch`/`Rotation`（只读 `Peek`） |
| `feature/feature.go` | +2 | **+8/-3** | `StartTrade` 改用 `selectConfiguredCoins(..., ModeTrade)` |
| `feature/feature_test_strategy.go` | — | **+33/-33** | 测试交易改用 `selectConfiguredCoins(..., ModeTest)`；**删除 `offsetId`/`getSymbols` 顺序游标** |
| 文档 | — | 更新 | Phase 文档 §3–§9 与实施报告（434 行）全面反映新设计，含轮转、双模式、opt-in 因子与 Preview 语义 |

Go 侧合计 `13 files changed, +1299/-58`。

---

## 3. 首轮发现处理核对

| 首轮编号 | 问题 | 现状 | 证据 |
| --- | --- | --- | --- |
| **F1**（P2，需知情） | TradeCount 因子与 Symbol tie-break 位于共享函数，会改变既有 `ScanTop30` 消费者（market_scan/Opportunity、`scan_symbols`）的排序 | ✅ **彻底解决** | 所有新因子改为 `PrefilterOptions` 开关且**默认 false**（`prefilter.go:23-32`）；仅 `smart_local_v2` 显式传 `UseTradeCountScore/StableSymbolTieBreak/SymmetricChangeScoring`（`local_selector_v2.go:147-149`），其它调用点均未传（`git grep` 确认）→ **探针 PR-1 实证**：同一 fixture 下 Generic 87.60，Smart 87.60+Δ，Δ 恒等于受控增量（+8 / -12 / +4 / 0，四组精确匹配），且 Generic 的 reasons/risks 中不出现任何 TradeCount 文案；文档 §4 亦补上"不会改变既有 market_scan / Opportunity / scan_symbols 的评分与同分顺序" |
| **F2**（P3） | `?limit=100` 回退默认 5（而非 clamp 到上限 30） | ✅ **已修** | limit 语义改为 `<=0 → 60`、`>pool(60) → clamp 60`（`local_selector_v2.go:91-97`），meta 回显 `requested_limit/effective_limit`；永久测试 `TestSmartLocalV2LimitClampsToSixty`；Generic 仍 30（`TestGenericPrefilterStillCapsAtThirty`） |
| **F3**（P3） | 四个阈值不可配置、文档未说明 | ⚠️ 保持现状（可接受） | 实施报告 §235 明确"阈值当前固定在代码中：Pool=60、Batch=5、Cooldown=5m、MaxDataAge=30s、MinQuoteVolume=5M" ✓ 已文档化 |
| **F4**（P3） | 冷却覆盖面对齐真实写入方（此前测试传合成 map，SQL 从未被执行） | ✅ **已解决** | 新增 `local_selector_v2_db_test.go`：真实 `order` 表（`side='close'`）、`test_strategy_results`（已平仓）两条冷却源；并断言两源互不干扰。**我另行核对真实写入方**：`insertCloseOrder` 写 `Side="close"` + 毫秒 `UpdateTime`（`feature.go:638-640`）；模拟盘平仓两处均写 `UpdateTime = now`（`feature_test_strategy.go:289/321`）、开仓写 `UpdateTime = CreateTime`（`:401`）→ 数据源语义一致 ✓ |
| **F5**（P3） | 冷却 SQL / 全量 stale / DB 端到端 / limit 语义缺永久测试 | ✅ **已补** | `local_selector_v2_test.go` 10 个测试、`..._db_test.go` 4 个、`..._round_robin_test.go` 4 个（共 18 个），覆盖首轮探针的全部断言点 |
| **F1（V4-2 遗留）** | 发布=管理员级提示词变更的文档说明 | ⚠️ 仍未处理 | `doc/agent/v4/02-*.md`、README 中仍无相关表述（跨阶段遗留，与本阶段无关） |
| **F5（V4-2 遗留）** | `DraftStore.List` 对损坏 `draft.json` 静默 `continue` | ⚠️ 仍未处理 | `portableskill/draft.go:172-175` 未变（跨阶段遗留） |

---

## 4. Gate 逐项核对（§9 九条，含 4 条新增）

| # | Gate | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | 相同本地数据重复执行结果完全一致 | ✅（对"候选池"成立；措辞见 R2） | 池：`sort.SliceStable` 三级键 + 纯函数评分（永久测试 `TestSmartLocalV2DeterministicAndHardFilters`）；**探针 PR-1/PR-2** 证明 Generic 与 Smart 的分数差精确等于受控增量、tie 顺序符合各自开关。注意 **Batch 由轮转状态决定**，同一调用序列下才一致（实施报告 §6 措辞准确："相同本地数据重复执行，**Candidate** 顺序完全一致"） |
| 2 | 不出现随机抽样 | ✅ | 新链路（selector → prefilter → round-robin）无 `math/rand`；轮转由 `lastServed/firstSeen/sequence` 决定，非随机（探针 PR-3/PR-4 的确定性与可预测性实证） |
| 3 | disabled / stale / low-liquidity 不进入候选 | ✅ | Stage A 过滤 switch（`local_selector_v2.go:115-139`）+ Prefilter 阈值；永久测试覆盖四类排除；**首轮探针结论已被永久测试化**（含全量 stale fail-closed、DB 端到端） |
| 4 | Selector 本身不产生 Binance REST | ✅ **结构性证明** | `go list -deps ./scanner` 中 Binance 依赖数 = **0**（复评时重测一致）；`coin_selection.go` 仅 import `coin/models/scanner`；`GetAllSymbols()` 为本地 `symbols` 表查询（`feature.go:600-604`）；`meta.rest_api_used=false` |
| 5 | Top60 候选池有明确 reason/risk | ✅ | 候选结构含 `Reasons/Risks/Missing`；对称评分新增方向性文案（"24h 主方向未出现明显反转影线"/"存在明显反转影线/回撤风险"）。探针 PR-1 断言 Smart 路径确实产出 `"24h 成交笔数活跃"/"24h 成交笔数偏低"` 等文案 |
| 6（新增） | 每轮只返回 5 个；稳定池 12 轮覆盖全部 60 | ✅ **实证** | 探针 PR-4：连续 12 轮各返回 5 个（sequence 1→12），**60 个符号恰好各服务 1 次**，第 13 轮回绕到起始 5 个；永久测试 `TestSmartLocalV2RoundRobinCoversStableTop60` 另断言 rank 精确序列 |
| 7（新增） | Preview 只能查看下一批，不能推进真实/测试 Round-Robin 状态 | ✅ **实证** | 控制器用 `PeekSmartLocalV2BatchFor`（`controllers/local_selector.go:28`）；探针 PR-3 在**全局 scope** 上验证：连续 Peek 不变（sequence 恒 0）、随后 Next 返回的批次与 Peek 完全一致、且 Peek 不加裁状态；探针 PR-5 在应用层验证：调用前 Peek == 实际服务批次；调用 test 模式后，trade 的下一次仍为同一批 |
| 8（新增） | 真实/测试共用同一评分、Top60 与 Batch=5；测试不再按 ID 顺序轮询所有 Enable 币 | ✅ | 统一入口 `selectConfiguredCoins`（trade/test 仅 mode 不同）；`getSymbols/offsetId` 已删除；冷却源按 mode 分离（`order` vs `test_strategy_results`）且永久测试断言互不污染；探针 PR-5 验证 **scope 独立**（test 选择不推进 trade 游标） |
| 9（新增） | StartTrade / TestTrade 都只把结果交给 Line Strategy | ✅ | `feature.go:58 selectConfiguredCoins(...)` → `feature.go:410 GetCanLongOrShort(...)`；测试路径 `NoticeAllSymbolByStrategy` 同样仅用选币结果做开仓判定，未新增任何下单旁路 |

---

## 5. 本轮实证（6 组探针，全部通过）

| 探针 | 内容 | 结果 |
| --- | --- | --- |
| **PR-1** Generic/Smart 分数差精确性 | 4 组正涨幅正动量 fixture，同一 `MinQuoteVolume`，仅开关不同 | ✅ Generic 恒 **87.60**；Δ = **+8 / -12 / +4 / 0**，与受控因子（TradeCount ±8/−4/0 与涨跌档位 ±0/−8）逐一吻合；Generic 无任何 TradeCount 文案 → **既有消费者评分零漂移** |
| **PR-2** tie-break 门控 | 两枚完全同分同量、输入逆序 | ✅ Generic 保持输入顺序（BBB 在前），Smart 使用 Symbol ASC（AAA 在前） |
| **PR-3** 全局 Peek 语义 | 全局 scope 上 Peek×2 → Next → Peek | ✅ Peek 不推进（sequence 恒 0）、两次 Peek 相同、Next 与 Peek 完全一致、第二次 Peek 从 rank 6 开始 |
| **PR-4** 稳定池覆盖 | 60 池 × 12 轮 + 第 13 轮 | ✅ 每轮 5 个、60 个各服务 1 次、第 13 轮回绕 |
| **PR-5** 应用层接线 | `selectConfiguredCoins`（nil / smart+trade / smart+test / coin1 / 空配置） | ✅ nil → 空；smart+trade 返回 5 个且**等于调用前 Peek**；test 模式选择**不推进 trade 游标**；`coin1` 回落旧选择器（4 个启用币，随机 2 涨 + 2 跌）且不消耗 smart 轮转；空配置回落 coin1 语义 |
| **PR-6** 共享入口成本 | 800 符号 + 一次冷却查询 | ✅ **3.99ms**（含 SQLite 查询；纯评分环节首轮实测 0.13ms）→ 对 2 秒循环可忽略 |

---

## 6. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **R1** | **P2（需知情）** | **测试/模拟盘路径改为跟随 `FutureStrategyCoin`**：对非 `smart_local_v2` 配置（coin1～6），测试路径从"按 ID 顺序覆盖所有 Enable 币"变为"该 selector 的选币结果" —— 例如 coin1 = **随机 2 个涨幅榜 + 2 个跌幅榜**（`feature/strategy/coin/coin1.go`），即模拟盘同时继承了旧 selector 的**随机性与更窄的覆盖**。文档 §9 已声明"测试不再按 ID 顺序轮询所有 Enable 币"，但未点明"会继承旧 selector 的随机抽样/覆盖面收窄" | 影响面：`autoTestToTrade`（模拟盘连续盈利达标 → 提示人工开启实盘）所依据的样本集合发生变化；用户的既有模拟统计可比性下降。建议：① 在 Phase 文档/README 明确"测试路径 = 配置的选币策略"；② 若希望模拟盘保持全市场覆盖，可考虑为测试路径保留独立 selector（如 `FutureTestStrategyCoin`）或固定使用 `smart_local_v2`。注：`doc/TODO.md` 已登记"删掉 coin1~6、只保留自定义策略"，长期看该问题会随之消失 |
| R2 | P3（文档措辞） | Phase 文档 §9 第一条 Gate 仍写"相同本地数据重复执行结果完全一致"，但引入轮转后**只有候选池**满足该性质（同一调用序列下 Batch 才一致） | 实施报告措辞已准确（"Candidate 顺序完全一致"）；建议把 Phase 文档 Gate 同步为"同一份本地数据下 Top60 候选池完全一致；Batch 由 Round-Robin 状态决定" |
| R3 | P3 | `SmartLocalV2Result.NextBatch/Rotation` 只由 Preview 控制器填充；scanner 的 `SmartLocalV2*` 纯函数返回时为零值 | API 契约一致（文档 §8 描述的是预览响应）；但其它调用方（如后续新增的调试入口）需自行调用 `Peek`。建议在类型注释或文档标注"仅 Preview 填充" |
| R4 | P3 | 绕过统一入口直接调用 `GetCoinStrategy("smart_local_v2").SelectCoins(...)` 会落到 **trade** scope 并消耗真实轮转 | 当前无生产调用方（`selectConfiguredCoins` 在 smart 分支短路，`GetCoinStrategy` 仅注册表与测试使用）；建议加注释或让 `SmartLocalV2.SelectCoins` 显式记录"trade scope"语义，避免未来新增调用点误用 |
| R5 | P3 | 轮转状态为**进程内**且不落库：重启后从"池内前 5 名"重新开始；多实例部署会各自轮转 | 实施报告 §166 已声明"状态只存在进程内，不写数据库，重启后重新开始" ✓；与项目"单实例自用"前提一致 |
| R6 | P3 | 差分精确性（Generic vs Smart 分数差恒等于受控增量）无永久测试：现有 `TestGenericPrefilterDoesNotUseSmartLocalV2TradeCountOrTieBreak` 只对两枚币断言"分值相同/顺序不变" | 建议把探针 PR-1 的四组 fixture 与期望 Δ 固化进 `local_selector_v2_test.go`，作为"新因子绝不泄漏到 Generic"的回归护栏 |
| R7 | P3（跨阶段） | V4-2 遗留两项仍未处理：发布=管理员级提示词变更的文档说明、`DraftStore.List` 对损坏 `draft.json` 静默跳过 | 与本阶段无关，仅作跟踪 |

---

## 7. 自动化验证结果（用户工作区原地执行）

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./...` | ✅ 无输出 |
| `go test -count=1 ./...` | ✅ **61 个包 ok，0 FAIL** |
| `go test -count=1 -race ./scanner ./feature ./controllers` | ✅ scanner 1.328s / feature 1.734s / controllers 1.893s |
| `gofmt -l`（V4-3 改动 Go 文件） | ✅ 无输出 |
| `git diff HEAD --check -- '*.go'` | ✅ 无异常 |
| 隔离副本 `go build ./...` + 探针 PR-1～PR-6 | ✅ 全部通过 |
| 副本去掉探针后重跑 | ✅ `./scanner` 0.740s、`./feature` 0.783s 全 ok |
| 依赖隔离复测 | ✅ `go list -deps ./scanner` 中 Binance 依赖 = 0 |
| 前端 `pnpm typecheck` / `pnpm build` | ⚠️ 无法直接复跑（源码在独立仓库）；**产物交叉核对**：新增 `SmartLocalV2PreviewDialog-DRWvUymy.js`（含 `mode` prop、`next_batch`/`pool_size`/`batch_size`/`rotation`/`sequence`/`batch_ranks`、`selector-next-batch` 区块与 `selectorPreview.titleWithMode` 文案）；`service-hu5uzR7C.js` 仍引用 `selectors/smart-local-v2` |

---

## 8. 审计边界与未验证项

- **未验证**：前端源码与 pnpm 流水线（仅构建产物交叉核对）；`dist` ↔ `static` 逐文件 diff。
- **未验证**：真实行情下连续多轮 StartTrade 的轮转推进（探针在隔离副本内以合成池 + 真实全局 scope 验证；未在真实行情下持续观察 12 轮覆盖）。
- **未验证**：模拟盘路径的端到端（`NoticeAllSymbolByStrategy` → 开仓 → 平仓 → 冷却）真实联调；本轮以代码路径 + `UpdateTime` 写入方核对替代。
- **未验证**：MySQL 上的查询计划（本地为 SQLite；冷却查询命中既有 `Side+UpdateTime` 复合索引）。
- 未做任何真实下单动作。

---

## 9. 复审结论摘要（一句话）

首轮的两处结构性问题（共享评分影响面、限额回退语义）已被**opt-in 化 + clamp 化**彻底解决，冷却数据源与测试盲区也已补齐 18 个永久测试；新引入的 Round-Robin（Top60/每轮 5/双模式独立作用域/Preview 只读 Peek）经 6 组探针实证符合其全部声明。剩余仅 1 项需知情的 P2（**模拟盘路径现在跟随 `FutureStrategyCoin`，非 smart 配置下会继承旧 selector 的随机性**）与 5 项 P3，均不阻塞进入 V4-4。
