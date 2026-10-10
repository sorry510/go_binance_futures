# 币安合约自动带单 — Stage 4-5 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage4-5-实施记录.md` 与 `Stage4-5-开发计划.md` 对应的 **Stage 4-5 暂停、熔断与故障保护**（工作区未提交改动）。基线 `HEAD = 473aa5d`（`feat: stage 4`，Stage 4-1～4-4 已提交）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库与 `strategy_templates/` 未被改动）
- **审查类型**：review-only
- **结论**：**Stage 4-5 完成**（离线状态机 + 故障分级 + 告警事件接口 + 与开仓/平仓适配器的接线，零生产接线）。开发计划 §六 的 10 个验收场景中，自动化可覆盖的部分**均有永久测试**，本轮再用 4 组探针补齐"原因码映射全表 / 故障定义全表 / Guard 公开面与事件结构 / 通知去重与失败安全"。**Gate 4-5：通过；Gate 4 总验收按计划仍待 Stage 4-6**。未发现 P0/P1；新增 4 项 P3（含 3 项沿用）。
- **审计日期**：2026-10-10

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| 计划 §一 5 项开发任务（状态机 / 暂停与独立退出门禁 / 熔断分级 / 故障事件与脱敏 / 恢复条件与 Mock 回归） | ✅ 全部落地（`state.go` 317 / `fault.go` 56 + 接线 + 14 个永久测试） |
| 计划 §二 状态与优先级（`account_unavailable > reconcile_required > risk_tripped > disabled/paused > open_ready`） | ✅ 代码按此优先级聚合（`statusLocked`），且原因码按**固定顺序**输出 ✓ |
| 计划 §四 5 条安全不变量 | ✅ ① 开仓门禁与退出门禁分离（`permitsOpen` vs `ExitWriteSafe`）② **无 Enable/Resume/ResetPending**（探针反射断言 Guard 公开面恰为 `{Pause, RecordFault, Status}`）③ 状态/告警独立于 Main（Main 不创建也不调用 Guard；永久测试用计数断言 Main 不受影响）④ 告警失败不撤销阻断（探针实证）⑤ 只交付状态判断与 Mock 告警，无 WS/重启恢复/真实撤单 ✓ |
| 是否新增生产激活捷径 | ❌ 无：`NewLeadRuntimeGuard(nil)` 生产 notifier 为 **nil**（不发送任何告警）；`RiskController` 仍无 Enable/Resume；`ReadOnlyOrderBroker` 写仍恒锁 ✓ |
| DB / 配置 / 路由 / Schema | ❌ 全部未改（仍 v19）；无迁移、无前端改动 ✓ |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 计划 §六 验收场景逐项核对

| # | 场景（计划） | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | 默认状态/未授权/配置不完整 → 0 次 Lead 新订单、0 次生产写请求 | ✅ | `NewLeadRuntimeGuard` 默认 `disabled=true`；`permitsOpen()` 要求 `!paused && !disabled && observedReady && len(Reasons)==0`；4-3/4-4 的生产接线仍 0 命中 ✓（永久测试 `TestStage45GuardDefaultDisabledAndSeparateExitEvaluation`） |
| 2 | 主动暂停后出现 LONG/SHORT → 双方向均不得新开仓；Main 原流程不变 | ✅ | `PauseOpens()` 同时暂停 Guard 与 `RiskController.AllowNewOpens` ✓；`feature` 层永久测试断言 Lead 故障不停止 Main 循环与 Main 真实开仓 Mock ✓ |
| 3 | 暂停状态下已有受控仓位触发 TP/SL → 策略退出仍被评估；仅当持仓/API 数据可信时才允许安全处置 | ✅ | `ExitWriteSafe = !hard && !temporary && !reconcile`（暂停不影响）；永久测试 `TestStage45PausedOpenStillAllowsVerifiedManagedExit` ✓ |
| 4 | 日亏损/回撤达到阈值 → 持续阻断开仓，不能被新一轮循环覆盖 | ✅ | 日亏损/回撤为 `trip`（sticky）；仅 `observeFreshEvidence` 能清**瞬态**类，且它显式**不**清 trip/identity/reconcile ✓（永久测试 `…RiskThresholdTripNeverClearsWithUTCDateOrHealthyData`） |
| 5 | WS 过期、余额/持仓快照缺失 → `account_unavailable`，原因与告警明确，禁止不确定写 | ✅ | `lead_ws_not_healthy`→temporary/WS、快照/余额类→temporary/SnapshotInvalid；`hard || temporary` → `account_unavailable` ✓；退出写被 `permitsExitWrite()` 拒绝 ✓（永久测试 `…RateLimitAndProtectionFaultBlocksExitWrite`） |
| 6 | 下单超时/Algo 状态不确定 → `reconcile_required` 持续存在，反复循环与并发调用不重试 | ✅ | 预留 `pendingReconcile` 时 `RecordFault(FaultPendingReconcile)`；`reconcile` → `reconcile_required` 且不被瞬态证据清除 ✓；并发由实例锁串行 ✓（永久测试含 race） |
| 7 | 仅得到一条终态订单查询结果 → 不能自行解除 pending | ✅ | `RecoveryEvidence.RequiresReconcile` 恒 true（4-4 探针）+ Guard 无解除接口（本轮探针复核）✓ |
| 8 | 告警失败/重复故障 → 阻断不丢失；重复告警去重，关键故障可升级 | ✅ **探针实证** | 首次即发、2 分钟内去重、超窗重发、**429→418 严重度升级立即通知**、notifier 返回错误后状态不变 ✓ |
| 9 | Lead 故障与 Main 并发 → 只改变 Lead 状态；Main 策略/订单路径完全独立 | ✅ | Guard 只接受 `AccountID=lead`（Main 报告被拒 ✓ 探针实证）；`feature` 永久测试计数断言 Main 不受影响 ✓ |
| 10 | Race、Vet、Build、默认拒绝及恢复边界全部通过；无实时 Lead 激活入口 | ✅ | 本轮独立复跑：5 包 `-race` 全绿、`go vet` 干净、根包 `build` 成功、`git diff --check` 干净 ✓ |

---

## 3. 实现逐项核对

### 3.1 `state.go`（317）
- **状态/故障码/类别/严重度**：6 状态、14 故障码、5 类别、3 严重度，命名与计划表一致 ✓。
- **默认拒绝**：`NewLeadRuntimeGuard` 构造即 `disabled=true`；`Pause()` 单向（代码注释明确"停止暂停**故意不导出**"，未来需单独评审授权与证据）✓。
- **聚合优先级**：`statusLocked` 先取 disabled/paused/observedReady，再由 `hard||temporary → account_unavailable`、`reconcile → reconcile_required`、`trip → risk_tripped` 覆盖 ✓；`Reasons` 按 `allFaultCodes()` **固定顺序**输出（可稳定序列化）✓；`ExitWriteSafe`、`CanAttemptOpen` 明确标注为**模型观测**、非实盘许可 ✓。
- **信任边界**：`observeBoundState`/`observeFreshEvidence`/`permitsOpen`/`permitsExitWrite` **均为非导出** ✓（外部无法直接把账户"置为 ready"）；`observeFreshEvidence` 要求 lead 账户 + WS 已验 + 位置/订单/PnL 完整 + `Reconciled && !UnknownOrders` + Hedge + 余额有限 + 10s 新鲜度 + PNL 日界与 `PNLAsOf ≥ CapturedAt` ✓ —— 即"可信快照"的定义与 4-2 风控证据口径一致 ✓，且**只清瞬态五类** ✓。
- **事件结构**：`LeadFaultEvent` 仅 8 个字段（动作/账户/码/严重度/类别/前后状态/时间戳），**结构上无法携带** key/secret/symbol/URL/原始错误 ✓（本轮探针断言字段集合 + 序列化无敏感词）。

### 3.2 `fault.go`（56）
- `ClassifyLeadOpenRejection`：把 Stage 4-2 的**固定**原因码映射到（类别，故障码）；日亏损/回撤→trip、账户身份/绑定→hard、快照/WS 类→temporary、pending→reconcile，**普通交易拒绝（额度/保证金/槽位/白名单/未授权等）一律 skip** ✓ —— 与计划"不把正常无候选或单笔拒绝升级为账户级熔断"完全一致 ✓（本轮 37 条原因码全表实证）。
- `recordRiskDecision`：按故障码去重后逐条 `RecordFault` ✓；主账户与未知码被忽略 ✓。

### 3.3 `executor.go` / `close.go` 接线
- 适配器构造时创建独立 Guard（notifier=nil）✓；`RuntimeStatus()` 只读诊断、`PauseOpens()` 单向、`ReportLeadFault()` 只接受固定码 ✓（三者均无实盘授权能力）。
- `checkOpenLocked`：**先** `observeBoundState` + `permitsOpen`（拒绝时把 Guard 状态名作为原因码返回）→ 再原有六布尔门禁 → 快照失败记 `FaultSnapshotInvalid` → 成功后 `observeFreshEvidence` → 风控决策后 `recordRiskDecision` ✓ —— 即"状态机 → 账户门禁 → 证据 → 风控"四层顺序正确 ✓。
- `ExecuteOpen`/`ExecuteManagedClose`：在预留 `pendingReconcile` 时记 `FaultPendingReconcile` ✓；平仓在 `pendingReconcile` 之后、风险门禁之前增加 `permitsExitWrite()` 检查 ✓（`account_unavailable`/`reconcile_required` 时禁止退出写 ✓ 符合"故障时不盲写"✓）。
- `ReportLeadFault`/开仓路径共用实例锁 ✓（故障报告与下单意图串行，避免"提交前最后一刻的故障报告被竞争状态转换吞掉"✓）。

---

## 4. 本轮实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-S4-5-1** 原因码映射全表 | 37 条原因码（含 4-2 全部固定码 + 未映射码 + 空串 + 未知文本） | ✅ 风险/账户/对账类映射正确；**普通交易拒绝与所有未映射/未知原因一律 `skip`**（不触发账户熔断）；同一决策同时含 skip 与 trip 时**只记 trip**；Main 账户报告与未知故障码被拒且状态不变 |
| **PR-S4-5-2** 故障定义全表 | 14 个故障码的类别/严重度 + 固定顺序清单一致性 | ✅ 全部符合；**418 升级为 critical** ✓；固定顺序清单与定义表**无重复/无遗漏** |
| **PR-S4-5-3** Guard 公开面与事件结构 | 反射枚举方法/字段 + 序列化检查 | ✅ 公开面恰为 `{Pause, RecordFault, Status}`（**无 Enable/Resume/ResetPending**）；事件字段恰为 8 个允许字段；payload 不含 `api_key/secret/signature/symbol/http/url` |
| **PR-S4-5-4** 通知去重/升级/失败安全 | 注入会失败的 notifier + 受控时钟 | ✅ 首次立即、2 分钟内去重、超窗重发、429→418 严重度升级立即通知；**notifier 失败不改变故障与 `ExitWriteSafe=false`**；临时故障使状态呈 `account_unavailable`（优先级正确） |

> 既有永久测试（`state_test.go` 12 例 + `feature/account_trade_cycle_stage4_5_test.go` 2 例）已覆盖：默认拒绝与退出评估分离、故障矩阵优先级与 sticky、瞬态恢复需新鲜完整证据、风险熔断不因 UTC 换日或健康数据复位、skip 不触发账户熔断、事件脱敏/去重/通知失败不解锁、并发读写（race）、执行器故障与暂停门禁、暂停下允许受控平仓、不安全退出被拒且不触碰人工仓位、限流/保护类故障阻断退出写、仅已知故障码与告警范围、以及 Lead 故障不停止 Main。**本轮探针补的是**：原因码/故障定义的**全表回归**、Guard 公开面与事件**结构**安全、以及通知**升级与失败安全**的精确行为。

---

## 5. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race ./service/leadaccount ./feature ./feature/api/binance ./scanner ./service/futuresownership -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` | ✅ 全部 ok（leadaccount 3.08s、feature 3.45s、api/binance 3.24s、scanner 2.94s、ownership 3.77s） |
| `go vet ./service/leadaccount ./feature ./feature/api/binance` | ✅ 干净 |
| 根包 `go build -o /tmp/s45_bin .` | ✅ 成功（未在仓库留产物） |
| `git diff HEAD --check -- '*.go'` | ✅ 干净 |
| 探针 PR-S4-5-1～4 | ✅ 全部通过（含 `-race`）；副本去掉探针后原测试 ok |
| Schema / 迁移 / 路由 / 配置 / 前端 | ✅ 无变更（仍 v19），与记录 §5 一致 |
| 真实 Binance / 真实 Lead Key / 真实启停 | ❌ 未执行（Gate 0-LIVE 与 Stage 7 授权前均不执行） |

---

## 6. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | P3（接口语义） | `RuntimeStatus()` 名为"只读诊断"，但内部会调用 `observeBoundState(risk.state)` **写** Guard 的 `disabled/observedReady` 镜像 → 读操作带副作用；若 Stage 6 面板高频轮询，会顺带刷新内部状态 | 建议：① 在文档/注释标注该副作用（当前注释只强调"不是许可"）；② 或拆成"纯读视图（基于上次观测）"与"刷新观测"两个接口，避免调用方误以为它无副作用 |
| **F2** | P3（可观测性缺口） | 若某类**持续性**数据/规则拒绝未纳入映射（如 `lead_exchange_rules_unavailable`、`lead_order_type_not_supported`、`lead_margin_mode_invalid`、`lead_exposure_invalid`、`risk_context_unavailable`），会一律 `skip` → Guard 不会给出任何账户级信号，Stage 6 面板可能呈现"账户健康但一直不开仓" | 好处是不误升级熔断 ✓；建议为"持续性数据不可用"补一个 `temporary` 类映射（或至少在原因码清单中标注"skip 但需展示"），避免运维误判 |
| **F3** | P3（通知实现） | `LeadFaultNotifier` 在调用线程**同步**执行（记录 §5.5 已声明），生产 notifier 为 nil（不发送）✓ | 建议 Stage 5/6 接入时提供**有界非阻塞**实现，并明确是否在交易热路径上（当前路径持实例锁调用 notifier ✗ 若未来 notifier 阻塞会拖慢开仓/平仓）——建议在接入前把通知放到独立队列 |
| **F4** | P3（沿用） | 4-4 的 F1（恢复证据无落库路径）、F2（`Algo FINISHED` 无实际单一律 uncertain）、F3（平仓每次重拉 exchangeInfo）、F4（动态过滤器待实测）；4-3 的 F3/F4；4-2 的 F1/F2 | 均未处理，建议随 Stage 5/6 收口 |
| — | 正面 | ① 状态机把"开仓许可/退出可写/需权威对账"三种性质**分开建模**且默认拒绝，出口只有单向 `Pause`；② 瞬态（可被新鲜完整证据清除）与持久（trip / reconcile / identity / protection）**分层明确**，且恢复口径与 4-2 风控证据完全一致（同一套完整性/新鲜度判定）；③ 事件结构**结构性**无法泄露凭证或原始错误；④ 通知失败不撤销阻断、Main 完全隔离 ✓ |

---

## 7. 审计边界与未验证项

- **未验证**：真实 Lead 账户的 429/418、WS 健康、凭证变更等**真实故障**触发路径（本阶段仅离线/Mock；计划 §一 已把生产 WS 与重启恢复交 Stage 5）。
- **未验证**：通知链路的真实接入（生产 notifier=nil ✓ 有意）。
- **未验证**：跨进程/重启后的状态持久化与权威恢复（Stage 5）。
- **未验证**：前端（无改动）、MySQL（无变更）。
- 本审计未发起任何真实 Binance 请求、未改动用户仓库任何文件（探针仅存在于已删除的 `/tmp` 副本）。

---

## 8. 附：Stage 4-5 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `service/leadaccount/state.go` | 317（新） | `LeadRuntimeGuard`：6 状态 / 14 故障码 / 5 类别；默认 disabled；单向 Pause；固定顺序原因码；瞬态恢复仅由可信证据清除；事件结构安全 |
| `service/leadaccount/fault.go` | 56（新） | 风控原因码 → 故障类别映射（普通拒绝一律 skip）；`recordRiskDecision` 去重登记 |
| `service/leadaccount/executor.go` | 接线 | Guard 构造、`RuntimeStatus`/`PauseOpens`/`ReportLeadFault`、开仓前四层顺序检查、预留时登记 pending 故障 |
| `service/leadaccount/close.go` | 接线 | 退出门禁 `permitsExitWrite()`；预留时登记 pending 故障 |
| `service/leadaccount/state_test.go` | 302（新） | 12 个永久测试（默认拒绝/优先级 sticky/瞬态恢复/UTC 不复位/skip 不熔断/脱敏去重/并发/执行器门禁/暂停仍可安全退出/不安全退出拒绝/限流与保护类阻断/仅已知码与告警范围） |
| `feature/account_trade_cycle_stage4_5_test.go` | 54（新） | Lead 故障不停止 Main 交易循环；开仓阻断不删除安全退出/同步对账钩子 |
| 文档 | — | `Stage4-5-开发计划.md`（规格 + 10 项验收场景）、`Stage4-5-实施记录.md`、`Stage4-6-验收计划.md` 与方案文档同步 |

**建议下一步**：① 进入 **Stage 4-6 联合验收**（计划要求"跨模块联合 Mock Gate 4 验收矩阵 + 代码审计；Gate 4 在此之前仍为未通过"）；② 处理 **F1**（`RuntimeStatus` 副作用语义）与 **F2**（持续性数据拒绝的账户级信号），并把 **F3**（通知放独立队列）写入 Stage 5 接入要求；③ Stage 5 落实 Lead 私有 UserData WS、重启/跨进程权威对账与持久 Claim、per-account API 预算与告警接入；④ 真实只读/写能力仍等 **Gate 0-LIVE** 与 Stage 7 人工授权，本阶段结论不得当作交易许可。
