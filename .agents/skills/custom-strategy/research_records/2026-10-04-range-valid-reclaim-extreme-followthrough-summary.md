# RG19：取消区间收缩限制的极值确认 — 完整阶段总结

## 当前结论

2026-10-04北京时间20:58实际观察主84429、开23085、关98616、成本93763全部terminal exit0；会计/自然归因/汇总均exit0。RG19 invalidated，尚无达标策略。12完整49月run、1548交易账目、8共享AF0/RG18完整控制exact、会计errors0。不是12个独立候选，也不重复启动旧句柄。

本地go_bn_test v29 ID114已本轮repeatable-read程序只读核验，与冻结原ARMv29技术/策略语义相同；AF0是有意研究v29C变体。用户另行授权的RG18已经入库ID124，本阶段没有新DB写/分配/启用/下单、App/production/frontend/conf变化或仓库测试文件/删除，所有dirty与旧失败证据保留。

## 唯一变化和不变门槛

只从RG18补充入口去掉recent_width<older_width条件，保留两段正宽度合法性；其余closed扫边+收回、首次性、合法反向主动quote多数和放量、weak OR aligned strong、daily排除、ATR恢复及old-close锚追价限制、strict current超过closed High[1]/Low[1]、90%观察累计QPS、原9指标/基础入口整对象/完整uniformRG4关闭不变。不用forming taker[0]或MarketCondition，保留故意[0]。

UTC2022-09-01..2026-10-01exclusive共49月/1491天/213周；每币>=0.9次/周至少192笔。原engine_v7 standard_1m/已观察close→next open、cash1000起、当前cash10%margin×8、每侧fee0.0005/slip5bps、真实funding与原分钟Close标记价回退、outer5/5及AutoStop=false保持。不降低频率、缩短四年、改变年界或静态删除组。

## 四币实际组合表现

金额USDT，频率用全213周。

|币|笔数|次/周|毛收益|净收益|PF|最大回撤%|平均持仓小时|
|---|---|---|---|---|---|---|---|
|BTCUSDT|145|0.681|876.631|708.765|1.499|11.910|32.918|
|ETHUSDT|144|0.676|937.069|780.919|1.476|15.618|19.502|
|SOLUSDT|172|0.808|510.216|338.375|1.164|27.766|13.200|
|XRPUSDT|136|0.638|712.276|573.079|1.344|25.839|13.975|

四频率全部失败。597实际组合交易=188补充+409组合内基础；不能用597减独立AF0的429推算新增交易，持仓/现金/机会占用会改变组合基础路径。

|币|2022-09..2023-08|2023-09..2024-08|2024-09..2025-08|2025-09..2026-08|额外2026-09|
|---|---|---|---|---|---|
|BTCUSDT|77.710|55.015|160.249|433.065|-17.274|
|ETHUSDT|-8.665|167.339|254.448|323.361|44.437|
|SOLUSDT|-238.141|263.427|118.024|203.618|-8.553|
|XRPUSDT|317.492|-291.191|263.319|253.375|30.084|

BTC四完整年正但频率失败；ETH/SOL/XRP存在完整亏损年度。

|币|日历2023|日历2024|日历2025|2026-01..09|去最佳5描述性剩余|
|---|---|---|---|---|---|
|BTCUSDT|190.720|77.806|150.124|318.689|363.368|
|ETHUSDT|143.062|130.951|180.496|448.452|285.365|
|SOLUSDT|13.253|-29.931|45.333|304.356|-16.982|
|XRPUSDT|223.470|86.239|-211.180|456.005|120.768|

日历SOL2024及XRP2025仍负，SOL组合最佳5之外也负。2022 Sep–Dec另见完整accounting，不冒称完整年。各年度/45月/组均为原完整顺序账本退出时点归因，不是独立初始化或删除交易后新PnL。

## 补充机制与自然归因

|币|补充笔数|毛收益|净收益|去最佳5|
|---|---|---|---|---|
|BTCUSDT|51|92.483|37.411|-222.494|
|ETHUSDT|42|349.812|310.323|-69.026|
|SOLUSDT|44|250.311|210.634|-28.579|
|XRPUSDT|51|-72.188|-120.179|-340.193|

前三币补充净正，XRP毛净负；四币去最佳5补充均负。组合正额不能等同补充族有稳定共同alpha，没有单独family完整回测。

|币|弱笔数/净|强笔数/净|弱去最佳5|强去最佳5|
|---|---|---|---|---|
|BTCUSDT|26 / -10.895|25 / 48.306|-140.583|-179.417|
|ETHUSDT|26 / 118.123|16 / 192.200|-46.279|-159.826|
|SOLUSDT|20 / -74.680|24 / 285.314|-123.634|46.100|
|XRPUSDT|31 / -84.253|20 / -35.926|-217.729|-226.162|

|币|收缩笔数/净|非收缩笔数/净|收缩去最佳5|非收缩去最佳5|
|---|---|---|---|---|
|BTCUSDT|30 / -46.768|21 / 84.180|-256.041|-101.458|
|ETHUSDT|25 / 51.986|17 / 258.337|-116.199|-77.792|
|SOLUSDT|20 / 42.139|24 / 168.496|-132.979|-41.436|
|XRPUSDT|31 / -85.704|20 / -34.475|-290.113|-145.103|

全部188逐笔配对真实开关，weak/strong在原closed4h ADX20边界，收缩在recentWidth<olderWidth边界，收益前已定义，没有扫分组阈值。收缩106/非收缩82不是“RG18旧106必然同一交易集合”的证明；实际现金路径不同。XRP两组均负，非收缩全币去最佳5负，无可静态删组实现共同盈利的依据。唯一强SOL集中度较好也不能代替跨币验证，不从亏损推可交易逆向。

## 完整核验和成本范围

- Go/Expr88475：4862/0，完整唯一替换、独立numeric oracle、recent/older负/零/正与等宽/扩宽边界、strict price和90%活动联合及原close matrix；不是历史/live/forward/盈利通过。
- 真实frontend validator隔离VM两个issue=null/9enabled/四type，限定311portable709enabled entry/0精确重复；不是App/UI/build/API或全语义新颖。
- 开23085：188补充/15040closed字段/+940四小时/+564日线/+376ATR/+1128range/+564活动/+188strict extreme，原200input199closed，failed0；活动最小比0.9000338217046567。RangeValid全188，不再把ContractionPass作入场必要条件。
- 关98616：597正常/0forced，old497/added101/both1/added-only100，failed0；全部真实Position/ROI/outer gate/whole uniform RG4独立重算。
- 成本93763：all1548/3850资金费应用/1069缺mark原模型回退/0零活动/0failed；自身597/1450资金费/438回退/0failed。最大qty误差1.8189894035458565e-12，最大net误差1.8474111129762605e-13，没有改容差。all/own actualall_passed=true，是原模型核验通过，不是精确venue执行资格。

完整normal关闭与唯一ROI<=-20灾难例外保持；普通ROI越+5/-5且技术确认全无仍false。详见冻结protocol矩阵，不把outer资格替换为ROI-only立即平仓。

官方[资金费率历史接口](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data#Get-Funding-Rate-History)定义markPrice为结算相关标记价。既有rg8 raw probe的12个first/middle/last旧缺mark例子均HTTP200、时间/费率匹配但markPrice为空，本轮只读复核而非重复请求；不将空值变成精确标记价或虚构0资金费。精确历史mark、订单簿容量/真实slip仍pending，不称真实成本全齐。未阅AAVE/ATOM/ETC/LINK继续保留，当前开发失败不消耗验证币。

工作树另一个4x/45月/ROI-only/different gate诊断相关开发收益已看，不能替代本原合同或称全局未阅假设；原8x/49月/完整关闭候选在本protocol保存前未读收益。不改变其文件/六额外验证币结果，不混入本轮对照。

## 完整文件身份

两完整portable：temp_strategy/20261004-range-valid-reclaim-extreme-followthrough/，family SHA0943023388c6b9c5b813fdfef4ba0f3cb55b42b87d79480215fee9587cfc2f66/version4e20ef722c3fb61dbd347499f1c6c7325d39c643cbf1c1e7fcfa585224987782；combo SHA99ccf758dc78e33596487f81268150918371bc8396301cb88f8b69b8a393b1ce/version7b3157b2dbc861867601057919739be19240325b50403d0afc82c66ed2851cd5。

收益前protocol SHA151fe931905b0b7c73df696131ac4e3b018776eec9de9d2050cdb5feedb4bcf6。
结果根/Users/zhz/Library/Caches/go-binance-strategy-research/results/：

|完整文件|SHA256|
|---|---|
|20261004-rg19-development4-canonical-repaired-v2-funding-tail-v1.json|ee260c777e6ecb10a4c696841ac878f32923defa6933b18c8125db6f566f89e3|
|20261004-rg19-accounting-summary.json|f49b0a4606fe638168fe7f40eb084c95060b63aba54c132cee28e6daca47fe02|
|20261004-rg19-expr-checks.json|a52e06a0d0be15f3fecc735053cc51e1293f77237c2cfce4114426541517f45c|
|20261004-rg19-canonical-actual-seed-open-signal-audit.json|5ba0e7fe2ae1c9ceb9d5080668040159eb9770923ac60d041b282cd671114b37|
|20261004-rg19-original-position-close-signal-audit.json|a8bf5db51caa7b9e1cf2f1ee0aae001fa917dedcecc90eb42bdcd5d6fc41f164|
|20261004-rg19-execution-original-cost-fallback-audit.json|067bde525b85319ac9c2e31b9b4f6e72ba260dcf04271d95c1a9393657e122f0|
|20261004-rg19-natural-entry-regime-attribution.json|8063dfa23a88c7fb0163326039d6278f9d331f95f941f8a3246448f03d20dbb6|

phase-evidence-summary及本地一致快照/前端/identity SHA由当前文件另核，不覆盖旧raw。源engine/environment/cache、protected conf/正式SKILL与funding/data/main身份仍冻结见protocol；用户metrics timestamp-max行、dirty和所有pending SkillMax草案保留，未新委派或晋级。

## 下一单机制假设

只检验“追价上限是否应锚定刚确认的闭合极值而非闭合Close”：LONG current <= High[1]+0.35ATR；SHORT current >= Low[1]-0.35ATR，仍要求strict current超过High/Low。原旧cap和strict cross共同蕴含信号wick必须<0.35ATR，较长合法wick会使确认入口不可能通过；这是代数限制，不是已知被拒交易收益。

保留同一0.35ATR系数、全部闭合价量/首次性/日线/原9/90%累计活动/基础和关闭；新cap会扩大相对Close的允许位移，可能增加追价损失，不能先宣称可盈利。下一RG20完整JSON/独立cap边界和两侧可达、旧程序唯一替换、真实前端/protocol需先完成，再AF0/RG19/RG20×四币完整12run及全部核验；不得把本描述当已生成/启动。

本goal阶段为实际progress，目标active但未达成，不自complete/paused/blocked。下次用量恢复前先阶段总结和真实状态检查，不重poll本轮已terminal句柄。

