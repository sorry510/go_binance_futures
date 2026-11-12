# CODE_REVIEW_06-7-2 整改复评：Spot 自动轮询与执行接线

日期：2026-11-12  
分支：`feat/trade_spot`，基线 HEAD `5c1d4fa`；所有 06-7-2 与本次整改仍为**未提交的本地工作区**。  
原始审查：[CODE_REVIEW_06-7-2.md](CODE_REVIEW_06-7-2.md)（保留原文，不覆盖）  
实施记录：[06-7-2 现货自动轮询与执行接线](开发方案-06-7-2-现货自动轮询与执行接线-实施记录.md)

**总评：F2/F3/F4/F6/F7 的代码缺陷已整改并通过离线回归，F8 增加默认 ROI 阈值提示；F5 校正实例/调度防重入责任；F1 仍是有意的 P1 真实交易发布阻塞，不把 `exec=nil` 的两秒调度虚报为实盘自动交易。06-7-3 的 Fake Processor 集成可以继续，06-7-4 的 Testnet/主网及生产发布不可放行。**

## 一、Findings 逐项处理

| ID | 等级 | 复评状态 | 证据与实际处理 |
| --- | --- | --- | --- |
| **F1** 运行时 `exec=nil` 空转，五项业务完成标准不可达 | P1 | **继续 BLOCKED，明确边界** | 生产入口 `feature.StartSpotQuantTrade` **立即返回，不初始化 ORM、不读取行情、更不创建本地 NEW 或真实 Binance 订单**。`main.go` 保留与 Futures 相同的 2 秒 `loopRun` 骨架；仅测试注入 Fake Executor 走完整 BUY/SELL 路径。增加编译期断言，06-4 `spotexecution.Processor` 与 Runner 接口匹配；实际注入必须留在单独 Testnet/发布验收后，现有两个业务开关不是授权凭据。**这不是“可以自动实盘”**。 |
| **F2** 固定 1 分钟 bucket，5m/1h 同一 K 线可重复交易 | P2 | **已修复本阶段可验证范围** | 新增 `spotSignalInterval`：从**已启用技术指标的 `kline_interval`** 和规则显式 `kline_* `引用中选择最长时间框架，默认为项目 1m；`spotCandleOpen` 按 UTC 周期边界生成事件时间，1w 以 UTC 周一、1M 以月初为边界；用**币种 + 方向 + SELL position_id + 事件时间**生成稳定的 34 字符 Binance ClientID，不包含可编辑 `RuleHash`。测试实际推进 5m/1h 中间分钟并编辑策略文本，BUY/SELL 各一次，同一根 K 线不会重进；新 K 线才允许下一周期。**说明**：事件起点按 UTC 周期推导，不宣称直接读取了交易所每根实际 Kline 的 OpenTime；跨配置切换主时间框架的策略变更属于新信号配置，仍需后续 Fake Processor 联调复核。 |
| **F3** 信任物化 `Position.status`，篡改 CLOSED 后跳过订单恢复 | P2 | **已修复** | `service/spotledger/query.go` 新增按 symbol 精确查询的 `ListUnsettledOrders`，不依赖父仓位缓存，只针对 NEW/PARTIALLY_FILLED/PENDING_CANCEL；每轮**先恢复所有未结订单，再进入信号评估**。`activeSpotPosition` 按页枚举本策略历史周期并逐个 `Ledger.Accounting` 从真实 Fill 重新投影：被改成 CLOSED 的 OPEN/PENDING_BUY 仍会正确识别；账本不一致则 fail-closed。**不增加数据库表或修改旧 Fill。** 性能以个人单账户规模为前提，历史订单大量积累时的 MySQL 8 性能留待真实验收。 |
| **F4** 空转路径先初始化 ORM；回调 panic 影响全进程 | P2 | **已修复** | 生产 `StartSpotQuantTrade` 的 nil 执行器分支在任何配置或 DB 操作前返回；Spot 专用 `feature.RunSpotQuantProtected` 只隔离这条两秒回调的 panic，不触碰现有 Futures `loopRun`。增加不需要私人 `app.conf` 的可重复测试，模拟 Spot panic 后另一个回调仍可执行。初版曾将回调测试放在 root，导致无配置的干净副本执行 `main.go` 初始化而失败；已**移至 `feature` 包并移除 root 测试文件**，保留生产 DB 初始化逻辑不变，最新干净副本全仓测试通过。 |
| **F5** 每轮新建 Runner 导致实例级 `atomic.Bool` 无效 | P3 | **职责澄清，生产重建已移除** | `StartSpotQuantTrade` 不再每两秒构造一个新 Runner。主程序持续轮询唯一生效的防重入由原有 `loopRun` CAS 负责；`SpotQuantRunner.running` 仅保护**同一注入实例**被并发调用，专项测试实际验证这一点。后续授权真实执行器时应复用持久 Runner 实例，不可把每轮新实例的 `atomic.Bool` 当作跨轮防重。 |
| **F6** 每两秒重复记录相同异常 | P3 | **已修复本地逻辑** | Runner 逐币记录上次错误文本，只在**错误内容变更或恢复**时输出；`feature.ReportSpotQuantScheduler` 保留进程级最近摘要，仅不同错误状态首次记录，杜绝同一个错误在每两秒产生双重重复日志。专项测试覆盖同一错误重复 12 次及状态恢复。授权后 Runner 应为持久实例以保留逐币状态；这里不引入复杂任务中心、指数退避或第三方报警。 |
| **F7** 一轮只对账第一笔未结订单 | P3 | **已修复** | 以 `ListUnsettledOrders` 查出的**全部**待恢复订单为源，顺序 `Reconcile`；若有错误则汇总，仍尝试其他订单，任何一次恢复所在轮都**不触发新 BUY/SELL**，等待下轮 Fill 投影。新增两笔孤立未结订单的临时 SQLite 回归。 |
| **F8** 默认 `profit=100/loss=100` 无运行提示 | P3 | **已添加提示（不改默认值）** | 注入执行器并检测到启用币种同时保留默认 100/100 时，针对该币首次记录警告：这是 **+100% 止盈/-100% 止损**，`close_long=true` 不是立即卖出；不修改历史配置、不另增审批开关。生产 `exec=nil` 不扫描币种，不会凭空弹出提示或开启订单。 |

## 二、关键回归证据

| 验证 | 结果 |
| --- | --- |
| 06-7-2 原 8 项 Fake Runner 测试及 Review 新增用例 | **PASS** |
| 新增 `TestReview0672KlineIdentitySurvivesMinuteRollAndRuleEdit` | **PASS**：5m/1h 逐分钟推进，策略编辑后同根 K 线不重复；下一根才新开 |
| 新增 `TestReview0672TamperedCLOSEDCacheStillReconcilesPendingFill` / `TestReview0672TamperedClosedCacheStillClosesVerifiedOpenPosition` | **PASS**：物化 CLOSED 不得替代真实 Fill |
| 新增 `TestReview0672ReconcileALLPendingEvenWithoutActivePosition` | **PASS**：两笔孤儿未结订单均优先恢复，无新 BUY |
| 新增 Spot callback panic / 错误状态去重 / 默认阈值提示回归 | **PASS** |
| `go test ./feature -run 'TestReview0672\|TestSpotQuant0672' -count=5 -p=2` | **PASS** |
| `./scripts/test-go.sh -short`（应用包；不包括个人研究目录） | **73 个有测试的包 PASS，0 FAIL** |
| `go test -race` 10 个相关包 | **10/10 PASS** |
| `go vet` 10 包，`go build -o /tmp/spot0672_review_final_build .`，`git diff --check` | **PASS** |
| **干净 `git archive HEAD` + 本轮 Go 改动**，无私人 `conf/app.conf`、无 `strategy_templates/research`，`go test ./... -count=1` | **73 个测试包 PASS、13 个无测试文件、0 FAIL，退出码 0**；独立目录 `/tmp/spot0672_review_final_clean.BfKTJY` |

不能将隔离副本的 73 包 PASS 误称为用户含研究归档的工作区 `go test ./...` 全仓 PASS。Go 运行过程中原有非致命 macOS 链接提示不影响退出码。

## 三、仍未解决的上线问题

1. **F1 P1 运行时真实自动交易仍不可达：** 业务总/单币开关开启后，主程序虽然每 2 秒调用 Spot Runner 入口，但生产路径无真实执行器，**不会自动 BUY/SELL**。06-7-3 只在 Fake Exchange 下完成真实 Processor 端到端，不能把测试环境直接连接到用户 Mainnet。
2. **06-7-3**：把真实 `spotexecution.Processor` + 独立 Fake Exchange 注入持久 Runner，验证迟到成交、部分 Fill、UNKNOWN/重复、重启和 Dust/手续费；不同时间框架之间的策略切换须再做幂等验证。
3. **06-7-4**：Spot Testnet `-2015` 授权、真实 Testnet BUY/SELL、WS/REST 恢复、MySQL 8 物理 Schema/P95、主网订单许可、发布和回滚仍需**单独授权和实际证据**。未修改 `conf/app.conf`、生产数据库或真实交易凭据。
4. **性能边界**：为了防止伪造缓存状态绕过账本，Runner 目前每轮对启用币种的历史 Spot Managed Position 做完整 Fill 投影。这在个人使用、小规模历史下取安全优先；如果长期历史积累很大，需要在 MySQL 8 Gate 评估成本，不能未经审核以缓存状态代替金融权威。

**复评结论：06-7-2 的 Fake/隔离自动轮询逻辑与 Code Review 发现的正确性问题已整改，可继续 06-7-3；实际自动实盘、Testnet 与生产发布仍 BLOCKED，不得声称整体交付完成。**

本轮仅修改相关 Runner、Spot Ledger 查询、Spot 专用回调保护、测试和阶段文档；原始审查不覆盖，不修改用户研究目录、数据库 Schema、`conf/app.conf`、前端 `static/`；没有真实下单、没有自动 commit/push。
