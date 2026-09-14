# V3-7 Phase 审计报告：交易复盘与策略比较

- **审计对象**：V3-7 实现**在工作区未提交**（`controllers/agent_outcome.go`、`service/outcomereview/`、`routers/router.go`、文档）。基线 `HEAD = ef87871`（master，tag v2.0.4，已含 V3-6）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（`/tmp/phase_audit`，审计后已删除；用户仓库未被改动）
- **审查类型**：review-only
- **结论**：**已完成（个人版范围），可以进入 V3-8**。6 条验收 Gate 中 5 条完全满足，1 条（Live 追踪下钻）按 §9 收窄实现、未做到"页面内跳转 Proposal → Risk → Execution → Audit"，属**非阻塞缺口**。未发现 P0/P1 缺陷。
- **审计日期**：2026-09-14

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| V3-7 是否完成 | ✅ **完成**（范围以文档 §9「个人自用轻量实现」为准） |
| 是否可进入 V3-8 | ✅ **可以**（无阻塞项；Gate #3 的缺口建议先与需求方确认口径） |
| 是否需要 `sync db` | ❌ **不需要**：未新增表/字段，`dbVersion` 仍为 13（`main.go:40`），`models/`、`main.go` 无改动 |
| 是否存在 P0/P1 缺陷 | ❌ 未发现 |
| 新增依赖 | 无（`go.mod` 未改动） |

---

## 2. 验收 Gate 逐项核对（文档 §7 + §9 最终 Gate）

| # | Gate（§7） | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | Backtest 汇总与 `agent_backtest_runs/trades` 可逐项对账 | ✅ 通过 | SQL 聚合：`service/outcomereview/service.go:102-158`（单条 `COUNT/SUM` 聚合 + 3 组 `GROUP BY`）；固定 Fixture：`service_test.go:107-156`（断言 trades=2、net=6、return=0.6%、DD=7.5、PF=2.5、分组数）。**另由我的探针 P1 复核：SQL 与 Go 两套聚合在混合样本上逐项一致** |
| 2 | 模拟盘汇总与 `test_strategy_results` 可逐项对账 | ✅ 通过 | 复用既有实现：`service.go:160-179` → `strategy.Service{}.ListTestResults`；其 `profitSQL` **不带 LIMIT**（`service/strategy/service.go:101,115-123`），故 Stats 覆盖全量筛选结果。**探针 P2 实证**：4 条样本（3 平仓 + 1 未平仓）在 `Limit=1` 的包装下仍返回 `total=4 / closed=3 / open=1`，未平仓不污染已实现净收益；`strategy_template_id` 过滤生效；paper 拒绝 `market_condition` 维度 |
| 3 | 真实交易能追踪到 Proposal → Risk → Execution → Audit | ⚠️ **部分满足** | Live 仅返回 5 个计数（`service.go:59-66,181-210`），**不含 proposal 级标识**；构建产物 `static/static/js/outcomeReview-f6JMKBUT.js` 中无 `agents/trade/proposals` 跳转。完整链条仍可在既有「受控交易」页面查询。**探针 P2b 实证**：Live 只统计 `agent_trade` 归属（`auto_strategy` 的仓位/挂单被正确排除），筛选经 proposal 子查询生效 |
| 4 | 策略 A/B 比较不因模板后来新增而改变旧 Run 历史结果 | ✅ 通过 | Backtest 只读 Run 自身字段（`r.strategy_template_id`、`r.metrics_json`，`service.go:139,353-356`），**不 JOIN `strategy_templates`**（§6 要求）；Paper 的分组键含 Snapshot Hash（`service/strategy/stats.go:106-125`，key 形如 `template|<id>|<hash>`），Hash 不同不合并 |
| 5 | 聚合计算有固定 Fixture 测试 | ✅ 通过 | `service_test.go:107-156`（SQLite 建表 + 固定样本 + 精确数值断言）；另 `:52-67` 覆盖 Go 聚合器、`:81-105` 覆盖筛选 SQL 拼接 |
| 6 | 页面查询不能明显影响行情和交易主循环 | ✅ 通过（实测） | **探针 P3**：40 Run / **20,000 trades** 下 `Backtest()` 全量 4 条聚合查询 **26.9~28.6ms**；带 `symbol+side+market_condition+template` 筛选 **9.3~13.3ms**（SQLite，本地文件库）。查询为用户触发、非交易循环内调用 |

| 附加 | 判定 | 证据 |
| --- | --- | --- |
| §9「新增 `service/outcomereview` 与 3 个只读 API」 | ✅ | `routers/router.go:25-27` 三个 **GET** 路由；控制器薄封装 `controllers/agent_outcome.go:26-51` |
| §9「Live 不伪造 Net PnL」 | ✅ | `service.go:208` 恒置 `PnLAvailable=false`，接口不返回任何 PnL 字段（探针 P2b 断言） |
| §9「未新增数据库表 / 版本保持 v13」 | ✅ | `models/`、`command/`、`main.go` 均无工作区改动；`main.go:40 dbVersion = 13`；涉及的 6 张表全部由 V2-12/V3-4/V3-5 创建 |
| §9「前端 Backtest/模拟盘/Live 三 Tab + 策略对比≤3」 | ✅（产物级） | 构建产物含三个接口调用与 `by_symbol/by_side/by_market_condition/profit_factor/max_drawdown_pct/win_rate/managed_orders` 字段及 `strategy_template_id`/`market_condition` 筛选参数；前端源码在独立仓库，未审（见 §5） |
| §8「明确不做」项 | ✅ 未越界 | 无强化学习、无自动改策略、无自动淘汰/晋级、无 Model/Prompt 排行榜、无 BI 组件 |

---

## 3. 实证过的隐式契约（探针方法 + 结果）

探针写在隔离副本 `/tmp/phase_audit/service/outcomereview/zz_probe_test.go`（副本已删除，未污染用户仓库）。

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **P1 SQL/Go 聚合一致性** | 同一样本（胜/负/平局混合、双 Symbol、双方向、3 个 MarketCondition、非整除持仓时长）分别走 SQL 路径与 `summarizeBacktest` | ✅ 完全一致：`trades=5 net=11 fees=2.5 funding=0.3 winRate=0.4 pf=2.6923 avgHold=1400`；`market_condition` 分组键为可读字符串 `1/2/3`（不存在 int→string 扫描失败） |
| **P1b 空结果集** | 不存在的 Symbol 过滤 | ✅ 返回全零 Summary、分组为**非 nil 空切片**（JSON 输出 `[]` 而非 `null`），不报错 |
| **P2 模拟盘聚合范围** | 4 条样本 + 包装层强制 `Page=1,Limit=1` | ✅ `total=4 / closed=3 / open=1 / wins=2 / losses=1`，净收益仅含已平仓 → 分页 limit **未泄漏**进 Stats（这是复用既有函数的关键风险点） |
| **P2b Live 归属与筛选** | 2 个 Proposal + 3 个托管仓位（含 1 个 `auto_strategy`）+ 2 个托管订单 | ✅ `Proposals=2 Executed=1 OpenPositions=1 ClosedPositions=1 ManagedOrders=1`，`auto_strategy` 行被正确排除；按 `ETHUSDT/SHORT/MC=3` 过滤后只剩对应 proposal 的仓位，`PnLAvailable=false` |
| **P3 大历史成本（Gate #6）** | 20,000 trades 批量种入后计时 | ✅ 全量 26.9~28.6ms；带筛选 9.3~13.3ms（若数据量增至 20 万条，按线性外推约 0.3s 量级——个人自用可接受） |

---

## 4. 变更清单与实现要点（核对结果）

| 文件 | 规模 | 核对结果 |
| --- | --- | --- |
| `service/outcomereview/service.go` | 465 行 | Backtest 走纯 SQL 聚合（`102-158`），3 个维度 `GROUP BY`（`228-247`）；Paper 复用既有实现（`160-179`）；Live 只读计数（`181-210`）；全部条件参数化（`342-465`），`keyExpr` 为**硬编码常量**（`t.symbol`/`t.side`/`t.market_condition`）→ 无注入风险；`validateFilter` 校验时间区间顺序与 `side ∈ {LONG,SHORT}`（`332-341`） |
| `service/outcomereview/service_test.go` | 156 行 | 4 个测试：Go 聚合器、Filter 校验、2 个 SQL 拼接断言、SQLite 固定 Fixture |
| `controllers/agent_outcome.go` | 51 行 | 薄控制器，仅解析 query 参数并调用 service，无写入操作 |
| `routers/router.go` | +3 行 | 三个 GET 路由（`25-27`） |
| `doc/agent/v3/07-phase-v3-7-outcome-review.md` | +20 行 | 追加 §9「最终实现」+「最终 Gate」；`README.md` 状态标记 ✅ |

**性能设计核对**：Backtest 不在 Go 侧加载交易明细（仅加载 `runs` 的 `initial_equity`/`metrics_json`，见 `service.go:137-141`，量级为 Run 数而非 Trade 数）→ 与 §9 声明一致。

---

## 5. 风险与待改进（全部非阻塞）

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| F1 | P2（Gate 缺口） | Live 复盘无 proposal 级下钻，Gate #3「Proposal → Risk → Execution → Audit」未在复盘页打通 | `service.go:59-66` 响应无标识字段；构建产物无 `agents/trade/proposals`。建议：Live 增加"最近 N 条 proposal（id/状态/Risk/Execution/Audit 状态）"列表或至少返回 `proposal_ids` 供前端跳转 |
| F2 | P3（死代码/漂移风险） | `summarizeBacktest` + `backtestTradeRow` + `addGroup` + `finalizeGroups` + `groupAccumulator`（`service.go:68-100,249-331`）**仅被测试使用**，生产走 SQL。`TestSummarizeBacktest` 因此产生"已覆盖"的错觉 | 建议删除 Go 版聚合，或把该测试改为"对 SQL 路径结果的独立断言"。我的探针 P1 已实证两者当前一致，但无永久回归保护 |
| F3 | P3（内存/一致性） | Paper 路径把筛选范围内**全部** `test_strategy_results` 载入 Go 内存（`service/strategy/service.go:115-123`，无 LIMIT），与 §9「避免把大量历史行加载进内存」的原则不一致 | 既有 V3-3 页面同行为，非 V3-7 新引入；个人数据量可接受。若模拟盘数据长期增长，建议为该函数加 SQL 聚合或分页上限 |
| F4 | P3（文档） | §2 对 Live 要求 Entry/Exit/Stop/TP/实际 PnL/Risk/MarketCondition/执行异常，§9 收窄为"仅统计计数"——两节口径不一致；§9 标题前缺空行，且全文出现两个 Gate 章节（§7「验收 Gate」与 §9「最终 Gate」） | 建议在 §2 标注"已按 §9 收窄"，并合并/重排 Gate 章节 |
| F5 | P3（口径） | `ReturnPct = ΣNetPnL / ΣInitialEquity`（`service.go:144-156`）在跨 Run 聚合时会低估收益率（同一笔本金被重复计入分母）；`MaxDrawdownPct` 取 Run 级最大值而非组合级，另以 `max_drawdown_available` 标记（`service.go:143,147-152`） | 已如实暴露 flag，属个人版简化；建议在页面/文档标注口径，避免误读为组合曲线回撤 |
| F6 | P3（测试） | 现有 Fixture 只断言 `market_condition` 分组**数量**，未断言键值（我的探针已实证键为 `"1"/"2"/"3"`） | 建议把键值断言补进永久测试（可移植探针 P1 的断言） |
| F7 | 提示 | 工作区有 3 个**未被任何代码引用**的策略模板 JSON（`strategy_templates/{4h-trend-1h-atr-pullback-v12,daily-trend-4h-pullback-recovery-v12,daily-trend-4h-pullback-recovery-long-core-v13}.json`） | 应为人工验证样本；`git add -A` 会一并提交，提交前请确认是否有意纳入 |
| — | 提示 | `Live().OpenPositions` 用 `status != 'closed'` 计数，未附加 `managed_qty > 0` 条件 | 与 ownership 服务 `ActivePositions`（要求 qty>0）口径略有差异；正常流程下不会出现 qty=0 且非 closed 的行 |

---

## 6. 自动化验证结果（在用户工作区原地执行，只读）

| 项目 | 结果 |
| --- | --- |
| `go vet ./...` | ✅ 无输出（早期的 `main.go unreachable code` 噪声已消失） |
| `go test -count=1 ./...` | ✅ **59 个包 ok，0 FAIL** |
| `go test -count=1 -race ./service/outcomereview ./controllers ./routers` | ✅ outcomereview 1.319s / controllers 1.803s（routers 无测试文件） |
| 隔离副本 `go build ./...`（`git archive HEAD` + V3-7 文件） | ✅ 通过 |
| 隔离副本探针 | ✅ 6 个探针全部 PASS（见 §3） |
| 前端双 Gate（`vue-tsc --noEmit`、`pnpm build`） | ⚠️ **未验证**（前端源码在独立仓库）；但构建产物 `static/static/js/outcomeReview-f6JMKBUT.js` 含三个接口与全部指标字段，可佐证已构建并同步 |

---

## 7. 审计边界说明

- 本次审计覆盖**后端实现 + 数据口径 + 性能**；前端源码（独立仓库 `go_binance_futrues_new_ui`）未审，仅通过构建产物做交叉核对。
- 生产库为 MySQL 时未实测（探针使用 SQLite 文件库）；SQL 为 ANSI 风格 `CASE WHEN / COALESCE / EXISTS`，MySQL 兼容性无风险，但 `AVG()` 返回浮点→`int64` 截断行为在两库上一致。
- 未做真实数据量压测（生产 `agent_backtest_trades` 实际行数未获取）；§3 的 P3 为合成数据实测。
- 未运行前端 E2E/手工验收（V3-7 的 UI 验收仍建议由用户按页面实际操作确认三个 Tab、筛选与策略对比）。

---

## 8. 复查处理记录（2026-09-14）

本轮根据审计结果完成以下收口：

- **F1 已修复**：Live Outcome API 返回最近 20 条受筛选条件约束的 Proposal 摘要；交易复盘 Live Tab 可在当前页面打开只读详情，追踪 Proposal → Risk → Execution → Managed Position → Audit。
- **F2 已修复**：删除仅供测试使用的 `summarizeBacktest`、`backtestTradeRow`、`groupAccumulator` 等 Go 侧旧聚合实现；永久 Fixture 直接验证生产 SQL 路径。
- **F4 已修复**：Phase 文档已明确 Live 的个人版确定性数据边界，并将 §9 改为“最终实现与验收记录”，避免与 §7 Gate 重复。
- **F5 已澄清口径**：不改现有公式；页面与文档明确 `ReturnPct = ΣNetPnL / ΣInitialEquity` 是按初始资金加权的 Run 收益口径，不是组合权益曲线；子集无法确定性计算 Max Drawdown 时继续显示不可用。
- **F6 已修复**：SQLite Fixture 新增 MarketCondition 分组键 `1/3` 的永久断言。
- **OpenPositions 口径已修复**：与 Ownership ActivePositions 对齐，只有 `managed_qty > 0` 且未关闭的受控仓位计为 Open Position；新增 Live Fixture 覆盖零数量未关闭记录。
- **F3 暂不修改**：Paper 仍复用既有 `ListTestResults` 全量统计路径。该行为并非 V3-7 新引入，个人数据规模下为非阻塞项；本阶段不扩大到重写既有策略统计服务。
- **F7 不处理**：策略模板 JSON 属用户工作区独立变更，不纳入 V3-7 修复。

复查后重新通过：`go test ./...`、`go vet ./...`、`go test -race ./service/outcomereview ./controllers ./routers`、`vue-tsc --noEmit`、`pnpm build`；最新 `dist` 已重新同步至后端 `static`。

