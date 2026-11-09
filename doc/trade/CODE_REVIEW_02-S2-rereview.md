# CODE_REVIEW_02-S2 整改复核 R1

日期：2026-11-09  
代码基线：`feat/trade-upgrade`，HEAD `e40b01a`，在既有 **S0/S1/S2 未提交工作区**上增量修复。  
原审查：[CODE_REVIEW_02-S2.md](CODE_REVIEW_02-S2.md)（保留原文，不覆盖）。  
实施记录：[开发方案-02-S2-私有REST与订单状态.md](开发方案-02-S2-私有REST与订单状态.md) §9。

## 一、结论

**本地/Mock 的 S2-F1、S2-F2 两个 P1 阻塞已修复并有永久回归。S2 可作为 S3 本地/Mock 开发输入；绝不代表可以真实交易或标记真实核验。**

- F1：自动恢复不能将 `reconcile_required` 降为 `unknown`；已终态历史 Fill 冲突后，就算 `pending_slot=NULL`，也禁止新 BUY。
- F2：`rest_verified` 无论使用 Mock Engine 还是将 Fake 注入 ReadOnly Engine，都**只能写 0**。S2 暂无能够证明实际带签名 Binance REST 响应来源的授权路径，因此没有把任何数据标成 1。
- F3：冻结原因存于 `spot_strategy_orders.last_error`；首次状态变化记录结构化审计日志；**数据库级 append-only 事件审计未实现，列为真实交易/人工解冻前的未决 Gate**。
- F4/F5：严格原始 clientOrderId + 期望 orderId 双身份校验、拒绝外部订单操作、REST 错误脱敏。
- F6：修改了阶段设计和验收口径，不把并存的未提交 S1 改动误算为 S2。

## 二、Finding 逐项验收

| 审查项 | 修复位置 | 新证据 | 复评 |
| --- | --- | --- | --- |
| **S2-F1 P1** 终态隔离被自动恢复解冻 | `s2_engine.go` 的 `markUnknown` 排除 `reconcile_required`；`s2_reconcile.go` 的 `Reconcile` 对冻结态仅作诊断性查询与检查时间更新，事务锁后若状态已冻结也不允许改回普通状态 | `TestS2F1FrozenTerminalRecoveryCannotUnfreeze`：真实等待 3.1 秒、不手动修改时钟，测试终态→Fill 冲突→冻结→429 `RecoverPending`→仍冻结→第二 BUY 拒绝；另检验 418、timeout、-2013、直接 `markUnknown` | **PASS（本地）** |
| **S2-F2 P1** Fake 冒充真实 REST 证据 | `s2_reconcile.go` 插入 Fill 的 `rest_verified` 不再使用 `!e.mockOnly`，S2 固定写 `false`；`s2_read.go` 读取的结果如实反映 0 | `TestS2F2FakeReadOnlyEngineCannotForgeRESTVerified` 使用 `NewReadOnlyEngine(db,fake)` 进行真实 DB 事务回放，核对 Fill 和列值皆非已验证 | **PASS（阻止假阳性）**；真实认证来源仍 **BLOCKED** |
| **S2-F3 P2** 终态回退冲突/冻结无审计 | §三、§8.2、§9 统一“终态 → reconcile_required 是唯一安全隔离例外”及禁止自动解冻；`markReconcile` 通过旧状态条件更新，仅首次冻结保存原因并输出 `order_id/old_status/new_status/reason/source/at_ms` 结构化日志 | F1 回归与测试警告日志验证“旧状态 filled → reconcile_required”；重复扫描不覆写 `last_error` | **PARTIAL**：日志并非数据库级 append-only 审计；未创建自动/人工解冻通道 |
| **S2-F4 P2** 外部委托误识别/单测不足 | `s2_readonly.go.QueryOrder` 始终通过原 `origClientOrderId` 查单，必要时同时比对预期 `orderId`；`s2_reconcile.go` 对返回 client ID、exchange ID、symbol/side 不符直接冻结，取消后也不自动认领“同 orderId 不同 client ID”；`s2_engine.go.loadOrder` 对不存在/非合法 Owner 返回 `ErrUnmanagedOrder` | `TestS2F4UnknownOrderAndForeignIdentityCannotBeManaged` 验证外部本地 ID 的 Load/Reconcile/CancelMock/ListFills，以及伪造终态外部 clientId；`TestS2ReadOnlyAdapterSDKLocalREST` 新增 orderId 不符拒绝并验证查询只用 origClientId | **PASS（本地）** |
| **S2-F5 P3** Prepare 传出带签名的 SDK 原错 | `s2_engine.go.Prepare` 的 exchangeInfo、account/balances、openOrders 路径使用 `safeReadError` 固定分类码，不包装原始 `url.Error` | `TestS2F5PrepareRedactsAllSignedRESTFailures` 向三路径注入 `signature=do-not-leak-this` 和 URL，断言三种均不泄露签名或主机名 | **PASS** |
| **S2-F6 P3** 交付清单与全仓测试口径 | 原 S2 文档 §9 和本复评区分 S1 整改未提交文件、S2 原实现与本次 R1；旧策略模板等全仓红点不纳入此次 S2 Gate | 实测受影响包、race、主程序 build、vet；未谎称 `go test ./...` 绿 | **PASS（文档）** |

### F3 保留的边界

当前冻结状态与首因通过现有 v25 的 `status`、`last_error` 持久化，自动流程没有解除入口；结构化操作日志可用于排障，但它**不是可证明不可变的数据库事件历史**。为避免在此 Review 轮次擅自增加第 4 张业务表、升级 schema 或意外影响其它开发中的 v25 数据，暂不实现自动解除/手动解除。

**真正开放人工解冻、生产策略模式切换或真实委托之前，必须设计有权限和持久化审计的解除流程，并重新验证。** 本地 S3 Mock 不允许消费 `rest_verified=0` 去声称“实盘已核实盈亏”。

## 三、重新验证

| 检验 | 结果 |
| --- | --- |
| `go test -count=1 -run '^TestS2F[1245]' -v ./service/spotstrategy` | **PASS**，F1/F2/F4/F5 永久复现和覆盖 |
| `go test -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./command` | **PASS 4/4** |
| `go test -count=1 ./controllers -run '^TestS1SpotStrategy'` | **PASS**（S1 API 行为无回退） |
| `go test -race -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./command` | **PASS 4/4**（仅 macOS linker 非致命 `LC_DYSYMTAB` 警告） |
| `go vet ./service/spotstrategy ./spot ./spot/api/binance ./command ./controllers` | **PASS** |
| `go build -o <临时目录>/go_binance_futures .` | **PASS**，临时主程序构建不覆盖现有二进制 |
| `git diff --check` | **PASS** |

**未执行或未宣称通过**：全仓 `go test ./...`（原报告已有与本轮范围无关的 controllers、tests 配置和 leadaccount 并发红点）。也未进行真实 Binance 响应验证/账户余额查询、真实撤单、真实下单或生产压测。

## 四、后续约束

1. **允许 S3 本地/Mock 开发**：必须独立区分回放 Fill 与真正已核实的实盘 Fill；S2 当前 `rest_verified` 一律为 0。禁止从历史隔离订单自动建立新持仓周期。
2. **真实交易和解除隔离保持 BLOCKED**：新的手工解除权限、真实 REST 证据归属、可持久查询的审计轨迹、真实旧单/余额对账都需要另行设计、评审和授权。
3. **Schema 仍 v25**：没有新增审计表、修改迁移或执行 `sync db`；没有修改 `app.conf`、生产库、前端、`eatRate`、`static`，没有 commit/push。

本文件为本次 R1 整改的独立记录，原 `CODE_REVIEW_02-S2.md` 的发现及证据保持原样。
