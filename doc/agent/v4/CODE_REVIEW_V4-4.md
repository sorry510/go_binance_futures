# V4-4 Phase 审计报告：Binance API Usage Observability

- **审计对象**：V4-4 实现**在工作区未提交**（新增 `service/binanceapiusage/` 894 行；改动 15 个文件 `+320/-119`，含 futures/spot/delivery API 封装、feature/ownership、ownership service、agenttrade、market、historicalmarket、controllers、systemhealth、router）。基线 `HEAD = b928c0d`（Merge PR #57，V4-2/V4-3 已合入 master）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（`git archive HEAD` + 拷贝全部未提交文件；探针与副本已删除，用户仓库未被改动）
- **审查类型**：review-only
- **结论**：**实现已完成，§9 六条 Gate 全部通过；但 §8「真实运行报告」的数据部分仍待采样**（文档自身已声明完成规则）。未发现 P0/P1 安全缺陷；发现 **1 项 P2（观测层成本随事件量线性增长）+ 5 项 P3**。可进入 V4-5 开发，但 V4-5 的优化优先级需等真实采样结果。
- **审计日期**：2026-09-23（22:35）

> **处理更新（2026-09-23）**：F1 已修复：Collector 改为固定容量 50,000 的 O(1) ring buffer，Record 路径不再在全局锁内复制窗口切片；按“先填满 5 万事件，再连续写 1 万次”的临时微基准，当前平均写入约 106ns/次。F6 已修复：Snapshot 新增 `truncated / dropped_events / last_dropped_at / retained_events`，Dashboard 在最近 5 分钟发生截断时明确警告。F3 的 Dashboard 文案已进一步明确“estimated weight 仅用于 endpoint 排序，不是权威限额”。F2（低频 Snapshot 成本）继续保留为 P3；F4 的跨 source 合并视图放到 V4-5，避免提前改变本阶段的数据模型；F5 仍需重启最新 binary 后完成真实 5 分钟采样。F7 属旧阶段跟踪项，其中 V4-2/V4-3 文档语义此前已完成收尾，不作为 V4-4 阻塞项。

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| V4-4 代码实现是否完成 | ✅ **完成**（统一 Transport 观测、四类客户端覆盖、rolling window、header 权威用量、动态限额分母、429/418、source attribution、read/trade 区分、只读 API 与看板区块） |
| §9 Gate 是否通过 | ✅ **六条全部通过**（其中 4 条为**端到端实证**，详见 §3） |
| §8 真实运行报告 | ⚠️ **部分完成**：Top20 与 1m 峰值仍为占位符；文档 §11/§7 明确"采样完成前不猜测"并要求重启新二进制后采集 ≥5 分钟窗口 |
| 是否可进入 V4-5 | ✅ **可以**（代码侧无阻塞）；但 V4-5 的优化清单需以真实采样数据为依据 |
| 是否改变业务行为 | ❌ **未改变**（15 个改动文件逐一 `git diff -w` 核对：仅 context 透传、客户端包装、新增端点、限额写入；无请求节奏/重试/缓存/sleep/交易决策变更） |
| DB / 配置变更 | ❌ 无（纯内存观测，Schema 仍 v18） |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 验收 Gate 逐项核对（§9 六条）

| # | Gate | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | 所有 go-binance REST 请求基本都能被统一观测 | ✅ **实证** | 全覆盖核对：`futures.NewClient`→`WrapClient`（`feature/api/binance/index.go:135,148`）、`delivery.NewClient`→`WrapClient`（`:142,151`）、`binance.NewClient`（spot）→`WrapClient`（`spot/api/binance/index.go:39-40`）、系统看板健康检查客户端→`WrapClient`（`service/systemhealth/service.go:189`）；全仓再无其它 Binance 客户端创建点。**探针 PR-1**：用**真实 go-binance 客户端**（`futures.NewClient`）+ 假 Transport 发起签名 `GET /fapi/v2/account`、`POST /fapi/v1/leverage`、`GET /fapi/v1/depth?limit=500`，三个请求均被记录，`path/weight(5,1,10)/type(read,trade)/source/product/environment` 全部正确 |
| 2 | WebSocket 不计入 REST weight | ✅ | WS 走 `feature/api/binance/proxy.go` 的包级 `futures.WsUserDataServe/WsKlineServe/delivery.WsAllMarketTickerServe`（自带 dialer），**不经过 HTTP `RoundTripper`**；唯一经观测层的相关 REST 是 `GetListenKey`（本应计入 weight 1） |
| 3 | signed query 不泄露 | ✅ **实证** | 实现：只持久化 `req.URL.Path`（`transport.go:86`）+ `normalizePath` 二次剥离 `?`（`collector.go:425-437`）；传输错误统一记 `"transport_error"`（`transport.go:93-97`，附注释说明原因）。**探针 PR-1**：线上 URL 含 `timestamp=…&signature=…`，而快照 JSON 中**不含** `signature`/`timestamp`/`recvWindow`/API key/secret，且所有 endpoint `Path` 无 `?`。永久测试另有 2 条（`transport_test.go:20/83`） |
| 4 | 429 / 418 可在 Dashboard 明确看到 | ✅ **实证** | **探针 PR-2**：连续 429（`Retry-After: 30`）与 418 后，`window_5m` 计数 429=1/418=1、单 endpoint 行 `count_429/count_418/error_count` 正确、`recent_rate_limits` 2 条（含 `retry_after=30` 与 `source/path` 归属）、限额状态写入 `last_rate_limited_at` 与 `retry_after` |
| 5 | Trade/Testnet 请求和普通 Read 请求可区分 | ✅ **实证** | `RequestType` 只在 POST/PUT/DELETE 且路径命中交易端点时为 `trade`（`weights.go:133-156`）。**探针 PR-1**：`POST /fapi/v1/leverage` → `trade` + `testnet`；`GET /fapi/v2/account` → `read` + `mainnet`；**两个环境的限额状态相互独立**（`ExchangeLimits` 2 条） |
| 6 | 观测本身不会明显增加交易延迟 | ✅（附 F1 成本增长提示） | **探针 PR-3**：包装层单请求开销 **4.539µs**（500 次均值，含真实 go-binance 请求构造）；`Record` 在低量级为亚微秒。**但成本随窗口内事件数线性增长**（见 F1），当前项目量级下约 ~90µs/请求，可接受 |

**§3–§8 要求核对**

| 要求 | 判定 | 证据 |
| --- | --- | --- |
| §2 在 Transport 层统一包装（非逐 wrapper 埋点） | ✅ | `WrapClient` 克隆客户端并包装其 Transport（`transport.go:43-57`），位于现有代理 transport **外层**；代理轮询/SOCKS/签名/时间戳重试路径不变 |
| §2/§3 Testnet 与 Mainnet 分开统计 | ✅ | futures 按 `binance::testnet` 判定（`index.go:143-147`）；spot/delivery 固定 mainnet；health 检查同样按 testnet 判定（`systemhealth/service.go:183-188`） |
| §3 采集字段齐全 | ✅ | `RequestEvent`/`ExchangeLimitState`/`RateLimitEvent`（`collector.go:18-59`）覆盖 product/env/source/type/method/path/status/latency/estimated weight/429/418/last error；header 侧读取 `X-MBX-USED-WEIGHT-1M`、`X-MBX-ORDER-COUNT-10S/1M`、`Retry-After`（`transport.go:69-72`） |
| §3 禁记项 | ✅ **实证** | 见 Gate 3 |
| §4 权威用量优先 response headers | ✅ **实证** | **探针 PR-4**：`X-MBX-USED-WEIGHT-1M=900`/`ORDER-COUNT-10S=7`/`1M=11` 被解析并保留；**第二次响应缺 header 时保留上次值**（不归零）；`last_status_code/last_response_at` 更新 |
| §4 限额分母：exchangeInfo 优先、reference 兜底 | ✅ **实证** | 三端均接 `SetWeightLimit(..., "exchange_info")`（`index.go:694`、`delivery.go:39`、`spot/...:67`）；reference 兜底 futures/delivery 2400、spot 6000（`collector.go:439-448`）。**探针 PR-4**：`SetWeightLimit(futures,mainnet,3000)` 后 `limit_source=exchange_info`、`weight_limit_1m=3000`、百分比按新分母重算（900/3000=30.00%）；PR-1 另一路径显示 reference 兜底时 1234/2400=51.42% |
| §5 内存 rolling window，不逐请求写库 | ✅ | `sync.Mutex` + 事件切片（5 分钟、上限 5 万）+ 限额 map + 限流环形日志（50）；无任何 DB 写入 |
| §6 看板区块 + 只读 API | ✅ | 新增 `GET /system/binance-api-usage`（`routers/router.go`、`controllers/system_dashboard.go:BinanceAPIUsage`），**仅读内存快照、不调用 Binance**；`service/systemhealth` 亦已纳入观测（`system_health` 标签） |
| §7 Source attribution | ✅ **实证** | 标签注入点：`start_trade`（`feature.go:52`、`ownership.go:81/428`）、`ownership_reconcile`（`reconcile.go:46`）、`agent_trade`（`agenttrade/default.go` ×4）、`historical_market`（`binance_source.go` ×3）、`funding_rate`（`feature.go:1000`）、`market_intelligence`（`market/service.go` ×6）、`manual_api`（`account.go` ×3、`listenCoin.go` ×2）、`system_health`、owner 映射 `new_coin_rush`/`notice_auto_order`（`execution.go:ownerAPISource`）。**关键机制**：go-binance `callAPI` 执行 `req.WithContext(ctx)`（模块源码 `client.go:365`）→ context 标签确实到达 Transport；**探针 PR-1 实证** `start_trade`/`agent_trade` 出现在记录中；未打标调用回落配置默认 `go_binance` ✓ |
| §8 输出报告 | ⚠️ 部分 | `v4-4-binance-api-usage-report.md` 已含：放大链路分析（§2 A–E，明确标注为**代码审查结论而非运行时排名**）、必须保留实时调用的清单（§3）、V4-5 可复用候选（§4）、采样方法（§1）；**Top20 与 1m 峰值为占位符**，且文档 §7 明确"采样前不猜测" ✓ 诚实 |
| §10 本阶段不做 | ✅ 未越界 | 未改 Rate Limit、未全局 sleep、未自动重试交易 Mutation；`git diff -w` 逐文件确认无业务逻辑改动 |

---

## 3. 实证过的隐式契约（探针，全部通过）

探针写在隔离副本 `/tmp/phase_audit`（跑完已删除），以 `-run TestProbe` 单独执行，并另跑一次去掉探针的包测试确认为 `ok`。

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-1** 真实 SDK 端到端 + 密钥不泄漏 | 真实 `futures.NewClient` + 假 Transport：签名 GET、交易 POST、`depth?limit=500` | ✅ 三个请求全部被观测；`/fapi/v2/account` weight=5/read/mainnet/`start_trade`；`/fapi/v1/leverage` trade/**testnet**；`/fapi/v1/depth` weight=**10**；**快照 JSON 无 signature/timestamp/recvWindow/key/secret**；所有 Path 无 query；mainnet/testnet 限额状态各一条 |
| **PR-2** 429/418 可见性 | 429（`Retry-After: 30`）+ 418 | ✅ 窗口计数、endpoint 计数、`recent_rate_limits`（含 Retry-After 与归属）、限额状态时间戳全部正确 |
| **PR-3** 观测开销 | 包装层 500 次真实请求；稳态 Record；Snapshot | ✅ **4.539µs/请求**（包装层）；低量级 Record 亚微秒；稳态成本见 F1；Snapshot 见 F2 |
| **PR-4** 权威 header 与动态分母 | 有/无 usage header 两次响应 + `SetWeightLimit` 覆盖 | ✅ 头值解析并保留（缺头不归零）；`exchange_info` 覆盖 reference 且百分比重算 |
| **PR-5** 稳态裁剪成本（精确微基准，按时序插入） | 5 分钟窗口填 20,000 事件后继续按生产顺序追加 500 条 | ⚠️ **平均 596µs/请求，最大 2.1ms**（保留 20,001 条）→ 见 F1 |
| **PR-6** 快照基数成本 | 20,000 事件 × 120 个 endpoint 键 | ✅ **8.24ms/次**（含每键延迟样本分配与 p95 排序）→ 见 F2 |

> 说明：PR-3 首次测量的稳态值（103ns）因探针自身**违反"按时间顺序追加"前提**（先灌入未来的 500 条、再灌入过去的事件）而失真；PR-5 用符合生产顺序的写法复测，才得到 596µs 的真实成本。这也从侧面说明：**该 collector 的正确性与成本都依赖"事件按时序追加"这一隐含约定**（生产中成立）。

---

## 4. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | **P2（性能）** | `pruneLocked` 在**每次 Record** 若发现存在过期事件，就 `append([]RequestEvent(nil), c.events[first:]...)` **整片复制**保留事件 → 单请求成本 **O(窗口内事件数)**，且复制在**全局互斥锁内**执行（并发请求会互相排队）。实测：2 万事件（≈67 请求/秒）时 **平均 596µs、峰值 2.1ms**；5 万上限时预计 ~1.5ms/请求 | 当前项目量级（2400 权重/分钟、单请求权重 1–40 ⇒ 窗口内约 1–3k 事件）下约 90µs/请求，尚未"明显增加延迟"；但 V4-5 若提高调用量或出现突发重试，观测层会成为**新的瓶颈**。建议：改为**摊销裁剪**（每 N 次或 `len` 超阈值才裁剪）、或**环形缓冲 + 头部索引**、或只保留聚合桶 + 有界延迟样本（p95 用蓄水池采样），把单请求成本降为 O(1) |
| F2 | P3 | `Snapshot` 无缓存：2 万事件 / 120 个 endpoint 键 → **8.24ms**（含每键 `latencies` 分配与排序），并持锁复制整个事件切片（O(n)） | 看板为手动/低频刷新，可接受；若未来做自动轮询或事件量增大，建议加 1s TTL 缓存或增量维护聚合 |
| F3 | P3 | `EstimateWeight` 未列入表的路径一律按 **1** 计，可能**低估**（如批量/新端点） | 文档 §3 已声明"已知时"、代码注释已声明 best-effort、headers 才是权威 ✓ 一致；建议在看板/报告标注"估计权重为下界，仅用于排序" |
| F4 | P3 | `endpointStats` 聚合键含 `source`，同一 path 会按 source 拆成多行，Top20 视觉上"分散" | 与 §6 设计一致（有意区分调用来源）；建议看板提供"合并 source"的第二视图，便于判断"同一 endpoint 是否被多个来源重复调用"（正是 V4-5 的关键问题） |
| F5 | P3（交付） | §8 报告的 Top20 / 1m 峰值仍为占位符 → 按文档 §7 完成规则，V4-4 尚未"完全完成" | 需要用户重启新二进制并采集 ≥5 分钟正常流量后回填；建议把采样结果同时作为 V4-5 的优先级输入 |
| F6 | P3（上限语义） | `maxEvents=50_000` 为硬上限：若 5 分钟内写入超过 5 万条，窗口统计（`request_count`/`estimated_weight`）会被截断且**无任何提示** | 建议在 Snapshot 暴露 `truncated`/`dropped_events` 计数或输出一次警告日志，避免"看起来正常但统计偏低" |
| F7 | P3（跨阶段） | V4-2 遗留两项（发布=管理员级提示词变更的文档说明、`DraftStore.List` 对损坏 `draft.json` 静默跳过）仍未处理；V4-3 复评的 R1（模拟盘路径跟随 `FutureStrategyCoin`）也仍未在文档点明 | 与本阶段无关，仅作跟踪 |

---

## 5. 自动化验证结果（用户工作区原地执行）

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./...` | ✅ 无输出 |
| `go test -count=1 ./...` | ✅ **62 个包 ok，0 FAIL**（比 V4-3 多 1 个：`service/binanceapiusage`） |
| `go test -count=1 -race ./service/binanceapiusage ./feature/api/binance ./spot/api/binance ./service/... ./controllers` | ✅ 全部 ok（binanceapiusage 1.328s、backtest 5.589s、historicalmarket 2.946s 等） |
| `gofmt -l`（V4-4 改动 Go 文件） | ✅ 无输出 |
| `git diff HEAD --check -- '*.go'` | ✅ 无异常 |
| 隔离副本 `go build ./...` + 探针 PR-1～PR-6 | ✅ 全部通过（F1/F2 为量化观测，非测试失败） |
| 副本去掉探针后重跑 `./service/binanceapiusage` | ✅ ok |
| 前端 | — 本阶段无前端改动（路由/看板字段由既有前端在后续构建中消费） |

---

## 6. 审计边界与未验证项

- **未验证**：真实运行采样（Top20/1m 峰值/429 实际分布）——这需要重启新二进制并运行正常业务 ≥5 分钟，属用户侧动作；本轮所有结论均来自代码 + 探针，未使用任何真实账号流量。
- **未验证**：真实代理（`binanceproxy`）路径下的包装层行为（探针使用假 Transport 绕过真实网络；包装位于代理 transport 外层，逻辑上不影响代理轮询，但未实测）。
- **未验证**：MySQL 无涉及；前端未涉及。
- **未验证**：WS 长时间连接下"listenKey 续期"等 REST 调用是否被计入（结构上会，权重 1；未实测）。
- 未对生产环境发起任何真实下单或修改操作。

---

## 7. 附：V4-4 交付物清单

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `service/binanceapiusage/transport.go` | 100 行（新） | `WrapClient`/RoundTripper；只持久化安全字段；`transport_error` 兜底；`WithSource`/`SourceFromContext` |
| `service/binanceapiusage/collector.go` | 452 行（新） | 事件缓冲（5m、上限 5 万）、限额状态、限流环形日志、10s/1m/5m 窗口、Top20（count/weight）、source 聚合、p95 |
| `service/binanceapiusage/weights.go` | 156 行（新） | 分产品估算权重（klines/depth/openOrders/24hr…）+ read/trade 分类 |
| `service/binanceapiusage/transport_test.go` | 186 行（新） | 签名不泄漏、错误不存原文、rolling window、Top 排序、动态限额、分类、估算权重 |
| `feature/api/binance/{index,delivery}.go`、`spot/api/binance/index.go` | +196/-? | 三端客户端包装（含 testnet）、`*Context` 变体、`exchangeInfo` → 动态限额分母 |
| `feature/{feature,ownership}.go`、`service/futuresownership/{execution,reconcile}.go`、`service/agenttrade/default.go`、`service/market/service.go`、`service/historicalmarket/binance_source.go`、`controllers/{account,listenCoin}.go` | +150/-? | 仅 context 透传与 source 标签（`start_trade`/`ownership_reconcile`/`agent_trade`/`historical_market`/`funding_rate`/`market_intelligence`/`manual_api`/owner 映射） |
| `controllers/system_dashboard.go`、`routers/router.go`、`service/systemhealth/service.go` | +20 | 只读快照 API、路由、健康检查纳入观测（`system_health`） |
| 文档 | — | `04-phase-v4-4-*.md` §11 实现状态；新增 `v4-4-implementation-report.md`（14 节）与 `v4-4-binance-api-usage-report.md`（含采样规则与待填数据） |
