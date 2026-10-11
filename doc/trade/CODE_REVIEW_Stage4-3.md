# 币安合约自动带单 — Stage 4-3 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage4-3-实施记录.md` 对应的 **Stage 4-3 执行适配器与账户数据证据**（工作区未提交改动）。基线 `HEAD = 3201634`（`feat: stage3`，分支 `feat/lead-trading`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库与 `strategy_templates/` 未被改动）
- **审查类型**：review-only
- **结论**：**Stage 4-3 完成**（账户绑定执行适配器 + 只读证据/规则 + 受控平仓，全部离线且**零生产接线**）。防御纵深充分：即使有人误接线，**默认状态下也无法下单**（状态门禁 + 快照恒 `Reconciled=false`/`UnknownOrders=true` + 只读 broker 恒锁 + 适配器无激活入口）。未发现 P0/P1；发现 6 项 P3（含 1 项错误粒度问题与 3 项"更保守"的行为说明）。
- **审计日期**：2026-10-10

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| 计划 §4.3「保持既有 Executor 语义」 | ✅ 复用 Stage 2 的 `futuresownership.Executor`（持久 `ClientOrderID`、`ClaimOrder`、unknown → Lookup/Reconcile、`managed_qty` 累计、账户隔离）；适配器不另造下单原语 |
| Gate 4 相关「无可绕过风控的带单写入口」 | ✅ **四层**：① `ExecuteOpen`/`ExecuteManagedClose` 先过状态门禁（六布尔/四项）→ ② 风控核验（自读证据，忽略外部布尔）→ ③ `ReadOnlyOrderBroker.Submit/Cancel` **恒返回** `ErrLiveExecutionLocked` → ④ 适配器公开方法仅 `{CheckOpen, ExecuteOpen, ExecuteManagedClose}`（探针反射断言，无 authorize/enable/unlock/reset） |
| 生产是否可能被启用 | ❌ 不能：`RiskController` 无 Enable/Resume；`NewLeadExecutionAdapter`/`NewReadOnlyExecutor`/`ExecuteOpen`/`ExecuteManagedClose` **无生产调用方**（`git grep` 0 命中）；`RiskEvidenceSource` 恒输出 `Reconciled=false`、`UnknownOrders=true`（Stage 5 才能置真） |
| 计划「平仓量 ≤ min(managed_qty, live_account_qty)」 | ✅ **探针精确边界实证**：恰好等于 `min` 放行且仅 1 次 broker 提交；**超一微单位即拒且 0 次提交**；`live=0` 拒绝；managed/live 两侧谁小谁管 |
| DB / 配置 / 路由 / Schema | ❌ 全部未改（`routers/`、`models/`、`appversion/`(仍 19)、`command/`、`conf/`）；`feature/*` 仅沿用 4-2 的 38 行接线 |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 计划 §4.3 条目核对

| 计划要求 | 判定 | 证据 |
| --- | --- | --- |
| 保持既有 `Executor` 的方向/数量验证、ClientOrderID 持久化、unknown → reconcile、成交后改 managed_qty、手工减仓不回补 | ✅ | `executor.go:175` 直接调用绑定的 `ownership.Execute`；方向/数量校验在适配器入口再确认（`ExecuteOpen` 的 side↔positionSide、数量严格等于风控请求）；账户级 `BindAccount(lead)`（Stage 2）✓ |
| Lead Broker 独立保存/查询正常单与 Algo STOP/TP；拒单/超时/断连先查真实订单，不得以"请求失败"推断未成交 | ✅（写侧）+ 🔶（Algo 恢复交 4-4） | 写路径经账户绑定 Broker（Stage 1/2）；`execution.go` 另提供 **ReadOnlyOrderBroker**（`Submit`/`Cancel` 恒 `ErrLiveExecutionLocked`，`Lookup` 可用）作为恢复/对账专用通道 ✓；Algo STOP/TP 的独立恢复与异常场景明确留到 Stage 4-4（记录 §4） |
| 新开仓在可交易对筛选后再设置该账户杠杆/逐仓；不修改 main 配置；仓位模式不符时拒单或阻塞，**严禁自动切换规避验证** | ✅（本阶段更强） | 适配器**要求账户当前杠杆与保证金模式已与请求一致**（`rules.go:73-79`：positionRisk 行的 marginType/leverage 必须等于期望值），否则拒绝；全程无任何 leverage/margin 写调用 ✓（比"先设置再检查"更保守，见 F4） |
| 普通账户与 lead 各自独立下单、状态分别记录 | ✅ | Main 完全不经过 lead 适配器（4-2 计数型测试零调用 + 本阶段无生产接线）✓ |
| 平仓不超过 `min(managed_qty, live_account_qty)`；人工加仓不并入、人工减仓缩小可关闭量；绝不误平仓 | ✅ **实证** | `close.go`：读本账户 Ownership 的 `ManagedQty`（须 `AccountID=lead` 且 >0）+ **实时** Fresh 持仓（按 symbol+positionSide 累加 |PositionAmt|，解析异常即拒），`request.Quantity > min(managed, live)+1e-10` 即拒 ✓ |
| 手动暂停 = 停新开仓、保持风险退出/同步；停机/无法确认账户拒绝不确定写并报警 | ✅ | `ExecuteManagedClose` **不要求** `Enabled`/`AllowNewOpens`，但要求 `Stage7Authorized && PortfolioBound && ReadOnlyVerified && WSHealthy`；`pendingReconcile` 时拒绝第二次写 ✓ |
| 以带单 Portfolio 自身订单与回报为真相源 | ✅ | 证据全部取自绑定 Lead 客户端（`lead_income.go`/`lead_rules.go`/`snapshot.go`），无 Main 客户端回退 ✓ |

---

## 3. 逐文件核对

### 3.1 `feature/api/binance/lead_income.go`（23）/ `lead_rules.go`（28）
- `GetIncomeHistoryContext`：**仅** Lead 账户（`a.ID() != LeadAccountID` 直接报错 ✓）；窗口与 `limit ∈ [1,1000]` 校验 ✓；经 `doAccountSigned`（读窗口）✓；无 Main 回退 ✓。
- `GetLeadExchangeInfoContext`（lead-only + `signedMu` 串行）与 `GetLeadLeverageBracketContext`（签名 GET）✓；两者均不含任何写接口 ✓。

### 3.2 `service/leadaccount/rules_source.go`（62）与 `rules.go`（155）
- `LeadRuleSource.Verify`：exchangeInfo → 定位 symbol（`Status=TRADING`、`QuoteAsset=USDT`）→ 该币 Fresh positionRisk → 签名 leverageBracket；任一失败即 fail-closed ✓。
- `ValidateLeadOrderRules`：账户现状与请求**逐一比对**（margin 需一致，`CROSS`→`CROSSED` 显式规范化 ✓；leverage 必须等于请求值 ✓）；杠杆档位取 `InitialLeverage ≥ 请求` 中 `NotionalCap` 最大者，并回填 `MaxNotional/MaxLeverage` ✓；`LOT_SIZE`/`MARKET_LOT_SIZE` 按订单类型择一，校验 min/max/step 与步长对齐 ✓；`PRICE_FILTER` 必存在，**LIMIT** 额外校验 min/max/tick 对齐（MARKET 不校验价格区间 ✓ 正确）；`MIN_NOTIONAL` 与 `NOTIONAL` 两种过滤器形态都支持 ✓；三类过滤器缺一即拒 ✓。

### 3.3 `service/leadaccount/snapshot.go`（240）
- 任一失败即返回 `invalid`（`UnknownOrders=true`）→ 风控层必拦 ✓；账户需 `CanTrade` 且非 Multi-Assets、**恰好一条 USDT 资产**且余额有限非负、非 USDT 资产保证金须为 0 ✓；Hedge Mode 必须开启 ✓。
- 持仓：零仓跳过、非 LONG/SHORT 拒绝、mark/leverage/PnL 校验 ✓；挂单：状态必须 NEW/PARTIALLY_FILLED、方向/持仓方向合法、**平仓意图行跳过**、`filled ≤ orig`、**price 必须为正**（市价挂单 price=0 → 整份快照失效，见 F3）✓。
- 日净 PnL：`REALIZED_PNL + COMMISSION + FUNDING_FEE` ✓；**未知类别为负数即拒绝**（不得静默忽略借方 ✓ 探针实证）；转账等未知正项不计入（不得掩盖亏损 ✓）；分页 1000/页、最多 30 页、毫秒**重叠读 + `tranId:incomeType` 去重**、游标未推进（同毫秒 1000 条）即拒绝、非 USDT 资产拒绝 ✓；`Reconciled` 始终 false、`UnknownOrders` 始终 true、不填高水位 ✓。

### 3.4 `service/leadaccount/executor.go`（185）
- 构造函数：强制 lead 客户端 + 白名单；`NewAccountExecutor` 后校验 `Ownership.AccountID == "lead"` 且 **Broker 绑定的就是同一 client 指针** ✓（无跨账户借壳）；构造期不访问网络/DB ✓。
- `checkOpenLocked`：`pendingReconcile` → 拒 ✓；**状态门禁先于任何网络请求**（未授权时零 SAPI//fapi/exchangeInfo 调用 ✓）；随后**自读**规则与证据并覆盖外部声明的额度/杠杆/已验证布尔 ✓（不能靠传入布尔跳过检查 ✓）；`RiskDecision` 不允许即拒 ✓。
- `ExecuteOpen`：实例互斥串行 ✓；意图校验（`IntentOpen`+`auto_strategy`+MARKET/LIMIT）✓；**与风控请求逐字段对齐**（symbol/方向/类型/数量精确相等、LIMIT 价相等、MARKET 价必须为 0 ✓ → 消除参数漂移/TOCTOU）；再执行一次**新鲜**风控核验；通过后才置 `pendingReconcile=true` 并委托 Ownership 执行 ✓（**先预留后写** ✓，且成功也保持阻塞 ✓）。

### 3.5 `service/leadaccount/close.go`（77）与 `execution.go`（68）
- 平仓：仅 `auto_strategy` 的 CLOSE/STOP/TP + MARKET/LIMIT/STOP_MARKET/TAKE_PROFIT_MARKET ✓；方向必须反向（LONG→SELL、SHORT→BUY）✓；Ownership 记录须属于 lead 且 `ManagedQty>0` ✓；**实时 lead 持仓夹取** ✓；置 `pendingReconcile` 后执行 ✓。
- `ReadOnlyOrderBroker`：`Submit`/`Cancel` **永远** `ErrLiveExecutionLocked`（即使 Lead 客户端有效 ✓ 探针实证）；`Lookup` 走账户绑定 broker ✓；`NewReadOnlyOrderBroker(main)` 被拒 ✓；`NewReadOnlyExecutor` 组合 lead Ownership + 只读 broker → 一个"直接调用也无法下单"的执行器 ✓。

---

## 4. 本轮实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-S4-3-1** 平仓夹取精确边界（5 子例） | managed<live 取 managed、live<managed 取 live；恰好等于边界放行、超 1e-6 拒绝；live=0 拒绝 | ✅ 恰好边界：放行且 broker **提交 1 次**；越界：拒绝且**提交 0 次** |
| **PR-S4-3-2** 适配器公开面 | 反射枚举 `LeadExecutionAdapter` 方法 | ✅ 恰为 `{CheckOpen, ExecuteManagedClose, ExecuteOpen}` —— 无 authorize/enable/unlock/reset |
| **PR-S4-3-3** 先预留后写 / pendingReconcile | broker 本地失败后 | ✅ 提交 1 次、executor 报"uncertain 需 reconcile"、`pendingReconcile=true`、`CheckOpen` → `ErrLeadPendingReconcile`、**再次 `ExecuteOpen` 被拒且 0 额外提交** ✓（记录行为，含 F1 发现）|
| **PR-S4-3-4** PnL 未知类别 | 未知**借方**类别 / 未知正项（含转账） | ✅ 未知借方 → 整份快照拒绝；转账与未知正项被排除（净 PnL 仍为 -4 ✓） |
| **PR-S4-3-5** 市价挂单无价 | 待成交 MARKET 单 `price=0` | ✅ 整份快照拒绝（保守规则，代码注释已说明） |
| **PR-S4-3-6** 只读 broker | 有效 Lead 客户端下的 Submit/Cancel；main 绑定 | ✅ 写操作恒 `ErrLiveExecutionLocked`；main 绑定被拒 |
| 生产接线 | `git grep NewLeadExecutionAdapter(\|NewReadOnlyExecutor(\|ExecuteOpen(\|ExecuteManagedClose(` 于非测试代码 | ✅ **0 命中** → 整层执行代码在生产不可达 |

> 既有永久测试（executor 7 / close 4 / snapshot 4 / rules 3 / account_client 4-3 用例）覆盖：默认 0 API/0 真实写、Main Key 拒绝、隔离 SQLite + Fake Broker 下 LONG/SHORT × MARKET/LIMIT、OrderClaim/累计成交、未知提交、**并发 8 次仅 1 次成功**、方向/账户/参数错配拒绝、安全平仓与人工减仓限制、余额/分页去重/同毫秒拒绝、过滤器与 CROSS/CROSSED/空过滤器 fail-closed、以及所有 Lead 专属 GET 的签名与只读性。**本轮探针补的是**：平仓夹取的**精确边界**、适配器**导出面**、`pendingReconcile` 的持久性、PnL **未知类别**规则、市价挂单无价的整体拒绝、只读 broker 恒锁（均无永久测试）。

---

## 5. Gate 4 相关结论

- **mock 覆盖**：LONG/SHORT、MARKET/LIMIT、CLOSE/STOP/TP、未知提交、部分成交（Stage 2 累计成交）、保证金不足、精度/步长/tick 与杠杆档位拒绝、并发只成功一次 ✓（永久测试 + 本轮探针）。**未覆盖**（按记录属 Stage 4-4）：真实拒单/撤单与 Algo STOP/TP 恢复。
- **无可绕过风控的带单写入口**：✅（见 §1 四层）。
- **真单仍需 Stage 7 人工授权**：✅ 无任何生产授权机制。

---

## 6. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race ./service/leadaccount ./feature ./feature/api/binance ./scanner ./service/futuresownership -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` | ✅ 全部 ok（leadaccount 1.56s、feature 1.63s、api/binance 1.42s、scanner 2.16s、ownership 1.99s） |
| `go vet ./service/leadaccount ./feature ./feature/api/binance` | ✅ 干净 |
| 根包 `go build -o /tmp/s43_bin .` | ✅ 成功（未在仓库留产物） |
| 探针 PR-S4-3-1～6 | ✅ 全部通过（含 `-race`）；副本去掉探针后原测试 ok |
| Schema / 迁移 / 路由 / 配置 | ✅ 无变更（仍 v19），与记录 §5 一致 |
| 真实 Binance / 真实 Lead Key / 真实资金 | ❌ 未执行（Gate 0-LIVE 与 Stage 7 授权前均不执行） |

---

## 7. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | P3（错误粒度） | **写路径把一切预写拒绝压成同一个错误**：`executor.go:165-167` 丢弃 `checkOpenLocked` 的具体错误，统一返回 `ErrLeadOpenBlocked`；而 `CheckOpen`（诊断）会保留具体原因（如 `ErrLeadPendingReconcile`）。调用方（Stage 3 的 `riskAllowsOpen` 也只记固定 warning）无法区分"待对账"/"风控拒绝"/"未授权" | 探针 PR-S4-3-3 实证。建议：`ExecuteOpen`/`ExecuteManagedClose` 返回**包装后的具体错误**（至少让 `errors.Is(err, ErrLeadPendingReconcile)` 可分辨），并让 runner 记录错误类别（呼应 4-2 报告 F3） |
| **F2** | P3（可用性） | `pendingReconcile` 在**本地失败**（broker 本地错误、DB 错误）后同样永久生效且**无公开解除入口** → 该实例此后一直拒绝新开仓与平仓 | 记录已声明"Stage 4-3 无公开安全重置、Stage 5 负责权威恢复"✓。建议 Stage 5 提供**按账户事务性**的恢复流程，并在拒绝原因中体现"待对账"（与 F1 联动） |
| **F3** | P3（保守性副作用） | 待成交 **MARKET 挂单 price=0** 会让**整份快照失效**（进而全线停开新仓），代码注释已说明"无法验证参考价时不得按 0 计算名义额" | 建议 Stage 5 用该币 mark price/最近参考价计价，避免正常市价挂单造成长时间无法开仓 |
| **F4** | P3（部署前置） | Lead 开仓要求账户**当前**杠杆与保证金模式已与请求一致（否则拒绝且**不自动切换** ✓ 安全），意味着 Stage 6 之前必须由人工或显式流程预置该币杠杆/逐仓 | 建议在文档/Stage 6 UI 中把"预置杠杆与逐仓"列为启用前置检查项，避免上线后"全被拒但不知原因"（与 F1 联动） |
| **F5** | P3（覆盖） | 平仓路径**不**重校验 tick/step/minNotional（依赖交易所拒单）；对 reduce-only 平仓风险低 | 建议 Stage 4-4 视需要为平仓补最小规则校验 |
| **F6** | P3（沿用） | 4-2 的 F1（`api_order_budget` 由 Stage 1 限流承担）、F2（`MaxPositions/MaxLosingPositions` 必配但原因泛化）未处理 | 非本阶段问题，建议一并在文档/原因码层面收口 |
| — | 正面 | ① **意图对齐**（数量/价格精确相等 + MARKET 价必须为 0）消除了"风控按 A 参数评估、下单用 B 参数"的 TOCTOU；② `ReadOnlyOrderBroker` 提供恢复通道同时**结构性禁止**写；③ PnL 采用"重叠分页 + 事务 ID 去重 + 同毫秒不可证即拒 + 未知借方拒绝"四重保守；④ 快照恒 `Reconciled=false`/`UnknownOrders=true` 使 Stage 4-3 即使被误接线也无法开仓 ✓ |

---

## 8. 审计边界与未验证项

- **未验证**：真实 Lead Portfolio 的余额/仓位/挂单/收入分页/杠杆档位返回形态（全部以 Fake/Mock 验证；真实格式差异交 Gate 0-LIVE 与 Stage 7）。
- **未验证**：真实拒单、撤单、Algo STOP/TP 触发与恢复（Stage 4-4）。
- **未验证**：跨进程/重启的额度与待对账恢复（Stage 5，须 DB 事务/唯一键）。
- **未验证**：前端与 MySQL（无变更）。
- 本审计未发起任何真实 Binance 请求、未改动用户仓库任何文件（探针仅存在于已删除的 `/tmp` 副本）。

---

## 9. 附：Stage 4-3 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `service/leadaccount/executor.go` | 185（新） | 账户绑定执行适配器：状态门禁 → 自读规则/证据 → 风控 → 先预留后写；公开面仅三个方法 |
| `service/leadaccount/close.go` | 77（新） | 受控平仓：意图/方向/归属校验 + `min(managed, live)` 精确夹取 |
| `service/leadaccount/execution.go` | 68（新） | `ErrLiveExecutionLocked`、`ReadOnlyOrderBroker`（写恒锁、查可用）、`NewReadOnlyExecutor` |
| `service/leadaccount/snapshot.go` | 240（新） | 风险证据采集：余额/持仓/挂单/Hedge/UTC 日净 PnL（重叠分页去重、未知借方拒绝、同毫秒拒绝） |
| `service/leadaccount/rules.go` / `rules_source.go` | 155 / 62（新） | exchangeInfo 过滤器 + 签名杠杆档位 + 账户现状一致性校验（不自动切换模式） |
| `feature/api/binance/lead_income.go` / `lead_rules.go` | 23 / 28（新） | Lead 专属只读 GET：收入历史（窗口/分页/lead-only）、exchangeInfo 与签名 leverageBracket |
| 测试 | executor 7 / close 4 / snapshot 4 / rules 3 / 4-3 SDK 用例 | 见 §4 说明 |
| 文档 | — | `币安合约自动带单-Stage4-3-实施记录.md`；方案文档 §4.3 追加实施结果 |

**建议下一步**：① 处理 **F1**（写路径错误粒度）与 **F2**（本地失败导致的持久阻塞）——两者影响 Stage 5/6 的运维可诊断性；② 进入 **Stage 4-4**（拒单/撤单/Algo STOP/TP 的异常与恢复、真实只读查单验证），继续保持 `enabled=false`、无真单；③ Stage 5 需落实：权威对账置真 `Reconciled`/清 `UnknownOrders`、按账户事务性恢复 pending 占用、市价挂单计价、per-account 预算与告警；④ 真实只读/写能力仍等 **Gate 0-LIVE** 与 Stage 7 人工授权。
