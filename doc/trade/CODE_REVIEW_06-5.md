# CODE_REVIEW_06-5：四菜单页面与本地账本联动

日期：2026-11-11
分支 / HEAD：`feat/trade_spot` / `f2f3c9b feat: 06-4`（本地 master `7ed6c93`，Schema **v25**）
前端仓库：`go_binance_futrues_new_ui` HEAD `89f15bd`
审计对象：**工作区未提交的 06-5 改动**（HEAD 已包含 06-0～06-4 全部代码及 `CODE_REVIEW_06-4.md` / `-rereview.md`）
依据：`开发方案-06-现货量化交易-复刻合约简化版.md`（§6 06-5 行、§8）、`开发方案-06-5-四菜单页面配置与账本展示-实施记录.md`、`CODE_REVIEW_06-4.md` / `CODE_REVIEW_06-4-rereview.md`、`CODE_REVIEW_06-2.md`（F7）、`开发方案-06-发布前Schema验证检查单.md`
原报告保留：复评须另起 `CODE_REVIEW_06-5-rereview.md`

> **一句话结论**：06-5 的**四菜单、只读账本 API、中英文案、资金展示门控与离线 Gate 均达标**，是一份合格的 UI/只读研发基线；但把 UI 模糊搜索下推到 06-4 执行层共享的 `ListPositions` 造成**跨币种匹配回归**，且 API 在 `fully_priced=false` 时仍下发 `total_pnl`。**不认可 06-5 无条件完整验收；F1 必须在 06-6 接自动信号前修复**；生产 / 实盘 / Testnet 真实下单全部 BLOCKED。

---

## 一、基线与工作区归属

后端 `git status --porcelain` 共 **9 项**，全部属 06-5：

| 状态 | 路径 |
| --- | --- |
| M | `doc/trade/开发方案-06-现货量化交易-复刻合约简化版.md`、`doc/trade/开发方案-总览与实施顺序.md` |
| M | `routers/router.go`（+4 路由）、`service/spotledger/query.go`（symbol 精确 → LIKE） |
| ?? | `controllers/spot_quant_account.go`（143 行）、`controllers/spot_quant_account_test.go`（42 行） |
| ?? | `service/spotledger/readmodel.go`（280 行）、`service/spotledger/readmodel_test.go`（167 行） |
| ?? | `doc/trade/开发方案-06-5-四菜单页面配置与账本展示-实施记录.md` |

前端 10 项中 06-5 相关：`M locales/{en,zh-CN}.yaml`（各 +104 行 / -4）、`M src/views/dashboard/configShow.vue`（新增独立配置卡片）、`?? src/api/spotQuant.ts`、`?? src/router/modules/spotQuant.ts`、`?? src/views/spotQuant/`（新增 `account.vue` 473 行、`controlledTrade.vue` 410 行、`positionDetail.vue` 255 行、`format.ts` 31 行；`symbol.vue` / `strategyTemplate.vue` 沿用 06-2）。`M src/api/order.ts`、`M src/composables/useStrategyTemplateOptions.ts`、`M src/views/order/index.vue` 为**用户既有订单页改动**，不计入本阶段、未修改。`main.go` / `conf/app.conf` / `appversion/version.go` 本阶段无改动。

## 二、前置 Gate 独立复核（关键：06-4 整改是否真的闭环）

HEAD `f2f3c9b` 提交中含开发侧撰写的 `CODE_REVIEW_06-4-rereview.md`（47 行，非本次审计撰写），声称 F1～F8 已整改。**未直接采信，逐项独立核实**：

| 06-4 Finding | 独立复核结论 |
| --- | --- |
| **F1 P1** SELL 留 0.5% dust、仓位永不 CLOSED | ✅ **属实且独立探针验证生效**。`rules.go::planSell` 已移除 `sellBaseFeeReserve`，改为卖出精确策略库存并校验 step 整除；探针 `PROBE065_LIFECYCLE buy=0.99 sell=0.99 remaining=0 status=CLOSED places=2`、`PROBE065_NEW_CYCLE err=<nil> places=3`（对比 06-4 首轮 `remaining=0.005 status=OPEN` / 新周期被拒）。 |
| **F2 P1** 权威 -2013 与网络错误不分流、无恢复入口 | ✅ 属实。`types.go:19 ErrNotFound`、`processor.go:391/402` `ReconcileByClientID` / `ConfirmNeverAccepted`、`lookup.go:30`；`TestReview064ExplicitAbsentOrderRecovery` 等 4 个专项 PASS。 |
| **F3 P2** WS 指向主网 | ✅ 属实。`spot_quant_stream.go:23` 固定 `wss://ws-api.testnet.binance.vision/ws-api/v3`，`spotQuantStreamEndpoint()` 校验 `a.c.BaseURL` 精确等于 testnet REST；`spot_quant_adapter.go:178/201` 在 `Place` / `Cancel` 实际请求时复查 BaseURL；不再调用 `sdk.WsUserDataServeSignature`，不改包级 `sdk.UseTestnet`。 |
| **F4 P2** FILLED 先于 myTrades 断流 | ✅ 属实。`applySpotQuantOrderEvent` 对 `ErrNeedsReview` 走只读 `ReconcileByClientID`，不停止流；`TestReview064WSFilledBeforeMyTradesKeepsStream` PASS。 |
| **F5～F7** gate 时序、REST 放大、FeeAsset | ✅ 均有代码（`needsRemoteCheck`）与对应 `TestReview064*` 永久测试。 |
| **F8/F9** 文档口径、Testnet/MySQL 8/部署 | ⚠️ F8 文档已同步；F9 **仍 BLOCKED**（06-6）。 |

**`TestReview064*` 共 10 个永久测试实跑全 PASS**（`service/spotexecution` 8 个 + `spot/api/binance` 2 个）。**结论：06-4 首轮 F1/F2 两个 P1 阻塞已在本地代码层闭环，06-5 的准入成立。** rereview 同时明确「06-5 仅可做页面/只读联动研发，自动信号循环与真实执行接线仍受 Gate 限制」——本次核实 06-5 确实遵守。

## 三、隔离副本与等价校验

- `/tmp/phase_audit_065` = `git archive HEAD | tar -x` + 依 `git status --porcelain -z` 同步全部 9 项 M/?? 文件（删除项 0）。
- **665 个 Go 文件 SHA256 全等**，`MISMATCH=0 / MISSING=0`；探针只写副本内，结束随副本删除；全程 `unset SPOT_STRATEGY_LIVE_TRADING_APPROVAL SPOT_LEDGER_MYSQL_DSN`，探针为 Fake/SQLite，不联网。

## 四、离线 Gate 实测

| 项目 | 结果 |
| --- | --- |
| `go test -count=1` 7 个目标包 | ✅ **7/7 ok** |
| 同 7 包 `-race` | ✅ **7/7 ok** |
| 06-5 专项 `TestStage065*`（controllers + spotledger） | ✅ **5/5 PASS** |
| 06-4 整改 `TestReview064*` | ✅ **10/10 PASS** |
| `go vet` 7 包 / `go build -o /tmp/... .` / `git diff --check`（后端） | ✅ EXIT=0 / 0 / 0 |
| 前端 `tsc --noEmit`、`vue-tsc --noEmit --skipLibCheck` | ✅ 双双 EXIT=0 |
| 前端 ESLint（6 个新增 spotQuant 文件，`--max-warnings 0`） | ✅ EXIT=0 |
| **前端 ESLint（含 06-5 新增 `configShow.vue`）** | ❌ **EXIT=1**，见 F4 |
| 前端 `git diff --check` | ❌ EXIT=2，经核实属用户既有 `order/index.vue:659`，见 F4 |
| `dist` ↔ `static` SHA256 | ⚠️ 见 F3：`dist=115 static=106 common=36 identical=35 only_dist=79 only_static=70 differing=1(index.html)` |
| 干净副本 `go test -count=1 ./...` | ❌ EXIT=1，**71 包 ok**；红点为既有归因（`controllers` 缺 `../strategy_templates`、`tests`/root 缺 `conf/app.conf`）。**不得写「全仓测试通过」** |

⚠️ 实施记录 §四声称 `pnpm build` PASS（2550 模块）：**本次未独立复跑**，因为 `pnpm build` 会 `rimraf dist` 并覆盖用户工作区产物，违反只读审计边界；仅做只读类型检查与 ESLint。`dist` 现状按用户工作区原样保留。

## 五、静态审查（正向不变量）

1. **零自动交易接线**：`routers/router.go` 仅新增 4 条路由 —— `GET /spot-quant/positions`、`GET /spot-quant/positions/:id`、`GET /spot-quant/orders`、`POST /spot-quant/positions/:id/rebuild`；**没有** Place / Cancel / ExecuteSell / Reconcile / ConfirmNeverAccepted 的 HTTP 接口。`main.go` / `loopRun` / `init` 无 06-5 接线，`Processor` 仍无运行时注册。
2. **页面交易按钮真禁用**：`controlledTrade.vue:270/273/388` 为 `<el-button link disabled>` + 未开放 tooltip；本地重算需 `ElMessageBox.confirm` 二次确认且 `needs_review` 时拦截；后端 `LocalRebuild` 再校验 `confirm:true` 与 `ReviewPosition` 无异常，构成双保险。
3. **账本权威与不伪造盈亏**：只读 API 全走 06-3 `Accounting` 的 Fill 重放，不拿物化 `realized_pnl` 当权威；价格只读本地 `spot_symbols.close + updateTime` 且校验 5 分钟新鲜度，绝不取 Futures markPrice / PositionRisk；USDT/USDC/FDUSD 按报价币分组、不跨币相加；`fully_priced=false` 时前端 `hasPreciseSpotPnL()`（`format.ts:17`）门控不显示总盈亏。
4. **输入与 SQL 安全**：`assetPattern ^[A-Z0-9]{1,32}$` 拒绝 `%` / `_` 通配符，SQL 全参数化；`orderStatuses` 白名单拒 SHORT；异常 id 在访问 DB 前被拒（永久测试 4 例）。
5. **响应与鉴权**：继承 `SpotQuantController.respond` 的 `JSONResp`（控制器测试用 `Accept: application/xml` 验证返回 400 而非 500/panic）；全局 `JwtMiddleware` 白名单不含 `/spot-quant/*`。
6. **前后端契约**：四菜单 `showLink: true`、`rank=2.1`；`menus.spotQuant*` 四个 key 与 `spotQuantPage:` 段在中英文件均存在；`src/api/spotQuant.ts` 的 `SpotPositionView` / `SpotQuoteSummary` / `SpotPositionDetail` 与后端 DTO 字段一一对应；`saveField('spot_quant_enable', value)` → `editData({ [field]: value })` 单字段 PUT，符合 06-1 白名单。
7. **精度**：`formatSpotDecimal` 用正则裁剪而非 JS Number，PnL 与数量保持 DECIMAL 字符串。

## 六、探针实证（副本内，Fake/SQLite，不联网）

### 探针 A：共享 `ListPositions` 的 LIKE 语义扩散
```
PROBE065_LIKE_SPILLOVER query=BTCUSDT total=2 symbols=WBTCUSDT,BTCUSDT,
PROBE065_LIKE_SPILLOVER query=BTC     total=2 symbols=WBTCUSDT,BTCUSDT,
```
方法：创建 `BTCUSDT` / `WBTCUSDT` / `ETHUSDT` 三个合法仓位，调用 `Ledger.ListPositions(PositionFilter{Symbol: ...})`。**用完整精确 symbol `BTCUSDT` 查询仍返回 2 个 symbol**，因为 `query.go:45` 已改为 `symbol LIKE '%BTCUSDT%'`。

### 探针 B：CLOSED + 无报价 + 未折价手续费
```
PROBE065_CLOSED_UNPRICED status=CLOSED has_price=false fully_priced=false realized=10.000000000000000000 total="10.000000000000000000" unpriced_fees=map[BNB:0.001000000000000000]
```
方法：BUY/SELL 各一笔完整成交，BUY 手续费为 BNB，不写 `spot_symbols` 行情行，调用 `SpotReadService.Position()`。

### 探针 D：06-4 整改独立复核
```
PROBE065_LIFECYCLE buy=0.99 sell=0.99 remaining=0.000000000000000000 status=CLOSED places=2
PROBE065_NEW_CYCLE err=<nil> places=3
```

## 七、Findings

| ID | 级别 | 结论 | 证据 | 建议 |
| --- | --- | --- | --- | --- |
| **06-5-F1** | **P2 / 阻塞 06-6 接自动信号** | **UI 模糊搜索被下推到 06-4 执行层共享的 `Ledger.ListPositions`，把精确 symbol 契约改成 `LIKE`，造成跨币种匹配** | `service/spotledger/query.go:45` 由 `symbol=?` 改为 `symbol LIKE ?` + `"%"+f.Symbol+"%"`；`processor.go:344 reconcileSymbolOrders` 以精确 `Symbol` 调用同一方法；探针 A 用完整 `BTCUSDT` 查到 `WBTCUSDT`。`assetPattern` 只防注入、不防语义扩散 | 在 `PositionFilter` 新增 `SymbolLike` 字段或在 `readmodel` 独立实现模糊查询，`ListPositions` 恢复 `symbol=?`；补「精确 symbol 不跨币」永久测试。**当前 06-5 无自动信号接线故不可达真实执行，但 06-6 接线后必然显现** |
| **06-5-F2** | **P2** | **API 在 `fully_priced=false` 时仍下发 `total_pnl`，与「不显示为完整净收益」的契约不一致** | `readmodel.go:120-123` 在 `CLOSED && !HasMarkPrice` 无条件 `TotalPnL = RealizedPnL`，不检查 `FullyPriced`；探针 B 显示 BNB 手续费未折价却给出 `total="10"`。当前前端 `hasPreciseSpotPnL()` 靠 `fully_priced` 兜住显示，但任何直接消费 API 的 06-6 报表/导出会误读 | 改成 `&& view.FullyPriced` 才填 `TotalPnL`；或保持空并让 UI 明确标「未折价」；补该组合的永久测试（现有 `TestStage065ReadModelStalePriceAndUnpricedFees` 只覆盖 OPEN 仓位，未覆盖 CLOSED） |
| **06-5-F3** | **P2 / 06-6 硬 Gate** | **`dist` ↔ `static` 由历史 106/106 全等断裂；`index.html` 是唯一同名不同内容文件，后端 `static/` 不含 06-5 四菜单页面** | 产物比对 `dist=115 static=106 common=36 identical=35 only_dist=79 only_static=70 differing=1(index.html)`；实施记录 `:76` 已声明 06-6 才同步 | 属**已知且被接受的阶段边界**，不判 06-5 失败；但 06-6 发布必须整体替换并附回滚预案，**严禁只换部分 chunk 或只换 `index.html`**，否则引用不存在 chunk 会白屏。与 01-ALL F1 / 02-S4-F2 半发布风险同类 |
| **06-5-F4** | P3 | **验收声明不严谨**：实施记录称「新增现货 UI/路由/API ESLint `--max-warnings 0` PASS」，但 06-5 明确新增的 `configShow.vue` 有 1 个 prettier 错误 | 实测：`pnpm exec eslint --max-warnings 0`（6 个 spotQuant 文件）EXIT=0；加入 `src/views/dashboard/configShow.vue` 后 EXIT=1，`:277` 三元表达式换行。另前端 `git diff --check` EXIT=2（`src/views/order/index.vue:659 trailing whitespace`），经核实属用户既有订单页改动、不属 06-5 | 修正格式或把声明限定为实际检查范围；阶段记录中区分「06-5 改动」与「用户既有改动」的白空格 |
| **06-5-F5** | P3 | **06-2-F7 承诺后移到 06-5 的控制器 `authorized()` 二次校验仍未落地** | `SpotQuantController` / 新增 `SpotQuantAccountController` 均无 `authorized()`（对比 `controllers/orders.go:36`、`lead_account.go:57`）；本阶段唯一写接口 `POST /positions/:id/rebuild` 同样没有。全局 `JwtMiddleware` 白名单不含 `/spot-quant/*`，**不是未授权漏洞**，但不符合 015-F1 既定规范 | 06-6 统一补 `authorized()` 校验 JWT `Username` claim；`respondRead` 应区分 4xx/5xx 并记日志，当前所有错误统一 400 |
| **06-5-F6** | P3 | **跨页 summary 为无上限 N+1 查询** | `readmodel.go:160-209` 对全部筛选匹配仓位逐条 `s.Position()`，每条含 Fill 重放 + `spotPrice` + `ReviewPosition`；分页只限制返回列表 100 条，summary 全量遍历，无上限/缓存/超时。06-3 已知 `trade_time` 无索引，MySQL 8 P95/EXPLAIN 未测 | 加总量上限、批量预取或物化汇总；06-6 做 EXPLAIN / P95。另 `QuoteSummary` 只输出 USDT/USDC/FDUSD，其他 quote 仓位静默不进 summary（当前 `validateIdentity` 保证仅这三种，风险低） |
| **06-5-F7** | P3 | 异常仓位仍把物化行下发给前端 | `readmodel.go:102-108` 在 `ErrNeedsReview` 时 `view.Position = stored`（含 `realized_pnl`、`remaining_qty`），仅靠 `NeedsReview=true` 标记。`RealizedPnL`/`TotalPnL` 已留空、前端也会隐藏，但 API 层仍含脏字段，与 06-3-rereview「物化字段不可作为金融权威」的严格读法有张力 | 异常时清空这些字段或显式标 `untrusted: true` |
| **06-5-F8** | P3 / Gate | Testnet 真实成交 / WS 长连、真实 SELL/CANCEL/REST 对账、MySQL 8、发布前 v25 Schema 检查单、`dist→static` 成套发布与回滚**均未执行**；`pnpm build` 本次未独立复跑（避免覆盖用户 `dist`） | 实施记录 `:73-76` 已如实写明；本次复核确认 | 全部归 06-6；不得以本阶段离线 PASS 宣称实盘可用 |

## 八、验收矩阵与结论

| 项目 | 结论 |
| --- | --- |
| 06-4 前置 Gate（F1/F2 P1） | ✅ **独立复核闭环**（10 个整改永久测试 + 生命周期探针） |
| 后端 9 项 / 前端 06-5 改动归属与副本等价 | ✅ 665 Go 文件 SHA256 全等 |
| 代码与离线 Gate（test/race/vet/build/diff/tsc/vue-tsc） | ✅ **PASS 研发基线** |
| 四菜单、配置卡片、中英文案、前后端契约 | ✅ 符合方案 §6 06-5 验收行 |
| 零自动交易接线 / 页面按钮禁用 / 只读账本权威 / 不伪造行情 | ✅ 均有静态或探针实证 |
| **06-5 阶段整体验收** | ⚠️ **有条件，不认可无条件完整验收**（F1/F2 需修） |
| **06-5 → 06-6 接自动信号** | ⚠️ **F1 必须先修**；F3 成套发布为硬 Gate |
| 方案 06 完成 / Testnet 真实下单 / 生产发布 | ⛔ **全部 BLOCKED** |

**建议整改顺序**：① F1（恢复 `ListPositions` 精确匹配，模糊查询另开字段）→ ② F2（`fully_priced` 门控 `total_pnl` + CLOSED 组合测试）→ ③ F4/F5/F7（格式、二次鉴权、异常字段）→ ④ F6 性能与 06-6 EXPLAIN → ⑤ F3/F8 统一在 06-6 做 Testnet 端到端、MySQL 8、发布前 Schema 检查单与 `dist/static`/二进制成套发布。整改复评另起 `-rereview.md`，不覆盖本报告。

## 九、审计边界声明

本次仅做只读代码审查、本地 `go test/vet/build`、`Fake`+临时 SQLite 探针、只读前端类型检查/ESLint 与静态产物比对；**未**跑 `pnpm build`（避免覆盖用户 `dist`）、**未**同步 `static/`、**未**连接真实 Binance（含 Testnet）、**未**建立 WS 长连、**未**发送任何真实订单、**未**执行生产或实验库 `sync db`、**未**修改 `conf/app.conf`、**未**重启服务、**未**改用户业务代码、**未**自动 commit/push。探针仅在 `/tmp/phase_audit_065` 内，收尾随副本删除。实施记录中「MySQL 5.6 Docker 复跑 PASS」属 06-3-rereview 的开发侧声明，本会话未复跑，MySQL 8 仍未验证。
