# CODE_REVIEW_Lead-01：合约带单架构收缩与配置精简

> 范围：Lead-01，2026-11-17；基于本地源码、Mock/SQLite 测试与前端构建。  
> 判定：**阶段 1 核心交付完成；仍保持真实交易锁定。** 与后续交易执行相关的内部风控/生命周期组件应在 Lead-02/03 接线后再清理，禁止在阶段 1 通过删门禁假装交易闭环完成。

## 一、Lead 当前依赖关系

| 用户操作/模块 | 当前调用链 | Lead-01 处理 |
| --- | --- | --- |
| 管理页账户状态 | `leadTrading.vue` → `GET /futures/lead/status` → Lead Credential Manager / ReadOnlyHost | 保留只读；无启用交易 |
| 独立凭证 | `PUT /futures/lead/credentials` → `Manager.SaveCredentials` → AES-GCM 文件 | 保留独立加密、账号绑定与轮换；取消“审计文件必须可写”前置条件 |
| 只读验证 | `POST /futures/lead/verify` → Lead 专属凭证 / Binance 只读请求 | 保留；日志可选，不是审批门禁 |
| 账户风险 | `GET/PUT /futures/lead/config` → `LeadConfigStore` → 本地私有 JSON | 只保留两项金额上限 + 可选日亏损；仅接受新 JSON 格式，旧字段拒绝 |
| 自动交易/安全暂停 | `POST /futures/lead/pause` → 进程内单向锁 + 尽力持久化 | 保留安全减权操作；页面增加不可逆提示 |
| 真实启动/写式对账 | `POST /futures/lead/start` / `reconcile` | 均仍 HTTP 423；不因普通审计文件缺失改变拒绝结果 |
| 持仓与订单 | `/positions` / `orders` → `futuresownership` 本地 Lead 账本 | 保留且只读，不伪装为 Binance 实时仓位 |
| API 状态 | `/metrics` → `binanceapiusage` / WS ReadOnlyStatus | 复用 V4 统一采集；管理页不再显示不必要的审计/空告警列表 |
| 私有 WS | `ReadOnlyHost` 由 `BINANCE_LEAD_OBSERVER_MODE=read_only` 显式开启 | 阶段 1 保留；待统一 Runtime 完成后再收敛 |
| 可写交易链路 | `NewLeadExecutionAdapter` / `LeadDurableOpenGate` / `ReadOnlyOrderBroker` | 仍无生产可写入口；仅供后续阶段开发与测试 |

## 二、已完成的代码变更

### 1. 唯一配置结构与严格校验

- 直接在 `service/leadaccount/management.go` 定义 `LeadSettings` 和 `LeadRiskConfig`，只含唯一开关、三项额度和更新时间；不再有自定义 JSON 兼容层。
- **旧版配置不兼容、不迁移。** 文件读取使用严格字段校验；旧 `enabled`、回撤及持仓数量字段一律拒绝，并且不会自动改写已有文件。
- HTTP 与配置文件都只接受当前新结构；未知字段直接拒绝。`allow_new_opens=true` 仍在业务校验层拒绝。
- 新版 `PUT /config` 必须为两项正数上限且单笔 ≤ 总额；默认文件两者均为零仍视为**未配置**，不会当作可开仓额度；`GET /config` 提供 `risk_configured`。
- `allow_new_opens` 是唯一持久化开仓意向字段，当前固定为 false。`/status` 显示 `live_trade_authorized=false`，不再输出重复的 `enabled`。

### 2. 风控配置与默认值

- 用户可配置：单笔最大名义金额、总最大名义金额、可选每日最大已实现亏损，`0` 为关闭额外每日亏损限制。
- 内部 RiskController 同步将 `daily loss=0` 解释为不检查该额外 PnL 限制；当非零时，缺少当日 PnL 的风控仍 fail-closed。
- 旧 `MaxDrawdownPct`、`MaxPositions`、`MaxLosingPositions` **仅在旧内部风控实现中保留**，前端和持久化配置已取消；后续应复用 Main 的持仓和风险规则。此时不能因此放开内部 `Stage7Authorized`、WS、身份或账户绑定校验。

### 3. 审计与页面

- 不再因 `BINANCE_LEAD_AUDIT_FILE` 缺失/不可写而拒绝**普通风险配置、加密凭证保存或只读验证**；有审计时仍尽力记录事件，且不包含 API Secret。
- 删除 Lead 前端审计事件列表、REST 轮询，以及无人使用的后端 `/futures/lead/events` 路由和 Controller。可选内部审计日志保留。
- 页面风险输入减少为 **3 项**；单个灰置“允许新开仓”开关明确标明尚未开放；危险的“暂停”操作增加确认，其他操作和状态仍保留。
- 未对现有 `conf/app.conf`、生产密钥、数据库 Schema 或 Main 交易逻辑做变更。

## 三、验证矩阵

| 用例 | 目标 |
| --- | --- |
| `TestLead01NewConfigOnlySecurePersistence` | 新配置读写、三项额度和私有文件权限 `0600` |
| `TestLead01OldConfigFileIsRejectedWithoutMigration`、`TestLead01RejectUnknownOrRetiredConfigurationFields` | 旧格式直接拒绝，不读、不迁移；未知字段拒绝 |
| `TestLead01RiskCapPairAndOptionalDailyLoss` | 必填上限/部分零值/可选日亏损行为 |
| `TestLead01ZeroDailyLossDoesNotDemandDailyPNL` | 日亏损零值不强制依赖 Income History；非零仍需要真实 PnL |
| `TestLead01SaveConfigWithoutAuditFileAndCompactAPI` | 不配置审计仍保存普通风险，接口不回显冗余字段 |
| `TestLead01InvalidRiskFormNeverWrites` | 失效额度、跨账户、强制开启、废弃 HTTP 字段均不会写文件 |
| `TestLead01CredentialsCanSaveWithoutExtraAuditFile` | 不配置审计仍 AES-GCM 保存，输出无密钥、文件权限 0600 |
| `TestStage6*`、`TestStage7*` 与 Main 测试 | 现有只读模式、鉴权、停开仓、账户隔离、Order/Fill、Mock 交易门禁不回归 |
| 前端 TypeScript、ESLint、Vite build | 简化页面保持可编译，构建结果同步后端 `static` |

执行记录：

- `./scripts/test-go.sh -p=2 -timeout=120s`：**EXIT=0**，完整 Go 包回归通过。
- `go test -race ./service/leadaccount ./controllers -run 'TestLead01|TestStage6|TestStage7' -count=1 -p=2`：**EXIT=0**，无 Data Race；macOS 链接器有非阻断警告。
- `go vet ./controllers ./service/leadaccount ./feature ./service/futuresownership ./feature/api/binance`：**EXIT=0**。
- `pnpm typecheck`、`pnpm build`：**EXIT=0**，`dist/` 已同步后端 `static/`。
- `git diff --check`：**EXIT=0**，Go 变更文件 `gofmt -l` 无输出。

**离线测试不是 Binance 实盘验收。** 没有发送真实订单、没有操作生产数据库或部署运行中的服务器实例。

## 四、后续保留项及风险

1. **Lead-02：** 主合约选币/策略与 Lead 独立账户适配层尚未接通；RiskController 的内部最大持仓数/最大亏损数仍需与 Main 风控统一，不能简单删除检查后放行。
2. **Lead-03：** `ReadOnlyHost` 与 `LeadSupervisor`、多层 Stage 授权状态还未合并为单一生产 Runtime；当前 `LeadDurableOpenGate` 永久锁定，`ReadOnlyOrderBroker` 永久拒绝提交/撤单。只有可验证的完整交易链路才能替换这些门禁。
3. **旧安全暂停：** `POST /pause` 仍为进程级单向 latch，恢复需要按后续设计实现；**不能把关闭新开仓误认为停止已有 Managed 退出**。现在无真实自动平仓，不应声称已经验证这个运行时场景。
4. **可选事件日志：** `/events` 兼容接口已删除；内部审计文件不再参与普通配置保存/只读验证，后续可继续精简。
5. **日亏损为 0 的性能边界：** 风控**决策**在 0 时不再要求完整每日 PnL，但 `RiskEvidenceSource.Snapshot` 的旧快照组装仍可能查询 Income History。Lead-02/04 必须在共享快照与可选限制接入时取消不必要的外部调用；不能把本阶段的单元测试写成 REST 优化已完成。
6. **API 负载：** Main + Lead 同时扫描的权重、SAPI UID/IP 统计、10/30/60 币 P95 尚无真实数据；放在 Lead-04 验证。
7. **安全前提：** 独立凭证、Lead AccountID、真实 Fill Ownership、WS 健康、Binance 规则和幂等安全仍是硬约束，不能为了少代码而省略。

## 五、阶段判定

**Lead-01 的唯一配置结构、业务 UI、审计解耦、旧配置拒绝和回归覆盖已完成。** 内部策略/风险执行器重构在 Lead-02/03 按计划进行；目前只有只读/Mock 能力，不存在可启用的 Lead 实盘下单入口。进入 Lead-02 前应先审阅本报告与四阶段方案，不要以配置卡片变简洁代替完整交易闭环。
