# 币安合约自动带单 — Stage 4-2 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage4-2-实施记录.md` 对应的 **Stage 4-2 Lead 账户开仓风控**（工作区未提交改动）。基线 `HEAD = 3201634`（`feat: stage3`，分支 `feat/lead-trading`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库与 `strategy_templates/` 未被改动）
- **审查类型**：review-only
- **结论**：**Stage 4-2 完成**（离线风控核验器 + 共享循环 Mock 接线）；**零生产接线**（无路由/表/迁移/后台任务，风控控制器与 lead 运行器工厂均无生产调用方）。**上轮 Stage 4-1 报告的唯一 P3（F1 超时契约）已被修复并转为 4 个永久测试**；本轮 34 例边界矩阵、控制器 API 面与并发探针全部通过。未发现 P0/P1；新增 5 项 P3。
- **审计日期**：2026-10-10

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| 计划 §4.2「最少必要的带单补充配置」 | ✅ 已实现（`RiskLimits` + `riskRunState`）：`enabled`/`allow_new_opens` 默认 false 且**无 Enable/Resume 入口**、`account_id` 固定 lead、三个额度「未配置=拒绝」、`max_drawdown_pct` 可选且需可证高水位、`exposure_rule` 以每账户 `MaxPositions/MaxLosingPositions` 实现；`api_order_budget` 由 Stage 1 的 per-Key 订单限流（18/10s）承担 |
| Gate 4 相关「无可绕过风控的带单写入口」 | ✅ **结构性满足**：`accountTradeRunner.validate()` 对 lead 强制要求 `PreflightOpen`；`riskAllowsOpen` 在 `EnsureConfig`/`SubmitOpen` 之前逐方向判定 |
| 上轮（Stage 4-1）F1「超时返回 nil error」 | ✅ **已修复**：ctx 已取消 → `invalidateVerification` + 返回该 ctx 错误；运行后 `requestCtx.Err()!=nil` → 强制 `ReadOnlyChecksPassed=false` + 追加固定 reason（防"晚到成功快照"）；新增 4 个 review 测试 |
| Main 路径是否受影响 | ✅ 不变：`riskAllowsOpen` 对 main 直接返回 true、不调用风控；永久测试断言 main 场景下 preflight **零调用** 且照常下单 |
| DB / 配置 / 路由 / Schema | ❌ 全部未改（`routers/`、`models/`、`appversion/`(仍 19)、`command/`、`conf/`）|
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 计划 §4.2 配置字段逐项核对

| 字段（计划） | 判定 | 证据 |
| --- | --- | --- |
| `enabled` = false | ✅ | `riskRunState.Enabled` 零值即 false；`RiskController` 仅暴露 `Pause`/`SetLimits`/`CheckOpen`（**探针用反射断言导出方法恰为这三个**）→ Stage 4-2 无任何启用路径 ✓ |
| `allow_new_opens` = false | ✅ | `!state.AllowNewOpens` → `new_opens_paused`；`Pause()` 只会置 false ✓ |
| `account_id` = lead 固定 | ✅ | `CheckOpen` 要求 `request.AccountID` 与 `snap.AccountID` **都**是 lead，否则 `account_not_lead` 立即返回 ✓ |
| `max_total_notional_usdt` 未配置=禁止 | ✅ | `ExposureBefore`（全部持仓+挂单剩余量）+ `OrderNotional` > 上限 → `lead_total_notional_limit`；未配置(0) → `lead_risk_limits_not_configured` ✓ |
| `max_order_notional_usdt` 未配置=禁止；不能突破各币 `Usdt*leverage` | ✅ | 单笔上限按**含 MARKET 缓冲**的名义额判定 ✓；共享币种预算按**原始计划名义额**判定（`RawOrderNotional > ConfiguredUSDT*Leverage` → `lead_shared_coin_budget_exceeded`），代码注释说明了为何不用缓冲值（否则正常 `qty=Usdt*lev/price` 会被全拒）✓ |
| `max_daily_realized_loss_usdt` 未配置=禁止；按账户与交易所时间边界 | ✅ | 需 `PNLComplete` + `PNLDayStartUTC == now.UTC().Truncate(24h)` + `PNLAsOf ≥ CapturedAt` 且不超前 2s，否则 `lead_daily_pnl_unavailable`（fail-closed）；`DailyNetRealizedPNL ≤ -limit` → `lead_daily_loss_limit` ✓ |
| `max_drawdown_pct` 可选、默认不启用 | ✅ | 仅在 `>0` 时校验，且要求 `PeakEquityVerified` + 正高水位 + `Equity ≤ Peak`，否则 `lead_drawdown_baseline_unverified` ✓ |
| `api_order_budget` 严格低于官方限制、优先为平仓/撤单预留 | ✅（由 Stage 1 承担） | Stage 1 `leadOrderLimiter`（18/10s，预留 2）在建单请求上生效；风控层不重复实现 → 见 F1（建议文档交叉引用）|
| `exposure_rule` per_account | ✅ | `MaxPositions`/`MaxLosingPositions` 仅统计本账户快照；亏损仓位沿用 Main 的 ROI 口径（`UnrealizedPNL/(|qty|*markPrice)*leverage*100 < -0.1`，与既有 `FuturesLeveragedROI` 阈值一致）✓ |

---

## 3. 风控核验器逐项核对（`service/leadaccount/risk.go`，319 行）

`CheckOpen` → `evaluateLeadOpen(...)`：仅 ctx 与"非 lead 账户"两种情形**提前返回**（fail-closed 立即）；其余问题**累积为固定 `blocking_reasons`**，`Allowed = len(reasons)==0` ✓。

| 门禁组 | 判定 | 说明 |
| --- | --- | --- |
| 上下文 / 账户 | ✅ | `risk_context_unavailable`；请求与快照账户都必须是 lead（无 main 回落 ✓） |
| 总闸（6 项） | ✅ | `lead_disabled`、`new_opens_paused`、`lead_identity_not_verified`、`portfolio_binding_not_confirmed`、`lead_ws_not_healthy`、`stage7_live_authorization_required` —— 六项全部由 `riskRunState` 布尔控制且**Stage 4-2 无 setter** → 结构上恒不可开仓 ✓ |
| 符号/方向/白名单 | ✅ | 必须 `*USDT` + LONG/SHORT；`symbols.Allows` 白名单（Stage 3 缓存，本地、无 REST）✓ |
| 快照完整性/新鲜度 | ✅ | `Positions/Orders/PNL Complete + Reconciled + !UnknownOrders`；`CapturedAt` 为空/超 10s/超前 2s → `lead_snapshot_stale` ✓ |
| 交易模式与余额 | ✅ | 必须 Hedge Mode；钱包/权益/可用/峰值需有限非负且 `Available ≤ Equity+1e-6`，否则 `lead_balance_invalid` ✓ |
| 额度已配置 | ✅ | 三个额度需 >0、`MaxPositions/MaxLosingPositions >0`、`MaxDrawdownPct ∈ [0,100)`，否则 `lead_risk_limits_not_configured`（见 F2） |
| 日 PnL | ✅ | 完整性 + UTC 边界 + 时点覆盖三重校验；阈值比较用 `<= -limit`（**恰好触阈即拦**，探针实证）|
| 回撤（可选） | ✅ | 需可证高水位；`100*(peak-equity)/peak ≥ pct` → `lead_drawdown_limit` ✓ |
| 订单类型/交易所规则/杠杆 | ✅ | 仅 MARKET/LIMIT；`PriceAndQuantityValidated`+`MarginModeVerified` 必须由账户适配器置真；`Leverage ≤ BinanceMaxLeverage` ✓ |
| 名义额与保证金 | ✅ | `RawOrderNotional = qty*price`；MARKET 用 `×1.005` 得保守 `OrderNotional`；交易所最小值按 Raw、最大值按缓冲值；`InitialMargin = OrderNotional/leverage*1.02` 且需 `≤ Available` ✓ |
| 仓位/敞口 | ✅ | 跳过零仓行（注释说明 `/fapi` 非活跃合约）；校验 side/markPrice/PnL 有限性；槽位 = 持仓 + 挂单剩余量；同币同向已存在 → 阻断；总敞口 = 现值 + 本单保守名义额 ✓ |
| 线程安全 | ✅ | 锁内只拷贝 limits/state/now 再评估；并发 `CheckOpen` 与 `SetLimits`/`Pause` 由探针在 `-race` 下验证零放行且无竞态 ✓ |

---

## 4. Stage 3 共享循环接线核对

| 要求（记录 §2） | 判定 | 证据 |
| --- | --- | --- |
| 账户级 `PreflightOpen` 注入式风险检查 | ✅ | `accountTradeRunner.PreflightOpen` 字段（`account_trade_cycle.go:42`）|
| Lead 缺失该依赖时 `validate()` 明确失败 | ✅ | `r.AccountID==lead && r.PreflightOpen==nil` → 报错（永久测试 `TestStage42LeadRunnerMissingPreflightRejected`）|
| Main 完全不依赖该 Hook | ✅ | `riskAllowsOpen` 对非 lead 直接 `return true`；永久测试用**计数型 fake** 断言 main 场景下 hook **零调用**且照常开仓 ✓ |
| LONG/SHORT 分别在精度化价格/数量、设置杠杆保证金与 `SubmitOpen` **之前**执行 | ✅ | `feature.go` 两个分支各插入 `if runner.riskAllowsOpen(...) { EnsureConfig → SubmitOpen }` → 顺序为「算价量 → **风控** → 账户配置 → 下单」✓（比记录措辞更严）|
| 被拒方向跳过、另一方向照常评估 | ✅ | 每个方向独立包在 if 内；永久测试 `TestStage42LeadDirectionPreflightIndependent` ✓ |
| 只限制新开仓（平仓/对账/撤单/通知不经过） | ✅ | `PreflightOpen` 仅在开仓分支调用；退出循环与 `CancelExpired`/`SyncPositions` 不在其内 ✓ |
| 日志不泄漏凭证/签名 | ✅ | `riskAllowsOpen` 失败只记固定 warning（**不打印 err**），并注明"future Lead adapter 应只返回类别化错误" ✓ |

---

## 5. 上轮（Stage 4-1）发现处理核对

| 上轮编号 | 问题 | 现状 | 证据 |
| --- | --- | --- | --- |
| **F1**（P3，唯一实需处理项） | 超时/失败返回 `nil` error，调用方只看 error 会误判成功 | ✅ **已修复** | `verify.go:123-128`：ctx 已取消 → `invalidateVerification` + **返回 ctx 错误**（且不触网）；`:146-153`：运行后 `requestCtx.Err()!=nil` → 强制 `ReadOnlyChecksPassed=false` + 追加 `verificationContextReason`（防"晚到成功快照"）；新增 `verify_review_test.go` 4 例（deadline 返回错误 / 已取消不触网 / 晚到 reader 不得通过 / reason 不含原始数据） |
| 探针补充的 8 项永久测试缺口（符号链接父目录、可执行位、版本不符、临时文件卫生、报告 JSON、白名单 USDT 过滤、多资产两分支、超时语义） | — | ✅ **已全部转永久测试** | `credentials_review_test.go`（路径与权限、被拒保存不破坏既有凭证、envelope 版本 fail-closed）；`verify_review_edges_test.go`（报告隔离与 JSON、白名单过滤与多资产）；`verify_review_test.go`（超时四例） |
| F2（人工对照 Portfolio）/F3（无清除入口）/F4（验证用量不展示）/F5（45s 为整体预算） | 未处理 | — | 均属 Stage 6/观察项，本轮无变化 |

---

## 6. 本轮实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-S4-2-1** 34 例边界矩阵 | 单笔/总名义**恰达上限放行、超一厘即拦**；日亏损**恰达阈值即拦**；保证金恰好等于可用则放行；交易所最大名义恰好等于缓冲后名义则放行、低一分即拦；共享币种预算恰好用尽放行、超一分即拦；快照 9s 新鲜/11s 过期/超前 1s 容忍/超前 3s 拒绝；`PNLAsOf` 早于采集时刻 → `lead_daily_pnl_unavailable`；`PNLDayStartUTC` 非 UTC 零点 → 同因；回撤恰达阈值拦、未达但**无已验证高水位**仍拦；槽位/亏损位恰达上限拦；零仓非活跃行忽略；人工反向持仓不冲突；同向挂单冲突拦；非 USDT/白名单外/main 账户/规则未验证/杠杆超限/未知订单类型各自固定 reason | ✅ 全部符合（34/34） |
| **PR-S4-2-2** 控制器 API 面 | 反射枚举 `RiskController` 导出方法 | ✅ **恰为 `{CheckOpen, Pause, SetLimits}`** —— 结构上不存在 `Enable`/`Resume`；零值控制器任何请求都不放行 ✓ |
| **PR-S4-2-3** 并发 | 8 worker × 50 次 `CheckOpen` 与 20 轮 `SetLimits`/`Pause` 竞争（`-race`） | ✅ 零放行、无竞态 ✓ |
| 生产接线 | `git grep RiskController|NewRiskController|PreflightOpen|leadMockTradeRunner` 于非测试代码 | ✅ 仅见 `PreflightOpen` 的**字段与调用者**（`account_trade_cycle.go`）；风控控制器与 lead 工厂**无生产调用方** ✓ |

> 既有永久测试已覆盖：默认控制器恒拦、`SetLimits` 不能启用、`Pause` 不能启用、完整假想证据的保守名义额与保证金（`100.5` / `100.5/4*1.02`）、逐项安全屏障 fail-closed、双向与全账户敞口、UTC 跨日、取消上下文与白名单失效、正常 MARKET 共享预算可用、以及 feature 层 4 例接线测试。**本轮探针补的是**：两端**精确边界**语义、时钟超前容差、`PNLAsOf≥CapturedAt` 耦合、控制器**导出面**断言、以及并发无竞态（均无永久测试）。

---

## 7. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race ./service/leadaccount ./feature ./feature/api/binance ./scanner ./service/futuresownership -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` | ✅ 全部 ok（leadaccount 1.58s、feature 2.35s、api/binance 1.88s、scanner 2.05s、ownership 3.09s） |
| `go vet ./service/leadaccount ./feature ./feature/api/binance` | ✅ 干净 |
| 根包 `go build -o /tmp/s42_bin .` | ✅ 成功（未在仓库留产物） |
| 探针 PR-S4-2-1/2/3 | ✅ 全部通过；副本去掉探针后 `./service/leadaccount` 原测试 ok |
| Schema / 迁移 / 路由 / 配置 | ✅ 无变更（仍 v19），与记录 §5 一致 |
| 真实 Binance / 真实 Lead Key / 真实资金 | ❌ 未执行（Gate 0-LIVE 与 Stage 7 授权仍阻塞） |

---

## 8. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | P3（口径统一） | 计划 §4.2 的 `api_order_budget` 在 `RiskLimits` 中**没有对应字段**，实际由 Stage 1 的 per-Key 订单限流（18/10s）承担 | 行为正确且分层合理（传输层限速 vs 业务风控），但建议在 Stage 4-2 记录/文档加一句交叉引用，避免后续读者以为风控层漏实现 |
| **F2** | P3（易误用） | `MaxPositions` / `MaxLosingPositions` 必须 **>0** 才视为"已配置"，否则整体报 `lead_risk_limits_not_configured`（全拒）。计划表中它们未被标注为"未配置=禁止启动"，只标了三个额度字段 | 若用户只设名义额与日亏损而留 0，会看到"额度未配置"这一泛化原因而难定位。建议：① 文档把这两个字段并入"必配"；② 或把原因细分为 `lead_position_limits_not_configured` |
| **F3** | P3（可观测性） | `riskAllowsOpen` 失败只打印固定 warning（不打印 `err`、不打印 reason 码）→ 运行期无法区分"额度不足/日亏损熔断/快照过期"等 | 建议让未来 lead adapter 的错误**携带类别码**（如 `risk:lead_daily_loss_limit`，不含凭证），并在日志中记录该类别；同时保留"绝不打印原始 HTTP/签名"的现有约定 ✓ |
| **F4** | P3（证据真实性，已声明） | `RiskSnapshot` 完全由调用方提供，风控层无法验证证据来源与完整性 | 记录 §4.1 已明确交 Stage 4-3/5 采集（专用 Key、真实余额/仓位/挂单/净 PnL、分页覆盖 UTC 起点）✓ 非本阶段缺陷 |
| **F5** | P3（并发保留，已声明） | `CheckOpen` 是**单次只读预检查**，不能替代跨请求的额度保留与幂等 | 记录 §4.2 已声明须由 Stage 4-3 用 DB 事务/唯一键实现（并强调不得用 Go Mutex 当分布式锁）✓ 非本阶段缺陷 |
| — | 正面 | ① 风控层的"总闸六项布尔 + 无 setter"使 Stage 4-2 **结构上不可能开仓**（探针反射证实导出面）；② 保守性分层清晰（账户级用 MARKET 缓冲、共享币种预算用原始计划额，避免正常下单被误拒）；③ 阻断原因**全部为固定码**，不泄漏任何账户/凭证数据（与 Stage 4-1 报告口径一致）；④ 上轮唯一实需处理项已被主动修复并补齐永久测试 ✓ |

---

## 9. 审计边界与未验证项

- **未验证**：真实 Lead Portfolio 的资金/仓位/挂单/净 PnL 证据采集（Stage 4-3/5）；真实 Binance 只读或写请求（Gate 0-LIVE 与 Stage 7 授权前均不执行）。
- **未验证**：真实跨请求额度保留与 unknown-submit 阻断（Stage 4-3）。
- **未验证**：前端（本阶段无改动）、MySQL（无 DB 变更）。
- 本审计未发起任何真实 Binance 请求、未改动用户仓库任何文件（探针仅存在于已删除的 `/tmp` 副本）。

---

## 10. 附：Stage 4-2 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `service/leadaccount/risk.go` | 319（新） | `RiskController`/`RiskLimits`/`RiskOpenOrder`/`RiskSnapshot`/`RiskDecision`；13 类开仓门禁；固定 reason；MARKET 保守缓冲与保证金垫 |
| `service/leadaccount/risk_test.go` | 237（新） | 7 个永久测试：默认恒拦、假想完整证据、逐项屏障 fail-closed、双向/全账户敞口、UTC 跨日、取消上下文与白名单失效、正常 MARKET 预算可用 |
| `feature/account_trade_cycle.go` | +28 | `PreflightOpen` 注入 + `validate()` 对 lead 强制 + `riskAllowsOpen`（main 直通、lead 失败即跳过） |
| `feature/feature.go` | 开仓分支包装 | LONG/SHORT 各自在配置/下单前判定风控；被拒方向不影响另一方向 |
| `feature/account_trade_cycle_stage4_2_test.go` | 89（新） | 默认风控拒发单、方向独立、Main 不调用 Lead 风控（计数断言）、Lead 缺 Hook 拒绝 |
| `service/leadaccount/{credentials_review,verify_review,verify_review_edges}_test.go` | 93/105/70（新） | 上轮 Stage 4-1 复评发现的永久回归测试（超时契约、路径权限、版本 fail-closed、报告 JSON、白名单与多资产） |
| 文档 | — | `币安合约自动带单-Stage4-2-实施记录.md`；方案文档 §4.2 追加实施结果；Stage 4-1 记录同步更新 |

**建议下一步**：① 采纳 F2 的文档/原因码细化，避免"未配置额度"泛化原因误导；② F3 让未来 lead adapter 返回**类别化错误**并记录（不打印原始错误）；③ 进入 **Stage 4-3**（账户绑定 Executor、订单 Claim/幂等、unknown-submit 阻断、跨请求额度保留用 DB 事务/唯一键）——期间继续保持 `enabled=false`、无真单；④ 真实只读/写能力仍等 **Gate 0-LIVE** 与 Stage 7 人工授权。
