# Code Review：方案 02 · S2「Spot 私有 REST、真实成交与订单恢复」

- 审计日期：2026-11-09
- 审计对象：**工作区未提交代码**。HEAD `e40b01a doc: update doc`（`feat/trade-upgrade`，`origin/feat/trade-upgrade` ahead 1）；工作区为 **S1 + S1 整改 + S2 全部未提交叠加**：14 个已跟踪文件修改 + 12 个未跟踪新增文件/目录
- 方案依据：`doc/trade/开发方案-02-S2-私有REST与订单状态.md`（工作区版，状态已改为「S2 本地 Mock/REST 适配/状态恢复 Gate 已验收」，§7 五项全部勾选、§8 为实施与验收记录）；前置 `doc/trade/CODE_REVIEW_02-S1-rereview.md`；上游 `开发方案-02-S0-现状审计与安全边界.md`、`开发方案-02-S1-配置数据与安全门禁.md`、总览与 S3 方案
- 副本：`/tmp/phase_audit_s2`（`git archive HEAD` + 同步 26 个改动项，排除未跟踪的 `strategy_templates/research/**`；S2 关键 16 个文件 SHA256 逐项核对 **0 不一致**）。探针写在副本内、结束随副本删除
- 报告性质：**新文件**，不覆盖 `CODE_REVIEW_02-S1.md` / `CODE_REVIEW_02-S1-rereview.md` / `CODE_REVIEW_01-*.md` 等历史报告

> **对「S2 → S3 已可开始」的口径说明**：本轮实证确认 S2 的 Mock 私有 REST 适配、订单身份、分页 `myTrades` 去重、部分成交撤单、UNKNOWN 与重启恢复**确实落地**，T2-01～T2-11 均有命名永久测试，离线目标包 Gate 全绿，生产零接线。但发现两条 **P1 阻塞项**：一条是「终态历史成交冲突冻结」在**正常恢复流程**中会被降级、随后新 BUY 被错误放行；另一条是 `rest_verified` 只由构造器标志决定、Fake Client 也能产出「真实 REST 已核实」账本。两者都直接破坏方案 §8.2 自述的不变量，且 S3 即将消费这些 Fill 做库存。因此本报告**不认可 S2 → S3 放行**。

---

## 一、结论先行

| 项目 | 结论 |
| --- | --- |
| §7 五项完成条件 | **3 项 PASS、1 项部分（第 3 项，状态/冻结不变量有洞）、1 项 PASS（第 4 项含全量红点归因）**；逐项见第三节 |
| T2-01～T2-11 覆盖 | ✅ 11/11 有命名永久测试（共 21 个测试函数）；⚠️ T2-11 断言偏弱（见 S2-F4） |
| 离线 Gate（build / 目标包 test / race / vet / diff-check / 探针） | ✅ **目标包全绿**，纯净副本 `go build ./...` 与主程序 build 均 EXIT=0 |
| `go test -count=1 ./...` 全量 | ❌ EXIT=1（controllers 5 项既有红点 + tests 包缺 `conf/app.conf` panic + leadaccount 并行抖动），**不得写「全仓测试通过」** |
| 生产接线 | ✅ S2 的 `Engine` / `S2ReadOnlyClient` **零生产调用**；无新增 Web API、无新 WS、无后台常驻进程、Schema 仍 v25 |
| 生产上线 Gate / `strategy`/`off` 切换 / 真实 Binance | ❌ **全部 BLOCKED，未授权** |
| **S2 → S3** | ⛔ **不放行**。先闭环 S2-F1、S2-F2（均 P1），再复评；F3/F4 建议同期整改 |

**本轮新发现 6 项**：S2-F1（P1，终态对账冻结被恢复流程降级 → 新 BUY 解冻）、S2-F2（P1，`rest_verified` 信任边界由构造器而非查询证据决定）、S2-F3（P2，方案正文与实施记录对「终态回退」口径冲突且降级无独立审计）、S2-F4（P2，T2-11 覆盖不足 + `ErrUnmanagedOrder` 定义未用 + `QueryOrder` 按 orderId 查询时忽略 clientId）、S2-F5（P3，`Prepare` 透出 SDK 原始错误）、S2-F6（P3，全量红点归因与文档既有偏差）。

**现在什么已经可用**：本地 Mock/SQLite 层面，`Client` 接口与 5 个只读 REST 适配方法、`Prepare/Submit/Reconcile/CancelMock/RecoverPending/ListFills` 状态机、精确 Decimal 汇总、`myTrades` 分页去重、ACK≠Fill、UNKNOWN 不重发、部分撤单保留 Fill、`SellLimit` 数量/单价修复，均可复现验证。**尚不可用**：把 S2 账本当作「已核实真实成交」、在终态历史冲突未人工核对时继续开新仓、以及任何真实 Spot 委托。

---

## 二、审计基线与生产接线事实

| 事实 | 证据 |
| --- | --- |
| Schema 版本 | `appversion/version.go` = **25**；S2 文档 §8.1 声明「未更改 v25 DDL 和迁移」，静态探针 `PROBE_SCHEMA version=25`；`SchemaDDL` 未改动，三表 PK/UNIQUE 与 `DECIMAL(36,18)` 与 S1 一致 |
| 订单写路径 | `NewMockEngine` / `NewReadOnlyEngine` 仅在 `service/spotstrategy/s2_engine.go` 定义；**非测试 Go 文件中调用点 0**（`PROBE_PROD_WIRING refs=1`，唯一命中是 `s2_readonly.go:20` 的编译期接口断言 `var _ spotstrategy.Client = S2ReadOnlyClient{}`） |
| 只读适配器 | `PROBE_ADAPTER write_constructors=NONE`：`s2_readonly.go` 不含任何 `NewCreateOrderService` / `NewCancelOrderService`；`Place` / `Cancel` 恒返回 `ErrLiveWriteDisabled`；5 个读方法均带 `s2ReadContext` 且 `TestS2ReadOnlyAdapterSDKLocalREST` 断言所有请求为 GET |
| `SellLimit` 修复 | `PROBE_SELLLIMIT calls_helper=True price_arg_correct=True`：生产函数 `SellLimit`（`index.go:209-226`）**确实调用**修复后的 `s2BuildLegacySellLimit`，不是「只加了一个测试用 helper」；`Price(strconv.FormatFloat(price, ...))` 已改正 |
| 新增路由 | `PROBE_ROUTER added_lines` 只有 S1 的 `/spots/strategy/config` 与 `/spots/strategy/mode`；**S2 未新增任何 Web API / 下单 / 撤单路由** |
| 生产依赖闭包 | `go list -deps . \| grep -c spotstrategy` = **1**，来源是 S1 门禁（`spot/spot.go`、`spot/api/binance/index.go` 引入 `config.go`）。**S2 的 `Engine` 不在生产运行路径**；但 `Reconcile` 本身会写本地账本，`NewReadOnlyEngine` 名字里的「只读」仅指交易所侧，不是本地零写入 |
| 静态资源 / 配置 | 未触碰 `static/`、`conf/app.conf`、`strategy_templates/`；本轮无前端改动（S4 范围）；未 commit/push |

---

## 三、§7 五项完成条件逐项核对

| # | 方案验收项 | 本轮证据 | 结论 |
| --- | --- | --- | --- |
| 1 | 完整可 Mock 的私有 REST 客户端，真实写调用默认关闭；无新私有 WS 或内部任务平台 | `Client` 7 方法最小接口；`S2ReadOnlyClient` 只实现 `account/exchangeInfo/openOrders/order/myTrades`；`Place/Cancel` 硬禁；`TestS2ReadOnlyAdapterCannotWrite` + `TestS2LiveAdapterWriteDisabled` + `PROBE_LIVE prepare/submit/cancel=true`；无 goroutine/scheduler，`RecoverPending` 为显式有界调用 | ✅ **PASS** |
| 2 | T2-01～T2-11 有永久测试和复现命令，含重复 Fill、未知提交、重启恢复、部分撤单、事务失败 | `PROBE_T2_COVERAGE T2-01..T2-11=True`（21 个测试函数）：T2-01/02/03/04-05/06/07/08/09/11 在 `s2_test.go`，T2-10 在 `s2_readonly_test.go`；另有分页、历史冲突、跨币预算、Fee、MARKET_LOT_SIZE、页上限等 10 项额外测试 | ✅ **PASS**（T2-11 断言强度见 S2-F4） |
| 3 | 订单状态转换、唯一订单身份、查询分页、资金/库存预留可核对，且不生成无事实 Fill | 大部分成立：ACK 不产 Fill、按 `tradeId` 去重、内容冲突 fail-closed、分页 `fromId` 递增 + 20 页上限、事务回滚无幻影 Fill、`qtyTotal/quoteTotal` 与交易所累计量比对、跨币同报价预算。**但 §8.2 两条自述不变量被实证打破**：冻结状态可被 `RecoverPending` 降级后放行新 BUY（S2-F1）；`rest_verified` 可由 Fake 客户端置 1（S2-F2） | ⚠️ **部分 / 不通过** |
| 4 | 受影响 Go 测试/构建、race、`git diff --check` 通过；已有独立红点单列，不称全仓绿 | 本轮独立复跑：目标包 4/4、controllers S1 专项、race 4/4、vet 5 包、build 2 项、`git diff --check` 全部 EXIT=0；`go test ./...` EXIT=1 且文档 §8.4 已明确列出红点并声明「未宣称全仓绿」 | ✅ **PASS** |
| 5 | 填写最终文件列表、状态机代码与模拟输出；真实成交、取消或账户写接口均未执行 | §8.1 文件表与 §8.2/8.3 输出完整；本轮静态 + 运行时探针确认生产零接线、无真实 Binance 调用；所有测试用临时 SQLite / Fake / localhost `httptest` | ✅ **PASS** |

**§7 结论**：第 3 项构成实质性缺口。方案 §7 的「S2 → S3 Gate：订单创建、查询、取消、真实 Fill 去重和恢复经 Mock 全部可证实」中，「恢复」的可证实性被 S2-F1 否定，「真实 Fill」与「模拟 Fill」的区分被 S2-F2 否定。**不放行 S3。**

---

## 四、离线 Gate 实跑结果（均在隔离副本内）

| 检验 | 命令 | 结果 |
| --- | --- | --- |
| 全包构建 | `go build ./...` | ✅ EXIT=0 |
| 主程序构建 | `go build -o /tmp/s2_main_bin .` | ✅ EXIT=0（未覆盖用户可执行文件） |
| 目标包单测 | `go test -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./command` | ✅ **4/4 PASS** |
| controllers S1 专项 | `go test -count=1 ./controllers -run '^TestS1SpotStrategy'` | ✅ PASS |
| race | `go test -count=1 -race ./service/spotstrategy ./spot ./spot/api/binance ./command` | ✅ **4/4 PASS**，仅 macOS `ld: malformed LC_DYSYMTAB` 非致命警告 |
| vet | `go vet ./service/spotstrategy ./spot ./spot/api/binance ./command ./controllers` | ✅ EXIT=0 |
| diff-check | `GIT_OPTIONAL_LOCKS=0 git diff --check`（用户工作区） | ✅ EXIT=0 |
| 全量测试 | `go test -count=1 ./...`（副本） | ❌ EXIT=1，归因见下 |
| 探针清除后回归 | 删除 `zz_s2_probe_test.go` 再跑目标包 4/4 + controllers | ✅ 全绿，结论未被探针污染 |

**全量红点归因（均非 S2 引入）**：

| 包 | 表现 | 归因 |
| --- | --- | --- |
| `controllers` | 5 项 FAIL：`TestQueryStrategyTemplatePageOrdersByIDDescending`（`DataBase alias default already registered`）、3 项 strategy template 校验/LLM fixture、`TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals`（`lstat ../strategy_templates: no such file or directory`） | 隔离副本未同步未跟踪 `strategy_templates/**` 与既有 fixture 冲突，属 S1 报告已记录的既有红点 |
| `tests` | `panic: the target config file: /tmp/phase_audit_s2/conf/app.conf don't exist` | `conf/app.conf` 未纳入 Git，纯净 HEAD 副本必然缺配置；与 01-ALL 的 ALL-F3 同源 |
| `service/leadaccount` | `TestStage56SupervisorRecoveryStartsBeforeWSAndStopsAllWorkers` 全量并行下失败 | 单包复跑 `ok 0.926s`，判定并行时序抖动（01-ALL ALL-F6 已记录同类现象） |
| root 包 | `? go_binance_futures [no test files]` | 无测试文件，本轮未复现 01-ALL 的 root panic |

---

## 五、探针证据汇总

探针文件 `/tmp/phase_audit_s2/service/spotstrategy/zz_s2_probe_test.go`（临时，已随副本删除）。

| 探针 | 输出 | 含义 |
| --- | --- | --- |
| `PROBE_TERMINAL` | `status=filled pending_slot=<nil> reserved=0` | 终态按设计释放槽位与预留 |
| `PROBE_FREEZE` | `status=reconcile_required new_buy_blocked=1` | 历史 Fill 冲突确实冻结该币（与永久测试一致） |
| `PROBE_RECOVER` | `real_timeline=1 scanned=1 status=unknown last_error=spot_rest_rate_limited_429` | **不手工改时间戳**、仅等待 3.1s 超过文档规定的最小检查间隔后，正常 `RecoverPending` 把冻结态降级为 `unknown` |
| `PROBE_FINDING` | `P1_CONFIRMED quarantine_lost_after_recovery new_BUY_accepted=1 orders_in_table=2` | **S2-F1 实证：新 BUY 被接受** |
| `PROBE_CONTROL` | `before status=reconcile_required slot=main:auto_strategy:BTCUSDT` → `after status=unknown new_BUY_blocked=true err=duplicate Spot order or pending symbol` | 对照组：非终态冻结因 `pending_slot` 唯一仍生效，**影响面精确限定为已终态（槽位为 NULL）的订单** |
| `PROBE_REST_VERIFIED` | `status=filled verified=1/1 client=*spotstrategy.fakeS2Client` | **S2-F2 实证：Fake 客户端产出的 Fill 被标为真实 REST 已核实** |
| `PROBE_LIVE` | `prepare=true submit=true cancel=true` | `NewReadOnlyEngine` 三个写路径均返回 `ErrLiveWriteDisabled` |
| `PROBE_MODE` | `legacy=<nil>`；`off=spot_mode_reconciliation_required: legacy -> off`；`strategy=strategy mode unavailable during S1` | S1 整改回归：无跨模式切换 |
| `PROBE_GATE` / `PROBE_INTENT` | 空来源、`SPOT_RUSH`、空/未知意图全部 blocked；三个白名单来源与 `new_risk/exit` 放行 | S1 整改回归：来源 + 意图白名单生效 |
| `PROBE_SELLLIMIT` | `calls_helper=True price_arg_correct=True` | 生产 `SellLimit` 真实接线修复 |
| `PROBE_ADAPTER` | `write_constructors=NONE place_disabled=True cancel_disabled=True` | 生产适配器无任何交易所写构造器 |
| `PROBE_PROD_WIRING` | `refs=1`（仅接口断言） | S2 `Engine` 生产零接线 |
| `PROBE_T2_COVERAGE` | `T2-01..T2-11=True total=21` | 11 项用例均有命名测试 |
| `PROBE_MARKUNKNOWN` | `excludes_reconcile=False excludes_terminal=True` | **S2-F1 根因 1** |
| `PROBE_PREPARE_QUARANTINE` | `WHERE ... AND status='reconcile_required'`（单一条件） | **S2-F1 根因 2** |
| `PROBE_RECOVER_SCAN` | `status IN (...,'reconcile_required')` | **S2-F1 触发条件** |
| `PROBE_ERRUNMANAGED` | `definitions=1 usages_outside_types=0` | S2-F4：哨兵错误定义后从未使用 |
| `PROBE_ROUTER` / `PROBE_SCHEMA` | 仅 S1 两条路由；`version=25` | 无新增 API、Schema 未变 |

---

## 六、Findings 详述

### S2-F1（P1，阻塞 S2→S3）终态对账冻结可被正常恢复流程降级，随后新 BUY 被错误放行

**代码链**：

1. `s2_reconcile.go:268-271`：终态时 `reserved_qty=0`、`pending_slot=NULL`，唯一槽位约束不再保护该币。
2. `s2_reconcile.go:322-324`：`RecoverPending` 的扫描集合**包含** `reconcile_required`（方案 §四.7 明确要求）。
3. `s2_reconcile.go:96-101`：`QueryOrder` 失败（429/418/timeout/-2013）→ `e.markUnknown(ctx, id, safeReadError(readErr))`。
4. `s2_engine.go:339-343`：`markUnknown` 的 WHERE 仅 `status NOT IN ('filled','canceled','expired','rejected')`，**未排除 `reconcile_required`**，于是把冻结态改写为 `unknown`。
5. `s2_engine.go:237`：`Prepare` 的冻结检查只有 `... AND status='reconcile_required'`；降级后不命中，且 `pending_slot` 已为 NULL，其余检查（余额、预算、openOrders、signal_key）全部可通过。

**实证**（真实时间线、无人工改时间戳）：`filled` → 历史 Fill 冲突 → `reconcile_required`（此时新 BUY 被拒）→ 3.1s 后 `RecoverPending` 遇 429 → `unknown` → **新 BUY 成功，`orders_in_table=2`**。对照组证明非终态冻结因 `pending_slot` 唯一仍安全，故影响面精确为「**已结算订单的历史成交被篡改/不一致后，恢复流程把它静默解冻**」。

**违反**：方案 §8.2「即使旧订单已经终态，后续 Fill 内容若发生冲突也重新进入对账状态，阻止新买入」；永久测试 `TestS2TamperedTerminalTradeBlocksNewRisk` 的设计意图（该测试只断言冲突当刻，未覆盖后续恢复）；S2→S3 Gate 的「恢复经 Mock 全部可证实」。§三「杜绝从终态回退/重复计量」亦被间接破坏。

**为什么现有测试没抓到**：21 个永久测试中，冻结与恢复是两条独立路径；`TestS2T09` 只验证 429 后 `unknown` 且新 BUY 被拒（那时订单仍持有 `pending_slot`），`TestS2TamperedTerminalTradeBlocksNewRisk` 只验证冻结当刻，二者从未串接。

**修法建议（需同时满足 1 + 2 或采用 3）**：
1. `markUnknown` 的 WHERE 增加 `AND status <> 'reconcile_required'`，与终态同等不可降级；`markReconcile` 亦应明确其允许覆盖的状态集合。
2. `RecoverPending` 对 `reconcile_required` **只做只读核查**，不得调用会改写状态的 `Reconcile`；或至少保证任何恢复路径都不能把冻结态迁移为其它状态。
3. 更稳妥：冻结证据不依附于可迁移的 `status` 列 —— 增加独立、不可被自动流程清除的隔离标志（列或事件表），`Prepare` 按该标志冻结，仅人工核对后可显式解除；并为解除动作写审计。

**必须补的回归测试**：终态 → 历史 Fill 冲突 → 等待超过最小检查间隔 → `RecoverPending` 遇 429/418/timeout/-2013 → `Load` 仍为冻结态 → 新的 `Prepare` 仍被拒；同断言覆盖 `markUnknown` 的其它触发分支（`spot_order_identity_mismatch` 等）。

### S2-F2（P1，阻塞 S2→S3）`rest_verified` 由构造器标志决定，Fake 客户端可产出「真实 REST 已核实」账本

**代码**：`s2_reconcile.go:183-188` 插入 Fill 时 `rest_verified = !e.mockOnly`；`s2_engine.go:25-30` 的 `mockOnly` 只由 `NewMockEngine` / `NewReadOnlyEngine` 决定，**不校验 `Client` 是否真的发起过带签名的 REST 查询**。`s2_read.go::ListFills` 已把 `RESTVerified` 暴露给调用方。

**实证**：`NewReadOnlyEngine(db, f)`（f 为 `*fakeS2Client`）调 `Reconcile` → `verified=1/1`。即 Mock 数据可产生与真实 REST 核实**不可区分**的账本行。

**违反**：方案 §8.2「Fake 测试插入的 Fill 的 `rest_verified=0`；未来仅真实只读 REST 凭证核验通过的 Fill 才允许标记 1；不能把模拟收益标成真实已核实」；§五「无法确认时不要标为 verified」。

**当前风险边界**：生产零接线，尚未污染真实库。但 S3 的核心工作就是消费 Fill 生成 Holding 与盈亏，一旦 S3 或任何后续代码用 `NewReadOnlyEngine` 配合测试/回放客户端，就会把模拟成交写成已核实事实，且下游无法撤销或辨别。这是**信任边界设计缺陷**，必须在 S3 消费前修掉，而不是留到实盘 Gate。

**修法建议**：把 `rest_verified` 绑定到可验证事实而非构造器 —— 例如 `Client` 接口增加 `SignedRESTEvidence() bool`（只有 `S2ReadOnlyClient` 返回 true，且仅在本次 `Trades` 确由其执行时生效），或由 `Reconcile` 显式接收调用方提供的「REST 已核实」凭据；补负向永久测试：`NewReadOnlyEngine(db, fake)` 的 Fill 必须 `rest_verified=0`，只有真实适配器路径才允许置 1。

### S2-F3（P2）方案正文与实施记录对「终态回退」口径冲突，且降级缺少独立审计

- 方案 §三：「状态转换需有明确的可测试允许路径，**杜绝从终态回退**/重复计量」。
- 方案 §8.2 与 `TestS2TamperedTerminalTradeBlocksNewRisk`：**明确要求** `filled` → `reconcile_required`。
- `markReconcile`（`s2_engine.go:347-351`）是无条件 `UPDATE`，无终态守卫、无审计事件/告警，只有一列 `last_error` 记录原因；`markUnknown` 还能继续覆盖它（S2-F1）。

这一 fail-closed 回退本身**方向正确**（历史成交被篡改必须冻结），但方案正文未同步修订，且运维无法区分「数据被篡改」「交易所短暂不一致」「人工介入」，也无法追溯冻结发生与解除的时间线。S1-rereview 已要求「S2 若真正开放转换，必须有可持久追踪的转换审计」，S2 未建立对应事件记录。

**建议**：在方案正文写明「唯一允许的终态回退是 → `reconcile_required`，且必须配不可覆盖的冻结证据 + 审计事件」；补结构化事件记录（至少含订单 id、旧/新状态、原因、来源、时间戳、操作者或自动流程标识）；统一 §三、§8.2 与代码注释的表述。

### S2-F4（P2）T2-11 覆盖不足、`ErrUnmanagedOrder` 定义未用、`QueryOrder` 按 orderId 时忽略 clientId

- `PROBE_ERRUNMANAGED definitions=1 usages_outside_types=0`：`s2_types.go:20` 定义的 `ErrUnmanagedOrder` 全局无使用点。
- T2-11 永久测试 `TestS2T11_InvalidScopeAndExternalOrderRejected` 只覆盖 `Prepare` 的 `lead` 账户、`SELL`、以及 `OpenOrders` 存在外部挂单三种情形；**未覆盖**「用外部/未托管 orderId 调 `Reconcile` / `CancelMock` / `ListFills`」这一 T2-11 明确要求的「误取消、误记账」。
- 当前未实际越权：`loadOrder` 按本地自增 `id` 查询（外部订单不在表中）、`ValidateScope` 拒绝非 main/auto_strategy；但这是隐式防护，缺少显式契约与断言。
- `s2_readonly.go:124-133`：`QueryOrder` 在 `orderID != ""` 时只用 `OrderID(id)`，完全忽略传入的 `clientID`；配合 `s2_reconcile.go:111-117`「终态允许 clientOrderId 不同」的例外，若本地 `exchange_order_id` 脏数据指向同 symbol/side/requestedQty 的外部订单，可能把外部成交记入本策略账本。

**建议**：补 T2-11 变体（外部 orderId 注入 `Reconcile`/`CancelMock` 必须显式拒绝）；`QueryOrder` 的 orderId 分支仍校验返回的 clientOrderId 或在请求中同时带上 `origClientOrderId`（SDK 支持时）；使用或删除 `ErrUnmanagedOrder` 哨兵。

### S2-F5（P3）`Prepare` 路径透出 SDK 原始错误，脱敏仅覆盖 `Reconcile`

`safeReadError` 只在 `Reconcile` 的错误分类路径使用；`Prepare` 中 `fmt.Errorf("symbol verification unavailable: %w", err)`、`"Spot balance unavailable: %w"`、`"Spot openOrders unavailable: %w"` 保留底层 SDK / transport 错误，网络层 `url.Error` 可能包含带 `signature=` 的完整请求 URL。永久测试 `TestS2SafeReadErrorDoesNotEchoSecrets` 只测 `safeReadError` 本身，未覆盖 `Prepare`。当前 `Prepare` 仅 Mock 可用、不上生产，但 S3 将复用它。**建议**：`Prepare` 同样使用 `safeReadError` 分类或统一脱敏包装，并补对应测试。

### S2-F6（P3）全量红点与既有文档偏差

见第四节归因。另：文档 §8.1 只列 8 个 S2 新增文件，实际还包含 `service/spotstrategy/s2_read.go` 之外的 S1 整改新增文件（`controllers/spot_strategy_s1_test.go`、`spot/api/binance/s1_guard_test.go`、`spot/s1_notice_gate_test.go`、`CODE_REVIEW_02-S1*.md` 等），这些属于 S1 整改而非 S2，建议在阶段台账中明确区分，避免把 S1 未提交整改误算作 S2 交付。文档 §8.4 声明的 6 项回归本轮独立复跑**全部成立**，无夸大。

---

## 七、T2-01～T2-11 覆盖矩阵与断言强度

| 用例 | 永久测试 | 关键断言 | 本轮判定 |
| --- | --- | --- | --- |
| T2-01 | `TestS2T01_ACKDoesNotCreateFillOrHolding` | ACK/NEW 后 Fill=0、Holding=0、`filled_qty=0` | ✅ |
| T2-02 | `TestS2T02AndT08_PartialFillCancelKeepsRealFills` | 2@100 + 2@120 → base 4 / quote 440 / 2 条 Fill / 费用币 BTC+BNB / `rest_verified=0` / 终态重放不重复 / 撤单释放 reserved | ✅ |
| T2-03 | `TestS2T03_DuplicateTradeIdHundredTimes` | 100 次重复 + 5 轮 Reconcile → 1 条 Fill | ✅ |
| T2-04 | `TestS2T04T05_TimeoutAcceptedAndMissingNeverResubmit` | 超时后按原 clientId 找回、`f.calls=1`、二次 `Submit` 拒绝 | ✅ |
| T2-05 | 同上 | 查询亦不确定 → 保持 `unknown`、保留 pending_slot、不新开单 | ✅ |
| T2-06 | `TestS2T06_RestartRecoversUsingSameClientID` | 新 `Engine` 恢复、clientOrderId 不变、补 Fill 不重计 | ✅ |
| T2-07 | `TestS2T07_TransactionFailureNoInventedFills` | SQLite Trigger 中断 → 事务回滚、无幻影 Fill、不标 filled | ✅ |
| T2-08 | 合并在 T02 | 取消时新增成交也计入、先查→取消→再查 | ✅ |
| T2-09 | `TestS2T09_RateLimitAndMissingReadFailClosed` | 429/418/timeout 分类、`unknown`、新风险被拒 | ✅（**但未与终态冻结串接，见 S2-F1**） |
| T2-10 | `TestS2T10SellLimitUsesDistinctPrice` | localhost SDK 表单 `quantity=3, price=12` | ✅（测 helper；静态探针确认生产 `SellLimit` 调用同一 helper） |
| T2-11 | `TestS2T11_InvalidScopeAndExternalOrderRejected` | lead / SELL / 外部挂单 | ⚠️ 断言偏弱，见 S2-F4 |
| 额外 | 分页 505 条、页上限 10000、历史 Fill 冲突、跨币同报价预算、Fee 缺失、MARKET_LOT_SIZE=0、Accept/只读写禁、Decimal 与过滤器 | 10 项 | ✅ |

---

## 八、S1 整改基线复验（不采信「声明已修」，独立复跑）

| S1 项 | 本轮独立证据 | 结论 |
| --- | --- | --- |
| `RequestMode` 禁止跨模式切换 | `PROBE_MODE`：`legacy` 幂等 nil；`off` 返回 `legacy -> off` 对账要求；`strategy` 硬拒 | ✅ |
| `AuthorizeLegacyOrder` 来源 + 意图 | `PROBE_GATE` / `PROBE_INTENT`：未知来源、哨兵 `SPOT_RUSH`、空/未知意图全部拒绝；3 个白名单来源 + `new_risk/exit` 放行 | ✅ |
| 配置 API 固定 JSON / JWT 二次校验 | `controllers` S1 专项 `go test -run '^TestS1SpotStrategy'` PASS；`respond` 使用 `JSONResp` + `authorized()` | ✅ |
| 通知门禁不消费旧 SELL 信号 | `spot/s1_notice_gate_test.go` 在目标包测试中 PASS | ✅ |
| 旧下单构造器 AST 守卫 | `spot/api/binance/s1_guard_test.go` 在目标包测试中 PASS | ✅ |

---

## 九、生产 Gate（全部 BLOCKED，未授权）

| Gate | 状态 | 说明 |
| --- | --- | --- |
| 真实 Binance Spot `myTrades` 逐 `tradeId` 对照 | ⛔ BLOCKED | 分页 `fromId` 语义、跨页重复、累计量一致性仅在 Fake 验证 |
| 真实 `account` 余额 / `openOrders` / `free` `locked` 核对 | ⛔ BLOCKED | 仅 localhost fixture |
| 真实委托、撤单、部分成交与费用资产 | ⛔ BLOCKED | `Place/Cancel` 恒禁，S2 未执行任何真实写 |
| `strategy` / `off` 模式真实切换 | ⛔ BLOCKED | S1 整改后仍硬拒；旧退出职责、旧库存归属尚未核对 |
| 生产 MySQL 8 DDL / 索引 / EXPLAIN / P95 | ⛔ BLOCKED | v25 迁移只在 SQLite 与 DDL 文本验证 |
| 生产 v25 升级、二进制 + static + schema 成套发布与回滚 | ⛔ BLOCKED | 本轮未执行 `sync db`、未替换二进制、未重启服务 |
| T0 / 版本持久记录 / 二进制 hash | ⛔ BLOCKED | 沿用 01-ALL 未决项 |
| 旧 `eatRate` 手动路由停用确认 | ⛔ BLOCKED | S0 §7.3 要求，S2 未处理 |

---

## 十、整改与复评要求

1. **必须先闭环 S2-F1、S2-F2**（均 P1）：按第六节建议改代码，并各补一条能复现本轮探针场景的**永久回归测试**；F1 的测试必须覆盖「终态冻结 + 恢复 + 429 → 新 BUY 仍被拒」。
2. 建议同期处理 S2-F3（文档口径 + 审计）、S2-F4（T2-11 变体 + `QueryOrder` 身份双校验）、S2-F5（Prepare 脱敏）。
3. 复评时**另起 `CODE_REVIEW_02-S2-rereview.md`**，不覆盖本报告。
4. 复评通过前，S3 不得消费 `spot_strategy_fills` 生成 Holding/盈亏；不得把 S2 的 `rest_verified` 字段当作真实核实依据。
5. 无论本轮结论如何，**生产部署、真实 Binance 访问、模式切换、生产 schema 变更仍需单独明确授权**。

---

## 十一、审计纪律声明

本轮**未修改任何业务代码**，只新增本报告与项目记忆；未 commit/push、未改 `conf/app.conf`、未触碰 `static/` 与 `strategy_templates/`、未对真实库执行 `sync db`、未访问真实 Binance、未重启服务、未操作生产仓位。所有临时探针写入 `/tmp/phase_audit_s2` 隔离副本并在验证后删除，副本与主程序二进制 `/tmp/s2_main_bin` 在收尾时一并清理。报告中「离线 PASS」仅指本地 Go/SQLite/Fake/localhost fixture 验证，不代表任何实盘 Gate 通过。
