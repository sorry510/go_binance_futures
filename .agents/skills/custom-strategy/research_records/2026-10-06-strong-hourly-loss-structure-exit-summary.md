# RG31 强趋势亏损侧结构退出：完整开发回测总结

## 当前结论

**invalidated；release_qualified=false。** 固定四币组合净收益+2315.190660 USDT，四个完整Sep–Aug年度均正，每币完整213周频率均≥0.8，但预声明的完整日历2025组合净收益为**-9.083993825184308 USDT**，稳定性验收失败。不能把接近零的亏损四舍五入为通过，不因单币或单币年度亏损额外拒绝。

精确资金费结算mark、历史盘口容量/真实滑点和未阅跨币泛化仍未完成。原模型会计、信号及分钟活动通过不是完整真实成本或可发布证明；没有合格策略，不写入本轮候选、不分配/启用/下单。

本阶段为实际progress：上一goal轮完成RG30全审计及总结、冻结并完整运行RG31主/全部后审计；本轮核实全部37冻结文件、七个结果和phase身份，并保存本总结。最近单独RG18本地核验只是授权持久化的只读复查，不是本研究的放行或新版本入库授权。

## 已实际结束的工作与证据范围

- 主89784：北京时间2026-10-06 01:27:05实际返回exit0。AF0/RG30/RG31×BTC/ETH/SOL/XRP，12完整49月run；非partial检查点。
- 开2197、关19813、all12成本53288：01:29:50实际全部exit0。natural与phase：01:30:35实际exit0。旧句柄全部终止，不重poll/重启。
- 全会计12run/2276笔，八AF0/RG30共享control的整份snapshot/ledger/metrics/data/config/date精确复现，errors[]。原始交易和全部控制保留。
- 本轮只读限定pgrep未发现已结束主/审计进程。没有委派、App、研究DB写、生产/前端/conf修改、新仓库_test.go或globalmemory修改。

## 冻结候选、唯一变化与成本契约

父RG30与完整pre-strong RG20谱系全部已被观察，不是未阅验证。仅将RG30新增strong_structure_reversal的ROI资格从ROI≥5或≤−5改为ROI≤−5：closed4hADX≥20、严格破closed1反侧hourly极值、原trend_fail OR momentum_fail保持共同确认。盈利侧不再因为这条低gate附加分支提前离场。四个entry整对象/顺序、九个指标参数、全部旧weak/profit16/28/loss12及唯一ROI≤−20无信号灾难分支保持；无OpenStrategyHash依赖/MarketCondition/forming taker字段。所有原件保存在temp_strategy。

- family：`temp_strategy/20261006-strong-hourly-loss-structure-exit/00-strong-hourly-loss-structure-exit-family.json`，SHA`6f625a8a9a953b649f165e253986e3cbc0715c746e0bd0d6ea31843c196ebffe`。
- combo：`temp_strategy/20261006-strong-hourly-loss-structure-exit/01-v29c-strong-hourly-loss-structure-exit.json`，SHA`9074f901841467acd32fe99bab042b82fe82bae90d0cefa81effd45e4d57adb0`；raw version`acb71d48c855252ce5c6dc9432347586b52aee5325ff2caf8bde8c7438156643`。
- 预协议：`2026-10-06-strong-hourly-loss-structure-exit-protocol.md`，SHA`01314bc035edc90fd0c65f86a7d5bda31dce430cc51fdc49a03c0be856a5163b`。收益前spec保持原历史“未运行”状态，不事后改写冻结证据。
- UTC2022-09-01 inclusive至2026-10-01 exclusive，1491天/213周，每币至少171笔。每币1000等初始资金独立钱包；当前可用cash10%保证金×8，outer profit/loss5/5，每侧fee0.0005、adverse5bps、实际资金费率/时点、原缺mark分钟Close回退；standard_1m/v7下一分钟open成交。没有改费用、调日期、删币、筛侧/年或ROI网格。

## 各币完整49月结果

费用正值为成本，funding为有符号现金变化；net=gross−fees+funding。每币亏损允许且保留；引擎回撤不等于下面组合已实现平仓顺序代理。

|币|笔数|每完整周|gross|双侧费用|funding|net|净PF|引擎回撤%|均持仓小时|
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
|BTCUSDT|232|1.089202|447.201007|204.650672|-17.215442|225.334894|1.1346|17.5392|18.186|
|ETHUSDT|239|1.122066|859.541791|222.298272|-15.821503|621.422016|1.3220|14.9854|14.656|
|SOLUSDT|233|1.093897|1025.525255|253.806837|-28.930977|742.787441|1.2560|25.4764|10.437|
|XRPUSDT|211|0.990610|950.683154|218.885061|-6.151783|725.646309|1.3356|29.1493|9.697|

固定四独立等初始钱包net **+2315.190660** /4000，收益57.8798%；排除最大正贡献SOL的描述净额仍+1572.403220，不是重新配资撮合。组合已实现close-order DD代理10.3172%，不是完整MTM风险。

相对立即父RG30（+1976.0547964852399）净额增加+339.135863664476，但仍低于RG20（+2643.483725683202）；这只是同一已观察开发集的顺序撮合比较，不是未阅获利证明。

## 年度稳定性：完整年与日历年分开

下面年度均是同一完整顺序回测的**平仓时间cohort归因**，不是每年重新初始化的收益。额外2026-09单独列出，不凑成第五完整年。

|币|2022-09..2023-08|2023-09..2024-08|2024-09..2025-08|2025-09..2026-08|额外2026-09|
|---|---:|---:|---:|---:|---:|
|BTCUSDT|34.436291|83.011671|-79.499321|197.659042|-10.272789|
|ETHUSDT|-43.800932|105.917129|273.899099|250.336592|35.070129|
|SOLUSDT|-145.666523|579.812771|141.048918|184.221840|-16.629566|
|XRPUSDT|381.684218|-356.001046|287.438081|368.658165|43.866891|
|固定组合|226.653054|412.740525|622.886777|1000.875639|52.034665|

|币|日历2023|日历2024|日历2025|2026Jan–Sep|
|---|---:|---:|---:|---:|
|BTCUSDT|115.450591|44.692179|-51.675739|145.598888|
|ETHUSDT|157.179389|10.070479|257.512130|304.269130|
|SOLUSDT|135.396881|281.445738|-55.298797|330.633103|
|XRPUSDT|207.434848|23.924775|-159.621588|535.916504|
|固定组合|615.461708|360.133170|-9.083994|1316.417624|

四完整年正、完整日历2023/24正；完整2025负，guard失败。BTC/ETH/SOL/XRP的个别负年均披露，但不是“逐币年盈利”硬门槛。2026不足完整年，仅描述。

## 方向、补充与自然趋势组

|币|侧|笔数|net|
|---|---|---:|---:|
|BTCUSDT|LONG|139|-43.131500|
|BTCUSDT|SHORT|93|268.466394|
|ETHUSDT|LONG|136|390.798404|
|ETHUSDT|SHORT|103|230.623612|
|SOLUSDT|LONG|145|447.919502|
|SOLUSDT|SHORT|88|294.867939|
|XRPUSDT|LONG|128|509.535666|
|XRPUSDT|SHORT|83|216.110643|

四币剔除各自最佳五笔的描述net依次为BTCUSDT -102.934048；ETHUSDT 190.232068；SOLUSDT 271.697738；XRPUSDT 222.015367。BTC仍集中，不据此删币或反向。

RG31真实499补充+同一组合416基础入口=915，不得用独立AF0或父版本的基础/补充交易差分替代实际ledger。

|币|实际补充笔数|gross|fees|funding|net|剔除最佳五笔net|
|---|---:|---:|---:|---:|---:|---:|
|BTCUSDT|131|-170.604577|115.316615|-4.254214|-290.175406|-569.741042|
|ETHUSDT|139|153.020773|126.785412|-4.403928|21.831433|-302.347249|
|SOLUSDT|104|774.885620|110.384408|-3.670030|660.831182|258.641857|
|XRPUSDT|125|60.791921|127.382522|0.663234|-65.927368|-438.748478|

自然weak/strong按**入场时closed4hADX<20/≥20**固定描述；退出时ADX可以变化，weak入口也可以出现added-only strong退出。这不是新组过滤。各组完整年亏损仍保留，没有跨四币共同稳定组。

|币|入场组|笔数|net|完整年1|完整年2|完整年3|完整年4|
|---|---|---:|---:|---:|---:|---:|---:|
|BTCUSDT|weak|61|-96.760604|-77.696472|4.541718|-58.361061|-6.574475|
|BTCUSDT|strong|70|-193.414802|72.477729|10.311729|-187.381399|-79.692224|
|ETHUSDT|weak|79|19.734400|-15.741152|50.831729|55.980677|-114.851850|
|ETHUSDT|strong|60|2.097033|-65.575842|-8.401711|-32.218201|108.292787|
|SOLUSDT|weak|49|94.437884|36.504507|-20.104777|104.211864|-61.901765|
|SOLUSDT|strong|55|566.393299|-30.363890|496.192035|37.131953|63.433201|
|XRPUSDT|weak|79|-283.702955|24.306798|-85.732337|-63.945946|-168.971057|
|XRPUSDT|strong|46|217.775587|74.445766|-145.916382|140.289137|148.957065|

完整contraction/旧close-cap/方向/added-only归因保存在原natural JSON；这些分组不是反事实净额。特别是新分支只在ROI≤−5有资格，added-only亏损并不能证明删除该退出会赚钱；删止损后的资金、占用、后续入场都必须重新顺序撮合。

## 全信号与成本后审计

开仓：全499实际补充/39920 closed字段；4h2495、daily1497、ATR998、range2994、currentactivity1497、strict499，原200输入/199closedseed一致，0失败。literalQPS累计量最小ratio0.9000338217046567；极值cap499/499，最大归一化advance0.3493773462287001。旧closed-close-cap通过135/拒绝364、contracted257/非242仅归因，不是新规则。

关闭：915正常/0强制期末，old627/new365/both77/added-only288/0失败；全FormingIndicatorParity与OldPass==OldNumeric为真。**old comparator是完整pre-strong RG20关闭，不是立即父RG30**；288是相对RG20的added-only数，Go预检另外独立证明RG31决定是RG30子集。诊断overlay只fresh BuildMinuteClose；MAIN无overlay。使用实际Position/ROI/outergate/canonical closed extrema与199闭合种子+退出前已观察分钟独立复算forming EMA/RSI/4hEMA-DMI；同原函数fresh parity不是全部cache/private/live/forward或独立数学证明。

|成本范围|笔数|实际funding应用|原mark回退|零活动fill|核算失败|
|---|---:|---:|---:|---:|---:|
|all12控制＋候选|2276|4013|1206|0|0|
|RG31自身|915|1529|494|0|0|

所有控制、自身、all失败列表均[]；qty最大误差all3.183231456205249e−12/own2.0463630789890885e−12，net最大误差3.410605131648481e−13，下一分钟fill误差0，原tol未改。干净分钟activity与固定5bps只证明原模型执行，不证明历史盘口容量。funding缺mark回退仍494次自身；精确venue成本未通过，绝不回加费用或称已完全真实成本。

## 关闭决定矩阵与预检边界

Go/Expr98527已actualexit0，91248 passed/0，独立whole RG20/父RG30/现RG31三模式；frontend两issue=null、9指标、四type、限定336portable691close无whitespace-exact重复。API实际exit7，127.0.0.1:3333无法连接且无rule响应；真实Go fallback，不自动启动服务/App。只读本地v29 ID114语义exact/0writes。合成矩阵不是利润证明。

|LONG/SHORT共用场景|决定|
|---|---|
|普通ROI±5，没有任何确认|false|
|亏损gate≤−5＋closedADX≥20＋strict反侧hourly破线＋trend OR momentum失效|新增true|
|同强确认但+5≤ROI<16且全部旧分支false|新增false|
|缺严格破线或trend/momentum确认，全部旧分支false|false|
|反侧极值恰相等|新增false，旧完整确认分支独立保留|
|ROI在(-5,5)|outer不调用|
|所有旧weak/profit16/28/loss12确认分支|保持原逻辑|
|ROI≤−20|true，唯一无信号灾难例外|
|空/错OpenStrategyHash|相同，uniform无绑定|

## 下一单一假设：RG32（此处先声明，尚未生成/看新收益）

仅将RG31新增强趋势亏损侧结构退出的最后确认由**trend_fail OR momentum_fail改为trend_fail AND momentum_fail**；ROI≤−5、closedADX≥20、strict反侧closed1破线及全部旧RG20关闭/四入口/9参数保持exact。检验较高周期强趋势是否不应只因为局部小时动量疲软就提前认定失效，要求跨周期方向与动量共同失败。两项仍都来自价格，不能声称统计独立alpha。

这不是根据2025−9.08调整阈值或挑止损亏损单；没有ROI/lookback网格、币/侧/年筛选、静态删单。更严格确认可能加深亏损、增加持仓成本、减少频率；原loss12/weak结构及灾难20保护全部保留。必须先完整两JSON、单因子继承/可达性/普通gate与极值边界、frontend和收益前协议，再AF0/RG31/RG32×固定开发四币12完整顺序回测与全部后审计。

所有当前数值/信号/成本条件通过之前不读AAVE/ATOM/ETC/LINK收益；之后同冻结49月及风险/参数完整跨币验证，不能将开发反复试验或临时API成功替代泛化。

## 权威结果与身份

所有原JSON位于`/Users/zhz/Library/Caches/go-binance-strategy-research/results/`；本轮实际重算全部hash与37收益前freeze一致。

|结果|SHA-256|
|---|---|
|20261006-rg31-development4-canonical-repaired-v2-funding-tail-v1.json|`21d555bd3d9933c24d129d2997ba2f75aff3a600d5060eb6b08e8f78b25f1915`|
|20261006-rg31-accounting-summary.json|`6a4dd9ce350c8dd12e6468ae8913212e5b18cc5f2e75274fea0cbba13356a4df`|
|20261006-rg31-expr-checks.json|`088ff9709704c6ea8485e269c640cd3bd22d9e40227cf09f58c80aa3e9a19788`|
|20261006-rg31-canonical-actual-seed-open-signal-audit.json|`d96533102c2e8fdf1438d8c4f3d78ac4ebe5eb51a970793c723beed9fa3511e2`|
|20261006-rg31-original-position-close-signal-audit.json|`5b9601c773dc029a0557dcb1d0d73698ec24d4dc202b7a0306ff0e3e5ce26f16`|
|20261006-rg31-execution-original-cost-fallback-audit.json|`73967e86994316390b9849a63a0a416bdb266a7aebd224cf262f495f0ba710b4`|
|20261006-rg31-natural-entry-regime-attribution.json|`936006b6c3935fce0c2e1f05ceeeb78ecc7ae4f1b3d720cdf22664aa9e939bd5`|
|20261006-rg31-phase-evidence-summary.json|`f8ed65511e8facd2f83e399efb7bb5dc1f78adb8ce8c7e8b5945e363417b92d4`|

正式SKILL SHA270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd、conf SHA7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa不变。current-goal/成本诊断SkillMax草案仍trusted:false待真实行为评估/strict-win/用户批准，没有promote。

