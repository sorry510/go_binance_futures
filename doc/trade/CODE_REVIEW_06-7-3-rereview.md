# CODE_REVIEW_06-7-3：整改复评

复评日期：2026-11-13  
分支：`feat/trade_spot`；本次开始时 HEAD `1845572 feat: 06-7-2`；全部 06-7-3 改动仍为**未提交的本地工作区**。  
原始报告：[CODE_REVIEW_06-7-3.md](CODE_REVIEW_06-7-3.md)（保留、不覆盖）  
阶段记录：[开发方案-06-7-3-真实Processor与FakeExchange全链路-实施记录.md](开发方案-06-7-3-真实Processor与FakeExchange全链路-实施记录.md)

**结论：原审查 F1/F2 两个 P2 已通过代码与永久回归整改。F6 双币真实 Processor 测试已补齐；F3 严格行情环境、F4 实际 3d Kline 起点、F5 MySQL8 执行计划与 F7 旧版订单身份均保持明确的发布边界。Fake 端到端 Gate PASS，06-7-4 Testnet/Mainnet/生产 Gate 继续 BLOCKED。**

## 一、逐项复评

| Finding | 原等级 | 处理状态 | 代码与证据 |
| --- | --- | --- | --- |
| **F1** 客户端 Kline 事件时钟与 Ledger `created_at` 墙钟混用 | **P2** | **已修复** | 旧 SQL 的 `created_at >= simulatedEventStart AND created_at <= simulatedNow` 已从跨周期判断完全删除。新的 `sq4` 34 字符 ClientID 可还原**事件周期和 UTC 开始时间**，从 `spot_managed_orders` 已存在的策略 BUY 订单解码后比较 Kline 事件区间，不依赖物化时间列。旧测试中人为 `UPDATE created_at` 的代码已删除；新 `TestReview0673SeparatedLedgerClockCannotRepeatHistoricalSignal` 故意让真实 DB 时间晚于模拟事件时钟，在不改写 DB 时间、重启 Runner 后仍不重复 BUY，下一真实事件边界才放行；`TestReview0673ClockRollbackCannotBypassExistingBUY` 对事件时钟回拨返回 `ErrNeedsReview`。 |
| **F2** `FILLED` 超成交无限恢复、错误却静默清除 | **P2** | **已修复** | `ListUnsettledOrders` 继续包含 `FILLED AND filled_qty <> requested_qty`，因为 `<` 必须补迟到 myTrades；`reconcileUnsettled` 现在对 `>` / 无效数量明确返回 `spotledger.ErrNeedsReview` 并记录币种/订单号/超额值，不继续无意义的正常 Reconcile。原有按错误文本去重仍生效。`TestReview0673OverfilledFILLEDRaisesReviewInsteadOfSilentRetry` 连续执行 3 轮：每轮 fail-closed、仅第一次错误状态报警，无重复 POST；状态修复后恢复。 |
| **F3** 真实 Processor 集成中替换了严格 Spot 行情环境 | P2 | **明确未覆盖，06-7-4 Gate 保留** | 06-7-3 使用真实 Expr/`SpotRuleEvaluator`/无杠杆 ROI，但 `LoadEnvironment` 是注入的 `NowPrice/NowTime` Fake，**不是** `LoadVerifiedSpotRuleEnv`。严格本地 Spot ticker 5min、24h OHLC、`kline_*` 历史质量和指标样本完整性只在 06-7-1 专项单元测试覆盖；**未声称 Fake Processor E2E 覆盖这些项目**。06-7-4 要补 Fake 严格环境的端到端，及有授权后的真实行情验证；此前不解锁订单执行器。 |
| **F4** UTC 推断 3d Kline OpenTime 与 Binance 实际对齐未经核对 | P3 | **待 Testnet/官方行情验证** | 1m/5m/1h/1w/1M 等当前离线行为已测，`3d` 仍使用 UTC 72h 截断，**不声称与 Binance 实际 3d Kline OpenTime 一致**；首次接入真实市场数据必须核验并按其实际 OpenTime 调整，禁止据此默认启用 3d 信号实盘。 |
| **F5** 每轮历史 Fill 投影 + 既有订单查询的 MySQL8 性能 | P3 | **待 MySQL8 Gate** | F1 删除了无索引的 `created_at` 时间范围 COUNT，改为按币种读取现有 BUY 订单并在应用侧解码比较事件；避免双时钟，但历史订单规模扩大时 O(历史 BUY 数)。`activeSpotPosition` 仍逐个重建历史 Fill。没有声称 EXPLAIN/P95 通过；需后续真实库规模、索引及历史数据量测试，未经审核不改生产 Schema 或用可修改缓存取代 Fill 权威。 |
| **F6** 缺少真实 Processor 的 BTC/WBTC 精确隔离 | P3 | **已补测试** | Fake Exchange 现在区分 `BTCUSDT` / `WBTCUSDT` 的规则、订单、成交和可用 BTC/WBTC；`TestReview0673TwoSymbolsThroughRealProcessorStayExactAndIndependent` 用真实 Processor 同时 BUY 两币，关闭 WBTC 单币开关后仅 BTC SELL，重新开启后 WBTC 才 SELL。双方 Fill/持仓不互串。 |
| **F7** v2/v3 ClientID 不再兼容新订单去重身份 | P3 | **明确保守拒绝** | `sq4` 保存事件时间框架和事件起点，无法从已存在的 v2/v3 单向哈希还原 Kline。若数据库存在不可解析的历史策略 BUY 订单，Runner 现在明确返回 `ErrNeedsReview`、禁止新 BUY 并记录原因，**不再只靠不可靠的 `created_at` 窗口猜测**。`TestReview0673LegacyHashedClientIDsFailClosedBeforeAutomaticBUY` 覆盖。此前生产 `StartSpotQuantTrade(ctx)` 始终空转，未产生任何真实 v2/v3 自动订单；首次开放 Testnet 前需确认本地既有订单数据。 |

## 二、ClientOrderID v4 说明（无新表）

字符串结构为 `sq4` + **2 位 Kline 周期码** + **7 位 base36 UTC OpenTime 秒** + **22 位 SHA256 片段**，共 34 字符；哈希绑定 symbol、BUY/SELL、SELL position_id、Kline 周期与 UTC OpenTime，不使用用户可编辑的 RuleHash。

- 相同币种/方向/周期/事件返回同一个 clientID；不同周期即使整点相同也不冲突。
- 发生周期切换时从本地已有 BUY 委托取出原 Kline 事件区间，与当前事件区间比较重叠；不再混用 Ledger 的真实 `created_at` 与 Runner 的模拟时钟。
- 为安全起见，无法解析的旧版 BUY 委托不自动重发；用户需先进行可复核的历史订单检查，不能静默忽略。
- 不新增专属信号表、任务中心、Owner、审批、配置开关或数据库迁移；仍是个人账户简化设计。

## 三、本轮实际执行的质量验证

| 验证 | 结果 |
| --- | --- |
| `TestReview0673*` 新增隔离用例 | **PASS**：双时钟、时钟回拨、超成交告警、v2/v3 阻断、BTC/WBTC 双币 Processor |
| 原 `TestSpotQuant0673*` 10 个顶层端到端测试 + 06-7-2 旧专项 | **PASS** |
| `go test ./feature -run 'TestReview0673\|TestSpotQuant0673\|TestReview0672\|TestSpotQuant0672' -count=5 -p=2` | **PASS** |
| `./scripts/test-go.sh -short`（应用包） | **73 个测试包 PASS、13 无测试、0 FAIL** |
| `go test -race` 10 个相关包 | **10/10 PASS** |
| `go vet` 10 包、`go build -o /tmp/spot0673_rereview_build .`、`git diff --check` | **PASS** |
| 隔离 `git archive HEAD` + 本轮 Go 文件，`go test ./... -count=1` | **73 测试包 PASS、13 无测试文件、0 FAIL**（`/tmp/spot0673_rereview_clean.ZDJypb`） |

隔离副本没有私人 `conf/app.conf` 和研究归档文件，不等同于声称当前开发工作区 `go test ./...` 全仓通过。所有 Fake 测试均无真实 Binance API 访问。

## 四、仍然 BLOCKED 的发布事项

1. **P1 自动实盘接线**：生产 `StartSpotQuantTrade(ctx)` 仍仅返回 `ctx.Err()`；两个业务开关打开**不会真实下单**，不因本次 Fake 全绿而注入真实 Processor。
2. **06-7-4**：Spot Testnet 凭据（历史 `-2015`）、真实 REST/WS 和行情严格环境、实际 Kline OpenTime（含 3d）、BNB/dust/余额/异常订单恢复权限、MySQL 8 大历史量 EXPLAIN/P95、生产 Schema v25、发布/回滚证据仍未完成。
3. 历史 v2/v3 / 非可解析旧 BUY ClientID 在真实自动交易启用前需人工复核。运行时不会通过猜测旧订单事件周期冒险发新单。
4. 不应因静默告警已修复就把人工恢复 UI/CLI 视为已交付；06-ALL 异常订单恢复入口仍是独立未关闭项。

**最终判定：06-7-3 的 P2 代码问题已整改，Fake 联调可作为进入 06-7-4 前的技术基线；真实 Testnet、Mainnet、生产数据与发布门禁仍未验收。**

本次未修改 `main.go`、`conf/app.conf`、数据库连接/Schema、前端、`static` 或生产环境；未执行 `sync db`、启动运行服务、发送真实订单或自动 commit/push。原始 `CODE_REVIEW_06-7-3.md` 保持原样。
