# V4-5 Phase 审计报告：Binance API Budget & Optimization

- **审计对象**：V4-5 实现**在工作区未提交**（新增 13 个文件约 1,776 行 + 测试约 800 行：账户/行情缓存、Kline/MarkPrice WS 缓存、trade-config 去重、cycle snapshot、budget coordinator、optimization 计数；改动 24 个文件）。基线 `HEAD = 80ba5bd`（Merge PR #58，V4-4 已合入 master）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（`git archive HEAD` + 拷贝全部未提交文件；探针与副本已删除，用户仓库未被改动；探针中一处仅用于测试的 setter 只存在于副本）
- **审查类型**：review-only
- **结论**：**已完成，可进入 V4-6**。§11 七条 Gate **全部通过（其中 6 条为实测）**，§16 安全 Gate 逐条核对通过；无 P0/P1；发现 **1 项 P2（空仓位下 Position 快照完全不缓存）+ 5 项 P3**。Schema 与 `app.conf` 均无变更。**V4-4 报告中的 F1（Record O(n) 复制）与 F6（截断无提示）已被本阶段修复并实测**。
- **审计日期**：2026-09-24（16:28）

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| V4-5 是否完成 | ✅ **完成**（账户快照复用、账户/行情缓存与 singleflight、User Data mirror generation 门控、P0–P3 预算、429/418 处理、看板指标） |
| §11 Gate 是否通过 | ✅ **七条全部通过**（"5 候选同开"压测场景以真实 HTTP 计数实测） |
| 是否可进入 V4-6 | ✅ **可以**（无阻塞项） |
| 是否降低交易安全 | ❌ **未降低**：开仓前 ownership 安全双检查保留、6 处 mutation 后缓存失效、uncertain 订单不降频、预算拒绝的 mutation 不进入 reconcile |
| DB / 配置变更 | ❌ 无（`models/`、`conf/`、`appversion` 均未改，Schema 仍 v18，无需 `sync db` —— 报告 §17 声明属实） |
| 是否存在 P0/P1 | ❌ 未发现 |
| V4-4 遗留处理 | ✅ V4-4-F1（事件存储 O(n) 复制）→ **环形缓冲实测 596µs → 328ns**；V4-4-F6（截断无提示）→ Snapshot 新增 `retained_events/dropped_events/last_dropped_at/truncated` 并实测生效；⚠️ V4-4-F2（Snapshot ~8ms 无缓存）**仍未处理**（本次复测 7.58ms） |

---

## 2. 验收 Gate 逐项核对（§11 七条）

| # | Gate | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | Account Position / OpenOrders 不随 Symbol 数线性重复查询 | ✅ **实测** | **探针（真实 HTTP 计数）**：一轮 5 候选 → 全账户 `positionRisk` **1 次**、全账户 `openOrders` **1 次**（旧行为各 6 次）；候选内为 symbol-specific 各 5 次。第二轮在 TTL 内 → 全账户读取仍为 1/1，`start_trade/cache_hit=2` ✓。代码侧：`StartTrade` 每轮只构建一次 `cycleAccount`（`feature.go:84`），候选循环改用 map 查询（`HasPosition/HasOpeningOrder`） |
| 2 | 不再出现每个候选都调用全账户 weight-40 OpenOrders | ✅ **实测** | 候选内 `openOrders` 请求**全部带 `symbol=`**（权重 1，`GetOpenOrderContext(ctx, symbol)`）；两轮 5 候选的估算总权重 = **105**（openOrders 50 + positionRisk 55），旧模式仅一轮就约 270（5×45 候选内 + 45 周期）→ **每轮约降 70%+**；`ensureAccountOpenSlotAvailable` 的无镜像回退分支显式注释了这一意图（`feature/ownership.go:312-330`） |
| 3 | Order Count 不越限 | ✅ **实测** | `orderBudgetExhausted`（10s/1m ≥95%）触发时，P0 下单**在 base RoundTripper 之前**被拒（探针：base 调用数保持 0），而 P1 账户读不受影响；`SetOrderLimits` 由 futures/spot/delivery 的 exchangeInfo 接线写入（`index.go:874`、`spot/...:99`、`delivery.go:52`）→ 防护**非死代码** ✓ |
| 4 | 429 时没有 duplicate order | ✅ **实测 + 永久测试** | 探针：429（`Retry-After`）→ 进入 `exchange_throttled`，此后 **P0 下单也在发送前被拒**；`service/futuresownership/execution.go:154` 对 `ErrBudgetDeferred` 明确置 `OrderFailed` 且**不进入** uncertain lookup/reconcile；永久测试 `execution_test.go:103-117` 断言 `submit/lookup` 调用计数 |
| 5 | Background task 在预算紧张时自动让路 | ✅ **实测** | critical（≥85%）时 P3（historical/exchange_info）被 defer、P2 被 defer、P1 放行；warning（≥70%）时 P2/P3 需串行占用单个低优槽且**取到槽后重新校验**（防止等待期间账户转为 critical 仍放行）；被 defer 的请求**不计入用量统计**（`window_5m.request_count` 不增长） |
| 6 | User Data WS stale 时能安全 fallback REST | ✅ **实测** | `futuresUserDataMirrorUsable` 四条件全部实测：开关关闭 → false；无连接（generation=0）→ false；**reconnect 后 generation 不匹配 → false**；full sync 36 分钟 → false；匹配且 34 分钟 → true。镜像不可用时 `GetTransformPositionsContext`/`getTransformOpenOrdersContext` 走 REST 分支，`ensureAccountOpenSlotAvailable` 回退为 **symbol-specific** 读取 |
| 7 | Ownership Safety 不因缓存/复用被削弱 | ✅ | ① 每次真实开仓仍调用 `ensureAccountOpenSlotAvailable`（仓位 + 开仓单双重校验，`ownership.go:481-506`）；② 6 处 mutation 后失效缓存（`CreateOwnedOrder`/`CreateOwnedAlgoOrder`/`CancelOrder`/`CancelAlgoOrder`/`SetLeverage`/`SetMarginType`，`index.go:634/664/684/698/715/734`）；③ auto-strategy 挂单 reconcile 的 5s 降频**仅**作用于「非 `reconcile_required` + 有 exchange order id + 近期成功 reconcile」的稳定单（`ownership.go:164-176`），uncertain/无 ID 一律 fail-closed；④ `InvalidateAccountReadCache` 用 **generation 守卫**，失效前已在途的旧读取无法回写缓存（永久测试 `account_read_cache_test.go:135`） |

**§16 安全 Gate 补充核对**：Kline gap → 立即失效缓存 + 下次 REST bootstrap（永久测试 `v45_market_cache_test.go:98`）✓；ticker WS stale → 1s TTL + singleflight REST fallback（`ticker_read_cache.go`）✓；stable active order 不再每 2 秒 GET ✓（`autoStrategyOrderRESTReconcileInterval = 5s` + `MarkOrderReconciled` 写入 `last_reconciled_at`）；budget defer 的 mutation 不产生 exchange lookup ✓。

**§12「本阶段不做」核对**：未引入无限 sleep（defer 是**拒绝**而非排队，除 warning 下低优请求的单槽串行）；未降低 Ownership 检查；未把 Trade Mutation 自动 retry；权重不由用户手工计算（本地估算仅用于排序，headers 为权威，`budgetStaleAfter=75s` 后自动清零避免陈旧信号）✓。

---

## 3. 实证过的隐式契约（探针，全部通过）

探针写在隔离副本（已删除），以 `-run TestProbeV45` 单独执行，并另跑一次去掉探针的包测试确认为 `ok`。

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-1** 预算在 base RoundTripper 之前拒绝 | critical 下的 P2 读取；P0 下单在 critical/权重层面放行；throttle 下 P0 被拒 | ✅ 被 defer 的请求 **base 调用数 = 0**、**不计入 window 统计**；P0 只在 throttle/order-count 场景被拒；`BudgetSnapshot` 暴露 `exchange_throttled` + `throttle_until` |
| **PR-2** Order Count 守卫 | `order_count_10s = 10/10` | ✅ 下单被拒（base 调用 0）且**与权重等级无关**（此时 level=normal）；P1 账户读放行；快照含 `order_limit_10s=10/order_limit_1m=100` |
| **PR-3** warning/critical 分级 | 75% 与 90% 两档 | ✅ warning 下 P2 放行（串行，`pending_weight` 调用后归零无泄漏）；critical 下 P3 defer、P1 放行；快照含 `budgets` + `optimizations` |
| **PR-4** V4-4-F1 复测 | 20k 事件稳态写入 + 窗口内溢出 60k | ✅ 稳态 **328ns/请求**（V4-4 为 **596µs**，约 1800×）；溢出时 `retained=50000 / dropped=15001 / truncated=true`（V4-4-F6 修复实证） |
| **PR-5** 镜像门控四条件 | 开关/连接/generation 匹配/35 分钟 | ✅ 四条件全部生效（见 Gate 6）；**reconnect 不继承旧信任** |
| **PR-6** cycle snapshot 语义 | 多空独立、零仓不计、撤销单不计、只认开仓方向、pending open 只记一次 | ✅ 全部符合；slot 计数沿用「所有 open-status 订单」语义（与旧 `positionCount + len(allOpenOrders)` 一致） |
| **PR-7** 5 候选压测（真实 HTTP） | 两轮完整 cycle | ✅ 见 Gate 1/2；估算权重 105/两轮；`cache_hit`/`prevented_duplicate` 计数进入看板快照 |
| **PR-8** 空仓位缓存（**新发现 F1**） | 空仓账户 3 次全账户读取 | ⚠️ `positionRisk` 全账户读取 **3 次**（TTL 内完全未命中），同期 `openOrders` 仅 1 次 → 见 F1 |
| 附：Snapshot 复测 | 20k 事件 / 120 键 | ⚠️ **7.58ms**（V4-4 为 8.24ms，环形缓冲只把复制移出热路径）→ 见 F2 |

---

## 4. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | **P2（API 浪费）** | **空仓位账户下 2 秒 Position 快照缓存完全失效**：`loadPositionSnapshot` 要求 `len(entry.rows) != 0`，而 `loadOpenOrdersSnapshot` 只要求 `rows != nil`（`account_read_cache.go:75/103`）→ 返回 `[]` 的合法空快照被当作"未命中"，每次调用都重新请求全账户 `positionRisk` | **探针 PR-8 实测**：TTL 内连续 3 次全账户 position 读取 = 3 次真实请求，同期 openOrders = 1 次。空仓是最常见状态 → 每 2 秒一轮 StartTrade 都会重复一次 weight-5 全账户查询（≈150 weight/分钟，约占 futures 2400/分钟预算的 6%），且 `cache_hit` 指标在空仓期失真。建议：把两处判空条件对齐（缓存"新鲜的空快照"），或在代码中注明空结果被刻意视为不可缓存的原因 |
| F2 | P3（沿用 V4-4） | `Snapshot` 仍为 O(事件数)：20k 事件 / 120 键 = **7.58ms**，且持锁复制事件切片 | 看板手动/低频刷新可接受；若后续做自动轮询或事件量继续增长，建议加 1s TTL 缓存或增量维护聚合 |
| F3 | P3（文档口径） | 报告 §3 表述"账户读取不再随候选数线性放大"——**严格说仅"全账户"读取不线性**；候选内仍各有 2 次 symbol-specific 读取（这是设计允许的最终安全检查） | 建议在报告/Phase 文档补一句边界说明（本次审计按此口径核对，结论一致） |
| F4 | P3 | `EstimateWeight` 未列出的路径按 1 计 → 本文中的权重数字（如 105）是**下界** | 与 V4-4 一致（headers 为权威）；建议看板标注"估计权重"口径 |
| F5 | P3 | Spot 的 `SetOrderLimits`/`SetWeightLimit` 只在获取 spot exchangeInfo 时写入；若 spot 未启用，spot 的 order-count 防护为 0（不生效） | spot 不在自动交易主流程，风险低；若要严格，可在启动时对已配置的 product 主动拉一次 exchangeInfo |
| F6 | P3（跨阶段遗留） | V4-2 两项（发布=管理员级提示词变更说明、`DraftStore.List` 静默跳过损坏 draft.json）与 V4-3 R1（模拟盘路径跟随 `FutureStrategyCoin` 的文档说明）仍未处理 | 与本阶段无关，仅作跟踪 |
| — | 观察 | `TradeCycleAccountSnapshot` 的 slot 计数包含 reduce-only/保护单（与旧语义一致），保护单多时会更早触达 `FutureMaxCount` | 非回归；若希望"仅开仓意图"计数，需与业务确认 |

---

## 5. 自动化验证结果（用户工作区原地执行）

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./...` | ✅ 无输出 |
| `go test -count=1 ./...` | ✅ **63 个包 ok，0 FAIL**（比 V4-4 多 1 个） |
| `go test -count=1 -race ./service/binanceapiusage ./feature/api/binance ./service/futuresownership` | ✅ 全部 ok |
| `gofmt -l` / `git diff HEAD --check`（V4-5 改动 Go 文件） | ✅ 无输出 / 无异常 |
| 隔离副本 `go build ./...` + 探针 PR-1～PR-8 + Snapshot 复测 | ✅ 全部通过（F1/F2 为量化观测） |
| 副本去掉探针后重跑受影响包 | ✅ binanceapiusage / feature / feature-api-binance / futuresownership 全 ok |
| 前端 | 产物交叉核对：`observability-*.js` 含全部新字段（`budgets`/`optimizations`/`cache_hits`/`prevented_duplicate_calls`/`coalesced_requests`/`local_ws_hits`/`deferred_requests`/`dropped_events`/`truncated`/`retained_events`/`pending_weight`/`throttle_until`/`exchange_throttled`）✓ 与 §13 一致 |

---

## 6. 审计边界与未验证项

- **未验证**：真实行情/真实账号下的运行数据（本次全部为代码 + 探针；真实 429/418 分布与权重峰值需线上采样）。
- **未验证**：User Data WS 真实连接/断线重连路径（`index.go:1383/1465/1470/1493/1496`）——本轮以「副本内临时 setter 写同一个 active generation 原子」的方式验证门控逻辑；**真实 socket 生命周期未跑**（生产需至少一次真实断线验证）。
- **未验证**：`WsUserData` 之外的市场 WS（ticker/kline/mark-price）真实断流行为；其 freshness/gap 逻辑由永久测试覆盖。
- **未验证**：Delivery 与 spot 主流程未跑（当前未启用）。
- 未对生产发起任何真实下单或配置修改。

---

## 7. 附：V4-5 交付物清单

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `feature/trade_cycle_snapshot.go`(+test) | 95(+47) | 单轮不可变账户快照：O(1) 候选查询、pending open 追加、slot 计数 |
| `feature/api/binance/account_read_cache.go`(+test) | 218(+167) | 全账户 Position(2s)/OpenOrders(5s) 快照 + singleflight + generation 失效 + Fresh 旁路 |
| `feature/api/binance/ticker_read_cache.go`(+test) | 106(+79) | ticker WS 优先、REST fallback 1s TTL + 单飞 + 上限 256 |
| `feature/api/binance/kline_ws_cache.go`、`mark_price_ws.go`、`market_read_cache.go`、`market_indicator_cache.go`、`trade_config_cache.go`(+test)、`v45_market_cache_test.go` | ~1,074 | Kline 规范缓存 + WS 增量 + gap 失效、Mark/Premium WS、Depth 750ms/OI 2s/ratio 30s 缓存、同配置 5s 去重与失效 |
| `service/binanceapiusage/budget.go`(+test)、`optimization.go`、`collector.go`/`transport.go`（改） | 324(+136)+64+45/−6+33/−6 | P0–P3 预算协调器、拒绝前置、限流/order-count 守卫、优化命中计数、事件环形缓冲与截断可见性 |
| `feature/ownership.go`、`feature/feature.go`、`feature/feature_{userdata,rush,notice,util}.go`、`service/futuresownership/{execution,service}.go`、`service/agenttrade/lifecycle.go`、`controllers/{account,eatRate,listenCoin,system_dashboard}.go`、`spot/*`、`main.go`、`feature/strategy/*` | 24 文件 | 镜像门控与回退、候选内 symbol-specific 安全确认、挂单 reconcile 降频、source/预算接线、spot WS 启动、精度本地优先、funding DB-first |
| 文档 | — | `05-phase-v4-5-*.md` §13 实施状态；新增 `v4-5-implementation-report.md`（17 节：静态调用矩阵 + 安全 Gate） |
