# CODE_REVIEW_CM-7-rereview-2 · F8/P1 签名预检期配置竞态修复后复审

> 日期：2026-11-18，北京时间。分支：`feat/m-coin-futures`；代码基线：`63efbe8 feat: cm-6` + CM-7 工作区修复。**未提交、未部署。**
>
> 原始失败证据与“CM-7 不通过”判定见 [CODE_REVIEW_CM-7-rereview.md](CODE_REVIEW_CM-7-rereview.md)。**保留原报告，不覆盖失败结论。**

## 一、F8 原因与源码处理

原复审 F8/P1 验证：CM-6 Runner 在信号求值后虽然比较 `Strategy/Technology/Profit/Loss/TickSize/Leverage`，但 `service/coinmexecution/processor.go::Open` 随后执行 `Spec`、`TradeSettings`、`OpenAlgos`、`Position`、`Mark`、`Available`、`Query` 等远端网络预检。其间关闭 STOP/TP 或修改预算/杠杆等，旧订单仍可能走到 Binance POST，成交后没有可执行保护单。

**整改后处理：**

1. `Processor.Open` 读取初始配置后立刻执行 `coinMAutoProtectionReady`，即使不是由 CM-6 Runner 调用，也不允许初始 STOP/TP 配置不可执行的自动新仓。
2. 远端 `Query(clientOrderId)` 已确认订单不存在以后、尚未建立本地 `coin_m_managed_positions` 或 `coin_m_managed_orders` 之前，重新读取全局开关与 `coinmquant.SymbolView`，比较逐币开关、ID、合约面值、结算币、合约类型、逐/全仓、USD 预算、杠杆、tickSize、Profit/Loss、Strategy、Technology。**任何值发生变化，即使仍合法，也直接丢弃旧信号、不落账、不 POST。**
3. 对重新读取的有效配置再次执行 `coinMAutoProtectionReady`。无论网络预检期关闭总开关或逐币开关，均 fail-closed。
4. 在本地 PENDING_OPEN/订单意图持久化成功后、发出第一笔 `API.Create` 之前，再次执行同样的配置校验。如果恰好在本地事务期间修改配置，**不发 POST**，将确定未发送的本地订单标记 `REJECTED`、仓位周期标记 `CANCELED`；保留 clientOrderId 唯一记录，防止同一 ID 重试。
5. 未改 DAPI 下单/Algo 的归属规则、现货/U 本位/Lead 交易逻辑、数据库 v27 结构、全局/逐币开关语义；没有新增审批或第三个配置开关。

## 二、永久回归证据

新增 `service/coinmexecution/processor_preflight_race_cm7_test.go`：

| 回调注入位置 | 注入配置变动 | 安全断言 |
| --- | --- | --- |
| `Spec` | 15 种配置变化 | POST=0、OPEN 订单数=0、Managed 周期数=0 |
| `TradeSettings` | 同样 15 种 | 同上 |
| `OpenAlgos` | 同样 15 种 | 同上 |
| `Query`（预检末端） | 同样 15 种 | 同上 |

共 **60 个预检期竞态场景**：关闭全部 STOP/TP、分别禁用 STOP 或 TP、tickSize=0、杠杆=0、Profit/Loss/TickSize/Leverage 改为其他**仍合法**值、更改 USD 预算、更改逐/全仓、总开关 OFF、逐币开关 OFF、更改策略或技术指标。

补充测试：

- 已发生无效配置导致预检拒绝的原 clientOrderId，恢复有效配置后能在没有旧本地意图的情况下正常建立一个开仓，避免永久卡死。
- 已持久化但**确认未 POST** 的意图按 `REJECTED` + `CANCELED` 结案；再次提交原 clientOrderId 不会向交易所重复 POST。
- 前序 CM-6 F7 策略求值期竞态和 CM-5～CM-7 既有 Fake/SQLite 生命周期测试继续保持通过。

这些测试运行在**隔离 SQLite + Fake Gateway**，不使用真实 Binance API、用户凭证或生产数据库。原报告中 `TestAuditCM7ProtectionEditedDuringSignedPreflightStillOpens` 的“PASS”表示**漏洞重现成功**，不是正确业务行为；本轮永久测试以**未发单、未落新 OPEN/Managed**为正向验收条件。

## 三、本轮 Gate（修复后重新执行）

| 检查 | 结果 |
| --- | --- |
| `go test -count=5 -p=2 -run '^TestCM7' ./service/coinmexecution ./controllers` | **PASS** |
| 四个 CM 核心包 `go test -count=5 -p=2` | **PASS** |
| 所有 **88 个非历史研究 Go 包** `go test -count=1 -p=2` | **PASS** |
| 上述 88 包 `go vet` | **PASS** |
| 四核心包 `go test -race -count=2 -p=2` | **PASS**（macOS linker `LC_DYSYMTAB` warning 不致命） |
| `go build -o /tmp/go_binance_futures_cm7_f8_rereview_v27 .` | **PASS** |
| `gofmt` 和 `git diff --check` | **PASS** |

前端代码、中文/英文 locales 与 `dist/static` 在 F8 修复中均**未改动**；其先前 typecheck、production build、静态一致性以及原复审通过记录继续有效，不重复生成哈希资源。

## 四、风险边界与仍阻塞的发布事项

**已关闭的缺陷范围：** 配置在 Runner 信号评估阶段、Processor 的远端签名预检期及本地意图提交期间变化时，有明确的前置拒绝路径，原 P1 探针不再能够因 `Spec`、`TradeSettings`、`OpenAlgos`、`Query` 回调修改配置而发送真实开仓 POST。

**并未证明跨系统事务原子性：** 数据库最后一次读取与 Binance HTTP POST 是两个独立操作，配置在最后一次查询之后的极端并发修改仍无法由本轮 Fake 测试保证绝对不存在；要严格线性化需跨修改入口与下单调用加锁、版本化控制或等价机制，而个人使用的本轮最小修复未新增该级别的架构。不能声称“任何时刻并发修改均已得到数学意义的原子隔离”。

**仍未执行：** MySQL 8 v26→v27 实际迁移、独立 Demo/Testnet 凭证/WS/真实 LONG/SHORT 订单、真实 STOP/TP Algo 父子 Fill、ARM 生产部署和小额主网验证；前端手动受控交易写操作仍未开放；Lead/登录等并行前端修改未单独 Code Review。

## 五、复审结论

**F8/P1 在已复现的签名预检窗口内完成最小源码修复及永久隔离回归，88 包、Vet、Race 和 Build 均已 PASS。CM-7 在源码与离线测试范围重新判定通过。** 后续只可进入独立的 Demo/Testnet 发布前验收，不得据此开放真实交易。
