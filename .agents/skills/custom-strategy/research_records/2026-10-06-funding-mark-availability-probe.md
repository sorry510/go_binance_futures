# 2026-10-06 资金费精确结算价来源探测

## 当前发现与范围

RG31主89784原冻结回测运行期间，只读探测RG30原all12成本审计中每币**第一个实际使用缺mark的时点**，不是按收益筛选。四个GET实际terminal0/HTTP200，symbol与资金费时间exact、数值费率误差0，但markPrice都是空字符串。它们不能直接补齐精确结算价，不能把空mark当成资金费为0、制造合成mark或改本轮fallback。

[官方Get Funding Rate History](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data#get-funding-rate-history)定义markPrice为相应资金费结算价，支持inclusive start/end与最高1000 limit；实际这四个旧记录字段为空。**只证明本次四个案例，不能推断整个历史均不可用**。没有绕过限流/地区限制、改route或重试；4requests，无认证/App/DB写或冻结data/cache/runtime修改，也没有读验证币或收益。

|币|精确UTC资金费时间|原费率=API费率|API markPrice|RG30审计独有应用缺mark时点数|
|---|---|---:|---|---:|
|BTCUSDT|2022-09-05T08:00:00.013Z|-0.00000737|空|258|
|ETHUSDT|2022-09-05T08:00:00.013Z|-0.00010286|空|235|
|SOLUSDT|2022-09-09T00:00:00.011Z|0.0001|空|87|
|XRPUSDT|2022-09-02T08:00:00.013Z|-0.00001854|空|104|

独有时点数不是应用次数：同一资金费时点可被多个candidate持仓应用，原all12仍1379fallback。完整原JSON、raw API body及验证字段保存于/Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-funding-mark-availability-probe.json；原成本审计SHA097604ae1117258786e6cc7619ba9546bc06805d1a41d77316cfd64ed5276034。

精确venue marks与历史盘口/真实滑点仍pending，不因为原模型算术/分钟activity成功就声明“真实成本全部通过”。后续如获得独立实际mark，仅在另存、先核timestamp/rate/coverage/source identity的新研究数据版本下完整重跑可比controls，不能静默覆盖现冻结研究。

