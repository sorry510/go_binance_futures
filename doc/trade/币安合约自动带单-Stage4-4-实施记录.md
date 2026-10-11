# Binance Futures 自动带单 — Stage 4-4 异常订单、Algo 只读恢复与平仓过滤器

日期：2026-10-10

## 结论

**Stage 4-4 的离线安全实现与测试完成；真实取消/恢复自动化、重启后交易恢复仍未放行。** 新功能未接生产调度。Stage 0-LIVE/Stage 5/Stage 7 的权限、身份、权威 WS/REST 对账和人工授权仍是实盘 blocker。任何 `Lookup` 结果**不会**自动释放 `pendingReconcile`、重复发单或清除异常。

## 1. Lead 专属异常订单恢复证据

- `service/leadaccount/recovery.go`：`InspectOrderRecovery` 只依赖绑定的 Lead `AccountClient` 的签名 GET，支持普通 MARKET/LIMIT 与 Algo STOP/TAKE_PROFIT/STOP_MARKET/TAKE_PROFIT_MARKET/TRAILING_STOP_MARKET 的状态查证。接口没有任何 Binance mutation。
- 普通订单核对 Symbol、ClientOrderID、OrderType、交易所 OrderID；对累计成交量/平均成交价强制解析，不能把坏数据静默当成 0。
- Algo 首先核对 ClientAlgoID、币种、Algo OrderType、方向及仓位方向。若存在 `actualOrderId`，再以**相同 Lead 身份**查询实际订单并核对 ID、币种、方向、仓位方向；仅使用实际成交单的成交量及真实订单状态。无实际单的 `FINISHED` 不能证明成交，仍属不确定。
- `ClassifyLeadRecovery` 显式识别 FILLED、CANCELED、EXPIRED、REJECTED、部分成交及中间状态，检查 FILLED 必须有正成交量、REJECTED 不可有已成交量。未知状态、查单超时、错 ID、成交数据异常均返回 `ErrLeadRecoveryUncertain`。
- **仅产生可供 Stage 5 持久化对账使用的诊断证据**：终态也一律 `RequiresReconcile=true`，不得因撤单成功、订单拒绝或 Algo FINISHED 就断言仓位/保证金/受控量已经同步。

## 2. 受控平仓交易所规则（Stage4-3 审计 F5）

- 新增 `service/leadaccount/close_rules.go` 并在 `close.go` 的受控平仓执行前调用。顺序：状态和 Ownership/账户绑定 → Fresh Lead 仓位与有效 Mark Price → 公共 exchangeInfo 的 LOT_SIZE/MARKET_LOT_SIZE、PRICE_FILTER、MIN_NOTIONAL/NOTIONAL → `pendingReconcile=true` → 原绑定 Ownership Executor。
- 检查数量最小/最大/步长、LIMIT 价格精度和 Algo STOP/TP 的 TriggerPrice 精度；没有必要过滤器、缺失/错误触发价、坏 Mark Price 一律拒绝，不能提交给 Broker。
- 对 minNotional，**只允许平掉真实 Lead 全部该方向仓位**时采用 Binance 可能允许的完整关闭例外；若只平 managed 部分、账户还有人工加仓，则**不能**借用例外。例外属于允许发送请求而非交易所保证接受；保留交易所拒单及人工处置空间。
- 不自动更改杠杆、逐仓/全仓或 Main 交易路径；未读取 Main 私有账户数据。

## 3. 永久测试

- `service/leadaccount/recovery_http_test.go`：Mock HTTP 校验真实 SDK `/fapi/v1/order`、`/fapi/v1/algoOrder` 的 GET、Lead Key 签名以及无任何 POST/DELETE；覆盖取消后部分成交、拒单、异常累计成交、错 ClientOrderID、Algo 未触发/取消/无实际成交单的 FINISHED、触发后部分/完全成交与实际单错 symbol、504 查单超时。全程零真实 Binance 请求。
- `service/leadaccount/recovery_ledger_test.go`：隔离 SQLite 内存库使用原 Ownership 的累计成交更新，重复 PARTIALLY_FILLED 0.3、最终 CANCELED 0.5（重复上报），managed_qty 最终仍为 0.5、账户标记 lead；这个测试展示持久化幂等能力，**生产尚无此自动恢复入口**。
- `service/leadaccount/close_rules_test.go`：精度/价格 Tick/Stop 缺失/部分低于 minNotional/全平例外、人工加仓下不允许借用全平例外，以及无效价格时 Fake Broker 提交数=0、pending=false。
- 先前 `recovery_test.go` 原有纯分类测试保留。

## 4. 明确未完成和不能提前解除的 Gate

1. **Stage 5 负责权威持久恢复**：对账前必须按 account_id 加锁并核实全部未确认订单（普通单+Algo）、原订单数量、累计成交与所有 live 仓位；跨进程/重启 DB 唯一约束及交易所 WS/REST 同步后才能释放 pending，不能用 Go 实例 Mutex 代替。
2. **真实撤单/异常提交**：Stage4-4 仅实现 ReadOnly 检验与 Mock Ledger 路径，未提供任何实际 Cancel mutation；未来真实撤单需执行后无论是否返回成功都查累计成交并安全更新 DB。旧通用 `futuresownership.Executor.Cancel` 不可直接作为 Lead 实盘撤单成功即零成交的证明。
3. **交易所仍可能拒单**：平仓最小名义额的完整退出例外和 PERCENT_PRICE 等动态过滤器尚待 Gate 0-LIVE/Stage 7 真实只读/最小额实证，出现无法满足过滤器情况需报警或人工处理，不能绕过风控。
4. **没有授权路径**：`RiskController` 仍无 Enable/Resume、`pendingReconcile` 无重置方法，真实 Lead Runner 未挂生产循环，Stage 7 人工授权未实现。订单证据检查不等于许可下单。

## 5. 环境与限制

无真实交易/真实 Lead Key；不修改 `app.conf`、不执行 `sync db`、不写业务 MySQL、不更改 schema 或交易策略研究文件、不自动 commit/push；离线测试使用 Mock HTTP、Fake Broker 和 SQLite 内存库。测试临时产物在退出时删除。
