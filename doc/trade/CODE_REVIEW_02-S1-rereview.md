# CODE_REVIEW_02-S1 整改复核（R1）

审查依据：[CODE_REVIEW_02-S1.md](CODE_REVIEW_02-S1.md)
对应阶段：[开发方案-02-S1-配置数据与安全门禁.md](开发方案-02-S1-配置数据与安全门禁.md)
整改日期：2026-11-09
代码分支：`feat/trade-upgrade`；在既有未提交 S1 代码基础上直接修改，不重置其它工作区修改。
范围：**仅 S1 安全闭环、配置接口、静态/单元/迁移回归；不提前接入 S2 私有 REST，不下 Binance 真实委托。**

## 1. 总体复评

**S1 本地/Mock 代码可作为继续 S2 开发的安全基线**，前提是**运行模式的实际切换仍然不开放**。修复前最严重的 F1 由“禁止未核实模式切换 + 禁止门禁失败时消费通知自动订单”解决，不伪造已验证的旧退出仓位/订单。真实账户/订单/手续费/归属对账的生产 Gate 依旧 BLOCKED。

| 原 Finding | 本次落实 | 结论 |
| --- | --- | --- |
| **S1-F1 P1**：off 无旧 SELL 保护且消费通知信号 | `RequestMode` 从可写 UPDATE 改为**同模式幂等只读**：没有对账证据时任何 `legacy ↔ off`、`legacy ↔ strategy` 转换都拒绝，`strategy` 即使当前已是该状态也拒绝；`NoticeAndAutoOrder` 在任何 `has_notice/enable` 持久化和自动通知前进行来源/币种/BUY-SELL 门禁；`TryRush` 改成逐币逐方向检查，不再用 `SPOT_RUSH` 哨兵；门禁拒绝不消费信号。| **S1 内闭环**，不代表能实盘切 off；退出归属对账属于 S2/S3 |
| **S1-F2 P2**：Ctx.Resp 随 Accept 协商为 XML/YAML/Protobuf，状态码 200 | 所有新 Spot 策略 API 用 `JSONResp` 固定 JSON、`Output.SetStatus` 返回 400/401/409/415/503 等真实 HTTP 状态；同时检验 Content-Type；对 8 种 Accept 的成功/失败分支均有永久测试。 | **已修复** |
| **S1-F3 P2**：缺明确调用来源和意图 | `AuthorizeLegacyOrder(source,symbol,intent)` 明确 `spot_rush / spot_notice_auto_order / legacy_direct`、`new_risk / exit`，未知来源或意图拒绝。6 个旧 SDK 委托构造器在调用 Binance 前执行门禁，保留的抢新/通知入口传自己的来源。旧签名采用可选来源兼容，未指定时标记 `legacy_direct`（仅给已经废弃、不在本方案重构的直连方）。 | **已修复**；`exit` 仅为意图，不能据此宣称已核对旧库存 |
| **S1-F4 P2**：缺控制器级 JWT / mode 变更审计 | 新 4 个 API 全部进行已验证 `jwt.MapClaims.Username` 二次校验；无用户信息直接 401；非 JSON 写请求 415。模式申请日志包含**操作者、原状态、请求状态、结果和毫秒时间**，不记录凭据。S1 不发生实际模式转换，故不新增第四张专用模式审计表。 | **S1 已修复**；S2 若真正开放转换，必须有可持久追踪的转换审计 |
| **S1-F5 P2**：查询索引缺回归 / AST 门禁仅六函数名 | `command/db_update_test.go` 现在同时验证 5 个唯一键/PK 和 4 个常用查询索引；静态测试扫描整个 `spot/` Go 源码中的 SDK `NewCreateOrderService/NewCancelOrderService`，确认交易写操作之前有显式 source/symbol/intent 的顶层失败返回门禁；新增 `spot/s1_notice_gate_test.go` 固化自动通知落库前必须先过门禁。 | **已修复**；S2 引入真正撤单时须完善权限语义 |
| **S1-F6 P3**：expr/quote/UPDATE/并发/旧编辑器 | 按 Spot 基础变量、真实技术指标配置生成编译期允许的 expr 环境，未知变量如 `UnknownVar` 配置期拒绝；`CurrentMode` 限制 config 单行，`RequestMode` 不再使用全表 UPDATE；并发测试设 8 连接；旧 `SpotController.Edit` 补缺行拒绝、白名单字段及启用策略保护，`BatchEdit` 用参数化 ORM 更新并跳过已启用新策略。 | **其余已修复**；`exchangeInfo` 中 base/quote 真实核对属于 S2，不能仅凭旧 `spot_symbols.type` 上线 |

### F1 设计修订说明

原 S1 文档以为可以立刻提供 `off` 模式切换。但旧的 `new_symbols / notice_symbols` 自动 SELL 有可能正在承担账户退出职责，而 S1 尚没有 `openOrders / myTrades / free / locked` 的可信查询与归属关系。

因此首版**完全禁止跨模式切换**比放行 `off` 的“未核实 SELL”更安全、更简单：

- 旧模式原值 `legacy`，允许用户继续旧现货行为；本轮没有新增实盘交易入口。
- 新策略交易的配置可准备好、但交易模式仍无法开启 `strategy`。
- `off` 在 S1 **不是**一个可直接启用的安全停止入口；不能越过用户授权和旧仓退出责任。手工直接改数据库模式也不是受支持的切换路径，运行时依旧 fail-closed。
- S2 获取账户状态和订单/Fill 证据之后，必须另评审**旧 BUY 禁止、新 SELL 有归属才允许、安全模式切换的审计**，然后才能开放可变更的模式。

这一处理不改 `eatRate` 源码，不引入统一 Broker、不新建独立模式审计表。

## 2. 回归证据

- `go test -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./command`：**PASS**。
- `go test -count=1 ./controllers -run '^TestS1SpotStrategy'`：**PASS**，包含 8 个 Accept 的成功/失败 JSON、401/415 等。
- `go test -race -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./command`：**PASS**；macOS linker `LC_DYSYMTAB` 警告不阻断。
- `go vet ./service/spotstrategy ./spot ./spot/api/binance ./controllers ./command`：**PASS**。
- 主程序 `go build -o <临时目录>/go_binance_futures .`：**PASS**，临时产物删除，不覆盖原项目可执行文件。
- `git diff --check`：**PASS**。
- v25 SQLite 迁移重复执行与索引字段检查：**PASS**，未新增 Schema 变更。

新增/强化的永久测试：`spot/s1_notice_gate_test.go`、`spot/api/binance/s1_guard_test.go`、`controllers/spot_strategy_s1_test.go`、`service/spotstrategy/s1_test.go`、`command/db_update_test.go`。

此前全仓 `controllers` 策略模板测试与工作区未跟踪研究 runner 的编译冲突仍需独立处理；本次整改**不宣称** `go test ./...` 在当前工作区全绿。本轮**未再次执行真实 go_bn_test 迁移**：原 v25 DDL 没有变化；不连接 Binance 实盘，不修改数据库账号凭据、`conf/app.conf`、前端或服务进程。

## 3. 遗留 Gate / 下一阶段

- **S1 静态 + 本地单元 + Mock 基础：PASS**；S1-F1 的严重风险已通过禁止模式转换与保留旧自动 SELL 信号处理。
- **真实账户、旧未完成单、持币归属、安全下单/撤单与 `strategy/off` 模式切换：BLOCKED**；应在 S2/S3 的详细设计和 mock/最终受控验证之后再讨论。
- **真实交易对 `baseAsset/quoteAsset` 与状态/过滤器**：S2 获取并校验权威 `exchangeInfo`；S1 的字符串后缀仅能作为本地配置一致性检查，不能用于实际下单。
- **模式实际变更的持久审计**：在 S2/S3 真正允许转换之前落地；S1 目前只记录拒绝或同模式幂等请求的结构化审计日志，不表示已经完成真实切换审计。

**复核判断**：允许开始 **S2 本地/Mock 开发**。依旧禁止生产部署、真实 Spot 交易及通过 API 切换到 `off/strategy`。此文档不覆盖原 `CODE_REVIEW_02-S1.md`，只记录本次整改结果。
