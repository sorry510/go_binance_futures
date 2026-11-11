# CODE_REVIEW_06-2：现货币种、策略与独立模板

日期：2026-11-11  
分支：`feat/trade_spot`  
范围：方案 06 的 **06-2（现货币种、策略与独立模板）** 阶段代码与离线验收；含对 06-0/06-1 复评结论作为**前置 Gate** 的独立复核。  
结论摘要：**06-2 代码与离线 Gate 全绿，可作为研发基线；但前置 Gate 的「运行态 v25 Schema 校验」声明与代码不符，构成 06-2-F1（P1 准入阻塞）。不认可阶段整体验收完全通过，不认可方案 06 完成；生产 / 实盘 Gate 全部 BLOCKED。**

---

## 一、审计基线与范围

### 1.1 基线（本次实测，非引用文档）

| 项目 | 实测值 |
| --- | --- |
| 后端仓库 | `/Users/zhz/work/binance/go_binance_futures` |
| 分支 / HEAD | `feat/trade_spot` / `af07191`（2026-11-11 16:34:46 +0800 `Merge branch 'master' into feat/trade_spot`） |
| 本地 master | `7ed6c93` |
| Schema 真源 | `appversion/version.go:4` `DatabaseSchemaVersion = 25` |
| 后端工作区 | **8 项未提交改动**（`M` × 4、`??` × 4） |
| 前端仓库 | `/Users/zhz/work/binance/go_binance_futrues_new_ui`，HEAD `89f15bd`（`master`） |
| 前端工作区 | 8 项；其中 **6 项属 06-2**，另 2 项为用户既有订单页改动 |

后端 8 项：
`M doc/trade/开发方案-06-现货量化交易-复刻合约简化版.md`、`M doc/trade/开发方案-总体与实施顺序.md`、`M models/spot_quant.go`、`M routers/router.go`、`?? controllers/spot_quant.go`、`?? controllers/spot_quant_test.go`、`?? doc/trade/开发方案-06-2-现货币种策略与模板-实施记录.md`、`?? service/spotquant/`（`validation.go`、`repository.go`、`repository_test.go`）。

前端 8 项中，06-2 相关为 `locales/zh-CN.yaml`、`locales/en.yaml`、`src/composables/useStrategyTemplateOptions.ts`、`?? src/api/spotQuant.ts`、`?? src/router/modules/spotQuant.ts`、`?? src/views/spotQuant/`；**`src/api/order.ts` 与 `src/views/order/index.vue` 是用户此前已存在的订单页/API 改动，不属于 06-2，本次不纳入验收也不修改**（与 06-2 实施记录 §三 声明一致）。

### 1.2 隔离副本与等价性

- 副本：`/tmp/phase_audit_062`，由 `git archive HEAD` 加按 `git status --porcelain -z` 同步 8 项改动构成；探针只写在副本内。
- 等价校验（基于 git 索引构造集合，避免遍历 `strategy_templates/research/**` 超大未跟踪目录）：

```text
tracked=639 untracked_go=5 deleted_go=0 expected=645 copy_go=645
COPY_EQUIV sha_diff=0 missing=0 extra=0
```

`expected` = 639 tracked + 5 个未跟踪 Go 文件 + 1 个审计探针 `service/spotquant/zz_062_probe_test.go`；零内容差异、零缺失、零多余。

### 1.3 审计边界

未修改任何业务代码；未读取 / 复制 `conf/app.conf`；未连接真实 MySQL 或 Binance；未执行生产 `sync db`；未操作 `strategy_templates/` 与用户其他未提交文件；未 commit / push；未重启服务、未替换后端二进制、未同步 `static/`。所有测试进程显式清空 `SPOT_STRATEGY_LIVE_TRADING_APPROVAL`。本会话新增的唯一文件是本报告。

---

## 二、前置 Gate 独立复核（06-0 / 06-1 复评）

`CODE_REVIEW_06-0_06-1.md` 判 06-F1（P1：v25 版本号与已废弃旧方案 02 复用，且 `SyncDatabase` 在 `version == target` 时静默跳过）阻塞 06-2。`CODE_REVIEW_06-0_06-1-rereview.md` 声明已闭环并允许 06-2 本地开发。本次不采信声明，逐项复核：

| 复评声明 | 独立实证 | 判定 |
| --- | --- | --- |
| `SyncDatabase` 对已为 v25 的库每次只读校验 | `command/db_update.go:456`、`:469` 确有 `VerifyV25SpotQuantSchema` 调用；缺表返回 `database v25 Spot Quant schema mismatch` | ✅ 属实 |
| 永久回归覆盖缺表 fail-closed | `command/db_update_test.go:855` 子测试 `V25VersionReuseMissingSpotLedgerFailsClosed` 实测 PASS | ✅ 属实 |
| 06-F2/F3/F4 配置接口整改 | `controllers/index.go:24` `respondServiceConfig → JSONResp`；`:76` GET 返回 `spot_quant_enable`；`:147` camelCase `spotQuantEnable` 明确报错；`:153` 拒绝混合 PUT；`:186` `o.Update(current, "SpotQuantEnable")` 单列更新；`index_spot_quant_review_test.go` 两项永久测试存在且目标包全绿 | ✅ 属实 |
| **「`main.go` 的 `initializeRuntimeDatabase` 在版本门禁之后使用只读 `VerifyV25SpotQuantSchema`；旧方案 02 复用 v25 但缺新表的数据库无法正常启动新服务」**（rereview 第 12 行） | **与代码不符，见下** | ❌ **不成立** |

全仓 Go 文件中 `VerifyV25SpotQuantSchema` 的引用只有 4 处：

```text
command/spot_quant_migration.go:207  定义
command/db_update.go:456 / :469      SyncDatabase / UpdateDatabase 路径调用
command/db_update_test.go:855        永久测试
```

`main.go` 的启动链路：

```text
main.go:357  initializeRuntimeDatabase()
main.go:235  func initializeRuntimeDatabase() {
main.go:240      if systemConfig.Version < dbVersion { panic(...) }
main.go:243      SystemConfig = systemConfig
                 // 之后只有 EnsureDefaultSkillConfigs / LLM / Backfill / ownership reconcile
             }
main.go:279  command.SyncDatabase(dbVersion)   // 仅 runCommand 的 CLI `sync db` 分支
main.go:331  web.InsertFilter("*", web.BeforeRouter, middlewares.JwtMiddleware)
```

即：**正常启动路径只有「版本号 < 25」这一条门禁，没有任何 v25 表结构指纹校验；`VerifyV25SpotQuantSchema` 只在 CLI `sync db` 与测试中运行。** `command/spot_quant_migration.go:204-206` 的注释同样写着 "It is also called at application startup"，与实现不一致。详见 §七 06-2-F1。

---

## 三、06-2 验收矩阵

依据 `开发方案-06-现货量化交易-复刻合约简化版.md` §6 的 06-2 验收行（「现货币种模糊搜索、分页/报价类型、保存、启停、规则校验与模板导入导出测试通过；不需要真实交易」）、§7「必须明确不做的功能」，以及 `开发方案-06-2-现货币种策略与模板-实施记录.md` §一 / §四。

| # | 验收项 | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | 现货币种模糊搜索、分页、报价资产类型 | ✅ PASS | `ListSymbols` / `SpotQuoteAsset`；探针 `SEARCH_GUARD` 拒绝通配符与注入；永久测试 `TestStage062SpotSymbolsSeparateLegacyAndFutures` |
| 2 | 保存与字段隔离（不改写旧 `enable` / 行情 / 精度 / Futures 字段） | ✅ PASS | 探针 `WHITELIST` 7 类非法字段全拒 + `ROW_UNCHANGED` 原行未变 |
| 3 | 启停：币级 + 全局批量，且未配置币种不得被激活 | ✅ PASS | 探针 `ACTIVATE_NO_RULE` / `ACTIVATE_OK` / `GLOBAL_ENABLE activated=1 unconfigured_quant=0 flag2_err` |
| 4 | 规则校验：只 `long` / `close_long`，`short` / `close_short` 即使禁用也拒绝 | ✅ PASS | 探针 `DIRECTION` 两条；永久测试 `TestStage062SpotRuleBadDirectionsRejectedEvenDisabled` |
| 5 | 模板 CRUD / 导入 / 导出，且只读写 `spot_strategy_templates` | ✅ PASS | 探针 `TEMPLATE` + `FUTURES_ISOLATION rows=1 name="legacy-futures-must-stay"`；前端导出仅 `name/technology/strategy` |
| 6 | 规则试算明确不是实盘信号 | ✅ PASS（有 F4 提示） | `mode=synthetic` / `syntax_only`；永久测试 `TestStage062SpotRuleSyntheticPreviewNeverTrading`、`...UnknownIndicatorSyntaxOnly` |
| 7 | 独立 API，不复用 Futures `/features`、`/strategy-templates`、旧 `/spots` | ✅ PASS | 前端 `src/api/spotQuant.ts` 扫描：`/features=0 /strategy-templates=0 /spots=0 futures=0`，`spot-quant=1` |
| 8 | 无真实交易、无自动下单接线 | ✅ PASS | `main.go` 全文件无 `spotquant` / `spot_quant` 引用；`routers/router.go:127` 注释 `No order routes are registered`；探针全程无网络 |
| 9 | 离线测试命令（实施记录 §四 7 项） | ✅ PASS | 见 §五 |
| 10 | 前置 Gate（06-1 复评全部声明闭环） | ⚠️ **部分不成立** | §二；06-2-F1 |
| 11 | 正式顶级菜单 / 四页联动 / `dist→static` 成套交付 | N/A（属 06-5 / 06-6） | 路由 `showLink:false`；见 06-2-F3 |

---

## 四、逐文件静态审查要点

**后端**

- `service/spotquant/validation.go`：`ValidateBudget` 用精确十进制字符串校验，不经 `float64`；`ValidateTechnology` 校验指标 JSON 结构；`ParseRules` 校验 JSON 数组、`long`/`close_long` 方向与 Expr 语法；`ValidateActiveStrategy` 要求至少一条 `enable=true` 且 `type=long` 的入场规则；`SpotQuoteAsset` 识别 `FDUSD/USDT/USDC`。
- `service/spotquant/repository.go`：`ApplySymbolPatch` 显式字段白名单（`spot_quant_enabled`、`pin`、`usdt`/`profit`/`loss`、`technology`、`strategy`），未知字段直接报错；已启用币种保存时会重新校验 `ValidateActiveStrategy` + `ValidateTechnology`；`BatchSymbolPatch` 用事务且显式 `ids`；`ListTemplates` 的 `LIMIT/OFFSET` 由 `strconv.Itoa` 拼接（参数为已校验 int），WHERE 全部参数绑定；`ImportTemplate` 拒绝身份 metadata。
- `controllers/spot_quant.go`：`respond()` 统一 `JSONResp`（沿用 06-F4 整改，不受 `Accept` 影响）；`SpotRuleTest` 用本地 `spot_symbols.Close` 或用户输入价构造 env，明确 `mode=synthetic` / `syntax_only`。
- `models/spot_quant.go`：`SpotStrategyTemplate` 补 `orm:column` 映射以支持 `QueryRows`，**仍不注册 `orm.RegisterModel`**，避免 `RunSyncdb` 把 `DECIMAL(36,18)` 改成 VARCHAR。
- `routers/router.go:128-136`：9 条独立 `/spot-quant/*` 路由；不在 `middlewares/auth.go::excludeRoutes` 白名单内，受 `main.go:331` 全局 `JwtMiddleware` 保护。

**前端**

- `src/api/spotQuant.ts`：独立契约，不复用 Futures 路径。
- `src/router/modules/spotQuant.ts`：临时直达路由，`showLink:false`，正式顶级菜单仍属 06-5。
- `src/views/spotQuant/strategyTemplate.vue`：导出 JSON 只含 `name/technology/strategy`，与后端导入白名单兼容，不携带 `id/create_time/update_time`。
- `locales/*.yaml` 新增 `spotQuant062` 文案，其中 `configurationOnly` 明确「06-2 仅管理现货策略配置，尚未连接真实下单；启用币种不会立即下单」。
- `src/composables/useStrategyTemplateOptions.ts`：允许注入模板查询函数，旧 Futures 调用方式不变。

**产物边界**：`dist` ↔ `static` 实测 `dist=106 static=106 common=106 identical=106 only_dist=0 only_static=0`；`static_has_spotQuant062=False`。这与实施记录「Vite 构建隔离到 `/tmp/spot_quant_06_2_build`，未同步 `static/`」一致，但必须按 06-2-F3 明确交付口径。

---

## 五、离线 Gate 实测

全部在隔离副本 / 只读工作区执行，`go test` 均带 `-count=1`。

| 验证 | 结果 |
| --- | --- |
| `go test -count=1 ./command ./controllers ./models ./service/spotquant` | ✅ 4/4 PASS |
| `go test -count=1 -race` 同 4 包 | ✅ 4/4 PASS（`ld: malformed LC_DYSYMTAB` 为 macOS 非致命噪音） |
| `go test -count=1 -run '^TestStage062' -v ./controllers ./service/spotquant` | ✅ EXIT=0，**7 个顶层测试 / 10 个 PASS 断言**（4 controllers + 3 service，含 3 个子测试） |
| `go vet ./command ./controllers ./models ./service/spotquant` | ✅ EXIT=0 |
| `go build -o /tmp/062_main_bin2 .` | ✅ EXIT=0（未覆盖项目内二进制） |
| `go build ./command ./controllers ./models ./service/spotquant ./routers` | ✅ EXIT=0 |
| `git diff --check`（工作区） | ✅ EXIT=0 |
| 前端 `pnpm exec vue-tsc --noEmit --skipLibCheck` | ✅ EXIT=0；隔离 Vite build 亦 PASS（实施记录） |
| `dist` ↔ `static` SHA256 全量比对 | ✅ 106/106 全等；`static` 不含 06-2 页面 |
| 临时探针 `TestProbe062ImplicitContracts` | ✅ PASS（见 §六） |
| 副本 `go test -count=1 ./...` | ❌ EXIT=1：**69 个包 ok**；`controllers` 的 `TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals` 因副本缺未跟踪 `strategy_templates/` 失败；`tests` 包因缺 `conf/app.conf` panic（日志 `panic: the target config file: /tmp/phase_audit_062/conf/app.conf don't exist`）。均为既有环境归因，**不写「全仓测试通过」** |

---

## 六、探针实证（副本内临时探针，结束随副本删除）

探针文件：`/tmp/phase_audit_062/service/spotquant/zz_062_probe_test.go`（仅存在于副本）。

```text
PROBE062_CREATE tick=0.1 step=0.1 usdt=10 enable=0 quant=0
PROBE062_WHITELIST {"enable":1}  err=field "enable" is not editable by Spot Quant
PROBE062_WHITELIST {"tickSize":"0.001"} / {"close":"999"} / {"id":999} /
                   {"symbol":"ETHUSDT"} / {"spot_new_enable":1} / {"future_enable":1}  全部拒绝
PROBE062_ROW_UNCHANGED enable=0 tick=0.1 close=0 symbol=BTCUSDT quant=0
PROBE062_ACTIVATE_NO_RULE err=enable at least one valid LONG entry rule before activating Spot Quant
PROBE062_DIRECTION err=strategy[0] type "short" is not allowed for Spot (LONG/CLOSE_LONG only)
PROBE062_DIRECTION err=strategy[0] type "close_short" is not allowed for Spot (LONG/CLOSE_LONG only)
PROBE062_ACTIVATE_OK quant=1 strategy=[{"name":"buy entry","type":"long","code":"NowPrice > 0","enable":true}]
PROBE062_GLOBAL_ENABLE activated=1 unconfigured_quant=0 flag2_err=enabled must be 0 or 1
PROBE062_FUTURES_ISOLATION rows=1 name="legacy-futures-must-stay"
PROBE062_SEARCH_GUARD symbol_wildcard=invalid symbol search quote_inject=unsupported quote asset template_wildcard=invalid template name search
PROBE062_BATCH_MISSING_ID updated=0 err=symbol 99999999: <QuerySeter> no row found
PROBE062_IMPORT_METADATA err=import metadata cannot overwrite template identity
PROBE062_FUTURES_VARS ROI_err=<nil> FundingRate_err=<nil>
PROBE062_TEMPLATE id=1 import_action=updated import_err=<nil> same_name_rows=1
```

**正向不变量**：新建币种默认 OFF 且不改旧 `enable`；7 类越权字段拒绝且行零变更；激活必须有 enabled LONG；SHORT 家族即使 `enable=false` 也拒绝；全局批量启用只激活已配置币种且非法 flag 拒绝；Futures 模板表零污染；搜索通配符 / 注入被拒；批量不存在 ID 报错回滚；导入拒绝身份 metadata；模板 CRUD 与同名导入无重复行。

**边界观察**：`ROI` / `FundingRate` 等 Futures-only 变量在配置层不报错（见 06-2-F4）。

---

## 七、Findings

| ID | 级别 | 问题 | 实证与影响 | 建议 |
| --- | --- | --- | --- | --- |
| **06-2-F1** | **P1（准入阻塞）** | **运行态缺 v25 Schema 校验，前置 Gate 声明不实** | rereview 第 12 行与 `command/spot_quant_migration.go:204-206` 注释均声称启动态调用 `VerifyV25SpotQuantSchema`；实际全仓引用只在 `command/` 与测试，`main.go:235-261` 只做 `Version < dbVersion` 门禁，`SyncDatabase` 只在 CLI `sync db`（`main.go:279`）执行。若数据库被旧方案 02 标为 v25 但缺 06 的四张表，直接启动新二进制**不会拦截**；06-3 一旦开始写 Spot 表即运行期失败或账本不完整。**当前 06-2 无任何写路径，因此不构成当前资金安全风险**，但 06-1 复评「运行态已修复」的结论不成立，作为 06-2 准入 Gate 未闭环 | 在 `initializeRuntimeDatabase()` 版本门禁之后加只读 `command.VerifyV25SpotQuantSchema(orm.NewOrm())` 并失败 panic，补启动态永久测试；若选择不实现，则必须更正 rereview 与代码注释，并把「发布前必须执行 `sync db` 校验」写进发布 SOP。**推荐前者** |
| 06-2-F2 | P2 | 总方案文档基线口径过期 | `开发方案-06-现货量化交易-复刻合约简化版.md:5` 仍写 HEAD `4af6d73` / master `8f9a70b` / Schema v24；实测 `af07191` / `7ed6c93` / v25。状态行已改成「06-2 已完成」，但基线仍是 06-0 时点快照，后续阶段按此对照会误判 | 把该行标注为「06-0 时点基线快照」，另补一行当前基线（HEAD / master / v25 / 日期）；06-0 实施记录内的 v24 属历史记录，可保留 |
| 06-2-F3 | P2 | 交付边界：页面与后端 API 均未成套发布 | `dist`↔`static` 106/106 全等但 `static` 不含 `spotQuant062`，且后端二进制未替换 → 现有部署上隐藏路由 / 9 条 `/spot-quant/*` API 均不可用。与 01-ALL F1（static 已换、二进制未换）方向相反，但同属「前后端不成套」风险。**符合**实施记录「正式菜单与成套交付属 06-5/06-6」，不是 06-2 违约 | 报告与交付说明中明确「06-2 页面不是可用生产 UI，仅研发基线」；06-6 统一执行 `dist→static` + 后端二进制 + schema 的成套发布与回滚 |
| 06-2-F4 | P3 | Spot 规则仍接受 Futures-only 变量 | `ParseRules` 只校验 JSON / 方向 / Expr 语法；`controllers/spot_quant.go:266-271` 的 env 注入 `ROI` 且用 `expr.AllowUndefinedVariables()`；探针 `ROI_err=<nil> FundingRate_err=<nil>`。06-2 明确不做指标 runtime，`mode=syntax_only` 已把不可计算表达式降级，故当前不产生可交易信号 | 06-4 执行前建立 Spot 专属变量白名单，从 env 移除 `ROI` / 资金费 / 杠杆类变量，并对未知变量给出显式拒绝或 `syntax_only` 标记 |
| 06-2-F5 | P3 | 同名模板导入静默覆盖 | 探针 `import_action=updated same_name_rows=1`；符合实施记录「同名更新，否则新建」，但无版本、无确认提示，可能覆盖用户草稿 | 不建议新增审计中心（与方案准则「不加复杂限定」冲突）；仅建议在前端导入前提示「将覆盖同名模板」 |
| 06-2-F6 | P3 | 双轨开关：币级批量启用不检查全局总开关 | `/symbols/enable/:flag` 可独立把已配置币置 `spot_quant_enabled=1`，不读 `config.spot_quant_enable`；探针 `activated=1`。06-2 无触发器所以无资金风险 | 06-4 接线前必须明确「全局总开关是唯一权威门控」，定义关闭总开关时币级 enabled 的语义（保留配置但不再发新自动单），并补永久测试 |
| 06-2-F7 | P3 | 新控制器未沿用 `authorized()` 二次校验约定 | `SpotQuantController` 无 `authorized()`；而 01-5 整改后 `OrderController` / `LeadController` 均有。**实证**：`main.go:331` `web.InsertFilter("*", web.BeforeRouter, JwtMiddleware)` 且 `/spot-quant/*` 不在 `excludeRoutes`，当前不存在未授权访问 | 属纵深防御一致性问题，不判安全漏洞；建议 06-5 正式菜单发布前统一，避免未来路由误加白名单 |

**未发现**：资金安全 P1（无下单路径、无生产接线）、SQL 注入可绕过、旧方案 02 的模式 / Owner / 审批状态机复活、Futures 表被写、SHORT 方向被放行、自动下单接线。

---

## 八、Gate 结论

| Gate | 判定 |
| --- | --- |
| 06-2 本地代码与离线验收 | ✅ **PASS**（研发基线）：验收矩阵 1~9 全绿、645 文件副本等价、目标包 / race / 专项 / vet / build / diff-check / 前端类型检查全绿、探针正向不变量全绿 |
| 06-2 阶段整体验收 | ⚠️ **有条件**：06-2-F1（P1）使 06-1 复评「运行态已修复」不成立，前置 Gate 未实证闭环；**不认可「06-2 完整验收通过」** |
| 06-2 → 06-3 | ⛔ **建议先闭环 06-2-F1 再进入 06-3**；允许 06-3 纯本地代码 / SQLite 开发并行，但不得接入真实下单、不得对生产库执行升级 |
| 方案 06 完成 / 06-5 菜单对外 / 06-6 成套交付 | ⛔ **不认可、未放行** |
| 生产发布、真实 Binance Spot、真实账户下单、T0、MySQL 8 生产 DDL / 权限、`dist→static` + 二进制成套发布 | ⛔ **全部 BLOCKED**，需单独授权与预检 |

一句话：**06-2 本身写得很干净——独立 API、字段白名单、SHORT 硬拒、模板物理隔离、无自动下单接线；问题不在这阶段的代码，而在上一阶段复评里那条「运行态会拒绝启动」的声明没有落到代码上。** 这条不补，06-3 一旦开始写 Spot 账本表就会踩到旧方案 02 复用 v25 的坑。

---

## 九、遗留与操作边界

**遗留项（按优先级）**：① 06-2-F1 运行态校验；② 06-4 前统一总开关门控语义（F6）与 Spot 变量白名单（F4）；③ 06-5/06-6 成套发布与回滚（F3）；④ 文档基线口径（F2）；⑤ 模板导入提示（F5）、控制器鉴权一致性（F7）。MySQL 8 生产库 DDL / 权限预检仍属 06-F5 未闭环范围。

**本次未执行**：不改 `conf/app.conf`、不执行生产 `sync db`、不访问真实 Binance、不操作真实仓位、不重启服务、不替换后端二进制、不同步 `static/`、不 commit / push、不碰 `strategy_templates/` 与用户其他未提交文件。所有探针仅在 `/tmp/phase_audit_062` 内，收尾随副本删除。

**报告不覆盖历史**：`CODE_REVIEW_06-0_06-1.md` 与 `CODE_REVIEW_06-0_06-1-rereview.md` 原文保留；若后续整改 06-2-F1，复评须另起 `CODE_REVIEW_06-2-rereview.md`。
