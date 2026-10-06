# 资金费结算标记价格：2023实际缺失边界获取探查

当前结论：本轮八个官方funding GET均实际HTTP451/terminal exit22，原因是restricted-location访问限制。结果是**获取未成功**，不是markPrice为空或零资金费；不重试、不换路由绕过限制。Goal仍active，可继续其他证据工作。

## 收益前选点与范围

根据不可变RG31 all12成本审计 `73967e86994316390b9849a63a0a416bdb266a7aebd224cf262f495f0ba710b4`，每开发币每个2023–2026年选取第一个和最后一个实际被应用但mark缺失的时点。选择先于网络结果；只有2023存在此类时点，共八个，不重请求已有2022四例。

|币|2022 unique applied missing|2023 unique applied missing|2024/25/26此账本missing|
|---|---:|---:|---|
|BTCUSDT|82|127|0/0/0|
|ETHUSDT|34|140|0/0/0|
|SOLUSDT|24|51|0/0/0|
|XRPUSDT|24|69|0/0/0|

统计只涵盖RG31三个候选×四币实际应用的缺mark集合，不是全部历史funding可用性。2024+计数0不能证明未持仓时点的完整mark覆盖。

## 八个实际请求结果

|币/边界|funding timestamp ms|原rate|实际HTTP/terminal|
|---|---:|---:|---|
|ETHUSDT/first|1673049600006|0.00005444|451/22|
|SOLUSDT/first|1673280000006|0.0001|451/22|
|BTCUSDT/first|1672531200000|0.0001|451/22|
|XRPUSDT/first|1672531200000|0.0001|451/22|
|ETHUSDT/last|1698134400000|0.0001|451/22|
|SOLUSDT/last|1698652800000|0.00009376|451/22|
|BTCUSDT/last|1698076800001|0.00000648|451/22|
|XRPUSDT/last|1697932800000|0.0001|451/22|

使用原官方GET `/fapi/v1/fundingRate`，symbol及精确startTime=endTime、limit1；返回的是地点限制错误对象，未取得funding数组，settlement_verification和mark_price_available均未验证。最后仍活XRP首次请求49415已实际观察terminal22；八请求全部已结束，不再poll旧句柄。

既有2022四例是真实HTTP200/rate-time-symbol匹配但markPrice为空，证据独立保留；2023八个451不推翻该证据，也不能外推“全部历史无mark”。未修改cache、funding、fallback、engine、conf、数据库、策略参数或收益；没有造mark或将失败当零费。

## 可复核原始证据

原始完成结果as_of：2026-10-05 18:04:16 UTC（2026-10-06北京时间02:04:16）。每个curl完整命令、stdout/error、初始及最终真实句柄、HTTP/exit保存在raw JSON。

```text
27ababaf3c5eda7c43e5d7105077b8dbe6d490ae01a8fecaa1e68e08817f1132  results/20261006-funding-mark-2023-boundary-probe-selection.json
c97bc1e306905d023c965f595d855ca50915a7544e40e9b097a41e1a66740649  results/20261006-funding-mark-2023-boundary-probe.json
```

results根目录 `/Users/zhz/Library/Caches/go-binance-strategy-research/`。两SHA在当前目标轮重新读取核验exact。原2022完整证据见同目录 `results/20261006-funding-mark-availability-probe.json` 和项目 `2026-10-06-funding-mark-availability-probe.md`。

下一步不继续受限端点获取；对冻结RG32完整账本做容量风险描述及预声明成本压力验证，精确venue结算mark/历史盘口仍独立pending。此获取失败不构成策略无效或目标已经blocked/complete。

