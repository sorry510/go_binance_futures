# 币安合约自动带单 — Stage 1 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage实施方案.md` 的 **Stage 1 — 账户对象和独立 Binance Client**（工作区未提交改动）。基线 `HEAD = 287f956`（分支 `feat/lead-trading`，`feat: stage0`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库与 `strategy_templates/` 未被改动）
- **审查类型**：review-only
- **结论**：**Stage 1「离线 Gate」基本通过**（账户隔离的设计与实现扎实，Main 路径零改动）；**但有两项需要处理/确认**：① **Gate 1 字面要求的「429 不串账户」在"可用性"层面未满足**（一个账户收到 429 会经共享 IP 预算阻断另一账户，实测复现）；② 计划条目 4（Lead Credential Provider）实际未实现（已由实施记录显式延后到 Stage 6）。两项均**不阻塞**进入 Stage 2 开发，但**必须先就 429 语义做决策**，且真实带单下单仍被 Gate 0-LIVE 阻塞。
- **审计日期**：2026-10-09

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| 计划 Stage 1 条目 1–6 | ✅ 5 项达成/部分达成（条目 1 部分：缺 WSState/CircuitState）；❌ 条目 4（Credential Provider）未实现，记录已声明延后 |
| Gate 1「注入两个 fake 客户端，交叉签名请求、cache、时钟重同步及 429 均不串账户」 | 🔶 **签名/缓存/时钟重同步三项已由永久测试覆盖并通过；429 项在"密钥、状态、归因"层面隔离正确，但在"可用性"层面会跨账户阻断（见 F1）** |
| main 行为与测试 | ✅ 未触碰 Main 旧入口/旧缓存；`feature/api/binance`、`service/futuresownership` 测试与 `-race` 全绿 |
| 是否调用真实下单 | ✅ **无**：`NewLeadAccountClient` 无生产调用方；`BinanceOrderBroker{Account:...}` 仅出现在测试；无 DB 写入、无 app.conf 读取 |
| 是否可进入 Stage 2 | ✅ 可以（Stage 2 为 DB/account_id 迁移，属纯离线开发）；**但真实带单下单仍须等 Gate 0-LIVE** |
| Binance 真实资金操作 | ❌ 全程未发生（本审计亦未发起任何真实请求） |

---

## 2. 计划条目逐项核对

| # | 计划要求（Stage 1 §95-107） | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | 账户标识 `main`/`lead` 与 `AccountContext`（ID、kind、Broker、Cache、WSState、Limits、CircuitState；绝不暴露 Secret） | 🔶 **部分** | `AccountID`/`MainAccountID`/`LeadAccountID` ✓、`AccountClient`（id + SDK client + 签名互斥 + 账户缓存 + 配置缓存）✓、`leadOrderLimiter`（Limits）✓；**缺** `kind`、`Broker`、`WSState`、`CircuitState` 的显式建模（WS 状态/熔断按记录属 Stage 5）。结构中无 Secret 字段 ✓（密钥仅存在于 SDK client 内部） |
| 2 | `futuresClient` 改为实例化客户端；时间偏移锁、重试、http client、testnet/baseURL、proxy、api usage source、交易规则缓存全部账户隔离 | ✅ | `NewAccountClient`/`NewLeadAccountClient`（`account_client.go:43-71`）；`doAccountSigned` 用账户自有 `signedMu` 串行化签名调用并在 `-1021` 时**只改本 client** 的 `TimeOffset` 后重试一次（`:73-99`，永久测试 `TestStage1AccountTimestampResyncDoesNotChangeOtherAccount`）；Lead 固定 `https://fapi.binance.com`、经 `proxyPool.HTTPClient()`、`binanceapiusage` source=`lead_trading`（`:56-69`）；交易规则缓存为账户私有 `tradeConfigState`（`account_state.go:133-187`）；仓位/挂单缓存为账户私有 `accountReadState`（含 singleflight + generation 防回写，`:18-131`） |
| 3 | 保留 main 兼容包装；新增 lead 入口必须显式接受 AccountContext；行情不按账户重复抓取 | ✅ | Main 旧入口/旧缓存**完全未改**（本阶段仅改 `service/futuresownership/execution.go`）；`NewAccountClient` 拒绝未知 ID 与 nil client（`account_client.go:43-48`）→ 不存在"静默回落 main"；新增文件中**无** ticker/kline/depth 抓取（行情仍走公共路径）✓ |
| 4 | Lead Credential Provider（安全加密、IP 白名单提示、凭证轮换、测试连接、只读状态；内存不打印密钥） | ❌ **未实现**（记录已声明延后） | 仅提供"传入凭证的客户端工厂"`NewLeadAccountClient(apiKey, apiSecret, httpClient)` + SAPI 只读状态（`LeadTraderStatus`/`LeadTradingSymbols`→"测试连接/只读状态"部分等价）；**加密存储/轮换/IP 提示不在本阶段**（实施记录 §4 明确交 Stage 6、附录 A 把凭证表放在 Stage 2/4）。建议在计划文档中把条目 4 显式标注为跨阶段项，避免后续验收歧义 |
| 5 | 私有 REST 缓存按 `account_id` 分区，mutation 只失效对应账户，不跨账户使用镜像/余额缓存 | ✅ **实证** | 缓存为 `AccountClient` 内的私有字段 ✓；mutation（下单/Algo/撤单）调用 `InvalidateAccountReadCache()`，杠杆/逐仓仅在成功时失效并同时失效该币配置缓存（`account_client.go:185/199/211/218/225-238`）；永久测试 `TestStage1AccountIsolationOfSignedRequestsAndCaches`（失效 Lead 缓存后 main 计数不变、Lead+1）与 `TestStage1TradeConfigCacheNeverSharesAccounts` ✓ |
| 6 | 限速按 API Key + IP 双约束；lead 订单限速从 20/10s 起 | ✅ **实证** | `leadOrderLimiter`：仅统计 POST/DELETE 且路径以 `/fapi/v1/order`、`/fapi/v1/algoOrder` 开头者，滑动 10s 窗口上限 **18**（20 预留 2），超限在 **base RoundTripper 之前**返回 `binanceapiusage.ErrBudgetDeferred`（`lead_order_limiter.go:20-44`）；V4-5 全局 IP/weight 预算继续在外层生效（`account_client.go:69` 包装顺序：预算外层 → 限额外层 → 真实传输）。**探针 PR-S1-2 实证**：前 18 次放行、第 19 次被拒且**零网络调用**、读请求不计数、第二个 limiter 预算独立 ✓ |

**Gate 1 逐项**：

| Gate 子项 | 判定 | 证据 |
| --- | --- | --- |
| 交叉**签名请求**不串账户 | ✅ | 永久测试：测试服务器按 `X-MBX-APIKEY` 分别计数（main-key/lead-key），断言各自 1 次；**探针**：阻断场景下 main 请求从未使用 lead key（`mainCalls==0`） |
| **cache** 不串账户 | ✅ | 永久测试（失效隔离）+ 探针（每账户只用自己的 key 与缓存路径） |
| **时钟重同步**不串账户 | ✅ | 永久测试 `TestStage1AccountTimestampResyncDoesNotChangeOtherAccount`（只改本 client `TimeOffset`） |
| **429** 不串账户 | 🔶 **部分（见 F1）** | 密钥/状态/归因隔离 ✓（**探针**：429 记在 `source=lead_trading`、`Retry-After=1`、main 的 429 计数为 0、main 未出现在限流事件中）；**但** 共享 IP 预算把 `futures|mainnet` 置为 throttled → **main 的下一次请求被 `ErrBudgetDeferred` 拒绝**（探针实测，1.2s 后恢复） |
| main 行为和测试通过 | ✅ | `go test -count=1 ./feature/api/binance ./service/futuresownership ./service/binanceapiusage ./binanceproxy` 全 ok；`-race` 上述前两个包 ok；Stage 1 包 `go vet` 干净 |
| 不调用真实下单 | ✅ | `NewLeadAccountClient` 无生产调用方；`BinanceOrderBroker{}` 在生产路径仅两处未绑定实例（`execution.go:55`、`reconcile.go:42`）；无 DB/配置写入 |

---

## 3. 本阶段新增/修改代码审计要点

**`account_client.go`（262 行，新）**
- 结构清洁：无全局注册、无启动钩子、无 DB 访问、无真实下单（与文件注释一致）✓。
- `doAccountSigned[T]` 泛型封装统一承担：nil 保护、签名串行化、`-1021` 单次重同步+重试 ✓；重同步失败返回带账户名的错误（不含 URL/签名）✓。
- 私有 API 覆盖 Stage 1 所需集合：账户、仓位（含 fresh/按币直通）、挂单（同）、按 ClientOrderID/OrderID 查询、普通与 Algo 下单、Algo 查询/撤单、撤单、杠杆、逐仓、`EnsureTradeConfigContext`、listenKey 获取/续期 ✓。
- **下单参数格式化复用** `formatOwnedOrderDecimal` ✓（与 Main 同一份实现，避免精度分叉）。

**`account_state.go`（187 行，新）**
- 与 V4-5 主账户缓存同构但**实例私有**：`valid` 标记（空快照也可缓存，规避了 V4-5 复评 F1 那类缺陷）✓、singleflight、generation 防"失效前在途回写"✓、读/写均 clone ✓。
- `tradeConfigState.ensure` 复用 Main 的语义：`-4046` 视为已满足、失败不缓存、成功后 5s TTL 去重 ✓；失败时记录 `account_config_error` 优化计数 ✓。
- nil map 安全（`delete`/读取均安全）✓。

**`lead_readonly.go`（108 行，新）**
- 仅 `AccountID=lead` 可调用（`a.id != LeadAccountID` → 直接拒绝）✓；签名自实现（timestamp 使用**本账户** TimeOffset、`recvWindow=10000`、HMAC-SHA256、`X-MBX-APIKEY`）✓。
- **错误信息脱敏**：网络失败/非 200/解码失败/业务码失败统一为不含 URL、签名、正文的短错误 ✓（满足计划 §0.3"密钥绝不进入日志"）。
- 业务码 fail-closed：`userStatus` 必须 `code=000000 && success`；`leadSymbol` 必须 `code=000000`（`success` 存在时必须为 true），`data=nil` 返回**非 nil 空切片** ✓。

**`lead_order_limiter.go`（44 行，新）**：见上文条目 6 ✓；返回的 `ErrBudgetDeferred` 与 V4-5 同一 sentinel → **沿用既有"发送前拒绝"语义**（Executor 会置订单 failed 且不进入 lookup/reconcile）✓ 这是跨阶段复用得当的一点。

**`service/futuresownership/execution.go`（+60/−12）**
- `BinanceOrderBroker` 增加可选 `Account`；7 个内部辅助函数"绑定即走账户、未绑定走原全局函数" ✓ → **nil Account 时行为与改动前逐行等价** ✓。
- Algo 查询把 lookup 函数下沉（`exchangeFromAlgoLookupWithOrderLookup`），保证 **Algo 触发后查询实际订单**也走同一账户 ✓（永久测试 `TestStage1AlgoLookupUsesBoundAccountForTriggeredOrder`）。

**安全边界复测**：4 个新文件**无任何日志输出**、**不读取 app.conf**（`git grep` 0 命中）✓；未修改 `models/`、`conf/`、`feature/feature.go`、选币/策略/Ownership 模型、前端 ✓（与实施记录 §2 一致）。

---

## 4. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | **P2（Gate 字面未满足）** | **一个账户的 429 会阻断另一账户**：V4-5 预算按 `product|environment` 记录节流状态（同一 IP），因此 lead 收到 429 后 `futures|mainnet` 进入 throttled，**main 的下一次请求被 `ErrBudgetDeferred` 拒绝**（探针实测，`Retry-After=1` 期间阻断、1.2s 后恢复） | 与计划 §2.1 不变量"任一账户 API Key…失效只暂停该账户"存在张力；若 429 源于**共享 IP 权重**，共同降级反而是正确行为；若源于 **per-Key 订单限速**（Lead Key 20/10s），则不应牵连 main。建议：① 明确 Gate 措辞与设计意图（写清"IP 级 429 共享降级、Key 级 429 账户自限"）；② Stage 5 做 per-account 归因时落实区分（响应头已含 `X-MBX-USED-WEIGHT-1M` 与 order-count，可据此判定 429 属 IP 权重还是 per-Key 订单）；③ 短期可在文档中把该行为标注为"已知共享 IP 语义"，避免验收误判 |
| **F2** | P3（计划-实现偏差） | 计划条目 4（Lead Credential Provider：加密存储、轮换、IP 白名单提示）本阶段**未实现**，仅提供内存凭证工厂 + 只读状态 | 实施记录 §4 已显式声明延后（Stage 6 存储/前端、Stage 2/4 表结构）；建议同时在**计划文档**的 Stage 1 段落标注"跨阶段项"，保持计划与记录口径一致 |
| **F3** | P3（环境，非本阶段引入） | 当前工作区 `go test ./...` 与 `go vet ./...` 为**红色**：`controllers` 的 `TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals` 因 **未跟踪的** `strategy_templates/research/**/config.json` 含未知字段 `family` 而失败；另有 8 个 `strategy_templates/research/**` 包因重复 `main` 构建失败 | **与 Stage 1 无关**（Stage 1 未触碰 `strategy_templates/`；这些目录未被 git 跟踪且受计划 §0.3 保护）。影响：计划要求的"全量 `go test ./...` 通过"在当前工作副本**无法达成**。建议：① 该测试只扫描受控模板目录，或对 research 子目录排除；② 或把 research 目录移出 `strategy_templates/` 扫描范围。另：`webnotification` 在 77 包并行批量下偶发 `[setup failed]`，单独/小批量运行通过（环境抖动，非代码缺陷） |
| **F4** | P3（观察） | lead 订单限流只覆盖 `/fapi/v1/order*`、`/fapi/v1/algoOrder*`；若将来使用批量撤单（如 `/fapi/v1/allOpenOrders`）等端点，将**不计入** per-Key 订单预算 | 当前代码路径未使用该端点；建议在使用前把匹配规则扩展为"所有订单写端点" |
| **F5** | P3（已知阻塞） | SAPI 仅 Mock 验证，真实 Lead Portfolio Key 能力（身份/白名单/fapi 映射、Hedge Mode、Algo 支持）仍归 Gate 0-LIVE；Lead 未创建第二条 User Data WS、未做账户镜像 | 与记录 §4/§5 一致 ✓；Stage 2/5 继续，**真实下单仍须人工授权** |

**正面结论**：账户隔离的设计（实例私有客户端/缓存/签名锁/时间偏移、显式 ID 校验、无静默 main 回落）与实现质量良好；Main 路径零改动、真实下单零调用、密钥零日志，均与计划 §0/§2.1 的安全约束一致。

---

## 5. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 ./feature/api/binance ./service/futuresownership ./service/binanceapiusage ./binanceproxy` | ✅ 全 ok |
| `go test -count=1 -race ./feature/api/binance ./service/futuresownership` | ✅ 全 ok |
| `go test -count=1 -race ./...`（Stage 1 相关 4 包 + 全量） | ⚠️ Stage 1 包全绿；全量红点全部来自 F3（未跟踪 research 目录）+ `webnotification` 批量 setup 抖动 |
| `go vet ./feature/api/binance ./service/futuresownership ./binanceproxy` | ✅ 干净 |
| `go build -o /tmp/stage1_bin .`（根包） | ✅ 成功（38 MB 二进制，未在仓库内留产物） |
| 探针 PR-S1-1（429 隔离/归因/恢复） | ✅ 通过（含 F1 行为记录） |
| 探针 PR-S1-2（限流语义） | ✅ 通过 |
| 安全边界 `git grep`（日志/app.conf 读取） | ✅ 4 个新文件 0 命中 |

---

## 6. 审计边界与未验证项

- **未验证**：真实 Lead Portfolio Key 的任何能力（身份、白名单、下单、Algo、WS）——属 Gate 0-LIVE，需用户侧专用 Key 实测。
- **未验证**：真实代理池下的 lead 客户端行为（探针用测试服务器；包装顺序已在代码层核对）。
- **未验证**：`-1021` 重同步在真实时钟漂移下的行为（仅单测覆盖逻辑）。
- **未验证**：MySQL（本阶段无 DB 变更）。
- **未验证**：前端（本阶段无前端改动）。
- 本审计未发起任何真实 Binance 请求，未改动用户仓库任何文件（探针仅存在于已删除的 `/tmp` 副本）。

---

## 7. 附：Stage 1 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `feature/api/binance/account_client.go` | 262（新） | AccountID/AccountClient、独立 SDK 客户端、签名串行化与账户级时间重同步、私有 REST 全覆盖 |
| `feature/api/binance/account_state.go` | 187（新） | 账户私有仓位/挂单快照缓存（singleflight + generation）与交易配置去重 |
| `feature/api/binance/lead_order_limiter.go` | 44（新） | Lead Key 订单预发限流（18/10s，发送前拒绝） |
| `feature/api/binance/lead_readonly.go` | 108（新） | SAPI 用户状态/交易白名单只读签名查询（仅 lead、错误脱敏） |
| `feature/api/binance/account_client_stage1_test.go` | 246（新） | 双账户签名/缓存/配置/限流/SAPI/时间重同步 Mock 测试 |
| `service/futuresownership/execution.go` | +60/−12 | Broker 可绑定账户；Algo 触发后订单查询同账户 |
| `service/futuresownership/account_broker_stage1_test.go` | 107（新） | 绑定账户的 Submit/Lookup/Cancel 与 Algo 触发路由测试 |
| 文档 | 2 份更新/新增 | 计划文档 Stage 1 状态与实施记录表；`币安合约自动带单-Stage1-实施记录.md` |

**建议下一步（按优先级）**：
1. 就 **F1（429 跨账户阻断）** 在计划文档中明确语义（IP 级共享 vs Key 级自限），并把区分实现排入 Stage 5 的 per-account 归因；
2. 在计划文档标注 **条目 4 为跨阶段项（F2）**，消除验收歧义；
3. 处理 **F3**（research 目录导致的 `go test ./...` 红），以便后续 Stage 有可用的全量 Gate；
4. 继续 Stage 2（DB `account_id` 迁移 + account-scoped Ownership），保持"不接 Lead Executor、不开真单"的边界。
