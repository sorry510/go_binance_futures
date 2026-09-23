# V4-3 Local Selector V2 Implementation Report

> 日期：2026-09-23  
> 状态：完成

## 1. 目标

V4-3 把自动交易候选选择从随机抽样升级为确定性、本地数据、多因子评分，并继续把最终开仓方向交给现有 Line Strategy。

最终职责保持：

```text
Smart Local Selector V2
  ↓ 只决定“看哪些币”
Line Strategy
  ↓ 决定 LONG / SHORT / no trade
现有 Ownership / Risk / Executor
```

V4-3 不使用 LLM，不新增每 Symbol Binance REST，不改变真实下单安全边界。

## 2. 新 Selector

新增配置值：

```text
smart_local_v2
```

注册位置：

```text
feature.GetCoinStrategy
  ↓
feature.selectConfiguredCoins
  ↓
coin.SmartLocalV2 / test-mode shared entry
  ↓
scanner.SmartLocalV2FromSymbolsWithModeCooldown
```

旧 `coin1～6` 完整保留。

没有自动切换系统当前 selector；用户必须在配置页显式选择 `smart_local_v2`。
## 3. 数据源

Selector 只读取：

- `symbols` 本地表。
- Futures WebSocket 已写入 `symbols` 的 24h ticker 数据。
- 真实交易：本地 `order` 表，用于最近真实平仓 cooldown。
- 测试交易：`test_strategy_results`，用于最近模拟平仓 cooldown。

明确不调用：

```text
GET /fapi/v1/ticker/...
GET /fapi/v1/klines
GET /fapi/v1/depth
GET /fapi/v1/allOrders
其它 Binance REST
```

`SmartLocalV2Result.meta.rest_api_used` 固定为 `false`。

静态检查也确认 `scanner/local_selector_v2.go` 与 `feature/strategy/coin/smart_local_v2.go` 不 import Binance API package。

## 4. Stage A：硬过滤

默认参数：

```text
Candidate Pool     = Top 60
Round-Robin Batch = 5
Cooldown           = 5 分钟
Max Local Data Age = 30 秒
Min QuoteVolume    = 5,000,000 USDT / 24h
```

进入评分前必须满足：

- `symbols.Enable == 1`。
- `Type == USDT` 且 Symbol 以 `USDT` 结尾。
- `UpdateTime > 0`。
- 本地行情更新时间不超过 30 秒。
- 最近 5 分钟本地 close order 中没有该 Symbol。

之后继续复用已有 Prefilter 硬过滤：

- 当前价格/Open/High/Low 必须有效。
- QuoteVolume 达标。
- 24h 涨跌变化绝对值 > 60% 直接过滤。
- 当绝对变化 > 40% 时做方向对称的反转结构过滤：上涨看长上影/冲高回落，下跌看长下影/低位反弹。
- 价格不能过度远离 24h 中枢。

WS 中断后数据超过 30 秒会 fail-closed，不会拿陈旧行情继续自动选币。
## 5. Stage B：确定性评分

评分继续复用 `scanner.PrefilterTop30FromSymbols`，没有写第二套 scorer。

为了不改变既有 `market_scan / Opportunity / scan_symbols` 行为，V4-3 新增的 TradeCount 分数与 Symbol 末级 tie-break 都通过 Prefilter opt-in 开关启用；Generic Scanner 默认关闭这些 V4-3 专属增强。

`smart_local_v2` 当前因子包括：

- 24h QuoteVolume 流动性。
- 24h TradeCount 活跃度。
- 24h PercentChange 变化幅度质量（上涨/下跌对称）。
- 24h High / Low / Open / Close。
- 距离 24h 中枢。
- 上涨方向：Upper Wick Ratio + Retrace From High。
- 下跌方向：Lower Wick Ratio + Rebound From Low。
- 本地 LastClose → Close 短周期 momentum，按绝对方向性变化加分，向上/向下都可。
- 极端变化/反转风险。

V4-3 增补了 TradeCount 活跃度：

- >= 200,000 笔：较高正向加分。
- >= 50,000 笔：中等正向加分。
- >0 且 <10,000 笔：风险扣分。

24h Change 现在不再把“上涨”视为天然更优，而是看绝对变化幅度：`abs(change) <= 5%` 为低变化档，`5% < abs(change) <= 35%` 为活跃但不过度的高分档，`35%～60%` 降低分数并提示结构风险，`>60%`（无论涨跌）直接过滤。局部 momentum 同样方向对称，向下变化可获得和向上变化同类的方向性分数。

这仍然只是 Candidate Quality，不决定 LONG/SHORT；最终方向完全由 Line Strategy 决定。

## 6. 稳定排序

不再使用 `math/rand`。

候选稳定排序：

```text
Score DESC
  ↓ 同分
QuoteVolume DESC
  ↓ 再同分
Symbol ASC
```

因此相同本地数据重复执行，Candidate 顺序完全一致。

泛用 Scanner 以前默认排除 BTC/ETH 作为 benchmark；V4-3 为 Prefilter 增加兼容选项：

```text
IncludeBenchmarks
```

默认仍为 `false`，所以现有 Opportunity/Scanner 的 benchmark 语义不变；只有 `smart_local_v2` 设置为 `true`，因此 BTC/ETH 如果在交易配置中 Enable，也可以参与自动交易候选。

同样，`UseTradeCountScore` 和 `StableSymbolTieBreak` 默认也为 `false`，只有 `smart_local_v2` 设为 `true`。因此 Generic Scanner 不会因为 V4-3 引入 TradeCount 加分或新的 Symbol tie-break 而改变既有候选排序。
## 7. Runtime 轮询与性能

`StartTrade()` 每 2 秒可能调用 Coin Selector。

运行时不是把 Top60 一次性全部交给 Line Strategy，而是：

```text
Top60 Candidate Pool
  ↓
Round-Robin
  ↓
每轮 5 个
  ↓
Line Strategy
```

稳定 Top60 下需要 12 轮覆盖全部 60 个；按约 2 秒一轮计算，大约 24 秒完整覆盖一次。Round-Robin 状态只存在进程内，不写数据库，重启后重新开始。

实现使用“最久未轮到优先（least-recently-served）”，并记录首次进入候选池的顺序。这样 Top60 排名动态变化时，新进榜 Symbol 不会持续插队导致原有未轮到候选饿死；只要某个 Symbol 持续留在 Top60，就会持续获得机会。

为避免每轮都给 800+ disabled symbols 构造 debug exclusion：

- Runtime selector 默认 `IncludeExcluded=false`。
- 只计算候选所需数据。
- Preview API 才使用 `IncludeExcluded=true`。

因此候选解释不会给真实交易循环增加大量无意义分配和排序。

Cooldown 只查询本地数据，不请求 Binance。

- 真实交易模式读取本地 `order` 表中本系统写入、`side='close'`、且 `UpdateTime` 在最近窗口内的订单。
- 测试交易模式读取 `test_strategy_results` 中最近已平仓的模拟交易。
- 两种模式的 cooldown 数据源独立，互不污染；Round-Robin 状态也分别维护。
- Cooldown 按 Symbol 生效，不区分 LONG/SHORT，因此平多后 5 分钟内该 Symbol 也不会立即被候选去做空。

手工在 Binance App 平仓但本地没有对应 real close row 的情况仍不会进入真实交易 cooldown，这是现有边界。

## 8. Preview API

新增只读接口：

```text
GET /futures/selectors/smart-local-v2?limit=60
```

返回：

```text
selector
generated_at
source
candidates[]   # Top60 候选池
  rank
  symbol
  score
  grade
  quote_volume_24h
  trade_count_24h
  percent_change_24h
  local_momentum_pct
  reasons[]
  risks[]
  missing[]
next_batch[]   # 只读预览下一轮 5 个
rotation
  pool_size
  batch_size
  sequence
  batch_symbols[]
  batch_ranks[]
excluded[]
  symbol
  reason
meta
  pool_limit
  pool_size
  batch_size
  cooldown_minute
  max_data_age_ms
  min_quote_volume
  rest_api_used=false
```

Preview、真实交易与测试交易使用同一个 Candidate Service，不维护第二套选币逻辑。Preview 调用对应 mode 的 Round-Robin 只读 `Peek`，不会推进真实或测试 sequence，也不会裁剪运行时轮询状态。`SmartLocalV2Result.NextBatch/Rotation` 仅由 Preview/debug 调用方通过 `Peek` 填充，普通 Candidate Service 返回时保持零值。

阈值当前固定在代码中：Candidate Pool=60、Batch Size=5、Cooldown=5m、MaxDataAge=30s、MinQuoteVolume=5M。Generic Scanner 仍保持默认 Top30，只有 `smart_local_v2` 通过 Prefilter `MaxLimit=60` 放宽候选池。Preview 默认请求 60，并在 meta 回显 `requested_limit` / `effective_limit`；`limit>60` 会 clamp 到 60。Preview 还接受 `mode=trade|test`，两者复用同一个前端组件和同一个 Candidate Service。
## 9. Web UI

配置中心与“测试结果”页面共用同一个 `SmartLocalV2PreviewDialog` 组件。配置中心使用 `mode=trade`，测试结果页使用 `mode=test`。

配置中心的选币策略增加：

```text
智能本地选币 V2
```

选择后显示：

```text
[预览候选]
```

预览弹窗展示：

- Rank。
- Symbol。
- Score。
- 24h QuoteVolume。
- 24h TradeCount。
- 24h Change。
- Local Momentum。
- Reasons。
- Risks。
- Excluded Symbol / Excluded Reason。
- 数据源、cooldown、freshness、REST used。
- Top60 Candidate Pool 实际大小。
- 每轮 Batch Size=5。
- 已完成轮询批次数 sequence。
- 下一批的 Rank / Symbol，并在 Top60 表格中高亮标记。

这是只读解释入口，不是新的策略实验平台。

## 10. 实际本地数据验证

在开发数据库做了只读验证。

当前数据库：

```text
symbols.Enable=1 数量 = 0
```

因此当前 Smart Local V2 返回 0 candidates 是正确结果；旧 `coin1～6` 同样要求 `Enable=1`。

同时读取当前 Futures WS 写入数据，USDT symbols 的行情年龄约：

```text
0.7 ～ 0.8 秒
```

所以默认 `MaxDataAge=30s` 对正常 WS 足够宽裕，同时又能在 WS 断流时快速停止候选选择。
## 11. 自动测试

新增永久测试覆盖：

- enabled 硬过滤。
- stale 硬过滤。
- low liquidity 排除。
- local cooldown 排除。
- 相同输入重复执行 Candidates 完全一致。
- 同 Score/QuoteVolume 使用 Symbol ASC 稳定 tie-break。
- `rest_api_used=false`。
- Runtime 不生成完整 Excluded debug 明细。
- `smart_local_v2` 允许已启用的 BTC/ETH。
- Generic Prefilter 默认仍排除 BTC/ETH，确保旧 Scanner 兼容。
- Generic Prefilter 默认不启用 V4-3 TradeCount 加分，也不启用 Symbol 末级 tie-break；旧 market_scan/Opportunity/scan_symbols 排序行为保持。
- 真实本地 `order` 表 SQL：窗口内 close 触发 cooldown，旧 close/open 不触发。
- DB 端到端 SmartLocalV2：cooldown、stale、disabled、Preview 路径。
- 全量 stale 时 DB 路径 0 candidates，fail-closed。
- `limit=100` clamp 到 60，并回显 requested/effective limit；Generic Scanner 仍固定默认 Top30。
- 稳定 Top60 下 Round-Robin 每轮 5 个，12 轮完整覆盖 1～60，第 13 轮回到 1～5。
- Top60 动态变化时，新进榜候选不会插队饿死原有未轮到候选。
- Preview `Peek` 不推进真实 Round-Robin sequence。
- 最低 QuoteVolume 5M：7M 可进入、4M 被过滤。
- 上涨/下跌 24h Change 对称评分，向下 momentum 也可加分。
- ±61% 极端变化都被对称过滤。
- trade/test cooldown 数据源独立：real close 只影响 trade，模拟 close 只影响 test。
- trade/test Round-Robin scope 独立，互不推进 sequence。
- Generic/Smart 精确差分永久测试：V4-3 opt-in 因子只影响 Smart，Generic 分数与 reasons/risks 不漂移。
- `GetCoinStrategy("smart_local_v2")` 正确注册。

最终 Gate：

```text
go test -count=1 ./...                              PASS
go test -count=1 -race ./scanner ./feature ./controllers PASS
go vet ./...                                        PASS
go build ./...                                      PASS
git diff --check                                    PASS

pnpm typecheck                                      PASS
pnpm build                                          PASS
```

race 只有既有 macOS linker `LC_DYSYMTAB` warning；前端只有 Browserslist/caniuse-lite 过旧提示。

V4-3 没有数据库 Schema 变化，因此仍为 Schema v18，本阶段没有执行 `sync db`。
## 12. 人工测试

### A. 准备 Symbol

在现有合约配置页面启用一批流动性较好的 USDT 合约；要完整验证 Top60 轮询，建议启用至少 60 个满足硬过滤条件的合约。候选不足 60 时 Pool 会自然缩小，但每轮仍最多取 5 个。

不要直接改数据库；使用现有 Web 配置方式启用。

确认 Futures WS 正常运行。

### B. Preview

1. 打开系统配置。
2. 把“选币策略”切换为“智能本地选币 V2”。
3. 点击“预览候选”。
4. 顶部应显示：

```text
source = local_db.symbols
REST = no
pool = x/60
batch = 5
cooldown = 5m
freshness = 30s
next batch = #...
```

5. Candidate Pool 最多显示 60 个，并按 score 从高到低。
6. “下一批”应最多标记 5 个 Rank/Symbol。
7. 每个 Candidate 应有 Reasons/Risks。
8. disabled / low volume / cooldown / stale symbol 应出现在 Excluded 中并有具体原因。

### C. 测试交易预览与真实逻辑一致

测试运行始终跟随当前 `FutureStrategyCoin`。当配置为 `smart_local_v2` 时，真实/测试共用同一评分与 Top60/Batch=5 算法，仅 cooldown 数据源与 Round-Robin scope 分离；若显式配置旧 `coin1～6`，测试也会继承对应旧 selector 的随机抽样/覆盖面，这是为了保持测试与真实交易一致，而不是继续做“全 Enable 币遍历”。

1. 打开“测试结果”页面，点击“测试选币预览”。
2. 弹窗应显示 `mode=测试交易`，使用与真实交易相同的 Top60 排名、Score、Reason/Risk 和 Batch=5 轮询算法。
3. 测试预览的 cooldown 来自最近已平仓 `test_strategy_results`；真实预览的 cooldown 来自本地 `order` close，两者互不污染。
4. 连续打开 Preview 只查看下一批，不应推进真实或测试 Round-Robin sequence。
5. 测试运行日志中的候选批次应与 `mode=test` Preview 的下一批逻辑一致，不再按 symbols ID 顺序扫遍全部 Enable 币。

### D. 确定性

在行情变化很小的几秒内连续点两次 Preview。

同一份本地 ticker 状态下排序应稳定，不再出现旧 coin selector 的随机换币。

### E. Round-Robin

在 Top60 相对稳定时观察运行日志：

```text
smart_local_v2 round-robin batch
```

每轮应只出现最多 5 个 Symbol。稳定 60 个候选时，前 12 轮应覆盖完整 60 个，不应一直只检查排名前 5；第 13 轮开始重新轮询最久未轮到的一组。打开 Preview 只能看到“下一批”，不应推进运行时 sequence。

### F. Cooldown

让某 Symbol 产生一条本地 close order。

5 分钟内重新 Preview，该 Symbol 应显示：

```text
最近交易冷却中
```

5 分钟后才允许重新进入候选。

### G. Freshness Fail Closed

只建议在人工维护窗口测试：暂停 Futures ticker WS，让 `symbols.UpdateTime` 超过 30 秒。

Preview 中原候选应变为：

```text
本地行情数据过旧
```

恢复 WS 后候选应自动恢复，无需 REST 补数据。

### H. StartTrade

1. 开启 `smart_local_v2`。
2. 保持原 Line Strategy 不变。
3. 查看日志/订单行为。
4. Coin Selector 只缩小候选范围。
5. 没有满足 Line Strategy LONG/SHORT 条件时必须仍然不下单。

这一步用于确认 V4-3 没有绕过原有开仓规则。

## 13. 本阶段未做

- 不把 `smart_local_v2` 自动改成系统默认。
- 不删除旧 `coin1～6`。
- 不使用 LLM / ML ranking。
- 不逐币请求 Kline / Depth / Open Interest。
- 不增加历史回测或参数优化。
- 不处理 Binance 全局 API Budget；那属于 V4-4 / V4-5。

