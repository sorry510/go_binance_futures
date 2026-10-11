# 币安合约自动带单 — Stage 2 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage实施方案.md` 的 **Stage 2 — Account-scoped Ownership、镜像和历史订单**（工作区未提交改动）。基线 `HEAD = 8a9fac5`（`feat: stage1`，分支 `feat/lead-trading`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库与 `strategy_templates/` 未被改动）
- **审查类型**：review-only
- **结论**：**Stage 2 后端离线实现完成，Gate 2 的五项在 SQLite 模拟层全部实证通过**；Schema 已升至 **19** 且迁移**失败安全/幂等/拒绝不损数据**均已实测。唯一未完成项是**真实 MySQL 迁移与线上 Main 回归**（需用户授权执行，记录已如实声明）。未发现 P0/P1；发现 1 项 P2（部署等待）+ 5 项 P3。
- **审计日期**：2026-10-09

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| Gate 2「历史数据完整」 | ✅ **实证**：真实 `SyncDatabase(19)` 在含历史行的库上回填 `account_id='main'`、行数不变、五张表逐表校验 |
| Gate 2「两个账户可同币同方向」 | ✅ **实证**：账户级唯一索引允许 main/lead 同 symbol+side 并存，同一账户内重复被 DB 拒绝 |
| Gate 2「任何关闭/撤单都无法跨账户」 | ✅ Ownership 全链路 `account_id` 过滤（service.go 29 处）+ 执行器绑定一致性检查 + Cancel 先校验账户（永久测试） |
| Gate 2「断线清理不会删除另一账户镜像」 | ✅ **实证（探针）**：主账户清理只删 main，lead 镜像行完整保留 |
| Gate 2「所有已存在 owner 的行为与当前一致」 | 🔶 **基本一致，含 1 处有意改进**：`feature/order.go::UpdateOrderStatus` 不再在仓位读取失败时删除开仓历史（原为静默删除，属安全改进，记录 §2-D 已声明） |
| 迁移可用性 | 🔶 **代码就绪，尚未执行真实升级**：二进制要求 Schema 19，库仍在 18 时启动会 panic 提示 `sync db`（预期行为，已核对 `main.go:229`） |
| 是否启用 lead | ✅ 未启用：无生产调用方绑定 lead Ownership/Executor/WS；lead 行只能由测试或后续 Stage 写入 |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 计划条目逐项核对（Stage 2 §112-125）

| # | 计划要求 | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | `futures_managed_positions`/`futures_managed_orders` 增 `account_id`，历史回填 main；全部 GetPosition/CloseQuantity/ClaimOrder/ActiveOrders/Reconcile/Cancel、冲突检查、SQL 查询带 account_id | ✅ | 模型新增 `AccountID orm:"size(16);default(main);index"`（`models/futures_managed.go` 等）；`service/futuresownership/service.go` 29 处 `Filter("account_id", accountID)`（逐一核对含 ClaimOrder/ActivePositions/ListOrders/CloseQuantity/UpdateOrderStatus/SetOrderStatus 等）；`service/futuresownership/account.go::BindAccount` 未绑定默认 main、未知 ID 报错 ✓ |
| 2 | 账户镜像表增 account_id + 联合唯一键；WS UPDATE/DELETE/清理按账户隔离 | ✅ | `futures_positions(account_id,symbol,side)`、`futures_orders(account_id,order_id)` **账户级唯一索引**（迁移创建，探针实证生效）；`feature/api/binance/user_data_mirror.go::persistFuturesAccountEvent` 按账户写入并拒绝未绑定 lead/未知账户；`feature/feature_userdata.go` 的全表清理、超时清理、REST 全量同步均带 `account_id='main'` |
| 3 | 旧 `order` 历史表增账户维度；历史查询/关联/统计按账户隔离，旧 main 数据无损 | ✅ | `models/tableStruct.go` 增 `AccountID`；`insertOpenOrder/insertCloseOrder` 显式写 main；`UpdateOrderStatus` 查询与关联均限定 main；`controllers/orders.go` 列表/删除固定 main；`service/outcomereview`、`service/systemhealth`、`feature/strategy/coin/common.go` 增加账户过滤 ✓ |
| 4 | 账户更新须在有效 DB 约束/事务中：同账户同 symbol/position_side 不允许多个活动 Owner；不只用全局 Go mutex；保持 fail-closed | ✅ | `ClaimOrder`/`ApplyFill` 使用 ORM 事务（记录 §2-B 明确"不声称覆盖多进程锁竞争"，口径诚实）；账户级唯一索引提供 DB 级约束；未知账户 fail-closed（永久测试 `TestStage2UnknownOwnershipAccountFailsClosed`） |
| 5 | `ClientOrderID` 全局唯一生成（带 account 前缀）+ DB 唯一约束；unknown 只 reconcile；不同账户相同交易所 OrderID 不串账 | ✅ | `execution.go:169-170`：lead 订单 ID 加 `lead_` 前缀，main 保持原格式（`aut_/rush_/notice_/fund_/agt_` + 随机后缀）；`futures_managed_orders.client_order_id` 原全局唯一约束保留；账户级唯一索引保证同交易所 OrderID 可跨账户共存但同账户不可重复（探针实证）；unknown 走既有 reconcile 逻辑不变 |
| 6 | 统一 `syncStrategyExitPositions`/`ensureAccountOpenSlotAvailable`/`submitManagedStrategyClose`/`ownershipAccountQuantities` 作用域；先覆盖 main 全部 owner 再接入 lead | ✅ | 这些函数均经 `Service.accountID()` 过滤；`NewAccountReconciler` 绑定一致性检查；**Lead 默认只对 `auto_strategy` Owner 对账**（不会误管其它功能 Owner）；Main Reconciler 行为不变 |
| 7 | 升级走 `sync db`；先备份、迁移、校验再启用；默认 lead disabled；迁移仅加字段/索引/表 | ✅（离线） | 迁移仅新增列/索引（`command/account_scope_migration.go`，无 DROP/无数据改写）；`appversion` 18→19；二进制启动门禁核对 `main.go:229-231`；记录 §4 给出备份/校验/维护窗口流程 |

---

## 3. 实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-Stage2-1** v18→v19 迁移（含脏数据）与恢复 | 造 v18 库 + 历史行（含未知账户 `rogue` 与重复槽位）→ 真实 `SyncDatabase(19)` | ✅ 迁移**拒绝**且 `config.version` 停留 **18**、全部行数不变、**未创建唯一索引**；修复数据后重试成功 → 版本 19、五张表全部回填 main、5 个复合索引齐全；两账户同 ID/同槽位合法、同账户重复被拒；**重复执行幂等** |
| **PR-Stage2-2** 重复槽位拒绝路径 | 仅造重复（无未知账户）→ 迁移 | ✅ 报错信息明确：`refuse unique mirror index futures_positions: 1 duplicate account-scoped slots exist; inspect and resolve before retry`；**不静默合并/删除**、版本停留 18、重复行保留待人工处理 |
| **PR-Stage2-3** 镜像清理隔离（Gate 2 项） | 造 main+lead 的仓位与订单镜像 → 调用真实 `deleteOldUserData()` | ✅ main 仓位与挂单被清理、main 已成交记录保留；**lead 的 2 条订单与 1 条仓位完整保留** |

> 说明：上述三条均**没有永久回归测试**覆盖（永久测试覆盖的是"回填成功/索引存在/重复升级幂等/账户隔离/伪造 WS 事件"等），建议把"迁移拒绝路径（未知账户、重复槽位）+ 清理隔离"固化进 `command/db_update_test.go` 与 `feature` 测试。

---

## 4. 其它核对要点（未使用探针的静态/测试证据）

1. **迁移接线正确**：`command/db_update.go` 在 `oldVersion < 19 && newVersion >= 19` 时依次执行 `accountScopeMigration`（回填+未知账户校验）→ `ensureAccountScopeIndexes`（5 个普通复合索引）→ `ensureAccountMirrorUniqueIndexes`（2 个唯一索引）✓；顺序保证了先补数据再加唯一约束 ✓。
2. **DSL 幂等与跨库差异处理得当**：SQLite 用 `CREATE [UNIQUE] INDEX IF NOT EXISTS`；MySQL 无该语法，改用 `information_schema.statistics` 存在性查询 + 逐条跳过 ✓（§6.4 的"部分成功后重跑补齐"经 PR-Stage2-1 实证 ✓）。
3. **MySQL 5.6 767 字节限制的修复**（记录 §6）：对 `account_id/symbol/side/position_side/order_id/owner/status` 使用**前缀索引**（最宽按 utf8mb4 估算 320 字节 ✓），并在建索引前用 `CHAR_LENGTH` 只读校验历史字段长度、超限即**拒绝升级**（不截断数据）✓；模型侧只声明新表尺寸、不修改既有 MySQL 列 ✓。`account_scope_migration_test.go` 覆盖 7 组索引的字节预算与 SQLite 全字段行为 ✓。
4. **两处顺带修复**（值得记录）：① `feature/feature.go:1109-1110` 挂单 SQL 加括号 —— 修复前 `account_id='main' and status='NEW' or status='PARTIALLY_FILLED'` 会把**其它账户**的 PARTIALLY_FILLED 行读成 main（真正的跨账户读串）；② `controllers/futuresOrders.go` 改为参数绑定 + 字段白名单 + 分页边界（page≥1、1≤limit≤500），修掉字符串拼接注入面 ✓。
5. **fail-closed 语义增强**：`feature/order.go::UpdateOrderStatus` 现在在 `GetTransformPositions()` 失败时**直接返回**（不再用 nil 仓位走"删除孤儿开仓历史"分支）✓ —— 属**有意的主账户行为变更**（改进），与"既有 owner 行为一致"的字面要求有差异，建议在 Stage 交付说明中保留该差异记录。
6. **lead 仍未启用**：`LeadAccountID` 仅出现在 account client 定义/校验/SAPI 守卫；无生产代码绑定 lead 的 Ownership/Executor/Reconciler，未启动 lead WS ✓ 符合计划边界。
7. **版本门禁**：`main.go:229` 在 `config.version < dbVersion` 时 panic 并提示执行 `sync db` ✓；`sync db` 使用同一 `appversion.DatabaseSchemaVersion`(=19) ✓（记录 §4.3 的"预期行为"属实）。

---

## 5. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race ./command ./service/futuresownership ./feature/api/binance ./service/outcomereview ./service/systemhealth ./controllers ./feature -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` | ✅ **全部 ok**（与记录 §2 声明一致，本轮独立复现） |
| `go test -count=1 ./appversion ./models ./spot/api/binance` | ✅ 全 ok |
| `go vet ./command ./service/futuresownership ./feature/api/binance ./service/outcomereview ./service/systemhealth ./controllers ./feature` | ✅ 干净 |
| 探针 PR-Stage2-1/2/3 | ✅ 全部通过 |
| 排除 `strategy_templates/research` 的批量编译 | ⚠️ 仍出现 `webnotification [setup failed]` 与根包批量标记（**单独/小批量运行均通过**，属本机批量并行环境抖动，与 Stage 2 无关；同 Stage 1 报告的 F3） |
| 完整 `go test ./...` | ⚠️ 仍红：未跟踪研究目录（重复 `main`、`config.json` 未知字段 `family`）——记录 §2 已如实说明并采用 `-skip` ✓ |
| 真实 MySQL 8/5.6 迁移、真实 Binance Lead Key | ❌ 未执行（用户侧；记录 §4/§6 已给出流程与容器自测结论） |
| 前端 | — 本阶段无前端改动 ✓ |

---

## 6. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | **P2（部署等待）** | 代码要求 Schema 19，当前真实库仍为 18 → **新二进制直接启动会 panic**（预期门禁），迁移与 Main 回归尚未在真实 MySQL 上执行 | 按记录 §4：备份 → 在备份库执行 `./go_binance_futures sync db` → 校验行数/回填计数/索引 → 用原 Key 验证启动与业务 → 再择维护窗口升级。注意 MySQL 5.6 前缀索引已由 §6 修复，若再遇 1071 需回报完整错误，**不要手工删表或关唯一约束** |
| **F2** | P3（理论边界） | 账户级唯一索引在 MySQL 上使用**前缀**（`symbol(32)`/`side(8)`/`order_id(32)`）：若历史值超过前缀长度，理论上不同 ID 可能被判重 | 已有 `CHAR_LENGTH` 只读校验 + 超限拒绝升级（不截断）✓，且 Binance ID 远短于前缀 ✓ → 可接受；建议在方案文档保留"前缀假设"一行，便于未来换库/字段变更时复核 |
| **F3** | P3（约定变更） | v19 迁移用 **Go 函数**（`command/account_scope_migration.go`）而非 `command/sql/version/19.sql` | 该选择是必要的（条件 DDL、存在性检查、拒绝路径无法用纯 SQL 跨库表达）；建议在方案文档/项目约定中写明"19 起允许 Go 迁移函数，但必须在版本循环内、失败不升版本、可重复执行" |
| **F4** | P3（测试缺口） | 迁移的**拒绝路径**（未知账户、重复槽位）与**主账户清理隔离**没有永久回归测试（本轮探针已实证） | 建议把 PR-Stage2-1/2/3 的关键断言固化进 `command/db_update_test.go` 与 `feature` 包测试 |
| **F5** | P3（行为差异记录） | `UpdateOrderStatus` 不再删除孤儿开仓历史（有意改进），与 Gate 2"既有 owner 行为一致"的字面表述存在差异 | 建议在 Stage 2 交付说明（或 Gate 结论）中显式列出该差异与理由，避免后续验收误判 |
| **F6** | P3（环境，非本阶段） | `go test ./...` 在本机仍红（未跟踪研究目录 + 批量 setup 抖动），Stage 1 报告 F3 的同一问题 | 建议后续把 `strategy_templates/research` 排除出静态模板扫描或迁出该路径，以便 Stage 3+ 能用全量测试作为 Gate |

**正面结论**：本阶段实现了"账户维度贯穿 Ownership/镜像/历史/统计"的完整落地，且对**失败安全**（拒绝而非静默合并、失败不升版本、幂等重跑、清理不越界）处理得相当严谨；跨库可移植性（SQLite 全字段 / MySQL 前缀+长度校验）考虑周到。

---

## 7. 审计边界与未验证项

- **未验证**：真实 MySQL（5.6/8）上的 DDL/索引创建、旧行计数、Main 启动与业务回归（用户侧执行；记录 §6 的 Docker 容器自测为其自述结论，本审计未复现）。
- **未验证**：真实 Binance 只读/交易能力（属 Gate 0-LIVE）。
- **未验证**：Lead 侧端到端（当前无 lead 数据写入方，属 Stage 4/5）。
- **未验证**：前端（本阶段无改动）。
- 本审计未发起任何真实 Binance 请求、未对真实数据库执行迁移、未改动用户仓库任何文件（探针仅存在于已删除的 `/tmp` 副本）。

---

## 8. 附：Stage 2 交付物清单与建议下一步

| 类别 | 文件 | 作用 |
| --- | --- | --- |
| 模型 | `models/futures_managed.go`、`futures_orders.go`、`futures_postions.go`、`tableStruct.go` | 五张表新增 `account_id`；新表字段尺寸声明（symbol 32/side 8/order_id 32） |
| 迁移 | `appversion/version.go`(18→19)、`command/db_update.go`、`command/account_scope_migration.go`(+test)、`command/db_update_test.go` | 版本循环接线、回填/校验/索引、MySQL 前缀与长度校验、v18→v19 永久测试 |
| Ownership | `service/futuresownership/account.go`、`service.go`、`execution.go`、`reconcile.go`(+`account_scope_stage2_test.go`) | 账户绑定、全查询账户过滤、事务化 Claim/ApplyFill、执行器/对账器绑定一致性、lead ID 前缀 |
| WS/镜像 | `feature/api/binance/user_data_mirror.go`(+test)、`index.go`、`feature/feature_userdata.go` | 账户化事件落库、陈旧事件保护、主账户清理/全量同步限定 main |
| Main 兼容 | `feature/feature.go`、`order.go`、`ownership.go`、`strategy/line/line_custom.go`、`strategy/coin/common.go` | main-only 镜像读取、挂单 SQL 修正、历史更新 fail-closed |
| 页面/统计 | `controllers/account.go`、`orders.go`、`futuresOrders.go`、`service/systemhealth`、`service/outcomereview` | 历史/统计按 main 隔离、参数化与分页边界 |
| 文档 | `币安合约自动带单-Stage2-实施记录.md` + 方案文档 §实施记录 | 交付与 Gate 结论、MySQL 5.6 修复记录 |

**建议下一步**：① 按记录 §4 在**备份库**执行一次真实 MySQL 迁移并校验（含 5.6 前缀索引路径）；② 把 F4 的三条探针断言固化为永久测试；③ 在文档中记录 F3（Go 迁移约定）与 F5（行为差异）；④ 之后进入 Stage 3（提取共享交易流程 + golden tests），继续保持 lead 不接 Executor、不开真单。
