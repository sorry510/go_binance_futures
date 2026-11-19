# CODE_REVIEW_UF-4 · Main 合约交易「为什么没有开仓」

> 日期：2026-11-19。后端 `master`；依据：[UF 六项精简优化方案](开发方案-UF-U本位合约交易六项精简优化.md)、[UF-3 复审](CODE_REVIEW_UF-3-rereview.md)。
>
> **阶段边界：源码/离线/构建 Gate 与真实生产验收分开。** 此功能只采集现有 Main 自动策略交易循环中的观察结果；不产生买卖信号，不改变开平仓条件，不使用新的数据库表，不调用额外的 Binance API。

## 一、实现范围

| 文件 | 变更 |
| --- | --- |
| `feature/entry_diagnostics_uf4.go` | 内存有界的 Main 最近一轮决策快照（最多 200 个币种/方向条目及 200 个成功委托提交时间）；`RWMutex`、轮次 generation 防止旧回调覆盖新轮次；进程重启后只返回 `not_started` |
| `feature/feature.go` | 在原有 Main 信号、候选池、仓位/挂单、风险、深度、UF-2 配置 RPC、订单提交结果分支插入只读事件；沿用原顺序和原交易 API，不向日志写入每 tick 新事件、不为诊断多做 DAPI/策略计算 |
| `feature/account_trade_cycle.go` | Main 总开关 OFF 且不需要运行交易循环时也发布 `global_off` 状态；已有 Managed CLOSE 仍依照 UF-1 原路径工作 |
| `controllers/entry_diagnostics_uf4.go`、`routers/router.go` | 新增 **仅 GET** `/futures/entry-diagnostics`；JWT `Username` 验证，symbol/position_side 严格校验，仅返回本进程内存快照 |
| `src/api/trade.ts` | TypeScript 只读接口与返回类型；请求时使用既有 `skipProgress` 避免展开后的轮询闪动全局进度条 |
| `src/views/futures/EntryDiagnostics.vue`、`symbol.vue` | 在现有合约交易币种页增加**默认收起**的「为什么没有开仓」区，按币种和 LONG/SHORT 筛选；展开后每 15 秒读一次快照，收起/组件卸载停止轮询 |
| `locales/zh-CN.yaml`、`en.yaml` | 双语全局/逐币原因和时间说明，区分运行错误、正常策略未触发及 Binance 委托确认 |
| `feature/entry_diagnostics_uf4_test.go`、`controllers/entry_diagnostics_uf4_test.go` | Main 实际循环回调、JWT/参数、并发、200 条内存上限、主从账户隔离、拒绝时不发开仓、委托 ACK 时间跨轮次保留 |

## 二、原因真实性、范围和优先级

**只报告本轮真正执行过的门禁：**

- `global_off`：Main 总开关 OFF；不等于已有 Managed 持仓立即强平。若 Main 循环未启动，不会仅为诊断多请求 Binance。
- `symbol_off`：已加载 `symbols.enable != 1`；`no_candidate`：币种启用但未进入**实际** `SelectCoins` 返回的候选池。由于当前选币器不返回逐币失败明细，**不会凭猜测标记 `volume_low`、`market_data_stale` 或 `cooldown`**。
- `max_positions`、`max_losing_positions`、`direction_off`：现有风险与方向开关真实触发；同轮 LONG 订单消耗最后名额后，SHORT 侧也会准确记录 `max_positions`。
- `existing_position`、`pending_order`：既有内存账户快照检查，含原 Managed 持仓整个 symbol 禁止重复 OPEN 的分支。
- `strategy_not_matched`：策略方法实际返回 `CanLong/CanShort=false`；不是网络出错。
- `depth_unavailable`、`trade_config_failed`、`order_submit_failed`：分别来自已有深度读取、UF-2 的 `EnsureConfig` 和原 `SubmitOpen` 返回错误，**不向接口泄露异常文本、凭证、签名 URL 或 RPC 请求参数**。
- `invalid_quantity`：已有开仓计算遇到非正/非有限值时观察到的事实；该观察器不额外更改原交易路径。若后续订单请求返回错误，以 `order_submit_failed` 为最终返回原因。
- `order_submitted`：`SubmitOpen` 无错误，记录 `last_order_submitted_at`；**只是交易所接受委托，不代表有真实成交 Fill 或开仓成功**。
- `not_evaluated`：该方向本轮尚未经过下一项开仓检查；`symbols_unavailable`、`positions_unavailable`、`open_orders_unavailable`、`ownership_sync_failed`、`selector_unavailable`：分别标识原交易循环提前退出的真实原因。

**轮次语义：** 当前进程最新一次 Main 交易 tick 的记录，不做跨进程同步或持久化；最多保留 200 个 symbol/side 最近记录，同样最多保留 200 个成功委托提交时间。旧轮次写入通过 generation 检查丢弃；Main/Lead 不共享观察器。页面必须显示本轮时间与超过 30 秒的过期提示。

**注意**：对于未入选的币种，仅说明本轮未入选；不会为页面再跑选币服务以推断更细粒度原因。前端读取不写数据库、不查私有 Binance REST；执行过程中如有行情、配置错误，按对应错误类型展示。

## 三、验收测试

| 证据 | 内容 |
| --- | --- |
| `TestUF4MainEntryDiagnosticActualCycleGates` | 实际 Mock Main 循环：候选为空、选币错误、策略未命中、方向 OFF、亏损仓位和总仓位上限、已有持仓/挂单、深度异常及 UF-2 配置失败，验证不会因诊断多发 OPEN/CLOSE |
| `TestUF4MainEntryOrderSubmitFailureAndShortConfigFailure` | LONG/SHORT 提交失败与两方向杠杆/保证金校验拒绝均返回正确原因 |
| `TestUF4MainEntrySubmissionIsACKNotFillAndLastTimePreserved` | 真实执行路径中的 LONG/SHORT 委托 ACK 被标为 submitted，且上次提交时间在下轮策略未命中后仍保留 |
| `TestUF4DisabledSymbolFromActualLoadedSettings`、`TestUF4DoesNotInferVolumeFromNoCandidate` | 区分显式禁用与候选落选；不虚构成交量/行情过期原因 |
| `TestUF4EntryObserverNeverAddsBinanceQueries` | 多次快照只读查询不会增加交易账户 RPC 调用 |
| `TestUF4EntryObserverDoesNotOverwriteMainFromLead` | Lead 不干扰 Main 诊断 |
| `TestUF4MemoryBoundedConcurrentSnapshotAndGeneration` | 并发读写、200 条硬上限、轮次隔离、上次 ACK 时间 |
| `TestUF4ExplicitUnavailabilityIsNotStrategyNotMatched` | 原行情账户/币种配置读取失败显示明确不可用，不伪装策略无信号 |
| `TestUF4MainEntryDiagnosticsEndpointIsJWTOnlyReadOnly` | 未认证 401，非法币种/方向 400，有效 GET 200，不触达 DB/交易所 |

## 四、工程 Gate

| 项目 | 结果 |
| --- | --- |
| `go test -count=5 -p=2 -run '^TestUF4' ./feature ./controllers` | **PASS** |
| 五个核心包 `go test -count=5 -p=2` | **PASS** |
| 88 个非历史研究 Go 包 | **PASS** |
| 88 包 `go vet` | **PASS** |
| 五个核心包 `go test -race -count=2 -p=2` | **PASS** |
| `go build -o /tmp/go_binance_futures_uf4 .` | **PASS** |
| 前端 `pnpm typecheck` | **PASS** |
| 前端 `pnpm build` | **PASS** |
| `dist → backend/static` 精确比对 | **PASS** |
| `gofmt` / Prettier / `git diff --check` | **PASS** |

历史研究目录 `strategy_templates/research/` 含独立研究过程中的重名文件；除该目录外的正式 Go 工程包全部纳入全包验证。

## 五、风险和发布边界

1. **不会改变交易行为**：现有 Main/Lead 账户、Owner、订单唯一 ID、开平仓、实际 Fill/结算与风控接口继续原逻辑。观察器是附加的只读进程内缓存，不能作为交易或订单真相源。
2. 未修改 `conf/app.conf`、生产 DB schema、后台服务或运行中生产开关，未调用真实 Binance 签名下单接口；本轮 Testnet/ARM MySQL 8/主网均未现场验收。
3. 进程内存数据重启丢失，非完整历史事件；非候选币种不会获得未经实际扫描证实的详细淘汰原因。若用户需要历史趋势，应单独立项而非混入 UF-4。
4. 在 Main 多实例部署时，接口仅对应当前实例，不能代替统一分布式监控。当前个人单实例项目不增加 Redis/新表。
5. 因本轮增加必要的前端组件，完整 `pnpm build` 会把既有当前前端工作区一并打包进 `dist/static`，无需在 UF-4 中触碰非相关功能或把之前的 UF-3 改动另行重写。

## 六、阶段判定

**UF-4 全部源码/隔离测试与前后端构建 Gate 已 PASS。** 通过后按开发方案进入 UF-5「实盘策略效果统计」，不把本地进程诊断结果视为实时账户真实成交核验或生产发布授权。
