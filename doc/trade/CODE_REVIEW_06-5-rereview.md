# CODE_REVIEW_06-5：整改复评

日期：2026-11-12  
后端：`feat/trade_spot`，HEAD `f2f3c9b`（本轮未提交工作区）  
前端：`go_binance_futrues_new_ui`，保留用户已有订单页修改  
原审查报告：[CODE_REVIEW_06-5.md](CODE_REVIEW_06-5.md)（原文不修改）  
结果：**F1/F2 的 P2 代码缺陷已修复并有永久回归；F4/F5/F7 已整改；F6 已增加安全上限但性能优化尚未完全验收；F3/F8 属 06-6 发布与实盘硬 Gate，继续 BLOCKED。**

## 一、逐条整改结论

| ID | 严重度 | 结果 | 代码与永久证据 |
| --- | --- | --- | --- |
| **F1** | **P2 / 06-6 自动策略接线阻塞** | **已修复** | `service/spotledger/query.go` 的 `PositionFilter.Symbol` 恢复默认 `symbol=?` 精确查询；新增 `SymbolLike` 为**仅 UI 显式开启**的 `LIKE '%...%'`，后端 `SpotQuantAccountController.Positions()` 明确传 `SymbolLike: true`。06-4 的 `Processor.reconcileSymbolOrders()` 沿用默认精确查询，**无需修改执行核心**。新增 `TestReview065ExactSymbolRecoveryNoCrossCoin`：`BTCUSDT` 精确查仅得一条，UI `BTC` 模糊查得 `BTCUSDT + WBTCUSDT`，旧大小写归一化保持。 |
| **F2** | **P2** | **已修复** | `SpotReadService.Position()` 在 `FullyPriced=false` 时**不设置 `TotalPnL`**，包括 `CLOSED` 且没有现货报价的情形。允许显示已重放的 `RealizedPnL`，但不会冒充「完整最终净收益」。新增 `TestReview065ClosedUnpricedFeeNeverPublishesTotalPnL`：`CLOSED + BNB 手续费 + 无现货价格` 的仓位和汇总均返回空 `total_pnl`。原 `TestStage065ReadModelStalePriceAndUnpricedFees` 的断言同步收严。 |
| **F3** | **P2 / 发布硬 Gate** | **仍待 06-6** | 本轮仅重跑前端 Vite 构建，生成**前端本地** `dist/`；**没有复制后端 `static/`，也没有部分覆盖 index/chunk**。06-6 必须成套替换、验证哈希及具备回滚步骤。当前部署没有新的现货页面，不能表述成已发布。 |
| **F4** | P3 | **已修复** | 对 `src/views/dashboard/configShow.vue` 应用 Prettier，修正三元表达式换行；**包括配置卡片在内**的 06-5 现货 UI/路由/API ESLint `--max-warnings 0` PASS。前端全仓 `git diff --check` 的已知 `src/views/order/index.vue:659` 空格属于用户原有未提交修改，本轮**没有擅自修改**；06-5 已跟踪文件自身的 diff check PASS。 |
| **F5** | P3 | **已修复** | `SpotQuantController.authorized()` 二次检查经过 `JwtMiddleware` 写入 Context 的 `jwt.MapClaims.Username` 非空。币种/模板列表/增删改/导入/规则预览以及新 `SpotQuantAccountController` 的四条 API 均在读库或操作前检查，缺 claim 返回 **HTTP 401 JSON**。原控制器测试 fixture 显式注入合法 JWT claim，新增 `TestReview065SpotQuantControllerRejectsMissingOperatorClaim`、`TestReview065SpotAccountRejectsMissingOperatorClaim`，验证缺失及空白 claim 拒绝。`respondRead` 按非法入参 400、未找到 404、需复核 409、超限 422、其他数据库故障 500 分类，服务端记录异常并对客户端隐藏内部 SQL 错误。 |
| **F6** | P3 / 性能 | **安全限额已实现，性能验收未完成** | `SpotReadService.Positions` 在 `COUNT(*) > 1000` 时直接返回 `ErrSummaryTooLarge`（HTTP 422，要求缩小筛选），**不会重放一千多份 Fill 后返回假完整汇总**。对于 1～1000 条仍是逐仓位精确 Fill 计算，保留跨页且跨报价币隔离的财务语义；原 `viewByID` 缓存复用当前页已算数据。新增 `TestReview065SummarySizeFailClosedBeforeReplayingThousandsOfFills`（1100 条测试记录），以及 HTTP 422 测试。**这不等于 N+1 已优化**：06-6 仍需 MySQL 8 的 EXPLAIN / P95，若真实历史接近上限应再优化批量查询或汇总设计，不能把 SQLite 的 PASS 当 MySQL 性能证明。 |
| **F7** | P3 | **已修复** | 对 `Accounting` 返回 `ErrNeedsReview` 的仓位，仅保留可用于定位的 id/symbol/状态/时间；`BoughtQty`、`RemainingQty`、`AvgCostQuote`、`RealizedPnL` 在 API 返回的 `Position` 中置空，且顶层财务字段也留空。详情仍可查看原始 Fill/订单作为诊断证据，但不可把错误缓存作为可信仓位或盈亏。原 `TestStage065ReadModelErrorRequiresReviewWithoutMutableProfit` 增强断言。 |
| **F8** | P3 / 实盘硬 Gate | **仍待 06-6** | Testnet 真实 REST/WS、MySQL 8、v25 Schema 发布检查、风险/精度 dust、前后端/二进制成套发布、回滚和真仓授权均**未执行**；总开关、单币开关及交易操作不可因本地测试通过就直接放行。 |

## 二、影响范围

本轮整改修改（或补充）：

- `service/spotledger/query.go`：精确匹配契约恢复；UI 模糊匹配显式开关
- `service/spotledger/readmodel.go`：对外 PnL 严格门控、异常物化字段抹除、超大汇总硬限额
- `service/spotledger/readmodel_test.go`：F1/F2/F6/F7 针对性永久回归
- `controllers/spot_quant.go` / `controllers/spot_quant_account.go`：JWT 二次鉴权、UI symbolLike、HTTP 错误分类
- `controllers/spot_quant_test.go` / `controllers/spot_quant_account_test.go`：授权回归和响应测试
- `src/views/dashboard/configShow.vue`：修正 Prettier 不合格格式
- 06-5 实施记录与方案总览：如实同步结论与未完成的发布 Gate

**没有**修改 06-4 实际交易适配器的 Mainnet/Testnet 限制，没有新增现货真实买卖、撤单或自动信号路由；没有新增审计表/Owner 层，未修改 `main.go`、`conf/app.conf`、`appversion/version.go`、`static/`。原 `CODE_REVIEW_06-5.md` 保留。

## 三、实际执行的离线验证

| 验证 | 结果 |
| --- | --- |
| `go test ./command ./controllers ./models ./service/spotquant ./service/spotledger ./service/spotexecution ./spot/api/binance -count=1` | **PASS，7/7 目标包** |
| 同 7 包 `go test -race -count=1` | **PASS，7/7**（macOS 链接器有非致命 `LC_DYSYMTAB` 警告） |
| `go test ./service/spotledger ./controllers -run '^(TestReview065\|TestStage065)' -count=5 -p=2` | **PASS** |
| `go vet` 同 7 包 | **PASS** |
| `go build -o /tmp/spot_065_rereview_build .` | **PASS**，不覆盖运行中的二进制 |
| `git diff --check`（后端） | **PASS** |
| F1：完整币种精确 vs UI 模糊，含 `WBTCUSDT` | **PASS** |
| F2：CLOSED/BNB/无报价不下发 total_pnl；OPEN/BNB 已有码门控同步更新 | **PASS** |
| F5：无 JWT claim/空白 Username 均 401；超 1000 条汇总 422 | **PASS** |
| F6：1100 条匹配记录提前拒绝汇总 | **PASS** |
| F7：reconciliation required 时物化数量/盈亏 API 字段为空 | **PASS** |
| 前端 06-5 现货文件**加** `configShow.vue` 的 ESLint `--max-warnings 0` | **PASS** |
| `pnpm typecheck`（tsc + vue-tsc） | **PASS** |
| `pnpm build` | **PASS，2550 modules；仅覆盖本地 `dist/`** |
| Binance Testnet/Mainnet 实际交易与 WS | **没有执行** |
| MySQL 8 EXPLAIN/P95、正式 v25 数据库升级、成套部署/回滚 | **没有执行** |

**测试范围说明**：没有声称全仓 `go test ./...` 通过；原审计已记录该命令因历史 `strategy_templates` 和测试目录 `conf/app.conf` 环境依赖存在红点。06-5 F4 涉及的用户原有订单页面格式错误不属于本轮改动，仍由用户工作区保留；不把全仓前端 `git diff --check` 描述为 PASS。

## 四、阶段结论和 06-6 继续条件

- **06-5 原报告的 F1/F2 代码阻塞已解除**；F4/F5/F7 完成整改；F6 已限制最大处理量，但依然需要生产规模的性能实测与优化。
- 可以将当前 06-5 作为**UI/只读账本代码研发基线，继续 06-6 的隔离环境检查与 Testnet 联调准备**。这不是实盘发布批准，也不代表所有功能已经具备可执行 HTTP API。
- **生产、真实交易、自动信号与成套部署 Gate 仍 BLOCKED**：必须完成 F3/F8 的 Testnet、MySQL 8、数据库检查、`dist→static` 全量哈希校验、回滚、人工授权以及其他 06-6 条目；未经明确授权不得修改生产库或发主网订单。

本轮仅在用户授权开发机的本地仓库改代码和文档，测试使用 SQLite/原有 Fake，**未触发真实 Binance 订单、数据库升级、服务替换、自动 commit/push**。
