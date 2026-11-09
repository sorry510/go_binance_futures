# 方案 02 · S2：Spot 私有 REST、真实成交与订单恢复

更新：2026-11-09
状态：**S2 已按 CODE_REVIEW_02-S2 完成 P1 整改复测，恢复本地 Mock 准入；生产实盘及模式切换仍 BLOCKED（2026-11-09）**；前置 [S1 配置数据与安全门禁](开发方案-02-S1-配置数据与安全门禁.md)，后续 [S3 策略执行与库存风控](开发方案-02-S3-策略执行与库存风控.md)。
定位：提供可 Mock、可重启恢复的现货委托和真实 Fill 记录；本 Phase **不启用真实 Binance 写接口**。

## 一、开发目标与安全约束

- 对于 `account_id=main` 且 `owner=auto_strategy`，完成一条最小私有链：**查询余额 → 本地持久预留订单 → Mock 下单 → 按原 clientId 查单 → myTrades 补真实成交 → 查单/取消 → 状态恢复**。
- 不使用 `futures_managed_orders/futures_trade_fills` 或 U 本位 `service/futuresownership` 结算实现。仅借鉴去重、事务、恢复的不变量。
- 保持原有 `spot/api/binance` 公共行情/代理/限流能力，不建私有用户流、消息队列或新的定时调度平台；现货私有订单一期走 REST 按需轮询。
- 所有执行写调用必须通过 S1 账户模式、具体币种和 Mock 门禁；未经本阶段验收和将来的独立授权，不得提交真实请求。

## 二、需要补齐的接口与文件

| 交易能力 | 对应 Spot REST/SDK | 最低约束 |
| --- | --- | --- |
| 账户余额 | `GET /api/v3/account` | 查询 base/quote 的 free、locked；失败拒绝新增风险 |
| 创建委托 | `POST /api/v3/order` | 新 `clientOrderId` 先入库；在 S2 只 Mock |
| 按 ID 查询 | `GET /api/v3/order` | `symbol+origClientOrderId` / 真实 orderId；响应丢失可继续恢复 |
| 未完成订单 | `GET /api/v3/openOrders` | 切换模式时核实所有账户委托；常态只查询必要范围 |
| 取消委托 | `DELETE /api/v3/order` | 仅属于该策略的已确认订单，S2 仅 Mock；撤单成功仍要补成交 |
| 实际成交 | `GET /api/v3/myTrades` | 分页、精确 tradeId 去重；查单状态不能取代成交事实 |
| 交易规格 | `GET /api/v3/exchangeInfo` | 已有缓存复用；数量/价格/最小成交额过滤器失效则拒单 |

建议在 `spot/api/binance/` 内增加小型 `Client` 接口（带 Context 和明确 account_id），S2 状态机/账本逻辑放 `service/spotstrategy/`。仅包装需要的六项私有操作，不扩展到多市场统一 Broker。私有 REST 的具体 SDK API、分页参数和状态枚举以当前 SDK 与官方文档核实结果为准。

**既有缺陷**：`spot/api/binance/index.go` 中 `SellLimit` 将 `quantity` 误传给 `Price`，本 Phase 修正并用数量与单价不同的用例覆盖。新订单数量使用精确十进制及交易所过滤器，不让 float64 引发不受控精度变化。

## 三、订单状态机（本地统一口径）

| 状态 | 定义及允许行为 |
| --- | --- |
| `reserved` | 已在 DB 写入 `client_order_id/signal_key` 和资源预留；无成交事实 |
| `submitting` | 发起提交，未获得可靠订单结果；不得重建同信号订单 |
| `unknown` | 提交超时、响应丢失、DB 更新失败后结果不确定；**仅按同 clientId 查单，不盲重发** |
| `new` | 交易所确认委托存在且尚无成交；继续占用预算或待卖数量 |
| `partially_filled` | 有部分成交；仅 S2 Fill 首次入库的数量允许 S3 后续更新库存 |
| `filled` | Binance 确认委托终态且全部真实 Fill 已补齐，账本可对账 |
| `canceled/expired/rejected` | 终态但仍可能已有 Fill；必须完成最终补查并释放剩余预留 |
| `reconcile_required` | 订单累计量与 Fill 或库存不一致；暂停该币新风险并提示人工核对 |

**ACK ≠ Filled；CANCELED ≠ 零成交；交易所累计 executedQty ≠ 可凭空制造 Fill。** 无法确认时不要标为 verified。除“**已终态但历史 Fill 冲突 → reconcile_required（单向人工核对冻结）**”外，禁止从终态迁移；一旦冻结，自动恢复/429/418/超时都不可降级或自动解除，且绝不重复计量。

## 四、提交流程、持久化与异常恢复

1. 验证 S1 的交易模式、账户、币种以及 Mock 限制。
2. 在数据库事务内锁定 `account+symbol` 的稳定记录，检查已有挂单、已存在 `signal_key/client_order_id`，写入 `spot_strategy_orders` 的 reserved 状态并预留预算/可卖数量。**首次 BUY 也必须有锁对象**，不能只锁尚不存在的 holding。
3. 原子提交本地预留后，将订单转为 submitting 并通过注入的 Fake Client 提交；只写请求身份、orderId 和状态，不因为 ACK 增加库存。
4. 如提交失败且订单是否被接受不明，保留 unknown + 原 clientId；下一轮只查询该 ID，拒绝新开同方向订单。
5. 仅对本策略未完成或尚未核实订单安排低频轮询；查询时按 symbol+orderId 补录 `myTrades`，必要时做受限分页，禁止扫描所有币。
6. 超时撤单需采用**查询→取消→再次查单/成交**，不能仅凭取消成功就清掉已经成交量。没有证据的外部委托禁止撤。
7. 重启扫描 reserved/submitting/unknown/new/partially_filled/reconcile_required，保持原身份恢复；**已冻结的 reconcile_required 只允许诊断性查询、更新 last_checked_at，不允许修改冻结状态与首因**。DB 异常后不标记完成、不自动提交另一个新单。
8. REST 429/418、网络失败采用已有限流与有上限退避；无法读取可靠状态时 stop-new-risk，但继续保留历史和诊断。

## 五、真实 Fill 记录的精确约定

- `spot_strategy_fills` 唯一键为 `(account_id,symbol,order_id,trade_id)`，严格按 Binance 有效成交写 `qty,price,quote_qty,commission,commission_asset,trade_time,side`。
- 精确通过账户 + 币种 + **原始 clientOrderId 优先查询且同时比对 orderId** 匹配本地 `spot_strategy_orders`；终态取消也不允许自动接纳不同 clientOrderId，归属不明时 `reconcile_required`，不可按价格、时间或数量猜归属。
- 无手续费币种且手续费非零时拒绝“完整核实”标记；base、quote、BNB 的费用单独记录；不要无依据转换 USDT。**S2 的 rest_verified 一律存 0**，不依据 Engine 构造方式推断真实 REST 签名/来源，未来实际核实机制未完成前严禁展示“真实已核实”。
- Fill 数据库事务和订单累计的 delta 处理须有**唯一生效路径**：重复拉取不能重复增库存；S3 再消费本阶段核实的新增 Fill，避免入库和状态同步都计一次。
- 成交详情可按订单查询，缺交易所历史范围或者数量不符时显示 incomplete，不伪造已实现净收益。

## 六、必须实现的自动化用例

| 测试 | 输入 | 预期 |
| --- | --- | --- |
| T2-01 | 下单得到 ACK/NEW，成交为 0 | 订单存在、Fill 为 0、策略库存不增加 |
| T2-02 | BUY 10，先成 2@100、2@120，余 6 撤 | 真实成交 4、成交均价 110，保留撤单与 Fill |
| T2-03 | 同一 tradeId 导入 100 次及跨分页重复 | Fill 仅 1 条、数量与手续费不重复 |
| T2-04 | 请求超时但 Binance 已受理 | 原 client ID 找回委托，无第二次提交 |
| T2-05 | 请求超时且查询无法证实是否受理 | 保持 unknown，禁止新单/可报警 |
| T2-06 | 程序重启、有部分 Fill 未同步 | 按原订单补录，数量不重计 |
| T2-07 | 事务写入失败/中途退出 | 回滚或待核对，不产生虚假成交成功 |
| T2-08 | 取消与成交并发、撤单有实际成交 | 仅释放未成交额度，已成事实不清除 |
| T2-09 | REST 429/418 或挂单缺失 | 受限退避、fail-closed，不无限重试 |
| T2-10 | SellLimit quantity=3, price=12 | API 传出 price=12，不是 3 |
| T2-11 | 非 main/其它 Owner/手工委托 | 拒绝越权、误取消、误记账 |

## 七、阶段完成条件及记录

- [x] 完整可 Mock 的私有 REST 客户端，真实写调用默认关闭；没有新私有 WS 或内部任务平台。
- [x] T2-01～T2-11 有永久测试和复现命令，含重复 Fill、未知提交、重启恢复、部分撤单与事务失败。
- [x] 订单状态转换、唯一订单身份、查询分页、资金/库存预留在数据库中能核对，且不生成无事实 Fill。
- [x] 受影响 Go 测试/构建、race（可运行部分）及 `git diff --check` 通过；已有独立红点单列，不称全仓绿。
- [x] 填写最终文件列表、状态机代码与模拟输出；真实成交、取消或账户写接口**均未执行**。

**S2 → S3 Gate**：订单创建、查询、取消、真实 Fill 去重和恢复经 Mock 全部可证实。**本地模拟验证 PASS，可开始 S3 本地/Mock；实盘和模式切换仍 BLOCKED。**

官方参考：[Spot REST API](https://developers.binance.com/en/docs/products/spot/rest-api)。


---

## 八、S2 实际实施记录（2026-11-09）

### 8.1 修改范围与用途

| 文件 | 本轮实现内容 |
| --- | --- |
| `service/spotstrategy/s2_types.go` | 最小的 Spot `Client` 接口、Balance/Instrument/OrderSnapshot/Trade、错误状态、严格 decimal 数量及交易身份证据 |
| `service/spotstrategy/s2_engine.go` | `NewMockEngine`、`NewReadOnlyEngine`、订单预留 `Prepare`、Fake 提交 `Submit`；客户端订单 ID **先落库后提交**，含 quote 余额与同报价资产跨币预留 |
| `service/spotstrategy/s2_reconcile.go` | 查单/分页 `myTrades`、事务内 Fill 去重、状态机、取消前后再查询、重启 `RecoverPending`、异常/限流分类 |
| `service/spotstrategy/s2_read.go` | 本地单订单 Fill 查询，精确保留各费用资产与 `rest_verified` 来源标记 |
| `spot/api/binance/s2_readonly.go` | 复用项目现有 Spot SDK、代理和 REST 限流：`GET account`、`exchangeInfo`、`openOrders`、`order`、`myTrades`；**`Place/Cancel` 永远返回 `ErrLiveWriteDisabled`** |
| `spot/api/binance/index.go` | 修复原有 `SellLimit` 的 `Price(quantity)` → `Price(price)`；不改变 S1 对既有委托构造器的模式 Gate |
| `service/spotstrategy/s2_test.go` | T2-01～T2-09、T2-11，真实数据库事务回放、分页/异常/冻结测试 |
| `spot/api/binance/s2_readonly_test.go` | T2-10 使用 localhost SDK 发单测试数量与价格，私有读链的签名/路径/分页参数本地 HTTP Fixture 测试 |

本次**未更改** `Schema v25` DDL 和迁移；新增表复用 S1 建立的 `spot_strategy_orders`、`spot_strategy_fills` 及唯一索引。无新增 Web API、消息队列、私有用户 WS 或后台常驻进程。未修改 `eatRate`、生产库、`app.conf`、前端代码或 `static`。

### 8.2 实际状态机与完整性约束

- **预留**：必须先确认 `main/auto_strategy`、每币配置开关、交易所 `base/quote/status/spot allowed`、数量/价差/成交额过滤器、Mock 账户 `free/locked`、账户已有 Spot 挂单、同报价资产其它待 BUY 预留；用数据库事务锁定单账户配置 + 目标币种稳定记录，唯一 `pending_slot/signal_key/client_order_id` 兜底。
- **身份（整改后）**：`clientOrderID=s510-<随机 hex>` 提交前落库，超时/断线保持 `unknown` 且总是查询**原 clientOrderId**；已知 `exchange_order_id` 仅作为双重比对，绝不按该字段单独接管另一订单。取消后 Binance 若给出不同 clientOrderId，一律冻结并待人工确认。
- **ACK ≠ Fill**：只有 `myTrades` 的真实逐笔证据可以插入 `spot_strategy_fills`；成交量、quote 金额和手续费由**精确 Decimal** 汇总，不通过订单 ACK 或累计成交量制造 Fill。
- **逐笔成交**：按同 `account+symbol+orderID+tradeID` 去重；对重复 tradeId 若价量/手续费内容变化则标为需要核对。页长上限 500、最多 20 页，使用递增 `fromId`；截断、缺页、不一致会进入 `reconcile_required`，不会释放正在占用的订单槽位。
- **终态与冻结（整改后）**：只有订单快照的 `executedQty/cummulativeQuoteQty` 与已落库所有 Fill 汇总相符才能将订单最终标记为 `filled/canceled/expired/rejected`；部分成交撤单保留 Fill，`reserved_qty=0`、`pending_slot=NULL`。历史终态发现 Fill 冲突后**唯一允许转入 reconcile_required**；首个冻结原因保存在 `last_error`，输出带 `order_id/old_status/new_status/reason/source/time` 的结构化日志。之后恢复与 `markUnknown` 不能覆盖冻结，新的 BUY 会被独立历史冻结检查拒绝。
- **恢复**：`RecoverPending(limit=1..50)` 读取可恢复记录、3 秒最小检查间隔；对 `reconcile_required` 仅诊断性探测，不改变首次冻结原因/状态。它从不重发订单、不扫描所有币、不创建常驻任务。REST 429/418/网络超时返回脱敏分类错误；`Prepare` 的 exchangeInfo、balances、openOrders 三种读失败也只对外暴露分类码，不回显签名 URL。
- **来源区分（整改后）**：无论 `NewMockEngine` 还是 `NewReadOnlyEngine`，S2 的 `rest_verified` 一律为 **0**；不能根据 Engine 构造方式推断数据真实性。真实 REST 凭证/来源证明尚未实现，后续 S3 只能将 S2 回放结果视为**模拟**，不得作为“实盘已核实盈亏”依据。

**S2 安全边界**：本阶段只对 **BUY 创建 Mock 预留**，不执行真实 SELL。真正 `CLOSE_LONG` 的库存归属、安全可卖量、基础/报价资产手续费与剩余库存/盈亏计算属于 S3。只有 Fake 客户端可处理委托或取消；生产只读适配器的 `Place/Cancel` 都硬禁。Mock 模式使用显式构造器，不尝试以当前数据库 `legacy` 模式冒充已授权的 `strategy` 实盘。

### 8.3 自动验收证据

| 用例 | 实测 |
| --- | --- |
| T2-01 ACK/NEW 没有成交 | PASS：订单已预留、Fill=0、Holding=0 |
| T2-02 部分成交、撤余 | PASS：先 2@100、再 2@120，合计 base 4、quote 440，取消余 6，费用币分开记录 |
| T2-03 重复 tradeId | PASS：100 次重复回放 + 多次 Reconcile，只产生 1 个 Fill |
| T2-04 提交超时但已受理 | PASS：同一 clientOrderId 找回，Mock 创建调用数仍为 1 |
| T2-05 提交/查询均不确定 | PASS：维持 unknown、保留 pending_slot，不重新开另一单 |
| T2-06 中断重启 | PASS：新 Engine 只读扫描并补真实模拟 Fill，旧身份不变 |
| T2-07 写库失败 | PASS：SQLite Trigger 模拟写入中断，事务回滚、无虚假 Fill |
| T2-08 取消与成交重叠 | PASS：取消时新增的成交也计入，先查→取消→再次查，终态释放余量 |
| T2-09 429/418/超时/缺资料 | PASS：分类、保留未知状态、停止新增风险 |
| T2-10 `SellLimit(3,12)` | PASS：实际 SDK 发往本地 `httptest` 的表单为 `quantity=3,price=12`，未访问 Binance |
| T2-11 其它账户/Owner/人工挂单 | PASS：不允许越权、冒名归属或错误取消 |
| 额外：分页、历史 Fill 冲突、跨币同报价资产预算、Fee/Quote、MARKET_LOT_SIZE=0 | PASS：包括超页数锁死和终态数据异常冻结 |

所有测试仅使用**临时 SQLite / Fake Client / localhost HTTP Fixture**，没有对 Binance 创建/取消订单，也没有直接访问用户实际账户。

### 8.4 本地回归与构建结果

| 检验 | 结果 |
| --- | --- |
| `go test -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./command` | **PASS 4/4** |
| `go test -count=1 ./controllers -run '^TestS1SpotStrategy'` | **PASS**，旧 S1 API 验证不回归 |
| `go test -race -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./command` | **PASS 4/4**，仅 macOS linker 的非阻断 `LC_DYSYMTAB` 警告 |
| `go vet ./service/spotstrategy ./spot ./spot/api/binance ./command ./controllers` | **PASS** |
| `go build -o <临时目录>/go_binance_futures .` | **PASS**，不覆盖原有可执行文件 |
| `git diff --check` | **PASS** |

未宣称全仓 `go test ./...` 全绿；以前 S1 报告中的策略模板与独立研究 runner 红点依然不在本次 S2 解决范围。生产/历史真实余额、实际交易手续费、外部挂单与旧 SELL 的归属尚未得到实盘证据；S1 的 `strategy/off` 模式转换依旧被硬拒绝。

### 8.5 S2 → S3 准入及遗留

**准入：S2 的 Mock/本地 REST 适配、订单身份、真实模拟 Fill 去重、部分成交、取消与重启恢复已通过，允许进入 S3 的本地/Mock 实现。**

S3 必须将 Fill 消费写入独立 Holding 周期，加入真实策略 LONG/CLOSE_LONG 语义、外部资产变化防误卖、部分卖单与分币种费用/收益计算。S2 的 `Prepare` 是 **BUY Mock 入口**，不具有真实交易权限；必须由 S3 进一步检查现有 Holding/资金归属后才可使用。实际实盘开关、真实委托与取消仍被禁止，除非之后通过独立验收及用户授权。

**本次未修改数据库 Schema、未再次执行 `sync db`、未读取/下单真实 Binance 账户、未更改 `app.conf`、未 commit/push。**


## 九、CODE_REVIEW_02-S2 整改复评（R1，2026-11-09）

原审查：[CODE_REVIEW_02-S2.md](CODE_REVIEW_02-S2.md)；详细独立复评：[CODE_REVIEW_02-S2-rereview.md](CODE_REVIEW_02-S2-rereview.md)。原 Review 文档未覆盖。

- **F1 P1 修复**：对账冻结 `reconcile_required` 为**粘性隔离态**，`markUnknown` 不可降级；`Reconcile` 在冻结时仅做诊断性查询，交易事务若观察到冻结立即中止；历史终态出现 Fill 冲突后新 BUY 仍拒绝。新增 `TestS2F1FrozenTerminalRecoveryCannotUnfreeze` 用真实 3.1 秒时间线验证 429 后不解冻，继续覆盖 418/超时/-2013/错误身份调用路径。
- **F2 P1 修复（安全性）**：删除 `!e.mockOnly` 作为 REST 核验依据；S2 所有插入的 Fill 均 `rest_verified=0`。新增 `TestS2F2FakeReadOnlyEngineCannotForgeRESTVerified`：即使 `NewReadOnlyEngine(db, fake)`，也无法生成真实核验标志。**“如何把真实签名 REST 正式认证为 1”不是本轮实现内容；这仍然阻止任何实盘核验声明。**
- **F3 P2 部分修复**：统一“唯一允许的终态回退是冻结”口径；首次冻结原因留在 DB `last_error`，并输出结构化操作日志，重复扫描不得改写首次原因。**数据库级不可变、可查询的事件表尚未实现；本轮不贸然扩张 Schema/表数；开放真正的人工解冻或实盘模式切换之前必须另行补持久审计和授权。**
- **F4 P2 修复**：`QueryOrder` 始终按原 `origClientOrderId` 查询，已知 orderId 只作为期望值进行二次比对；客户端身份或 orderId 不同就冻结，不凭终态例外自动接管。未托管订单返回 `ErrUnmanagedOrder`；新测试覆盖外部 ID 注入 `Reconcile/CancelMock/ListFills`，禁止误归属或误撤单。若确有 Binance 改名情形，保守保持人工核验，不自动接管。
- **F5 P3 修复**：`Prepare` 的 `exchangeInfo/account/openOrders` 底层 SDK 错误统一使用 `safeReadError` 分类；新测试注入带 `signature=...` 的错误，确保不会在返回消息泄露原始 URL。
- **F6 P3 文档修复**：本阶段仅含上述 S2 模块及测试的修改；工作区额外未提交 S1 整改/其它历史文档不计算为 S2 交付。原 Review 提到的 `go test ./...` 既有红点及生产 Gate 不变。

**整改后 Gate**：F1、F2 的 S3 前置 P1 风险已在本地 Fake/SQLite 场景下闭环，**可继续 S3 本地/Mock**。F3 的数据库持久事件审计、真实 REST 来源证明以及所有生产环境的账户/订单对账、真实交易、生产迁移和模式切换仍然 **BLOCKED**。Schema 仍 v25，本轮不执行 `sync db`，不触碰真实 Binance 或生产库。
