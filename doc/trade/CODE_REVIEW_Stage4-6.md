# 币安合约自动带单 — Stage 4-6 代码审计报告（Gate 4 离线验收复核）

- **审计对象**：`doc/trade/币安合约自动带单-Stage4-6-验收计划.md` 与 `Stage4-6-实施记录.md` 对应的 **Stage 4-6 Mock 全流程与 Gate 4 验收**（工作区未提交改动）。基线 `HEAD = 473aa5d`（`feat: stage 4`，Stage 4-1～4-4 已提交；4-5/4-6 未提交）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库与 `strategy_templates/` 未被改动）
- **审查类型**：review-only（本轮是**对验收证据本身的独立复核**——记录 §五 亦明确"本轮自检不替代独立审计文档"）
- **结论**：**Gate 4 离线验收成立**（12 行验收矩阵逐行有命名测试 + 我独立复跑 9 包 `-race`/vet/build/diff 全绿）。**Stage 4-5 审计的 F2（持续性数据拒绝无账户级信号）已被修复**（新增 `FaultRulesUnavailable`）。我另补了一项**比 AST 扫描更强的结构证据**：生产二进制（根包）依赖闭包 **不含** `service/leadaccount`。未发现 P0/P1；新增 4 项 P3（含 3 项沿用）。
- **审计日期**：2026-10-10

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| 验收计划 §完成规则 ①（Mock/race/vet/build 通过） | ✅ 我独立复跑 9 包 `-race` **全绿**、`go vet` 干净、根包 `go build` 成功、`git diff --check` 干净（与记录 §四 一致） |
| §完成规则 ②（每个场景结果写入独立实施记录） | ✅ `Stage4-6-实施记录.md` 12 行矩阵逐行给出测试名 + 结论 + 数据来源说明 ✓ |
| §完成规则 ③（显式划定未覆盖项） | ✅ 记录 §五 六条：Stage 5 WS/重启/跨进程恢复与持久 Claim、notifier 接入、`RuntimeStatus` 副作用与 exchangeInfo 缓存、Stage 6/0-LIVE 身份与资产归属、Stage 7 真实 TP/SL 与动态过滤器、Stage 3 在线业务验收与 Stage 2 MySQL v19 迁移 ✓ |
| §完成规则 ④（只有 4-5/4-6 完成且无阻断缺陷才标"离线通过"，不得标实盘可用） | ✅ 记录与验收计划均只标 **Offline PASS**，并反复声明"不代表实盘可用" ✓ |
| 上轮 Stage 4-5 **F2**（持续性数据/规则拒绝一律 skip → 无账户级信号） | ✅ **已修复**：新增 `FaultRulesUnavailable = "lead_rules_unavailable"`（temporary/**warning**），在 `executor.go` 的空规则源与规则请求失败两处上报，且纳入"可由可信证据清除"的瞬态集合 ✓ |
| 上轮 Stage 4-5 **F1**（`RuntimeStatus()` 副作用）/ **F3**（notifier 同步执行） | 🔶 未处理但**已在记录 §五.3 明确留待 Stage 5/6** ✓（F1 被点名引用 ✓） |
| 生产是否可执行 Lead 写路径 | ❌ **结构性不可能**：`go list -deps .`（生产二进制依赖闭包）中 `service/leadaccount` 出现 **0 次** ✓ —— 比 AST 扫描更强（与命名/间接调用无关） |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 验收矩阵逐行复核（12 行）

| 矩阵行 | 记录指向的测试 | 我的复核 |
| --- | --- | --- |
| LONG/SHORT 独立开仓（账户正确、策略一致、白名单与额度独立） | `TestStage46AccountScopedMockOpenMatrixAndStickyReconcile`（4-3 行） | ✅ 四组合各写 Lead 独立 Ownership；首单后再次提交被待对账门禁拒绝（与 4-3 探针一致） |
| CLOSE_LONG/SHORT、TP、SL（优先级沿用 Main；不超 `min(managed,live)`；人工单不认领） | `TestStage46MockProtectiveStopsAndTargetsWithManagedCaps` + 4-5 的退出测试 | ✅ 四组合归属仅 Lead、满额关闭后 CLOSED；`min(managed,live)` 精确边界已在 4-3 探针实证 |
| MARKET / LIMIT（策略价格数量、Tick/Step/名义额/杠杆/保证金模式符合 Lead 过滤器） | 4-3 `rules`/`close_rules` 测试 + Stage 3 Golden | ✅ 规则校验唯一实现；LIMIT/方向开关另有我的 4-3 探针 |
| 部分成交 → 撤单 → 重复 WS/REST 回报（累计幂等、取消不代表 0 成交） | `TestStage46PartialFillCancelIdempotentAndAccountSeparation`（SQLite 内存） | ✅ 累计 0.3→0.3→0.5→0.5 受控净量恒 0.5；Main 同币同向独立（与 4-4 ledger 测试一致） |
| 拒单/429/418/网络超时/未知结果（查单不盲重试、阻断与告警原因正确） | `TestStage46DeterministicRejectionAndUnknownTimeoutDifferentLedgerOutcomes` + 4-4 恢复测试 | ✅ 明确拒单→FAILED 不查未知单；超时/未知→RECONCILE 且仅查一次、后续重试被阻断 |
| 普通单与 Algo STOP/TP 恢复（仅绑定 Lead 查询；触发单必须核对真实子订单） | `TestStage46RecoveryTerminalEvidenceCannotReleaseOpenOrCancelGate` + 4-4 Mock HTTP | ✅ 终态仍 `RequiresReconcile`；Algo 触发链核对已在 4-4 探针全表验证 |
| 凭证轮换/白名单失效/快照过期/缺 WS（不开新仓、不影响 Main、无敏感数据） | `TestStage46CredentialToRiskToExecutionRemainsLocked` + 4-1/4-3/4-5 相关 | ✅ 报告 JSON 无 Secret；轮换清理旧验证；事件结构仅 8 字段（4-5 探针） |
| Lead paused / risk_tripped / reconcile_required（不开新仓、退出意图仍判断、不确定写受阻） | `TestStage46SharedExitSignalsBothSidesEvenWhenLeadPaused` + 4-5 状态机测试 | ✅ 暂停/熔断不影响退出评估；`ExitWriteSafe=false` 于 reconcile/hard/temporary（4-5/4-6 探针） |
| Main 与 Lead 双 Runner 并行（Main 对齐 Golden、Lead 故障不影响 Main） | `TestStage46SharedStrategyConcurrentMainLeadIsolation` + `TestStage3MainMarketEntryGolden` | ✅ 并行各产生 2 个模拟信号；Lead 预检查失败后 Main 信号数不变（与 4-2 计数型测试一致） |
| 默认生产编译与公开入口扫描（无可调用的未授权 Submit/Cancel/Reset/Enable） | `TestStage46NoProductionLeadExecutionEntrypoint`、`TestStage46RepositoryProductionLeadWritesRemainUnreachable` | ✅ 两个 AST 扫描（目录级 + 全仓级，含 `seen >= 40` 防走空）；**我另加结构证明**：`go list -deps .` 不含 `service/leadaccount` ✓ |
| 测试结束/DB/配置/Git（无遗留进程、无真实写、未执行主库迁移） | 记录 §四/§六 | ✅ 本轮验证同样未产生后台进程、未触碰主库与配置 ✓ |
| （记录新增）交易所规则读取不可用的可见状态 | `TestStage46ExchangeRuleOutageHasVisibleFailClosedStatus` | ✅ **探针实证**：`lead_exchange_rules_unavailable` → temporary/`FaultRulesUnavailable`，状态 `account_unavailable`，0 次 Broker 提交 |

---

## 3. 本轮实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-S4-6-1** 新故障码分类 | `lead_exchange_rules_unavailable` vs `lead_exchange_rules_unverified` | ✅ 前者 temporary/`FaultRulesUnavailable`（warning）；**后者仍是 `skip`**（每单拒绝不升级账户级熔断）；定义表与固定顺序一致、恰出现一次（共 **15** 码） |
| **PR-S4-6-2** 瞬态恢复仅限可信证据 | 同时存在 rules-outage + 日亏损 + 待对账 | ✅ 状态先呈 `account_unavailable`；**新鲜+完整+已对账+同账户+WS 已验**的快照只清除 rules-outage，trip 与 reconcile 保留；其后状态为 `reconcile_required`、`ExitWriteSafe=false`；**陈旧/未对账/WS 未验/跨账户**证据均清除不了任何故障 |
| **PR-S4-6-3** 分类体系闭合性 | 15 个故障码 × 固定顺序 × 定义表 | ✅ 无重复、无遗漏、定义可达；空/残缺证据被忽略 |
| 结构证明 | `go list -deps .`（生产二进制依赖闭包） | ✅ `service/leadaccount` **0 次** → 生产二进制不可能包含或调用 Lead 写路径（与命名无关） |

> 既有永久测试（`stage46_integration_test.go` 9 例 + `account_trade_cycle_stage4_6_test.go` 4 例，加上 4-1～4-5 的全部测试）覆盖矩阵；**本轮探针补的是**：新故障码的**语义边界**（与同名前缀的 `unverified` 区分）、瞬态恢复的**四类证据门槛**（新鲜/完整对账/WS/同账户）、分类体系闭合性，以及比 AST 更强的**依赖闭包结构证明**。

---

## 4. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race ./service/leadaccount ./feature/api/binance ./service/futuresownership ./feature ./feature/strategy/line ./scanner ./service/backtest ./command ./controllers -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` | ✅ **9 包全部 ok**（leadaccount 2.05s、backtest 6.71s、feature 3.65s、command 3.46s 等） |
| `go vet`（同 9 包） | ✅ 干净 |
| 根包生产 `go build -o /tmp/s46_bin .` | ✅ 成功（临时二进制已删除） |
| `git diff HEAD --check -- '*.go'` | ✅ 干净 |
| `go list -deps . \| grep -c service/leadaccount` | ✅ **0**（生产二进制不含 Lead 执行包） |
| 探针 PR-S4-6-1～3 | ✅ 全部通过（含 `-race`）；副本去掉探针后 `./service/leadaccount` 原测试 ok |
| Schema / 迁移 / 路由 / 配置 / 前端 | ✅ 无变更（仍 v19） |
| 真实 Binance / 真实 Lead Key / 真实 WS / 真实下单 | ❌ 未执行（Gate 0-LIVE、Stage 5/6/7 均未开放） |
| 端口 3333 遗留进程 | ✅ 无（本轮同样未启动服务） |

---

## 5. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | P3（安全网强度） | 生产入口保护是**双层 AST 名字扫描**（5 个固定标识符：`NewLeadExecutionAdapter`/`ExecuteOpen`/`ExecuteManagedClose`/`NewReadOnlyExecutor`/`leadMockTradeRunner`）。若未来**新增**一个方法名（如 `PlaceLeadOrder`）或经接口/反射间接调用，扫描无法覆盖 | 建议补两项更强检查：① **依赖闭包断言**（本轮已实测为 0：`go list -deps .` 不含 `service/leadaccount`，建议固化进测试）；② 断言 adapter 的**导出方法集合**等于白名单（4-3/4-5 探针已用反射做过，可提为永久测试） |
| **F2** | P3（API 语义陷阱） | 记录 §三 自述：联合 TP/SL 测试首轮误用 `Ownership.GetPosition()`（只返回**活动**仓位）查询已关闭仓位而得到 `ErrNoRows`，改为 `ListPositions(..., false)` 修正 | 这是**测试**误用而非执行层缺陷 ✓，但暴露了真实陷阱：Stage 5 做权威对账/恢复时必须使用能返回**已关闭**记录的接口。建议在 Stage 5 的恢复设计文档中显式标注该 API 语义差异，避免真实实现踩同一坑 |
| **F3** | P3（措辞固化） | "Gate 4 离线 PASS ≠ 实盘可用"已在记录与验收计划中多处强调 ✓ | 建议把该声明作为 Gate 4 的**固定措辞**写入方案文档的 Gate 章节（避免后续被误引用为上线依据） |
| **F4** | P3（沿用） | 4-5 的 F1（`RuntimeStatus()` 写内部镜像，记录 §五.3 已承接）、F3（notifier 同步执行，需独立队列）；4-4 的 F1（恢复证据无落库）、F2（Algo FINISHED 无实际单）、F3（exchangeInfo 重拉）、F4（动态过滤器）；4-3 的 F3/F4；4-2 的 F1/F2 | 均未处理，建议随 Stage 5/6 收口 |
| F5 | P3（仍未做的外部确认） | 记录 §五.6 明确：Stage 3 的 **Main 在线业务验收** 与 Stage 2 的 **MySQL v19 迁移** 仍需另行确认；两者均为真实环境动作 | 与本阶段无关但不应遗漏——Gate 4 离线通过**不代替**这两项前置 Gate ✓（记录措辞与我一致） |
| — | 正面 | ① 验收矩阵把"真实只读检查"部分如实留白（Gate 0-LIVE）而**不冒充已通过** ✓；② 生产入口保护有**两套独立扫描**（目录级 + 全仓级）且带 `seen>=40` 走空保护 ✓；③ 主动修复了上轮审计的 F2，并把它变成"新增故障码 + 恢复条件 + 永久测试"的完整闭环 ✓；④ 记录显式声明"自检不替代独立审计"（本轮即为该审计）✓ |

---

## 6. 审计边界与未验证项

- **未验证**：真实 Lead Portfolio 的身份/资产归属与只读权限（Gate 0-LIVE）、真实私有 WS 与重启/跨进程恢复（Stage 5）、真实 Algo STOP/TP 与 `PERCENT_PRICE` 等动态过滤器（Stage 7）、Stage 3 的 Main 在线业务验收、Stage 2 的 MySQL v19 迁移（后两项记录已声明自行确认）。
- **未验证**：生产 notifier 接入（当前 nil，有意）。
- 本审计未发起任何真实 Binance 请求、未改动用户仓库任何文件（探针仅存在于已删除的 `/tmp` 副本）。

---

## 7. 附：Stage 4-6 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `service/leadaccount/stage46_integration_test.go` | 368（新） | 9 个跨模块联合 Mock 验收：凭证→风控→执行仍锁定、账户级开仓矩阵与 sticky 对账、部分成交/取消幂等与账户隔离、TP/SL 受控上限、风险与故障矩阵（0 次 Fake 提交）、终态证据不能解锁、Guard 不能把证据升格为实盘授权、规则读取不可见状态、拒单 vs 未知提交的账本差异 |
| `feature/account_trade_cycle_stage4_6_test.go` | 224（新） | Main/Lead 并行隔离、暂停下多空退出信号、生产入口目录级 AST 扫描、全仓级生产写路径 AST 扫描（含走空保护） |
| `service/leadaccount/state.go` / `fault.go` | 调整 | 新增 `FaultRulesUnavailable`（temporary/warning）：纳入固定顺序、瞬态可恢复集合、`lead_exchange_rules_unavailable` 映射 |
| `service/leadaccount/executor.go` | 调整 | 空规则源与规则请求失败两处上报该故障（原因码固定） |
| 文档 | — | `Stage4-6-验收计划.md`（矩阵 + 完成规则）、`Stage4-6-实施记录.md`（12 行结果 + 修复说明 + 未覆盖 Gate 清单） |

**建议下一步（按优先级）**：① 采纳 F1 的两项加固（依赖闭包断言 + adapter 导出方法白名单测试）；② 把 F2（`GetPosition` 只返回活动仓位）写入 Stage 5 恢复设计文档；③ 按计划推进 **Stage 5**（Lead 私有 UserData WS、ListenKey/重连、按 `account_id` 的持久权威对账与跨进程 Claim/额度预留、per-Key API 预算与告警接入、`pendingReconcile` 的安全释放流程），期间继续保持 `enabled=false`、无真单；④ 平行的外部确认项不要遗漏：Stage 3 Main 在线业务验收、Stage 2 MySQL v19 迁移、Gate 0-LIVE 的 Portfolio 身份与资产人工核对；⑤ 只有在 **Gate 0-LIVE + Stage 5/6/7** 全部按授权完成后，才具备讨论实盘启用的前提。
