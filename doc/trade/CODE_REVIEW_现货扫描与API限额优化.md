# CODE_REVIEW_现货扫描与API限额优化：现货扫描 / 行情缓存 / 入口开关语义变更审查

日期：2026-11-16（22:18 起）
后端基线：`master`，HEAD **`993916a fix: update login`**（含 `v2.2.1` / `v2.2.2` tag），工作区 **16 项未提交改动**（13 M + 3 ??），本次审查对象即这批改动
前端：本轮无前端改动
用户声明的实现目标：**现货扫描与 API 限额优化**
变更说明依据：`doc/readme/zh-CN/03-现货量化交易.md`、`doc/readme/en/03-spot-quant.md`（均新增「现货扫描与 API 限额优化」小节）
副本：`/tmp/phase_audit_spotscan`（只写探针，审计结束删除）
本报告为**新增文件**，未覆盖任何历史 `CODE_REVIEW_*` 报告。

---

## 结论摘要

| 维度 | 结论 |
| --- | --- |
| 用户声明的技术目标（扫描优先级 + 行情缓存 + 限额优化） | **✅ 达成**：6 轮模拟调度 REST 从 `24h=6 / klines=12` 降至 `1 / 2`（各 −83.3%），TTL 失效重载、环境隔离、并发合并、失败不缓存均有实证 |
| 离线 Gate | **✅ PASS**：689 个 Go 文件全等；build/vet/diff-check/全量 73 ok + 13 无测试 + 0 FAIL、16 项专项、`count=5`、race 7 包、脚本 EXIT=0、opt-in 实网用例 SKIP |
| 新增/改写回归 | **✅ 16 项专项 PASS**，中英文文档同步 |
| **diff 夹带的权限语义变更** | **❌ 未获契约裁决**：双开关从「自动交易总闸」降级为「新 BUY 闸」，关闭开关后仍初始化真实交易所、启动 signed WS、每 2 秒扫描并自动 SELL → **F1 P1** |
| 生产 / 实盘 / 发布 Gate | **⛔ 全部 BLOCKED**（沿用 `CODE_REVIEW_06-7-ALL.md`，本次未解除任何一项） |
| 整体判定 | **⚠️ 有条件通过（研发基线）**：可合并本地代码；**不认可「可直接上线」，不认可「语义变更已完成验收」** |

一句话：**缓存与扫描这部分做得扎实、可量化、可回归；但同一批改动同时把「关闭开关」从停机动作改成了「只停新开仓」，这是任务标题没有体现的权限语义变更。项目里 AI 交易与带单都有独立的 kill switch，Spot Quant 现在没有任何能停止全部自动下单的开关；而 06-7-ALL 的 F1（主网放行需书面裁决与持久化授权）尚未闭环，本次改动进一步削弱了它。**

---

## 一、审计范围与基线

### 1.1 前置状态（承接上一轮）

`doc/trade/` 已入库，含 `CODE_REVIEW_删除ListenRateEat.md` 与 `CODE_REVIEW_删除ListenRateEat_复核.md`。复核结论已由**用户书面确认**：EatRate 功能从未实际使用、无历史订单，故 F1（遗留仓位）不适用、F2 保留旧表、F3 专项预检取消、F4 README 已修复；同时明确「通用发布检查及其他功能原有 Gate 不受此决定影响」。

> 因此本次审查中，**06-7 系列的生产 / 发布 Gate 仍全部 BLOCKED**；上次「功能从未使用」的裁决**不能**外推到本轮——现货量化已接线主网、是主动启用中的功能。

### 1.2 本次改动清单

`git diff --stat`：**13 files / +333 −101**（已跟踪文件），另有 3 个新增文件共 **402 行**：

| 文件 | 作用 |
| --- | --- |
| `spot/api/binance/spot_quant_market_cache.go`（新，149 行） | 每 `QuantAdapter` 独立的行情缓存：snapshot TTL 4s、Kline TTL ≤7s 且按最新 K 线 `CloseTime+1` 截断、上限 256、`singleflight` 合并并发、失败不缓存、Kline 深拷贝 |
| `spot/api/binance/spot_quant_market_cache_test.go`（新，161 行） | 6 项缓存单测：环境隔离、失败不缓存、并发合并、克隆/interval/limit 隔离、K 线边界、跨环境 |
| `spot/api/binance/spot_quant_market_adapter_cache_test.go`（新，92 行） | mock transport 验证主网/测试网各自只发 1 次 24h + 1 次 klines |
| `spot_quant_market.go` / `spot_quant_adapter.go` | `MarketSnapshot` / `MarketKlines` 拆出 `fetch*`，外层走缓存；主网额外用新鲜 WS 价覆盖 `Price`/`AsOf`；adapter 持有独立 cache |
| `feature/spot_quant_runner.go` | 新增 `managedSpotSymbols`（managed 仓位/未结订单 UNION 查询）；`Run` 先处理 managed 再扫启用币种、`visited` 去重；`runSymbol` 移除「币种开关 OFF 直接 return」，BUY gate 后移 |
| `feature/spot_quant_runtime.go` | 新增 `spotQuantNeedsRuntime`：全局 OFF 但存在 managed 数据时仍初始化 Runtime |
| `service/spotexecution/{processor,types,gate_db}.go` | `ExecuteSell` 移除 `p.enabled()`；`submitOnce` 中多处 gate 改为仅 `local.Side=="BUY"` 生效；接口注释改为 ENTRY gate |
| 4 个测试文件 + 中英文 readme | 回归语义改写、新增「扫描与限额」说明 |

### 1.3 隔离副本与等价校验

副本 = `git archive HEAD` + 同步 13 个 M + 3 个 ?? 新增文件（无 D）+ 副本内 `conf/app.conf.example → conf/app.conf` 兜底。

```
WORKTREE_GO 689  COPY_GO 689  IDENTICAL 689  MISSING 0  DIFFER 0  COPY_ONLY 0
```

剪枝沿用既有口径：`.local/` 私有探针、`strategy_templates/research/**`、`research_specs/` 不进入副本；全程用 `git ls-files` 清单逐文件 SHA256 比对，避免全仓 `os.walk` 触发 `SIGTERM(137)`。

---

## 二、离线 Gate 实证

显式清空 13 个 opt-in 许可 env（`BINANCE_TESTNET_*` / `SPOT_QUANT_TESTNET_*` / `SPOT_QUANT_VALIDATE_*` / `SPOT_LEDGER_MYSQL_DSN` / `SPOT_STRATEGY_LIVE_TRADING_APPROVAL`）。

| Gate | 结果 |
| --- | --- |
| `go build -o /tmp/... .` | **EXIT=0** |
| `go vet ./...` | **EXIT=0** |
| `git diff --check`（工作区） | **EXIT=0** |
| `go test ./... -count=1` | **EXIT=0，73 ok + 13 无测试包 + 0 FAIL** |
| **探针删除后复验** `go test ./... -count=1` | **EXIT=0，73 ok + 13 无测试包 + 0 FAIL**，build EXIT=0 |
| 专项回归（16 项） | `TestSpotQuantMarket*` 7 项 + `TestSpotQuantDisabled*` / `ManagedExits` / `RuntimeOff` / `TestReview064GateOff*` / 0673 双币 / 0672 开关 → **16 PASS / 0 FAIL** |
| 重复稳定 `go test -count=5 -p=2 ./spot/api/binance ./feature ./service/spotexecution` | **EXIT=0** |
| `go test -race -count=1`（7 个相关包） | **EXIT=0，无 DATA RACE** |
| `./scripts/test-go.sh -short` | **EXIT=0** |
| opt-in 实网用例 | 3 个真实 Testnet 用例全部 **SKIP**，标准回归不联网、不发单 |
| 前端 | 无改动，未运行 `tsc` / `eslint` |

> 全量测试的红点归因沿用既有结论：`conf/app.conf` 未入 Git、未跟踪 research runner 会让用户工作区 `go test ./...` 变红；副本用 example 兜底且不含 research 目录，**不代表「用户工作区全仓测试通过」**。

---

## 三、正向结论（已实证）

1. **缓存设计正确且可量化**。探针 `PROBE_SPOTSCAN_REST`：同一 adapter 连续 6 轮调用 `MarketSnapshot + MarketKlines(1m,100) + MarketKlines(5m,100)`，缓存版 `24h=1 / klines=2`，同结构但 `cache=nil` 的对照 `24h=6 / klines=12`，**各减少 83.3%**；手动令 snapshot 条目过期后 REST 计数从 1 升至 2（TTL 失效重载生效）；两个独立 adapter 各自 `24h=1`（环境隔离）。
2. **缓存边界稳妥**：失败响应不缓存、Kline 深拷贝防止调用方改写、`symbol|interval|limit` 键隔离、`CloseTime+1` 截断保证不跨 K 线边界复用、并发 24 路合并为 1 次 fetch；`singleflight` 依赖 `golang.org/x/sync v0.20.0`（已在 `go.mod` / `go.sum`）。
3. **真实请求仍受权重预算约束**：`fetchMarketSnapshot` / `fetchMarketKlines` 保留 `binanceapiusage.WithSource(ctx, "spot_quant_24h" / "spot_quant_klines")`，缓存命中则根本不发请求（探针计数为 0 增量）。
4. **扫描优先级实现**：`managedSpotSymbols`（`runner.go:354`）用 UNION 一次性列出需退出/对账的币种，`Run`（`:372`）先处理它们并用 `visited` 去重，再分页扫描启用币种；`BUY` 仍受 `DBGate` 双开关约束（`submitOnce` 的 `if local.Side == "BUY"`），`SELL` 仍以 06-3 Fill 投影 + wallet free 为准，未放宽资金校验。
5. **文档同步**：中英文 readme 均新增「现货扫描与 API 限额优化」小节（zh `:27` / en `:26`），写明 2 秒调度、managed 优先、4s/7s 缓存、环境隔离、REST 仍受 weight 预算与 429/418 退避保护。

---

## 四、隔离探针输出（3 组，全部 PASS）

```
PROBE_SPOTSCAN_OFF runtime_needed=true seeded_buys=1 buys=1 sells=1 run_errors=0
PROBE_SPOTSCAN_UNSETTLED global_off=true runtime_needed=true managed_symbols=[BTCUSDT]
                         review_cycles=5/5 reconciles=0 buys=0 sells=0
PROBE_SPOTSCAN_CLOSED_HINT symbols=[BTCUSDT] runtime_needed=true
PROBE_SPOTSCAN_REST cycles=6 cached{24h=1 klines=2} uncached{24h=6 klines=12}
                    saved_24h_pct=83.3 saved_klines_pct=83.3
PROBE_SPOTSCAN_TTL snapshot_after_expiry=2
PROBE_SPOTSCAN_ISOLATION adapterA_24h=1 adapterB_24h=1
PROBE_SPOTSCAN_STATIC sell_no_longer_checks_entry_gate=true buy_gate_before_post=true
PROBE_SPOTSCAN_STATIC 24h_rest_still_weight_budgeted=true klines_rest_still_weight_budgeted=true
PROBE_SPOTSCAN_FRESHNESS ticker_age_always_zero=true ws_override_sets_AsOf_to_local_now=true
PROBE_SPOTSCAN_CONTRACT doc_trade_06_7_plan_present=false
```

探针只写在副本内（同包 `_test.go` + 独立 `auditprobe/`），使用 mock transport / 临时 SQLite / 只读源码检查，**不联网、不访问真实库、不下单**；删除后已复验干净副本全量 0 FAIL。

---

## 五、Findings

### F1 · P1 · 关闭双开关不再停止自动 SELL，且不存在「停止全部自动下单」的替代开关

**代码事实**（本批改动）：

| 位置 | 变化 |
| --- | --- |
| `service/spotexecution/processor.go:135` `ExecuteSell` | 移除全部 `p.enabled(ctx, ...)`，改为仅 `ctx.Err()` |
| `processor.go:232,249,286` `submitOnce` | gate 检查改为 `if local.Side == "BUY" { p.enabled(...) }` |
| `service/spotexecution/types.go` / `gate_db.go` | `Gate` 接口与注释重定义为 ENTRY gate |
| `feature/spot_quant_runner.go:521` `runSymbol` | 移除 `coin.SpotQuantEnabled != 1` 的提前 return；退出/对账不再看开关 |
| `runner.go:372` `Run` | `!on && len(managed)==0` 才返回；OFF 但有 managed 时继续完整扫描 |
| `feature/spot_quant_runtime.go:127` `spotQuantNeedsRuntime` | 全局 OFF 但存在 managed 仓位/未结订单时返回 `true`，`spotQuantRunLive` 随即初始化真实 `QuantAdapter` + `Processor` 并 `go streamLoop(context.Background())`（`:185`） |
| `feature/spot_quant_runner_test.go` / `processor_test.go` | `TestReview064GateOffBeforeAnySellREST` 被改名并反转为 `TestReview064GateOffAllowsVerifiedSell`；0672 用例断言从「OFF 不得 SELL」改为「OFF 必须 SELL 1 次」 |

**探针实证**：全局与单币双 OFF 后 `runtime_needed=true`、4 轮扫描产生 1 次 SELL、`buys` 不增加；永久未结订单场景 5/5 轮持续进入扫描。

**契约与治理冲突**：

- 当前 master 的**用户可见文档**（中英文 readme）已同步新语义，这一点是正向的；但 `06-7` 主方案 §五 #8「总开关关闭不自动卖出」及 `CODE_REVIEW_06-7-ALL.md` 中「不自动卖出已实现 ✅」的验收结论仍在 `feat/trade_spot` 分支，**未随语义变更修订**（探针 `doc_trade_06_7_plan_present=false` 证实 master 上已无该方案）。
- `CODE_REVIEW_06-7-ALL.md` 的 **F1 P1**（主网自动下单路径解锁、实单授权缩减为双开关 + WS ACK + DBGate、缺持久化授权凭据与 append-only 审计、需用户书面裁决）**仍未闭环**。本次改动把「双开关」这最后一道用户可控闸门也限制为「只管新 BUY」，属于对该 P1 的进一步削弱，而非闭环。
- 项目内 `service/agenttrade/risk.go` 有 `kill_switch`、`service/leadaccount/*` 有 emergency halt，**Spot Quant 没有任何等价机制**。运维把 `config.spot_quant_enable=0` 当作紧急停机时，系统仍会连接交易所、启动签名 WS、每 2 秒扫描，并在退出条件成立时发出 MARKET SELL。
- 上次删除审查的「功能从未使用」裁决不适用于本条：现货量化是当前主动启用、已接线主网的功能。

**要求（部署前必须书面裁决，二选一）**：

1. **接受新语义**：同步修订 06-7 主方案 §五 #8 与 06-7-ALL 验收结论；补一个能停止**全部**自动下单的独立闸（含持久化授权、有效期、操作者、撤销与 append-only 审计）；在 readme 写明「关闭业务开关 ≠ 停机」及停机操作。
2. **恢复原语义**：让 OFF 停止所有自动下单，把「已有仓位退出」改为显式的、带授权的恢复动作（HTTP/CLI + 审计），而不是常驻自动行为。

裁决前：不得部署到 `testnet=false` 生产、不得打开主网双开关、不得把本批改动宣称为「仅性能优化」。

### F2 · P2 · OFF 状态下 Runtime / signed WS 一旦启动永不停止，与 API 限额优化目标相悖

`spotQuantRuntimeState` 是 `sync.Once` + `streamLoop(context.Background())`（`runtime.go:185`），循环内每轮先 `SyncSpotMarketServerTime`（REST），失败则 5 秒重试并重新订阅签名 WS，**没有终止条件、不受请求 ctx 或服务生命周期控制**。探针显示：一条永久 `FILLED AND filled_qty <> requested_qty` 的异常订单会让 OFF 状态 `spotQuantNeedsRuntime` 恒为 `true`、5/5 轮持续扫描并每轮返回 `ErrNeedsReview`。

缓存只覆盖 `MarketSnapshot` / `MarketKlines`，**不覆盖** `myTrades`、`GetOrder`、`serverTime`、WS listenKey 续期与重连。因此「关闭功能 + 遗留异常数据」这一组合会持续产生非缓存 REST 与连接开销，而文档只笼统写「实际每轮耗时取决于币种与 Binance 响应」。

**要求**：为恢复态设置终止条件或指数退避；对持续 `ErrNeedsReview` 提供告警与人工处置入口（同时闭环 06-ALL F5「异常恢复运维入口」）；补充「OFF + 遗留数据」场景的 API 权重基线。

### F3 · P2 · OFF 状态下 SELL 仍受 WS ready 门禁约束，无 REST-only 降级

`spotQuantLiveExecutor.ExecuteSell`（`runtime.go:34`）在 `wsReady == false` 时返回 `ErrDisabled`；`Reconcile` 允许 REST 只读，但没有受控的 REST-only 退出下单路径。当 Testnet 凭据失效、网络异常或签名订阅未 ACK 时，F1 承诺的「OFF 仍能平掉已有仓位」不成立：仓位既不会被自动退出，也没有合规恢复入口；错误还会被 `ReportSpotQuantScheduler` 按文本去重，运维侧可能只看到一条告警。

**要求**：在文档中明确该 fail-closed 边界，或提供带授权与审计的受控 REST-only 退出；补一条「WS 未 ready + OFF + managed 仓位」的回归与告警说明。

### F4 · P2 · 行情新鲜度门控结构性自证，缓存混合时间源放大了它

`feature/spot_quant_market_strategy.go` 把 `snapshot.UpdateTime = tick.AsOf.UnixMilli()`，并把同一个 `tick.AsOf` 作为 `LoadVerifiedSpotRuleEnvWithSource(..., at, ...)` 的 `now`；而 `feature/strategy/line/spot_rule_env.go:72` 的 `validSpotTick` 计算 `age := now.UnixMilli() - ms`。**两者同源，`age` 恒为 0**，该函数「拒绝陈旧 ticker」的注释与实际行为不符。探针 `ticker_age_always_zero=true`。

缓存进一步引入混合时间源：`fetchMarketSnapshot`（`market.go:62`）用本地 `time.Now().UTC()` 作 `AsOf` 并校验交易所 `v.CloseTime` 在 ±5 分钟内（这是仍有效的防线）；缓存命中返回最多 4s 前的 `AsOf`；主网 WS 覆盖时把 `Price` 换成 WS 最新价、`AsOf` 重设为本地 `time.Now()`，但 `Open/High/Low/PercentChange` 仍来自 4s 前的 24h stats。当前 4s TTL 使实际偏差可接受，但「本地时钟自证新鲜度」的设计缺陷应在缓存场景下修正。

**要求**：`now` 使用独立时钟（`time.Now()`）或让 `UpdateTime` 使用交易所 `v.CloseTime`；补一条「人为构造陈旧 24h stats 必须被拒」的回归，并验证缓存命中 + `LoadVerifiedSpotRuleEnvWithSource` 组合。

### F5 · P3 · 缓存实现边界细节

- 淘汰逻辑 `for key,item := range ... { if !item.until.After(now) || key != symbol { delete(...); break } }`：`key != symbol` 几乎恒真，会在超过 256 项时随机删一个非当前条目，**非 LRU**，可能淘汰仍有效的热条目。
- `singleflight` 使用第一个调用者的 `ctx` 执行整组 fetch；该 ctx 超时/取消会把失败传播给所有等待者。`SpotMarketStrategy` 用 15s timeout，风险有限但不为零。
- `NewSpotQuantAdapter()` 构造的 adapter `endpoint=""`，`marketOnly()` 恒失败；目前仅 `spot_quant_adapter_test.go` 调用，属既有问题，但本次给它加了 cache 容易造成「可用于行情」的误读。
- 缓存键不含环境，隔离依赖「每个 adapter 一个 cache」实例；探针已证两个 adapter 各自请求 1 次，设计成立，但需在跨环境复用同一 adapter 时复核。
- 无「本轮 REST 请求数 / `USED-WEIGHT-1M`」可观测指标；文档要求运维到系统看板观察，但没有缓存前后的基线值可比对。

### F6 · P3 · 每轮全量遍历历史周期的 DB 开销未被优化

`activeSpotPosition` 对每个 symbol 分页遍历**所有**历史 `spot_managed_positions` 并逐个 `Accounting`（逐笔 Fill 重放），2 秒一轮 × 启用币种数 × 历史周期数。缓存只优化 REST，不优化 DB；`managedSpotSymbols` 的 `remaining_qty > 0` 提示会把「status 已 CLOSED 但物化数量未清零」的行永久纳入扫描（探针 `PROBE_SPOTSCAN_CLOSED_HINT` 证实）。这沿用了 06-3「物化状态不可信、Fill 才是权威」的正确方向，但缺少范围限制与 MySQL 8 真实体量 EXPLAIN/P95 证据（06-ALL 历史欠账）。

### F7 · P3 · 文档与回归覆盖缺口

- readme 未写明「关闭开关后仍会启动 signed WS、仍消耗 API 权重、如何彻底停止」，与 F1 直接相关。
- 16 项专项覆盖了 OFF SELL、缓存 TTL/并发/环境隔离/克隆/K 线边界、扫描优先级、双币独立；**缺少**：缓存命中与 `LoadVerifiedSpotRuleEnvWithSource` 组合、权重预算计数回归、永久 `ErrNeedsReview` 的限额回归、WS 未 ready 的 OFF 退出降级。
- 中英文 readme 表述一致，章节编号与术语无冲突。

---

## 六、Gate 判定

| Gate | 状态 | 依据 |
| --- | --- | --- |
| 删除/改动完整性与副本等价 | ✅ PASS | 689 Go 文件 SHA256 全等 |
| 离线 Gate（build/vet/test/race/脚本/diff-check/count=5） | ✅ PASS | §二，全部 EXIT=0 |
| 缓存与扫描优先级的正向目标 | ✅ PASS | §三、§四，REST −83.3% |
| **入口开关语义变更的契约裁决** | **⛔ 未决** | F1，需用户书面二选一 |
| 真实 Spot Testnet / Mainnet 自动交易长稳、429/418、多币、BNB fee/dust | ⛔ BLOCKED | 沿用 06-7-ALL，opt-in 用例默认 SKIP |
| 生产 v25 Schema 检查单、MySQL 8 P95、ARM Linux 成套发布与回滚 | ⛔ BLOCKED | 沿用 06-7-ALL |
| 持久化部署授权 / append-only 模式切换审计、异常恢复运维入口 | ⛔ BLOCKED | 06-7-ALL F1 / 06-ALL F5 未闭，本轮 F2/F3 使其更紧迫 |

---

## 七、后续准入条件（按顺序）

1. **F1 书面裁决**（最高优先）：选择「修订历史验收标准 + 补真正的 kill switch」或「恢复 OFF 停止全部自动下单」。裁决前不得部署、不得打开主网双开关、不得将本批改动定性为纯性能优化。
2. **F2/F3 闭环**：给恢复态 Runtime/WS 设终止条件或退避；明确 WS 未 ready 时的 fail-closed 边界与受控恢复入口；补相应回归。
3. **F4 修正**：新鲜度 `now` 与 `UpdateTime` 解耦，补陈旧行情回归与缓存组合测试。
4. **F5/F6/F7 收尾**：缓存淘汰改 LRU、补 `USED-WEIGHT-1M` 可观测基线、DB 扫描范围与 MySQL 8 EXPLAIN/P95、readme 补充「OFF ≠ 停机」。
5. 之后才进入 06-7-ALL 既定生产 Gate 顺序：生产库只读预检七项 → MySQL 8 真实体量 EXPLAIN/P95 → ARM Linux 构建 + static + schema 成套发布与回滚 → 持久化部署授权 → 最后主网真实小额。**禁止因本次离线全绿或 Testnet 历史成功而跳过。**

---

## 八、最终结论与审计边界

**⚠️ 有条件通过（研发基线）**：认可「现货扫描与 API 限额优化」在代码与离线证据层面达成，缓存使行情 REST 下降 83.3%，扫描优先级、环境隔离、失败不缓存与 16 项回归均扎实；**❌ 不认可「本批改动可直接上线」或「语义变更已完成验收」**。F1（P1）是本轮的核心：diff 在标题之外把双开关降级为「新 BUY 闸」，而 06-7-ALL 的治理 Gate 尚未裁决。可以合并本地代码；不得部署到 `testnet=false` 生产、不得打开主网双开关、不得把关闭业务开关当作停机手段。

**审计边界**：未修改任何业务代码、未 commit/push、未改 `conf/app.conf`（副本用 `app.conf.example` 兜底）、未碰 `static/` / `strategy_templates/` / 前端；未对真实库执行 `sync db` 或预检 SQL（探针只作用于临时 SQLite 文件与 mock transport）；未访问真实 Binance、未下单、未重启服务、未操作生产仓位。隔离探针只写在副本内并已删除，删除后复验干净副本 `go test ./... -count=1` 为 73 ok / 0 FAIL；副本与 `/tmp/spotscan_*` 临时产物在审计结束清理。
