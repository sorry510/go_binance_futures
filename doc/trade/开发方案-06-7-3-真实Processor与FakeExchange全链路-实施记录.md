# 06-7-3 实施记录：真实 Processor + Fake Exchange 全链路

日期：2026-11-12  
分支：`feat/trade_spot`，实施前 HEAD `1845572 feat: 06-7-2`  
总方案：[06-7 现货量化自动策略执行链路](开发方案-06-7-现货量化自动策略执行链路.md)  
状态：**06-7-3 隔离 Fake 联调与 2026-11-13 首轮 Code Review 的 F1/F2 P2 整改已通过本地测试；严格 Spot 环境 E2E、MySQL8、Testnet 和生产 Gate 仍 BLOCKED。**

## 一、联调范围

本阶段不引入新的现货执行框架，直接串联现有生产业务代码：

```text
SpotQuantRunner.Run
  -> DBGate（数据库总开关 + 单币开关）
  -> SpotLedger.ListUnsettledOrders / Accounting（不可变 Fill 为准）
  -> line.SpotRuleEvaluator（实际 expr 表达式、无杠杆 ROI）
  -> spotexecution.Processor（真实 ExecuteBuy / ExecuteSell / Reconcile）
  -> spotexecution.Exchange（本次只有 Fake 实现，无任何 Binance 网络）
  -> SpotLedger（真实 SQLite 表、Order、Fill、Position）
  -> spotledger.NewSpotReadService（06-5 真实只读聚合）
```

- 使用 `feature/spot_quant_runner_0673_integration_test.go` 新建**临时 SQLite + Fake Spot Exchange**；不会访问用户服务器或真实 Binance。
- **执行器使用真正的 `spotexecution.NewProcessor`**，不是 06-7-2 的 `spotRunnerFakeExec`；总/单币开关由真实 `spotexecution.DBGate` 检查。
- 策略采用真实 `line.SpotRuleEvaluator` / expr / Spot ROI；其行情加载函数仅注入 Fake Spot 市价，**不宣称真实 Binance WS/REST、主网技术指标网络链路已验**。
- ReadModel 测试真实读取 BUY/SELL 订单、持仓 `CLOSED`、报价资产 PnL；模拟钱包有大量人工 BTC，SELL 只能按策略 Fill 剩余量发单。
- 不修改 Futures/旧 Spot 交易路径、AI 配置、前端菜单、数据库版本/DDL 或 `conf/app.conf`。

## 二、测试覆盖及实际结果

| 场景 | 结果 |
| --- | --- |
| **完整生命周期**：两个开关 OFF 无 POST；启用后真实 Processor BUY → Verified Fill → ReadModel → 策略 SELL → CLOSED → 新周期 | **PASS**；连续 20 轮不重单，手工钱包 50 BTC 也只卖策略持仓数量 |
| **部分成交**：首次 PARTIALLY_FILLED，原 clientID 多轮 REST 恢复；补齐第二笔 `myTrades` 后下一轮才允许 SELL | **PASS**；无第二次 BUY POST |
| **交易所已经接收，但客户端超时**：POST 返回 UNKNOWN，模拟重建 Runner/Processor，从持久化 clientID 只读查回 | **PASS**；重启恢复无重复 POST |
| **未被接受、权威查单 -2013**：本地意图保持 NEW，新的分钟/Kline 也不可重发 | **PASS**；必须显式人工确认，不自动释放 |
| **FILLED ACK 领先于 myTrades**：第一轮 `FILLED` 但成交数量为 0，随后真正成交迟到 | **PASS（新增代码修复）**；完整 Fill 记录后才允许 SELL |
| **第三币种 BNB 手续费** | **PASS**；真实 Fill 与仓位可继续平仓，但不假称完整净收益已可精确折算 |
| **基础币手续费产生不可交易 dust** | **PASS**；`ErrNeedsReview`，不会虚构 CLOSED 或挪用人工 BTC |
| **余额不足、全局关闭后的只读对账** | **PASS**；余额不足无下单；关闭开关停止 BUY/SELL，仍可显式 Reconcile |
| **5m、1h 时间框架的真实 Processor 防重**：多分钟轮询+策略文本编辑 | **PASS**；原周期不重单，新 Kline 才允许新的交易周期 |
| **WS/REST 重复成交**：同一 TradeID WS 重复传 3 次，再 REST 补充核对 | **PASS**；不可变 Fill 不会重复记录，最终仍可关闭仓位 |
| **动态修改主周期（1h→1m、1m→1h）** | **PASS（新增代码修复）**；旧/新 Kline 交界重复 BUY 被既有订单历史阻断；整点 1m 正常新周期不被误拦截 |

**专项目录测试**：`go test ./feature -run '^TestSpotQuant0673|^TestSpotQuant0672|^TestReview0672' -count=5 -p=2` —— **PASS**。其中新增 `TestSpotQuant0673*` 为 10 个顶层测试（包含各类子测试）。

## 三、联调发现并实际修复的两个问题

### 3.1 FILLED 状态可能领先成交明细入库

旧 `ListUnsettledOrders` 只查 `NEW/PARTIALLY_FILLED/PENDING_CANCEL`。实测当交易所返回 `FILLED`、但是 `myTrades` 暂为空时，Processor 会 fail-closed 返回 `ErrNeedsReview`，本地订单已为 `FILLED`、`filled_qty=0`。下一轮 Runner 无法再看到未结订单，导致**永远不会尝试导入迟到成交**。

现于 `service/spotledger/query.go` 增加对 `status='FILLED' AND filled_qty <> requested_qty` 的检查。只对**本地成交数量尚未完整核对的 FILLED 订单**执行只读 REST 恢复；补足后不再每轮查询老 FILLED。这不是重新下单，也不修改最终已核实成交的历史订单。

### 3.2 动态切换策略周期：v3 初版和复评后的 v4

**初版 06-7-3**：曾以单向 SHA256 v3 将币种、方向、周期和 Kline 开始时间合成 clientID；同根 Kline 可阻止重复信号，但跨周期回归依赖订单 `created_at`（Ledger 实际墙钟）与 `r.clock()`（可注入事件时钟）处于同一时间轴。原测试还人工 UPDATE DB 时间，因此 **CODE_REVIEW_06-7-3.md F1 P2 指出混钟漏单**。这不是现有最终设计。

**整改后的实际代码**：[CODE_REVIEW_06-7-3 整改复评](CODE_REVIEW_06-7-3-rereview.md) 将 ClientID 更新为 `sq4` v4 的可解析身份：**2 字符 Kline 周期 + 7 位 base36 事件 UTC OpenTime 秒 + 22 位 hash**，总长度仍为 34。BUY 前从该币现有策略 BUY 委托还原**原事件周期和时间**，比较原/当前 Kline 窗口是否相交，而**不再混用 `created_at` 与 `r.clock()`**。

- 新增双时钟永久回归：模拟事件时间在过去，DB 保留真实创建时间，不改写数据；1m→1h 的 BUY/SELL 不会重复，重启后仍不能绕过，新周期正常恢复。
- 对时钟回拨（数据库已有未来事件）主动返回 `ErrNeedsReview`，而不是忽略已提交订单重新 BUY。
- 原 v2/v3 不可逆哈希在本地若仍有 BUY 记录，按 `ErrNeedsReview` 停止该币自动新单并提示检查。**这不是历史订单迁移**，06-7-4 接入前必须检查真实本地记录；之前生产入口始终没有真实执行器。
- 仍不新增信号表、数据库字段、任务中心或复杂状态机。

### 3.3 FILLED 超成交须显式异常，不能静默空转

最初的 `ListUnsettledOrders` 增加 `FILLED AND filled_qty <> requested_qty` 是为了恢复迟到的 myTrades（`<`）。复评发现该条件同时包含 **`filled_qty > requested_qty`**，而原 Runner 对这种异常每次 Reconcile 都得到 nil、下一轮继续空转。现于 `reconcileUnsettled` 对超成交、非法数量直接返回 `spotledger.ErrNeedsReview` 并通过已有错误去重日志记录；`<` 继续执行只读补账，**不触发新的 Binance POST**。

新增永久回归：超成交状态下 3 次轮询均不下单、提示币种与数量，人工在**临时测试数据库**纠正测试数据后错误状态可恢复；不会自动篡改真实已成交历史。

### 3.4 已验收及未验收的实际边界

本轮严格区分三种测试：**真实 Processor + Fake Exchange + 注入的极简 Spot 价格环境**（已验），**`LoadVerifiedSpotRuleEnv` 严格 Spot ticker/Kline/24h OHLC/完整指标样本检查**（06-7-1 单测已验，但未通过本次真实 Processor 端到端测试），**真实 Binance Testnet/WS/REST**（未验）。严禁将其全部写成已验证。

新增真实 Processor 的 `BTCUSDT` / `WBTCUSDT` 双币隔离：两币独立 BUY，关闭 WBTC 开关后仅 BTC 自动 SELL，再开启后 WBTC 才 SELL；仓位和 Fill 不互串。

实际 3d Kline OpenTime 的交易所对齐、较大历史数据规模下 MySQL8 EXPLAIN/P95 与长期逐轮历史 Fill 投影仍留在 06-7-4。v4 不再用无索引 created_at 时间窗口 COUNT，但会按币种枚举历史 BUY ClientID，**性能尚未实测**。

**06-7-3 首轮 Code Review 整改（2026-11-13）**：原始 [CODE_REVIEW_06-7-3.md](CODE_REVIEW_06-7-3.md)；逐项修复及剩余 Gate 见 [整改复评](CODE_REVIEW_06-7-3-rereview.md)。新增 `TestReview0673*` 专项回归；F1/F2 已代码修复、F6 双币补测，F3/F4/F5/F7 的验证和放行边界明确保留。

## 四、总体质量门禁

| Gate | 实际结果 |
| --- | --- |
| 应用测试：`./scripts/test-go.sh -short` | **73 个含测试包 PASS，0 FAIL** |
| 10 个相关包 `go test -race` | **10/10 PASS** |
| 06-7-2 + 06-7-3 专项重复 5 次 | **PASS** |
| 10 包 `go vet` | **PASS** |
| `go build -o /tmp/spot0673_main_build .` | **PASS**；未覆盖任何正在运行的程序 |
| `git diff --check` | **PASS** |
| 干净 `git archive HEAD` + 本次改动覆盖的隔离副本 `go test ./... -count=1` | **73 包 PASS、13 包无测试、0 FAIL**（与用户工作区可能含私有研究归档的全仓测试不同） |

## 五、仍未解除的限制与后续 06-7-4

1. **主程序真正自动交易仍为只读禁用**：`feature.StartSpotQuantTrade(ctx)` 继续在无授权真实执行器时立即返回；开启 Spot 业务开关**仍不会向真实 Binance 发送订单**。
2. **Testnet 凭据 / REST / WS**：此前 `-2015` 未解决；本轮 Fake Exchange 不验证真实 Binance 延迟、WS 重连、真实余额变动或 Testnet 成交价格。
3. **物理 MySQL 8 与发布检查**：需要在后续授权阶段验证现有 Kline/订单事件防重 SQL 的 EXPLAIN 和真实历史体量；当前为个人账户简化设计，但未声明 P95 已达标。
4. **手续费/dust**：基础币佣金造成不能整手卖出的数量时，系统继续 fail-closed 并等待人工处理，**不是自动抹去 dust 或认领人工币**；属于已有的真实交易验收约束。
5. **未做操作**：未发真实订单、未启动服务、未改生产/测试数据库、未执行 `sync db`、未修改 `app.conf`、前端或 `static`，未自动 commit/push。

**06-7-3 结论：真实 Processor + Fake Exchange + 真正 Spot Fill/ReadModel 的隔离端到端链路通过。下一阶段 06-7-4 要先取得独立有效 Testnet 授权，再按测试账户→MySQL 8/发布核查顺序逐项验证；目前不能宣称现货量化实盘可用。**
