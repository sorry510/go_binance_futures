# 币安合约自动带单 — Stage 7-1 ~ 7-4 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage7-{1,2,3,4}-实施记录.md` 对应的 **Stage 7-1 历史 Gate/Schema 静态核查、7-2 只读观察者接线、7-3 持久安全门禁/恢复审计/价格过滤器、7-4 双账户 Mock 联合回归**（工作区未提交）。基线 `HEAD = ced269c`（`feat: stage 6`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除；未改动后端与前端任何文件）
- **审查类型**：review-only
- **结论**：**Stage 7-1～7-4 完成（离线/只读口径）**。本工作线**首次出现真实生产运行路径**（显式 opt-in 的 **只读观察者**：私有 User Data WS + 内存镜像 + SELECT-only 检查），但**下单方向结构性不可达**（`LeadDurableOpenGate` 末尾**无条件**返回锁定，且探针在"缺失配置 / 手写 enabled 配置 / 急停闩后"三种情形下均拒绝）。**我 Stage 6 的 F1～F4 全部已处理**。未发现 P0/P1；新增 4 项 P3。**Gate 0-LIVE / 5 / 6 / 7 真实环境验收仍未通过**。
- **审计日期**：2026-10-11

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| 7-1 Schema/历史 Gate | ✅ `DatabaseSchemaVersion = 19` + 新增 `TestStage71SchemaVersionStaticGate` 把版本变更变成需复审的事件 ✓；**明确声明**真实 MySQL 8 库版本/`sync db`/主账户无损回归**未验收** ✓ |
| 7-2 只读观察者接线 | ✅ `BINANCE_LEAD_OBSERVER_MODE` **必须严格等于 `read_only`**；未设置 → 立即返回（**不读凭证、不建 WS、不访问 Lead API**）✓；非法值 → 报错 ✓；随后要求加密凭证可用（否则 `ErrCredentialsUnavailable` fail-closed）✓；`main.go` 仅经 `controllers.StartLeadReadOnlyOnExplicitOptIn()` 一行委托进入 ✓ |
| 7-2 只读语义 | ✅ `LeadReadOnlyHost.Run` 先做 **20 秒有界**的 `ReadOnlyInspectOwnership`（**仅 SELECT**：`OutstandingOrders` + 含 CLOSED 的 `ListPositions(...,false)`，逐行校验 `account_id=lead`），且要求 `AccountID=lead && ReadOnly && **未**宣示 Reconciled`；随后**只**启动 `stream.Run`（只读镜像）——**明确不调用** `LeadSupervisor.Recovery`/`ApplyFill`/`Execute*`/`Submit`/`Cancel` ✓（代码注释亦明示"该操作可能写 DB，故本阶段不使用"）✓ |
| 7-3 持久安全门禁 | ✅ `LeadDurableOpenGate()` 每次未来开仓预检**重新读盘**；**末尾无条件 `return ErrLeadLiveGateLocked`** → 即使有人**手写** `enabled=true/allow_new_opens=true` 的配置文件也**不可能通过** ✓（探针实证，见 §4）；执行器生产预检已接该闸（`executor.go:157/163`，Mock 证据钩子除外）✓ |
| 7-3 价格过滤器 | ✅ `PERCENT_PRICE`/`PERCENT_PRICE_BY_SIDE` 因缺乏权威动态标记价依据而**拒绝开仓**；`ValidateLeadCloseRules` 对需动态触发/限价证据的条件单保守拒绝 ✓（不在静态 exchangeInfo 上臆测 mark price ✓ —— 与我 4-4 F4 的建议方向一致）|
| 7-3 恢复审计 | ✅ 审计新增固定动作 `recovery_inspection` + 固定理由码；无 Secret/无原始 HTTP ✓；**明确声明**它不替代"订单恢复写回后的完整证据链审计" ✓（与 4-6/Stage5 的口径一致）|
| 7-4 联合回归 | ✅ `feature/account_trade_cycle_stage7_4_test.go`：Main/Lead 四种开关组合 × 8 轮加速并行 Fake；Lead 风控故障下 Main 继续 12 轮且信号数不变；生产 Lead runner 仍**缺少测试专用 permit 因而无法构造** ✓；记录明确"加速逻辑 Mock ≠ 真实 2s 网络负载" ✓ |
| 上轮 Stage 6 审计项 | ✅ **F1** 改为**白名单守卫**（仅 `controllers/lead_account.go` 可导入 `service/leadaccount`，调用符号须在精确白名单内，控制器禁 `Run`/`NewLeadSupervisor`/`NewLeadUserStream`/`ExecuteOpen`/`Cancel`/`Submit`/`Reconcile`，含别名导入/点导入/跨包导入/构造/Run 的**反例自测**）✓；**F2** 新增进程内单向**急停闩** `LatchLeadEmergencyPause()`（先闩后尽力持久化；202 + `pause_persisted=false`/`pause_in_memory=true`；审计失败不阻断暂停）✓；**F3** `alert_queue_present` + 前端仅在 true 时显示丢弃数 ✓；**F4** 前端产物**已同步**（100/100 文件一致、旧哈希资源清理、保留 `platform-config.json`）✓ |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 只读观察者链路核对（7-2 核心）

```
main.go（DB 版本检查之后）
  └─ controllers.StartLeadReadOnlyOnExplicitOptIn()      // 一行委托
       └─ leadaccount.StartConfiguredReadOnlyHost()
            ├─ env 未设置 → return nil（无副作用）
            ├─ env != "read_only" → error
            ├─ 加密凭证（0600/0700/非符号链接）必须可用，否则 ErrCredentialsUnavailable
            ├─ NewLeadAccountClient(lead key) + NewLeadUserStream(镜像)
            ├─ 注册表抑制重复启动（ErrLeadSupervisorAlreadyRunning）
            └─ go host.Run(signal ctx)
                 ├─ 20s 有界 SELECT-only 检查（含 CLOSED；逐行校验 lead）
                 ├─ 审计 best-effort（未配置则 AuditRecorded=false）
                 └─ stream.Run(ctx)  ← 只读镜像；**不启动 supervisor、不写 DB、不下单**
```

- `StopReadOnlyHost()`：摘除注册表 + `revoked=true` + cancel + **有界等待（3s）** ✓，用于凭证轮换前先停旧 Key 的 WS ✓；`Status()` 的 `LiveTradeAuthorized` 恒为 false（探针断言 nil 宿主与注册表状态均如此）✓。
- **无可写路径**：该链路只构造 `LeadUserStream`/`LeadAccountMirror`（Stage 5 已验证 WS 仅更新只读投影）与 SELECT-only 检查 ✓；执行器/监督器**完全未参与** ✓。

---

## 3. 下单向不可达的四层叠加

| 层 | 机制 | 证据 |
| --- | --- | --- |
| ① 配置写入面 | `ValidateLeadSettings` 拒绝 `enabled=true`/`allow_new_opens=true`/非 lead/非法数值 | `management.go:37-53` + Stage 6 永久测试（含主账户）✓ |
| ② 每次开仓预检 | `LeadDurableOpenGate()` **重新读盘**，且**末尾无条件返回锁定** | `management.go:161-177`（`return ErrLeadLiveGateLocked` 在函数最后一行）✓ |
| ③ 急停闩 | `LatchLeadEmergencyPause()` 单向；`LeadEmergencyPaused()` 在闸与执行器双重判断 | `management.go:32/36`、`executor.go:157` ✓ |
| ④ 执行器其余门禁 | Stage 4-2 风控 + Stage 4-3 状态门禁 + 只读 Broker 写恒锁 + Stage 4-5 Guard | 4-2/4-3/4-5 审计与本轮回归 ✓ |

**探针实证**：缺失配置文件 → 拒绝；**手写 `enabled=true/allow_new_opens=true` 配置** → 拒绝（因 `LeadConfigStore.Load` 自校验，错误身份为 `lead_config_storage_unavailable`，见 F3）；闩后 → `ErrLeadLiveGateLocked` ✓。

---

## 4. 本轮实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-S7-1** 持久门禁不可通过 | 缺失配置 / 手写全启用配置 / 闩后 | ✅ 三种情形**全部拒绝**；闩后为 `ErrLeadLiveGateLocked`；闩为单向（`LeadEmergencyPaused()` 恒真） |
| **PR-S7-2** 只读宿主与 opt-in | 宿主构造约束、nil/注册表状态、opt-in 三态 | ✅ `NewLeadReadOnlyHost(nil,nil)` 拒绝；nil 宿主与注册表 `LiveTradeAuthorized=false`、`Running=false`；env 未设置=no-op；非法模式报错；`read_only` 无凭证 → `ErrCredentialsUnavailable`（fail-closed） |

> 说明：`Run` 内"检查结果必须是 `lead` + `ReadOnly` + **非 Reconciled**"这一守卫**本轮为读取核对**（探针无法在无真实 Lead 客户端时构造 stream）；其生命周期行为由永久测试 `TestStage72ReadOnlyHostLifecycleNoExchangeMutations` 覆盖 ✓。既有永久测试另覆盖：`TestStage72DisabledObserverDoesNotReadCredentialsOrNetwork`、`TestStage72CredentialRotationBeforeRunCannotStartOldWS`、`TestStage72OwnershipReadOnlyInspectionDoesNotReconcile`、`TestStage73DurablePauseFileBlocksAcrossInstancesAndAudit`、`TestStage73DynamicPriceNeedsAuthoritativeReference`、`TestStage71SchemaVersionStaticGate` + 白名单守卫反例自测 + 控制器 202 语义测试 ✓。

---

## 5. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race`（leadaccount / feature-api-binance / futuresownership / feature / line / scanner / backtest / command / controllers / middlewares，`-skip` 研究模板用例） | ✅ **10 包全部 ok**（leadaccount 1.98s、backtest 7.23s、command 4.72s、feature 4.13s 等） |
| `go vet`（含 `./routers`） | ✅ 干净 |
| 根包生产 `go build` | ✅ 成功（临时二进制已删除） |
| `git diff HEAD --check -- '*.go'` | ✅ 干净 |
| 探针 PR-S7-1/2 | ✅ 全部通过（含 `-race`）；副本去掉探针后 `./service/leadaccount` 原测试 ok |
| 前端 | ✅ 记录 §F4 声明已 build + 同步（100/100 文件一致、旧哈希清理）；本轮未复跑构建 |
| 真实 Binance / 真实 Lead WS / MySQL 8 / 主账户在线回归 | ❌ **未执行**（记录逐项声明 ✓） |

---

## 6. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | P3（架构语义） | 进程启动接线放在 HTTP 控制器包（`controllers.StartLeadReadOnlyOnExplicitOptIn` = 一行委托）。这是为满足"仅 `controllers/lead_account.go` 可导入 `service/leadaccount`"的白名单守卫而做的取舍 | 建议在守卫白名单中允许一个**专用 wiring 文件**（如 `controllers/lead_wiring.go`）或把委托改成 `leadaccount.StartFromEnv()` 由 `main.go` 直接调用（同时把该调用点纳入白名单/例外），避免"控制器承担进程启动职责"的语义混杂。当前有注释与守卫 ✓，不阻塞 |
| **F2** | P3（部署告知） | `BINANCE_LEAD_OBSERVER_MODE=read_only` 一旦设置，进程会**真实**创建/续期 listenKey 并连接 Lead 私有 WS（**真实私有 API 流量**），并读取账户私有数据 | 建议在部署/运维文档显式写明：① 该开关产生真实私有 WS 与 listenKey 生命周期；② 需要外网/代理条件与凭证文件就绪；③ 关闭方式（取消 env 或进程退出，`StopReadOnlyHost` 仅在轮换/退出时触发）；④ 不产生任何订单写 |
| **F3** | P3（错误身份） | **手写**完全启用配置时，`LeadDurableOpenGate` 因 `LeadConfigStore.Load` 自校验而返回 `lead_config_storage_unavailable`，与"锁定"（`ErrLeadLiveGateLocked`）不是同一身份 → 运维日志中两类拒绝可能被混淆 | 两者都**拒绝**（安全无损 ✓）。建议统一为锁定语义，或在 reason 中区分"配置被拒载"与"未授权"，便于诊断 |
| **F4** | P3（沿用） | 4-5 F1（`RuntimeStatus()` 写内部镜像）、4-4 F1~F4、4-6 F1~F3/F5、Stage 5 恢复审计流水、Stage 6 F5（"运维请求审计"≠"订单恢复证据审计流水"） | 建议在真实 Gate 前逐项收口；其中 **订单恢复证据审计流水** 是 7-3 明确点名的未完成项 |
| F5 | 观察（口径） | 7-1 的"Schema v19 / 主账户无损 / 在线回归"三项与 7-4 的"8/12 轮加速 Mock ≠ 真实 2s 负载"均**如实标注未验收** ✓ | 保持该口径；不得因离线 Mock PASS 推导实盘能力 |
| — | 正面 | ① **只读观察者与写路径彻底分离**（不复用 supervisor 的 recovery，避免"观察导致 DB 写入"）；② **下单向四层叠加**且最内层是"无条件拒绝"，即使运维手写配置文件也无法放行；③ 上轮 F1 用**白名单 + 反例自测**收口（比名字黑名单强），F2 用**先闩后持久化 + 202 语义**保证"能停"；④ 动态价格过滤器选择**保守拒绝**而非臆测 mark price ✓ |

---

## 7. 审计边界与未验证项

- **未验证**：真实 Lead Portfolio 的只读联调（真实 `userStatus`/`leadSymbol`/账户/持仓/挂单/`positionRisk`/`leverageBracket`/income）、真实私有 WS 连接与 ListenKey 生命周期。
- **未验证**：真实 MySQL 8 上的 Schema 19 迁移与主账户历史/在线业务回归（7-1 明确列出）。
- **未验证**：`LeadReadOnlyHost.Run` 的守卫分支（`Reconciled=true`/`ReadOnly=false`/错账户）——本轮为读取核对，永久测试覆盖其生命周期。
- **未验证**：前端 `pnpm` 构建与静态同步结果（按记录结论；本轮未复跑）。
- 本审计未发起任何真实 Binance 请求（未设置 `BINANCE_LEAD_OBSERVER_MODE`）、未改动任何仓库文件。

---

## 8. 附：Stage 7-1 ~ 7-4 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `service/leadaccount/read_only_host.go` | 242（新） | 只读观察者：严格 opt-in、20s 有界 SELECT-only 检查、只启动只读 WS 镜像、有界停止、`LiveTradeAuthorized=false` |
| `service/leadaccount/stage7_readonly_test.go` | 新 | 观察者禁用/生命周期/轮换/只读检查不写回/门禁不可通过等永久测试 |
| `feature/account_trade_cycle_stage7_4_test.go` | 新 | 双账户四开关组合 × 8 轮加速并行 + Lead 故障不影响 Main + 生产 Lead runner 不可构造 |
| `main.go` | +5 行 | DB 版本检查后，**显式 opt-in** 方可启动只读观察者 |
| `service/leadaccount/management.go` | 调整 | `LeadDurableOpenGate`（每次预检读盘且**无条件锁定**）、急停闩、`alert_queue_present` |
| `service/leadaccount/{executor,rules,close_rules,audit}.go` | 调整 | 生产预检接持久闸；动态价格过滤器保守拒绝；审计动作/理由码扩充 |
| `controllers/lead_account.go` | 调整 | 启动委托 + 暂停改 202/200 语义 + 状态暴露 `pause_in_memory`/`alert_queue_*` |
| `feature/lead_management_stage6_review_test.go`（+ `controllers/lead_account_stage6_review_test.go`） | 新 | 白名单守卫（含反例自测）与控制器 202 语义测试 |
| 文档 | — | `Stage7-1` ~ `Stage7-4` 实施记录 + `Stage6-审计修复记录` + 方案文档同步 |

**建议下一步（真实 Gate 顺序）**：① 先做**只读联调**：配置真实 Lead Portfolio 专用 Key + `BINANCE_LEAD_OBSERVER_MODE=read_only`，核对身份/白名单/余额/持仓/挂单/Hedge 与官方页面一致（Gate 0-LIVE）；② 完成 **MySQL 8** 备份 → `sync db` → 索引/回填核对 → Main 在线业务回归（Stage 2/3 前置 Gate）；③ 补齐**订单恢复证据审计流水**与 `RuntimeStatus` 等 F4 项；④ 处理 F1/F2/F3 的文档与语义收口；⑤ 上述全部通过并获**人工明示授权**后，才讨论 Stage 7 的最小额受控真实下单与回滚演练——**当前 Gate 0-LIVE/5/6/7 均未通过，本报告不构成任何实盘许可**。
