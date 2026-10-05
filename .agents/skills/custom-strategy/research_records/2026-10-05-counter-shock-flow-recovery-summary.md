# RG27 反向冲击失败闭合主动成交回收：完整49月结论

当前裁定：**invalidated**，release_qualified=false。12完整run/所有后审计与失败分类均结束；目标仍active，没有满足每币0.9/真实成本/四年稳定/跨币泛化的可用新策略。不是暂停、完成、挑币发布或零成本诊断。新RG28仅生成与前端合同通过，尚未Go/Expr或回测。

## 原合同与真实证据

本地go_bn_test原v29 ID114，程序化read-only27915 actualterminal0，精确名称唯一、技术/策略semantic与冻结基线一致，0writes。RG27只换RG26两补充entry，整个基础对象/order、原9指标、whole统一信号确认RG4 closes保留；不读forming taker或MarketCondition。两闭合bar quote/taker合法与反向shock→闭合回收机制属于独立新候选，不是反转旧亏单或已有alpha证明。

收益前protocol 2026-10-05-counter-shock-flow-recovery-protocol.md SHA7407ae9d104ef761896ef8c4bf9678049662ca259f6a3a09c0d2ba48cd08ae00。真实Go/Expr10934/0、openingbuild0、前端两issue=null/9/四types、限定328portable759entries0exactdup。HTTP真实exit7拒绝连接，code200不存在；Go/Expr回退只允许推进独立历史研究，HTTP/live/forward仍待服务恢复，未自动启服务或操作App。

UTC2022-09-01 inclusive→2026-10-01 exclusive，49月1491日213周。AF0/RG26/RG27组合×BTC/ETH/SOL/XRP原standard1m/backtest_engine_v7十二完整run。初始1000USDT/币/currentcash10% margin/8x/outer5-5/每侧fee.0005/每侧不利slip5bps/实际历史funding，原缺失精确mark的observed funding-minuteClose回退保持。Main41061北京时间16:07:44启动、16:13:43真实terminal0；不得重poll/重启。

所有累计值为原模型中的USDT，不是实际交易承诺。XRP包含下述尚无成交活动证明的模型平仓，因此尤其不能作为真实可执行盈利结论。

## 组合全量账目

|币|笔数|每周次数|毛收益|模型净收益|净PF|最大回撤%|平均持仓h|
|---|---:|---:|---:|---:|---:|---:|---:|
|BTCUSDT|169|0.793|569.831|413.847|1.303|20.769|29.880|
|ETHUSDT|160|0.751|232.373|96.357|1.061|20.780|18.488|
|SOLUSDT|184|0.864|415.384|237.130|1.109|27.693|13.109|
|XRPUSDT|125|0.587|656.159|530.726|1.354|23.489|15.023|

全部freq<.9，最低需要ceil(213*.9)=192笔；SOL184不能以四舍五入冒充.9，BTC169/ETH160/XRP125也失败。四币累计净正并不改变频率、年度或实际成本失败。

|币|双侧手续费|Funding净值|
|---|---:|---:|
|BTCUSDT|137.694|-18.291|
|ETHUSDT|124.623|-11.392|
|SOLUSDT|153.909|-24.346|
|XRPUSDT|118.645|-6.788|

每币原可用资金复利仓位，不是每笔固定最初1000的10%。gross/fees/funding/net逐笔与全量账目配对，没有双扣fees或单边费用省略。

## 四完整年度与完整日历归因

|币|2022/09–2023/08|2023/09–2024/08|2024/09–2025/08|2025/09–2026/08|额外2026/09|
|---|---:|---:|---:|---:|---:|
|BTCUSDT|-78.948|28.885|-3.482|483.040|-15.648|
|ETHUSDT|-47.456|-62.768|139.369|80.676|-13.463|
|SOLUSDT|-184.729|257.855|-46.144|229.134|-18.985|
|XRPUSDT|249.994|-237.164|190.364|268.777|58.757|

每币都有负完整年度，全部四年稳定门槛失败。BTC两年负、ETH两年负、SOL两年负、XRP一年负。这里是原完整运行的exit-time cohort，不是分别重置资金的年度利润。

|币|2023|2024|2025|2026 Jan–Sep|移除最佳5笔的描述净和|
|---|---:|---:|---:|---:|---:|
|BTCUSDT|100.715|-82.587|217.738|298.215|123.052|
|ETHUSDT|104.497|-106.788|90.317|132.533|-216.986|
|SOLUSDT|136.908|-102.960|-107.292|309.687|-157.871|
|XRPUSDT|103.054|160.968|-246.271|480.654|99.164|

完整2023/24/25各单列；四币都有负日历完整年，SOL2024/25为负。ETH和SOL最佳5笔之外的描述净和为负，显示集中度；移除交易只是诊断，不是新可执行回测或独立初始化收益。45月退出cohort和2022SepDec保留在完整accounting原件，不用45月替代用户要求的四年。

## 新家庭是否增添共同优势

|币|补充数|补充毛收益|补充费用|补充Funding|补充净收益|weak净|strong净|
|---|---:|---:|---:|---:|---:|---:|---:|
|BTCUSDT|71|-99.246|57.210|-1.792|-158.248|0.173|-158.421|
|ETHUSDT|62|-218.545|47.773|-1.002|-267.320|-118.427|-148.894|
|SOLUSDT|54|192.086|44.398|-2.643|145.045|-6.199|151.244|
|XRPUSDT|41|-61.192|38.775|-1.015|-100.982|-11.330|-89.652|

全部228补充+组合自身410基础=638；不能直接拿AF0独立429基础的收益与RG27做静态减法。持仓占用、后续机会及复利资金不同。BTC/ETH/XRP补充毛收益就为负，不能只归因费用，组合累计净盈利主要由基础规则贡献。SOL补充正也不泛化到其他币或完整年；最佳5笔以外的补充净和四币均负。

|币|补充侧|数|毛收益|净收益|
|---|---|---:|---:|---:|
|BTCUSDT|LONG|41|-197.532|-235.363|
|BTCUSDT|SHORT|30|98.285|77.114|
|ETHUSDT|LONG|28|15.748|-8.777|
|ETHUSDT|SHORT|34|-234.292|-258.543|
|SOLUSDT|SHORT|29|-2.386|-26.897|
|SOLUSDT|LONG|25|194.472|171.942|
|XRPUSDT|LONG|18|-102.117|-122.251|
|XRPUSDT|SHORT|23|40.925|21.269|

不存在共同稳定的预宣告weak/strong组；BTC strong四完整年都负，其余组也包含负年，不能挑SOL strong、XRP short或删weak组发布，不把净亏推导成可交易反向edge。

|币|自然regime|2022起完整年|2023起完整年|2024起完整年|2025起完整年|
|---|---|---:|---:|---:|---:|
|BTCUSDT|weak|15.567|-42.221|-63.847|43.248|
|BTCUSDT|strong|-104.292|-11.205|-38.571|-4.353|
|ETHUSDT|weak|33.479|-38.539|-33.560|-79.807|
|ETHUSDT|strong|-86.600|-45.323|19.348|-36.318|
|SOLUSDT|weak|17.907|11.803|-10.311|-25.598|
|SOLUSDT|strong|-20.605|79.922|-30.745|100.719|
|XRPUSDT|weak|42.403|-31.750|9.710|-20.432|
|XRPUSDT|strong|17.033|-74.191|-68.046|-6.230|

## 全会计与原接收器审计

会计actualexit0，12run/1830交易/8共享AF0-RG26完整controls与父RG26精确复现/errors[]；候选realpath/fileSHA和其余snapshot字段均严格一致，唯一source.cache_hit观测允许变化，不删其他字段。

开87219 actualterminal0：228真实补充/18240closed字段、1140四h、684daily、456ATR、456unusedEMA、1824priorquote、684activity、228strict followthrough全部0失败。Closed1/2两实际quote/taker与canonical相同，两flow均检查；原199closed指标种子、literalFloat64 QPS累计和完整两bar max/min独立核验。minimumactivity=.900006697395139、maxadvance=.3496442184652745ATR、shockbody .1503075423751498..1.8937496285262752ATR、minimumhalf=.5、maxrecovery/shockquote=.997005508764503、minimumrecovery/priorMean=1.0022460675189904。没有形成中taker信用。原函数重算不是独立数学实现，fresh receiver不是整个sequential/private/live/forward证明。

关70807 actualterminal0：638normal/0forced/old523/added116/both1/addedonly115/0失败。所有whole信号确认关闭通过，不存在静默ROI-only普通平仓替换。自然归因228全配对，phaseJSON actualexit0只是完整汇总成功，不代表失败成本被修成通过。

## 成本失败的准确分类

成本75755 actualterminal1：all1830交易/4267实际funding应用/1168mark回退，1零活动fill、1综合失败；自身638/1499/427，1零活动和综合失败；controls failed_rows=[]。全数值maxQtyError2.728e-12/NetError2.274e-13，没有超过原数值容差的价格、费用或资金费公式错误。控制台“1 arithmetic failed”是旧helper的笼统label，actual Passed同时要求!ZeroLiquidityFill与numeric parity；不据该label误诊为费用公式错，也不把综合audit改称passed。

具体为RG27 fullversion df95104b236f2507b25f6d11616dfc4f11c69f62d31f97e37067bf15764d3bee，XRP sequence59 SHORT，开rg27_counter_shock_flow_recovery_short，entry2025-01-14T12:47:00Z、exit2025-01-14T13:32:00Z（北京时间21:32）。原模型net−10.987742031380655。Entry2143 prints/quote2348749.08897；Exit0prints/quote0，baseOpen2.5809，模型slippedfill2.58219045。**完整失败row与损失保留；不删单、不将损失加回净值，不更换原结果。**

独立immutablecache上下文8613 actualterminal0、XRP dataSHA8aa4...774e3匹配：13:31有11671 prints/quote16388029.28938；13:32/33均0且OHLCcarry2.5809；13:34恢复1965 prints/quote2449016.77451/open2.5797。源修复与缓存policy不变，固定分钟链完整。诊断SHA582e6721f1cbc1796833ff3fe0f73b399e4ad9cb98f1da5522031f9777a7eaa8。

当前engine.go pendingclose在下一bar.Open直接closePosition，没有TradeCount/QuoteVolume门控。历史零prints本身不等于一定无订单簿/绝对不能成交，但当前研究缺少精确可执行fill证据，故该realcost/执行门槛失败。不能只替换这一笔价格或延后两分钟就宣称新利润；任何fill policy变化需合适授权、独立model/runtime标识和全部控制/候选重新完整运行。本轮没有修生产或引擎，也没有隐藏数据更换。

其余精确venue settlement mark、历史orderbook容量/滑点与原未阅AAVE/ATOM/ETC/LINK仍pending。分钟有交易或数值绿色也不自动等于这些真实成本通过；当前开发门槛失败，不读验证币来筛选赢家。

## 下一固定假说与已生成文件

RG28两完整JSON于temp_strategy/20261005-observed-midshock-recovery/：
familySHAf6899268718a0ffe0d7c90a112a91e7241fd3dcc0d4c613886cc72cb4347020b；
comboSHAdb7577065577596bd97bddc0b31ccfe0960fe6cfa4a813ee06664bf9112549e6。

仅换supplement家庭：最新闭合反向放量shock/反向主动quote多数后，在观察中的当小时出现同向body、反侧极值保留、严格收回shockBODY中点且不追超中点+.35ATR（SHORT相应）。当前累计quote>0且<closed shock quote，原literalQPS90%保持；不等一个完整闭合recovery和下一完整极值才介入。原9、base/order/fullclose/风险成本和所有门槛保留。

这是测试早期价格回收，不是凭旧亏损反转交易、降低.9门槛、挑币或黑名单失败时间；也不据“回收已超过shock open”分组PnL选阈值。形成中quote只代表截至signal的累计观测，不能称为闭合整小时缩量、elapsedrate或确认当前buyer主导；0taker缺陷仍不碰。新形成规则改变可接受集合，必须全路径重撮合，不能用旧228条静态删选估利润。

RG28前端actualexit0：两issue=null/9/四types/shape，限定330portable765entries0exactdup。仅结构，不是Go/Expr/API/真实开仓/收益；exactprojectversion尚未取得，main/protocol/新canonical审计均未准备或启动。下一独立legalOHLC/闭合shock/observedvolume/midpoint oracle和真实Go/Expr；新frozen protocol后AF0/RG27/RG28×四币十二完整原49月run，所有后审计仍必须保留并分类零活动失败。新策略仍研究候选，不放release strategy_templates或数据库。

## 平仓矩阵与技能状态

LONG/SHORT方向对应、wholeRG4保持：outer(-5,5)普通不评估；ordinaryROI5/-5/16/28/-12无信号false；ROI>=16 AND(trendfail OR(momentumfailAND反向活动实体)) true；ROI<=-12 AND(trend OR momentumfail) true；ROI>=28 AND(momentum OR反向impulse) true；(ROI>=5 OR<=-5) ANDclosed4hADX<20AND破闭合反侧极值true；ROI<=-20灾难为唯一无信号确认例外。AutoStop=false，5/5/8x保持。RG28继承完整程序的值对象一致，但新整体runtime仍未核验；详细matrix在research_specs/20261005-rg28-observed-midshock-recovery.json。

按skill-creator/SkillMax仅把成本failed的行级分类补入既有未审批候选父qps-fixture。custom-strategy-cost-diagnosis-20261005 applied1/rejected0/budget4（机械编辑预算，不是行为评分）；quick_validateactual0、全parent+唯一段落精确验证0，trusted:false。候选SKILLSHA2ee260ff759fd4a2d52f6a8e66a07f1e12c280ec36c63dd84a62eb2b3ca7e3c7，父4949dabb...不变，正式SKILL270d20...不变。3新增现实请求及原cost/algebra/QPS集合pending独立baseline/candidate行为评估，没有score/strictwin/promote/用户审批或新委派，不阻断研究也不宣称技能已安装生效。

本轮没有template insert/assignment/activation/order、App、生产/前端/conf修改、仓库_test文件、删除或新委派。所有新策略和失败证据保留；旧不相关dirty edits原样。Goal active，未完成；scope成本1失败已真实terminal，不用它虚假报告整个研究阻塞。

## 产物SHA-256

原件都在/Users/zhz/Library/Caches/go-binance-strategy-research/results/；小协议/总结/规格在项目custom-strategy下。

```text
0ac3a03d2561494ce9ec096b4427a0f6c15f15398d36eb7401668f5cae5b1ea1  20261005-rg27-development4-canonical-repaired-v2-funding-tail-v1.json
0dc369024b24679f4bdc4ce5cd9bccc4ff2c9414c606cd39a9c61cde11a4d7a8  20261005-rg27-accounting-summary.json
4f69faa48225ffaa6f86ba8b5f93d51d6c340a94e44aa0e446479cfeeda2d987  20261005-rg27-expr-checks.json
5622b200a74e3b5395490b3ce47f425f3e99f999ebcc671319a1a8ac9c8c4ce1  20261005-rg27-canonical-actual-pattern-open-signal-audit.json
bd7c1afbd77ef1a645eedb430b6f56518061e32d506b456938c373261b0191bd  20261005-rg27-original-position-close-signal-audit.json
1584b37bf8a3331cd5dc805f6d2350083fa52059968576e60af6dbdf374b1b1e  20261005-rg27-execution-original-cost-fallback-audit.json
11d502158174bfa38d08f88fe3f5aa5ea121bf424bcbbc29a33c0966e60a2943  20261005-rg27-natural-entry-regime-attribution.json
3753fdc40a31b9e56207fbc6e57b3a0c84e2b192bf457402bb4302a34c46d973  20261005-rg27-phase-evidence-summary.json
582e6721f1cbc1796833ff3fe0f73b399e4ad9cb98f1da5522031f9777a7eaa8  20261005-rg27-zero-activity-exit-context.json
```
