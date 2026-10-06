# RG32 强趋势亏损侧趋势动量联合退出：完整开发阶段总结

当前结论：**promising under tested conditions；release_qualified=false**。这是本轮谱系中第一个通过全部预声明开发数值门槛的完整候选，不是已验证可用/盈利保证。精确资金费结算标记价、历史盘口容量/真实滑点、未阅跨币验证仍未完成；不得用原模型成本算术或分钟成交活动替代这些证据。

## 实际完成与当前核验

2026-10-06北京时间01:58:02.924启动主60518，02:04:16实际观察terminal0；opening63189、closing28001、all12成本51804于02:06:12.218启动，02:10:21实际观察全部terminal0；natural/phase在02:12:06实际exit0。旧句柄全部结束，不再poll/restart。当前轮只读进程检查无本轮遗留主/审计；重新核验收益前61文件SHA、七结果引用SHA全部exact。后续摘要不是新的回放。

前一目标轮属于progress：完成冻结RG32的12次完整顺序回放、全部新开/关/成本审计与自然归因，得到了数值门槛首次全通过的证据；下一步应冻结该候选补成本证据，不再自动生成RG33或按已见2025利润继续调参。

## 目标、风险与唯一变化

窗口UTC2022-09-01 inclusive至2026-10-01 exclusive，1491天/213完整周，四个完整Sep–Aug年度及额外2026Sep。BTC/ETH/SOL/XRP各独立1000USDT等初始钱包。每币至少171笔（ceil(213×0.8)），无需每币或每币每年盈利。不能使用active-week分母、四舍五入近零亏损、筛币/侧/年或事后更换年界。

standard_1m/backtest_engine_v7；当前available cash 10%保证金×8倍，outer profit/loss=5/5，每侧fee=0.0005、adverse5bps、实际历史funding。当前minute-close决定、下一分钟open成交，不是立即固定±5%成交。原缺mark时使用funding所在分钟Close回退。MAIN无诊断overlay，未改变engine/data/config或真实风险。

唯一变化：RG31新增strong_structure_reversal的trend_fail OR momentum_fail改为AND；ROI≤−5、closed4hADX≥20、strict反侧closed1小时极值突破不变。完整旧pre-strong RG20 weak/profit16/28/loss12/ROI≤−20 emergency保留，尤其旧loss12仍OR确认。四entry整对象/顺序和九指标参数全部不变。

假设先在RG31总结声明并冻结：局部动量失败未必等于高周期趋势失效，检验共同失效能否减少强趋势内误退出。两个确认均是价格派生，不是统计独立alpha。增加持仓/资金费或亏损深度是接受检验的反面风险；不是2025补尾、ROI网格或静态删止损。

无MarketCondition/形成中taker[0]/OpenStrategyHash路由；有意保留forming[0]。九指标跨1h/4h/1d，是继承基线研究组合，不声称已简化为低开销发行版。

## 完整开发数值

|币|笔数|每213周频率|gross|双侧fees|funding|模型net|PF|单币引擎DD%|平均持仓小时|
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
|BTCUSDT|214|1.004695|555.548914|194.602757|-23.966763|+336.979393|1.182003|14.738487|27.066277|
|ETHUSDT|227|1.065728|1268.435968|241.394361|-20.314930|+1006.726677|1.421141|15.685370|19.517034|
|SOLUSDT|224|1.051643|1264.569794|259.655153|-33.125719|+971.788922|1.307552|26.703528|12.324553|
|XRPUSDT|206|0.967136|783.321901|203.617902|-5.524921|+574.179078|1.255730|33.756070|11.677265|

自身871笔=480补充+391在组合内base；不能减去独立AF0交易数推算贡献。12完整run/2215笔，全会计八AF0/RG31完整控制snapshot、ledger、metrics、source/config/data/date精确复现，errors=[]。四独立钱包组合net=+2889.674070，初始4000下描述累计72.241852%；四完整年度组合和完整日历2023/24/25组合全部正。

|完整年度（Sep–Aug）|组合net|BTC|ETH|SOL|XRP|
|---|---:|---:|---:|---:|---:|
|2022-09至2023-08|+264.320477|+9.797419|-11.897526|-145.195533|+411.616117|
|2023-09至2024-08|+572.931254|+99.020026|+278.655793|+624.637438|-429.382002|
|2024-09至2025-08|+737.010045|-5.291729|+287.590129|+240.024222|+214.687424|
|2025-09至2026-08|+1303.257347|+265.610549|+422.720419|+277.685350|+337.241030|

|日历平仓归因|组合net|BTC|ETH|SOL|XRP|
|---|---:|---:|---:|---:|---:|
|2023|+742.023200|+131.747987|+200.061336|+173.350188|+236.863689|
|2024|+563.891922|+119.395901|+210.768688|+273.964724|-40.237392|
|2025|+90.205331|+4.055447|+211.689687|+45.624559|-171.164362|
|2026 Jan–Sep|+1547.763892|+144.895485|+510.593463|+435.877716|+456.397227|

2022Sep–Dec合计-54.210274；额外2026Sep合计+12.154947，都已计入49个月总账。年/日历/额外Sep是同一顺序账本的平仓归因，不是各年重新初始化回报。

单币亏损年度完整保留：BTC在2024Sep–2025Aug亏损，ETH/SOL在2022Sep–2023Aug亏损，XRP在2023Sep–2024Aug亏损且日历2024/25负。这不独立否决用户允许的固定四币组合，但不能称每币稳定盈利。

排除最大正贡献ETH后的描述net=+1882.947393，不是重新平衡/现金合并回测。组合realized-close-order DD=7.957187%，不是完整mark-to-market DD。

|币|LONG笔/net|SHORT笔/net|每币去最大五笔描述net|
|---|---:|---:|---:|
|BTCUSDT|126/+92.244508|88/+244.734885|+13.661333|
|ETHUSDT|129/+572.852128|98/+433.874549|+414.568783|
|SOLUSDT|137/+584.151936|87/+387.636985|+459.855409|
|XRPUSDT|125/+482.069259|81/+92.109819|+92.864772|

以上去最大贡献/最大五笔仅集中度描述，不是静态删除交易后的新策略PnL。相对RG31总net增加574.483410，但两者同一已观察开发期，不是独立验证。2025组合只有+90.205331的模型余量，成本增大可能改变年度门槛，应完整顺序再撮合验证而不是静态扣费后宣布通过。

## 所有实际信号与关闭审计

Go/Expr91276 passed/0 failed，独立typed RG20/RG31/RG32三模式numeric oracle；验证联合确认、单项拒绝、当前决定为父RG31子集、ULP/ROI/ADX/price-scaling、整对象继承及入口priority。不是private/live/forward或盈利证明。

开仓480补充/38400闭合字段，4h2400、daily1440、ATR960、range2880、current activity1440、strict followthrough480，0失败。真实200输入/199闭合种子，独立重建截至signal已观察分钟quote与canonical闭合量。累计QPS最小ratio0.9000338217；480/480 extreme cap通过，old-close cap124通过/356拒绝；最大标准化strict推进0.3493773462。不是elapsed flow rate或forming buyer确认。

关闭871 normal/0 forced，old827/added56/both12/**added-only44**，0失败，所有FormingIndicatorParity和OldPass==OldNumericPass通过。Old comparator实参是完整pre-strong RG20，不是立即父RG31；44是相对RG20，不能宣称相对父新增44次。fresh BuildMinuteClose overlay仅诊断，原MAIN不使用；Position/ROI/outer/cash/funding、canonical极值/closedADX和forming EMA/RSI/DMI按实际退出前分钟重算。

|LONG/SHORT共同场景|决定|
|---|---|
|普通±5 ROI gate，无技术确认|false|
|−12<ROI≤−5、ADX≥20、严格反侧破线，仅trend_fail或仅momentum_fail且所有旧分支false|false|
|同上trend_fail AND momentum_fail|true|
|ROI≤−12、旧loss12单项确认|旧分支true，新AND不会取消|
|缺严格破线/恰等极值、所有旧分支false|false|
|positive ROI且所有旧分支false|新增false|
|ROI在(-5,5)|outer不调用|
|原weak/profit16/28/loss12有支持信号|完整旧逻辑保留|
|ROI≤−20|true；唯一无信号灾难例外|
|空/错OpenStrategyHash|决定相同，uniform不依赖hash|

## 原模型成本审计与真实成本缺口

|范围|交易|funding applications|缺mark分钟回退|零活动成交|综合失败|
|---|---:|---:|---:|---:|---:|
|all12（含控制）|2215|4562|1404|0|0|
|RG32自身|871|1940|642|0|0|

按原data/hash重建current-cash compounding数量、下一分钟adverse fill/双fee/所有真实funding时点rate及原mark回退；all/own/control失败行全部[]。all最大qty误差2.2737367544e−12、net误差3.4106051316e−13，自身最大net误差2.4158453016e−13，在原容差内；fill误差0，不放宽精度。

这证明原模型计算自洽且成交分钟有活动，不证明5bps是该时点实际可成交滑点/盘口深度，亦不补齐精确settlement mark。自身funding1940与回退642高于父RG31的1529/494，持仓成本风险必须保留。四个2022 GET返回空mark的证据和八个2023 GET HTTP451的获取失败证据分开记录，不将451当空mark/零funding，也不换路由绕过限制。

## 全480补充自然归因：仅描述、不筛掉亏损币

|币|补充笔|gross|fees|funding|net|去五大盈利描述net|弱ADX笔/net|强ADX笔/net|
|---|---:|---:|---:|---:|---:|---:|---:|---:|
|BTCUSDT|123|-170.873411|111.492356|-7.515027|-289.880793|-575.851746|60/-88.572671|63/-201.308122|
|ETHUSDT|133|563.054670|137.195231|-6.730958|+419.128481|-80.561942|79/+14.394570|54/+404.733911|
|SOLUSDT|100|996.475345|113.176243|-4.828594|+878.470508|+425.499254|49/+95.006492|51/+783.464016|
|XRPUSDT|124|21.649914|120.712943|+0.760815|-98.302214|-457.969900|79/-275.429699|45/+177.127485|

|币/闭合入场ADX组|四完整年补充net依次|
|---|---|
|BTCUSDT/weak|-70.465646 / +5.047676 / -61.003502 / -7.947266|
|BTCUSDT/strong|+76.508832 / -7.471421 / -110.107153 / -142.579044|
|ETHUSDT/weak|-15.723500 / +47.977182 / +67.687551 / -139.816592|
|ETHUSDT/strong|-5.053936 / +186.963528 / -14.069237 / +236.893557|
|SOLUSDT/weak|+36.159922 / -21.639416 / +106.575782 / -66.645466|
|SOLUSDT/strong|-7.531130 / +580.594922 / +80.035421 / +130.364803|
|XRPUSDT/weak|+23.647882 / -84.075220 / -67.944044 / -156.764023|
|XRPUSDT/strong|+101.973078 / -180.438964 / +101.789355 / +153.804016|

|币|收缩/非收缩笔net|old-close cap通过/拒绝笔net|
|---|---|
|BTCUSDT|contracted 60/-301.381925；noncontracted 63/+11.501132|old_close_cap_pass 31/+77.127754；old_close_cap_reject 92/-367.008547|
|ETHUSDT|contracted 76/+180.997840；noncontracted 57/+238.130642|old_close_cap_pass 25/+147.409805；old_close_cap_reject 108/+271.718676|
|SOLUSDT|contracted 46/+244.415228；noncontracted 54/+634.055280|old_close_cap_pass 30/+38.174816；old_close_cap_reject 70/+840.295692|
|XRPUSDT|contracted 62/-65.828822；noncontracted 62/-32.473392|old_close_cap_pass 38/-53.215776；old_close_cap_reject 86/-45.086438|

没有跨币、跨年一致稳定的自然组。BTC/XRP补充net仍负，ETH收益更依赖少数大单，SOL补充表现较强；不能据此删除BTC/XRP补充、套用强弱筛选或把added-only亏损删掉并静态加回PnL。完整组/侧/年度/exit属性已保留原natural JSON。

## 预检、范围与文件身份

frontend实际VM两issue=null、9 enabled、四type及shape；338portable/695close限定whitespace-exact扫描0重复，仅身份不证明全语义新颖/UI/API/盈利。close build/三个Node syntax实际0。本地v29只读71161实际0：go_bn_test ID114技术/策略语义exact、0writes。API实际exit7/3333无连接/rule响应[]，Go fallback；没有启动App/服务。

family SHA839c4d33e23ce0786cdd72ca705512ff468daf737bcd744ce1330b808acf4f7d；完整combo SHA51b182991491ef122942bc5f29a33f4a28e2439234da03da7359d31584112638，raw versionb38ad611ee214d7cad803337a9287a2eb03a2522af8d121a2e6dbeee8b94f8e7。所有候选含失败均留temp_strategy；family没有单独正式回放。

收益前protocol SHA6d365401de7af9de5a7680f87553f73fcf2dd928ae46c4a6a346247d9fedf20c；spec SHA9932d0314d1d5bc325c23bc1c36ba7d2faa50a2a6d79a9fe55c83d007dd8846a。协议/规格中的尚未回放是冻结时历史，未回写成收益后协议。正式skill/生产/frontend/conf不变；保留全部用户dirty。

结果根目录 `/Users/zhz/Library/Caches/go-binance-strategy-research/results/`：

```text
05b465ff70930572ca343d77db01f539963afb0ab7afc99fd05d7eca3ac90a1b  20261006-rg32-development4-canonical-repaired-v2-funding-tail-v1.json
25071c979e0b3d66f56eaa69097a3ca995a11ae18ae4fd844b899554f4d2c9be  20261006-rg32-accounting-summary.json
e7be471628c69429ff1d3540f825d37d57578925615e747efe9a70f84d9e4dae  20261006-rg32-expr-checks.json
7d8df6572292e497443ba4aa0e38e759906d01fa988c88a73e9e5066e4ebd553  20261006-rg32-canonical-actual-seed-open-signal-audit.json
5d6551e36f0f038a23dea78a78e2dba4dd1fdda495a88d570f2397e5af49e1fa  20261006-rg32-original-position-close-signal-audit.json
7a740473798629e4cca8f304b9d539cebbdae4100eaeb89852d22f5c86ada47e  20261006-rg32-execution-original-cost-fallback-audit.json
8c9ee876d9270544283268db678c52263a1e83a9414e3e1b99b6fa0cb2bba0e7  20261006-rg32-natural-entry-regime-attribution.json
c699329ec9bf3bccf8017212dd29fdc291fab56184e24449848dbb12f1de7a5b  20261006-rg32-phase-evidence-summary.json
```

## 下一步与验收界线

冻结RG32，不立即生成新参数候选；先对所有2215交易及RG32自身871的原成交名义金额/分钟quote和trade-count进行完整配对诊断，明确这只是容量风险描述，不是盘口/真实滑点证明。精确mark及历史book获取仍需可核验来源；不重试/绕过451，不修改cache/fallback，不造数据。

若做更严成本压力测试，必须先独立声明成本配置和原模型控制、使用同一固定strategy/data/49月原engine完整顺序再撮合，单独输出/审计；不能静态扣费、参数网格或把压力模型当实际盘口。AAVE/ATOM/ETC/LINK收益仍未读取，不降低已声明开发数值/信号/成本先行门槛。

goal保持active，最终每币频率/真实成本/四年稳定/跨币泛化仍未全部完成。本阶段可用的是完整研究候选JSON和开发审计证据，不是发行策略。没有研究DB写入、分配、启用、下单、App/生产/frontend/conf/_test.go/globalmemory变更、新委派或技能晋级。

