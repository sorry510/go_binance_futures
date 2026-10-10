# 币安合约自动带单 — Stage 4-4 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage4-4-实施记录.md` 对应的 **Stage 4-4 异常订单、Algo 只读恢复与平仓过滤器**（工作区未提交改动）。基线 `HEAD = 3201634`（`feat: stage3`，分支 `feat/lead-trading`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库与 `strategy_templates/` 未被改动）
- **审查类型**：review-only
- **结论**：**Stage 4-4 完成**（离线只读恢复证据 + 平仓过滤器，零生产接线）。**我上一轮（Stage 4-3）的 F1（写路径错误粒度）与 F5（平仓不重校验交易所规则）均已修复**；F2（`pendingReconcile` 无重置入口）属**有意保留**（记录 §4.4 已声明，Stage 5 提供权威恢复）。未发现 P0/P1；新增 4 项 P3。
- **审计日期**：2026-10-10

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| 上轮 Stage 4-3 **F1**（写路径把一切预写拒绝压成 `ErrLeadOpenBlocked`） | ✅ **已修复**：`executor.go:166-168` 改为 `fmt.Errorf("%w: %w", ErrLeadOpenBlocked, err)` → `errors.Is` 同时可判 `ErrLeadOpenBlocked` 与底层类别（如 `ErrLeadPendingReconcile`）；永久测试同步断言两者 ✓ |
| 上轮 Stage 4-3 **F5**（平仓不校验 tick/step/minNotional） | ✅ **已修复**：新增 `close_rules.go` 并在 `close.go` 的 `pendingReconcile=true`/`ownership.Execute` **之前**调用（`close.go:82`）✓ |
| 上轮 Stage 4-3 **F2**（`pendingReconcile` 本地失败后持久阻塞且无解除） | 🔶 **有意保留**：记录 §4.4 明确"`pendingReconcile` 无重置方法"，权威恢复交 Stage 5；本轮探针再次确认**恢复接口不能清除**它 ✓ |
| 恢复证据是否只读 | ✅ `recovery.go`/`close_rules.go` **零写调用**（无 `CreateOwnedOrder`/`CancelOrder`/`SetLeverage`/`SetMarginType`）✓；Mock HTTP 测试断言 GET-only、Lead Key 签名、无 POST/DELETE ✓ |
| 生产接线 | ✅ 零：`InspectOrderRecovery`/`ValidateLeadCloseRules`/`verifyCloseRules` 在非测试代码 **0 命中**；`RiskController` 仍无 Enable/Resume ✓ |
| 端点行为 | ✅ **恢复不释放任何阻塞**：终态证据仍 `RequiresReconcile=true`，不重复发单、不清除异常（记录 §结论）✓ 探针实证 ✓ |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 记录条目核对

| 记录要求 | 判定 | 证据 |
| --- | --- | --- |
| §1 `InspectOrderRecovery` 仅用绑定 Lead 客户端的签名 GET，支持普通 MARKET/LIMIT 与 5 种 Algo 类型，**无任何 mutation** | ✅ | `recovery.go:30-62`；非 algo 且非 MARKET/LIMIT → 直接 uncertain ✓；Algo 类型集合 `{STOP, TAKE_PROFIT, STOP_MARKET, TAKE_PROFIT_MARKET, TRAILING_STOP_MARKET}` ✓ |
| §1 普通订单核对 Symbol/ClientOrderID/OrderType/交易所 OrderID；**累计成交量与均价强制解析，不得把坏数据当 0** | ✅ | `verifiedLeadOrder`：`OrderID>0`、`Symbol`、`ClientOrderID` 必须相等；`ExecutedQuantity`/`AvgPrice` 解析失败即 uncertain ✓（无 `_ =` 忽略） |
| §1 Algo 先核对 ClientAlgoID/币种/类型/方向/仓位方向；有 `actualOrderId` 时以**同一 Lead 身份**查实际单并核对 ID/币种/方向/仓位方向；**无实际单的 FINISHED 不算成交** | ✅ | `inspectAlgoRecovery`：全部字段核对 ✓；`actualID` 为空或 "0" 时只按 Algo 回执分类（FINISHED 不在已知状态集 → `ErrLeadRecoveryUncertain` ✓）；有实际单时再 `GetOrderByOrderID` 并校验 `OrderID/Symbol/Side/PositionSide` 与 Algo 一致 ✓，且把证据的 ClientOrderID/OrderID 还原为 **Algo 的**（与 Ownership 认领口径一致 ✓） |
| §1 `ClassifyLeadRecovery` 显式识别终态/部分成交/中间态；FILLED 必须有正成交量、REJECTED 不可有成交；未知状态/查单超时/错 ID/异常成交 → `ErrLeadRecoveryUncertain` | ✅ **探针矩阵实证** | 终态集 `{FILLED, CANCELED, CANCELLED, EXPIRED, EXPIRED_IN_MATCH, REJECTED}`；中间态 `{NEW, PARTIALLY_FILLED, PENDING_NEW, PENDING_CANCEL, TRIGGERED, NOT_TRIGGERED, WORKING}`；一致性：FILLED⇒成交>0、REJECTED⇒成交=0、PARTIALLY_FILLED⇒成交>0、NEW⇒成交=0，任一矛盾即 uncertain ✓ |
| §1 终态也一律 `RequiresReconcile=true`，不得因撤单/拒单/Algo FINISHED 断言已同步 | ✅ **实证** | `RecoveryEvidence{RequiresReconcile: true}` 在所有返回路径保持 true（探针 18 个状态 + 5 个非法载荷全部断言）✓ |
| §2 新增 `close_rules.go` 并在受控平仓执行前调用；顺序：状态/Ownership 绑定 → Fresh 仓位与 Mark Price → exchangeInfo 过滤器 → `pendingReconcile=true` → Executor | ✅ | `close.go:55`（Fresh 仓位）→ `:69`（Mark Price 解析）→ `:82 verifyCloseRules(ctx, request, markPrice, min(managed, live), live)` → `:87 pendingReconcile=true` → `:88 ownership.Execute` ✓ |
| §2 校验数量 min/max/step、LIMIT 价格精度、**Algo STOP/TP 的 TriggerPrice 精度**；过滤器缺失/触发价缺失或错误/Mark Price 异常一律拒绝 | ✅ **探针矩阵实证** | `ValidateLeadCloseRules`：类型-价格组合严格（MARKET 价与触发价必须为 0；LIMIT 须正价且无触发价；STOP_MARKET/TAKE_PROFIT_MARKET 须正触发价且无价格）✓；三类过滤器缺一即拒 ✓；市价执行用 `MARKET_LOT_SIZE`、限价用 `LOT_SIZE` ✓（探针确认市价单**不要求** LOT_SIZE）|
| §2 minNotional **只允许平掉真实 Lead 全部该方向仓位**时使用完整关闭例外；只平 managed 部分且有人工加仓时**不能**借用 | ✅ **实证** | 判定为 `qty*effectivePrice < minNotional && qty < fullLiveQty-1e-10 → 拒绝`；探针：全平低于 minNotional → 放行；部分低于 minNotional → 拒绝；**人工加仓（live > cap）时例外不可达**（cap 已限制请求量）✓ |
| §2 不自动更改杠杆/逐仓/全仓、不读 Main 私有数据 | ✅ | `close_rules.go` 无任何写调用 ✓；规则仅来自 exchangeInfo + Lead 客户端 ✓ |
| §4 明确未完成 Gate（Stage 5 权威持久恢复、真实撤单、动态过滤器待实测、无授权路径） | ✅ 与代码一致 | `pendingReconcile` 无重置方法（探针反射断言公开面仅 4 方法）✓；无 Cancel mutation ✓ |

---

## 3. 本轮实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-S4-4-1** 恢复分类矩阵 | 18 个状态 + 5 个非法载荷（NaN/Inf/负成交/空 OrderID/错 ClientOrderID） | ✅ 终态/中间态判定、四类一致性约束、未知状态拒绝全部符合；`RequiresReconcile` **恒为 true** |
| **PR-S4-4-2** 未知类型/空标识 | `orderType="SOMETHING"`、空 symbol | ✅ 均为 `ErrLeadRecoveryUncertain`，且**不触发任何查单** |
| **PR-S4-4-3** 平仓规则边界矩阵 | 16 个用例（含 minNotional 全平例外、人工加仓不可借用、数量越界/离步长、限价离 tick、类型-价格组合、不支持类型）+ 4 个缺失过滤器 + 市价单不依赖 LOT_SIZE | ✅ 全部符合预期 |
| **PR-S4-4-4** 公开面 / 恢复不能解锁 | 反射枚举 + 交叉调用 | ✅ 公开面恰为 `{CheckOpen, ExecuteManagedClose, ExecuteOpen, InspectOrderRecovery}`；恢复路径**不能**清除 `pendingReconcile` ✓ |

> 既有永久测试（`recovery_test.go` 分类、`recovery_http_test.go` Mock HTTP 覆盖取消后部分成交/拒单/异常累计成交/错 ClientOrderID/Algo 未触发-已取消-无实际单 FINISHED/触发后部分与完全成交/实际单错 symbol/504 超时、`recovery_ledger_test.go` 幂等累计成交、`close_rules_test.go` 精度与全平例外）覆盖了主路径。**本轮探针补的是**：完整状态矩阵与非法载荷、未知类型不查单、平仓规则的**精确边界**与"市价单不要求 LOT_SIZE"、以及公开面/不可解锁断言（均无永久测试）。

---

## 4. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race ./service/leadaccount ./feature ./feature/api/binance ./scanner ./service/futuresownership -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` | ✅ 全部 ok（leadaccount 1.52s、feature 2.22s、api/binance 3.24s、scanner 2.46s、ownership 3.46s） |
| `go vet ./service/leadaccount ./feature ./feature/api/binance` | ✅ 干净 |
| 根包 `go build -o /tmp/s44_bin .` | ✅ 成功（未在仓库留产物） |
| 探针 PR-S4-4-1～4 | ✅ 全部通过（含 `-race`）；副本去掉探针后原测试 ok |
| Schema / 迁移 / 路由 / 配置 | ✅ 无变更（仍 v19）；`feature/*` 仍为 4-2/4-3 的 38+9 行接线 |
| 真实 Binance / 真实 Lead Key / 真实撤单 | ❌ 未执行（Gate 0-LIVE 与 Stage 7 授权前均不执行） |

---

## 5. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | P3（落库缺口） | 恢复证据**仅存在于内存返回结构**，本阶段没有把它写入 DB 的路径（`recovery_ledger_test.go` 只演示"原 Ownership 累计成交幂等"能力） | 记录 §4.1 已将权威持久恢复交 Stage 5 ✓。建议 Stage 5 明确：`account_id + client_order_id` 唯一键 + 事务内"先查后写"，并禁止把 `InspectOrderRecovery` 的 `Terminal` 当作释放 `pendingReconcile` 的依据（记录已声明 ✓） |
| **F2** | P3（保守性可能过强） | `AlgoStatus=FINISHED` 且无 `actualOrderId` 一律 uncertain；若某交易所状态组合确实表示"未触发即取消"，会永久 uncertain（保守但可能长期卡住） | 建议 Stage 5 结合 Algo 状态与 User Data WS 事件复核（记录已交 Stage 5 ✓）；必要时在诊断中区分"Algo 已完成但无实际单"与"状态不可识别"两类原因 |
| **F3** | P3（调用成本） | 每次平仓校验都会**重新拉取 exchangeInfo**（公共 GET，权重 1）；与 V4-5 已为 futures/spot 建立的 exchangeInfo 长 TTL 缓存未打通 | 建议 Stage 5 复用既有长 TTL 交换信息缓存（或给 Lead 规则源加 TTL），在不影响正确性的前提下省掉每单一次的公共请求 |
| **F4** | P3（待实测） | 平仓规则未覆盖 `PERCENT_PRICE` 等**动态过滤器**；minNotional 例外按 `qty*effectivePrice` 判定，未考虑交易所其它动态约束 | 记录 §4.3 已列为 Gate 0-LIVE/Stage 7 待实测项 ✓ 建议届时补最小额真实只读/最小额验证 |
| F5 | P3（沿用） | 4-2 的 F1（`api_order_budget` 由 Stage 1 限流承担）、F2（`MaxPositions/MaxLosingPositions` 必配但原因泛化）；4-3 的 F3（市价挂单 price=0 使整份快照失效）、F4（开仓要求账户当前杠杆/逐仓已一致） | 均未处理，建议随 Stage 5/6 一并收口 |
| — | 正面 | ① **Algo 触发链核对**（先 Algo 回执、再同身份查实际单、再校验实际单的方向与仓位方向、最后以实际单成交量为准）设计完整，避免"Algo FINISHED 即视为成交"的常见误判；② `fmt.Errorf("%w: %w", …)` 双层包装使既保留安全哨兵又可 `errors.Is` 分辨底层类别；③ 平仓过滤器把"数量越界 + 规则校验"都放在 `pendingReconcile` 与写之前 ✓；④ 恢复全程只读且**不解除任何阻塞**，符合"先对账再放行"的原则 |

---

## 6. 审计边界与未验证项

- **未验证**：真实 Binance 的订单/Algo 状态字符串全集与 `EXPIRED_IN_MATCH` 等边界（以 Mock HTTP 覆盖常见集合；全集交 Gate 0-LIVE/Stage 7）。
- **未验证**：真实撤单、真实恢复自动化与重启后恢复（Stage 5，须 DB 唯一键/事务）。
- **未验证**：`PERCENT_PRICE` 等动态过滤器与真实 minNotional 例外行为（Stage 7 实测）。
- **未验证**：前端与 MySQL（无变更）。
- 本审计未发起任何真实 Binance 请求、未改动用户仓库任何文件（探针仅存在于已删除的 `/tmp` 副本）。

---

## 7. 附：Stage 4-4 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `service/leadaccount/recovery.go` | 166（新） | Lead 专属只读异常订单恢复证据：普通单与 Algo 触发链核对、严格成交解析、`ClassifyLeadRecovery` 终态/中间态判定、恒 `RequiresReconcile` |
| `service/leadaccount/close_rules.go` | 126（新） | 受控平仓交易所规则校验（LOT_SIZE/MARKET_LOT_SIZE、PRICE_FILTER、MIN_NOTIONAL/NOTIONAL、STOP/TP 触发价精度）+ minNotional 全平例外 |
| `service/leadaccount/close.go` | 接线 | 在 `pendingReconcile` 与写之前调用 `verifyCloseRules`（cap=min(managed,live)、liveQty=实时全量） |
| `service/leadaccount/executor.go` | 错误粒度修复 | `fmt.Errorf("%w: %w", ErrLeadOpenBlocked, err)` 保留底层类别 |
| 测试 | `recovery_test.go` / `recovery_http_test.go`(141) / `recovery_ledger_test.go`(52) / `close_rules_test.go`(65) | 见 §3 说明 |
| 文档 | — | `币安合约自动带单-Stage4-4-实施记录.md`；方案文档 §4.4 追加实施结果 |

**建议下一步**：① Stage 5 落实**权威持久恢复**（`account_id` 唯一键 + 事务、全部未确认订单与 live 仓位核实后方可释放 `pendingReconcile`），并把 Lead 规则源的 exchangeInfo 与既有长 TTL 缓存打通（F3）；② 处理 4-3 遗留的 F3/F4（市价挂单计价、启用前置杠杆/逐仓）与 4-2 的 F1/F2（预算归属、原因码细化）；③ Stage 5 还需：Lead 私有 UserData WS、启动/重启恢复、per-account API 预算与告警；④ 真实只读/写能力仍等 **Gate 0-LIVE** 与 Stage 7 人工授权，本阶段结论不得当作交易许可。
