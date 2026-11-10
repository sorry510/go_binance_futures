# Code Review：方案 02 · S3「现货自定义策略执行与库存风控」

- 审计日期：2026-11-09
- 审计对象：**工作区未提交代码**。HEAD 已推进到 `95b5a77 feat: spot s1 s2`（S1+S2 连同 S2-R1 整改已提交，`origin/feat/trade-upgrade` ahead 1）；工作区为 **S3 增量**：8 个已跟踪文件修改 + 14 个未跟踪新增文件
- 方案依据：`doc/trade/开发方案-02-S3-策略执行与库存风控.md`（工作区版，状态改为「S3 本地代码 Gate 通过；实盘 BLOCKED」，§7 六项全部勾选、§8 为用户调整后的实施边界、§9 为实施与验收记录）；前置 `doc/trade/CODE_REVIEW_02-S2-rereview.md`；上游 S0/S1/S2 方案与总览
- 副本：`/tmp/phase_audit_s3`（`git archive HEAD` + 同步 22 个改动项，排除未跟踪研究目录；`service/spotstrategy`、`spot/api/binance`、`spot`、`feature/strategy/line`、`main.go` 五组文件清单与 SHA256 **全等价**）。探针写在副本内、结束随副本删除
- 报告性质：**新文件**，不覆盖 `CODE_REVIEW_02-S1*.md` / `CODE_REVIEW_02-S2*.md` / `CODE_REVIEW_01-*.md`

> **本轮最重要的基线变化**：与 S2 的「生产零接线」不同，`main.go:533-539` **无条件**把 `spot.RunRealSpotStrategies()` 挂到既有 2 秒 `loopRun`，`spot/api/binance/s3_live.go` 实现了真实 `POST /api/v3/order` 与 `DELETE /api/v3/order`。即 S3 是**真实生产运行路径变更**，且真实资金写能力已经落地。默认关闭成立且已实证，但「默认关闭」的强度与「真实写路径的测试覆盖」成为本轮审查重点。

---

## 一、结论先行

| 项目 | 结论 |
| --- | --- |
| §7 六项完成条件 | **4 项 PASS、2 项部分**（第 1 项「默认不能发单」成立但授权强度不足；第 5 项「隔离测试验证」缺真实写路径覆盖）；逐项见第三节 |
| 离线 Gate（build / 目标包 test / race / vet / diff-check / S2 回归） | ✅ **目标包 5/5 全绿**，race 5/5、vet 6 包、build 2 项、`git diff --check`、S2-R1 四项回归全 PASS |
| `go test -count=1 ./...` 全量 | ❌ EXIT=1（controllers 5 项既有红点 + tests 包缺 `conf/app.conf` panic），**不得写「全仓测试通过」** |
| 默认安全性 | ✅ 未设 `SPOT_STRATEGY_LIVE_TRADING_APPROVAL` 时 runner 在 `orm.GetDB` 与任何私有 IO 之前返回，探针与静态检查双重确认 |
| 库存归属 / 幂等 / S2 冻结语义 | ✅ 设计扎实，独立探针复验 PASS（见第七节） |
| 真实 POST / DELETE payload | ✅ 经隔离探针实证正确（LIMIT/MARKET/DELETE）；但**永久测试一条都没覆盖** |
| 生产上线 Gate / 模式切换 / 真实 Binance | ❌ **全部 BLOCKED，未授权** |
| **S3 → S4** | ⚠️ **有条件**：允许 S4 中**与资金安全无关的纯前端 / 配置 / 只读验收开发**并行；但**不认可 S3 本地代码 Gate 已完全通过**，S3-F1/F2（均 P1）必须在开放任何真实交易之前闭环并复评 |

**本轮新发现 6 项**：S3-F1（P1，实盘启用只需「一个环境变量 + 一行 DB 值」，无持久化授权审计，且 S2-R1 明确列为前置 Gate）、S3-F2（P1，真实 POST/DELETE 写路径零永久测试覆盖，而测试本身可行）、S3-F3（P2，`rest_verified=1` 的「签名 REST 证据」只是 BaseURL 字符串比较）、S3-F4（P2，`HoldingReport` 把未核实 Fill 计入成本/毛收益数值）、S3-F5（P2，默认关闭与门禁的永久测试断言不足）、S3-F6（P3，SELL free 口径、成本舍入、IncompleteReason 覆盖、panic 隔离、端到端串接缺口等）。

**现在什么已经可用**：本地 SQLite 与拦截 transport 层面，`Client` → `Prepare/PrepareExit` → `Submit` → `Reconcile` → `ApplyVerifiedFills` → `HoldingReport` 的账本闭环、归属净量重建、部分成交、dust 保留、移动加权成本、分币费用与 `incomplete` 标识、已收盘 K 线信号与并发防重，均可复现验证。**尚不可用**：把 `rest_verified=1` 当作「真实签名已核实」的强证据、把「默认关闭」当作不可绕过的人工授权流程，以及任何真实委托/撤单/模式切换。

---

## 二、审计基线与生产接线事实

| 事实 | 证据 |
| --- | --- |
| HEAD | `95b5a77 feat: spot s1 s2`（2026-11-09 22:43）；S1+S2 及 S2-R1 整改已提交，S3 全部未提交 |
| Schema | `appversion/version.go` = **25**，S3 未改 DDL / 迁移；`PROBE_SCHEMA version=25`。方案 §9.3 亦声明未执行 `sync db` |
| **生产接线** | `main.go:533-539` 新增 `loopRun(func(){ spot.RunRealSpotStrategies() }, time.Second*2)`，`PROBE_MAIN worker_wired=True unconditional_loopRun=True`。`go list -deps .` 中 `spotstrategy=1`。**S3 是生产运行路径变更** |
| 真实写适配器 | `spot/api/binance/s3_live.go::S3LiveClient` 嵌入 `S2ReadOnlyClient`，覆盖 `Place`（SDK `NewCreateOrderService`）与 `Cancel`（`NewCancelOrderService`）；`PROBE_ADAPTER` 确认 `Place/Cancel` 不再硬禁，改由门禁保护 |
| 默认关闭 | `RunRealSpotStrategies()` 前两条语句为 `LiveOrdersApproved()` 与 `AuthorizeLiveMode()`，均在 `orm.GetDB` / `NewLiveEngine` / `S3LiveClient{}` 之前（`PROBE_RUNNER_FIRST_LINES`）。`SPOT_STRATEGY_LIVE_TRADING_APPROVAL` 默认不存在 → 零私有 IO |
| 双重许可 | `s3_live_gate.go`：`AuthorizeLiveMode()` = env 精确串 `SPOT-S3-REAL-ORDERS-EXPLICITLY-APPROVED` + `CurrentMode()==ModeStrategy`；`AuthorizeLiveWrite()` 再校验作用域、币种启用、持久化 `client_order_id` / `status=submitting`（或撤单时 new/partial）、side/type/quantity 一致 |
| provenance | `S3LiveClient.SpotSignedRESTProvenance()` = `client != nil && client.BaseURL == sdk.BaseAPIMainURL`。全仓生产非测试代码**未对该 Spot client 赋 BaseURL**（反代经 `HTTPClient.Transport` 实现），故该条件在生产恒为 true |
| 新增路由 / WS | `PROBE_ROUTES s3_new_routes=[]`；无新 Web API、无私有用户流、无独立调度平台；复用既有 `loopRun`（CAS 防重入） |
| 静态资源 / 配置 | 未触碰 `static/`、`conf/app.conf`、`strategy_templates/`；无前端改动（S4 范围）；未 commit/push |

**审计过程安全声明**：所有测试与探针进程均显式 `export SPOT_STRATEGY_LIVE_TRADING_APPROVAL=''`。唯一构造「双许可齐备」的探针位于隔离副本，且只调用纯 DB 本地的 `AuthorizeLiveMode()` / `AuthorizeLiveWrite()`，或把 SDK `HTTPClient.Transport` 换成**永不联网、只捕获请求并返回错误/本地 JSON** 的探针 transport。**全程零真实 Binance 请求**。

---

## 三、§7 六项完成条件逐项核对

| # | 方案验收项 | 本轮证据 | 结论 |
| --- | --- | --- | --- |
| 1 | 策略求值只依赖 Spot 变量，有闭合 K 线防重，LONG/CLOSE_LONG | `spot_closed_env.go` 过滤 `CloseTime <= bar.CloseTime`、拒绝 >3 分钟旧 bar、拒绝未收盘 K 线、env 无 Futures/FundingRate/ROI 变量；`s3_signal.go` 的 `signal_key = sha256(account+owner+symbol+direction+closeMS+strategy+technology+rule.Code)`；`TestS3ClosedSignalDeterminismAndDirection` / `TestS3ClosedSpotEnvironmentNoFuturesVariables` / `TestS3ClosedCandleSelectsOnlyFinalizedBar` PASS | ✅ **PASS** |
| 2 | BUY/SELL 用 S2 订单 + Fill 驱动库存，分批成交/手续费/撤单 | `s3_ledger.go::ApplyVerifiedFills` 只消费 `rest_verified=1` Fill，base/quote/BNB 分币、CAS `holding_id IS NULL` + RowsAffected=1 幂等、禁止加仓；`s3_exit.go::PrepareExit` 从 linked verified Fill 重建 `provenBase` 与 `h.OwnedQty` 精确比对；`TestS3VerifiedBuyHoldingPartialFeeAndIdempotency` / `TestS3SellBelongsToHoldingNotAccountFree` / `TestS3SellDustBelowExchangeMinimumStaysOwned` / `TestS3ForgedHoldingWithoutSignedFillsCannotSell` PASS | ✅ **PASS**（但「signed REST → verified Fill → Holding」端到端未串接，见 F6） |
| 3 | 库存周期/Owner 隔离、free 限制、外部订单不明 fail-closed | `ValidateScope` 限定 main/auto_strategy；`UNIQUE(active_slot)` 单周期；free<owned 拒绝、外部挂单非零拒绝、伪造 Holding 拒绝；100 并发同 K 线只 1 单 | ✅ **PASS**（free 口径与方案文字有偏差，见 F6） |
| 4 | 成本/收益/费用分币与 incomplete 可靠 | `s3_pnl.go::HoldingReport` 移动加权成本、毛/净收益、BNB 第三资产 → `Complete=false` + 隐藏 `RealizedNetQuote`；`TestS3...` 断言 BNB 手续费不转 USDT | ⚠️ **部分**：未核实 Fill 仍计入成本/毛收益数值（S3-F4） |
| 5 | 实现完整真实交易代码但保持默认不能发单，隔离数据库 + 本地 HTTP fixture 验证 | 默认关闭成立（默认 worker 零私有 IO）；但 `s3_live_test.go` 只有 23 行否定测试，**真实 POST/DELETE 从未被任何永久测试执行**，真实响应解析亦未覆盖（S3-F2）；方案 §9.1 声称的「签名 CREATE/DELETE」只有代码、无 payload 证据 | ⚠️ **部分 / 不通过** |
| 6 | Go 测试/构建、`git diff --check` 通过，既有 Spot 监听与 USDM 无回退 | 目标包 5/5、controllers S1 专项、race 5/5、vet 6 包、build 2 项、diff-check 全 PASS；S2-R1 四项回归 PASS；`go test ./...` 红点已归因且不称全仓绿 | ✅ **PASS** |

**§7 结论**：第 5 项「隔离测试验证」在**唯一涉及真实资金的写路径**上落空；第 1 项虽成立，但「默认不能发单」的授权强度不足以支撑 S2-R1 设定的实盘前置。故**不认可 S3 本地代码 Gate 完全通过**。

---

## 四、离线 Gate 实跑结果（均在隔离副本内）

| 检验 | 命令 | 结果 |
| --- | --- | --- |
| 全包构建 | `go build ./...` | ✅ EXIT=0 |
| 主程序构建 | `go build -o /tmp/s3_main_bin .` | ✅ EXIT=0（未覆盖用户可执行文件） |
| 目标包单测 | `go test -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./feature/strategy/line ./command` | ✅ **5/5 PASS** |
| controllers S1 专项 | `go test -count=1 ./controllers -run '^TestS1SpotStrategy'` | ✅ PASS |
| S2-R1 回归 | `go test -count=1 ./service/spotstrategy -run '^TestS2F[1245]'` | ✅ **4/4 PASS**（F1 冻结、F2 伪造 verified、F4 外部身份、F5 脱敏） |
| race | `go test -count=1 -race <目标 5 包>` | ✅ **5/5 PASS**，仅 macOS `ld: malformed LC_DYSYMTAB` 非致命警告 |
| vet | `go vet <目标 5 包> ./controllers` | ✅ EXIT=0 |
| diff-check | `GIT_OPTIONAL_LOCKS=0 git diff --check`（用户工作区） | ✅ EXIT=0 |
| 全量测试 | `go test -count=1 ./...`（副本） | ❌ EXIT=1，归因见下 |
| 探针清除后回归 | 删除两个 `zz_*_probe_test.go` 再跑目标 5 包 + S2-R1 | ✅ 全绿，结论未被探针污染 |

**全量红点归因（均非 S3 引入）**：`controllers` 5 项（`DataBase alias default already registered`、strategy template / LLM fixture、`lstat ../strategy_templates: no such file or directory`）；`tests` 包 `panic: the target config file: /tmp/phase_audit_s3/conf/app.conf don't exist`（`conf/app.conf` 未入 Git，纯净 HEAD 副本必然缺）；root 包 `[no test files]`。本轮 `service/leadaccount` 未再出现并行抖动。

---

## 五、探针证据汇总

探针位于 `/tmp/phase_audit_s3/service/spotstrategy/zz_s3_probe_test.go` 与 `/tmp/phase_audit_s3/spot/api/binance/zz_s3_probe_test.go`（临时，已随副本删除）。

| 探针 | 输出 | 含义 |
| --- | --- | --- |
| `PROBE_GATE` | `default_denied=true`；`env_only_denied=true`；`env_plus_one_db_row authorized=true err=<nil>` | **F1：正确 env + 一行 `UPDATE config SET spot_auto_trade_mode='strategy'` 即完成「授权」** |
| `PROBE_AUDIT` | `tables=6 audit_or_event_tables=[]` | **F1：全库无任何 audit / event / approval / mode_history 表** |
| `PROBE_LIVE_WRITE_GATE` | `gate_passed=true`；数量不符与陌生 clientId 均 blocked | **F1：两开关 + 一条本地 `submitting` 订单即可让真实 HTTP 写门禁放行** |
| `PROBE_AUTH` / `PROBE_POST` | `live_mode_authorized=true`；`reached_http=1 method=POST path=/api/v3/order` | 真实 CREATE 路径可达（探针 transport 拦截，未联网） |
| `PROBE_POST_PAYLOAD` | `symbol=BTCUSDT side=BUY type=LIMIT quantity=3 price=12 newClientOrderId=s3-probe-cid timeInForce=GTC` | **F2：真实 LIMIT payload 正确，但永久测试零覆盖** |
| `PROBE_MARKET_PAYLOAD` | `seen=1 type=MARKET quantity=3 price="" timeInForce="" signed_query=true` | MARKET 变体不携带限价/TIF，签名 query 存在 |
| `PROBE_DELETE` | `reached_http=1 method=DELETE path=/api/v3/order ret_err=<nil>` | 真实 DELETE 与响应解析可达且成功 |
| `PROBE_DELETE_PAYLOAD` | `symbol=BTCUSDT origClientOrderId=s3-probe-cid signed_query=true` | 撤单按原 clientId，未用自由 orderId |
| `PROBE_PROVENANCE` | `non_main_baseurl_blocked=true extra_http=0` | 非 main BaseURL 会短路；但生产 BaseURL 恒为 main（F3） |
| `PROBE_LIVE_TEST` | `payload_assertions=False only_denial=True` | **F2：`s3_live_test.go` 只有否定测试** |
| `PROBE_MAIN` / `PROBE_RUNNER_FIRST_LINES` | `worker_wired=True unconditional_loopRun=True`；前两条为双重许可 | 生产 2s worker 无条件挂载；默认零私有 IO 成立 |
| `PROBE_S3_RECOVERY` | `status=reconcile_required new_buy_blocked=true` | **正向：S3 恢复链路不破坏 S2-R1 冻结** |
| `PROBE_UNVERIFIED` | `complete=false reason=fill_source_not_verified cost=200 gross=0 net="" owned=2` | **F4：未核实 Fill 已计入成本/持有量数值** |
| `PROBE_EXIT_FREE` | `owned=4 free=3.5 whole_exit_rejected=true` | **F6：与方案 `min(owned, free)` 口径不符（方向安全）** |
| `PROBE_S3_TESTS` | `count=12`，`PROBE_GUARD expected_writers=8` | 12 个 S3 命名永久测试；AST 守卫期望 8 个写构造器 |
| `PROBE_PROVENANCE_DEFINITION` | `client != nil && client.BaseURL == sdk.BaseAPIMainURL` | **F3：静态配置比较，非响应级证据** |

---

## 六、Findings 详述

### S3-F1（P1）实盘启用门槛仅为「一个环境变量 + 一行 DB 值」，无持久化授权审计

**代码链**：`main.go:533-539` 无条件挂载 2 秒 worker → `RunRealSpotStrategies()` → `LiveOrdersApproved()`（`os.Getenv` 精确串）+ `AuthorizeLiveMode()`（`CurrentMode()==strategy`）→ `NewLiveEngine(db, S3LiveClient{})` → `RecoverAndSettle` / `executeSpotClosedSignal` → `Submit` → `S3LiveClient.Place` → `AuthorizeLiveWrite()`（校验本地订单记录）→ 签名 POST。

**实证**：设置 env 精确值 + 一条 `UPDATE config SET spot_auto_trade_mode='strategy'` → `AuthorizeLiveMode()` 返回 nil；再插入一条 `status='submitting'` 本地订单 → `AuthorizeLiveWrite()` 返回 nil。全库 6 张表中**无任何审计/事件/授权表**。也就是说，S1 用 `RequestMode` 禁止模式切换、要求「先核对旧退出职责、余额、挂单」的安全语义，在代码层可由直接改库绕过；方案 §9.4 只在文档中劝阻「不要靠修改前端开关、直接手动调整生产数据库来绕过」。

**违反的前置约束**：`CODE_REVIEW_02-S2-rereview.md` §四.2 明确「真实交易和解除隔离保持 BLOCKED：新的手工解除权限、真实 REST 证据归属、**可持久查询的审计轨迹**、真实旧单/余额对账都需要另行设计、评审和授权」；§三 F3 亦把「数据库级 append-only 事件审计」列为真实交易前的未决 Gate。S3 方案 §9.4 自己也称「S4 必须完成持久化模式切换/解除冻结审计」。**该前置项未实现，真实下单能力已完整接线。**

**风险**：一旦部署误带该 env，或有人改库把 mode 置为 strategy，全生产进程每 2 秒对最多 3 个启用币自动执行真实委托，无灰度、无单币白名单、无一次性授权、无启停事件、无操作者记录。虽然把 mode 改回 legacy 可即时 kill switch，但**已发出的订单不可逆**。

**修法建议**：
1. 把实盘许可改为「有权限主体签发、一次性/有有效期、可撤销、可持久查询」的凭据（授权事件表 + 操作者 + 时间 + 币种白名单 + 有效期 + 撤销事件），而不是进程环境变量；
2. 让 `RequestMode` 的合法切换成为**唯一**能把 mode 置为 `strategy` 的路径（例如 mode 列加签名/HMAC 或独立审计校验），使直接改库无法生效；
3. `RecoverAndSettle` / `ApplyVerifiedFills` 失败时的 `markReconcile` 以及人工解冻，均需写入 append-only 事件；
4. 在 S4 之前补「未授权时零私有 IO」的可计数永久测试（拦截 transport 计数），并复评。

### S3-F2（P1）真实 POST/DELETE 写路径从未被永久测试执行，而测试完全可行

`spot/api/binance/s3_live_test.go` 仅 23 行，只有 `TestS3RealSpotSDKWritesDeniedByDefault`（断言 env 空时返回 `ErrLiveWriteDisabled`）。`PROBE_LIVE_TEST payload_assertions=False only_denial=True`。12 个 S3 永久测试中**没有一个**驱动 `NewCreateOrderService().Do` / `NewCancelOrderService().Do` 并断言 payload 或解析响应。方案 §9.1 声称实现「真实 Binance SDK 签名 CREATE（MARKET/LIMIT）、DELETE 和受控原 clientOrderId 校验」，§9.3 却只用「默认拒绝」的测试为其背书。

本轮用**永不联网的拦截 transport** 第一次执行了这两条路径，结果正确（LIMIT `quantity=3 price=12 newClientOrderId timeInForce=GTC`；MARKET 无 price/TIF；DELETE `origClientOrderId` 正确、含签名 query、合法响应返回 nil）。这同时证明「因为 `SpotSignedRESTProvenance` 要求 BaseURL=main 所以本地无法测」不成立——只要保持 `BaseURL=sdk.BaseAPIMainURL` 并替换 `HTTPClient.Transport` 即可（改 `BaseURL` 为 localhost 才会被 provenance 挡住，探针 `PROBE_PROVENANCE` 已验证）。

**这是唯一涉及真实资金的写路径，却零覆盖**；`AuthorizeLiveWrite` 的数量/身份/状态矩阵、`NewOrderRespTypeRESULT` 响应解析、撤单响应解析，均无回归保护。S2 阶段 T2-10 尚且用 localhost fixture 验证了 `SellLimit` payload，S3 反而退步。

**修法建议**：把本轮探针固化为永久测试（LIMIT/MARKET payload、DELETE payload + 响应解析、非 main provenance 拒绝、门禁正负例矩阵），作为任何实盘授权前的硬门槛。

### S3-F3（P2）`rest_verified=1` 的「签名 REST 证据」只是 BaseURL 字符串比较

`spot/api/binance/s3_live.go:22-26` 的 `SpotSignedRESTProvenance()` 返回 `client != nil && client.BaseURL == sdk.BaseAPIMainURL`。全仓生产非测试代码未对 Spot client 赋 BaseURL → **该条件在生产恒为 true**，不构成有效安全门。`canAttestLiveFills()` 依赖它 + `AuthorizeLiveMode()`；`s2_reconcile.go` 据此把 Fill 标 `rest_verified=1`；`s3_ledger.go::ApplyVerifiedFills` 据此建立库存并计算盈亏。

即：只要 env + mode 齐备，任何 `GET /api/v3/myTrades` 返回的 JSON（含代理缓存、重放或篡改内容）都会被当作「真实签名已核实」并转成库存。这与 S2-R1「真实认证来源仍 BLOCKED」「不能把模拟收益标成真实已核实」的结论存在**边界跳跃**：S3 未经独立评审就打开了 `rest_verified=1` 的生产写入路径，而证据判定是静态配置、不是本次响应。

**修法建议**：证据绑定到本次响应（请求-响应关联、时间戳/recvWindow 校验、响应来源记录到 Fill 行），并补负向测试证明非签名来源无法置 1；在实盘前明确「代码路径可达」≠「响应已核实」。

### S3-F4（P2）`HoldingReport` 把未核实 Fill 计入成本/毛收益/持有量数值，只隐藏净收益

探针：一条 `rest_verified=0` 的 BUY Fill → `Complete=false`、`IncompleteReason=fill_source_not_verified`、`RealizedNetQuote=""`（这部分正确），但 `CostBasisQuote=200`、`OwnedQty=2` 均包含该未核实数据，`RealizedGrossQuote` 亦由全部 Fill 累加。方案 §四要求「仅从本周期已核实成交取得」「不能证明实际来源时保持 incomplete，不计算已核实收益」，§9.2 要求不把模拟成交当作实盘已验证 Fill。

正常生产链路由 `ApplyVerifiedFills` 保证关联 Fill 均为 verified=1，故主要风险在脏数据、回放与手工修复场景；但展示层把不可信数据算进「成本」仍会误导人工对账。**建议**：`Complete=false` 时把成本/毛收益/持有量一并标注或分离为「已核实 / 未核实」两列，而不是只隐藏净收益。

### S3-F5（P2）默认关闭与门禁的永久测试断言不足

- `spot/s3_runner_test.go::TestS3WorkerDefaultOffNoPrivateExchangeIO` 仅调用 `RunRealSpotStrategies()`，**没有任何断言**证明零私有 IO；「默认零 IO」这一结论本轮靠审查侧静态探针（`PROBE_RUNNER_FIRST_LINES`）与代码阅读得出，不是永久回归。
- `TestS3DefaultLiveAuthorizationDenied` 只测 env 空 + Fake 无 provenance；未覆盖「env 正确但 mode=legacy」「env 正确 + mode=strategy + 合法订单」「数量不符 / 陌生 clientId / 错误 status」组合。
- `s1_guard_test.go` 的 AST 守卫期望 `writers=8`，但只按「函数内出现过 `AuthorizeLiveWrite` 调用」判定 `guarded=true`，不校验调用是否在实际 SDK 写之前、参数是否完整（S1-F5 遗留的同类弱断言）。

**建议**：补组合矩阵 + 拦截 transport 的 IO 计数断言，把「默认零私有 IO」变成可回归的永久证据。

### S3-F6（P3）若干实现/文档口径偏差与工程风险

1. **`PrepareExit` 的 free 口径**：方案 §三写「可卖上限 = `max(0,min(策略净 owned − 待卖, free))`」，实现是 `free < owned` 直接拒绝整个 SELL（探针 `owned=4 free=3.5` 全拒）。方向更安全，但方案文字需同步，否则人工对账时会误判为缺陷。
2. **成本舍入**：`s3_ledger.go:251` `avg = basis.DivRound(owned, 18)` 每笔 Fill 重算并舍入，多笔成交存在累积误差；方案 §四要求「不因显示四舍五入改变真实结算」。DB 列为 `DECIMAL(36,18)`，属存储上限，但建议改为保存累计 `cost_basis_quote` 原值而非反复折返 avg。
3. **`IncompleteReason` 被覆盖**：`s3_pnl.go` 循环中 `fill_source_not_verified` 会被后续 `third_asset_fee_requires_rate_at_trade_time` 覆盖，多原因只保留最后一个。
4. **panic 隔离**：`main.go:567` 的 `loopRun` 与 runner 均无 `recover()`（`PROBE_RUNNER recover=False`）。启用后任何 expr 求值 / 反射 / SDK panic 会终止整个交易服务进程。属既有架构模式，但 S3 新增链路放大了影响面。
5. **端到端串接缺口**：S3 测试普遍用 `UPDATE spot_strategy_fills SET rest_verified=1` 手工置位、或直接 SQL 插入订单状态与 Fill，**没有**「签名 REST `myTrades` → `Reconcile` 标 verified → `ApplyVerifiedFills` 建库存 → `HoldingReport`」的完整回放测试。
6. **`RecoverAndSettle` 失败时无条件 `markReconcile`**：S2-R1 F3 的 append-only 审计与人工解冻流程仍未实现，冻结/解冻无法追溯。

---

## 七、正向实证（不因 Findings 抹杀已落地的安全设计）

| 项 | 证据 |
| --- | --- |
| 默认零私有 IO | runner 前两条语句为双重许可检查，位于 `orm.GetDB` / `NewLiveEngine` / `S3LiveClient{}` 之前；`SPOT_STRATEGY_LIVE_TRADING_APPROVAL` 默认不存在 |
| 归属不可伪造 | `PrepareExit` 从 `holding_id` 关联的 verified Fill 重建 `provenBase`，与 `h.OwnedQty` 精确相等才允许 SELL；`TestS3ForgedHoldingWithoutSignedFillsCannotSell` PASS |
| 幂等 / 不加仓 | `ApplyVerifiedFills` 用 `UPDATE ... WHERE holding_id IS NULL AND rest_verified=1` + `RowsAffected==1` CAS；`Prepare` 检查 `active_slot` 已有 active 周期；`TestS3VerifiedBuyHoldingPartialFeeAndIdempotency` / `TestS3ClosedCandleHundredConcurrentTriggersSingleIntent` PASS |
| 不超卖 | 只用策略净量、交叉校验 `free`、`OpenOrders` 非零即拒绝、dust 保留、LOT/PRICE/NOTIONAL 过滤器齐全 |
| 账号与信号隔离 | `ValidateScope` 仅 main/auto_strategy；`signal_key` 含账户/Owner/币种/方向/收盘时间/策略定义/规则代码，闭合 K 线防重 |
| 分币费用与 incomplete | base/quote 手续费分别影响库存与成本，BNB 第三资产 → `Complete=false` + 隐藏净收益；`TestS3...` 断言 BNB 不转 USDT |
| S2-R1 整改未被破坏 | 4 项回归测试 PASS；`PROBE_S3_RECOVERY` 证明 S3 的 `RecoverAndSettle` 不会解冻 `reconcile_required` |
| 真实写 payload 正确 | 探针实证 LIMIT/MARKET/DELETE 三类 payload 与响应解析均正确（见第五节） |
| 交付边界 | Schema 仍 v25、无新增路由/WS/前端/static 改动、未执行 `sync db`、未访问真实 Binance |

---

## 八、生产 Gate（全部 BLOCKED，未授权）

| Gate | 状态 | 说明 |
| --- | --- | --- |
| 真实 Binance 下单 / 撤单 | ⛔ BLOCKED | `Place/Cancel` 默认被双重许可拒绝；本轮零真实请求 |
| `strategy` / `off` 模式真实切换 | ⛔ BLOCKED | `RequestMode` 仍硬拒；代码无法区分审计切换与直接改库（F1） |
| 旧自动 SELL 退出职责 / 库存归属对账 | ⛔ BLOCKED | S0 §7.3 与 S1 的核心前置，未核实 |
| 真实 `myTrades` 逐 `tradeId` 与手续费资产核对 | ⛔ BLOCKED | 仅 Fake / 手工 SQL fixture，provenance 仅为 BaseURL（F3） |
| 持久化授权 / 模式切换 / 解冻审计 | ⛔ BLOCKED | 无任何审计表（F1），S2-R1 与 §9.4 均列为前置 |
| 生产 MySQL 8 DDL / 索引 / EXPLAIN / P95 | ⛔ BLOCKED | v25 仅 SQLite 与 DDL 文本验证 |
| 生产 v25 升级、二进制 + static + schema 成套发布与回滚 | ⛔ BLOCKED | 未替换二进制、未重启服务 |
| T0 / 版本持久记录 / 二进制 hash | ⛔ BLOCKED | 沿用 01-ALL 未决项 |
| 旧 `eatRate` 手动路由停用确认 | ⛔ BLOCKED | S0 §7.3 要求，S3 未处理 |
| 014-F1（`applySettledRisk` 唯一风控入口）等 01 线未决项 | ⛔ BLOCKED | 未随 S3 关闭 |

---

## 九、整改与复评要求

1. **F1、F2 为 P1**：F2 应在本阶段就地闭环 —— 把本轮真实写请求探针固化为永久测试（LIMIT/MARKET/DELETE payload + 响应解析 + 门禁矩阵），并补「默认零私有 IO」的可计数断言。F1 可列入 S4 的实盘准入前硬门槛，但**不得在 F1 闭环前开放任何真实交易**。
2. **F3 必须在 S3 消费 `rest_verified=1` 之前明确**：区分「代码路径可达」与「本次签名响应已核实」，并把证据判定提升到响应级；否则 S2-R1 的结论被静默推翻。
3. F4（未核实 Fill 展示）、F5（断言矩阵）、F6（文档口径与工程风险）建议同期整改。
4. 复评时**另起 `CODE_REVIEW_02-S3-rereview.md`**，不覆盖本报告。
5. **S4 范围限定**：仅前端 / 配置 / 只读验收开发可并行；实盘启用所需的持久化授权审计、旧单/余额对账、单币受控首单授权与回滚演练，必须在独立验收与用户明确授权后进行。

---

## 十、审计纪律声明

本轮**未修改任何业务代码**，只新增本报告与项目记忆；未 commit/push、未改 `conf/app.conf`、未触碰 `static/` 与 `strategy_templates/`、未对真实库执行 `sync db`、未访问真实 Binance、未重启服务、未操作生产仓位。所有临时探针写入 `/tmp/phase_audit_s3` 隔离副本并在验证后删除，副本与主程序二进制 `/tmp/s3_main_bin` 在收尾时一并清理。报告中的「离线 PASS」仅指本地 Go/SQLite/Fake/拦截 transport 验证，**不代表任何实盘 Gate 通过**。
