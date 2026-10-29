# 事故修复复审（R2）+ 手动认领归属功能 审查报告

- **审查对象**：工作区未提交的改动（基线 `HEAD = 666b571`，`master`）。包含 ① 上一轮事故修复的**更新**；② **新增功能：未归属仓位的手动认领/归属调整**（模型 + 服务 + 路由 + 前端）
- **审查方式**：只读代码审查 + 隔离副本探针（v20 迁移探针、手动认领边界探针）+ 实跑回归；**未改动任何仓库文件**
- **结论**：
  - **A. 修复复审**：上一轮我提的 **F1（读失败中断整轮）、F5（退出列表丢弃新鲜持仓）、F6（坏行中断整轮）均已修复** ✓，并各配了回归测试。仅 F3（时钟基准）与 F4（无告警通知）仍为 P3。
  - **B. 新功能**：**设计严谨、实现正确，可以上线**（在完成 v20 迁移的前提下）。它把"人工修数据"做成了**受控、可审计、可回滚取证**的单一路径，且不触碰交易所。**发现 6 项 P3**（其中"入场价为 0 未拒绝"建议一并修掉）。
  - **C. 部署**：Schema 已升到 **20**；**新二进制在 `sync db` 之前无法启动**（版本门禁）；而 `static/` 已经换成新产物 → **在旧进程重启前，页面上的"手动修复"按钮会打到不存在的路由（404）**，属临时不一致，需按顺序发布。

---

## A. 事故修复复审（相对上一轮报告）

| 上轮项 | 级别 | 现状 | 证据 |
| --- | --- | --- | --- |
| **F1**：`pendingOpen` 检查中 `GetOrder` 读失败 → `return err` → 中断整轮交易 | P2 | ✅ **已修复** | 抽出 `autoStrategyOpenUnresolved(...)`：读失败 → `logs.Warning` + **返回 true（视为未解决）** → 调用方 `continue`，只跳过该腿，不中断整轮（`feature/ownership.go:161-179`）；新增 `TestUnresolvedOpenReadFailureOnlyProtectsAffectedLeg` ✓ |
| **F5**：退出列表只用"数量"、丢弃新鲜持仓行 → 镜像缺行时该轮无法平仓 | P3 | ✅ **已修复** | 新增 `feature/ownership_verified_snapshot.go`：`resolveOwnershipAccountSnapshot` 在"本地量 < 已记录量 / 强制刷新 / 本地行解析失败"时取 **Fresh REST**，并**返回新鲜持仓行**；`syncAutoStrategyOwnership` 与 `syncStrategyExitPositions` 的 `accountByKey`/结果循环改用 `verified.positions` ✓；新增 `TestVerifiedOwnershipSnapshotRestoresMissingExitRows` ✓ |
| **F6**：本地镜像坏行改为报错后，持久坏行会让每轮报错停摆 | P3 | ✅ **已修复** | 同一快照入口：本地解析失败 → warning + 用 Fresh REST 恢复（"preserving all other positions' exit evaluation"）✓；新增 `TestMalformedMirrorQuantityRecoversFromRestWithoutLosingExit` ✓ |
| **F3**：全量快照 `updateTime` 守卫混用"交易所事件时间 / 本机墙钟" | P3 | ⚠️ 仍在 | `shouldApplyFuturesFullSnapshot(currentUpdateTime, nowTime)` 未改；建议统一基准或加 ±1–2s 容差 |
| **F4**：仓位翻 `reconcile_required/closed` 只有 warning、无通知 | P3 | ⚠️ 仍在（部分补偿） | 本轮新增的手动认领页可作为人工闭环，但仍建议把该状态变化接入通知（限频 + 固定字段） |
| 其余（零观测两阶段 / 新鲜证据 / 镜像锁 / dirty→REST / 终态无数量） | — | ✅ 保持 | 上轮已验证；本轮回归全绿 |

**复审结论**：修复的主要缺口已闭合，**可以进入灰度**（仍需按 §C 的发布顺序）。

---

## B. 新功能：未归属仓位的手动认领/归属调整

### B.1 实现范围
| 层 | 文件 | 内容 |
| --- | --- | --- |
| 模型（v20） | `models/futures_ownership_adjustment.go` | `futures_ownership_adjustments` **追加型审计表**（14 字段：账户/币/方向/新 owner/账户量/认领量/原 owner/原状态/原 source_ref/原量/position_id/操作人/原因/时间） |
| 服务 | `service/futuresownership/manual_claim.go` | `ManualClaimPosition`（前置校验 + Fresh 交易所核对）→ `claimVerifiedManualPosition`（互斥锁 + 事务 + 审计 + CAS 写入）；`ManualClaims` 只读最近 100 条 |
| 接口 | `POST /agents/trade/ownership/manual-claim` | `AgentTradeController.ManualClaimOwnership`；`GET /agents/trade/ownership` 增加 `manual_adjustments` |
| 前端 | `src/views/ai/controlledTrade.vue` + `src/api/agent.ts` + locales | `owner=unmanaged && account_qty>0 && 无 account_error` 才显示"手动修复"；弹窗选 owner、数量、原因（确认后提交）；新增"手动修复记录"标签页 |
| 文档 | `doc/trade/手动认领未归属仓位.md` | 功能说明、安全约束、升级与部署顺序、人工测试清单 |

### B.2 安全与正确性核对（逐项）

| 检查项 | 判定 | 证据 |
| --- | --- | --- |
| 仅主账户 | ✅ | 服务层二次校验 `accountID == "main"`；**Lead 绑定被拒**（探针 PR-MC-3 ✓） |
| 可指派的归属类型受限 | ✅ | 仅 `auto_strategy` / `new_coin_rush` / `funding_rate`；`notice_auto_order`、`agent_trade`、空串、大小写变体、任意字符串**全部拒绝**（探针 PR-MC-1 ✓）。理由充分：这两类还依赖提醒配置/提案生命周期，简单加归属会"看似受管、实际退不出" ✓ |
| 必须用 Fresh 交易所持仓核对 | ✅ | 断言 `Account` 实现 `FreshPositions` → 读失败即拒；仓位不存在即拒；`expected_account_qty` 必须与交易所一致（相对 1e-9）；`quantity ≤ 实际`（探针 PR-MC-2：8 类非法输入 + 无仓位 + 读失败全部拒绝 ✓） |
| 不得重复认领/覆盖 | ✅ | 同 (account,symbol,side) 下：任何 active/suspended/reconcile_required 且 `managed_qty>0` → "already managed"；任何 `status != closed || managed_qty>0` → "reconcile first"；任何未终结订单 → 拒绝。探针：**第二次认领被拒** ✓；既有永久测试覆盖未决订单/已有 owner ✓ |
| 可复活历史 `closed/0` 行 | ✅ | 复用最新行并保留原 owner/status/qty/source_ref 进审计 ✓（`TestManualClaimRevivesClosedHistoricalOwnership` ✓） |
| 审计与变更同事务 | ✅ | 先 `tx.Insert(audit)` 拿 ID → 写 position → 回填 `audit.position_id` → `Commit` ✓；失败即回滚 ✓ |
| 并发 | ✅ | 复用 `mutationMu` + 对旧行的 **CAS**（id+account+status+managed_qty+updated_at，`affected!=1` 报"并发变更，刷新重试"）✓ |
| 审计不可篡改 | ✅ | 只有追加路径；唯一的 `Update(audit,"position_id")` 发生在同一事务内 ✓ |
| 操作人不可伪造 | ✅ | `ManualClaimInput.Operator` 为 `json:"-"`（浏览器无法提交）；控制器从 JWT claims 取（`middlewares/auth.go:88` 已 `SetData("user", claims)` ✓），`"web_admin"` 仅为兜底 ✓ |
| `source_ref` 与退出循环兼容 | ✅ | `manual_repair:<audit_id>`；`funding_rate` 额外前置 `funding_rate:manual_repair:<id>`，满足既有 funding 过滤前缀（否则"加了归属但被自动退出静默跳过"）✓（探针 PR-MC-1 + 永久测试 ✓） |
| 不触碰交易所 | ✅ | 全程只有 `FreshPositions`（GET）与 DB 写；无下单/撤单/改杠杆 ✓ |
| 前端只对未归属行开放 | ✅ | `openManualClaim`：`owner !== 'unmanaged' || !(account_qty>0) || ownership.account_error` 即返回 ✓（**account_error 时禁用**，避免用未知账户状态认领 ✓）；`expected_account_qty` 取自**账户实际数量行**（非受管行）✓；二次确认弹窗 ✓；历史标签页展示 `manual_adjustments` ✓ |
| Schema v19→v20 | ✅ **探针实证** | 造 v19 库 + 既有 ownership 行 → 删审计表模拟旧库 → `SyncDatabase(20)`：审计表 14 列齐全 ✓、既有行无损 ✓、无伪造审计行 ✓、**二次执行幂等** ✓ |

### B.3 发现的问题（均为 P3，建议顺手修）

| # | 级别 | 问题 | 建议 |
| --- | --- | --- | --- |
| **M1** | **P3（建议修）** | **入场价未校验为正**：`claimVerifiedManualPosition` 只拒绝 `NaN/Inf/<0`，`entryPrice == 0` 可通过 → 修复后的仓位 `EntryPrice=0`，退出逻辑（ROI/止盈止损）会算出异常收益率 → **可能立即触发一次非预期的市价平仓**。实盘中活跃仓位的 entryPrice 通常非 0，但不应依赖该假设 | 改为 `!finitePositive(entryPrice)` 即拒绝（或回退到 mark price 并记录）；一行改动 |
| **M2** | P3 | 控制器用 `json.Unmarshal` **无体积上限**（Lead 管理 API 有 4096 上限）；错误直接返回 `err.Error()`（可能含内部/SQL 细节） | 加 4KB 上限 + 固定原因码（如 `manual_claim_invalid_input`），细节只进日志 |
| **M3** | P3 | "交易所无该仓位"与"数量已变化"共用同一句错误（`account quantity changed...`），排查时容易误判 | 拆分两条固定原因：`position_not_found` / `account_quantity_changed` |
| **M4** | P3 | 若交易所对同一 symbol+side 返回**多行**，`actual` 会累加、`entryPrice` 取最后一行 | 断言匹配行数 ≤1（hedge 模式下应恒为 1），否则拒绝 |
| **M5** | P3 | 认领后下一轮可能**立即触发策略退出并真实市价平仓**（文档已说明） | 建议弹窗里再加一句强提示（"保存后可能立即被策略平仓"），或在最近一次修复后对该仓位延迟一个周期再纳入退出（可选，按你的偏好） |
| **M6** | P3 | `GET /agents/trade/ownership` 的 `manual_adjustments` 上限固定 100，无分页/时间过滤 | 记录量大后加时间范围/分页；当前可接受 |

---

## C. 发布与部署（重要）

1. **版本门禁**：`appversion.DatabaseSchemaVersion = 20`；`main.go` 在 `config.version < 20` 时 **panic 拒绝启动** → **必须先 `sync db`**。
2. **发布顺序**（与文档一致）：备份库 → 停旧进程 → 换新二进制 → `./go_binance_futures sync db` → 核对 `futures_ownership_adjustments` 表与 `config.version=20` → 启动 → 核对页面。
3. ⚠️ **当前状态**：`static/` 已经是**新构建产物**（`controlledTrade-DUv0q5o_.js` 等新哈希、旧哈希已删），而监听 3333 的仍是**旧二进制** → 若用户现在刷新页面点"手动修复"，请求会 **404**（不影响数据，但会报错）。建议**尽快按上述顺序重启**，或在重启前不要把新静态产物暴露给使用者。
4. 迁移探针已验证：升级**不会**动 `futures_managed_positions/orders` 既有数据 ✓，可重复执行 ✓。

---

## D. 实跑验证结果

| 项目 | 结果 |
| --- | --- |
| `-race`：`service/futuresownership`、`feature`、`feature/api/binance`、`service/binanceapiusage`、`scanner`、`feature/strategy/line`、`service/backtest`、`command`、`middlewares` | ✅ 全部 ok |
| `controllers`（跳过 4 个**既有**失败：静态模板 + 3 个策略模板 AI/校验） | ✅ ok（说明本轮改动未引入回归） |
| 专项：`TestManualClaim*`（6）、`TestObservedZero*`/`TestReconcilerSingleZero`/`TestTerminalReceipt*`/`TestOwnershipSnapshot*`/`TestFuturesFullSnapshot*`/`TestUnresolvedOpenReadFailure*`/`TestVerifiedOwnershipSnapshot*`/`TestMalformedMirrorQuantity*` | ✅ 全部 PASS |
| 探针：v20 迁移、认领白名单/输入守卫/重复与账户范围 | ✅ PASS（已随副本删除） |
| `go vet` / 根包 `go build` / `git diff --check` | ✅ 干净 |

---

## E. 边界

- 未连接生产库与交易所；未执行 `sync db`、未改任何数据；v20 迁移仅在临时 SQLite 上验证（**MySQL 8 未验证**，与既有约定一致，需在维护窗口实测）。
- 未验证真实账户的 `EntryPrice` 返回值范围（见 M1）。
- 未做真实浏览器点击测试（前端为源码 + 构建产物交叉核对）。
