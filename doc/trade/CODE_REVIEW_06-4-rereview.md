# CODE_REVIEW_06-4：整改复评

日期：2026-11-11  
分支：`feat/trade_spot`；开发基线 HEAD `6d8eecc`（06-4 仍在未提交工作区）  
原审查：[CODE_REVIEW_06-4.md](CODE_REVIEW_06-4.md)（原文保留）  
阶段结论：**F1/F2 主要业务阻塞与 F3～F8 代码级整改已完成本地 Fake 验证；Testnet 实际签名 REST/WS、主网、MySQL 8、部署验收仍未通过。**

## 一、逐项整改

| Finding | 处理状态与依据 |
| --- | --- |
| **F1 P1：SELL 永远留 0.5% dust** | **正常可交易数量闭环已修复。** `rules.go::planSell` 不再乘以 0.995，发出精确且符合 MARKET_LOT_SIZE 的策略剩余库存数量；`submitOnce` 发送前二次核验全量库存与钱包 free。现货手续费由 06-3 Fill 逐笔按实际资产扣除，不在下单前二次虚拟扣费。新增 `TestReview064FullSellClosesCycleAndNewBuy` 断言 SELL 之后 `CLOSED + remaining=0`，然后新 BUY 周期正常创建。**边界**：若 BUY 基础资产手续费使剩余数量不满足 step/min，或 SELL 真的额外收取 base 手续费超过已核实库存，返回 `ErrNeedsReview`，不能假装 CLOSED、不能伪造成交。此类 dust 的真实处理仍需 06-5 页面提示与 06-6 实测/人工处置规范。 |
| **F2 P1：权威未找到订单却永久 NEW** | **新增显式确认恢复入口。** 本地已有 client ID 的再次 `Reconcile`，收到交易所权威 -2013 时返回可识别 `ErrNotFound`（网络异常仍返回 `ErrUncertain`），**从不自动 POST 第二次**。新增 `Processor.ConfirmNeverAccepted(ctx, orderID, operatorConfirmed)`：要求明确人工确认、原委托 NEW 且无 exchange_order_id、距创建至少 30 分钟、两次远端权威查询均不存在；事务内 `Ledger.ConfirmNeverAccepted` 检查无 Fill/无其他委托，标记原订单 REJECTED、空仓位 CLOSED。绝不复用旧 client ID；若之后出现迟到 Fill，阻止重新打开旧仓位并报 `ErrNeedsReview`。有永久测试。**入口仅为 Go 方法，06-5 页面尚未接入；对于交易所实际接受的订单不能人工标记为不存在。** |
| **F3 P2：Testnet REST 但 WS 指向主网** | **代码级安全隔离已修复。** `spot_quant_stream.go` 不再调用会读取包级 `sdk.UseTestnet` 的 `WsUserDataServeSignature`；改为显式 `wss://ws-api.testnet.binance.vision/ws-api/v3`，复用现有 proxyPool WebSocketDialer 与 SDK 的签名请求构造。只允许 Testnet REST URL 精确匹配，SDK 全局变量不变。且 `Place/Cancel` 在实际请求时重新检查 SDK Client 可变 BaseURL，防止构造后被改到主网。订阅响应、封装事件的 Fake/离线解析测试已通过。**真实 Testnet WS 握手/订阅仍未验证，主网不可用。** |
| **F4 P2：FILLED 先于 myTrades 触发 WS 断开** | **已修复。** `applySpotQuantOrderEvent` 对 `ErrNeedsReview` 尝试 `ReconcileByClientID` 只读 REST 对账，失败则记录日志但不停止正常 WS；后续自动订单仍须通过账本的 `ReviewPosition` 门控。其他非预期错误仍使连接停止并要求显式恢复。新增 `TestReview064WSFilledBeforeMyTradesKeepsStream`。 |
| **F5 P3：总开关 OFF 后先查 REST** | **已修复。** `ExecuteSell` 先从本地仓位取得 symbol，校验双开关，再对账；`TestReview064GateOffBeforeAnySellREST` 验证 OFF 时 REST 查单数增长为 0；已有订单仍可 `Cancel/Reconcile`。 |
| **F6 P3：重扫老委托导致 REST 放大** | **已修复为有限核对。** `needsRemoteCheck` 总是核对 NEW/PARTIALLY_FILLED/PENDING_CANCEL、FILLED 但 myTrades 不全、最近 24 小时 CANCELED/EXPIRED；已有完整 Fill 的 FILLED、旧已终态记录不再反复访问远端。重启恢复先快照活动仓位 ID 后再循环，避免页面数据变动导致 OFFSET 跳页。新增 `TestReview064OldFullyReconciledBUYDoesNotBlockSELL`、`TestReview064RemoteCheckScope`。**旧终态窗口后发生的极晚成交仍要通过 WS 事件或人工 REST 指定委托核对，不把历史订单长期可查当作保证。** |
| **F7 P3：WS 错误判断与手续费资产不完整** | **已修复。** `errors.Is(err, orm.ErrNoRows)`；正数手续费必须明确 FeeAsset；字符串 `0.00000000` 等零手续费无需伪造币种；`TestReview064MissingFeeAssetFailsClosedAndZeroFeeWorks` 通过。 |
| **F8 P3：文档历史口径不符** | **已同步。** 06-3 复评属于 HEAD `6d8eecc` 已提交范围，不再标注“待独立复审”；06-4 明确为“首轮 Review 后代码整改/离线 Gate 已通过、Testnet 未验、待独立二次复审”。本复评保留原报告。 |
| **F9 P3：Testnet、MySQL8、部署 Gate** | **仍待 06-6**。本轮未进行真实 Testnet 签名 REST/WS、真实成交、MySQL8 EXPLAIN/P95、生产只读 Schema 预检、二进制与前端成套发布/回滚；不得因 Fake 全绿称已可实盘。 |

## 二、额外边界说明

- 手工确认从未被交易所接受的订单，是**显式人工操作**，不会在 `RecoverOutstanding` 或重试中自动触发。即使已过 30 分钟，网络异常、订单已存在、任何已入账 Fill 都不能释放原周期。
- 初次新订单在本地落库并完成交易所权威“无此单”检查后，才允许首次 POST；对持久化旧 client ID 的后续重试永远只查不发。两者语义明确区分。
- 不新增数据库版本/新交易状态表，不修改 Futures 表，不把钱包里人工持有的现货认领为策略资产。对不满足交易精度的残余库存 fail-closed，不伪造零余额或最终盈亏。
- 06-5 可进行**页面/只读联动研发**，但自动信号循环与真实执行接线仍受 Testnet 验证、dust 处置、成交恢复及发布前 Schema 检查 Gate 限制。

## 三、实际回归

| 命令 / 检查 | 结果 |
| --- | --- |
| `go test ./command ./controllers ./models ./service/spotquant ./service/spotledger ./service/spotexecution ./spot/api/binance -count=1` | **PASS，7/7 包** |
| 同 7 包 `go test -race -count=1` | **PASS，7/7 包**；macOS 非致命 `malformed LC_DYSYMTAB` 链接提示 |
| `go test -race ./service/spotexecution ./spot/api/binance -count=5 -p=2` | **PASS** |
| `go vet` 同 7 包 | **PASS** |
| `go build -o /tmp/spot_review_064_final_build .` | **PASS**；未覆盖现有服务程序 |
| `git diff --check`（后端） | **PASS** |
| F1 全量 SELL→CLOSED→新 BUY、F2 权威不存在/人工确认/旧单不重发/迟到 Fill | **PASS** |
| F3 WS 固定 Testnet 地址/订阅 ACK/封装事件、F4 FILLED 比 Fill 先到的只读恢复 | **PASS** |
| F5 开关 OFF 不查询 REST、F6 完整旧单免查/最近撤单仍核查、F7 手续费资产 | **PASS** |

**未运行/不宣称**全仓 `go test ./...` 绿（原审计已证明既有 `strategy_templates` 和 `conf/app.conf` 依赖不全）；本轮没有连接 Binance、修改真实/实验数据库、执行 sync db、进行 Testnet 真实订单、部署 static、修改 main.go 或自动 commit/push。

## 四、Gate

**06-4 首轮 Review 的 F1/F2 主要阻塞已在本地代码层处理，可继续 06-5 的 UI/只读功能开发，待独立复审。** 在真实执行/自动信号接线前仍必须通过 Testnet REST+WS、dust 例外处理、订单对账以及 06-6 的 MySQL8/发布验收；**生产与实盘 Gate 维持 BLOCKED**。
