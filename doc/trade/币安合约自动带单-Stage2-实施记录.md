# Binance 合约自动带单 — Stage 2 实施记录

> 日期：2026-10-09  
> 状态：**Stage 2 后端离线实现完成，专用 Lead 实盘 Gate 仍关闭**。  
> 实际数据库升级：**未执行**；代码所需 Schema：**19**（原始 Schema：18）。
>
> 任务边界：仅账户维度 Ownership、仓位与订单镜像、对账、历史查询、迁移和离线验收；没有启用 Lead 自动交易，没有运行带单 WS 连接，没有修改策略 JSON、`app.conf`、前端或者真实数据库，没有执行真实下单/撤单，也没有 Git commit/push。

## 1. 已完成的实现

### 2-A. 数据模型与 v19 迁移

新增以下五张表的 `account_id` 字段（varchar(16)，ORM 默认 main，含普通索引）：

| 表 | 主要作用 |
|---|---|
| `futures_managed_positions` | 受控持仓所有权账本 |
| `futures_managed_orders` | 受控订单所有权及 ClientOrderID |
| `futures_positions` | User Data/REST 账户持仓镜像 |
| `futures_orders` | User Data/REST 账户挂单/成交镜像 |
| `order` | 合约开平仓历史及收益关联 |

- `appversion.DatabaseSchemaVersion` 从 18 升至 19，保留唯一的 `./go_binance_futures sync db` 升级命令。
- ORM 同步完成新字段后，`command/account_scope_migration.go` 显式将五张表历史空值账户回填为 `main`，校验未知账户 ID，并创建五个幂等的复合查询索引，并对 `futures_positions(account_id,symbol,side)`、`futures_orders(account_id,order_id)` 创建**账户级唯一索引**（MySQL / SQLite）。若发现旧镜像存在重复逻辑行，则迁移拒绝创建唯一索引，**不会擅自删除/合并历史数据**，需先备份核对处理。
- 原有 `futures_managed_orders.client_order_id` 全局唯一约束保留；未自动根据历史成交认领任何持仓。
- `command/db_update_test.go` 新增 17→18→19 的 SQLite 升级、历史数据回填、索引和重复升级测试，已通过。
- **注意**：SQLite 测试不能替代真实 MySQL 8 的迁移验证；应在备份后单独验证 DDL、旧行计数、索引和实际主账户恢复。尤其需要预检查两个镜像表中的重复 `account_id + symbol + side` / `account_id + order_id` 组合；不允许升级工具静默清除重复记录。

### 2-B. Account-scoped Ownership、Broker 与事务

- `service/futuresownership/account.go`：`BindAccount(main/lead)`；未绑定的 `DefaultService` 和原有零值 Service 均明确按 `main` 兼容。未知账户 ID 直接失败。
- `service/futuresownership/service.go`：`ClaimOrder`、`ensureSlotAvailable`、`ApplyFill`、`getPosition`、`GetPosition`、`GetOrder`、`ActivePositions/ActiveOrders`、`ListPositions/ListOrders`、`CloseQuantity`、`ReconcilePosition`、`SuspendPosition` 及订单状态更新均按服务自身 `account_id` 过滤。
- `ClaimOrder` 和 `ApplyFill` 采用数据库事务，分别将校验/认领与订单/仓位数量更新放在同一原子提交边界；保留内存 mutex 用于当前单进程的竞争保护。**没有声称覆盖未来多进程写同一 DB 的所有锁竞争**。
- 保留原有受控数量上限、不接管手工仓位、重复累计成交幂等和未知结果需 reconcile 规则。
- `service/futuresownership/execution.go` 新增 `NewAccountExecutor`，绑定的 Ownership 与 BinanceOrderBroker 账户必须一致；Lead 不能借用无账户绑定的 Main Broker。Lead 自动产生的 ClientOrderID 加 `lead_` 前缀。
- `Cancel` 在请求交易所之前校验订单记录的 `account_id`；`Reconcile`、`ApplyObservedExchange`、`Execute` 同样检查 Broker/Ownership 绑定。
- `service/futuresownership/reconcile.go` 新增 `NewAccountReconciler`，其账户读取源、Ownership 与 Executor 必须绑定一致；Lead 默认仅对 `auto_strategy` Owner 对账，不会无意管理其他功能的 Owner。Main 现有 Reconciler 行为不变。

### 2-C. WS 镜像、旧查询与事件数据路径

- `feature/feature_userdata.go` 中原 Main 全表清理、超时清理、REST 全量同步 SELECT/INSERT/UPDATE 等均添加 `account_id='main'`，不会清理 Lead 镜像。
- `feature/api/binance/user_data_mirror.go` 新增可复用 `persistFuturesAccountEvent(accountID, client, event)`：对 ACCOUNT_UPDATE、ORDER_TRADE_UPDATE、ACCOUNT_CONFIG_UPDATE 进行按账户写入/更新及缓存失效；拒绝未知账户和未绑定 Client 的 Lead 事件；时间早于已持久化事件的旧事件不覆盖较新记录。
- `feature/api/binance/index.go::WsUserData` 原 Main WS 调度仍然运行，并调用上述 `main` 写入路径；ListenKey 续期/重连逻辑保持现有方式。**尚未启动 Lead WS**。
- `feature/feature.go`、`feature/strategy/line/line_custom.go`、`controllers/account.go` 的本地镜像读取继续只访问 `main`。原本挂单 SQL 的 `AND/OR` 优先级问题已修正为带括号条件。
- `controllers/futuresOrders.go` 旧订单列表改为带参数绑定的 SQL，`main` 账户过滤不能通过用户传入的筛选字符串绕过；列表分页加入安全边界。
- 对 Main 本地镜像编辑/删除操作增加账号过滤，防止通过 ID/JSON 改写 Lead 镜像账户。

### 2-D. 历史订单与外围关联

- `feature/feature.go::insertOpenOrder/insertCloseOrder` 显式写 `account_id=main`，开平仓历史匹配也只选 Main。
- `feature/order.go::UpdateOrderStatus` 只修复 Main 历史数据，Main 仓位读取失败不再删除“未找到对应持仓”的开仓历史。
- `controllers/orders.go` 历史列表、筛选删除、删除关联平仓订单的 SQL 均固定在 Main。
- `feature/strategy/coin/common.go` 原历史成交选币分支限定 Main。
- `service/systemhealth/service.go` 和 `service/outcomereview/service.go` 原 Main 受控交易统计增加账户过滤。对应测试夹具明确指定 `main`。

## 2. 自动化验证及结果

| 验收项目 | 结果 |
|---|---|
| v19 SQLite `SyncDatabase`，历史空 ID 回填、复合唯一约束及重复执行 | 通过 |
| 同币同方向 Main/Lead 并存，独立开仓/受控数量/暂停/对账 | 通过 |
| 重复累计成交不重复增加 managed_qty | 通过 |
| Main/Lead 跨账户订单读取/状态修改拒绝 | 通过 |
| 未知账户及未绑定 Broker fail-closed | 通过 |
| Main/Lead 模拟 WS，相同 symbol / OrderID 独立订单与持仓镜像 | 通过 |
| 旧 WS 事件不覆盖新数据；Lead 无绑定 Client 时拒绝 | 通过 |
| Ownership / Command / Binance API / OutcomeReview / SystemHealth / Controllers / Feature `-race` | 通过（排除一个已有研究模板静态加载测试，见下） |
| 排除 `strategy_templates/research` 的后端 `go test ... -run '^$'` 编译 | 通过 |
| 完整 `go test ./... -run '^$'` | 未通过：研究脚本存在重复 main / 常量声明；不属于 Stage 2 修改 |
| 真实 MySQL 8 升级、真实 Binance Lead API 权限验证 | **未执行 / 待验收** |

测试过程出现 macOS `ld: malformed LC_DYSYMTAB` 的链接器警告，但上述通过的目标测试最终退出状态为 0。

另有原项目 `controllers.TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals` 会把研究目录的 `config.json` 当作策略模板而报 `unknown field "family"`；本阶段不修改 `strategy_templates/research` 以避免破坏并行量化研究成果，Race 回归中已明确跳过这一项。

## 3. 交付路径

- 账户模型：`models/futures_managed.go`、`models/futures_orders.go`、`models/futures_postions.go`、`models/tableStruct.go`。
- 数据库迁移：`appversion/version.go`、`command/db_update.go`、`command/account_scope_migration.go`、`command/db_update_test.go`。
- Ownership：`service/futuresownership/account.go`、`service.go`、`execution.go`、`reconcile.go`、`account_scope_stage2_test.go`。
- WS/镜像：`feature/api/binance/user_data_mirror.go`、`user_data_mirror_stage2_test.go`、`index.go`、`feature/feature_userdata.go`。
- Main 业务兼容：`feature/feature.go`、`feature/order.go`、`feature/ownership.go`、`feature/strategy/line/line_custom.go`、`feature/strategy/coin/common.go`。
- 账户与历史页面、统计：`controllers/account.go`、`controllers/orders.go`、`controllers/futuresOrders.go`、`service/systemhealth`、`service/outcomereview`。

## 4. 真实升级前的必要工作

1. **先备份**目标 MySQL（本地 5.6 或部署端 8）的五张业务表及相关 `config`/Schema 信息。
2. 在预期部署代码配套的备份库上执行 `./go_binance_futures sync db`，检查相应 MySQL 版本的复合索引、历史行总数与 `account_id='main'` 回填计数，并用原 Main Key 验证启动/仓位/挂单/历史展示。
3. 再按明确的维护窗口升级真实运行环境。代码已要求数据库 Schema **19**：如数据库仍停留在 18，正常启动将按项目既有版本门禁拒绝，**这是预期行为**，不能靠改 `app.conf` 跳过。
4. Stage 2 仍不能启动 Lead 自动交易。Stage 3 才提取 `StartTrade` 复用，Stage 5 才启动 Lead WS，Stage 7 的 Lead 专用 Key 实盘 Gate 仍未完成。

## 5. Gate 2 结论

**Stage 2：后端离线实现、SQLite 模拟 Gate 与独立 MySQL 5.6 索引验证通过；真实业务数据库迁移及线上 Main 回归仍需确认。**

项目继续遵守：个人使用、单进程部署、严禁自动 commit/push、无真实交易、测试后不保留监听 3333 的服务进程。

## 6. MySQL 5.6 兼容性修复（2026-10-09）

**触发错误**：在本地 MySQL 5.6 执行 `./go_binance_futures sync db`，创建 `uq_fp_account_symbol_side` 时得到 `Error 1071 (42000): Specified key was too long; max key length is 767 bytes`。

**根因**：原 `FuturesPosition` 的 `symbol` 和 `side`、原 `FuturesOrder` 的 `order_id`、历史 `order` 的 `side/symbol` 均可能是早期 ORM 建立的 `VARCHAR(255)`。MySQL 5.6 的 `utf8mb4` InnoDB Antelope/无 large_prefix 环境最多支持 767 字节复合索引；Stage 2 之前直接对完整字段建立复合唯一索引，因此失败。

**修复**：

1. `command/account_scope_migration.go` 的 MySQL 复合普通索引及唯一索引现在统一对 Binance 身份字段应用前缀：`account_id(16)`、`symbol(32)`、`side(8)`、`position_side(8)`、`order_id(32)`、`owner(32)`、`status(32)`，最宽索引按 utf8mb4 估算也只有 320 字节。SQLite 仍使用**完整字段索引**。
2. 在创建任何索引前，对历史字段长度进行只读检查：如果旧数据超出预定前缀大小，**拒绝升级**并提示表/字段，不裁剪数据，也不以错误的前缀将不同的正常账户记录混为一谈。正常 Binance 交易对、订单 ID 及持仓方向应远小于这些界限。
3. 在模型中明确规定未来新建表的 `symbol(32)`、`side(8)`、`order_id(32)`，但**不自动缩短已有 MySQL 列**，从而避免旧数据截断和潜在锁表操作。
4. 现有 DDL 迁移保持**幂等**：失败时 Schema 版本不会升至 19，重新运行会跳过前次已成功创建的索引，仅补齐未完成的索引。不要手工清理已建索引。

**测试**：
- 新增 `command/account_scope_migration_test.go`，检查全部七种索引组合的 MySQL 5.6 前缀、767 字节预算及 SQLite 完整字段行为；原 v19 SQLite 历史迁移及唯一性测试继续通过。
- 另使用一次性、**独立**的 MySQL 5.6.51 Docker 容器（`innodb_large_prefix=OFF`、`Antelope`、`utf8mb4`、旧 `VARCHAR(255)` 字段）执行五个普通复合索引和两个唯一复合索引：**全部创建成功**。同一 BTCUSDT 持仓键和同一交易所订单 ID 分别保存于 main/lead，均成功。容器在测试完成后已自动删除；未访问业务 MySQL 5.6 数据库。
- `go test -race ./command ./service/futuresownership ./feature/api/binance ./service/outcomereview ./service/systemhealth ./controllers ./feature -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'`：通过。被跳过的原研究模板静态用例及 macOS 链接器警告见上一节。

**下一步**：你本地备份 MySQL 5.6 数据库后，使用更新后的本地代码重新编译 `go_binance_futures`，再运行 `./go_binance_futures sync db`。**我没有重新执行你本地业务库的迁移命令。** 若再次出现报错，需要把完整错误发来，避免手动删表或关闭唯一约束。

## 7. CODE_REVIEW_Stage2.md 审计跟进（2026-10-09）

**审计结论**：没有 P0/P1。F1 是真实数据库部署 Gate，不能通过离线代码修改替代；F2/F3/F5 是约定或安全性假设；F4 是实际需补齐的永久测试缺口；F6 与未跟踪策略研究文件有关，本阶段不修改其内容。

- **F4 — 已修复（永久回归）**：在 `command/db_update_test.go::TestSyncDatabaseInitializesAndIsIdempotent` 增加两段真实 `SyncDatabase(19)` 失败路径，断言未知账户会拒绝、重复仓位槽位会拒绝、`config.version` 留在 18、所有五张表行数保持不变且错误不生成唯一索引；修正测试数据后再次运行迁移成功，唯一索引恢复，再次同步幂等。
- **F4 — 已修复（WS 清理）**：新增 `feature/feature_userdata_stage2_test.go::TestStage2DeleteOldUserDataKeepsLeadMirror`，使用同一币种的 Main/Lead 持仓与 NEW/PARTIALLY_FILLED/FILLED 订单，调用真实 `deleteOldUserData()` 后确保仅 Main 持仓和 Main 活动订单删除，Main FILLED 及全部 Lead 镜像保持原样。修改 `feature/feature_test_strategy_guard_test.go` 的测试 SQLite 初始化，保证两组测试共享明确的模型注册且可独立运行。
- **F2 — 约束已明确**：MySQL 5.6 的 `utf8mb4` 前缀索引采用 `account_id(16), symbol(32), side(8), position_side(8), order_id(32)` 等限长键，`validateAccountIndexData` 在建索引前校验历史值超限并拒绝；以后如扩展 ID 长度，须同步审计模型、长度校验及唯一约束，不得将截断等价于真实唯一性。
- **F3 — 迁移约定已明确**：从 v19 起，允许在 `UpdateDatabase` 的版本门禁内使用 Go 函数执行数据回填、跨库条件 DDL 和失败前校验；所有升级仍只能通过二进制的 `sync db` 命令。成功前不更新 Schema 版本，重复迁移必须幂等，MySQL 已成功的 DDL 不会随着失败自动回滚。Go 迁移逻辑位于事务版号循环之前，避免误以为 DDL 受事务保护。
- **F5 — 明确记录兼容差异**：`feature/order.go::UpdateOrderStatus` 的 Main 读取失败不再删除无法确认的历史开仓记录（属于明确的安全改进）；其余 Main 账户原有查询与交易路径保持兼容。
- **F1 — 待用户侧上线验收**：未主动运行真实 MySQL 5.6/8 的 `sync db`，未进行真实 Binance 下单；需使用备份库核对行数、重复索引、Main 原有业务行为。
- **F6 — 维持原范围**：保留 `strategy_templates/research` 未跟踪文件；当前完整 `go test ./...` 对其中部分研究脚本存在重复 `main` 定义或静态扫描问题，采用受控模块测试并记录过滤条件，不据此修改正在研究的策略数据。

本节是对审计报告的实施回复，不修改原始 `CODE_REVIEW_Stage2.md`。

## 8. 运行时 MySQL 1213 死锁排查与修复（2026-10-09）

**用户日志**：`update futures symbols trade precision error: Error 1213 (40001): Deadlock found when trying to get lock; try restarting transaction`。

**变更归属检查**：日志位于 `feature/feature.go::UpdateSymbolsTradePrecision`，执行 `UPDATE symbols AS s JOIN (...) v ON s.symbol=v.symbol SET tickSize,stepSize,type`。此 SQL、每 12 小时精度刷新任务及前端编辑触发刷新均在 Stage 2 之前存在，`git show HEAD:feature/feature.go` 对照确认 Stage 2 没有修改该 SQL；Stage 2 的新索引和事务作用于 futures order/position 管理表，而非 `symbols`。因此**未发现 Stage 2 直接引入此死锁的代码证据**。

**最大嫌疑并发路径**：`feature/api/binance/index.go::flushLatestWsTickers` 每秒将行情通过 `INSERT INTO symbols ... ON DUPLICATE KEY UPDATE` 分批写入，同时间精度刷新以 300 币为一个批次执行 `UPDATE ... JOIN`。原 WS map 遍历顺序不稳定，两个批量写入可能以不同的 row-lock 顺序碰撞。这是**有代码证据支持的竞争风险**，但仅凭应用日志无法确认 MySQL 当次死锁环的确切另一条 SQL；必要时在复现后读取 `SHOW ENGINE INNODB STATUS\G` 的 `LATEST DETECTED DEADLOCK` 验证，不要假定 Stage 2 是根因。

**最小风险修复**：
1. 精度更新与行情 WS 刷新都按 `symbol` 升序排列后生成批次，减少逆序行锁竞争，仍保持相同字段值、更新频率与行情含义。
2. 新增 `service/mysqlretry`，仅匹配 Go MySQL 驱动的 1213 / 1205，最多四次，递增 25/50/100 ms 等待；只用于这两个**独立、幂等、自动提交的 SQL 语句**，不对订单提交或多 SQL 事务使用重试。超出重试次数仍保留原错误并退出当前刷新任务。
3. WS 刷新最终失败时仍重新入队未成功的币，但不会用失败批次的旧 tick 覆盖重试期间收到的新 tick。
4. 保留原有 MySQL 5.6/8 支持，不修改 Schema、不需要运行 `sync db`，不修改 `app.conf`。

**验证**：`service/mysqlretry/retry_test.go` 覆盖 1213/1205 恢复、非重试错误、次数上限与 context 取消；`go test -count=1 -race ./service/mysqlretry ./feature ./feature/api/binance ./service/futuresownership ./command ./controllers -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` 通过。**未在用户正在运行的 MySQL 上强制制造死锁**，是否完全消除其运行时 1213 需部署后观察。
