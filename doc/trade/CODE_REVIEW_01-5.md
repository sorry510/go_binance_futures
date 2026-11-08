# Code Review：方案 01-5「后端 API 和前端主子表」

- 审计日期：2026-11-08
- 审计基线：HEAD `5356d0c`（`feat/trade-upgrade`，01-0～01-3 + R3 整改已提交）
- 审计对象：工作区未提交改动中的 **01-5 增量**，叠加同一工作区的 01-4 与 01-4 整改记录
- 方案依据：`doc/trade/开发方案-01-合约订单审计与部分成交.md` §四 01-5（179–190 行）、§十六 01-5 实施记录（457–510 行）
- 对照文档：`doc/trade/CODE_REVIEW_01-4.md`、`doc/trade/CODE_REVIEW_01-4_整改记录.md`
- 审计方法：文档验收清单 → 源码逐行 → 永久测试 → 隔离副本临时探针 → 编译/测试/race/vet/diff-check

---

## 一、结论先行

| 项目 | 结论 |
| --- | --- |
| 01-5 逐项工作落地 | ✅ 6/6 全部落地，实现质量高于前几阶段 |
| 账务口径（`big.Rat`、净盈亏、收益率、完整性门禁） | ✅ 与方案定义一致，10 组额外探针全部 fail-closed |
| 主子表三层展开、只读、旧 `order` 不访问 | ✅ 源码 + 探针 + 前端交叉核对一致 |
| 离线 Gate（build / test / race / vet / diff-check） | ✅ 目标包全绿 |
| `controllers` 全包 | ❌ 4 项既有红点（与 01-5 无关，见 §五） |
| 前端类型检查（`tsc` + `vue-tsc --noEmit`） | ✅ 均 `EXIT=0` |
| 实盘 Gate / T0 / 生产部署 | ❌ **未通过、未授权**，且新增 3 项发布前置 |
| 是否建议合并 | **离线代码建议合并**；但 015-F1 建议合并前修复，015-F2/F3/F4 必须在生产发布前闭环；不得据此部署或启用 T0 |

**本轮新发现 9 项**：015-F1（P1，新增 API 的 `Accept` 头契约缺陷，含 panic）、015-F2（P1，前后端发布产物不同步）、015-F3（P2，性能/索引 Gate）、015-F4（P2，账户级读权限边界未定义）、015-F5（P2，前端响应类型 `any` 弱化）、015-F6～F9（P3）。

---

## 二、审计范围与基线

### 2.1 当前工作区叠加的三层改动

`GIT_OPTIONAL_LOCKS=0 git status --porcelain` 共 22 个已跟踪改动 + 8 组未跟踪文件，`git diff --stat` 为 `22 files changed, 412 insertions(+), 622 deletions(-)`。按内容分为：

1. **01-4 旧 `order` 业务退役**（上一轮已审计）：删 `feature/order.go`、`controllers/orders.go` 三端点 410、删 30 分钟任务、`models.Order` 仅 `sync db` 条件注册等。
2. **01-4 整改记录**（上一轮 F1～F8 的整改，本轮首次进入审查视野）：`service/tradefills/cooldown.go`、`legacy_order_gate_stage01_4_test.go`、`feature/order_retirement_stage01_4_test.go` AST 增强、`scanner/local_selector_v2.go` / `feature/strategy/coin/common.go` 冷却切到真实 Fill `trade_time`、`feature/feature.go` 删死代码。
3. **01-5 增量（本轮主体）**：
   - 新增 `service/orderhistory/history.go`、`history_test.go`
   - 重写 `controllers/orders.go`（410 tombstone → 新只读历史 API）、扩展 `controllers/orders_test.go`
   - `routers/router.go` 新增 `/orders/:position_id/details`
   - 前端 UI 仓库 `src/api/order.ts`、`src/views/order/index.vue`、`locales/zh-CN.yaml`、`locales/en.yaml`
   - 方案文档 §十六 实施记录

`appversion.DatabaseSchemaVersion = 23` 保持不变，01-5 未新建表、未做迁移，**与实施记录一致**。

### 2.2 副本等价性

按 `agent-phase-audit` 纪律建立 `/tmp/phase_audit_015`：`git archive HEAD` + 逐个拷贝全部未提交文件（含未跟踪目录 `service/orderhistory/`），并**按 ` D ` 行显式删除** `feature/order.go`（HEAD 中存在但工作区已删除，否则副本不等价）。9 个关键文件 sha256 逐一比对一致，`appversion/version.go` 为 v23。探针全部写在副本内，结束随副本删除。

---

## 三、验收矩阵（方案 §四 01-5）

| # | 验收项 | 代码证据 | 验证 | 结论 |
| --- | --- | --- | --- | --- |
| 1 | `GET /orders` 按 `futures_managed_positions.id` 分页，返回周期汇总、状态、完整性 | `Store.List`（history.go:234）、`Position`/`Page` 结构（:86/:130）、`OrderBy("-created_at","-id").Limit(...)`（:270） | 探针 `PROBE_FILTER` 7/7 筛选生效；`PROBE_PAGE_SCOPE` 分页不泄漏 | ✅ |
| 2 | `GET /orders/:position_id/details` 按 `account_id + position_id` 返回子订单及 Fill，展开才加载 | `Store.Details`（:300）双条件 `Filter("account_id").Filter("id")`；`project(..., includeFills=true)`；router.go:153 | 前端 `onExpand` 懒加载 + `detailCache[account:id]`；跨账户 404 由永久测试覆盖 | ✅ |
| 3 | `DELETE /orders`、`DELETE /orders/:id` 仍 410 | `controllers/orders.go:98-103`；router.go:152/154 | `TestStage015LegacyDeleteRoutesRemainGoneWithoutDB` + 探针 A `default code=410` | ✅ |
| 4 | 只统计新关联可确认历史；零成交未建仓订单仍在既有挂单/待成交列表 | `history.go` import 仅 `models` + `tradefills`；`QueryTable` 仅 `FuturesManagedPosition` / `FuturesManagedOrder`；Raw SQL 仅 `futures_trade_fills` | `TestStage015ZeroFilledUnboundOpenIsNotFabricated`；全目录 Grep 无 `models.Order` / `futures_orders` / 交易所 import / HTTP 客户端 | ✅ |
| 5 | 一轮 OPEN + 两次 CLOSE 主表 1 行、子表 3 行；第二轮再开新增一行 | `TestStage015OneCycleMultipleClosesExactFeesAndNominalReturn`、`TestStage015SecondSameSymbolCycleSeparatedAndAccountScoped` | 实测：`Total=1`、3 子订单、4 Fill、VWAP 110 / 137.5、手续费 0.8 USDT、净盈亏 109.2 USDT、收益率 `24.818181818181818182%`；同币同方向第二周期独立 | ✅ |
| 6 | 页面与实际 Fill 账务一致（前端主子表、后端算账、错误/不完整可见、删除控件移除） | `index.vue` 三级展开、`order.ts` 类型定义、i18n | 22 个 `incomplete_reason` 在 `zh-CN.yaml:1703-1725` 与 `en.yaml` **双语全覆盖**；无删除按钮/筛选删除/`closeOrderId`；`el-alert` 明示口径 | ✅ |

---

## 四、账务口径与完整性门禁逐项核对

### 4.1 计算规则（全部在后端，前端只做展示）

| 指标 | 实现 | 与方案一致性 |
| --- | --- | --- |
| 开/平量 | 按 `client_order_id` + 真实交易所订单号关联、只累加正 `qty` 的 Fill（history.go:440-466） | ✅ |
| 开/平均价 | `Σ(big.Rat(qty × price)) / Σ(qty)`（:457/:496/:499） | ✅ 精确十进制，不用 `requested_qty`、不用 REST 累计均价二次加权 |
| 净盈亏 | `Σrealized_pnl − Σcommission`（:589） | ✅ 仅成交毛盈亏与手续费，不含资金费 |
| 收益率 | `net / openValue × 100%`（:607） | ✅ 名义成交额口径，非杠杆保证金 ROI；i18n 已明示 |
| 费用 | 按 `commission_asset` 分桶（:484-489），不完整时整组清空（:617-625） | ✅ |
| 杠杆 | `Position.Leverage *int` 从不赋值 → 恒 `null`；前端 `row.leverage == null ? "-"` | ✅ 不用当前杠杆伪装历史 |
| `page_verified_net` | 仅 `Completeness=="verified"` 且 `NetPnL!=nil` 的当前页、按 `PnLAsset` 分桶（:281-297） | ✅ 探针实证按资产且不跨页 |

### 4.2 显示净盈亏的门禁（history.go:579-612）

`settlement_status=settled` 且同时满足：`status=closed`、`settlement_processed_at>0`、`opened>0`、`closed>0`、`pnlAsset ∈ {USDT,USDC}`、`sign(gross−fees) == SettlementPNLSign`、所有费用资产与 `pnlAsset` 可直接相减、开平量平衡、每张订单 `matches(qty, FilledQty)`、所有 Fill `rest_verified=1`、无活动订单、无身份/资产/订单号冲突。任一不满足即 `net_pnl/return_pct/gross_pnl = null`、`completeness=incomplete`、给出机器可读 `incomplete_reason`，并清空 `Fees`/`PnLAsset`。**未发现任何可冒充 0 收益的路径**。

### 4.3 永久测试覆盖

`service/orderhistory/history_test.go` 7 个测试：一次 OPEN 4 笔 Fill + 两次 CLOSE 的精确账务、同币再开与账户隔离、5 种不完整场景不伪造盈亏、非法分页/时间/注入式筛选、物理旧 `order` 表存在时只读不改、Algo `algoId ≠ actualOrderId` 与撤单后部分成交、零成交未建仓订单不造周期。`controllers/orders_test.go` 2 个测试共 8 个子例覆盖 410 与无 DB 的参数校验。

---

## 五、离线 Gate 实证（隔离副本 `/tmp/phase_audit_015`）

| 项 | 命令 / 范围 | 结果 |
| --- | --- | --- |
| 全量编译 | `go build ./...` | `EXIT=0` |
| 主程序二进制 | `go build -o /tmp/015_main_bin .` | `EXIT=0`，37 MB |
| 单元测试 | `go test -count=1` 11 个目标包（orderhistory / tradefills / futuresownership / feature / coin / scanner / leadaccount / agenttrade / binance / command / models） | **11/11 ok** |
| 竞态检测 | `go test -count=1 -race` 7 个核心包 | **7/7 ok**（`ld: malformed LC_DYSYMTAB` 为 macOS 非致命噪音） |
| 静态检查 | `go vet` 12 个包 + 主包 | `EXIT=0` |
| 控制器专项 | `go test -count=1 -v -run 'TestStage015\|TestStage014' ./controllers` | 2 个测试 / 8 个子例全 PASS |
| 控制器全包 | `go test -count=1 ./controllers` | ❌ 4 FAIL（见下） |
| 补丁检查 | 用户工作区 `git diff --check` | `EXIT=0` |
| 前端类型检查 | 仓库本地 `node_modules/.bin/tsc --noEmit`、`vue-tsc --noEmit --skipLibCheck` | 均 `EXIT=0` |

### `controllers` 全包红点归因（**与 01-5 无关**）

- 隔离副本全包：3 项 `strategy_template_ai_runtime_test.go` 失败（`technology 至少需要一个已启用的技术指标…`）+ 1 项 `TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals` 因副本不含未跟踪 `strategy_templates/` 目录报 `lstat ../strategy_templates: no such file`。
- 用户工作区（含 `strategy_templates/`）用 `-skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` 复跑：4 项失败，即 `TestQueryStrategyTemplatePageOrdersByIDDescending`、`TestStrategyTemplateAITaskUsesRuntimeAndPreservesConversation`、`TestStrategyTemplateValidationAcceptsMarketConditionStringComparison`、`TestStrategyTemplateValidationAcceptsMultilineLetExpression`。
- 上述文件均**不在本轮 01-5 改动清单**（改动仅 `controllers/orders.go`、`orders_test.go`、`futuresOrders.go`）；01-4 报告已用纯 HEAD 干净副本复现同类红点。结论：**既有测试隔离 / 策略模板 fixture 问题，单独立项处理，禁止写「全仓测试通过」，也不得为凑全绿删除用户的研究材料**。

### 前端检查

`pnpm` 不在非交互 shell 的 PATH（本机环境事实），改用仓库本地工具链实跑，两个命令均 `EXIT=0`（`vue-tsc` 耗时约 7 分钟）。未执行 `vite build`（会重写 `dist`，避免污染工作区）；`dist` 已存在且 mtime 为 2026-11-08 21:21。实施记录声明的 ESLint / `pnpm build` 本轮未重复执行。

---

## 六、临时探针实证

**探针 A（`controllers/zz_015_probe_test.go`）** —— `Ctx.Resp` 的 `Accept` 头契约：

| Accept | HTTP | 响应 |
| --- | --- | --- |
| 默认 / `application/json` | 410 | 标准 JSON `{"code":410,"data":null,"msg":...}` |
| `application/x-yaml` | 410 | **YAML 文本**：`code: 410\ndata: null\nmsg: ...`，非 JSON |
| `application/xml` / `text/xml` | **500** | `xml: unsupported type: map[string]interface {}` |
| `application/x-protobuf` | **panic** | `data.(proto.Message)` 类型断言失败，recover 前状态未写出 |

**探针 B/C（`service/orderhistory/zz_015_probe_test.go`）**：

```
PROBE_GATE          10/10 场景全部 fail_closed=true，reason 与预期逐字匹配
                    （sign_mismatch / status_conflict ×2 / unknown_settlement_asset /
                     mixed_pnl_assets / outstanding_managed_order /
                     duplicate_exchange_matching_order / nonzero_fee_missing_asset /
                     unsupported_order_intent / order_fill_quantity_mismatch）
PROBE_ACTIVE        completeness=pending net=<nil> gross=<nil> ret=<nil>
                    fees=[{USDT 0.5}] page_sum=map[]
PROBE_PAGE_SUM      total=2 list=2 sum=map[USDC:109.8 USDT:109.8]
PROBE_PAGE_SCOPE    page=1 sum=map[USDC:109.8] / page=2 sum=map[USDT:109.8]（不跨页累加）
PROBE_FILTER        symbol / owner / side / status / settlement / time_range /
                    account_scoped 共 7/7 精确收敛到 1 条
PROBE_NPLUS1        project_dbcalls=2 range_loops=5 dbcalls_inside_loops=0
```

> 说明：beego `orm.Ormer` 含未导出方法，外部包无法实现计数包装器，故「无 N+1」采用 AST 结构实证（`project()` 全部 `s.db()` 调用均在 range 循环外）+ 源码 `position_id IN (...)` 批量查询证据；未做 SQL 级计数。

---

## 七、发现与风险

### 015-F1 — P1｜新增 API 的 `Accept` 头契约缺陷（含 panic），建议合并前修复

`controllers/orders.go:18-24` 用 `ctrl.Ctx.Resp(map[string]any{...})`。beego `Context.Resp` 按 `Accept` 分发到 JSON / YAML / XML / Proto（已核对 `context.go:108-120` 与常量 `ApplicationProto="application/x-protobuf"`、`ApplicationYAML="application/x-yaml"`）。探针 A 实证：YAML 返回非契约文本、XML 返回 500、Proto 直接 panic。

前端 axios 未显式设置 `Accept`（`src/utils/http/index.ts` 无相关配置），默认 `application/json, text/plain, */*` 走 JSON，**正常路径不受影响**；但任意带上述 Accept 的客户端（curl、网关、监控、第三方集成）都会得到 500 / panic / 非 JSON 响应，panic 还会污染错误日志与告警。项目其它控制器用 `utils.ResJson` + `ServeJSON()` 固定 JSON。

**建议**：改用 `ctrl.Data["json"] = ...; ctrl.ServeJSON()`（或 `ctrl.JSONResp(...)`），并把「非 JSON Accept 仍返回 JSON」固化为 3 条永久测试。

### 015-F2 — P1｜前后端发布产物不同步，生产发布硬前置

后端 `static/index.html` mtime 为 2026-11-06 11:06，`git status --porcelain -- static/` 为空；前端 `dist` 构建于 2026-11-08 21:21。实施记录已声明「静态资源构建到 dist 但未覆盖后端 static」，属有意留到部署阶段。但当前状态下一旦只合并/部署后端，线上仍是旧页面，与新 API 不配套（01-4 的临时升级页或更早版本）。

**建议**：作为 01-6 / 发布 SOP 的硬前置，明确「后端二进制 + 前端 dist → static」的发布顺序、灰度与回滚路径，并留存双端版本记录；在 SOP 落地前不得单独上线后端。

### 015-F3 — P2｜性能与索引 Gate 仍未关闭

- `command/futures_trade_fills_migration.go` 已创建 `idx_ftf_position_id ON futures_trade_fills(position_id)`，`project()` 的 `position_id IN (...)` 可走索引（本轮核实，非缺失）。
- 但 `Store.project` 每页最多 100 周期一次性批量拉取全量 Fill 并在内存做 `big.Rat` 运算，MySQL 8 生产数据量下的 P95 / 慢 SQL / `EXPLAIN` 未测。
- 01-4 整改新增的 `service/tradefills/cooldown.go::RecentClosedSymbols` 按 `trade_time` 查询，而 `SchemaDDL` 与所有迁移**均未给 `trade_time` 建索引**；该路径为高频选币冷却，与 01-5 共用同一张表，会放大全表扫描。

实施记录已把索引评估列入 01-6，本轮确认**仍未关闭**，不能用 SQLite 结果推断生产性能。

### 015-F4 — P2｜账户级读权限边界未在方案中定义

`/orders` 与 `/orders/:position_id/details` 只受 `main.go:333` 的全局 `JwtMiddleware` 保护（`middlewares/auth.go` 白名单不含 `/orders`）。任何登录用户传 `account_id=lead` 即可读取带单账户的完整历史，包括 `client_order_id`、`exchange_order_id`、`actual_order_id`、逐笔成交价格/数量/手续费/已实现盈亏、`source_ref`。方案 01-5 未定义「只读历史是否需要账户级查看授权」，与 Stage 6 Lead 只读 API 的权限语义也未对齐。

不阻塞离线合并，但实盘 Gate 前需明确并写入方案（是否所有登录用户可读 Lead 历史、是否需要角色区分、是否需要对 `source_ref` 等内部标识脱敏）。

### 015-F5 — P2｜前端响应类型被 `any` 弱化

`src/api/order.ts:110-116` 两个 API 均写 `http.get<any, ...>`。核对 `src/utils/http/index.ts:207-213`：`get<T, P>` 的 `T` 是**响应类型**，`P` 是 axios `data` 泛型而非 params 类型。因此页面 `index.vue:115/151` 只能靠 `(res?.data || {}) as ManagedOrderPage` / `res?.data as ManagedCycleDetail` 强转，后端字段改名或缺失在编译期完全不可见。

**建议**：定义统一响应包装类型（如 `ApiResult<T> = { code:number; data:T; msg:string }`），改为 `http.get<ApiResult<ManagedOrderPage>, ...>` 并删除强转；同时修正第二个泛型的误用。

### 015-F6 — P3｜前端展示细节

- `index.vue:66-74` `signed()` 以 `text-green-600` 表示正收益、`text-red-500` 表示负收益，与国内「红涨绿跌」习惯相反（当前写法是加密/美股口径）。需确认是否符合使用者预期，若面向国内展示建议对调。
- `onExpand` 的 `expandedRows.some(item => item.position_id === row.position_id)` 依赖 `row-key="position_id"`；列表按单账户过滤，同页不会重复 ID，当前安全。
- `el-pagination` page-sizes 含 100，与后端 `Limit <= 100` 一致。
- `locales/*.yaml` 中 `testOrderPage` 段属于另一个页面 `src/views/order/testOrder.vue`（仍被引用，含删除按钮），**保留正确**，不属于方案要求清除的「无用 i18n 键」。

### 015-F7 — P3｜币种为模糊匹配但 UI 未提示

`history.go:244` 使用 `Filter("symbol__icontains", q.Symbol)`；永久测试把 `Symbol:"BTC"` 命中 `BTCUSDT` 当作特性。输入 `USDT` 会命中全部 USDT 交易对。UI placeholder 为「例如 BTCUSDT」，未提示模糊语义。建议在 UI 注明或在方案中固化。

### 015-F8 — P3｜Owner 下拉与后端枚举不同步

前端 `index.vue:204-211` 只列 5 个固定 owner（`auto_strategy` / `notice_auto_order` / `new_coin_rush` / `funding_rate` / `agent_trade`），后端 `ownerPattern` 允许任意 `[a-z][a-z0-9_]{0,31}`。若系统出现其它 owner，用户只能选「全部」。建议由后端枚举或配置生成下拉。

### 015-F9 — P3｜详情接口直接序列化内部模型

`Order.Managed` / `Order.Fills` 直接输出 `models.FuturesManagedOrder` / `models.FuturesTradeFill`，包含 `account_id`、`owner`、`source_ref`（可能是 proposal ID 等内部标识）、`client_order_id` 等。方案未禁止，但应在 API 文档中标注这些为内部标识，避免被误当作可对外披露的数据（与 015-F4 一并处理）。

---

## 八、前端交叉核对小结

| 检查点 | 结果 |
| --- | --- |
| 一级 `position_id` 一行、二级 Managed 子订单、三级 Fill | ✅ `el-table` 三层 expand |
| 展开才加载 | ✅ `onExpand` + `detailCache` + `requestToken` 竞态保护 |
| 删除按钮 / 筛选删除 / 旧 `closeOrderId` 查询 | ✅ 全部移除（Grep 无命中） |
| 价格与盈亏计算在后端 | ✅ 前端仅 `display()` / `formatTime()` / `formatDuration()` |
| 错误与不完整状态可见 | ✅ `ElMessage.error` + `incomplete_reason` tooltip + `completeness` tag + 22 个双语 reason |
| `leverage=null` → `-` | ✅ `index.vue:539-543` |
| `page_verified_net` 注明「仅本页」 | ✅ `summary.scope` / `pageExplanation`（zh-CN.yaml:1654-1656） |
| 后端 `static/` 未覆盖 | ✅ 但构成 015-F2 |

---

## 九、实盘 / 生产 Gate 阻塞清单

以下均未通过或未授权，本轮**未执行任何相关操作**：

1. 真实 Binance 部分成交订单的 WS `l/L/n/N/rp/T` 与 REST `userTrades` 逐 `tradeId` 比对（沿用 01-0～01-3 首轮 F1）。
2. 已归零 symbol-scoped `PositionRisk` 是否返回明确方向 row 的实盘证据；`applySettledRisk` 作为 01-4 后唯一自动亏损控制入口的有效性（014-F1）。
3. 保护单 opt-in `BINANCE_AGENT_PROTECTION_AUTO_REPAIR=enabled` 实盘验证。
4. 生产库 schema v23 升级、二进制替换、服务重启、实盘验证；生产 5 条未归属非零镜像仓位只读复核。
5. 01-6 完整回归、T0 启用前留存账户维度 T0 与版本记录。
6. **015-F2**：前端 `dist` → 后端 `static` 双端同步发布 / 灰度 / 回滚 SOP。
7. **015-F3**：MySQL 8 生产压测、P95、慢 SQL、`EXPLAIN`，并决策 `trade_time` 索引（必要时正式版本迁移补建）。
8. **015-F4**：`account_id` 读权限矩阵与内部标识脱敏策略确认。
9. `controllers` 全包 4 项既有红点单独立项（014-F5 沿用）。

---

## 十、边界与纪律声明

- 本报告仅代表 **离线 PASS**（Go build / test / race / vet / diff-check + SQLite 夹具 + 临时探针 + 源码/AST 静态核对）。**实盘 Gate 一律未通过、未授权**。
- 未修改任何业务代码；未 commit / push；未改 `conf/app.conf`；未对真实库执行 `sync db`；未访问真实 Binance；未重启服务；未操作生产仓位；未触碰 `strategy_templates/`；未覆盖后端 `static/`。
- 前端 UI 仓库仅做只读核对与 `--noEmit` 类型检查，未执行会重写 `dist` 的 `vite build`。
- 隔离副本 `/tmp/phase_audit_015` 内的 2 个临时探针随副本删除，不进入用户工作区。
- 当前工作区还叠加 01-4 与 01-4 整改记录，本报告结论**不覆盖它们的独立复评**；014-F1 / 014-F2 等既有未决项仍按上一轮报告跟踪。

---

## 附录：关键代码位置索引

| 位置 | 内容 |
| --- | --- |
| `service/orderhistory/history.go:33-80` | `Filter.validated()` 白名单与分页/时间边界 |
| `history.go:155-192` | `big.Rat` 十进制、`val()` 定长 18 位、`matches()` 1e-9 容差 |
| `history.go:222-232` | `realExchangeID()`（Algo `algoId` 不作撮合 ID）、`closingIntent()` |
| `history.go:234-299` | `List()` 分页、`PageVerifiedNet` 按资产聚合 |
| `history.go:300-321` | `Details()` 双条件账户隔离、`ErrNotFound` |
| `history.go:322-382` | `project()` 批量 `position_id IN (...)`，无 N+1 |
| `history.go:384-627` | `summarize()` 全部账务与 22 个 `incomplete_reason` |
| `controllers/orders.go` | `Get` / `Details` / 410 `Delete` / `DeleteAll`，400/404/410/500 映射 |
| `routers/router.go:152-154` | 三条路由 |
| `middlewares/auth.go:19-28` | JWT 白名单（不含 `/orders`） |
| `command/futures_trade_fills_migration.go:18-38` | `idx_ftf_position_id` 索引 |
| `service/tradefills/schema.go:6-25` | `futures_trade_fills` DDL 单一真源（无 `trade_time` 索引） |
| UI `src/api/order.ts` / `src/views/order/index.vue` | 类型定义与三层主子表 |
| UI `locales/{zh-CN,en}.yaml` `orderPage.reason.*` | 22 个双语原因全覆盖 |

---

## 十一、复审后整改状态（2026-11-08）

本节是原 Code Review 之后的整改状态追记，不改写本报告的原始现场证据。**015-F1 已修复**：`/orders` 所有 200/400/401/404/410/500 JSON 回复不再依赖客户端 `Accept` 协商，XML/YAML/Proto 输入及成功响应的 JSON 回归已经加入永久测试。**015-F5/F7/F8/F9 已修复**：TS 泛型响应契约、模糊币种提示、可输入的新 Owner、内部 ORM/source_ref 响应白名单。**015-F4 在现有单一登录账户模型下已明确 main/Lead 读权限共用同一操作员，并增加 Controller 的 JwtMiddleware claims 防御性检查**；系统并无独立用户/角色 RBAC，未来多用户上线前必须设计账户级权限。

**015-F2/F3 仍是 01-6 生产发布硬 Gate**：未同步发布后端二进制/static，未对目标生产数据做 MySQL 8 真实查询性能/EXPLAIN，未对生产数据库执行新索引迁移。015-F6 保持全站盈利绿色/亏损红色视觉习惯（避免与同站其它历史页不一致）。

详细代码、测试结果、账号权限决定和双端 staging/同步切换/回滚 SOP 见 [CODE_REVIEW_01-5_整改记录.md](CODE_REVIEW_01-5_整改记录.md)。原 `controllers` 全包既有 fixture 失败仍另案处理，不能据此推断真实交易 Gate 通过。**禁止在 01-6 之前部署或启用 T0。**
