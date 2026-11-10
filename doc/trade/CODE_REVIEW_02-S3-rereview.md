# CODE_REVIEW_02-S3 整改复核（R1）

- 日期：2026-11-10
- 原 Review：[CODE_REVIEW_02-S3.md](CODE_REVIEW_02-S3.md)，保留原始证据，不覆盖
- 阶段方案：[开发方案-02-S3-策略执行与库存风控.md](开发方案-02-S3-策略执行与库存风控.md)
- 代码：`feat/trade-upgrade`，基线 HEAD `95b5a77`，保留所有既有未提交 S3 代码；**未执行 commit/push**
- 目标：修复 S3-F1～F6；真实 Spot SDK 代码继续保留，**没有实盘执行授权**

## 一、复核结论

**S3 的真实订单 SDK 构造及离线自动测试已闭环；S3 本地交易代码仍不得解除生产禁用。** 原 Review 发现的「一个环境变量 + 一行数据库模式可启用交易」风险已在本次二进制中采取强制封闭措施。但持久化授权审计和实际响应级来源证明尚未实现，因此**不把 S3 标成生产 READY**，也不把未授权状态视为真实资金的安全验收。

S4 允许并行开展前端、策略配置、只读库存和订单展示；S4 真实交易上线 Gate 仍需独立设计、实施、评审和用户授权。

| 原 Finding | 实施与证据 | 当前判断 |
| --- | --- | --- |
| **S3-F1 P1** 环境变量 + mode DB 更新即可启用实盘，无持久审计 | `s3_live_gate.go` 的 `spotRealOrderReleaseApproved=false` 使该构建**不可通过环境变量或数据库直接开启**；`AuthorizeLiveWrite` 在真实 SDK POST/DELETE 前必经同一封锁。测试 `TestS3F1ManualModeAndEnvCannotAuthorizeLiveWrites` 构造旧环境值 + DB mode=strategy + 持久化合法提交订单，仍拒绝；离线 adapter 测试统计 0 次 HTTP 调用 | **当前安全阻塞已闭合；持久审计/可撤销授权尚未实现，生产 Gate BLOCKED** |
| **S3-F2 P1** 真实 POST/DELETE 无永久测试 | `s3_live.go` 抽出生产共享的 `s3SendCreate/s3SendCancel` SDK 实际调用；`s3_live_review_test.go` 使用**完全拦截网络的 RoundTripper**、BaseURL 保留 mainnet 字符串，执行 LIMIT BUY、MARKET BUY、LIMIT SELL、DELETE，请求体、签名、ClientOrderID、TIF、数量/价格和 RESULT/撤单 JSON 均断言；错误返回身份不符也拒绝 | **PASS（离线、无实盘）** |
| **S3-F3 P2** `rest_verified=1` 仅凭 BaseURL | `canAttestLiveFills` 恒为 false，S3 不再以静态 BaseURL 或 Engine 类型给 Fill 颁发「已真实核实」标签；`TestS3F3EndpointAndFakeCannotProvideResponseEvidence` 证明即使配置 mode=strategy、错误构造携带 SDK provenance 的客户端，也只能写 `rest_verified=0` | **错误核验已封闭；真正响应级校验仍 BLOCKED** |
| **S3-F4 P2** 未核实 Fill 混入数值展示 | `HoldingReport` 跳过不可信 Fill，不使用它计算成本/库存/费用/毛净收益；只要 Holding 存在未核实 Fill，就隐藏账本成本、毛净盈亏、估值与以已核实名义展示的 `OwnedQty/AvgCostQuote`；`IncompleteReason` 用可排序多原因列表，避免覆写 | **PASS**；`TestS3F4UnverifiedFillsNeverEnterReportNumbers` 覆盖同时存在未核实 Fill/第三资产费用/持仓数量不匹配 |
| **S3-F5 P2** 门禁矩阵、runner 及 AST 永久证据不足 | `TestS3F5PersistentOrderIdentityAndStatusMatrix` 对正确/错误账户、Owner、client ID、symbol、side、type、quantity、submitting/new/reconcile_required 状态逐项验证；`TestS3F5RealWriteDefaultDenialCountsZeroHTTPRequests` 对真 SDK Adapter 的生产入口统计未授权 POST/DELETE 的 HTTP 调用数为 0；`s1_guard_test.go` AST 现在验证写前门禁必须带正确 `ctx,req,bool` 且 CREATE/DELETE 意图匹配，而非只看是否出现函数名 | **PASS（目前代码路径）** |
| **S3-F6 P3** free、成本精度、多原因、panic、E2E | free 采用更保守的 `free >= owned`（不是取 `min`）并与方案对齐；新 `s3_cost.go` 由完整 linked verified Fill 历史重建实际数量和精确 quote 成本，不再将已舍入的 `avg_cost_quote` 反复作为新成本；`HoldingReport` 对全部卖出采用全部剩余成本、部分卖出分摊保留高精度；S3 worker 新增 panic 隔离且不输出敏感 panic 内容；相关永久测试通过 | **已修复上述项**；真实响应级 `myTrades → verified → Holding` 仍受 F3 Gate 封锁，不能伪造为 E2E PASS |

### F1 最终安全语义

旧代码中的 `SPOT_STRATEGY_LIVE_TRADING_APPROVAL` 精确值加一条 `config.spot_auto_trade_mode='strategy'`，**不再构成实际可执行的实盘授权**。本次直接在编译的真实写 Gate 中封闭实盘发布许可，并保留真实 Binance SDK 的独立单元测试入口。它既不是可给用户随手打开的配置，也不是生产批准机制。

**不能视为 F1 的完整生产功能闭环**：发行时要先开发拥有权限主体、审核旧单/账户状态、明确币种白名单、失效/撤销语义、审计日志与只读查询证据的持久化授权，且实盘激活不能仅靠 DB 直接改值；人工解冻也必须留下 append-only 事件。完成并另行 Review 后才能讨论解除源代码层禁用。

### F3 最终证据语义

Binance 的签名 REST **验证客户端请求**，并不表示服务端为每份 JSON 返回了可独立校验的数字签名。SDK 对应的 BaseURL 是配置，不是 `myTrades` 本次响应完整性证明；代理、缓存、回放和无日志的夹具响应都不能冒充交易所来源。

本次采用保守处理：所有 S3 新增回放的 Fill 都保持 `rest_verified=0`。将来实际开放需要在交易所 TLS/信任链下实现每次响应的来源、请求参数、时间窗口与账户/订单/Trade 身份关联，保存可审计证据并测试来源错误场景。**在此之前，真实 Holding 的自动消费与已核实盈利不能启用。**

## 二、关键回归

| 验证项目 | 本轮结果 |
| --- | --- |
| `TestS3F2ProductionCreateSDKWirePayload` | **PASS**：三个 CREATE 变体检查 `symbol/side/type/quantity/price/timeInForce/newClientOrderId/newOrderRespType/timestamp/signature`，校验 RESULT 响应字段 |
| `TestS3F2ProductionCancelSDKWirePayload` | **PASS**：DELETE 参数在表单 body，timestamp/signature 在 URL query；校验返回原 ClientOrderID |
| `TestS3F2ResponseIdentityMismatchFailClosed` | **PASS**：交易所返回另一个 ClientOrderID 时拒绝 |
| `TestS3F1ManualModeAndEnvCannotAuthorizeLiveWrites` | **PASS**：环境+DB mode+真实业务表中的合法 `submitting` 订单仍不能启动实盘 |
| `TestS3F5PersistentOrderIdentityAndStatusMatrix` | **PASS**：账户/Owner/订单号、side/type/qty 和 CREATE/DELETE 状态校验 |
| `TestS3F5RealWriteDefaultDenialCountsZeroHTTPRequests` | **PASS**：真实 `Place/Cancel` API 的未授权 HTTP 调用计数为 0 |
| `TestS3F3EndpointAndFakeCannotProvideResponseEvidence` | **PASS**：不存在「仅靠客户端/BaseURL 把 Fill 变成已核实」 |
| `TestS3F4UnverifiedFillsNeverEnterReportNumbers` | **PASS**：不可信价格/库存/收益不展示为已结算数字；三类完整性原因同时存在 |
| `TestS3F6CostBasisDoesNotRoundTripThroughStoredAverage` | **PASS**：9 次部分成交期间反复污染旧均价缓存，最终仍从成交重新计算 quote 成本 909、均价 101 |
| `TestS3F6WorkerPanicIsIsolated` | **PASS**：仅此 Spot 策略 worker 发生 panic 时不让主服务进程退出 |

本轮实际执行的构建/回归：

- `go test -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./feature/strategy/line ./command`：**PASS 5/5**。
- `go test -count=1 ./controllers -run '^TestS1SpotStrategy'`：**PASS**。
- `go test -race -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./feature/strategy/line ./command`：**PASS 5/5**；macOS `ld LC_DYSYMTAB` 警告非阻断。
- `go vet ./service/spotstrategy ./spot ./spot/api/binance ./feature/strategy/line ./command ./controllers`：**PASS**。
- `go build -o <临时目录>/go_binance_futures .`：**PASS**，临时构建已清理，未覆盖项目可执行文件。
- `git diff --check`：**PASS**。

**不宣称** `go test ./...` 全仓通过：原 Review 已定位到不相关的 controllers 测试、配置 fixture 和独立研究代码问题；本次没有为此修改其它 AI 正在工作的文件。所有 SDK 离线测试均使用无网络出口的拦截 Transport，不联系 Binance。

## 三、生产 Gate 和下一阶段

| 事项 | 当前状态 |
| --- | --- |
| 本地 S3 真实 SDK CREATE/DELETE 请求构造与解析、状态机、库存核算回归 | **PASS** |
| 环境变量/直接改库触发 S3 自动真实资金写入 | **封闭（当前版本不可放行）** |
| 正式持久化授权、币种白名单、模式切换、撤销和操作审计 | **BLOCKED：尚未实现** |
| 响应级 `myTrades` 真实来源证明及 `rest_verified=1` | **BLOCKED：尚未实现** |
| 真实 Spot 手动/旧自动仓位及未完成单核验 | **BLOCKED** |
| 人工解冻审计 / 独立发行与回滚 / 生产库迁移 | **BLOCKED** |
| S4 前端、配置、只读订单和 Holding 展示 | **可以开展**，不得提供启用真实下单或直接解除冻结的接口 |

**未更改 Schema**：`appversion.DatabaseSchemaVersion=25`。本轮没有执行 `sync db`，没有触碰生产库、真实 Binance 下单/撤单、服务器部署、`conf/app.conf`、前端或 `static`，没有 commit/push。本复评是 S3 Review 的安全整改证据，不是上线授权。
