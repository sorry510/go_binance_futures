# Code Review：方案 02 · S4「现货策略前端集成、只读 API 与交付验收」

- 审计日期：2026-11-10
- 审计对象：**工作区未提交代码**。HEAD 为 `95b5a77 feat: spot s1 s2`（本地领先 `origin/feat/trade-upgrade` 2 个提交，`ad79ce7`）；工作区共 **180 项**改动 = S3 + **S3-R1 整改** + **S4 增量** + 前端 `static/` 产物整体重建（106 个旧 hash 文件删除、约 80 个新文件新增、`static/index.html` 修改）。Schema 仍 **v25**（`appversion.DatabaseSchemaVersion`）
- 方案依据：`doc/trade/开发方案-02-S4-前端集成与完整验收.md`（工作区版，§7 四项交付勾选 3 项、§8/§9 边界变更、§10 实施与 E2E 签收记录）；前置 `doc/trade/CODE_REVIEW_02-S3.md` 与 **`CODE_REVIEW_02-S3-rereview.md`（S3-R1 整改复核）**；上游 S0/S1/S2 方案
- 副本：`/tmp/phase_audit_s4`（`git archive HEAD` + 同步 180 项：110 文件复制、70 文件删除，排除未跟踪研究目录）。对 `service/spotstrategy`、`spot/api/binance`、`spot`、`feature/strategy/line`、`controllers`、`routers`、`main.go`、`appversion/version.go` 八组做文件清单 + SHA256 校验，**全部 0 mismatch**。探针写在副本内，结束随副本删除
- 前端仓库：`/Users/zhz/work/binance/go_binance_futrues_new_ui`（5 项未提交：2 个新文件 + 3 个修改），全程 `GIT_OPTIONAL_LOCKS=0`
- 报告性质：**新文件**，不覆盖 `CODE_REVIEW_02-S1*.md` / `02-S2*.md` / `02-S3*.md` / `01-*.md`

> **本轮基线要点**：S3-R1 已把真实 Spot 写权限做**编译期硬封闭**（`spotRealOrderReleaseApproved = false`、`canAttestLiveFills` 恒 false），S4 在此基线上只增加**只读面**：4 条 GET 路由、1 页 Vue、i18n/README 与静态产物。本轮未发现 S4 引入新的真实资金写路径；审查重点转为「只读边界强度、验收口径真实性、前后端成套发布风险」。

---

## 一、结论先行

| 项目 | 结论 |
| --- | --- |
| S3-R1 硬封闭独立复核 | ✅ **成立**。env 精确串 + DB `mode=strategy` + 合法 `submitting` 订单 + `strategy_trade_enabled=1` 五条件齐备，`LiveOrdersApproved()` 仍 false、`AuthorizeLiveWrite` 仍 `ErrLiveWriteDisabled`；12 个 S3-R1 永久测试独立复跑全 PASS |
| S4 只读边界 | ✅ **成立**。4 条新路由全为 `get:`，无 POST/DELETE；前端 5 个 GET + 1 个 S1 配置 PUT，0 POST/DELETE；探针实证 4 个读接口调用前后 5 张表行数不变，签名 URL 不泄漏 |
| 离线 Gate | ✅ **目标包 5/5**、race 5/5、vet 6 包、build 2 项、controllers S4 专项、S2-R1 4/4、S3-R1 12/12、`git diff --check`、前端 `tsc`/`vue-tsc --noEmit` 全 PASS |
| `go test -count=1 ./...` 全量 | ❌ EXIT=1（controllers 5 项既有策略模板红点 + `tests` 包副本缺 `conf/app.conf` panic），69 包 ok。**不得写「全仓测试通过」** |
| 前端产物一致性 | ✅ 独立实证 dist ↔ `static/` **107/107 文件相对路径与 SHA256 全等**，含 `index.html` 与 `spotStrategy-CeaCvaRO.js`；中英 i18n 各 64 键完全对称 |
| §7 四项交付 | **3 项勾选、第 1 项未勾**（「S0～S3 全部 Gate 留有测试证据；E2E-01～11 有可复现实测输出」）——方案自身未签收 |
| E2E-01～11 | ⚠️ **离线索赔过宽**：浏览器 E2E 全部未运行；且探针证明**真实运行链路上 `rest_verified` 恒 0、Holding 永不结算，S4 持仓周期页零数据**（见 S4-F1） |
| 生产 Gate / 实盘 / 模式切换 | ❌ **全部 BLOCKED，未授权** |
| **S4 → 方案 03 C0** | ⚠️ **有条件**：S4 本地只读代码、前端产物可作为**继续研发与人工 UI 验收基线**（离线 PASS）；但**不认可「S4 完整验收通过」，不认可方案 02 完成、不放行依赖生产证据的 03 C0**。S4-F1/F2/F3 应先闭环并复评 |

**Findings 级别分布**：本轮**无新增 P1**。S4 未引入新的真实资金写风险，S3-R1 的封闭语义未被回退。最高为 **P2**：验收口径（F1）、成套发布与版本口径（F2）、状态真源（F3）、必需字段缺失（F4）；其余为 P3 工程质量（F5/F6）。

---

## 二、审计基线与改动范围

| 范围 | S4 实际改动 | 审查判断 |
| --- | --- | --- |
| `service/spotstrategy/s4_read.go`（385 行，新） | `ListSettings/ListHoldings/ListOrders/ListOrderFills`，固定 `main/auto_strategy`、参数化 SQL、分页上限 50、symbol/status 白名单、`holding_id`/本地订单 ID 归属校验、安全诊断码 | 只读，无写语句；但见 F5/F6 |
| `controllers/spot_strategy_s4.go`（163 行，新） | 4 个 handler，`authorized()` JWT 二次校验 → `ValidateS4Scope` → 分页/ID 校验 → `orm.GetDB` → 服务层；错误统一映射 400/404/脱敏 503 | 门禁在 DB 之前，错误不回显原始 SQL/签名 |
| `routers/router.go`（+5） | 新增 `/spots/strategy/configs`、`/holdings`、`/orders`、`/fills`，**全部 `get:`**；全局 `web.InsertFilter("*", web.BeforeRouter, JwtMiddleware)`（`main.go:331`）覆盖，白名单不含这些路径 | ✅ 无 POST/DELETE/解冻/清仓路由 |
| `service/spotstrategy/config.go`、`controllers/spot_strategy.go` | `Setting` 增 `LastPrice/LastPriceAt`；`mock_only` → `live_disabled`；`GetMode/PutMode` 增 `live_approved:false`、`reason` | `UpdateSetting` 白名单未变（仅 `strategy_trade_enabled/budget_quote/strategy/technology`），**不改全局 mode、不碰旧 `enable`**；但见 F3 |
| `s3_cost.go`、`s3_review_test.go`（231 行）、`s3_live_review_test.go`（180 行） | S3-R1 整改：成本从 linked verified Fill 重建、`HoldingReport` 跳过未核实 Fill、真实 SDK payload 永久测试 | 属 S3-R1 范围，本轮独立复跑验证 |
| `main.go` | 与 S3 相同的 7 行 worker 接线，**S4 未新增生产接线**；runner 有 `recover()` panic 隔离 | ✅ 未放大 S3 风险 |
| 前端（另一仓库） | `src/api/spotStrategy.ts`（新）、`src/views/notice/spotStrategy.vue`（新，486 行）、`futures.ts` 增 1 条路由、中英 locales 各 +80 行 | 单页 3 tab、两层懒加载、无下单/撤单/切模式/解冻按钮 |
| `static/`、`README*.md`、方案文档 | `rsync --delete` 重建静态产物（107 文件）；README 增 S4 段落 + 实盘 BLOCKED 说明 | 见 F2 |

---

## 三、方案 §7 交付项与 §10 验收声明逐项核对

| # | 方案要求 | 本轮证据 | 结论 |
| --- | --- | --- | --- |
| §7-1 | S0～S3 全部 Gate 留有测试证据；E2E-01～11 有可复现实测输出 | 方案自身**未勾选**；§10.4 的 11 项中浏览器人工演示、真实成交、真实故障演练全部「未运行」；探针 `PROBE_S4_NOHOLDING` 证明业务数据闭环在真实链路不可复现 | ⚠️ **未完成**（F1） |
| §7-2 | Go/Vue、本地 build/static、多语言与 README 同步；Schema/迁移版本准确 | dist↔static 107/107 全等；i18n 中英 64/64；Go/Vue 构建与类型检查 PASS；**但 README 主版本声明仍写 v24，代码为 v25** | ⚠️ **部分**（F2） |
| §7-3 | 未访问/更改生产库、未提交真实委托、未 commit/push、仍显示实盘未授权 | 探针 + 静态扫描确认：零真实 HTTP 写、4 条新路由只读、硬封闭成立、工作区未提交 | ✅ **PASS** |
| §7-4 | 交付报告明确「通过/失败/跳过/未授权」四类 | §10.4 有「尚缺项」「明确跳过」列，诚实度合格；但 §10.3 未列 `go test ./...` 全量红点 §10.4 的「PASS（隔离业务）」易被误读为端到端通过 | ⚠️ **部分**（F1/F6） |
| §10.3 Go 目标包 / race / vet / build / diff-check | 独立复跑 | ✅ **全部 PASS** |
| §10.3 `pnpm typecheck` | 独立复跑 `tsc --noEmit`、`vue-tsc --noEmit --skipLibCheck` | ✅ **EXIT=0 / EXIT=0** |
| §10.3 `pnpm build` | **本轮未重跑**（脚本 `rimraf dist && vite build` 会删除并重建用户当前 dist）。改为只读核验：dist 含 `spotStrategy-CeaCvaRO.js` 且内含 `/spots/strategy/configs`，`index.html` 引用的 js/css 均在 107 文件清单内 | ⚠️ **产物一致性 PASS；构建命令未独立复现** |
| §10.4 E2E-08 | 方案自评「部分通过」 | 与探针一致（费用分币单测 PASS，真实 USDC 成交未运行） | ✅ 口径诚实 |

**§7 结论**：第 1 项未签收、第 2 项存在版本口径错误、第 4 项存在全量红点未披露。故**不认可「S4 完整验收通过」**。

---

## 四、离线 Gate 实跑结果（均在隔离副本内）

| 检验 | 结果 |
| --- | --- |
| `go build ./...` | ✅ EXIT=0 |
| `go build -o /tmp/s4_main_bin .` | ✅ EXIT=0（未覆盖用户可执行文件） |
| `go test -count=1` 目标 5 包 | ✅ **5/5 PASS** |
| `go test -count=1 ./controllers -run '^TestS[14]Spot\|^TestS4'` | ✅ PASS |
| S2-R1 回归 `^TestS2F[1245]` | ✅ **4/4 PASS** |
| S3-R1 回归 `^TestS3F` | ✅ **12/12 PASS**（含 LIMIT/MARKET/DELETE payload、门禁矩阵、零 HTTP 计数、成本精度、panic 隔离） |
| `go test -count=1 -race` 目标 5 包 | ✅ **5/5 PASS**（仅 macOS `ld: malformed LC_DYSYMTAB` 非致命警告） |
| `go vet` 目标 5 包 + controllers | ✅ EXIT=0 |
| `GIT_OPTIONAL_LOCKS=0 git diff --check`（用户工作区） | ✅ EXIT=0 |
| 前端 `tsc --noEmit` / `vue-tsc --noEmit --skipLibCheck` | ✅ EXIT=0 / EXIT=0 |
| `go test -count=1 ./...` | ❌ EXIT=1，69 包 ok；红点归因见下 |
| 探针删除后最终回归（目标 5 包 + controllers S4） | ✅ 全绿，结论未被探针污染 |

**全量红点归因（均非 S4 引入）**：`controllers` 5 项（`TestQueryStrategyTemplatePageOrdersByIDDescending`、`TestStrategyTemplateAITaskUsesRuntimeAndPreservesConversation`、`TestStrategyTemplateValidationAcceptsMarketConditionStringComparison`、`TestStrategyTemplateValidationAcceptsMultilineLetExpression`、`TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals`）；`tests` 包 `panic: the target config file: /tmp/phase_audit_s4/conf/app.conf don't exist`（`conf/app.conf` 未入 Git，纯净副本必然缺）。本轮 `service/leadaccount` 未出现并行抖动。

---

## 五、探针证据汇总

运行时探针 `/tmp/phase_audit_s4/service/spotstrategy/zz_s4_probe_test.go`（临时，已随副本删除）；静态探针为只读 Python/AST 文本统计。全程 `export SPOT_STRATEGY_LIVE_TRADING_APPROVAL=''`，零真实 Binance 请求、零真实库写入。

| 探针 | 输出 | 含义 |
| --- | --- | --- |
| `PROBE_S4_GATE` | `mode=strategy approved=false live_mode_err=...disabled live_write_err=...disabled can_attest=false release_const=false` | **正向**：五条件齐备仍硬封闭，S3-F1 未回退 |
| `PROBE_S4_NOHOLDING` | `reconcile_status=filled fills=1 verified=0 settled=0 holdings=0 view_total=0` | **F1**：完整本地链路跑通后，S4 持仓周期页零数据 |
| `PROBE_S4_CHAIN` | `complete=false owned="" cost="" unrealized="" reason="fill_source_not_verified"` | **正向**：未核实 Fill 的成本/库存/收益数值全部隐藏（S3-F4 整改有效） |
| `PROBE_S4_CHAIN` | `apply_panic="runtime error: invalid memory address or nil pointer dereference"` | **F5**：`NewReadOnlyEngine(db,nil).ApplyVerifiedFills` 无 nil 防御 |
| `PROBE_S4_READONLY` | `rows_before=5 rows_after=5` | **正向**：4 个 S4 读接口零写入 |
| `PROBE_S4_SECRET` | `category="order_outcome_unknown" signature_leaked=false last_error_field_exists=false` | **正向**：`last_error` 中的 `signature=SECRET` 未进入 DTO |
| `PROBE_S4_UNREALIZED` | `list="" list_complete=true direct_with_price="40" direct_complete=true` | **F4**：`HoldingReport` 支持估值，但 `ListHoldings` 传空价，页面永远拿不到 |
| `PROBE_S4_FILTERS` | `incomplete_total=0 active_total=1 holding_status_filter_allows_incomplete=true` | **F6**：白名单允许 `incomplete`，但 DB `status` 仅 active/closed |
| `PROBE_S4_READONLY` | `fills_symbol_ETH=1 fills_default=1 symbol_filter_effective=false` | **F6**：`ListOrderFills` 静默忽略 `symbol` 参数 |
| `PROBE_S4_FLAG` | `hardcoded_live_approved_false=2 calls_LiveOrdersApproved_in_controller=0` | **F3**：状态展示与唯一真源脱钩 |
| `PROBE_S4_ROWSERR` | `s4_rows_next=4 s4_rows_err=3 cost_rows_next=1 cost_rows_err=0` | **F5**：`ListHoldings` 与 `replayHoldingCost` 缺 `rows.Err()` |
| `PROBE_S4_ROUTES` | `total=6 s4_get_only=4 non_get_verbs=[]` | **正向**：S4 新增 4 条全 GET |
| `PROBE_S4_WRITERS` | `create_prod_files=1 cancel_prod_files=1` | **正向**：`s3SendCreate/s3SendCancel` 生产调用点各 1 处（`Place`/`Cancel`），其余仅在测试 |
| `PROBE_S4_README` | `README.md v24_current_claim=4 v25_mentions=1`；`README.EN.md 5 / 1` | **F2**：版本口径冲突 |
| `PROBE_S4_SCHEMA` | `version=25` | 唯一真源仍 v25，S4 无新 DDL/migration |
| `PROBE_S4_UI` | `get_calls=5 put_calls=1 post_calls=0 delete_calls=0` | **正向**：唯一写操作是 S1 配置 PUT |
| `PROBE_STATIC` | `dist=107 static=107 common=107 identical=107 only_dist=0 only_static=0 hash_diff=0 index_match=True` | **正向**：静态产物完整一致 |
| `PROBE_I18N` | `zh=64 en=64 only_zh=[] only_en=[]` | **正向**：中英键完全对称 |
| `PROBE_EQ` | 八组 Go 文件 `mismatch=0` | 副本与用户工作区等价 |

---

## 六、Findings 详述

### S4-F1（P2）E2E 验收口径宽于实际可复现能力，业务数据闭环未被 S4 打通

S3-R1 刻意把 `canAttestLiveFills` 封闭为恒 false，导致 `s2_reconcile.go` 写入的每个新 Fill 都是 `rest_verified=0`；而 `ApplyVerifiedFills`（`s3_ledger.go:115`）与 `RecoverAndSettle`（`s3_recovery.go:29`）只消费 `rest_verified=1` 的 Fill。探针 `PROBE_S4_NOHOLDING` 用完整本地 Mock 链路（`Prepare → Submit → Reconcile → RecoverAndSettle`）实证：`fills=1 verified=0 settled=0 holdings=0 view_total=0`。

由此产生三处口径问题：

1. §10.4 把 E2E-02/03/04（BUY 部分成交、SELL 归零、同币新周期）标为「PASS（隔离业务）」，其数据来源是 S3/S4 单测中**手工 SQL 插入 `rest_verified=1` 或手工建 Holding** 的夹具，不是任何运行路径可产生的数据。
2. 状态行与 §10.5 写「S4 前端与本地只读 API 可以用于继续研发、代码评审和人工 UI 验收」，但人工 UI 验收在真实/本地运行链路上**持仓周期 tab 必然为空**，只有「委托」tab 能看到订单与未核实 Fill。
3. §7 第 1 项 checkbox 未勾，方案自己未签收完整 E2E，却在 §4 标题写「本地前后端实现、构建和隔离测试已验收」，两处口径需统一。

**判断**：这不是安全缺陷（封闭是有意的），也不应由 S4 单方面解除 —— 解除需要 S3-R1 列出的响应级来源证明与持久化授权。但**必须把 S4 定位改为「只读 UI 骨架 + 夹具演示已验收；业务数据闭环 BLOCKED」**，不得据此宣布方案 02 完成或把可复用规则直接交付 03 C0。

**建议**：在 §10.4 增加一列「数据是否可由运行路径产生」；把 E2E-02/03/04 标为「夹具验证 PASS / 运行链路 SKIPPED-BLOCKED」；在页面空状态提示中说明「实盘未授权且 Fill 未核实，持仓周期暂无数据」。

### S4-F2（P2）版本口径冲突 + 静态产物半发布风险（01-ALL F1 在 S4 重现）

1. **版本**：`README.md:38`「当前二进制数据库 Schema 要求 **v24**」、`:159`「当前代码要求 **Schema v24**」、`README.EN.md:38/:162` 同类表述共 9 处；代码唯一真源 `appversion.DatabaseSchemaVersion = 25`，S4 新段落 `README.md:773` / `README.EN.md:672` 又写「Schema v25」。同一份部署文档自相矛盾，运维若按 v24 判断会**漏做 v25 迁移**。（`:104/:112` 描述 v23→v24 历史段落可保留。）
2. **半发布**：`static/` 已由 `rsync -a --delete` 整体替换为新前端（107 文件，含 `index.html`），但 §10.1 明确「没有重启服务」、未替换后端二进制。新页面依赖 4 个**新增** GET 路由；若只同步静态文件而后端仍是旧二进制，页面所有 API 返回 404。这与 01-ALL 的 ALL-F1 属同一类风险，S4 未关闭反而在工作区再次制造。方案 §六 虽要求「二进制与 static 同版本 release/hash」，但当前工作区状态本身即半发布状态。

**建议**：修正全部「当前要求 v23/v24」表述为 v25 并指向 `appversion.DatabaseSchemaVersion`；发布 SOP 明确「二进制 + static + schema 一次性成套替换 + 回滚包」，严禁只 `git pull` 静态资源；本条为生产部署前的硬门槛。

### S4-F3（P2）`live_approved` 与 `execution` 为硬编码字面量，未接唯一真源

`controllers/spot_strategy.go:123` 与 `:162` 把 `"live_approved": false` 写死，`config.go:284` 把 `Execution: "live_disabled"` 写死；探针 `PROBE_S4_FLAG` 显示控制器**0 次**调用 `spotstrategy.LiveOrdersApproved()`。当前与真实门禁一致（常量 false），故不构成现实风险；但「页面顶部实盘授权状态」是安全状态展示，一旦将来解除源码层封闭或新增其它放行路径，页面会继续显示「未授权」，形成**状态与事实脱钩**。

**建议**：`live_approved` 改由 `spotstrategy.LiveOrdersApproved()` 返回；`execution` 由同一状态推导；补永久测试断言两者恒等（当前断言 `false`，解除时强制同步更新）。

### S4-F4（P2）方案 §二必需字段「当前市价 / 未实现估值」在前后端均缺失

`ListHoldings`（`s4_read.go:207`）调用 `engine.HoldingReport(ctx, id, "")`，而 `HoldingReport`（`s3_pnl.go:136-144`）只在 `priceQuote != ""` 时计算 `UnrealizedEstimateQuote`。探针 `PROBE_S4_UNREALIZED`：`list=""`、`direct_with_price="40"`（同一 Holding 传价 120 可算出 40）。前端 `SpotHolding.unrealized_estimate_quote` 类型已声明，但 `spotStrategy.vue` 模板无对应列；「本地最新价」只出现在策略配置 tab，持仓 tab 没有市价列。

**建议**：二选一并与方案文字对齐 —— 要么 `ListHoldings` 从本地 `spot_symbols.close` 取价并明确标注「本地价、非实时、不构成估值依据」后计算，要么把该字段从 §二必需字段中正式移除并同步前端类型；当前「声明支持但永远空」最易误导。

### S4-F5（P3）读取健壮性与 nil 防御缺口

1. `s4_read.go::ListHoldings`（第 192-201 行）遍历 `rows.Next()` 后只检查 `rows.Close()` 错误，**未检查 `rows.Err()`**；同文件其余 3 个 List 都检查了。DB 读取中途失败会静默返回部分 Holding 列表。
2. `s3_cost.go::replayHoldingCost`（S3-R1 新增，用于真实账本成本重放）同样缺 `rows.Err()`（探针 `cost_rows_next=1 cost_rows_err=0`），影响面比只读页面更高。
3. `NewReadOnlyEngine(db, nil)` 正是 `ListHoldings` 构造报表用的引擎；若后续在该引擎上调用 `ApplyVerifiedFills` 等需要 client 的方法会 nil 解引用 panic（探针 `PROBE_S4_CHAIN` 复现）。当前生产调用链不触发，但缺少 `if e.client == nil { return ErrLiveWriteDisabled }` 类防御。

### S4-F6（P3）若干 API 语义、测试覆盖与文档口径细节

1. `ListOrderFills` 的 `req.Symbol` 经 `normalizedPage` 校验后**未进入 SQL**（`symbol` 来自本地订单记录），传任意合法 symbol 结果不变（探针 `symbol_filter_effective=false`）。不影响安全（作用域由本地订单 ID 锁定），但参数被静默忽略。
2. `holdingStatusFilter` 允许 `incomplete`，而 `spot_strategy_holdings.status` 实际值域只有 `active/closed`（`incomplete` 是 `HoldingSummary.Complete` 的展示语义）——该筛选永远返回空（探针 `incomplete_total=0`）。前端也只提供 active/closed，前后端口径已不一致。
3. `controllers/spot_strategy_s4_test.go` 只覆盖 9 组 fail 路径（无 JWT / 错误 scope / 非法参数），**没有合法 JWT + 合法参数 → 200 + DTO 字段断言**的正向串接测试；服务层测试直接调函数，不经过 HTTP 层。
4. §10.3 未列 `go test ./...` 全量红点，§10.4 未说明「PASS（隔离业务）」不等于端到端通过；建议补一行明确的四类结果汇总。
5. `ListSettings` 列出所有 `spot_symbols` 行（含 `type` 非 USDT/USDC/FDUSD 的历史币），但 `UpdateSetting` 会拒绝编辑这些币 —— 页面可见但不可编辑，属可用性瑕疵。

---

## 七、正向实证（不因 Findings 抹杀已落地的安全设计）

| 项 | 证据 |
| --- | --- |
| 真实写路径仍硬封闭 | `PROBE_S4_GATE` 五条件齐备仍拒绝；`spotRealOrderReleaseApproved=false` 为编译期常量且全仓唯一定义；`canAttestLiveFills` 恒 false；`s3SendCreate/s3SendCancel` 生产调用点各 1 处且均在 `AuthorizeLiveWrite` 之后 |
| S3-R1 整改未被 S4 回退 | 12 个 `TestS3F*` 永久测试独立复跑全 PASS；S2-R1 四项回归 4/4 PASS；runner `recover()` panic 隔离存在 |
| 只读边界 | 4 条新路由 GET-only、全局 JWT 覆盖（白名单不含）、`authorized()` 二次校验、scope 硬编码、6 秒超时；探针零写入；前端 0 POST/DELETE；无解冻/清仓/接管/批量删除按钮 |
| 敏感信息 | `last_error` 中的 `signature=SECRET` 不进入 `OrderView`；错误统一为安全诊断码或脱敏 503；`JSONResp` 固定 JSON（Accept XML/YAML 仍 JSON） |
| 数据完整性 | 未核实 Fill 的成本/库存/毛净收益数值全部隐藏并给出 `complete=false` + 多原因；BNB 等第三资产费用不折算成 quote；`remaining_qty = requested − filled` 精确 Decimal，不误用 BUY 的 quote 预留额 |
| 归属与注入 | `holding_id`/本地订单 ID 先校验 Owner 再查子表；symbol/status/page/limit 白名单；参数化 SQL；外部 `exchangeOrderId` 不能冒用；`TestS4...` 5 个永久测试 PASS |
| 配置写入隔离 | `UpdateSetting` 白名单仅 4 个私有字段，不改 `spot_auto_trade_mode`、不碰旧 `enable`/`notice`/`listen_symbols`；`RequestMode` 仍禁跨模式切换 |
| 前后端一致性 | 后端 `Holding.ID` JSON 为 `holding_id`，前端 `SpotHolding.holding_id` 对齐；dist↔static 107/107 SHA256 全等；i18n 中英 64/64 对称；`tsc`/`vue-tsc` 只读复跑 EXIT=0 |
| 交付边界 | Schema v25 未变、无新 ORM 表/migration、无新 WS/常驻进程；未执行 `sync db`、未访问真实 Binance、未重启服务、未 commit/push |

---

## 八、生产 Gate（全部 BLOCKED，未授权）

| Gate | 状态 | 说明 |
| --- | --- | --- |
| 真实 Binance 下单 / 撤单 | ⛔ BLOCKED | 硬封闭成立，本轮零真实写请求 |
| `strategy` / `off` 模式真实切换 | ⛔ BLOCKED | `RequestMode` 仍硬拒；改库不能解除封闭 |
| 持久化授权（签名 + 有效期 + 操作者 + 撤销）+ append-only 审计 | ⛔ BLOCKED | S3-R1 §一/F1 明确未实现 |
| 响应级 `myTrades` 来源证明 / `rest_verified=1` | ⛔ BLOCKED | `canAttestLiveFills` 恒 false，S4 页面数据闭环因此不可用 |
| 旧自动 SELL 退出职责 / 库存 / 余额 / 外部挂单对账 | ⛔ BLOCKED | S0 §7.3 前置，S4 未处理 |
| 旧 `eatRate` 手动路由停用确认 | ⛔ BLOCKED | S0 §7.3 要求 |
| 浏览器人工 E2E / 截图 / 多语言交互巡检 | ⛔ 未运行 | §10.4 全部「尚缺项」 |
| 真实 USDC、429、断线重启、DB 故障演练 | ⛔ 未运行 | 仅隔离夹具 |
| 生产 MySQL 8 DDL / EXPLAIN / P95 | ⛔ BLOCKED | v25 仅 SQLite 与 DDL 文本验证 |
| 生产 v25 升级、二进制 + static + schema 成套发布与回滚 | ⛔ BLOCKED | F2：半发布状态未解除 |
| 014-F1（`applySettledRisk` 唯一风控入口）等 01 线未决项 | ⛔ BLOCKED | 未随 S4 关闭 |
| 方案 01 原生产 Gate | ⛔ 独立 BLOCKED | 不因 02 离线测试通过而改变 |

---

## 九、整改与复评要求

1. **F1 优先**：统一 S4 状态与 §10.4 的验收口径，明确「持仓周期页在实盘授权与响应级证据实现前无运行数据」，把 11 个 E2E 按「PASS / SKIPPED / BLOCKED / 未授权」四类重列；§7 第 1 项保持未勾直至有可复现输出。
2. **F2 为部署硬门槛**：修正 README 中英全部「当前 Schema v24」表述为 v25；补齐「二进制 + static + schema 成套发布与回滚」SOP 后再谈部署。
3. **F3/F4** 建议同期整改：`live_approved` 接 `LiveOrdersApproved()`；未实现估值字段二选一落地或正式从方案中移除。
4. **F5/F6**：补 `rows.Err()`、nil client 防御、正向 200 串接测试、`symbol` 参数语义、`incomplete` 状态白名单、全量红点披露。
5. 复评时**另起 `CODE_REVIEW_02-S4-rereview.md`**，不覆盖本报告。
6. **方案 03 C0 准入**：S4 可交付的是「只读 API 契约、DTO/校验规则、静态产物校验方法、S2/S3 回归证据」，**不可**把 E2E-02/03/04 的夹具结果当作业务闭环复用证据；03 C0 若依赖生产证据，须等上述 Gate 逐一独立验收与用户明确授权。

---

## 十、审计纪律声明

本轮**未修改任何业务代码**，只新增本报告与项目记忆；未 commit/push、未改 `conf/app.conf`、未写入 `static/` 或前端源码（仅只读比对与 `tsc/vue-tsc --noEmit`）、未触碰 `strategy_templates/`、未对真实库执行 `sync db`、未访问真实 Binance、未重启服务、未操作生产仓位；未重跑 `pnpm build`（避免 `rimraf dist` 覆盖用户产物）。所有临时探针写入 `/tmp/phase_audit_s4` 隔离副本并在验证后删除，副本与临时二进制 `/tmp/s4_main_bin` 在收尾时一并清理。报告中的「离线 PASS」仅指本地 Go/SQLite/Fake/拦截 transport/只读类型检查验证，**不代表任何实盘、浏览器 E2E 或生产 Gate 通过**。
