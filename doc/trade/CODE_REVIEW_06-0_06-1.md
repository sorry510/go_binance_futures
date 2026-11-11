# 06-0 / 06-1 Code Review（方案 06「现货量化交易·复刻合约简化版」）

审查日期：2026-11-11
审查对象：方案 06 的 **06-0 现状清点与范围冻结**、**06-1 独立 Schema 与总开关**
报告性质：**新增报告**，不覆盖任何历史 `CODE_REVIEW_01-*.md` / `CODE_REVIEW_02-*.md`
依据文档：`开发方案-06-现货量化交易-复刻合约简化版.md`、`开发方案-06-0-现状清点与范围冻结-实施记录.md`、`开发方案-06-1-独立Schema与总开关-实施记录.md`

---

## 零、审计基线、范围与边界

### 0.1 基线（本次实测）

| 项 | 实测值 |
| --- | --- |
| 当前分支 / HEAD | `feat/trade_spot` / `4af6d73`（2026-11-10 23:43:27 +0800） |
| master | `8f9a70b`，`[origin/master]` 同步 |
| `master..HEAD` | 2 个提交：`acf108f doc: create spot doc`、`4af6d73 Merge branch 'master' into feat/trade_spot` |
| `git diff --stat master...HEAD` | **仅 1 个文档，175 行新增**；Go 业务代码零差异 |
| HEAD / master 的 `DatabaseSchemaVersion` | 均为 **24** |
| 工作区 | **13 项**：9 个 `M` + 4 个 `??`（均为 06-1 未提交改动） |
| 工作区 `appversion/version.go` | **v25**（06-1 未提交修改） |
| 前端仓库 | `/Users/zhz/work/binance/go_binance_futrues_new_ui`；4 项未提交（`locales/*`、`src/api/order.ts`、`src/views/order/index.vue`），与 06-0 记录一致，**未被本阶段触碰** |
| 隔离副本 | `/tmp/phase_audit_06`，`git archive HEAD` + 同步 13 项；**412 个 Go 文件 SHA256 全等**，`sha_diff=0 missing=0` |

> 审计对象是 **HEAD 之上叠加的 13 项工作区改动**；06-1 代码**尚未提交**，不能按「已提交代码」审计。

### 0.2 阶段边界

- 06-0：只读清点 + 文档冻结，**不建表、不下单**。
- 06-1：**仅**两处开关列 + 四张独立表 + 配置接口字段；**不得**出现扫描器、自动下单、账户页、模板 CRUD、前端页面（分属 06-2～06-5）。
- 方案 §8：「每阶段单独修改、单独测试、单独 Code Review 后再进入下一阶段」。

### 0.3 审查边界

未修改任何业务代码、未 commit/push、未改 `conf/app.conf`、未碰 `strategy_templates/` 与 `static/`（仅只读比对）、未对真实库执行 `sync db`、未访问真实 Binance、未重启服务、未操作生产仓位。探针与构建仅在 `/tmp/phase_audit_06` 副本内，结束删除。

---

## 一、总体结论（结论先行）

| 阶段 | 判定 |
| --- | --- |
| **06-0** | ✅ **通过**：对照清单、master 基线、旧方案残留、Schema v24 均经本次独立复核属实 |
| **06-1** | ⚠️ **代码功能与离线 Gate 通过，但存在 1 个条件触发的 P1 设计风险**（06-F1），**不认可「06-1 完全验收」** |
| **06-1 → 06-2** | ⛔ **建议先闭环 06-F1 再进入 06-2**；按方案 §8 的阶段 Gate 纪律，该风险属阻塞项 |
| 实盘 / 生产 Gate | ⛔ **全部 BLOCKED**：未部署、未连接生产库、未下单、无前端接入 |

**核心结论**：

> 06-1 的物理数据基础、默认 OFF、幂等迁移、DECIMAL 精度、唯一约束、旧数据不变等**实现质量良好**，并已通过永久测试与本次探针双重实证；旧方案 02 的模式/Owner/审批状态机**已彻底清除**，三条简化准则在新增代码中得到落实，无生产下单接线。
> 但 **v25 版本号与已废弃的旧方案 02 复用**，而 `SyncDatabase` 只依据 `config.version` 判断是否迁移、没有任何 schema 指纹校验。本次探针实证：当库已标记 v25 而四张表缺失时，`sync db` 返回 **成功且零补表**、无任何报错，`main.go` 的版本门禁也因 `25 < 25` 不成立而不拦截。这会在分支切换/回滚/手工改版本时留下**静默残缺 schema**，并在 06-2 起表现为运行期表不存在。
> 本轮**未发现资金安全漏洞**（06-1 无任何下单路径），风险性质为**迁移正确性 / 可运维性**。

---

## 二、06-0 验收逐项核对（方案 §6）

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| Futures 页面/API/表/服务一对一映射清单 | ✅ | `06-0 实施记录` §2，11 行映射表 |
| 核对已有 Spot 行情/开关/下单边界 | ✅ | 同文档 §3，7 条边界；`spot_symbols.enable` 与 `config.spot_enable` 保持独立 |
| 核对分支相对 master 基线 | ✅ **本次独立复核** | `git diff --stat master...HEAD` = 1 文件 175 行；HEAD 与 master 的 `DatabaseSchemaVersion` 均为 24 |
| 核对旧方案 02 专属代码残留 | ✅ **本次独立复核** | 630 个应用 Go 文件扫描：`spotRealOrderReleaseApproved` / `canAttestLiveFills` / `spot_strategy_holdings|orders|fills` / `AccountMain` / `ValidateScope` / `RequestMode` / `AuthorizeLegacyOrder` / `active_slot` **全部 0 处** |
| 核对 Schema 与实验库版本 | ✅ 采信记录 | 06-0 记录代码 v24、`go_bn_test` 只读查询 v24；本次未连库复核，按边界声明 |
| 不建表 / 不下单 / 不改前端未提交文件 | ✅ | HEAD 无迁移代码；`main.go` 中 `spot_quant` / `SpotQuant` / `RunRealSpot` / `spot.Run` **均 0 次**；前端 4 项未提交原样保留 |

> `OwnerAuto=29` / `reconcile_required=11` / `contract_version=5` 在应用源码中确实存在，但全部位于 **Futures 既有域**（`feature/ownership.go`、`service/futuresownership/*`、agent 版本管理），不是现货 02 残留，也不是 06-1 改动，不计为本阶段缺陷。

---

## 三、06-1 验收逐项核对（方案 §6）

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| 1 个 `config.spot_quant_enable` 默认 0 | ✅ | `models/tableStruct.go` 新增 `int8` + `default(0)`；探针 `PROBE06_FRESH spot_quant_enable=0`；永久测试 `command/db_update_test.go:728` |
| 1 个 `spot_symbols.spot_quant_enabled` 默认 0，不复用旧 `enable` | ✅ | `models/spot_symbols.go`；探针 `enabled_symbols=0`；永久测试 `:731` 断言存量行 `enable=1` 保持不变且新列为 0 |
| 4 张独立表，与 Futures 物理隔离 | ✅ | `command/spot_quant_migration.go` 显式 DDL；探针 `tables=4`；永久测试 `:737` 逐表断言 |
| 「按既有 `/service/config` 接入」 | ✅ | `controllers/index.go:69` GET 增 `spotQuantEnable`、`:139` PUT 增校验；`EditServiceConfig` 经 `orm.Update` 落库，探针 `PROBE06_PERSIST spot_quant_enable=1` |
| 本地 `sync db` 升级可重复执行 | ✅ | 探针 `PROBE06_IDEMPOTENT tables=4`；永久测试 `:793` 重复 `SyncDatabase(25)` |
| Futures 表及现有 Spot 旧数据不变 | ✅（本地/实验库口径） | 永久测试 `:731`、`:797`；实施记录 §三 记录 `spot_symbols` 809 行、`strategy_templates` 59 行等迁移前后一致。**MySQL `go_bn_test` 行数为记录自述，本次未连库复核** |
| 金额/数量用 `DECIMAL(36,18)` 而非 float | ✅ | DDL 10 个财务字段；永久测试 `:745` 逐列断言 `DECIMAL(36,18)`；`models/spot_quant.go` DTO 全部用 `string` 且**未 `RegisterModel`**，避免 ORM 改成 VARCHAR |
| 唯一键 / 索引 | ✅ | `client_order_id` 唯一、`(symbol, exchange_order_id)` 唯一、`(symbol, exchange_order_id, trade_id)` 唯一 + 4 个普通索引；永久测试 `:754`；重复 ClientOrderID / 重复 Fill 均被拒 |
| 所有开关默认 OFF，不存在自动下单路径 | ✅ | 探针 + 静态：4 个 DTO 类型**仅在其定义文件出现，外部引用 0 处**；`main.go` 0 引用；`static/` 中 `spotQuant` 命中 0 |
| 不复活旧 02 模式/Owner/审计状态机 | ✅ | 06-1 新增/改动文件中 `account_id=0`、`owner=0`、`approval=0`、`audit=0`、`mode=0`（`spot_quant_migration.go`）；详见 §八 |

---

## 四、离线 Gate 实证（本次实跑）

| # | 命令 | 结果 | 证据 |
| --- | --- | --- | --- |
| 1 | `go build ./...`（副本） | ✅ EXIT=0 | `/tmp/06_build.log` 空 |
| 2 | `go build -o /tmp/06_main_bin .` | ✅ EXIT=0 | 主程序构建成功 |
| 3 | `go vet ./command ./controllers ./models` | ✅ EXIT=0 | `/tmp/06_vet.log` 空 |
| 4 | `go test -count=1 ./command ./controllers ./models`（**用户工作区只读**） | ✅ **3/3 PASS** | `command 0.863s`、`controllers 1.189s`、`models 0.948s` |
| 5 | 同 3 包 `-race`（用户工作区只读） | ✅ **3/3 PASS** | 仅 macOS `ld: malformed LC_DYSYMTAB` 非致命噪音 |
| 6 | `git diff --check`（用户工作区） | ✅ EXIT=0 | 无输出 |
| 7 | `go test -count=1 ./...`（副本，删除探针后） | ❌ EXIT=1 | **67 包 ok**；3 处红点全部为既有/环境归因，见下 |
| 8 | `go test -count=1 -run '^$' ./...`（用户工作区，全包编译） | ❌ EXIT=1 | 红点全部在 `strategy_templates/research/**` 未跟踪研究目录 |
| 9 | `dist` ↔ `static` SHA256 | ✅ **106/106 全等** | `only_dist=0 only_static=0` |

**全量红点归因（既有，非 06-1 引入）**：

```text
--- FAIL: TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals   # 副本缺未跟踪 strategy_templates/
--- FAIL: TestStage56SupervisorRecoveryStartsBeforeWSAndStopsAllWorkers      # 并行全量偶发时序抖动
panic: the target config file: /tmp/phase_audit_06/conf/app.conf don't exist  # tests 包，conf 未入 Git
FAIL    go_binance_futures/tests
```

`service/leadaccount` 该项单独复跑 **EXIT=0 PASS**（`/tmp/06_lead.log`），确认为并行抖动（与 01-ALL 的 ALL-F6 同类）。按项目纪律**未复制用户敏感配置、未绕过、不写「全仓测试通过」**。

**永久回归**：`TestSyncDatabaseInitializesAndIsIdempotent`（`command/db_update_test.go:17`，v25 断言集中在 `:712–804`）、`TestApplyServiceConfigPatchSpotQuant`（`controllers/index_service_config_test.go:68`，7 种输入）。探针删除后目标包复跑 `EXIT=0`（`command` / `models` 全绿）。

---

## 五、探针实证的隐式契约

探针仅写在副本内，以 `-run` 单独执行（Beego `orm.RegisterDataBase` 为进程级全局，与包内既有测试并行会冲突，此为工具约束而非业务缺陷），跑完已删除。

```text
PROBE06_FRESH        version=25 spot_quant_enable=0 enabled_symbols=0 tables=4
PROBE06_IDEMPOTENT   tables=4
PROBE06_PERSIST      spot_quant_enable=1
PROBE06_VERSION_REUSE sync_err=<nil> tables_after=0 version_after=25
PROBE06_INCOMPATIBLE err=incompatible existing spot_managed_positions: missing base_asset
```

| 契约 | 结论 |
| --- | --- |
| 全新升级 v25 建四表、两个开关默认 OFF | ✅ 已实证 |
| 重复 `SyncDatabase(25)` 与重复 `ensureV25SpotQuantTables` 幂等 | ✅ 已实证 |
| 总开关经 `orm.Update` 真正持久化（`GetSystemConfig` 读回 1） | ✅ 已实证，永久测试**未覆盖写入路径**，建议补 |
| 同名但缺列的表会被拒绝、不静默复用旧实现 | ✅ 已实证（正向不变量） |
| **版本已为 25 但四表缺失时 `sync db` 自愈** | ❌ **不成立**，见 06-F1 |

配置接口契约探针（`controllers` 包纯函数）：

```text
{"spot_quant_enable":1}                   -> spot_quant_enable=1 err=<nil>
{"spotQuantEnable":1}                     -> spot_quant_enable=0 err=<nil>   ← GET 原名回传被静默忽略
{"spot_quant_enable":2}                   -> err=spot_quant_enable must be 0 or 1
{"spot_quant_enable":null}                -> err=spot_quant_enable must be 0 or 1
{"spot_quant_enable":"1"}                 -> err=spot_quant_enable must be 0 or 1
{"spot_quant_enable":1,"future_enable":1} -> spot_quant_enable=1 future_enable=1 err=<nil>
{"spot_quant_enable":0,"spot_enable":0}   -> spot_enable=0 err=<nil>
```

非法值（2 / -1 / 字符串 / true / null）均被拒绝且保持原值，为正向不变量；后两行显示**同一次 PUT 可无差别修改其它配置字段**，见 06-F3。

---

## 六、Findings

| ID | 级别 | 问题 | 证据 | 影响 / 建议 |
| --- | --- | --- | --- | --- |
| **06-F1** | **P1（阻塞 06-2）** | **v25 版本号与旧方案 02 复用，且无 schema 存在性校验**。`feat/trade-upgrade` 分支同为 v25，但其迁移文件是 `command/spot_strategy_v25_migration.go`（旧 02 三表 `spot_strategy_holdings/orders/fills`）；06 的 v25 是四张 `spot_managed_*` / `spot_trade_fills` / `spot_strategy_templates`。`SyncDatabase` 仅在 `systemConfig.Version < targetVersion` 时调用 `UpdateDatabase`，而 `ensureV25SpotQuantTables` 位于 `UpdateDatabase` 内；`version == target` 走 `else` 分支「already up to date」。 | 探针 `PROBE06_VERSION_REUSE sync_err=<nil> tables_after=0 version_after=25`；`command/db_update.go:372`、`:451–465`；`git show feat/trade-upgrade:appversion/version.go` = 25 | 任何曾被旧 02 二进制升到 v25 的库（含实验库切换、回滚、手工改 `config.version`），切到 06 后 `sync db` **返回成功但零补表、无报错**；`main.go:240` 版本门禁 `25 < 25` 也不拦截。06-2 起运行期报表不存在且难以定位。**本次未发现实验库已处于该状态**，属条件触发风险。建议：① 把 `ensureV25SpotQuantTables` 提升到 `SyncDatabase` **每次调用**执行（与既有 `ensureV22SettlementIndexes` 同款结构自愈，且该函数已幂等）；② 增加 schema 指纹/表存在性校验，缺表时明确报错而非 "already up to date"；③ 可选改号 v26，但单改号不能根治跨分支错配 |
| **06-F2** | P2 | **GET / PUT 字段命名不一致**。`GET /service/config` 返回 `spotQuantEnable`（camelCase），`PUT` 仅接受 `spot_quant_enable`（`models.Config` json tag 为 snake_case）。 | 探针 `{"spotQuantEnable":1} -> spot_quant_enable=0 err=<nil>`；`controllers/index.go:69`、`:139` | 前端若把 GET 的 `data` 原样 PUT 回去，开关**静默不生效且返回 200**，与方案 §3.1「以保存响应确认开关状态」冲突。建议 GET 统一返回 `spot_quant_enable`，或在文档中明确 PUT 契约并在 06-5 前端用 snake_case 提交 |
| **06-F3** | P2 | `EditServiceConfig` 无字段白名单：`applyServiceConfigPatch` 对 payload 任意 Config 字段 `json.Unmarshal`，随后 `orm.NewOrm().Update(&systemConfig)` 整行写回。 | 探针 `{"spot_quant_enable":1,"future_enable":1}` 同时改合约总开关；`{"spot_quant_enable":0,"spot_enable":0}` 同时改旧现货开关；`controllers/index.go:156`、`:173` | 属既有设计，但 06-1 把「唯一全局安全开关」接入了这条无差别写入路径，且无操作审计。建议 06-5 前为该开关设独立端点或显式白名单，至少记录变更审计 |
| **06-F4** | P2 | `GetServiceConfig` / `EditServiceConfig` 沿用 `ctrl.Ctx.Resp(map...)`（项目 015-F1 / S1-F2：按 `Accept` 分发，map payload 在 XML/Proto 下 500 / panic）。 | `controllers/index.go:53`、`:165`、`:170`、`:177` | 非 06-1 新缺陷，但新增字段扩大了该 map 使用面。06-5 前端接入时建议改 `JSONResp`，并补控制器内二次 JWT 校验说明（`/service/config` 不在 `middlewares/auth.go` 的 `excludeRoutes` 白名单，全局鉴权已覆盖） |
| **06-F5** | P2 | `ensureV25SpotQuantFlagTypes` 仅在 MySQL 分支执行，且用 `information_schema` 逐列查询；`SyncDatabase` 的触发条件是 `targetVersion >= 25`，与 `UpdateDatabase` 的 `oldVersion < 25` 不同源。 | `command/spot_quant_migration.go:163–202`；`command/db_update.go:424` | 实施记录只在 `go_bn_test`（MySQL 5.6）验证；MySQL 8、列缺失（`QueryRow` 返回 `sql.ErrNoRows` 会使整个 `sync db` 失败）、权限受限均未测。生产升级前需在目标库只读预检列类型与权限 |
| **06-F6** | P3 | DDL 无 `side` / `status` 约束（`VARCHAR(4)` / `VARCHAR(32)` 无 CHECK）。 | `command/spot_quant_migration.go:49`、`:55`、`:72` | 方案 §3.3 / §7 要求「服务端必须拒绝 SHORT」属 06-2 模板层；06-3 写入前应在应用层或 DB 层加约束，避免错误方向被持久化 |
| **06-F7** | P3 | 06-1 的迁移断言全部追加在既有长测试 `TestSyncDatabaseInitializesAndIsIdempotent`（`:712–804`）内，无独立命名测试；`TestApplyServiceConfigPatchSpotQuant` 未覆盖 `EditServiceConfig → DB` 真实写入、GET/PUT 命名契约、以及「v25 已存在但缺表」场景。 | §四、§五 | 建议把本次三条探针结论固化为永久回归，尤其 06-F1 的修复必须有对应的命名测试 |
| **06-F8** | P3 | 全量 `go test ./...` / `-run '^$' ./...` 仍 EXIT=1。 | §四 | 既有归因（未跟踪研究目录、`conf/app.conf` 未入 Git、并行抖动），不得写「全仓测试通过」 |

---

## 七、正向不变量（本轮实证确认）

1. **旧方案 02 彻底清除**：630 个应用 Go 文件中，`spotRealOrderReleaseApproved` / `canAttestLiveFills` / `spot_strategy_*` / `AccountMain` / `ValidateScope` / `RequestMode` / `AuthorizeLegacyOrder` / `active_slot` 全为 0 处；符合方案 §7。
2. **无生产接线**：`SpotManagedPosition` / `SpotManagedOrder` / `SpotTradeFill` / `SpotStrategyTemplate` 四个 DTO **仅在 `models/spot_quant.go` 定义处出现，外部引用 0 处**；`main.go` 无 Spot Quant 循环；`static/` 无 `spotQuant` 字样。06-1 确实只建数据基础，符合阶段边界。
3. **财务精度保护**：DTO 未 `RegisterModel`，`RunSyncdb` 不会把 `DECIMAL(36,18)` 改成 VARCHAR；10 个财务字段全部精确十进制。
4. **拒绝不兼容同名表**：`ensureV25SpotQuantTables` 逐列存在性检查，缺列即报错（探针实证），避免悄悄复用旧实现的残表。
5. **DDL 事务语义符合项目既有约定**：v15 起项目明确「MySQL online DDL 在事务外、幂等、版本仅在成功后推进」，v25 遵循同一模式（`command/db_update.go:330–332`），不是新缺陷。
6. **前后端产物成套**：`dist` ↔ `static` 106/106 SHA256 全等；前端 4 项未提交未被触碰；06-1 未改动 `static/`，不存在半发布风险。
7. **鉴权**：`/service/config` 不在 `middlewares/auth.go:19` 的 `excludeRoutes` 白名单，受全局 JWT 保护。

---

## 八、三条简化准则的落地情况

方案 06 已把上一轮用户提出的三条准则写入 §1 / §7，本轮据此复核 06-1：

| 准则 | 06-1 现状 | 判定 |
| --- | --- | --- |
| 尽量复刻合约交易逻辑 | 表结构仿 `futures_managed_positions/orders` + `futures_trade_fills` 的一行一生命周期、`position_id` 关联、逐笔 Fill；配置沿用既有 `/service/config`（未另造配置中心）；未新建独立菜单/页面（属 06-5） | ✅ 数据模型层面已复刻，且未引入第二套交互规范 |
| 不要账户绑定逻辑 | 四张表 DDL **无** `account_id` / `owner` / `active_slot`；`spot_quant_migration.go` 中 `account_id=0 owner=0`；`models/spot_quant.go` 无账户字段 | ✅ 符合 |
| 不增加复杂限定逻辑 | 无三态模式、无审批常量、无来源认证状态机、无 `reconcile_required`、无 `contract_version`；全局仅一个 `spot_quant_enable` + 每币一个 `spot_quant_enabled` | ✅ 符合 |
| 真实下单必要防护不可删（方案 §1.5） | 唯一键去重、`DECIMAL`、`position_id` 关联已就位；余额/精度/MIN_NOTIONAL/ClientOrderID 恢复属 06-4，尚未实现 | ✅ 阶段边界正确，未提前造复杂度也未提前删防护 |

> 结论：06-1 在「不复刻旧 02 复杂度」这一点上执行得比上一轮方案 02 干净得多，未发现任何账户绑定或审批状态机残留。

---

## 九、Gate 结论矩阵与放行建议

| Gate | 结论 |
| --- | --- |
| 06-0 静态清点 / 范围冻结 | ✅ **PASS** |
| 06-1 代码功能 / 目标包测试 / race / vet / build / diff-check | ✅ **PASS** |
| 06-1 默认 OFF / 幂等 / DECIMAL / 旧数据不变 / 无生产接线 | ✅ **PASS（本地与实验库口径）** |
| 06-1 完全验收 | ⚠️ **不认可**（06-F1 未闭环） |
| **06-1 → 06-2** | ⛔ **阻塞**：先修 06-F1 并补永久回归，再复评 |
| 生产 v25 迁移 / MySQL 8 / 二进制+static 成套发布 / 真实 Binance 下单 / T0 | ⛔ **全部 BLOCKED** |

**建议的最小闭环顺序**：

1. 把 `ensureV25SpotQuantTables` 移到 `SyncDatabase` 每次调用路径（幂等已具备），并加「四表存在性」校验；补命名测试覆盖「`config.version=25` 且四表缺失 → 自愈或明确报错」。
2. 统一 GET / PUT 字段名（06-F2），并在 06-5 前端接入前明确写入契约。
3. 06-F3 / 06-F4 可在 06-5 前端联调时一并处理；06-F5 在生产升级前用目标库只读预检。
4. 复评另起 `CODE_REVIEW_06-0_06-1-rereview.md`，不覆盖本报告。

---

## 十、审查边界与未验证项

- 本次**未连接任何业务数据库**；`go_bn_test` 的 809 行 `spot_symbols`、59 行 `strategy_templates`、MySQL `TINYINT` 校正、18 位小数实读等均为 06-1 实施记录自述，本次以临时 SQLite 探针与永久测试替代，未独立复核 MySQL 结果。
- 未执行 `./go_binance_futures sync db` 于任何真实库，未访问 Binance，未重启服务，未操作生产仓位。
- 副本缺 `conf/app.conf`（未入 Git）与未跟踪 `strategy_templates/` 目录，按项目纪律未复制敏感配置或研究材料；相应红点已单独归因。
- 探针以 `-run` 单独执行以避免 Beego ORM 进程级注册冲突；删除探针后目标包复跑全绿。
- 所有量化数字均为本次对当前工作区/副本的实跑统计；工作区在审查期间经二次校验未发生变化，副本 412 个 Go 文件 SHA256 全等。
