# RG32 单一10bps滑点压力：完整回放与审计总结

当前结论：**10bps压力场景 invalidated，release_qualified=false**，因为完整日历2025组合−86.23471611605807未通过既有稳定性guard。原5bps开发场景promising under tested conditions的完整证据保留；两者是同一策略不同RunConfig，不应混合成一个成本口径或以压力结果寻找刚好过线的bp参数。总体仍没有满足真实成本/四年稳定/跨币泛化的可用策略。

## 完成状态与唯一变化

77文件收益前冻结，protocol SHA721f44b1ca775d0c49c2223dda42beb6b705fcf5f90a769f3dd15da3e8869fea。北京时间09:51:41.848实际启动压力主47746，09:58:45实际观察terminal0；09:59:21.829–847实际启动opening8120/closing19414/all12cost17100，10:00:08实际观察全部terminal0；natural/phase 10:00:59实际exit0。所有句柄结束，不poll/restart；完成后再次核验全部77SHA exact。

原harness仅SlippageBps literal5→10，其余逐字节不变；独立新Go副本/build实际0，三Node syntax实际0。生产engine/environment/indicator-cache、data-loader、conf、规则/顺序/九指标不变。MAIN没有diagnostic overlay。engine仍standard_1m/backtest_engine_v7，新的Config/slippage10和独立输出准确识别压力模型，不伪造新的生产runtime或实际盘口。

窗口UTC2022-09-01 inclusive至2026-10-01 exclusive，1491天/213完整周；每币至少171笔/≥.8，不要求每币或每币每年盈利。四独立钱包各1000，current-cash10% margin×8、每侧fee.0005、outer5/5、真实历史funding时点rate及原缺mark分钟Close回退均不变。唯一更严参数是每侧adverse10bps，当前minute-close判定/下一分钟open成交；不是静态扣费或固定立即±5止盈止损。

## 完整场景比较

|指标|原5bps|单一10bps压力|
|---|---:|---:|
|自身交易|871|873|
|固定四钱包模型net|+2889.674070|+1710.473499|
|四完整Sep–Aug年度组合net|+264.320477 / +572.931254 / +737.010045 / +1303.257347|+45.163153 / +407.978628 / +500.552691 / +814.626586|
|完整日历2025组合net|+90.205331|-86.234716|
|逐币完整周频率gate|全部通过|全部通过|
|完整日历2023/24/25组合gate|全部通过|2025失败|
|realized-close-order DD%|7.957187|9.399904|

模型净额相对原5bps下降1179.200571USDT，变化包括成交价/ROI资格/退出时间、后续占仓与复利数量等完整顺序路径，不能当成固定旧账本的滑点扣费。双侧fee率没变，费用绝对额可能因资金与数量变化减少，不代表改了费率。独立钱包累计42.761837%是描述，不是实际账户收益保证。

|币|笔数|每完整周频率|gross|fees|funding|net|PF|引擎DD%|平均持仓小时|
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
|BTCUSDT|214|1.004695|364.909049|181.390782|-22.990389|+160.527878|1.089029|15.733694|27.081308|
|ETHUSDT|227|1.065728|881.300903|219.951388|-18.338473|+643.011041|1.279724|15.807510|19.660793|
|SOLUSDT|225|1.056338|791.037784|224.689971|-29.731716|+536.616098|1.186009|30.591768|12.187407|
|XRPUSDT|207|0.971831|566.724302|191.384579|-5.021240|+370.318483|1.168225|35.254358|11.703945|

873自身=482补充+391在组合base；压力全12run/2220笔、八AF0/RG31比较源/data/engine严格保留，errors=[]。这八个压力对照**不声称**复现原5bps trade ledger/metrics；会计检查仅删去本来不该相等的收益字段比较，其余全账目、快照、rule hash、chrono、fees/funding、source/data identity严格检查。原5bps主的八整对象控制精确复现证据独立保留。

|完整Sep–Aug年|组合net|BTC|ETH|SOL|XRP|
|---|---:|---:|---:|---:|---:|
|2022-09至2023-08|+45.163153|-17.426695|-68.194298|-242.630032|+373.414178|
|2023-09至2024-08|+407.978628|+73.572676|+233.244921|+537.503787|-436.342756|
|2024-09至2025-08|+500.552691|-57.146964|+240.110886|+162.009127|+155.579642|
|2025-09至2026-08|+814.626586|+196.120314|+266.356776|+103.327142|+248.822354|

|日历平仓归因|组合net|BTC|ETH|SOL|XRP|
|---|---:|---:|---:|---:|---:|
|2023|+476.779234|+98.546498|+127.879732|+53.363157|+196.989847|
|2024|+378.795979|+64.352357|+169.062509|+208.108072|-62.726959|
|2025|-86.234716|-28.203636|+159.118784|-8.439193|-208.710671|
|2026 Jan–Sep|+1020.205587|+95.645311|+314.595667|+252.780110|+357.184499|

2022Sep–Dec总-79.072585；额外2026Sep总-57.847560，都在总账。年度/日历/额外Sep只按同一账本exit-time归因，非年初重新初始化。第一完整年组合仅+45.163153，不隐藏其余量风险；单币负年保留但不独立拒绝。去最大贡献ETH后描述net+1067.462458，非重新平衡；组合DD仅realized close-order proxy，不是完整MTM。

|币|LONG笔/net|SHORT笔/net|去最大五笔描述net|
|---|---|---|---:|
|BTCUSDT|126/-6.135378|88/+166.663255|-147.707760|
|ETHUSDT|128/+334.833799|99/+308.177243|+112.945869|
|SOLUSDT|138/+252.401675|87/+284.214422|+106.698063|
|XRPUSDT|126/+346.728576|81/+23.589906|-78.064290|

## 新实际开/关/成本范围

开仓482实际补充/38560closed字段（4h2410、daily1446、ATR964、range2892、current activity1446、strict482），0失败；真实200输入/199闭合种子及截止signal observed分钟量/canonical闭合量不变。QPS90 min.9000338217，extreme cap482/482；old close cap125pass/357reject，contraction246/non236。不是elapsed activity rate或当前buyer确认。

关闭873normal/0forced，old828/added55/both10/added-only45，0失败；形成EMA/RSI/4hDMI与oldnumeric所有parity通过。Actual old comparator是完整pre-strong RG20，45相对RG20，不是立即父RG31。诊断BuildMinuteClose overlay只用于导出，并未进入压力MAIN。原冻结Go/Expr91276/0作为同一规则的预检证据引用，没有复制成新执行次数。

|LONG/SHORT场景|决定（与原5bps同一程序）|
|---|---|
|普通±5 ROI，无技术支持|false|
|−12<ROI≤−5，ADX≥20、strict反侧破线，仅trend或仅momentum失败且旧分支false|false|
|同上trend AND momentum失败|true|
|ROI≤−12、原loss12单项确认|旧分支true，不取消|
|缺strict/恰等极值，所有旧分支false|false|
|正ROI且旧分支false|新增false|
|ROI在(-5,5)|outer不调用|
|原weak/profit16/28/loss12支持信号|原样保留|
|ROI≤−20|true；唯一无信号灾难例外|
|空/错opening hash|同决定，不做hash路由|

|成本范围|交易|funding applications|mark回退|零活动|综合失败|
|---|---:|---:|---:|---:|---:|
|all|2220|4577|1388|0|0|
|own|873|1945|634|0|0|

all/own/control全部failed_rows=[]。独立重建current-cash数量、每侧10bps下一minute fill、双fees和所有实际funding原回退；all最大qty error3.6379788071e−12、net error1.9895196601e−13、fill errors0，原精度不变。Own1945funding/634fallback相对5bps1940/642改变来自占仓路径，不是mark数据补齐或质量提升。

这些只证明压力模型计算/分钟活动自洽，不证明10bps真实盘口成交或venue精确settlement mark。[官方资金费接口文档](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data#get-funding-rate-history)把markPrice作为funding记录的关联价格；本地原分钟Close回退不能冒充该精确来源。原2022四例HTTP200空mark和2023八例HTTP451获取失败分开保留，不再次调用受限端点或换路由，不零funding或造mark。

## 自然归因：全部482补充保留

|币|补充笔|补充gross/fees/funding/net|弱ADX笔/net|强ADX笔/net|去五大盈利描述net|
|---|---:|---|---|---|---:|
|BTCUSDT|123|-239.216149 / 104.021733 / -7.439608 / -350.677489|60/-116.781773|63/-233.895716|-616.162276|
|ETHUSDT|134|353.856558 / 126.600020 / -5.669769 / +221.586769|79/-90.504919|55/+312.091688|-222.856585|
|SOLUSDT|100|782.160417 / 97.324933 / -4.151235 / +680.684250|49/+41.364506|51/+639.319743|+291.782164|
|XRPUSDT|125|-79.347317 / 114.342431 / +0.836562 / -192.853186|80/-328.374631|45/+135.521446|-531.017420|

|币/ADX组|四完整年补充net依次|
|---|---|
|BTCUSDT/weak|-78.284226 / +3.617436 / -62.151975 / -16.444222|
|BTCUSDT/strong|+64.696114 / -14.073475 / -132.825409 / -135.425913|
|ETHUSDT/weak|-29.133459 / +39.807637 / +53.962277 / -196.701015|
|ETHUSDT/strong|-34.026310 / +165.659280 / -17.129389 / +197.588106|
|SOLUSDT/weak|+29.329829 / -28.914575 / +84.531041 / -72.576485|
|SOLUSDT/strong|-18.396391 / +499.496881 / +62.129229 / +96.090024|
|XRPUSDT/weak|+7.208370 / -86.196438 / -88.304525 / -167.449618|
|XRPUSDT/strong|+90.685842 / -181.095667 / +93.606010 / +132.325260|

没有跨币/年共同稳定自然组，BTC/XRP补充负、ETH更依赖大单，SOL更强，但不能因此删除币/侧/年/added-only止损或静态回加PnL。全部侧/收缩/old cap/exit属性原JSON已保存。入场regime不是关闭时regime，不能互相推断。

## 完整结果身份与下一动作

结果根 `/Users/zhz/Library/Caches/go-binance-strategy-research/results/`：

```text
501ae31ef8da0302a96d4bb68202f5710516ebbe5ecbcd392ca6d7b3be13bfae  20261006-rg32-slippage10bps-development4.json
1537f606dc063c0e3ff8ead6e8c3fa62ded1dd9e735afd5f1da57cccc12abc9f  20261006-rg32-slippage10bps-accounting-summary.json
e7be471628c69429ff1d3540f825d37d57578925615e747efe9a70f84d9e4dae  20261006-rg32-expr-checks.json
8aa90a02bd799ddfe850a434b1a84a8643ed9df4aa5662e57e2667063a940460  20261006-rg32-slippage10bps-canonical-actual-seed-open-signal-audit.json
14e1fa9df26bc35580db6b4b4fb570efbd4ddace46f20e7dc6e2b4431d38130c  20261006-rg32-slippage10bps-original-position-close-signal-audit.json
467a50112aa5eb305f28e3fe1c73c80205029fecccff3bf3a011e7d66fea0776  20261006-rg32-slippage10bps-execution-original-cost-fallback-audit.json
c8d59158f0ad0389ac42d8886a3283a4c944035e069c8779d632eab64a928aec  20261006-rg32-slippage10bps-natural-entry-regime-attribution.json
c64e4e59fb1597ad92eb3102c7efabedcfab99cb2a849c164b12eec917798917  20261006-rg32-slippage10bps-phase-evidence-summary.json
```

原RG32 comboSHA51b182991491ef122942bc5f29a33f4a28e2439234da03da7359d31584112638/raw versionb38ad611ee214d7cad803337a9287a2eb03a2522af8d121a2e6dbeee8b94f8e7不变，仍在temp_strategy，未覆盖旧JSON/失败/原5bps证据。压力是新模型场景，不额外造一份参数相同的“新策略”。

下一可执行动作：只读配对原5bps和10bps全正常关闭环境，按完整旧分支可观测证据划分loss/profit资格及弱/强结构确认，核实成本敏感来自哪些可重复逻辑；全49月/四币/两侧一起描述，不只盯2025、不删负组或当新PnL。只在证据指向明确结构问题后冻结一个逻辑维度的新完整候选，先temp_strategy/独立矩阵/协议，再同数据同5与10成本场景顺序回放；不再搜索滑点参数或放宽稳定性guard。

AAVE/ATOM/ETC/LINK收益仍未读取，精确mark/历史盘口未证明；缺数据是缺证据，不能从压力测试或公共分钟量中补造。当前goal active，正式可用策略目标未达成。现在可复核的是完整两个成本场景和失败定位，不是可启用策略。

没有App、生产/frontend/conf、数据库模板/分配/启用/下单、仓库_test.go/globalmemory或新委派变更；用户dirty和所有失败策略保留。SkillMax仅现有待审批草案窄修订r2/trusted:false，formal未晋级，行为比较pending。

