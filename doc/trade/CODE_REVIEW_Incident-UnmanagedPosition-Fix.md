# 生产事故修复审查：自动策略仓位"变为未归属 → 无法自动平仓"

- **审查对象**：工作区未提交的事故修复（基线 `HEAD = 666b571`，`master`）。改动 6 个文件 +333/−93，新增 4 个回归测试文件（208 行）
- **审查方式**：只读代码审查 + 实跑新增测试与相关包全量回归（未改动任何文件）
- **审查类型**：review-only
- **结论**：**修复方向正确、实现基本正确，可以进入灰度观察**。它命中了至少 5 条能导致"仓位失去归属"的真实链路，并为每条补了针对性回归测试；8 个相关包 `-race` 全绿。**但我发现 1 项需要修正的健壮性问题（P2）与 4 项建议（P3）**，其中 P2 会使"某类数据库读失败"变成整轮交易停摆（不开仓也不平仓），建议上线前改掉。
- **附加判定**：`controllers` 有 3 个测试失败（`TestStrategyTemplateAITaskUsesRuntimeAndPreservesConversation`、`...ValidationAcceptsMarketConditionStringComparison`、`...AcceptsMultilineLetExpression`）——我已用 **HEAD 干净副本**复现，**与本次修复无关**，是既有问题。

---

## 1. 事故根因与修复的对应关系

"明明是自动策略开的仓，却变成未归属、平不了"，在本代码里只有一条主链：**`futures_managed_positions` 的 `managed_qty` 被清零或状态被置 `closed` → 退出循环不再把它当作受控仓位 → 不评估退出、不下平仓单**。修复覆盖了 5 条能把 `managed_qty` 清零/把记录驱逐出活动集合的链路：

| # | 失效链路 | 修复 | 证据 |
| --- | --- | --- | --- |
| **A** | **单次"观测到 0"就清零归属**：本地镜像/快照瞬时缺行或滞后，`ReconcilePosition(..., 0)` 直接把 `managed_qty` 置 0、状态 `closed`，此后永远不再平仓 | 新增 `ReconcileObservedPosition`：**第一次 0 只置 `reconcile_required` 并保留 `managed_qty`**；**≥10s 后再次观测到 0** 才关闭 | `service.go:494-560`（零分支 516-523、保留量 518） |
| **B** | **数量字符串解析失败被当成"仓位不存在"**：`PositionAmt` 为空/异常时 `continue` → 该仓位从快照消失 → 归属被清零 | 解析失败/NaN/Inf/`nil` 行一律**返回错误**（fail-closed），不再静默跳过 | `feature/ownership.go:618-640`、`reconcile.go:236-248` |
| **C** | **用缓存快照做"收缩/清零"这类破坏性变更**：镜像滞后时把 `managed_qty` 调小或清零 | ① 调用方先判断"是否需要新鲜证据"（`needsFreshOwnershipSnapshot`：仅当观测值 < 已记录 `managed_qty` 或刚确认成交时才刷新）；② 需要时改用 **Fresh REST** 持仓 | `feature/ownership.go:592-616`、`reconcile.go:114-127`、`reconcile.go:222-232` |
| **D** | **全量 REST 快照覆盖更新的 WS 事件**（镜像回退）→ 本地 `futures_positions` 的 `amount` 回退/丢失 → 归属被"确认"为 0 | ① 新增**全局镜像锁**把 WS 写入与全量快照应用串行化；② 逐行比较 `updateTime`：**比快照起点更新的行跳过**（`shouldApplyFuturesFullSnapshot`）；③ 删除陈旧行仍以快照起点 `nowTime` 为界（更新的 WS 行不被删） | `user_data_mirror.go:17-47`、`feature_userdata.go:99/115/128-140/199-203` |
| **E** | **全量快照"部分失败但仍被标记为可用"**：任一 `Insert/Update` 失败被忽略 → 镜像缺行却仍被信任 | ① 任何读写失败 → `return false`，且**开始写之前先把 `LastFullSyncAt` 置 0**（写期间不可用）；② 失败/过期/脏标记时**每 30s 重试全量同步**；③ WS 持久化失败 → **dirty 标记** → 本地读取立即回退 REST | `feature_userdata.go:36-50/63-67/210-218`、`user_data_mirror.go:36-47` |
| **F** | **终态回执缺成交量**：`FILLED` 但 `FilledQty=0` 被当作已完成 → 订单"完成"而 `managed_qty` 未增加 → 仓位无归属 | `FILLED`/`PARTIALLY_FILLED` 且 `FilledQty ≤ ε` → 状态置 **`reconcile`**（不终结），留给对账补齐 | `execution.go:277-292` |
| **G** | **未决开仓单 + 观测到 0**：刚提交/部分成交的开仓单会在快照之后继续成交 | 观测到 0 且存在 `Pending/Submitted/PartiallyFilled/Reconcile` 的开仓单 → **跳过本轮归属核对**（保留归属） | `feature/ownership.go:113-137`、`reconcile.go:140-158` |

> 这 7 条链路中，A/B/E 任意一条单独发生都足以复现事故；修复同时覆盖，属**根因级修复**而非表层打补丁。

---

## 2. 修复正确性逐项核对

### 2.1 零观测的两阶段处理（A）—— 正确
- 第一次 0：`status = reconcile_required`、`managed_qty` **不变** ✓。
- 关键点：`ActivePositions` 只按 `account_id + owner` 过滤（`service.go:411`，**无状态过滤**），且 `isLivePositionStatus` 把 `reconcile_required` 视为存活（`service.go:71`）→ **进入 `reconcile_required` 后仓位仍在退出循环里、`managed_qty > 0`，仍可被平仓** ✓（这正是"修复后即使再遇到一次异常 0，也还能平仓"的关键 ✓）。
- 第二次 0 需 `now - LastReconciledAt ≥ 10s` ✓；10s 内重复 0 不改变任何状态 ✓。
- 观测值重新出现：`reconcile_required → active`，且只在 `观测值 < managed_qty` 时收缩 ✓；**人工加仓（观测值 > managed_qty）不并入受控数量** ✓（保持原有不变量 ✓）。

### 2.2 "二次确认"的独立性 —— 依赖调用方，当前调用方成立
- 服务层自身只用 10s 去抖，**不区分数据来源**。其安全性来自调用方：`syncStrategyExitPositions` / `syncAutoStrategyOwnership` / `ReconcileOwner` 在"观测值 < 已记录量"时都会先取 **Fresh REST** 持仓（`verifiedOwnershipAccountQuantities` / `FreshPositions`）✓ → 第二次 0 来自**新鲜交易所读数**，因此属于权威证据 ✓。
- **风险**：这是隐式契约（见 F2）。若未来有新调用方把缓存值/镜像值直接传进来，10s 去抖就失去意义。

### 2.3 并发与一致性（A）—— 正确
- `ReconcileObservedPosition` 的 UPDATE 带 **CAS 条件**（`id + account_id + status + managed_qty + updated_at`）✓，`n != 1` 即报错要求"用新鲜快照重试" ✓ → 不会覆盖另一进程刚写入的成交 ✓。
- 全量快照的加锁范围覆盖"读-写-删-清脏标记"整个过程 ✓；WS 写入持同一把锁 ✓ → **不会出现"清脏标记之后又来了失败写入"却仍被信任**✓（注释也点明了这一点 ✓）。

### 2.4 终态回执（F）—— 正确
- `FilledQty > 0` 时先 `ApplyFill`（累计成交幂等）再定状态 ✓；`≤ ε` 时不加量、状态 `reconcile` ✓ → 下轮 `reconcileAutoStrategyManagedOrder` 会再查单补齐 ✓。

### 2.5 未决开仓单保护（G）—— 正确，但两处实现不一致（见 F1）
- `feature/ownership.go`：`GetOrder` 失败 → **返回错误**（中断整轮）✗。
- `reconcile.go`：同样场景 → 视为"未解决"→ `continue`（跳过该仓位）✓。
- 两者安全方向都对，但**中断整轮**的代价更大（见 F1）。

---

## 3. 实跑验证结果

| 项目 | 结果 |
| --- | --- |
| 新增回归测试：`TestObservedZeroDoesNotPermanentlyLoseOwnership` / `...ClosesOnlyAfterIndependentDelayedConfirmation` / `TestReconcilerSingleZeroCannotCloseManagedPosition` / `TestTerminalReceiptWithoutQuantityRemainsRecoverable` / `TestOwnershipSnapshotRequiresFreshExchangeProofForShrinking` / `...RejectsMalformedNumbers` / `TestFuturesFullSnapshotNeverRewindsNewerWSUpdates` | ✅ 全部 PASS（`-race`） |
| `go test -count=1 -race`：`service/futuresownership`、`feature`、`feature/api/binance`、`service/binanceapiusage`、`scanner`、`feature/strategy/line`、`service/backtest`、`command`、`middlewares` | ✅ 全部 ok |
| `controllers` | ⚠️ 3 个策略模板相关测试失败；**已在 `git archive HEAD` 干净副本复现** → **与本次修复无关**（属既有问题） |
| `go vet`（修复相关包） | ✅ 干净 |
| 根包 `go build` | ✅ 成功 |
| `git diff --check` | ✅ 干净 |

---

## 4. 需要处理与建议改进

| # | 级别 | 问题 | 建议 |
| --- | --- | --- | --- |
| **F1** | **P2（建议上线前修）** | `feature/ownership.go:120-128`：观测到 0 且有未决开仓单时，会调 `sharedOwnership.GetOrder(...)`；**一旦该读失败就 `return nil, err`**，导致 `syncStrategyExitPositions` 整体报错 → `runAccountTradeCycle` 本轮直接返回 → **整轮既不开仓也不平仓**。`reconcile.go` 的同类逻辑把读失败当"未解决"跳过（`continue`），两者不一致 | 把这里改成与 `reconcile.go` 一致：读失败 → 视为"有未决开仓"→ `continue`（保留归属、跳过本仓位核对），**不要中断整轮**；同时记 warning 日志 |
| **F2** | P3（隐式契约） | "10s 二次确认"的独立性完全依赖调用方传**新鲜**数量；服务层无法自证 | 在 `ReconcileObservedPosition` 注释/文档写明"调用方必须传新鲜交易所读数"；更强的做法是给方法加一个 `source` 参数或在内部自行取一次 `GetPositionFreshContext` 复核 |
| **F3** | P3（时钟基准） | `shouldApplyFuturesFullSnapshot(currentUpdateTime, snapshotStartedAt)` 比较的是**两个时钟**：行内 `updateTime` 来自 WS 事件的**交易所时间**，`nowTime` 是**本机墙钟**。客户端虽有 `TimeOffset` 同步，但本地时钟漂移/跳变时判据可能失准 | 统一基准（全量快照行也写交易所时间），或给比较加 ±1–2s 容差；并在日志中带上两个时间值便于排查 |
| **F4** | P3（可观测性） | 仓位从 `active` 翻到 `reconcile_required`/`closed` 目前只有 `logs.Warning`，没有通知 | 建议接入现有通知通道（该事件正是本次事故的信号）：`symbol/side/observed_qty/managed_qty/原因码`，并限频；这是"下一次同类事故"的最快发现手段 |
| **F5** | P3（收盘窗口） | 退出列表仍**只遍历本轮快照中存在的行**（`feature/ownership.go:157` 的 `for _, account := range accountPositions`）：若本地行被过滤掉（`amount = '0'`），该轮不会评估该仓位；虽然 `verifiedOwnershipAccountQuantities` 已经拿到了新鲜 REST 持仓，但**只用了数量、丢弃了行数据** | 小改动即可彻底消除这个窗口：当发生 "fresh 刷新" 时，把新鲜持仓合并进 `accountByKey`（`managedAccountPosition` 会用 `managed_qty` 覆盖 `amount`，只差 `MarkPrice` 等字段） |
| **F6** | P3（失败放大） | B 项把"解析失败"从静默跳过改成**返回错误**（方向正确 ✓），但一条持久坏行会让**每轮**都报错、停止开仓（也不平仓） | 建议：记录具体 `symbol/side/原始值` 并告警；考虑对"坏行"做一次定向修复（用 Fresh REST 覆盖该行），避免长期停摆 |
| — | 正面 | ① 修复**没有**引入"自动认领"这类越过归属边界的行为（仍不把 `unmatched` 仓位并入受控）✓；② 零观测改为"保留量 + 二次确认"，且 `reconcile_required` 仍在退出集合中 → **异常时依然能平仓** ✓；③ 全量同步从"忽略写错误"改为 fail-closed + 30s 重试 + dirty→REST 回退 ✓；④ 每条链路都有对应回归测试，测试名直接描述不变量 ✓ |

---

## 5. 上线建议（灰度观察）

1. **修掉 F1**（一行级改动），再发版——否则某次 DB 读抖动会让交易循环整轮停摆。
2. 发版后重点观察（建议加到看板/日志检索）：
   - `auto_strategy ownership verification:` 警告的数量与 `symbol`（新增日志，F4 已具备）；
   - `futures_managed_positions` 中 `status='reconcile_required'` 的行数与停留时长（>1 分钟即异常）；
   - `reconcile` 状态订单的数量（F 项链路）；
   - `futuresUserDataMirrorDirty` 为 true 的时长（E 项链路，dirty 期间会回退 REST，API 用量会上升）。
3. 事故仓位恢复：本次修复**不会自动把已清零的历史记录恢复**——已被置 `closed/managed_qty=0` 的旧记录需要一次性人工处理（例如：确认交易所仍有该仓位后，用 `ClaimOrder` + `ApplyFill` 或直接修正 `managed_qty`），否则那笔仓位仍不会被退出循环管理。**建议先查一次线上库**：
   ```sql
   SELECT id, account_id, owner, symbol, position_side, managed_qty, status, updated_at
   FROM futures_managed_positions
   WHERE account_id='main' AND status IN ('closed','reconcile_required')
   ORDER BY updated_at DESC LIMIT 50;
   ```
   与交易所当前持仓对照，人工恢复那些"交易所仍有量、记录却为 closed/0"的行（这是**一次性数据修复**，与本次代码修复互补）。

---

## 6. 审查边界

- 未连接生产库/交易所，未执行任何真实下单或数据修改；以上 SQL 仅建议由你在确认后执行。
- 未验证真实 WS 断线/重连与真实 `ACCOUNT_UPDATE` 时序（离线测试覆盖）；建议灰度期间按第 5 节观察。
- `controllers` 的 3 个失败为**既有问题**（HEAD 干净副本可复现），不在本次修复范围内，但建议另开任务处理。
