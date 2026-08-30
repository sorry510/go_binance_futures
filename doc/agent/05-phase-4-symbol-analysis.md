# 05. Phase 4：单币/合约分析 Skill

## 目标

实现 `symbol_analysis`，让用户输入某个 U 本位合约后，Agent 基于最新数据生成结构化交易计划，而不是只返回自然语言观点。

## 输入

第一版 API 建议显式指定 Skill：

```json
{"skill":"symbol_analysis","input":{"symbol":"ONGUSDT","prompt":"分析当前是否适合交易"}}
```

先不做复杂 AI Intent Router；等 Skill 稳定后再支持纯自然语言自动路由。

## 数据分层

### 必需数据

- 本地 symbol snapshot：价格、24h 涨跌、成交额、高低开、更新时间。
- 当前 `MarketCondition`。
- 多周期 Kline，默认建议 5m/15m/1h/4h，具体由 Skill 配置。
- Funding Rate。
- 最近强平数据。

### 推荐数据

- Open Interest 及变化率。
- Taker Buy/Sell Ratio。
- Depth/盘口失衡。
- 交易量或成交额短周期变化。

数据获取失败必须写入 `data_missing`，禁止模型假装已经获取。
## Context Builder

不要一次把所有数据库字段和原始 Kline 全部塞给模型。建议先在 Go 中计算精简 Feature：

- 各周期趋势方向、区间涨跌、波动率/ATR 类指标。
- 当前价格相对近期高低点、均线/通道的位置。
- Funding 当前值与极端程度。
- OI 短周期变化。
- Taker 买卖占比。
- 强平多空方向和聚合金额。
- 盘口买卖深度差。
- MarketCondition 与 BTC/ETH 基准方向。

LLM 需要进一步细节时，再通过 Tool 按需获取原始数据，避免固定上下文过大。

## 输出结构

建议定义版本化的 `TradingPlanV1`，至少包含：

```text
symbol, as_of, market_condition
direction: long/short/neutral
confidence: 0..1
summary
entry_zones[]
stop_loss
 take_profits[]
long_trigger / short_trigger
invalidation_conditions[]
risks[]
data_missing[]
evidence[]
```

价格字段必须为数值并进行基本合理性校验；`confidence` 只能表达模型判断强弱，不能表示真实胜率。
## Skill 执行建议

1. 先通过 Context Builder 获取默认必要快照。
2. 调用 LLM 判断是否需要额外 Tool。
3. 允许补充 Kline/OI/Funding/Liquidation/Taker/Depth，但限制每类 Tool 次数。
4. LLM 返回 `TradingPlanV1`。
5. Validator 校验 Schema、价格关系、时间戳、symbol 一致性、data_missing。
6. Validator 失败进入 Repair；达到 MaxRounds 后返回失败，不生成半合法交易计划。

## API/UI

第一版建议新增统一 Agent API，而不是再增加专用 Controller Loop：

```text
POST /agents/tasks
GET  /agents/tasks/:id
```

前端显示 Task Progress、Tool 使用状态、最终结构化计划。后端同时保留原始 JSON 和适合 UI 的字段。


## 当前实施状态（已完成）

- 新增 `service/symbolanalysis` Context Builder；先读取本地 snapshot，再并发聚合 Phase 3A `MarketCondition`、5m/15m/1h/4h Kline、Funding、OI、Taker、Depth 与最近 1 小时强平。
- Kline 在 Go 中压缩为趋势、区间涨跌、波动率、区间高低和 taker buy share 等 Feature，不固定把原始 Kline 全量塞入 LLM。
- 除 symbol snapshot/价格不可用会终止外，其余数据源失败均写入 `data_missing`；本地 snapshot 超过 3 分钟额外标记 `symbol_snapshot_stale`。
- 新增只读 `get_symbol_analysis_context` Tool；Skill 默认必须成功调用该 Tool，仍允许按需补充 `get_klines`、`get_funding_rate`、`get_liquidations`、`get_symbol_snapshot`、`get_market_condition`。
- 新增 `symbol_analysis` Skill 和严格 `TradingPlanV1` Validator，校验 symbol、RFC3339 `as_of`、MarketCondition、方向、confidence、价格关系、neutral 语义和数组字段。
- Runtime 新增 run-level Validator 能力，将本轮成功 Tool Result 提供给 Skill；最终计划必须保留 Context Builder 的 `data_missing`、匹配实际 MarketCondition，且 `evidence.source` 必须来自本轮实际成功调用的 Tool。
- 新增通用 `agent/manager` 与 Memory Task Store 入口；Manager 预分配 Task ID，Runtime 使用同一个 ID 写入 queued/running/tool/validating/succeeded/failed 状态。
- 新增统一 API：`POST /agents/tasks`、`GET /agents/tasks/:taskId`。Controller 不维护 `symbol_analysis` 私有 Agent Loop。
- Phase 4 不注册任何 write/trade Tool，不调用真实下单 API。

## 测试

- BTC/ETH 等高流动性正常数据。
- 新币/数据不足。
- Funding/OI/强平其中一个 API 失败。
- LLM 要求重复 Tool，Runtime 能限制。
- LLM 返回不存在的 symbol 或明显异常价格，Validator 拒绝。
- neutral 场景可以合法没有 entry，而不是强迫生成交易。

## Phase 4 验收

- 任意有效 USDT 合约可启动分析 Task。
- 输出包含 `as_of` 与 `data_missing`。
- Agent 获取的数据都可以从 Tool 调用日志追溯。
- 不调用任何真实下单 API。
- 同一输入可使用不同 LLM Provider 而不改变业务接口。