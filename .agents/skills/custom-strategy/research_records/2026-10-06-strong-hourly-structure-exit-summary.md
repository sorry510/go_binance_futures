# RG30：强趋势小时结构确认退出 — 完整阶段总结

## 当前结论

2026-10-06北京时间01:03:47已实际观察主6548、opening7312、closing24015、all12cost53110全部terminal exit0；会计、natural及phase也实际exit0。全部12个49月run和真实开/关/原成本审计完成，旧句柄不可重复poll或restart。当前goal仍active，没有合格可发布策略。

RG30 verdict **invalidated**：每币频率、固定四钱包总净额、四个完整Sep–Aug组合年及排除最大贡献币检查均通过，但日历2025组合净−0.6560128038937307，收益前冻结guard失败。不是因某个币或某个币年度亏损拒绝，也不把小亏四舍五入成零或更换年界。精确venue结算mark、历史盘口容量/真实滑点和未阅跨币验证仍未完成；原模型核验通过不是“真实成本全部验证”或盈利保证。

现在可用的是RG30完整研究JSON及可复核的本轮证据，**不是可发布/启用的策略**。没有新增数据库写入、模板分配、交易启用、订单、App操作、生产/前端/conf修改或仓库测试文件；保留所有用户已有dirty和旧失败。上次RG18独立入库已完成，研究不重复插入。

## 唯一维度与固定约束

父级RG20来自已经多轮检查的开发集选择；这不是独立验证。四个完整entry对象/代码/名称/顺序和九指标全exact。补充名称故意仍rg20_，本轮按新raw snapshot version定位，不能拿旧477笔作为新开仓证据。

仅uniform close追加：outer ROI≥5或≤−5 AND closed4hADX≥20 AND当前价格严格破closed1反侧极值 AND已有trend_fail OR momentum_fail。旧RG4完整止盈16/28、止损12、weakADX<20小时结构及−20唯一无技术确认灾难例外保持。所有base/supplement都使用同一close，不绑定OpenStrategyHash；空/错hash无影响。保留用户故意forming[0]，不读forming taker[0]或MarketCondition。

UTC2022-09-01 inclusive到2026-10-01 exclusive，共1491天/213周、最低171笔/币。固定BTC/ETH/SOL/XRP各1000起始资本，当前available cash10%保证金×8复利、外部5/5资格、每侧fee0.0005及不利slip5bps、真实funding rate/时点和原缺mark→funding分钟Close回退。原standard_1m/backtest_engine_v7已观察分钟close→下一分钟open，MAIN无overlay、不改撮合。四年和日历表是原全程交易的exit-time归因，不是独立初始化年度收益。

## 完整组合与稳定性

|币|交易数|次/完整周|毛收益|净收益|PF|单币最大回撤%|平均持仓小时|
|---|---:|---:|---:|---:|---:|---:|---:|
|BTCUSDT|238|1.117371|+335.812823|+120.944717|1.076410|16.702841|15.352101|
|ETHUSDT|245|1.150235|+937.509181|+687.110080|1.358152|14.698362|13.036463|
|SOLUSDT|236|1.107981|+724.219897|+455.268136|1.165180|26.342998|10.044562|
|XRPUSDT|213|1.000000|+938.685219|+712.731864|1.336882|27.410671|8.887872|

固定四独立等初始资本钱包总净+1976.054796，对4000初始资本49.401370%；去掉最大正贡献XRP后描述净+1263.322933，不是重新配仓/反事实组合。已实现平仓顺序DD proxy 10.734631%，不是完整MTM组合回撤。

|币|2022-09..2023-08|2023-09..2024-08|2024-09..2025-08|2025-09..2026-08|额外2026-09|
|---|---:|---:|---:|---:|---:|
|BTCUSDT|+9.901676|+65.150529|-71.891723|+165.221513|-47.437279|
|ETHUSDT|-12.413698|+71.625079|+331.254544|+319.366371|-22.722216|
|SOLUSDT|-155.601279|+514.578955|+150.733150|-40.556614|-13.886075|
|XRPUSDT|+359.820097|-324.489739|+250.423162|+383.439745|+43.538599|
|固定四钱包合计|+201.706795|+326.864823|+660.519134|+827.471015|-40.506971|

|币|日历2023|日历2024|日历2025|2026-01..09|
|---|---:|---:|---:|---:|
|BTCUSDT|+58.436582|+14.005358|-73.294472|+119.288928|
|ETHUSDT|+119.940328|+42.055787|+318.607117|+274.786424|
|SOLUSDT|+121.605200|+218.323116|-75.516758|+140.246061|
|XRPUSDT|+191.404114|+12.217805|-170.451900|+547.798099|
|固定四钱包合计|+491.386224|+286.602065|-0.656013|+1082.119511|

原RG20全868笔、组合net+2643.483726；RG30全932笔、频率更高但总净下降667.428929，四完整年均下降，日历2025由−6.086969变成−0.656013仍失败。完整顺序撮合、机会占用和复利变化都保留，不能只用337 added-only归因解释为因果收益、也不能把日历改善当作下一微调的目标。

|币|LONG笔数/净收益|SHORT笔数/净收益|去最佳5笔描述剩余净额|
|---|---:|---:|---:|
|BTCUSDT|145 / -116.861278|93 / +237.805994|-178.378123|
|ETHUSDT|142 / +305.994972|103 / +381.115108|+228.306745|
|SOLUSDT|147 / +190.844246|89 / +264.423891|+3.072296|
|XRPUSDT|129 / +501.509684|84 / +211.222180|+215.509118|

单币和年度亏损完整披露，不强求逐币盈利；BTC LONG−116.861278、SOL有两个负完整年、XRP2023-09年负，不能静态删币/删侧让结果过关。BTC去最佳5笔后负，少数大赢家集中风险仍在。

## 补充机制、方向与自然趋势

全自身932笔=506补充+426组合内base；不能用932减独立AF0的429笔推算增量。全部506补充逐笔开仓核验配对，全部932正常关闭配对，0期末强平。

|币|补充笔数|补充毛收益|补充手续费|补充资金费净额|补充净收益|去最佳5笔剩余净额|
|---|---:|---:|---:|---:|---:|---:|
|BTCUSDT|133|-266.066293|+112.639584|-3.122990|-381.828867|-596.686122|
|ETHUSDT|142|+224.429328|+133.137544|-4.403564|+86.888220|-241.197132|
|SOLUSDT|104|+642.503420|+103.242725|-3.279038|+535.981657|+145.535378|
|XRPUSDT|127|+12.696197|+128.409419|-0.461393|-116.174615|-483.005440|

BTC/XRP补充净负；ETH补充正额去最佳5笔后负，SOL补充较强但入场组年度仍有损失。不能由静态分组回推出可交易反向或过滤收益。

closed4hADX20自然分界按收益前协议固定，入场weak/strong与退出strong判断时间不同；弱组入场仍可到强组退出，不能混用标签。下表年份顺序对应四完整Sep–Aug年，括号交易数，全部保留：

|币/入场自然组|笔数|净收益|LONG笔数/净收益|SHORT笔数/净收益|四完整年净归因|
|---|---:|---:|---:|---:|---|
|BTCUSDT/weak|61|-136.458048|28 / -75.580007|33 / -60.878041|-76.403191 (23笔) / +4.619508 (7笔) / -55.466406 (11笔) / -8.976977 (16笔)|
|BTCUSDT/strong|72|-245.370818|37 / -257.212615|35 / +11.841797|+16.125215 (20笔) / +2.156972 (11笔) / -177.965793 (22笔) / -77.334443 (18笔)|
|ETHUSDT/weak|79|-14.056868|41 / -30.857790|38 / +16.800922|+0.833614 (23笔) / +50.925940 (24笔) / +56.379209 (9笔) / -121.107503 (19笔)|
|ETHUSDT/strong|63|+100.945088|31 / +74.326454|32 / +26.618634|-52.961641 (23笔) / -15.565903 (12笔) / -2.490312 (18笔) / +171.962945 (10笔)|
|SOLUSDT/weak|49|+87.751414|26 / +99.951298|23 / -12.199885|+36.504507 (9笔) / -19.289568 (13笔) / +98.947998 (8笔) / -58.245281 (15笔)|
|SOLUSDT/strong|55|+448.230243|29 / +207.559915|26 / +240.670329|-41.374140 (15笔) / +436.002748 (18笔) / +34.901107 (7笔) / +18.700529 (15笔)|
|XRPUSDT/weak|79|-294.740950|43 / -119.622598|36 / -175.118351|+23.873682 (20笔) / -85.910598 (12笔) / -77.061093 (19笔) / -166.202902 (26笔)|
|XRPUSDT/strong|48|+178.566335|24 / +26.467810|24 / +152.098525|+42.938943 (13笔) / -133.323815 (10笔) / +118.815135 (14笔) / +150.136071 (11笔)|

这些表都是原组合交易归因，不是family独立回测。旧Close-anchor cap及收缩/非收缩更多完整归因在natural JSON保存，没有用净额删除分组或扫描阈值。新strong追加337独有正常退出中有原base也有补充，早退出可能减损也可能截断趋势赢家；已观察利润下降不能被解释为删掉这些单后就会更赚钱。

## 全部核验与成本边界

- 36冻结SHA和protocol238e9c45在主terminal后全exact。主1e0e0669；全会计12run/2229账目/八AF0-RG20 complete snapshot/ledger/metrics/source/config/data控制exact/errors[]。仅path realpath身份归一和source.cache_hit观测差异，其他全字段比较。
- Go/Expr56688/0；真实Go结构与独立typed数值wholeold/new关闭模型、单ULP严格极值/ADX/ROI/趋势动量/impulse/空错hash及旧entry/quote/QPS/cap完整矩阵。前端真实隔离validator两issue=null、九enabled及四types、限定334portable/687关闭程序0whitespace-exact重复。开仓与父相同属有意；不是全局语义新颖性或盈利证明。
- API尝试actualexit7，127.0.0.1:3333无listener/没有rule响应，Go/Expr按skill fallback，不自动开启服务或App。API未验证；固定mock即使成功也不证明真实仓位/历史。
- 本地v29 snapshot actualterminal0，go_bn_test ID114两JSON语义与冻结源一致，0writes；conf保持7461e8e，正式skill270d20e8不变。
- Opening四runs/506补充/40480闭合字段/2530四小时/1518日线/1012ATR/3036范围/1518当前activity/506strict，0失败，实际input200/closed199；最小累计activity ratio0.9000338217046567，cap全部506，最大相对同侧极值位移0.3493773462287001ATR。QPS除固定3599.999不是 elapsed速率，不把current quote/QPS当做buyer方向证据。
- Closing四runs/932normal/0forced，old595/added417/both80/added-only337，0失败。新诊断overlay仅导出原fresh BuildMinuteClose；全部original Position/mark-price-denominatorROI/outergate/closed1反侧极值/closedADX1及forming1h EMA/RSI0-1、4h EMA20/50/DMI、实际hour quote和ATR独立从canonical seed加已经观察分钟重算。所有FormingIndicatorParity和wholeold NumericPass均一致；同原指标函数fresh parity不是独立数学、full sequential cache、private/live/forward证明。
- All12cost：2229笔/4466实际资金费应用/1379原分钟mark回退/0零活动/0失败；own：932/1391/444/0/0。cash从独立prior nets复建，当前cash10%×8qty/下一分钟原不利fills/双侧fee/funding入出边界均核。maxqty error3.183231456205249e−12，maxnet1.8474111129762605e−13，容差未改、失败未删。Own/control failed rows均[]。
- 精确结算mark不齐时仍原Close fallback，必须明确原模型与venue精确资金费用之间差距。0打印缺陷未出现只代表该minute活动检查通过，不证明容量/订单簿/真实不利滑点。AAVE/ATOM/ETC/LINK仍未读取，不以开发成绩冒充跨币泛化。

## 平仓决策矩阵

|两侧场景|完整决定|
|---|---|
|ROI位于(-5,5)|outer不评估|
|普通profit/loss gate被跨越，无任何技术确认|false|
|strong closedADX≥20、严格破closed1反侧极值、trend_fail或momentum_fail、ROI跨±5|true|
|仅strong小时破线，缺趋势/动量确认|false，除已满足旧分支或灾难|
|有趋势/动量确认但未严格破线，ROI未达旧门槛|false|
|价格等于前小时极值，未满足旧门槛|false|
|weak closedADX<20严格反侧结构破线、ROI跨±5|旧分支true|
|旧ROI16/28确认止盈或−12确认止损|旧分支保持|
|ROI≤−20|true，唯一无技术确认紧急例外|
|空/错openinghash|same decision，未绑定|

## 保留文件与身份

temp_strategy/20261006-strong-hourly-structure-exit/：

- family00 SHA b6616be9111de06fd6665e1f09a0ec1776e4b96c55096a3e30b06f0b01c0aec6，version05de34ff1ac8407bdd548935f10464df7042b11e46df8151eafa0a854bf3cf09。
- combo01 SHA dbae210e400ba86d36f3a2b72ede38be8aba0777e4cd1dfeb8b5518ea21820d9，versionba1bb2000e297962525a766505bcec31f4ed9903c554a235d44cd4d05d3a1b05。
- 收益前protocol2026-10-06-strong-hourly-structure-exit-protocol.md SHA238e9c45cdc6d40a72db5db685a9d47aa894990c7e14df34d949136fd644bbaf；research_specs/20261006-rg30-strong-hourly-structure-exit.json保留冻结preflight字样，不事后改写收益前声明。

缓存根/Users/zhz/Library/Caches/go-binance-strategy-research/results/：

|证据文件|SHA256|
|---|---|
|20261006-rg30-development4-canonical-repaired-v2-funding-tail-v1.json|1e0e06693a190a20fde834978feb26cf9ab04cadf10f6a5fbb147f943aad7c48|
|20261006-rg30-accounting-summary.json|eeabf9072f4f918a96878b2b6b0e1f12be7b8f48a9352e8b8138bfbe10ecfb5d|
|20261006-rg30-expr-checks.json|d5e3f8fd7ee40c80ee2dbe3283ec86044ad2ce5f309533e374c1c26d29c226d4|
|20261006-rg30-canonical-actual-seed-open-signal-audit.json|fdc3b5d20f373f71a5296ebd358a6b15e169371e6a1d12f875545c369656392e|
|20261006-rg30-original-position-close-signal-audit.json|0335766897ed7c655b9470a487c70f7adfb6877478962edafcc537807e722c17|
|20261006-rg30-execution-original-cost-fallback-audit.json|097604ae1117258786e6cc7619ba9546bc06805d1a41d77316cfd64ed5276034|
|20261006-rg30-natural-entry-regime-attribution.json|32a7c1adde6494a0b515fd4b9c6e070917f78469c7cef3ff695aa49ff80a4656|
|20261006-rg30-phase-evidence-summary.json|39a8e6d5bb49d7929c96fd653c7f431ead09b0e16be1fe12af3da1cfe383760a|

Data BTCc521ace1/ETH7c4bb4e7/SOL64be83c4/XRP8aa4acab；实际fundtail54追加/7exact overlap/manifest78758a90身份全部见protocol，无新数据政策或回测模型修补。失败研究仍保留，不搬到发布目录。

## 下一单维度假设：只对亏损侧追加strong结构确认

预声明RG31：保留RG30所有entry、原weak branch、已有profit16/28/loss12/−20emergency和所有risk/cost/date/gates，只把新增strong_structure_reversal的ROI资格从“≥5或≤−5”改为“≤−5”。普通盈利侧仍依赖完整旧信号确认，不在低profit gate仅因小时回撤追加新exit；保留新强趋势损失确认用于缩短失败setup暴露。这检验“早确认损失”和“过早兑现赢家”应否共用普通门槛，而不是把2025−0.656填成正数或ROI数字网格。

尚未生成/运行或读取RG31收益。该假设仍可能加深回吐、降低机会频率、减少净额；不据静态RG30强组或单笔收益推断改后结果。必须先完整两JSON、独立profit/loss差异关闭矩阵、真实frontend和收益前协议，再AF0/RG30/RG31同次12完整顺序run及八control/所有actual开关/成本。旧策略和全部证据保留，开发失败不提前读未阅币。

