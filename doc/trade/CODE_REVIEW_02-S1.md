# Code Review：方案 02 · S1「现货策略配置、最小数据与安全门禁」

- 审计日期：2026-11-09
- 审计对象：**工作区未提交代码**。HEAD `e40b01a doc: update doc`（`feat/trade-upgrade`，`origin` 追踪引用 ahead 1），14 个已跟踪文件修改 + 6 个未跟踪新增文件/目录，其中 S1 方案文档自身也在改动内
- 方案依据：`doc/trade/开发方案-02-S1-配置数据与安全门禁.md`（工作区版 190 行，§6 八项已全部勾选、§8 为实施与验收记录）；上游 `开发方案-02-S0-现状审计与安全边界.md` §7.2/§7.3；总览 `开发方案-02-现货自定义策略交易.md` 与 `开发方案-总览与实施顺序.md`
- 副本：`/tmp/phase_audit_s1`（`git archive HEAD` + 同步 23 个改动文件，SHA256 逐项核对；排除未跟踪的 `strategy_templates/research/**`），探针写在副本内、结束随副本删除
- 报告性质：新文件，不覆盖 `CODE_REVIEW_01-*.md` 等历史报告

> **对「S1 本地验收完成、允许进入 S2」的口径说明**：本轮实证确认「本地代码 + v25 Schema + Mock 门禁」确实落地且离线 Gate 全绿；但方案 §6 第 3 项的后半段「**不得误终止必要平仓/退出**」未成立，且这正是方案 §1 写明「**S1 先停下重新审议**」的触发条件。因此本报告**不认可 S1→S2 条件已满足**。

---

## 一、结论先行

| 项目 | 结论 |
| --- | --- |
| §6 八项验收 | **7 项 PASS、1 项 FAIL**（第 3 项「不得误终止必要平仓/退出」） |
| 本地代码 / v25 Schema / 默认 OFF / fail-closed | ✅ 全部落地，可继续作为开发基线使用 |
| 离线 Gate（build / 目标包 test / race / vet / diff-check / 探针） | ✅ 目标包全绿；纯净副本 `go build ./...` 也 EXIT=0 |
| `go test -count=1 ./...` 全量 | ❌ EXIT=1（controllers 5 项既有红点 + tests 包缺 `conf/app.conf` panic），**不得写「全仓测试通过」** |
| 生产上线 Gate / `strategy` 模式 / 真实 Binance | ❌ **全部 BLOCKED，未授权** |
| **S1 → S2** | ⛔ **不放行**。先闭环 S1-F1（P1），再复评；F2/F3/F4 建议同期整改 |

**本轮新发现 6 项**：S1-F1（P1，`off` 无旧退出职责检查且静默消费旧自动 SELL 信号）、S1-F2（P2，新配置 API 重新引入 015-F1 的 `Ctx.Resp` 缺陷，8 种 Accept 头仅 4/8 返回 JSON）、S1-F3（P2，门禁不带「明确调用来源/意图」）、S1-F4（P2，新 API 无控制器级 JWT 二次校验、模式切换无审计）、S1-F5（P2，文档声明的索引/AST 守卫/构建红点缺少对应证据或存在逃逸面）、S1-F6（P3，expr 变量白名单、quote 后缀校验、全表 UPDATE、并发测试强度等偏差）。

**现在什么已经可用**：本地 Mock 与测试库层面，`spot_strategy_holdings/orders/fills` 三表（DECIMAL(36,18)、4 个补充索引、UNIQUE/PK 完整）、Schema 24→25 幂等迁移、默认 `legacy` + 每币 OFF、6 个旧 Spot 下单构造器的持久模式门禁、以及仅可配置的 4 个只读/配置 API 均可用且可复现。**尚不可用**：把 `off` 当作生产安全模式、进入 S2 的阶段放行、以及任何真实 Spot 委托。

---

## 二、审计基线与生产接线事实

| 事实 | 证据 |
| --- | --- |
| Schema 版本 | `appversion/version.go` = **25**；`command/db_update.go` 在 `oldVersion < 25 && newVersion >= 25` 与 `SyncDatabase` 的 `systemConfig.Version >= 25` 双路径调用 `ensureSpotStrategyV25` |
| 三张新表 | `service/spotstrategy/schema.go::SchemaDDL` 为唯一 DDL 真源；DTO 在 `models/spot_strategy.go` **不注册 ORM**，避免 `RunSyncdb` 把 DECIMAL 改成 VARCHAR（与 01 线 `tradefills.SchemaDDL` 做法一致，§8.2 设计修订合理） |
| 默认安全 | `models/tableStruct.go` `Config.SpotAutoTradeMode` 与 `models/spot_symbols.go` 两个新字段均为 `json:"-"`；`controllers/spot.go::Post`、`spot/spot.go::UpdateSymbolsTradePrecision` 显式写 0/"0" |
| 旧 API 隔离 | 探针 `PROBE_EDIT_RESULT enabled=1 budget="12.5"`：模拟 `SpotController.Edit` 的「查行 → BindJSON → 整行 Update」路径，旧 JSON 无法覆盖私有策略字段 |
| **生产依赖闭包** | `go list -deps . \| grep -c spotstrategy` = **1**。`spot/spot.go` 与 `spot/api/binance/index.go` 引入门禁，**S1 是真实生产运行路径变更，不是零接线**；这也是 S1-F1 影响面大的原因 |
| 静态资源 / 配置 | 未触碰 `static/`、`conf/app.conf`、`strategy_templates/`；本轮无前端改动（S4 范围） |

文档 §8.4 声称 `go build ./...` 因 `strategy_templates/research` 重复 `main` 失败。本轮在**排除该未跟踪目录**的纯净副本中 `go build ./...` 与 `go build -o /tmp/s1_main_bin .` 均 EXIT=0 —— 该红点属于工作区未跟踪产物，不是 S1 代码缺陷，交付说明应据此区分。

---

## 三、§6 验收清单逐项核对

| # | 方案验收项 | 本轮证据 | 结论 |
| --- | --- | --- | --- |
| 1 | 默认新策略全部 OFF；升级/首装/重启保持 legacy | 迁移幂等探针 `PROBE_MIGRATION_RESULT mode=legacy enabled=0 budget="0"`；`PROBE_DEFAULTS mode=legacy enabled=false execution=mock_only`；`db_update_test.go` v24→v25 断言；`main.go:240` 版本门禁 | ✅ |
| 2 | 主账户与 Owner 以外直接拒绝；不凭空产生实盘订单 | `ValidateScope` 仅接受 `main/auto_strategy`；4 个 API 端点全部校验；`s1_test.go::TestS1ScopeBudgetsAndExpr`；全仓无 `POST /order` 新路由 | ✅ |
| 3 | 仍保留的旧自动写入口都有守卫；**不得误终止必要平仓/退出** | 前半 ✅：6 个构造器全部调用 `AuthorizeLegacyOrder`，永久 AST 守卫 PASS。**后半 ❌**：见 S1-F1 | ❌ **FAIL** |
| 4 | 活跃委托/归属未知时切 `strategy` 明确拒绝 | `RequestMode` 仅允许 `legacy/off`，请求 `strategy` 硬拒；`s1_test.go` + 探针复核 | ✅ |
| 5 | 两个并行首次 BUY 只成功一次；重复 `signal_key/client_order_id` 不产生两笔 | 永久测试 PASS；真并发探针 `maxconns=8 ok=1 failed=1 rows=1`（由 `UNIQUE(pending_slot)` 保证） | ✅ |
| 6 | 3 张表结构、DECIMAL、索引、唯一约束、事务回滚；新库与升级库可复现 | 探针 `indexes=4/4`、3 表金额列 `DECIMAL(36,18)`、两次 `ensure` 幂等；`db_update_test.go` 覆盖 5 个 UNIQUE/PK | ✅（索引回归保护见 F5） |
| 7 | 模拟表达式可用、`short` 通知不等于平仓、现货监听不回归 | `ValidateRules` 拒 `short`、要求 `long+close_long`；`EvaluateRule` 纯 expr；`spot.ListenCoin` 未改 | ✅（变量白名单偏差见 F6） |
| 8 | 目标包单测/构建、`git diff --check`；标记已知失败，不宣称全仓绿 | 本轮独立复跑全部通过；§8.4 明确列出红点 | ✅ |

---

## 四、离线 Gate 实跑结果（均在隔离副本内）

| 检验 | 结果 |
| --- | --- |
| `go build ./...` | ✅ EXIT=0（纯净副本） |
| `go build -o /tmp/s1_main_bin .` | ✅ EXIT=0 |
| `go test -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./command` | ✅ 4/4 PASS |
| `go test -count=1 ./controllers -run '^(TestS1SpotStrategyPayloadWhitelist\|TestStage014\|TestStage015\|TestStage016)'` | ✅ PASS |
| `go test -count=1 -race ./service/spotstrategy ./spot ./spot/api/binance ./command` | ✅ 4/4 PASS（仅 macOS `LC_DYSYMTAB` 链接器噪音） |
| `go vet ./service/spotstrategy ./spot ./spot/api/binance ./command ./controllers` | ✅ EXIT=0 |
| `git diff --check`（用户工作区只读） | ✅ EXIT=0 |
| `go test -count=1 ./...` | ❌ EXIT=1：`controllers` 5 项（`TestQueryStrategyTemplatePageOrdersByIDDescending`、AI runtime 3 项、`TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals` 报 `lstat ../strategy_templates: no such file`） + `tests` 包 `the target config file: conf/app.conf don't exist` panic |

**红点归因**：全部为既有策略模板 fixture / 干净副本缺配置与新目录，与 S1 改动无关；文档 §8.4 已如实列出，本轮复现一致。探针删除后再跑目标包与 controllers 专项仍全绿，确认探针未污染结论。

文档 §8.4 声称的「真实 `go_bn_test` 执行 `sync db` 24→25」与「MySQL 只读物理验证」**按记录采信，本轮未复跑**（本轮只在隔离 SQLite 复现迁移），不把用户自述当成本轮实证。

---

## 五、探针实证的隐式契约

所有探针写在 `/tmp/phase_audit_s1` 内、以 `-run TestProbeS1` 单独执行，随副本删除。

| 探针 | 输出 | 结论 |
| --- | --- | --- |
| `PROBE_GATE` | `fail_closed=6/6`（off / strategy / unknown / "" / 空 config / 空 symbol 全部拒绝） | 门禁持久化与 fail-closed 成立 ✅ |
| `PROBE_OFF` | `orders=0 holdings=0 switch=allowed exit_writers=blocked reconciliation_queries_in_RequestMode=0` | **F1 直接证据**：无任何账户/订单/库存证据即可切 off，且切完旧卖出被全面阻断 |
| `PROBE_ORDER` | `notice_has_notice_assign=204 notice_gate=210 notice_continue=212 rush_gate=47 rush_first_trade=61` | **F1 加重证据**：旧通知链先落库消费信号、后门禁 `continue` |
| `PROBE_ACCEPT` | `fixed_json=4/8`；x-yaml→YAML、xml/text-xml→500 `xml: unsupported type`、x-protobuf→panic `not protoreflect.ProtoMessage` | **F2 直接证据** |
| `PROBE_AUTH` | `controller_methods=4 claims_getdata_calls=0` | **F4 直接证据**：无控制器级二次校验 |
| `PROBE_MIGRATION` | `indexes=4/4 mode=legacy enabled=0 budget="0"`；3 表 DECIMAL(36,18)；两次 ensure 幂等 | §8.2 声明属实，但永久测试未覆盖（F5） |
| `PROBE_CONCURRENCY` | `maxconns=8 ok=1 failed=1 rows=1` | 唯一槽位在真并发下有效（永久测试强度不足，F6） |
| `PROBE_EDIT` | `enabled=1 budget="12.5"` | 旧 `/spots/:id` 整行 Update 不覆盖私有字段 ✅ |
| `PROBE_EXPR` | `config_time=accepted eval_time_err=unknown name UnknownVar` | 配置期与运行期变量环境不一致（F6） |
| `PROBE_DEFAULTS` | `mode=legacy enabled=false execution=mock_only scope=main/auto_strategy` | 默认安全、无伪造实盘授权 ✅ |

---

## 六、Findings

### S1-F1 — P1｜`off` 切换无旧退出职责检查，且会静默消费旧自动 SELL 信号

**事实链**：

1. `service/spotstrategy/config.go:225-244` 的 `RequestMode` 只做 `mode ∈ {legacy, off}` 白名单，随后 `UPDATE config SET spot_auto_trade_mode = ?`（无 WHERE、无前置查询）。探针实证：在 0 订单、0 持仓、无余额证据时切 off 仍成功，函数体内 **0 个**对账/查询调用。
2. `AuthorizeLegacyOrder(symbol)`（`config.go:110-122`）只判断 `mode == legacy`，**不区分买/卖、新开/退出**。6 个构造器全部经它：`index.go:141/162/181/202/224/249`，其中 `SellLimit`、`SellMarket`、`OrderTakeProfit`、`OrderStopLoss` 是**卖出与保护/退出**路径。
3. `spot/spot.go:203-213`：`NoticeAndAutoOrder` 在 `canOrder` 时**先**写 `coin.HasNotice=1; coin.Enable=0; orm.NewOrm().Update(&coin)`（204-206 行），**再**检查门禁（210 行），被阻断则 `continue`（212 行）。因此 off/strategy 下，一条待触发的旧自动 SELL 会被标记为「已通知 + 已禁用」而**永远不再重试**，即使随后切回 legacy。
4. `spot/spot.go:47-50`：`TryRush` 在 mode≠legacy 时整函数提前 return，跳过 61/68 行的 buy/sell；此路径不落库，恢复 legacy 后仍可重试，相对温和。

**与方案的冲突**：

- S1 §4：「`off`：拒绝任何自动新开风险；保留受控只读/恢复路径，**安全退出职责依据 S0 做专门判断，不能简单全停**」。
- S1 §1：「若单一 `legacy|strategy|off` 模式无法安全表达**保留功能的**安全退出，**S1 先停下重新审议**，不可硬切断保护/退出」。
- S0 §7.3「额外锁定条件」：「如果旧流程还负有必要 SELL 退出职责，**不得简单全停，应拒绝模式切换并提示用户先处理**」；S0 §7.2 已预警旧通知链「委托失败也可能失去自动重试机会」。
- §8.3 只以文字说明「off 不等于清仓/旧单已取消」，没有代码级前置检查或拒绝路径，把「切断退出能力」与「不主动撤单」混为一谈。

**风险**：全局 JWT 下任何已登录操作员一次 `PUT /spots/strategy/mode {"mode":"off"}` 即可停止旧自动卖出/退出链；由于门禁位于真实生产下单路径（`go list -deps .` 闭包内），影响直接作用于运行中的现货账户，且消费过的信号不可自动恢复。

**建议（按优先级）**：

1. 门禁参数化为 `(source, symbol, intent)`，`intent` 至少区分 `new_risk` / `exit`；`off` 下只允许已确认归属的旧退出，不允许新开。
2. 切换 off 前执行 S0 §7.6 清单（旧活动委托、库存、余额、外部/弃用 `eatRate`）；证据不足**拒绝切换**并给出机器可读原因，而非当前无条件放行。
3. 把 `has_notice/enable` 落库移到门禁通过之后（或改为「门禁失败不消费信号」），保证恢复 legacy 后信号仍可重放。
4. 在 S2 恢复链落地前，考虑把 off 设为只读/需显式人工确认的状态；并补充「切换 mode 后旧退出链行为」的端到端回归测试。

### S1-F2 — P2｜新配置 API 重新引入 01-5 已修复的 `Ctx.Resp` 契约缺陷

`controllers/spot_strategy.go` 有 11 处 `c.Ctx.Resp(...)`，其中 `Get`/`Put` 直接传 `map[string]interface{}`；`utils.ResJson` 也返回 map。探针 8/8 实测仅 4/8 返回 JSON：`application/x-yaml` 返回 YAML，`application/xml`/`text/xml` 返回 **HTTP 500**，`application/x-protobuf` **panic**。01-5 整改已把 `OrderController` 改为 `ctrl.JSONResp(...)` 并在 `controllers/orders_test.go` 补永久测试（`CODE_REVIEW_01-5_整改记录.md` 015-F1），S1 未复用该范式。

附带问题：错误响应只写 body 里的 `code:400`，HTTP status 仍为 200（探针 `status=200`），非浏览器客户端需额外解析 body。

**建议**：全部改用 `c.JSONResp(...)` 并 `c.Ctx.Output.SetStatus(...)`；把 8 种 Accept 头的契约测试固化为永久测试。

### S1-F3 — P2｜门禁不带「明确调用来源」，无法按意图放行退出

方案 §4 与 S0 §7.3 均要求「所有 Spot 下单入口接收**明确调用来源**」「调用者带明确来源」。实现只有 `symbol string`：`spot/spot.go:47` 甚至传哨兵 `"SPOT_RUSH"` 而非真实 symbol，说明该参数退化为非空占位符。这直接导致 F1 无法按「新开 vs 退出」区别放行，也无法按来源做审计/限流。

**建议**：`AuthorizeLegacyOrder(source, symbol, intent)`，6 个构造器与 2 个定时任务逐一标注来源；AST 守卫相应升级为校验参数而不仅是「函数内出现过调用」。

### S1-F4 — P2｜新 API 无控制器级 JWT 二次校验，模式切换无审计

`middlewares/auth.go::excludeRoutes` **不含** `/spots/strategy/*`，全局 JWT 覆盖成立 ✅；但探针 `claims_getdata_calls=0` 表明控制器未做二次校验，与 01-5 整改后 `OrderController.authorized()` 的基线不一致。`PUT /spots/strategy/mode` 能改变生产交易行为，却没有任何审计表写入或结构化日志（`RequestMode` 仅一条 UPDATE）。

**建议**：补 `authorized()` 单操作员防御（要求有效 `Username` claim）、拒绝环境 cookie/非 JSON 请求，并落一条 mode 变更审计（操作者、时间、旧值→新值）。

### S1-F5 — P2｜文档声明与永久测试证据存在缺口 / 守卫有逃逸面

1. §8.2 声称 4 个补充查询索引，`command/db_update_test.go` 新增断言**只覆盖 5 个 UNIQUE/PK**；本轮探针独立实证索引确实存在（4/4），但没有防误删的回归保护。
2. §8.5 的 `s1_guard_test.go` 只检查 6 个具名函数体内「出现过」`spotstrategy.AuthorizeLegacyOrder`，不校验调用是否位于所有返回路径之前、是否被条件包裹，也不扫描其它文件里直接 `client.NewCreateOrderService()` 的新增写路径。本轮全仓检索确认 `spot/` 下 6 处 `NewCreateOrderService` 全部有门禁 ✅，但守卫本身可被「新增构造器名 / 换文件」绕过。建议改为**白名单文件 + 全仓 AST 扫描**（或依赖闭包断言）。
3. 当前 `spot/` 包**没有** `NewCancelOrderService` 封装（S0 §7.5 已记录、S2 补），因此方案 §4 提到「Create/**Cancel** 前后统一 Gate」的 Cancel 部分本阶段无对应实现 —— 不是当前缺陷，但 S2 接入撤单时必须同步纳入门禁，不能沿用「只守 6 个构造器」的假设。

### S1-F6 — P3｜实现偏差与测试强度

| 项 | 事实 | 建议 |
| --- | --- | --- |
| expr 变量白名单 | `ValidateRules` 用 `expr.AllowUndefinedVariables()`，S0 §7.4 要求明确白名单；探针证明配置期接受 `UnknownVar`、运行期 `EvaluateRule` 报 `unknown name` | 用统一的 `expr.Env` 变量表在配置期编译，避免「配置通过、执行报错」 |
| quote 校验 | `UpdateSetting` 用 `strings.HasSuffix(coin.Symbol, coin.Type)`；S0 §7.4 要求按 `exchangeInfo` 的 `baseAsset/quoteAsset` 验证，不靠后缀 | S2 接 `GetExchangeInfo` 后改为元数据校验 |
| 全表 UPDATE | `RequestMode` 无 WHERE，`RowsAffected()==0` 时靠 `CurrentMode()`（同样 `LIMIT 1` 无 WHERE）回退；多行 `config` 时语义含糊 | 明确单行约束或加 `WHERE id = ?` |
| 并发测试强度 | 永久测试 `db.SetMaxOpenConns(1)` 把两个 goroutine 串行化；探针用 8 连接真并发仍 1 行，说明约束正确但测试不足 | 提高连接数并断言真实并发 |
| 错误状态与异常路径 | `controllers/spot.go::Edit` 忽略 `One()` 错误，查询失败时零值整行 Update 可能把私有字段写回 0/""；正常路径探针已证明 `json:"-"` 有效 | 检查 `One()` 错误并改为字段白名单 Update |
| 运维语义 | `main.go:240` 在 `config.Version < 25` 时 panic，避免 v25 二进制跑 v24 库；但迁移中 mode 为空时 `CurrentMode()` fail-closed 会拒所有 legacy 单 | 在发布 SOP 写明「先 sync db、校验 mode=legacy，再启动」 |

---

## 七、生产 Gate 与 S1 → S2 判定

| Gate（§8.6） | 本轮复核 |
| --- | --- |
| S1 本地代码 + Mock/Schema | ✅ 代码与 SQLite 迁移确实 PASS；**但 §6 第 3 项未过** |
| 实际 Spot 余额/挂单/旧退出职责核对 | **BLOCKED**（未授权、未读真实账户/DB） |
| `strategy` 模式开启 | **BLOCKED**，服务端硬拒行为正确 ✅ |
| 现货真实买卖/撤单/资金安全 | **BLOCKED** |
| 全仓 controllers/research 红点 | **OPEN**，5 项既有 + tests 包环境 panic |

**判定**：§8.6 写「S1 → S2 结论：允许进入 S2 的本地开发与 Mock 测试」。就「代码可以继续写」而言可以接受；但方案 §6 的 **S1 → S2 条件**写的是「账户模式和真实下单守卫**全覆盖**」，而 F1/F3 证明门禁既未覆盖「安全退出」语义、也未覆盖「调用来源」契约。因此：

> **本报告不认可 S1 已通过阶段 Gate。建议先闭环 S1-F1 并做一次复评（报告另起 `CODE_REVIEW_02-S1-rereview.md`），再宣布进入 S2。** 在此之前不得部署 v25 二进制、不得在生产/真实账户切换 `off`、不得开启 `strategy` 或发送任何真实 Spot 委托。

---

## 八、纪律与边界声明

- 本报告只代表**离线 PASS**（纯净副本 build / 目标包 test / race / vet / diff-check + SQLite 迁移与临时探针 + 源码/AST 静态核对）。**实盘 Gate / 生产部署 / T0 一律未通过、未授权**。
- 未修改任何业务代码；未 commit/push；未改 `conf/app.conf`；未对真实库执行 `sync db`；未访问真实 Binance；未重启服务；未操作生产仓位；未触碰 `strategy_templates/`、`static/` 与前端仓库；未把临时探针留在用户仓库。
- `conf/app.conf` 含连接信息且干净副本缺失，本轮未尝试读取或构造真实凭据；`tests` 包据此 panic，结论已降级为「环境红点」。
- 副本 `/tmp/phase_audit_s1` 与 `/tmp/s1_main_bin` 随收尾删除；探针日志仅用于本报告，不入库。

---

## 附录：关键证据位置索引

| 位置 | 内容 |
| --- | --- |
| `service/spotstrategy/config.go:110-122` | `AuthorizeLegacyOrder` 只按 mode 判断，无来源/意图 |
| `service/spotstrategy/config.go:225-244` | `RequestMode` 白名单 + 无 WHERE 全表 UPDATE |
| `spot/spot.go:47-50, 203-213` | TryRush 提前 return / 通知链先落库消费信号再门禁 |
| `spot/api/binance/index.go:141,162,181,202,224,249` | 6 个构造器门禁（含 4 条卖出/退出路径） |
| `spot/api/binance/s1_guard_test.go:12-51` | 永久 AST 守卫（仅名字/出现性检查） |
| `controllers/spot_strategy.go:19-96` | 4 个配置/模式端点，11 处 `Ctx.Resp` |
| `service/spotstrategy/schema.go:16-92` | 3 表 DDL、DECIMAL(36,18)、UNIQUE/PK、`SchemaFor` |
| `command/spot_strategy_v25_migration.go:11-72` | 补列、3 表、4 个补充索引、幂等 |
| `command/db_update.go:372-376, 446-450` | v25 双路径接线 |
| `models/spot_symbols.go:10-11` / `models/tableStruct.go:8` | `json:"-"` 私有字段隔离 |
| `middlewares/auth.go:19-28` | `excludeRoutes` 不含 `/spots/strategy/*` |
| `main.go:68, 240` | `CopyRequestBody = true` / Schema 版本 panic 门禁 |
