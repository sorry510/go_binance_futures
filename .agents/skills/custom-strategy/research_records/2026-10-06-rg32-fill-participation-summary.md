# RG32 成交名义金额/分钟成交活动：完整风险诊断

当前结论：12run/2215笔全部一对一配对，RG32自身871、控制1344，缺行0/重复0/零活动0。没有定义通过阈值；这是容量风险描述，不是历史盘口或5bps实际可成交证据。所有原交易/PnL保持，不能筛去高占比或亏损交易。

只读新helper实际Node syntax0，11个数值/百分位/零活动/非法输入自检通过，完整运行actualexit0。输入study/cost/phase与原冻结SHA核对exact，全部portable/data/engine/config身份一致。nearest-rank百分位固定为sorted[ceil(p*n)-1]，空组null。每笔名义金额abs(quantity×fill price)/对应canonical分钟quote，分别记录entry和exit；整分钟quote是事后交易活动，不是瞬间book depth，也不是可用容量上界。

## 自身完整币别结果

|币/交易|entry notional中位/最大USDT|entry ratio p50/p95/p99/max %|exit ratio p50/p95/p99/max %|
|---|---|---|---|
|BTCUSDT/214|901.189406/1132.379094|0.005536 / 0.034902 / 0.094033 / 0.100948|0.003352 / 0.019939 / 0.035518 / 0.049526|
|ETHUSDT/227|1012.229859/1702.773206|0.011026 / 0.072552 / 0.137541 / 0.217613|0.004313 / 0.042509 / 0.072881 / 0.089755|
|SOLUSDT/224|1205.469630/1642.553598|0.061636 / 0.456153 / 1.433184 / 2.266656|0.033255 / 0.264733 / 0.640117 / 1.181369|
|XRPUSDT/206|957.537231/1296.112521|0.112791 / 0.791258 / 1.484501 / 8.017631|0.049693 / 0.420035 / 0.737158 / 0.912180|

|范围|交易|entry ratio p95/max %|exit ratio p95/max %|
|---|---:|---|---|
|all|2215|0.370474 / 8.017631|0.223001 / 1.188636|
|own|871|0.425253 / 8.017631|0.231679 / 1.181369|

自身开仓名义金额最小670.545593/最大1702.773206 USDT，中位966.226154；exit最小641.173111/最大1730.041906。不能根据中位数忽略XRP/SOL的高占比尾部；分币/侧完整quantiles、每笔原字段与top10 entry/exit都保留raw JSON。

## 高占比尾部（非删除名单）

|币/seq/side|leg|UTC时间|notional|minute quote|trades|ratio %|
|---|---|---|---:|---:|---:|---:|
|XRPUSDT/3/SHORT|entry|2022-09-05T01:51:00.000Z|800.375721|9982.695540|58|8.017631|
|XRPUSDT/23/SHORT|entry|2023-02-26T10:48:00.000Z|904.063515|11643.383230|159|7.764612|
|SOLUSDT/15/SHORT|exit|2022-12-06T16:27:00.000Z|749.860528|63473.877000|184|1.181369|
|XRPUSDT/1/SHORT|exit|2022-09-02T19:14:00.000Z|792.831036|86916.055430|197|0.912180|

最大entry是XRP seq3 SHORT，约800.38USDT占该分钟9982.70USDT/58笔的8.0176%；不能断言“没有盘口”或“必然无法成交”，但缺少5bps fill的直接证据。最大exit是SOL seq15 SHORT，约749.86USDT占该分钟63473.88USDT/184笔的1.1814%。这些不是改变candidate的币/时间/方向过滤理由。

## 身份和下一动作

```text
a9e3e107ecf1211f2cd6c6552230bca50e9bfea9201601198dcfdda14579d6a9  results/20261006-rg32-fill-participation-diagnostic.json
3aaadb34693240501cfdf70d65aba0adacfeee45fd004e781c9b8ccc521aa945  verification/rg32_fill_participation_diagnostic.cjs
b2059f909944cfffb27786ab320e6a1ae2fa596d33e08d72b1ad2c099d4abc1f  项目research_records/2026-10-06-rg32-fill-participation-protocol.md
```

results/verification根目录 `/Users/zhz/Library/Caches/go-binance-strategy-research/`。raw创建2026-10-06T01:46:25.654Z。无网络/DB/App/生产/frontend/conf/_test.go/globalmemory写入；原完整RG32候选不变。

下一动作已另预声明单一10bps（原5bps加倍）完整顺序压力回放，不做bp网格或静态扣费，AF0/RG31/RG32×同四币同49月数据；压力算术不补齐实际盘口或精确mark，跨币发布仍pending。容量诊断现已可复核，不是可用策略发行证明。

