# CODE_REVIEW_06-ALL：整改复评

日期：2026-11-12  
原始审查：[CODE_REVIEW_06-ALL.md](CODE_REVIEW_06-ALL.md)（原文不修改）  
基线：后端 `feat/trade_spot` HEAD `35da095`，前端 `feat/trade_spot` HEAD `89f15bdd2034121dd57f3e0184367a7c62ae25a0`（前端存在未提交修改）

**复评结论：F2 / F3 可立即修复的 P2 问题已完成代码整改并通过干净 Git 副本测试；F4 仅固定临时工作区哈希、未完成可发布的 Git SHA 组合；F1 / F5 / F6 的真实执行能力和门禁仍 BLOCKED；F7 / F8 / F9 未把离线逻辑测试伪装成生产验收。方案 06 不能标记为「整体实盘完成」。**

## 一、逐项问题处理

| ID | 严重度 | 状态 | 实际处理与保留边界 |
| --- | --- | --- | --- |
| **F1** | P1 | **BLOCKED（有意安全门禁）** | 未将 Spot Quant `Processor`、自动 LONG/CLOSE_LONG 信号循环或 Testnet User Data WS 自动注册到 `main.go`；当前执行能力只由 Fake 测试驱动。Spot Testnet 签名账户仍 -2015 且真实交易未授权，**不为消除审计发现而擅自启用主网/自动单**。 |
| **F2** | P2 | **已修复** | `README.md`、`README.EN.md` 共 6 处「当前二进制/Lead 所需 Schema」现在时表述由 v24 改为 **v25**，以 `appversion.DatabaseSchemaVersion` 为唯一代码真源。保留 v24 历史迁移章节；额外注明生产需要独立的 v25 **物理表结构只读预检**，不能只看版本号。 |
| **F3** | P2 | **已修复并完成干净检出证明** | `tests/default_test.go` 不再无条件调用硬编码 `conf/app.conf` 的 `TestBeegoInit`；当私人配置缺失时显式加载已经纳入 Git 的 `conf/app.conf.example`。真实本地私有配置存在时仍继续使用它，**未修改或提交凭据**。`controllers/strategy_template_static_test.go` 始终验证 `controllers/testdata/strategy_templates/ci-kline-long.json` 最小 LONG/指标 fixture；仅当本地私人模板目录存在时额外验证，并跳过其 `research/` 归档。**在纯 HEAD git archive + 本轮 3 个测试改动/fixture 的临时干净副本中，不存在 `conf/app.conf` 或任何个人模板：`./scripts/test-go.sh -short` 成功，且 `go test ./... -count=1` 73 包 PASS、13 包无测试文件、0 FAIL，退出码 0。**这是对本轮尚未提交修复的验证，不等于未修改的历史 HEAD 已经自带修复。 |
| **F4** | P2 | **暂未关闭** | 受用户「不得自动 commit/push、不得回退用户未提交修改」约束，没有创建或修改任何前端 commit。补充 [前端工作区源码 SHA-256 清单](CODE_REVIEW_06-ALL-前端工作区源码SHA256清单.txt)，记录前端 HEAD 和所有 14 个实际未提交文件的路径/哈希，可与 06-6 的 115 个 `dist/static` 静态哈希交叉核对。**这只是临时本地快照，并不等于可由固定 Git SHA 重建的发布组合。** 正式发布前必须由用户决定提交与版本固定。 |
| **F5** | P2 | **BLOCKED（需独立 Testnet 联调）** | `Reconcile` / `ConfirmNeverAccepted` 仍只有测试调用的 Go 方法，没有安全的运行时人工入口。现有 `GET /spot-quant/positions/:id` 可显示原始 Order/Fill 与复核异常，本地 `rebuild` **不能**查询交易所。新增 [异常订单恢复操作边界](方案06-现货异常订单恢复操作边界.md)，说明 30 分钟/两次权威查无订单的条件、禁止直接改数据库和未来最小化安全入口的准入标准；**本次未引入主网读写 API、未开放卖出/撤单按钮，也不把运行手册当作已修复 F5。** |
| **F6** | P1 | **BLOCKED** | Testnet `-2015`、真实 Testnet 成交/WS、生产 Schema 预检/备份、MySQL 8、生产部署与主网用户授权全部缺失。没有重放这些真实动作。 |
| **F7** | P3 | **性能待验证** | 维持 `maxSummaryPositions=1000` 与跨页准确账本重放，未在空表 MySQL 5.6 上伪报 MySQL 8 P95，也未仓促改变精确成本/手续费逻辑。 |
| **F8** | P3 | **真实 dust 处置待验证** | 维持 LOT_SIZE 不足时 fail-closed，不出售钱包里的人工库存，也不伪造 `CLOSED`。见新增操作边界文档。 |
| **F9** | P3 | **发布流程 Gate BLOCKED** | 遵守此前用户明确决定：不在 `main.go` 加回 v25 启动态物理 Schema 校验；`VerifyV25SpotQuantSchema` 仍只通过显式 `sync db` 校验。生产库上线必须另外做只读预检与备份，不将流程门禁描述为代码已自动执行。 |

## 二、可独立复现的测试

- **当前工作区针对性测试**：`go test ./tests ./controllers -run '^(TestLoginRejectsWrongCredentials|TestUnknownRouteReturnsNotFound|TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals|TestStrategyTemplateImportRequiresEnabledKlineIndicators)$' -count=1` —— PASS。
- **干净 Git 检出**：创建隔离 `git archive HEAD` 临时副本 `/tmp/spot06all_clean_ci.KZ2x8m`；只叠加 `tests/default_test.go`、`controllers/strategy_template_static_test.go` 和纳入版本控制的最小测试 fixture；没有复制或生成 `conf/app.conf`、私人策略模板、生产数据库连接或私有密钥。
- 该临时副本 `./scripts/test-go.sh -short` —— **73 个含测试的包通过、0 FAIL**。
- 该临时副本 `go test ./... -count=1` —— **PASS；73 个含测试的包、13 个无测试文件的包、0 FAIL**。这次 `./...` 包含当前纯 HEAD 归档中的应用代码，不需要私人配置；与旧审查报告中的纯旧 HEAD 失败不是同一次代码状态。
- README 修正针对性 diff check —— PASS。
- **本轮最终验证**：`go test -race ./command ./controllers ./models ./service/spotquant ./service/spotledger ./service/spotexecution ./spot/api/binance ./tests -count=1` **8/8 PASS**；`go vet` 对上述 8 包 **PASS**；`go build -o /tmp/spot06all_rereview_main .` **PASS**；`git diff --check` **PASS**。macOS `LC_DYSYMTAB` 属既有非致命链接提示。**没有重复 `pnpm build`，没有修改前端源码、后端 `static`、数据库或主程序运行态。**

## 三、仍需实际执行的发布检查

1. 用户决定并**自行授权前端 Git 版本固定**，确保已修改/未跟踪的 14 个实际源码文件与构建产物固定关联。
2. 在单独的 Spot Testnet 账户上排除签名 `-2015`；对真实订单数据执行小额 BUY/SELL、部分成交、手续费/dust 和断线恢复，先有明确环境与下单许可。
3. 在独立的 MySQL 8 测试库验证 v25 物理 Schema、`EXPLAIN`、有代表性持仓/订单/Fill 的 P95。当前本地 `go_bn_test` 是 MySQL 5.6、Spot managed 表为空。
4. 确认 F5 **仅人工触发**、testnet 先行的恢复入口与权限边界，保留 `client_order_id` 幂等；网络异常绝不当作权威 -2013 未找到。
5. 对生产库执行**授权的只读** v25 表/字段/索引预检并记录备份方案；如需实际数据库升级，仍只能使用项目的 `./go_binance_futures sync db` 且另行获得用户明确授权。
6. 经过独立 Code Review、实际 Testnet 与部署门禁后再考虑运行时自动 LONG/CLOSE_LONG 信号接线、主网准入、二进制/static/Schema 成套发布与回滚演练。

**总结：修复了可以不触碰真实交易/生产环境解决的版本和 CI 问题；其他风险不是改 README 或多跑 Mock 就能解除的 Gate，继续明确保留为未完成。**

本轮不修改 `conf/app.conf`、`main.go`、任何数据库连接、交易路由、前端用户未提交改动、`static/` 或运行中的服务；不执行 `sync db`，不创建真实订单，不进行自动 Git commit/push。
