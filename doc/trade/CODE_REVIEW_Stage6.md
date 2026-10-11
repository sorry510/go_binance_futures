# 币安合约自动带单 — Stage 6 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage6-{1..6}-实施记录.md` 对应的 **Stage 6 前后端账户配置、持仓与监控页面**（工作区未提交）。基线 `HEAD = 8622ff2`（`feat: stage 4`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针 + **前端仓库只读交叉核对**（未修改任何仓库；副本与探针已删除）
- **审查类型**：review-only
- **结论**：**Stage 6 离线 UI/API 验收通过（记录口径）**。本阶段**首次出现生产接线**（12 条 JWT 保护的只读管理路由），原先"生产二进制不含 `service/leadaccount`"的最强保证**按设计被替换**为"只读导入 + AST 禁止控制器调用写路径 + 路由数固定"。API 语义、凭证脱敏、审计-或-失败、配置不可启用等关键性质**均已核实**；未发现 P0/P1；发现 **2 项新 P3（其一建议优先）+ 4 项沿用**。
- **审计日期**：2026-10-11

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| 6-1 路由与鉴权 | ✅ 12 条 `/futures/lead/*`；`middlewares/auth.go` 的 `excludeRoutes` **不含**该前缀 → 全局 JWT 生效 ✓；控制器内 **再次**校验 `Authorization` Bearer + `Username` claim ✓；非 GET/DELETE 强制 `application/json`（拒绝环境 cookie 认证）✓；`body()` 限长 4096、`DisallowUnknownFields`、拒绝尾随数据 ✓ |
| 6-2 凭证与只读验证 | ✅ 复用 Stage 4-1 加密存储（env 密钥 + 仓库外私有文件）；PUT 保存/轮换（并自动使旧验证失效）；POST verify 仅显式触发；**状态只回显固定尾号**；DELETE 固定 423（无安全证据不删）；env 未配置则 fail-closed ✓ |
| 6-3 风控参数与暂停 | ✅ `LeadConfigStore`（`BINANCE_LEAD_CONFIG_FILE`，绝对路径 + 父目录 0700 + 非符号链接 + 文件 0600 + 原子写）；`ValidateLeadSettings` **拒绝 `enabled=true` / `allow_new_opens=true` / 非 lead / 负值/NaN/超范围 / 单笔>总额** ✓；Pause 幂等且**无法恢复开仓** ✓ |
| 6-4 持仓/订单/审计读取 | ✅ 严格 `BindAccount(lead)` → `ListPositions(..., false)`（**含 CLOSED**）+ `ListOrders`（`limit` 夹取到 [1,200]）；响应标注 `data_source=ownership_database` 且 `live_verified=false` ✓；`audit.go` 仅白名单 action/reason + 无密钥/无原始 HTTP ✓；`/reconcile` 保持未授权 ✓ |
| 6-5 前端页面与 i18n | ✅（跨仓库只读核对）`src/views/futures/leadTrading.vue`、`src/api/leadTrading.ts`、路由 `/futures/lead`（`menus.leadTrading`）、`locales/{zh-CN,en}.yaml` 双语；看板 `observability.vue` 引入 Lead 摘要且 `fetchLeadOverview()` **仅随页面加载/显式刷新**（未接入 5 秒静默轮询）✓ |
| 6-6 观测与 Gate 6 口径 | ✅ `/futures/lead/metrics` 复用全局 collector（`collector==nil → Default()` 空安全）并显式 `runtime_connected=false`/`ws_healthy=false`；`/symbols` 显式 `verified=false`；Gate 6 只按**离线 UI/API/Mock** 验收 ✓ |
| **生产入口守卫（被替换后的新边界）** | 🔶 有效但**有缺口**（见 F1）：全仓 AST 扫描允许 `service/leadaccount/supervisor.go` 内一处 `NewLeadExecutionAdapter`，并禁止 `ExecuteManagedClose/ExecuteOpen/NewReadOnlyExecutor/leadMockTradeRunner`；控制器内禁止 `Run/Submit/Cancel/ExecuteOpen` + 固定路由数。**但 `Run`（`LeadSupervisor.Run` / `LeadUserStream.Run`）/`NewLeadSupervisor`/`NewLeadUserStream` 未进入全仓禁止名单** |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 六个子阶段逐项复核

| 子阶段 | 记录声明 | 我的复核 |
| --- | --- | --- |
| 6-1 | 12 路由 + 双层 JWT + JSON 严格解析 + start/reconcile 423 且记审计 + AST 升级 | ✅ 路由数 12（`routers/router.go:155-166`）；`authorized()` 逐请求校验；`Start()/Reconcile()` 先审计后固定 423 ✓；控制器**只**引用只读/管理 API（`NewEncryptedFileStoreFromEnv`/`NewManager`/`Status`/`SaveCredentials`/`VerifyReadOnly`/`LeadConfigStore*`/`LeadAuditStore*`/`ListLeadManaged`/`LeadUsageSnapshot`）✓ |
| 6-2 | 复用加密存储、只回尾号、DELETE 423 | ✅ 探针：PUT 响应只含 `key_hint=****1234` 与 `readiness=disabled`，**不含原始 key/secret** ✓；DELETE → 423 ✓；无凭证时 verify → 503 且带固定 reasons（`credentials_not_verified` 等）✓ |
| 6-3 | 仓库外私有配置文件、只允许正当风控值、暂停幂等 | ✅ `ValidateLeadSettings` 结构上禁止启用；配置写入复用 `verifyCredentialPath`（同套路径/权限加固）+ `Chmod(0600)` + rename ✓ |
| 6-4 | 只读 lead 所有权（含 CLOSED）、审计白名单、`/reconcile` 未授权 | ✅ `ListLeadManaged` 绑定 lead + `limit` 夹取；探针确认 `limit=100000` 被夹取且 `live_verified=false`；`/reconcile` → 423 ✓ |
| 6-5 | Vue3 页面、6 Card、无歧义开关、密码输入、15 秒刷新 | ✅ 跨仓库核对文件/路由/i18n 存在；看板摘要为**显式刷新**（符合"不额外高频调用"）✓ |
| 6-6 | metrics 复用 Stage 5、跨模块测试、Gate 6 离线口径 | ✅ 探针与记录一致；记录明确列出未做的真实环境项 ✓ |

---

## 3. 本轮实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-S6-1** Pause 语义 | 未配置审计存储 / 已配置审计+配置文件 | ✅ 未配置审计 → **423 `lead_audit_not_configured`**（见 F2）；已配置 → 200 且 `enabled/allow_new_opens=false` 并写入审计 ✓ |
| **PR-S6-2** 凭证端点 | 轮换响应、删除、无凭证验证 | ✅ 响应仅 `key_hint ****1234` + `readiness disabled`（**无原始密钥**）；DELETE → 423；verify 无凭证 → 503 且带固定 reasons ✓ |
| **PR-S6-3** 严格报文与诚实只读 | 未知字段 / 超长 / 非 JSON；metrics/symbols/positions/events | ✅ 400 / 400 / 415；四个只读端点均 200 且分别断言 `runtime_connected=false`、`verified=false`、`live_verified=false`，无 secret 字段，`limit` 被夹取 ✓ |
| 当前激活调用核查 | `git grep` 生产目录中的激活型入口 | ✅ 当前**无** `NewLeadSupervisor`/`NewLeadUserStream`/`RecoveryAuditor` 的生产调用（命中项均为无关 `expr.Run`/AI runner）✓ |

> 既有永久测试覆盖：控制器五例（独立 JWT 要求、状态无启用无 Secret、**带有效 JWT 的 Start 仍 423 且落审计**、配置拒启用/拒主账户、手动对账仍锁定）+ `service/leadaccount/management_stage6_test.go`（配置文件 0600、审计白名单、Lead-only 所有权、只读无 Secret）+ 更新后的 4-6 生产入口守卫。**本轮探针补的是**：Pause 的依赖语义、凭证/删除/验证端点的脱敏与 fail-closed、严格报文校验、四个只读端点的诚实字段。

---

## 4. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race ./service/leadaccount ./feature/api/binance ./service/futuresownership ./feature ./feature/strategy/line ./scanner ./service/backtest ./command ./controllers ./middlewares -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` | ✅ **10 包全部 ok**（leadaccount 2.58s、feature 5.41s、backtest 8.33s 等） |
| `go vet`（含 `./routers`） | ✅ 干净 |
| 根包生产 `go build` | ✅ 成功（临时二进制已删除） |
| `git diff HEAD --check -- '*.go'` | ✅ 干净 |
| 探针 PR-S6-1～3 | ✅ 全部通过（含 `-race`）；副本去掉探针后 `./controllers` 原测试 ok |
| 前端（另一仓库） | 🔶 **只读交叉核对**：Lead 页面/API 模块/路由/双语词条/看板摘要均存在且刷新策略符合记录；`pnpm typecheck/eslint/build` 按记录通过（本轮未复跑构建） |
| `static/`（后端服务的构建产物） | ⚠️ **未更新**（3333 端口有既有服务在跑，记录称未改动其正在服务的 `static/`）→ 该 UI 尚未部署 ✓ 与记录一致 |
| Schema / 配置 / 迁移 | ✅ 无变更（仍 v19；未改 `app.conf`） |
| 真实 Binance / 真实 Portfolio / Stage 7 | ❌ 未执行 |

---

## 5. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | **P3（建议优先，安全网缺口）** | 生产侧最强的结构保证已被替换：现在生产**允许**导入 `service/leadaccount`（只读管理用途）。补偿控制是全仓 AST 扫描 + 控制器内禁止名单 + 路由数固定，**但禁止名单只覆盖 `NewLeadExecutionAdapter`（限 supervisor.go 一处）/`ExecuteManagedClose`/`ExecuteOpen`/`NewReadOnlyExecutor`/`leadMockTradeRunner`** —— **`NewLeadSupervisor`/`LeadSupervisor.Run`/`NewLeadUserStream`/`LeadUserStream.Run`/`RecoveryAuditor.Reconcile` 不在全仓禁止名单**；而"从 `main.go` 启动 supervisor"正是最可能的未来激活路径 | 建议把守卫改为**白名单文件策略**：除 `service/leadaccount/**` 与 `*_test.go` 外，任何生产文件出现该包的任何**激活型符号**（`NewLeadSupervisor`/`Run`/`NewLeadUserStream`/`Recover`/`Execute*`/`Submit`/`Cancel`）即失败；或至少把上述符号补入现有禁止名单。当前代码**无**此类调用（本轮 grep 已核）✓，属预防性加固 |
| **F2** | P3（安全方向） | **`POST /futures/lead/pause` 在审计存储未配置时返回 423**——暂停是**fail-safe（收敛）动作**，却因依赖不可用而被拒绝；同理 `Pause` 还需配置文件存储可用 | 探针实证（423 `lead_audit_not_configured`）。建议：暂停路径改为**尽力审计**（审计失败仅告警 + 在响应中标注 `audit_unavailable`），优先保证"能停"；并在部署文档列出 `BINANCE_LEAD_AUDIT_FILE`/`BINANCE_LEAD_CONFIG_FILE` 等必需 env（缺失时的行为也要写清） |
| **F3** | P3（看板口径） | 看板展示的"告警丢弃数"来自 `/futures/lead/metrics`，而该接口调用 `LeadUsageSnapshot(nil,nil,…)`**不接告警队列**（队列仅由未启动的 supervisor 构造）→ 该数字在生产恒为 0，容易被误读为"从未丢过告警" | 建议在响应中标注该字段的来源/可用性（如 `alert_queue_present=false`），或待 supervisor 接线后一并暴露真实队列指标 |
| **F4** | P3（部署操作） | 前端产物未同步到后端 `static/`（既有 3333 服务在跑）→ 交付的 UI 尚未对新用户可见；上线时需要计划一次受控的静态产物替换 | 与计划 §0.3「不启停既有服务」一致 ✓；建议在 Stage 7/上线清单里显式加入"静态产物同步 + 端口服务滚动替换"步骤（含回滚） |
| F5 | P3（沿用） | 4-5 F1（`RuntimeStatus()` 写内部镜像）、4-4 F1~F4、4-6 F1~F3/F5、Stage 5 的恢复审计流水与动态过滤器等 | 均未处理；其中"恢复审计流水"与本阶段新增的 `audit.go` 是同一条线，建议在 Stage 7 前统一为"运维操作审计 + 订单恢复流水"两类 |
| — | 正面 | ① **审计-或-失败**：所有敏感变更先写审计，审计不可用即 423/503，不留"未审计却已生效"的窗口；② `ValidateLeadSettings` 在**类型层面**禁止启用（`enabled/allow_new_opens` 为真即报错），从根上排除"配置页绕过门禁"；③ 只读端点**诚实地**标注 `ownership_database`/`live_verified=false`/`runtime_connected=false`，不捏造实时数据；④ 4-6/5 的守卫升级不是简单删除旧断言，而是**显式审计后替换**并留下"未来授权需新的显式评审"的注释 ✓ |

---

## 6. 审计边界与未验证项

- **未验证**：前端 `pnpm typecheck/eslint/build`（记录称通过；本轮仅做只读源码交叉核对，未复跑构建）。
- **未验证**：真实 Binance 只读联调（PUT 凭证后的 `verify` 真实调用）、真实 Portfolio 身份/资产归属、真实私有 WS、真实 PnL/持仓合并（Gate 5/0-LIVE）。
- **未验证**：`service/leadaccount/audit.go` 的逐行实现细节（记录声明 ≤200 条/次、>8 MiB 拒绝；本轮仅确认文件存在与用途）。
- **未验证**：部署态行为（静态产物替换、3333 服务滚动、env 未配置时的运维表现）。
- 本审计未发起任何真实 Binance 请求、未修改后端与前端任何文件（探针仅存在于已删除的 `/tmp` 副本）。

---

## 7. 附：Stage 6 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `controllers/lead_account.go` | 318（新） | 12 条管理路由的处理器：双层 JWT、严格 JSON、审计-或-失败、start/reconcile 固定 423、凭证脱敏、只读诚实字段 |
| `routers/router.go` | +12 行 | `/futures/lead/*` 路由注册（无写交易路由） |
| `service/leadaccount/management.go` | 新 | `LeadSettings`/`LeadConfigStore`（仓库外 0700/0600 原子写）+ `ValidateLeadSettings`（禁启用）+ `ListLeadManaged`（lead 所有权，含 CLOSED，limit 夹取） |
| `service/leadaccount/audit.go` | 3593B（新） | 本地运维审计（白名单 action/reason、无密钥） |
| `service/leadaccount/metrics.go` 等 | 沿用/微调 | `/metrics` 复用全局 collector（nil 安全），不虚报 WS/PnL |
| `controllers/lead_account_stage6_test.go` + `management_stage6_test.go` | 133 + 新 | 五/多项 API 与存储断言（JWT、423、无 Secret、0600、审计白名单） |
| `feature/account_trade_cycle_stage4_6_test.go` | 更新 | 生产入口守卫替换：允许只读管理导入 + AST 禁止控制器调用写路径 + 路由数固定 |
| 前端（另一仓库） | 新/改 | `leadTrading.vue`（6 Card）、`leadTrading.ts`、路由 + 菜单 + 双语词条、系统看板 Lead 摘要（显式刷新） |
| 文档 | — | `Stage6-1` ~ `Stage6-6` 实施记录 + `Stage5-审计修复记录` + 方案文档同步 |

**建议下一步**：① 落地 **F1** 的白名单式守卫（当前无违规调用，属预防性加固，价值最高）；② 调整 **F2**（暂停应能停）+ 写清必需 env；③ 标注 **F3** 看板字段来源；④ 进入 **Stage 7** 前的准备：Portfolio 身份/资产人工对照、受控最小额真实只读与下单授权、动态过滤器实测、静态产物同步与部署/回滚清单；⑤ 三项外部确认（Stage 3 Main 在线验收、Stage 2 MySQL v19 迁移、Gate 0-LIVE）仍待执行——**Gate 5/6/7 均未通过，本轮结论不构成实盘授权**。
