# Phase V3-4：Adaptive Resolution Backtest

> 状态：✅ 已完成（2026-09-11）
>
> 定位：V3-3 Historical Backtest 的精度增强层，不推翻 1m 主回放架构。
>
> 核心目标：以 1m 为主时间轴；当价格路径顺序无法确定，或 1m High/Low 证明 ROI Gate 可能在分钟内触发时，才按需升级到 1s，必要时继续下钻逐笔 trades。

## 1. 为什么做 V3-4

V3-3 已经建立固定 1m Replay、Historical Market Repository、Funding、MarketCondition 和确定性 Strategy 回放。

当前主要误差不是“1m 不知道价格范围”，而是：

```text
1m OHLC 知道 High / Low
但不知道 High 与 Low 谁先发生
```

因此不能直接把 High/Low 当作更精确的成交价格。V3-4 采用自适应精度：

```text
绝大多数分钟：继续 1m
发生价格路径歧义：下钻 1s
同一秒仍有冲突：下钻 trades
```

目标是提高执行顺序真实性，同时避免全历史 1s/逐笔回放带来的存储和性能成本。
## 2. 精度层级

### Level 1：1m 主回放

- 仍以 V3-3 的 1m execution timeline 为主。
- Strategy signal 默认仍在 1m close 产生。
- 无歧义时完全不读取 1s/trade 数据。
- V3-3 的指标周期、Funding、MarketCondition、手续费、滑点和数据哈希语义继续保留。

### Level 2：1s Intrabar Replay

只在当前 1m Bar 中存在多个可能价格事件，且 1m OHLC 无法判断先后时读取该分钟的 1s 数据。

典型候选事件：

- Hard Take Profit。
- Hard Stop Loss。
- 已存在的 Limit / Stop Order。
- 已知 trigger price 的其它确定性保护事件。
- Liquidation 仅在引擎已经有可信、事前确定的 liquidation price 时参与；V3-4 不虚构新的 Binance 保证金/维持保证金模型。

### Level 3：Trade Replay

如果同一个 1s Bar 内仍同时命中多个候选事件，则读取该秒逐笔 trades，按 `trade_time`、再按 `trade_id` 确定先后顺序。
## 3. 当前 StopLossPct / TakeProfitPct 的重要兼容边界

当前 V3-3 中：

```text
ROI 未越过 StopLossPct / TakeProfitPct
→ 不评估 close_long / close_short

ROI 越过门槛
→ 才评估 close strategy
→ close strategy=true 才产生平仓 signal
```

因此它们不是“价格一触及就立即成交”的硬止损/止盈单。

V3-4A 不改变这个语义，也不能用 1m High/Low 直接提前平仓。

V3-4A 可以用 High/Low 判断“该分钟是否可能曾越过 ROI Gate”，但这只能作为下钻候选条件，不能直接决定交易结果。

真正让现有 ROI Gate 获得秒级精度属于 V3-4B：在候选分钟内按秒重算 `NowPrice / ROI / Strategy Env`，再运行原来的 close rule。

这样既保持现有 Strategy Template 正确性，也不会把分钟结束时的 close rule 结果倒用到分钟中间造成未来数据泄漏。
## 4. Binance Public Data：Go 原生数据源

不引入 Python 运行时，直接在 Go 中复刻 Binance 官方 `binance-public-data` 的 URL 规则、ZIP/CSV 解析和 CHECKSUM 校验。

官方基地址：

```text
https://data.binance.vision/
```

USD-M Futures 使用 `um` 路径。

优先使用 daily archive 做按需解析，避免为了一个冲突分钟下载整月大文件：

```text
1s Kline:
data/futures/um/daily/klines/{SYMBOL}/1s/{SYMBOL}-1s-YYYY-MM-DD.zip

Trades:
data/futures/um/daily/trades/{SYMBOL}/{SYMBOL}-trades-YYYY-MM-DD.zip

Mark Price 1s（如该日期官方有文件）:
data/futures/um/daily/markPriceKlines/{SYMBOL}/1s/{SYMBOL}-1s-YYYY-MM-DD.zip
```

每个 ZIP 同目录读取 `.CHECKSUM`，默认 SHA256 校验通过后才允许进入回测证据链。
### 4.1 数据源优先级

```text
本地 sparse 1s / trades cache
        ↓ miss
本次 Run 的临时 archive cache
        ↓ miss
Binance Public Data daily 1s Kline
        ↓ 文件不存在
Binance Public Data daily trades → 聚合目标分钟 1s
        ↓ 仍无法获取
按 missing policy 明确失败或保守降级
```

同秒排序时直接读取 trades，不再用 1s OHLC 猜顺序。

### 4.2 Go Downloader 要求

- `net/http` 下载，复用项目现有 proxy/transport 能力。
- `archive/zip` + `encoding/csv` 流式解析，不执行外部脚本。
- 使用 `crypto/sha256` 校验 `.CHECKSUM`。
- 404 表示该 archive 不存在，进入下一数据源，不进行无意义重试。
- 网络错误 / 5xx 做有限指数退避；支持 context cancel。
- 同一个 `{kind,symbol,date}` 使用 singleflight 去重，多个 Backtest 不重复下载同一 ZIP。
- 单次 Run 可保留临时 daily ZIP，Run 结束后清理；跨 Run 只长期保存真正使用到的 sparse 数据。
## 5. V3-4A：Adaptive Intrabar Execution

### 5.1 数据模型

新增 sparse 高精度缓存，不把整个历史市场复制成秒级：

- `market_klines_1s`：只保存真正被 drill-down 使用过的 1s Bar。
- `market_trades`：只保存真正用于同秒排序的 trades slice。
- 如要支持真实 liquidation path，再增加独立 mark-price 1s 存储；不能把 contract trade price 冒充 mark price。

`market_trades` 至少包含：market、symbol、trade_id、trade_time、price、quantity、quote_quantity、is_buyer_maker、source、source_ref、archive_sha256。

索引重点：

```text
market_klines_1s: UNIQUE(market, symbol, open_time)
market_trades:    UNIQUE(market, symbol, trade_id)
market_trades:    INDEX(market, symbol, trade_time)
```

Schema 只通过 `./go_binance_futures sync db` 升级；不创建空 migration SQL。
### 5.2 ResolutionProvider

新增高精度数据抽象，Backtest Engine 不直接拼 archive URL：

```text
ResolutionProvider
├── MinuteBars(...)      // 现有 1m Repository
├── SecondBars(...)      // sparse 1s + archive fallback
├── Trades(...)          // sparse trades + archive fallback
└── MarkPriceSeconds(...) // liquidation 专用，可选
```

Provider 负责：本地命中、下载、CHECKSUM、解析、稀疏入库和 evidence hash。

### 5.3 PriceEvent / IntrabarResolver

把“顺序问题”从 Strategy Engine 中独立出来：

```text
PriceEvent
- type
- trigger_price
- direction: >= / <=
- known_at
- priority（仅用于完全同价/同时间确定性 tie-break）
```

`known_at` 必须早于事件被解析的时间；未来才产生的 trigger 不能反向参与当前分钟解析。
### 5.4 1m → 1s → trades 决策算法

```text
读取当前 1m Bar
        ↓
用 High/Low 判断候选 PriceEvent 是否可能触达
        ↓
0 个事件：继续 1m
1 个事件且不存在顺序冲突：1m 即可确定
2+ 个事件可能在同一分钟发生：读取该分钟 1s
        ↓
按 second 顺序扫描
        ↓
某秒只有一个事件首次命中：确定结果
某秒同时命中 2+ 事件：读取该秒 trades
        ↓
按 trade_time、trade_id 顺序扫描
        ↓
第一个 crossing trade 决定事件顺序
```

Limit Order 首版只做“价格触达即视为可成交”的简化模型，不模拟交易所排队位置和部分成交深度；此假设必须写入 Run metadata。

### 5.5 缺失高精度数据策略

Adaptive 模式默认 `strict`：只有真的出现歧义且需要下钻时，高精度数据缺失才失败；不会因为整个回测区间存在 2020 年之前的数据就预先拒绝所有 1m 回放。

后续可增加 `conservative`：数据缺失时按对策略不利的事件顺序处理。禁止 optimistic fallback 作为默认。
### 5.6 可复现性与审计

Adaptive Run 必须记录：

- `resolution_mode`：`standard_1m` / `adaptive`。
- `resolution_model`：例如 `adaptive_intrabar_v1`。
- 1s drill-down 分钟数。
- trade drill-down 秒数。
- archive cache hit / download 次数与字节数。
- unresolved / conservative fallback 次数。当前首版仅实现 `strict`：高精度数据无法取得时 Run 直接失败，不做 conservative fallback，因此成功 Run 的 `unresolved` 保留为 0；“rule=false”属于明确的不平仓结果，不计 unresolved。
- 实际使用的 archive URL、SHA256 或对应 evidence hash。

最终 `data_hash` 不能只包含 V3-3 的 1m/Funding/MarketCondition，还必须合并本次真正使用到的 1s/trade/mark-price evidence hash。

Audit Event 增加 `intrabar_resolution`，至少记录：候选事件、1m OHLC、使用精度、首次命中时间、最终事件、数据 source_ref/checksum。

Engine 语义变化后提升 EngineVersion。最终实现将 A/B 合并到同一 Adaptive 路径，统一使用 `backtest_engine_v8`；标准模式继续保持 V6，旧 V6 Run 不回写。
## 6. V3-4B：Adaptive Strategy Evaluation

V3-4B 才处理当前 ROI Gate / close strategy 在分钟内部可能成立又消失的问题。

### 6.1 候选分钟筛选

不做“所有历史全部 1s”。只在 1m 已证明存在高精度必要性时进入秒级 Strategy Eval，例如：

- 当前有仓位，1m High/Low 推导的 ROI 范围曾越过 StopLossPct / TakeProfitPct，但 Close ROI 未必越线。
- V3-4A 已识别到多个 PriceEvent 冲突。
- 已有订单/保护事件需要和 Strategy close 共同排序。

首版 V3-4 的生产引擎只接入 ROI Gate / close strategy 的高精度重放。`IntrabarResolver` / `PriceEvent` 作为后续硬 TP/SL、Limit 等确定性事件的基础能力保留；由于 V3-3 当前不存在这些订单对象，它们尚未进入生产执行链路。

不尝试通过静态解析任意 expr 猜所有可能的秒级开仓机会。

### 6.2 秒级 Env

在候选分钟按 1s 顺序构建 Env：

- `NowPrice`：当前 1s close；trade drill-down 时使用当前 trade price。
- `ROI`：按当前高精度价格重新计算。
- `MarketCondition`：只取 `time <= 当前时刻` 的最新历史值。
- Technology 指标严格遵循 as-of 语义，不能使用尚未闭合的未来 Kline。

开始实现前必须先审计 Live `InitParseEnv` 对“当前未闭合 Kline”到底采用 completed-only 还是实时更新值，V3-4B 应尽量复现真实执行，而不是自行发明另一套指标时间语义。
### 6.3 ROI Gate 精确化

当前 1m Close 模型：

```text
Bar Close
→ 计算 ROI
→ Gate 命中
→ evaluate close rule
→ Next 1m Open 成交
```

V3-4B Adaptive 模式在候选分钟改为：

```text
1m High/Low 表明 ROI Gate 可能曾被触发
        ↓
读取该分钟 1s
        ↓
按秒计算 ROI
        ↓
首次越过 Gate 的秒开始评估对应 close rule
        ↓
Rule=true 才生成 close signal
        ↓
成交时间/价格遵循新的高精度执行规则
```

如果同一秒内 Gate、TP/SL/其它价格事件发生顺序仍影响结果，再进入 trade replay。

V3-4B 不要求所有 long/short 开仓规则都变成秒级；“捕捉 1m Close 完全看不到的任意秒级开仓机会”属于更高频的独立能力，不纳入本 Phase。
### 6.4 同秒 trade 语义

同秒 trade replay 只解决价格路径顺序，不把整套 Strategy 对每一笔成交无限重算。

默认规则：

- trades 按 `trade_time ASC, trade_id ASC`。
- crossing 发生时更新 `NowPrice / ROI`。
- 只在会改变当前候选事件结果的 crossing 点评估必要规则。
- Market/Stop 类成交价以 first crossing price 为基础，再应用现有 SlippageBps。
- Limit 首版不模拟 queue position；仅保证不在价格从未触达时虚构成交。

若官方 archive 同一 timestamp 内 trade_id 顺序仍不足以表达撮合先后，则记录 deterministic tie-break，不宣称达到交易所撮合引擎级精度。

### 6.5 Engine Version

V3-4B 再次改变 Strategy 时间语义，建议使用 `backtest_engine_v8`，与 V7 的纯 Intrabar Execution 明确分离。
## 7. UI / API

回测页面增加精度模式：

```text
标准 1m
自适应精度（1m → 1s → trades）
```

首版保留标准 1m 作为对照模式；Adaptive 稳定后可考虑设为默认。

结果页增加：

- 1s drill-down 分钟数。
- trades drill-down 秒数。
- 高精度 cache hit / download。
- 高精度数据下载量。
- unresolved/fallback 次数。
- Trade 的 entry/exit resolution badge。
- Audit Event 可展开查看 Intrabar Resolution 证据。

现有“获取历史数据”仍只准备 1m + Technology intervals + Funding + MarketCondition，不允许因此全范围下载 1s/trades。

高精度 archive 必须在真正发生 drill-down 时 lazy fetch；可在 UI 显示 `resolving_intrabar_data` 阶段和当前日期。

实际公网复核（2026-09-11）：Binance Public Data 的 USD-M Futures `BTCUSDT` `1m` daily archive 可正常取得，但抽查多个日期的 `1s` daily archive 返回 404。因此当前 Provider 会按既定 fallback 使用该日 `trades` archive，并只把目标分钟聚合成 1s / sparse rows；这不是全区间 1s 入库。保留 1s archive 尝试是为了兼容 Binance 后续可能提供该数据。
## 8. 开发顺序

### V3-4A-0：冻结 V3-3 基线

1. 保留标准 1m 模式现有 Fixture。
2. 为同输入记录 V6 结果，作为 Adaptive 不触发时的回归基准。
3. 明确现有 ROI Gate、Next Open Fill、Funding、MarketCondition 时间语义。

### V3-4A-1：Go Binance Public Data Client

1. URL Builder：`um / daily / monthly / klines / trades / markPriceKlines`。
2. HTTP 下载、proxy、timeout、retry、context cancel。
3. `.CHECKSUM` SHA256 验证。
4. ZIP/CSV Parser。
5. 先完成 1s Kline + trades；mark price 单独 Gate。
6. 用 `httptest` 构造 ZIP/404/坏 checksum 测试，不让普通单测依赖公网。

### V3-4A-2：Sparse High-Resolution Repository

1. Schema / ORM：`market_klines_1s`、`market_trades`，必要时 mark-price 1s。
2. Repository local-first。
3. 只导入目标 minute/second slice，不把 daily trades 整包灌入数据库。
4. 升级数据库版本并执行 SQLite/MySQL sync Gate。
### V3-4A-3：ResolutionProvider

1. `SecondBars()`：本地 sparse → daily 1s archive → trades 聚合 fallback。
2. `Trades()`：本地 sparse → daily trades archive。
3. Run-scoped 临时 ZIP cache，避免同一天重复下载。
4. singleflight 去重并发请求。
5. 输出 source_ref、archive checksum、evidence hash。

### V3-4A-4：IntrabarResolver

1. 定义 `PriceEvent`。
2. 用 1m High/Low 做候选事件筛选。
3. 多事件分钟进入 1s。
4. 同秒多事件进入 trades。
5. 严格验证 `known_at`，禁止未来才产生的 trigger 参与过去时刻。
6. 添加 deterministic tie-break 和完整 Audit Event。

### V3-4A-5：接入 Backtest Engine

1. 增加 `resolution_mode`，保留 `standard_1m`。
2. Adaptive 模式只在真正有歧义时调用 Resolver。
3. 无歧义 Run 的交易结果必须与 V6 标准 1m 完全一致。
4. 合并 high-resolution evidence 到最终 DataHash。
5. EngineVersion 升至 V7。
### V3-4A-6：UI / Observability / Gate

1. 回测参数增加标准/自适应精度选择。
2. 结果显示 Resolution 统计和每笔 Trade 的精度来源。
3. 下载阶段显示 archive source/date/progress。
4. 支持 cancel 时终止 HTTP/parse，不留下半成品 canonical 数据。
5. 通过 V3-4A Gate 后才进入 B。

### V3-4B-0：Live 时间语义审计

1. 审计真实交易 `InitParseEnv` 的 Kline/Technology as-of 行为。
2. 明确 1m 尚未闭合时，Live 指标到底使用当前动态 Bar 还是上一根 completed Bar。
3. 把结论写成测试，不靠注释约定。

### V3-4B-1：ROI Candidate Detector

1. 根据 EntryPrice、Side、Leverage 和 1m High/Low 推导该分钟可能出现的 ROI 区间。
2. 只有 ROI 区间可能跨过 Gate 才加载 1s。
3. 没有仓位或完全不可能跨 Gate 时继续纯 1m。

### V3-4B-2：Second-level Environment

1. 在候选分钟按秒构建 NowPrice/ROI。
2. Technology、MarketCondition、Funding 严格 as-of。
3. 编译后的 expr Program 继续缓存，避免每秒重新 Compile。
### V3-4B-3：Second-level Close Strategy

1. 1s 内首次越过 ROI Gate 后评估对应 close rule。
2. Rule=false 继续下一秒，不因单纯触线强制平仓。
3. Rule=true 才产生高精度 close signal。
4. 如果同秒还有 Hard Stop/TP/Limit 等事件冲突，再进入 trade resolver。
5. 最终成交继续统一走 fee/slippage/position closing 逻辑，不复制第二套 PnL 计算。

### V3-4B-4：结果一致性与 Engine V8

1. 无候选分钟：Adaptive V8 必须与标准 1m 基准一致。
2. 有 ROI intrabar trigger：验证 V8 能捕获 V6 看不到的分钟内 close。
3. 不允许使用未来 1m close / 未闭合高周期指标。
4. EngineVersion 升至 V8；历史 V6/V7 Run 保留原结果。
5. 更新 DataHash / Audit / UI。

## 9. 测试矩阵

必须至少覆盖：

- 1m 无歧义：0 次 1s/trade 读取。
- 1m 同时触达 TP + SL：1s 决定先后。
- 1s 同时触达两个事件：trades 决定先后。
- trade_time 相同：trade_id 提供稳定 tie-break。
- 1s archive 缺失：trades 可聚合目标分钟 1s。
- CHECKSUM 错误：fail closed，不导入数据。
- Archive 404：进入 fallback，不无限重试。
- 同一 `{kind,symbol,date}` 并发请求只下载一次。
- Cache hit：重复回测不再次访问公网。
- Cancel：中断 download/parse，不写半截数据。
- Archive checksum 改变：最终 DataHash 必须改变。
- 未来 1s/trade 不可被当前秒读取。
- ROI Gate 触线但 close rule=false：不得平仓。
- ROI Gate 触线且 close rule=true：按首次真实成立时刻处理。
- MarketCondition 只使用当前时间之前的值。
- Funding 不重复计提。
- LONG / SHORT 两侧触发价方向分别测试。
- 高精度 PnL 与统一 fee/slippage 逻辑一致。

网络集成测试单独使用 build tag / manual test，普通 `go test ./...` 不依赖 Binance Public Data 在线可用性。

## 10. V3-4A Gate

- Go 原生 Public Data Client 能下载、验证和解析 USD-M 1s/trades archive。
- 普通无歧义回测不会下载 1s/trades。
- 多事件 1m Bar 可以被 1s 正确排序。
- 同秒冲突可以被 trades 正确排序。
- 所有高精度数据都是 lazy + sparse，不做全历史秒级入库。
- 不引入未来函数。
- Standard 1m 模式结果与 V3-3 基线不变。
- `go test ./...`、相关 race、build、SQLite/MySQL schema Gate 全部通过。
## 11. V3-4B Gate

- 当前 ROI Gate + close strategy 可以在候选分钟内按秒重放。
- 1m High/Low 只用于候选筛选，不直接决定 close rule 结果。
- 秒级 Env 与 Live 时间语义有明确测试约束。
- 同秒多事件继续下钻 trade，不猜 High/Low 顺序。
- Adaptive Run 的 high-resolution evidence 进入 DataHash 和 Audit。
- 无高精度候选时，性能应接近 V3-3 1m Engine。
- 不为了 Adaptive 模式预下载整段 1s/trades。

## 12. 本阶段明确不做

- 不把所有历史全量转换为 1s。
- 不把所有 trades 永久写入数据库。
- 不做 order-book queue / maker 排队 / 部分成交深度模拟。
- 不做毫秒级撮合引擎复刻。
- 不把 contract last price 当成 liquidation mark price。
- 不让 LLM 参与 Intrabar 顺序判断。
- 不做任意开仓规则的全天候 1s 扫描；V3-4B 只对 1m 已证明需要高精度的候选分钟下钻。
- 不修改旧 Backtest Run 结果。

## 13. Definition of Done

> V3-4 完成后，Backtest 仍以 1m 为主时间轴；绝大多数历史 Bar 保持 V3-3 的速度和存储成本。只有当 1m 无法确定执行顺序或当前 ROI Gate 在分钟内可能触发时，Engine 才按需取得 Binance 官方 1s/trades 数据进行更高精度重放，并把所有下钻证据、数据版本和最终决策完整记录，且不存在未来数据泄漏。
## 14. 实现结果（2026-09-11）

- Engine：`standard_1m` 保持 `backtest_engine_v6`；Adaptive 使用 `backtest_engine_v8`。
- 数据源：Go 原生 Binance Public Data Client，支持 1s Kline / trades、CHECKSUM、proxy、retry、cancel、singleflight。
- 存储：新增 sparse `market_klines_1s` / `market_trades`，只保存实际下钻使用的数据；写入使用事务避免取消时留下半截 canonical 数据。
- 时序：Live `InitParseEnv` 使用当前动态 Kline；Adaptive 以截至当前秒/当前 trade 已知数据构造 partial Kline，不读取未来 1m/高周期数据。
- ROI：1m High/Low 只做 Candidate Detector；真正 close rule 在 1s/trade Env 中重新执行，`rule=false` 不平仓。
- Funding：按当前 second/trade 时间推进，同秒较晚 Funding 不会泄漏给更早 crossing。
- 审计：Run 保存 resolution mode/model/stats；Trade 保存 entry/exit resolution；实际使用的高精度 evidence 合并进 DataHash，并写入 Audit Event。
- UI：支持标准 1m / 自适应精度选择，展示 drill-down/cache/download 统计、Trade resolution badge 和 Intrabar evidence。
- API：`/agents/backtests/:id/trades` 与 `/events` 改为分页响应 `{list,total,page,limit}`；默认 Trades 20 条/页、Events 50 条/页，避免大结果一次性加载造成页面卡顿。
- 执行假设：当前回测模型没有独立的 Limit 挂单/部分成交对象，因此 `IntrabarResolver` 的 Limit 排队简化假设尚未进入生产执行路径；未来加入 Limit 模型时必须显式记录对应 Run metadata。
- 数据库版本：V3-4 从现有 **v10** 顺序升级到 **v11**。若后续与另一个同样使用 v11 的分支合并，再由合并结果统一提升到下一个数据库版本，避免提前跳号。
- 验证：`go test ./...`、`go test -race ./service/historicalmarket ./service/backtest`、`go build ./...`、前端 `pnpm build`、`git diff --check` 均通过。SQLite schema/idempotency 自动测试通过；本次未直接修改服务器 MySQL/PostgreSQL，部署时统一执行 `./go_binance_futures sync db`。
