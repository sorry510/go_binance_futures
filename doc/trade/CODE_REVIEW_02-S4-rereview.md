# CODE_REVIEW_02-S4 · R1 整改复核

- 日期：2026-11-10
- 原始审查：[CODE_REVIEW_02-S4.md](CODE_REVIEW_02-S4.md)，**原文件保留不改**
- 方案记录：[开发方案-02-S4-前端集成与完整验收.md](开发方案-02-S4-前端集成与完整验收.md)
- 发布 SOP：[发布检查-02-S4-成套交付与回滚.md](发布检查-02-S4-成套交付与回滚.md)
- 仓库基线：`feat/trade-upgrade` HEAD `95b5a77`，保留 S3-R1/S4 既有未提交代码；**没有 commit/push**
- 范围：仅本地 S4 只读页面/API、状态契约、测试和文档。**没有启用真实 Binance 委托/撤单或触碰生产数据库**。

## 一、复评判断

**可交付**：S4 只读 UI 骨架、配置读取与白名单更新、Holding→委托→Fill 分页 DTO、本地已核实 Fixture 的价格展示、双语页面及 Go/Vue 自动回归证据。

**不可交付**：实盘 Spot 订单/撤单、生产部署、模式切换、由真实查询自动生成的 Holding 与成本/收益业务闭环、E2E-02/03/04 完整交易场景、浏览器人工 E2E、依赖真实生产证据的 03 C0 准入。

当前 S3-R1 的 `spotRealOrderReleaseApproved=false` 与 `canAttestLiveFills=false` **均保持不变**。真实运行产生的 Fill 无法被标记为 `rest_verified=1`，因此策略结算不能创建已核实 Holding。这是**必要封锁而不是 S4 业务已闭环**。

## 二、Finding 对照

| 审查项 | 修改与可复现证据 | R1 判断 |
| --- | --- | --- |
| **S4-F1 P2** 夹具“通过”被误称完整 E2E，运行无 Holding | 文档 §10.4 E2E-02/03/04 改为「**SQL Fixture PASS／运行链路 BLOCKED**」；§7 第一项继续未勾选。中英文 Holding 空状态说明 S3 来源认证/交易授权仍封闭。新 `TestS4F1RuntimeUnverifiedTradeNeverCreatesHolding` 覆盖 `Prepare→Submit→Reconcile→RecoverAndSettle→ListHoldings`：Fill 被写但 `rest_verified=0`、Holding 总数 0 | **表述与永久证据修复；真实运行闭环 BLOCKED** |
| **S4-F2 P2** README 当前 Schema v24 / static 与旧二进制半发布 | README 中文/英文中所有「当前二进制要求」统一为 **v25**，保留历史 v23→v24 章节。新增 [成套发布/回滚 SOP](发布检查-02-S4-成套交付与回滚.md)，包括 DB 备份、停机、授权后的 `sync db`、完整二进制+static manifest、验收和失败时恢复；mode GET 新增 `contract_version=4`，Vue 发现旧后端则**禁用全部 S4 查询/配置并显示错误**，不盲目访问 4 个 GET 导致 404。重新构建 Vue，**107/107 dist→本地 static SHA256 相等** | **版本/前端拒绝半发布已修；真实成套生产发布仍 BLOCKED** |
| **S4-F3 P2** `live_approved/execution` 字面量可能漂移 | 新 `spotstrategy.LiveStatus(mode)` 统一结合 **`LiveOrdersApproved()`、数据库 mode 和 Fill 核验 Gate**，返回 `execution/live_approved/reason`；mode GET、mode PUT、配置列表/详情共用决策。新增 `TestS4F3LiveDisplayComesFromExecutionGate` 与 `TestS4ControllerModeContractAndGateCoupled` | **PASS，仍 fail-closed** |
| **S4-F4 P2** 只读 Holding 估值永远空 | `ListHoldings` 从本地 `spot_symbols` 读 **币对完全一致、quote 资产一致、合法精确价格、时间戳在 0～60 秒内**的参考价，透传 `reference_price_quote/time/fresh`。仅经过 `HoldingReport` **已核实且完整**的 Fill 才计算 `unrealized_estimate_quote`；超时/币种不符/无价格/Fill 未核实均为空。Vue 增本地参考价、时间和已核实估值列；明确不是 Binance 可执行报价。新增新鲜/超时/资产错配/未核实回归 | **本地参考估值 PASS；真实市场/实盘仍 BLOCKED** |
| **S4-F5 P3** 漏 `rows.Err()` 与 nil client panic | `ListHoldings` 和 `replayHoldingCost` 读完游标后均检查 `rows.Err()`；`ApplyVerifiedFills` 发现 `NewReadOnlyEngine(db,nil)` 的空 client 即返回 `ErrLiveWriteDisabled`，不再解引用。新增 `TestS4F5ReadOnlyEngineCannotSettle` | **PASS** |
| **S4-F6 P3** Fill symbol 筛选被忽略、非法 incomplete、缺正向 200、历史币不可编辑 | `ListOrderFills` 按订单真源 symbol 严格执行附加筛选，币种不符返回合法空分页；Holding DB status 只允许 `active/closed`（`incomplete` 是报表完整性）；配置列表只返回与支持 quote 相符的 USDT/USDC/FDUSD 币对；新增独立 SQLite + 合法 JWT → 4 路 GET Controller **200/JSON/DTO 字段**实证，且继续覆盖 401/400/404/503 安全语义；本地旧单业务未修改。全量既有红点明确披露 | **PASS；全仓历史测试红点未覆盖** |

## 三、测试与构建证据（本地真实执行，不是实盘）

| 项目 | 结果 |
| --- | --- |
| S4-F1 未核实运行链路不能创建 Holding | **PASS**，新永久测试 |
| S4-F3 统一门禁、F4 本地有效/过期价、F5 空 client、F6 不支持币对与筛选 | **PASS**，新增服务层测试 |
| S4 Controller JWT + 正向 200 + JSON + version contract | **PASS**，独立 SQLite，无生产库 |
| `go test -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./feature/strategy/line ./command` | **PASS 5/5** |
| `go test -count=1 ./controllers -run '^Test(S1SpotStrategy\|S4)'` | **PASS** |
| `go test -race -count=1`（上述 5 包） | **PASS 5/5**；macOS 链接器 `LC_DYSYMTAB` 非阻断 warning |
| `go vet`（上述 5 包 + controllers） | **PASS** |
| Go 主程序 `go build -o <临时目录>/go_binance_futures .` | **PASS**；仅临时文件，已清理 |
| 前端 `pnpm typecheck` / `pnpm build` | **PASS / PASS** |
| 前端 dist→后端本地 static、相对路径与 SHA-256 | **PASS，107/107 文件**，包括新 `spotStrategy` 页面；**本地同步不代表生产部署** |
| `git diff --check`、改动文件 Go 格式与本地文档链接 | **PASS** |

**全仓测试真实口径**：本轮没有再次运行 `go test ./...`，因此**没有新全仓 PASS 声明**。原 [S4 Review](CODE_REVIEW_02-S4.md) 在隔离副本中发现 controllers 五项旧测试红点及 `tests` 包缺失未纳入版本控制的 `conf/app.conf` 造成 panic（69 包通过但整体 EXIT=1）。本轮未修改这两类既有问题，不将它们误报为已解决。

## 四、E2E-01～E2E-11 的 R1 签收状态

| 场景 | 离线 / Fixture | 真正运行与浏览器 E2E |
| --- | --- | --- |
| E2E-01 默认 OFF | Gate 单测 **PASS** | 新生产部署 **SKIPPED** |
| E2E-02 BUY 10 成 4 撤 6 | 旧 SQL/Fake **Fixture PASS** | `rest_verified=1` 来源不可得：**BLOCKED** |
| E2E-03 SELL 4 部分 1+3 | 旧 SQL/Fake **Fixture PASS** | 真实归属 Holding/SELL：**BLOCKED** |
| E2E-04 同币新周期 | 旧 SQL/Fake **Fixture PASS** | 无真实新 Holding，**BLOCKED** |
| E2E-05 去重/分页/重启 | 离线各模块 **PASS** | 真实查单恢复/浏览器 **SKIPPED/BLOCKED** |
| E2E-06 旧 Spot 安全切换 | 拒绝模式切换 **PASS** | 旧单/人工资产对账 **BLOCKED** |
| E2E-07 main/auto_strategy | 作用域与 4 GET 200/拒绝用例 **PASS** | 真实账户/外部订单 **SKIPPED** |
| E2E-08 USDC/费用币种 | Fixture 计算/配置 **部分 PASS** | 真实交易 **BLOCKED** |
| E2E-09 stale/429/unknown/DB | 模拟异常与安全字段 **PASS** | 真实故障演练 **SKIPPED** |
| E2E-10 旧系统无回退 | 目标 Go 回归 **PASS** | 旧页面手工巡检 **SKIPPED** |
| E2E-11 多语言/UI/分页/static | Vue 编译、Controller DTO、107 文件哈希 **PASS** | 实际浏览器点击/截图 **SKIPPED** |

**总判断**：离线代码及只读页面基线可继续使用；**完整 S4 E2E、方案 02 生产 Gate 和依赖真实证据的方案 03 C0 仍未通过**。

## 五、操作纪律与剩余阻断

- 保留 S3-R1 的 `spotRealOrderReleaseApproved=false`、`canAttestLiveFills=false`，**未发送 Binance 真实订单/撤单请求**。
- Schema 仍为 `appversion.DatabaseSchemaVersion=25`，本轮不运行 `sync db`，不操作生产数据库、`conf/app.conf`、运行中交易进程或账户资产。
- 只更新**本地**前端 `dist` 和后端源码仓库 `static`；复核时未发现正在运行的 `go_binance_futures` 服务，不做程序部署/重启，不提交或推送 Git。
- 未决：持久化授权/撤销与事件审计、Binance 实际响应级成交来源证明、旧自动/手动仓位与外部挂单核验、生产 MySQL 8 结构和性能、成套二进制 + static + DB 发布/回滚、浏览器 E2E 截图。**每一项须在另行授权与可审计证据下独立验收。**


## 六、R1 后续补充：真实浏览器 + 隔离 UI/API 夹具（2026-11-10）

本次后续验收以**真实 Vue dist + macOS Chrome headless**重放 S4 页面，新增轻量零 npm 依赖脚本 `scripts/spot-strategy-browser-e2e.mjs`（位于前端仓库），命令 `pnpm test:s4-browser`，**EXIT=0**。结果：

| 浏览器场景 | 结果 |
| --- | --- |
| 配置、Holding、本地参考价和可信夹具估值可见 | **PASS（浏览器隔离 Fixture）** |
| 点击订单 tab，展开 Fill，展示 tradeId、BNB 费用资产 | **PASS（浏览器隔离 Fixture）** |
| 持仓周期 0 行时，展示真实来源未核实的空状态解释 | **PASS（浏览器隔离 Fixture）** |
| 旧后端 API version 3，前端要求 4，页面拒绝查询/编辑 | **PASS（浏览器隔离 Fixture）** |
| 英文 locale：现货策略标题与 Holding 页签渲染 | **PASS（浏览器隔离 Fixture）** |
| 65 个本机 HTTP 请求全部 GET；无 PUT/POST/DELETE | **PASS（本地隔离）** |

脚本无用户真实 JWT、真实 Go 服务、数据库或 Binance 请求。由于没有验证真实服务端 JWT/业务数据库、实际截图/人工巡检，也没有真实来源为 `rest_verified=1` 的订单，**原评估中的「真实浏览器/生产 E2E 未完成」仍然成立**。这是缩小 S4-R1 UI 交互验证缺口，不改变 F1 的业务闭环阻断或方案 03 C0 生产依赖。
