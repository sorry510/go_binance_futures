# CODE_REVIEW_UF-6 · U 本位合约订单历史查询性能

> 日期：2026-11-19，后端 `master`，基线 HEAD `f31a7e6 feat: UF-5`。计划：[UF 六项精简优化](开发方案-UF-U本位合约交易六项精简优化.md)。
>
> **性质：只读查询改造。** Legacy GET 默认完全兼容，只有明确携带 `include_summary=0` 的请求跳过全范围汇总。未修改策略开平仓、资金风控、Managed/Fill 持久化、数据库结构或任何生产配置。

## 一、根因与改动

旧 `service/orderhistory/history.go::Store.List` 每请求都执行：
1. 筛选范围 `COUNT`，再取当前分页的 Managed 周期；
2. 批量 `project` 当前页关联的 Managed Orders/Fills、`symbols.close` 行情快照，计算当前页；
3. 除非第一页完整包含全部命中周期，否则再对筛选范围内 **每 100 个周期**重新读周期、子订单、Fill 和行情快照，计算跨全部筛选周期的 `filtered_pnl` 与 `pnl_unpriced_count`。

因此，翻第 2、3 页会重复整个第 3 步。瓶颈是**无谓的全范围重复计算**，不是原先缺少「全筛选结果总盈亏」功能。

UF-6 采用以下最小拆分：

| 文件/接口 | 行为 |
| --- | --- |
| `service/orderhistory/history.go` | `Store.List` 继续返回旧版完整 `Page`，保持默认契约；新增 `Store.ListPageOnly` 只计算当前分页、仍保留页内 `page_verified_net`；`Store.Summary` 复用原批量 `project` 和 `sumDisplayedPnL` 算法返回全筛选盈亏，并附带 `total`、`calculated_at`、`price_updated_at` |
| `controllers/orders.go`、`routers/router.go` | 原 `GET /orders` 默认保持旧汇总。只有 `include_summary=0/false` 时启用当前页快路径；新建 JWT 鉴权只读 `GET /orders/summary`；所有用户过滤参数严格验证，非法参数 HTTP 400，数据库故障返回错误而非伪造零 |
| `src/api/order.ts` | 类型定义 `ManagedOrderSummary`、分页可选参数 `include_summary`、独立只读汇总 API |
| `src/views/order/index.vue` | 列表每次只查 `include_summary=false`；首次打开、筛选改变、点击「查询/刷新」、批量处理成功后计算汇总；普通分页 1→2→3 / 每页数量变化不重扫；UI 缓存只在当前页面组件内使用，以完整过滤条件为键，进程切换/页面销毁不保留；有时戳、失败提示及手动刷新 |
| `locales/zh-CN.yaml`、`en.yaml` | 增加「汇总计算中/不可用」「计算时间」「最新行情价时间」文字 |
| `service/orderhistory/history_uf6_test.go`、`controllers/orders_uf6_test.go` | 旧新版同筛选结果对照、权限和非法输入、取消、Fill 更新、资产和历史人工覆盖隔离 |
| `service/orderhistory/history_uf6_benchmark_test.go` | 可重复运行的独立 SQLite 100/1000/10000 周期合成性能测量；只在显式设置 `UF6_BENCH_MODE` 后执行 |

**不变的计算口径：** USDT/USDC 不转换直接合并；已平仓价格差收益 + 未平仓价差估值原算法不变；缺失行情价对应 `pnl_unpriced_count` 而不是计入零；历史人工覆盖仅影响展示，不把它认证为 REST Fill；实际核实结算效果仍属于 UF-5 的独立统计，不将本页面行情估值称作「实盘已实现净收益」。

## 二、合成 SQLite 性能证据

使用本地隔离 SQLite 和 20 条/页、所有周期归属 Main、每周期同一种合约（1 单位）的 synthetic 数据。无真实交易所调用/生产数据。这一压测的各周期无关联 Order/Fill，目的是**隔离测量重复全范围投影的规模效应**；关联复杂、不同资产与真实 Fill 的语义另由下面严格对照测试覆盖。

**改动前 Legacy 路径基线（2026-11-19 本机，单次）：**

| 周期 N | 第 1 页 | 第 2 页 | 全范围盈亏 |
| ---: | ---: | ---: | ---: |
| 100 | 1.68 ms | 1.51 ms | 1,000 USDT |
| 1,000 | 17.97 ms | 19.15 ms | 10,000 USDT |
| 10,000 | 778.84 ms | 781.11 ms | 100,000 USDT |

**拆分后同一数据量的独立测量（本机另一轮，单次）：**

| 周期 N | 快速第 1 页 | 独立汇总（首次） | 快速第 2 页 |
| ---: | ---: | ---: | ---: |
| 100 | 0.36 ms | 1.55 ms | 0.47 ms |
| 1,000 | 1.03 ms | 18.37 ms | 0.69 ms |
| 10,000 | 10.72 ms | 1,006.76 ms | 5.76 ms |

改造后初次加载**仍然**需要完整的全范围汇总，其时间近似原单请求，不能宣称首屏加载同比例提速；优化收益主要发生于第 2、3 页和更多分页浏览，计算一次、反复分页复用。

根据源码读取路径静态推导的 SQL 往返数（**不是 DB trace 实测 QPS**），旧翻页是 `5 + 4×ceil(N/100)`，分别约 9/45/405 次；分页专用路径为约 **5 次**（COUNT、当前页、该页 symbols、orders、fills），实际查询数会因空页或条件而减少。**不增加冗余索引/汇总表/异步队列。**

运行方式：
- 基线：`UF6_BENCH_MODE=baseline go test -count=1 -p=2 -run '^TestUF6SyntheticBenchmark$' -v -timeout=300s ./service/orderhistory`
- 新版：`UF6_BENCH_MODE=after go test -count=1 -p=2 -run '^TestUF6SyntheticBenchmark$' -v -timeout=300s ./service/orderhistory`

## 三、正确性测试与交易安全边界

| 永久测试 | 验证 |
| --- | --- |
| `TestUF6LegacyListAndSeparatedSummaryExactAllFilters` | Main/Lead、Owner、币种、方向、真实状态、结算、日期、无匹配、页码超出、USDT/USDC、无行情估值：Legacy `filtered_pnl` 与新汇总严格相同；快页列表和页内核实净收益与旧版逐字段完全一致 |
| `TestUF6ManualOverrideAndIncompleteFillNotCertifiedBySummary` | 人工修改的历史状态/完整性不伪造 REST 验证；Fill 价格修改后再请求汇总真实反映新值，不因后台缓存返回陈旧结果 |
| `TestUF6InvalidFiltersAndCanceledRequestsNeverReturnValidEmptySummary` | 无效账户/owner/方向/日期/注入拒绝；取消 Context 不返回误导性的 0 盈亏或空列表 |
| `TestUF6CancellationMidSummaryNeverReturnsPartialFinancialTotals` | 超过 200 个周期时中途取消，禁止返回只汇总部分批次的收益 |
| `TestUF6EmptyAndOutOfRangePagesKeepAllFilterTotals` | 空集、超范围页面仍保持合法 total，完整汇总与页面无关 |
| `TestUF6NoQuotePriceTimestampIsZeroAndGenerationIsLocal` | 没有可信本地行情时不伪造已更新行情时间，汇总时间明确是本地计算时间 |
| `TestUF6OrdersSummaryAndPagingAuthFilterValidation` | 两接口 JWT；`include_summary` 仅识别明确的 0/false 与 1/true；SQL 前校验 |
| `TestUF6SyntheticBenchmark` | 100/1000/10000 周期；新旧汇总与页面数据逐项一致；比较第 2 页的重复扫描成本 |

前端请求次序：首次页面载入为当前页和汇总各一个请求；翻页仅请求当前页；条件变动会使过滤键变化并重算；「刷新」、成功的批量人工操作/删除均强制刷新。加载失败展示错误，不显示伪造零收益；`calculated_at` 是本地汇总计算时间，`price_updated_at` 是本地行情快照更新时间，**不是 Binance mark price 的实时核实时间**。

## 四、工程 Gate

| Gate | 结果 |
| --- | --- |
| `go test -count=5 -p=2 -run '^TestUF6' ./service/orderhistory ./controllers` | **PASS** |
| 五个 Go 核心包 `go test -count=5 -p=2` | **PASS** |
| 88 个非历史研究 Go 包 | **PASS** |
| 88 包 `go vet` | **PASS** |
| 五个核心包 `go test -race -count=2 -p=2` | **PASS** |
| `go build -o /tmp/go_binance_futures_uf6 .` | **PASS** |
| 前端 `pnpm typecheck` | **PASS** |
| 前端 `pnpm build` | **PASS** |
| `dist` 与 `static` 一致 | **PASS** |
| `gofmt`、Prettier、`git diff --check` | **PASS** |

> 2026-11-19 整改追踪：原 [独立复审](CODE_REVIEW_UF-6-rereview.md) 发现 F1/P1，UF-3 异常查询路由被 UF-6 行注释吞掉。整改已恢复实际注册并新增真实 HTTP 路由表测试，重跑全包/关键包/Race/前端 Gate 全部 PASS，详见 [F1 整改复查](CODE_REVIEW_UF-6-rereview-2.md)。原始失败报告未覆盖；UF-ALL 与生产验收仍需另行执行。

## 五、未执行验证

- 未对真实 MySQL 8 `EXPLAIN` / QPS / 10k 有 Fill 的全量数据实测。真实索引/连接池/缓存、网络延迟和线上大表的性能表现可能不同，**不能将 SQLite 加速比直接迁移成生产指标**。
- 未做 ARM Linux 部署、Binance Testnet/Mainnet 真实交易或服务重启；无数据库迁移、配置修改和真实账户写操作。
- 本方案将全量收益的重算时机改为首次打开、筛选变化、主动刷新或本页批量操作成功；**翻页时不会自动重新估算行情价格**，界面显示汇总计算时间与源价格时间供判断，必要时点击刷新。
- 后端仍保留旧兼容接口的 O(N) 全范围汇总成本；凡旧客户端不传 `include_summary=0`，仍会得到全量收益及相应执行成本。

**阶段结论：UF-6 源码、离线回归、Race/Vet、前后端构建与静态资源同步均 PASS。UF-1～UF-6 已具备进行 UF-ALL 独立整体复审的源码基础；真实 MySQL 与上线验收不包含在本阶段。**
