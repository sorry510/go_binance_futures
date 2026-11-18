# CODE_REVIEW_CM-5-rereview · COIN-M DAPI 执行原语 / 真实 Fill / 私有 WS / 保护单复审

> 日期：2026-11-18（北京时间）
>
> 分支：`feat/m-coin-futures`；HEAD `8664375 feat: cm-4`（父 `e6162bf feat: cm-3`）；master `0b3fa34 (tag v2.2.3)`。
>
> 本报告是**新增复审**，不覆盖 `doc/trade/CODE_REVIEW_CM-5.md`。审查对象为当前工作区 6 项改动：CM-4 F1 兼容修正、CM-5 DAPI 交易 API、独立执行服务、CM-5 自评与方案状态更新。本轮没有前端源码或静态资源新增改动。
>
> 审计期间未修改业务源码、未提交代码、未修改 `conf/app.conf`、未访问真实 Binance、未执行 `sync db`、未重启服务、未触碰真实仓位。探针只写在隔离副本中。

---

## 一、复审结论

| 项目 | 结论 |
| --- | --- |
| CM-4 F1 前置 Gate | ✅ 已闭环：隔离探针重新验证跨 symbol / positionSide Fill 被拒绝 |
| 独立 COIN-M DAPI 普通订单 / Algo / userTrades / 私有 WS API | ✅ 通过 |
| 开仓、平仓、双开关、合约精度、余额与远程持仓校验 | ✅ 通过 |
| 先落库后发单、稳定 clientOrderId、超时不盲重发 | ✅ 通过 |
| 真实 Fill 归属、tradeId 幂等、反向周期重放 | ✅ 通过 |
| 普通订单恢复、部分成交证据不足时 fail closed | ✅ 通过 |
| STOP / TAKE_PROFIT 双保护单、子订单身份确认后撤销兄弟单 | ✅ 通过 |
| 私有 User Data WS 生命周期与交易就绪门禁 | ✅ 通过 |
| 生产自动调度 / HTTP 手工下单接线 | ✅ 未接线，符合阶段边界 |
| 后端离线 Gate | ✅ 通过 |
| 真实 DAPI / Testnet / Demo / Mainnet / MySQL 8 / 生产发布 | ⛔ 全部 BLOCKED |
| 阶段意见 | **✅ 有条件通过：源码与离线范围可进入 CM-6；不得表述为真实下单或上线完成** |

**核心判定：** CM-5 在「执行原语、账本归属、失败恢复、保护单和私有 WS」的源码与 Fake/Mock 范围内完成，未发现 P0/P1 级缺陷。CM-4 首次复审的 F1 已由第二轮修复闭环，并在本轮 CM-5 基线上通过独立探针复验。

结论必须与以下边界同时成立：

1. CM-5 只是**可被 CM-6 显式调用的服务原语**，没有接入 `main.go`、路由、控制器或 2 秒调度；
2. 所有 API 调用、签名、成交和保护单验证只通过 HTTP Mock / SQLite / Fake 测试；
3. 没有真实签名订单、真实 Algo 父子订单、真实私有 User WS 或真实 MySQL 8 验证；
4. 不允许以「CM-5 离线完成」为理由直接打开真实交易开关或执行实盘下单。

---

## 二、基线与隔离副本

### 2.1 基线

```text
branch        feat/m-coin-futures
HEAD          8664375 feat: cm-4
parent        e6162bf feat: cm-3
master        0b3fa34 tag v2.2.3
audit date    2026-11-18
worktree      6 项变更（2 M + 4 ??）
frontend      11 项变更（沿用 CM-4/先前并行改动，本轮无 CM-5 源码改动）
```

工作区核心改动：

- `service/coinmquant/ledger_summary.go`：将 CM-5 PROTECT 子订单 Fill 作为 CLOSE 方向纳入周期汇总；
- `feature/api/binance/coin_m_trade_api.go`：独立 COIN-M 普通订单、Algo、userTrades、私有 WS API；
- `service/coinmexecution/*`：执行器、网关、账本、恢复、保护单、私有 WS；
- `doc/trade/CODE_REVIEW_CM-5.md`：实施方自评；
- 主方案状态更新为 CM-0～CM-5 源码/离线完成。

### 2.2 等价校验

后端隔离副本：

```text
copy          /tmp/phase_audit_cm5
SYNCED_CHANGES=6
FILES_CHECKED=1140
GO_FILES=752
STATIC_FILES=120
SHA256_MATCHED=1140
SHA256_DIFF=0
SHA256_MISSING=0
DELETED_CONFIRMED=0
APP_CONF_PRESENT=True
```

前端无本轮 CM-5 源码改动；仓库仍有 11 项历史/并行变更。`frontend/dist` 与后端 `static` 均为 120 文件，`diff -rq` 退出码 0。

### 2.3 审计限制说明

执行一次敏感文件直读时环境未授权，因此未逐行直读 `service/coinmexecution/types.go`。该限制不依赖推断：接口契约通过实现文件、测试中的 `var _ Gateway = (*mockCM5)(nil)` 编译期断言、全量 `vet/build/test` 与跨文件引用点交叉验证。未发现由该限制导致的未验证资金路径。

---

## 三、CM-5 契约逐项对照

| CM-5 要求 | 代码证据 | 判断 |
| --- | --- | --- |
| 独立 DAPI 普通订单 | `coin_m_trade_api.go:93-128` 仅 MARKET/LIMIT，要求 LONG/SHORT、`USD_PERP`、安全 clientOrderId，走受保护 COIN-M client | ✅ 通过 |
| 条件单 / TP-SL | `coin_m_trade_api.go:231-263` 仅 STOP_MARKET / TAKE_PROFIT_MARKET，`closePosition=true`、`MARK_PRICE`；独立 `/dapi/v1/algoOrder` | ✅ 通过 |
| 真实 userTrades | `coin_m_trade_api.go:219-230` 按 symbol + 精确 orderId 查询，最多 1000；满页由恢复层拒绝结案 | ✅ 通过 |
| 私有 User WS | `user_stream.go:53-172` 显式 `Run(ctx)`、listenKey、25 分钟 keepalive、断线重连；ALGO_UPDATE 停止就绪 | ✅ 通过 |
| 合约张数与预算 | `processor.go:132-188` 用 `contractsForBudget`、fresh exchangeInfo tick/step、预算 +2% 缓冲、余额校验 | ✅ 通过 |
| 双开关与策略方向 | `processor.go:16-65` 全局 `coin_m_quant_enable` + 单币 `enable`，并校验对应方向规则已启用 | ✅ 通过 |
| 不修改共享 UM/CM 模式 | `binance_gateway.go:146-177` 只读 leverage / marginType，不符则拒绝；未见改模式/杠杆/保证金写调用 | ✅ 通过 |
| 远程未归属持仓阻断 | `processor.go:142-165` 检查 open orders、open algos、双向远程持仓；任何未归属对象阻断新开仓 | ✅ 通过 |
| 只平本地 Managed 周期 | `processor.go:264-371` 要求本地 OPEN、远程同 symbol/hedge side 张数相等；OFF 不阻止退出但不认领手动仓位 | ✅ 通过 |
| 先落库后发单 | `processor.go:216-262` / `264-371`：Position + Order 事务提交后才 POST；超时保留同 clientOrderId 为 UNKNOWN | ✅ 通过 |
| 真实 Fill 与幂等 | `ledger.go:81-171` 校验 owned order、symbol/side/hedge、tradeId 唯一；REST/WS 重复不双算 | ✅ 通过 |
| 反向周期重放 | `ledger.go:180-279` 开仓均价用张数加权倒数均价；按 verified Fill 重算 open/close/remaining/realized | ✅ 通过 |
| 订单恢复 | `reconcile.go:65-146` 只按原 clientOrderId 查询，绝不重 POST；证据不足 `AWAITING_FILLS` / `ErrUncertain` | ✅ 通过 |
| CANCELED 零 Fill 处理 | `reconcile.go:123-131` 仅交易所 confirmed CANCELED/REJECTED/EXPIRED 且零 Fill 时把周期置 CANCELED | ✅ 通过 |
| 双保护单 | `protection.go:20-185` 只允许 STOP + TAKE_PROFIT 各一，归属校验、触发方向校验、本地先持久化 | ✅ 通过 |
| 触发后子订单归属 | `reconcile_protection.go:54-181` 先验证 Algo 身份和 actualOrderId，再查普通子订单与 userTrades；不认领未归属 WS 事件 | ✅ 通过 |
| 保护单取消竞态 | `reconcile_protection.go:133-138` 先验证子订单再撤兄弟单；`CancelProtection` 对格式异常 actualOrderId fail closed | ✅ 通过 |
| CM-4 账本兼容 | `ledger_summary.go` 将 PROTECT 视为 CLOSE；F1 三方归属校验保留 | ✅ 通过 |
| 不接生产自动交易 | `main.go`、`routers`、`controllers` 未引用 `coinmexecution`；无下单 UI/API | ✅ 通过 |

---

## 四、CM-4 F1 前置 Gate 复验

首轮 CM-4 复审发现跨 symbol / positionSide Fill 可污染 PnL。第二轮修复已在 CM-4 提交中合入，并在本轮 `CODE_REVIEW_CM-4-rereview-2.md` 记录为闭环。

本轮在隔离副本中重新写入独立探针：

`TestReviewCM5F1ClosureCrossSymbolAndSideRejected`

- 构造 BTC/LONG 周期；
- 注入 ETH/SHORT 的 OPEN + CLOSE Fill/order 对，内部一致且张数匹配；
- 提供有效 BTC mark；
- 断言周期为 `invalid`，未实现、总 PnL、资产汇总均为不可用。

结果：

```text
ok  go_binance_futures/service/coinmquant
F1_PROBE_EXIT=0
```

**结论：** CM-4 F1 在 CM-5 基线中保持闭环，不再阻塞进入 CM-5/CM-6 研发阶段。

---

## 五、安全边界与无自动接线

### 5.1 未接入生产入口

静态扫描结果：

```text
main.go coinmexecution refs              0
controllers coinmexecution refs          0
new HTTP open/close/algo routes          0
coinM UI createOrder/cancelOrder tokens  0
coinM UI /dapi|/fapi|/api/v3 refs         0
```

`coinmexecution.NewProcessor`、`NewUserStream`、`NewBinanceGateway` 仅在包定义和测试中出现，没有生产调用点。CM-5 不是可直接触发真实订单的后台服务。

### 5.2 无跨产品写入

CM-5 代码中出现的 `dualSidePosition` 均为只读判断或拒绝修改的注释/错误文案；未见 U 本位/Spot 订单接口，未见修改共享持仓模式、杠杆或逐全仓 API。

### 5.3 不确定结果 fail closed

- `ErrUncertain` 用于 POST 超时、查询失败、userTrades 不足、Algo 子订单身份不明、兄弟保护单状态不明；
- 任何不确定状态都保留原 clientOrderId，不重新 POST，不推断成交归属；
- 私有 WS 收到 ALGO_UPDATE 或事件处理失败时，将交易就绪状态置为未就绪，要求签名核对；
- 同一 tradeId 通过 REST/WS 重复到达时不重复写 Fill。

---

## 六、离线 Gate

执行时清空 Testnet、实盘、Demo、审批与数据库 DSN 等 env；所有 HTTP Mock 均在隔离副本内完成。

### 6.1 后端

```text
git diff --check（工作区）                                PASS
go vet ./...                                             PASS
go build -o /tmp/go_binance_futures_cm5_audit .          PASS
go test -count=1 -p=2 ./...                              PASS
    OK=75 NO_TEST=13 FAIL=0 TOTAL=88
go test -count=5 -p=2 -run '^TestCM5' ./service/coinmexecution \
    ./service/coinmquant ./feature/api/binance ./controllers  PASS
go test -count=5 -p=2 ./service/coinmexecution \
    ./service/coinmquant ./feature/api/binance ./controllers  PASS
go test -race -count=2 -p=2 ./service/coinmexecution \
    ./service/coinmquant ./feature/api/binance ./controllers  PASS
```

macOS linker `LC_DYSYMTAB` warning 非致命，退出码为 0。

### 6.2 前端与产物

```text
frontend UI changes       11（非本轮 CM-5 源码改动）
frontend/dist vs static   DIFF_EXIT=0
files per tree            120
coinM live-action tokens  0
```

本轮不重新构建前端：CM-5 没有前端源码或 i18n 改动；产物沿用 CM-4 已验证一致性。

---

## 七、观察项与阶段边界

### O1 · P2 · 无 HTTP / UI 手工下单入口是设计边界，不是能力缺失

CM-5 不提供手工开仓/撤单按钮，也不在 controller 暴露 `coinmexecution`。这符合「最小安全边界」和方案 CM-6 才接调度的安排。后续若增加手工入口，必须单独 Review，不得直接复用只读账本路由。

### O2 · P2 · userTrades 无分页，满 1000 笔直接 block

`CoinMUserTradesContext` 固定 `limit=1000`，恢复层在满页时返回 `ErrUncertain` 并保持 `AWAITING_FILLS`。这是安全的 fail-closed 行为，但真实高频部分成交/超长历史周期需要在 Testnet/Demo 单独验证；不能把离线小样本当作全分页能力通过。

### O3 · P3 · PlaceProtection 远端 Algo 身份匹配未直接比较 `OrderType`

`PlaceProtection` 已比较 `clientAlgoId`、`algoId`、`symbol`、`positionSide`、`side`；`clientAlgoId` 与 `algoId` 双重匹配使类型不一致风险很低。建议后续也加入 `remoteAlgo.OrderType == owned.OrderType`，与 `cancelSiblingProtectors` 的校验强度保持一致。

### O4 · P2 · TradeSettings 依赖交易所返回当前 symbol/side 的持仓设置行

若真实 COIN-M `positionRisk` 对零持仓合约返回稀疏结果，`TradeSettings` 会拒绝开仓。这是保守行为，但属于真实 API 契约风险，必须在 Demo/Testnet 验证后再进入自动交易。

---

## 八、未通过 / 仍 BLOCKED 的 Gate

1. 真实签名 DAPI 普通订单、查询、撤单、userTrades、Algo 与 openAlgoOrders；
2. 真实私有 User Data WS：listenKey、keepalive、断线重连、ALGO_UPDATE、订单事件；
3. 真实 Algo 触发后的 `actualOrderId` 与子订单父子关联；
4. COIN-M Demo / Testnet 开多、开空、部分成交、完全平仓、取消与重连；
5. Mainnet 小额实盘；
6. 真实 MySQL 8 v27 升级、备份、索引、P95、回滚演练；
7. ARM Linux + 静态资源 + schema 成套发布、灰度、监控、持久化授权；
8. CM-6 自动调度、出场优先、API 节流、持续 Algo/WS 回补；
9. CM-7 README、端到端、发布前检查和真实环境验收。

---

## 九、最终判定

**CM-5：✅ 有条件通过（源码与离线范围）。**

可进入 CM-6 研发，理由是：

- CM-4 F1 已修复并经本轮独立探针复验；
- DAPI 普通订单/Algo/userTrades/私有 WS 边界清楚，未发现跨产品或提前自动下单路径；
- 开平仓、Fill 幂等、恢复、保护单、取消竞态和 WS 就绪门禁均有实现与永久测试；
- 离线 Gate、五次重复测试、race、build、vet、产物一致性全部通过；
- 生产入口、HTTP 手工下单与 UI 交易动作均未接线。

同时明确不认可：

- 「真实 Binance 下单可用」；
- 「签名 DAPI / Testnet / Demo 已验收」；
- 「生产 MySQL v27 已通过」；
- 「自动交易已开启」；
- 「可以跳过 CM-6/CM-7 直接发布」。

**下一步只允许进入 CM-6 的离线接线与调度开发；任何真实订单仍需按生产统一 Gate 单独授权与验收。**


---

## 十、复审后整改追踪（2026-11-18）

原始 O1～O4 观察及当时证据保持不变。后续实施方已按 [整改与第二轮复审记录](CODE_REVIEW_CM-5-rereview-2.md) 处理：O3 补上远端 Algo `OrderType` 归属比较；O2 对满 1000 笔的 `userTrades` 显式标记本地 `AWAITING_FILLS`，继续返回 `ErrUncertain`，不宣称实现完整分页；O1 不增加手工下单入口，O4 仍须在真实 Demo/Testnet 确认。新增永久测试 `-count=5`、88 包 Go 测试、Race Detector、Vet 和 Build 均通过。**CM-5 结论保持有条件通过，仅允许继续离线开发，不代表实盘验收。**
