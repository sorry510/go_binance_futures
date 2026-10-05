# RG10：持续反向主动压力的闭合极值拒绝（首次收益前冻结）

2026-10-04北京时间07:22:12，真实Go/Expr 3482/0之后、RG10任何完整收益前冻结。当前goal active；本次继续第一条已先总结RG8/RG9 24完整run/4724执行账目及联合失败，实际旧主/审计无进程，不重复旧回测。

## 证据、假设与与旧规则的区别

91877 actual terminal exit0：新只读canonical诊断完整关联RG7 525/RG9 493共1018个真实补充入口及原审计，data/ledger/signal身份完全一致。第一后续闭合小时RG7 115/525、RG9 103/493失去此前脱离位置，实际这些交易净归因分别-789.590/-778.147；仍在外的组+60.379/+77.231。第二小时分别168/525和153/493失去位置；第一小时主动quote与自身价格反应分歧73/525、69/493。所有后续小时无缺失，但第一小时已有26/26笔实际退出；后续市场观察并不等于持仓内信息。两版本和两个horizon复用交易，不是2036个独立样本。LostEscape不一定落在完整区间内部。

文件results/20261004-rg10-prior-failure-path-attribution.json SHA2fd20a347cdba07ba3f1882c8ee5e44f4c69ae12ab53c4b2a2a93b28acb590ca。后续数据只作描述性诊断，不能进入原入场表达式，也不能从亏损组推断可交易反向alpha或静态删组收益。它仅支持提出另一个价量机制：在新入场自己的已闭合小时，持续主动压力推动极值突破但最终价格反向收回；需完整新撮合验证。

已完整只读相关v36、v41 adaptive、v64 daily-guard exprfix、v65 union、v165配置，没有读取它们的收益。v41/v64/v65多头主要用同向主动买入确认回收；RG10使用主动卖出多数而闭合价格向上，多头之前8小时主动卖出及价格下行一致，空头镜像。v36读forming回收并为ROI-only关闭，v165为强趋势且ROI-only退出；这些旧关闭不借用。不同程序不证明新alpha或吸收因果；主动quote/价格背离只是代理，不是钱包/挂单身份。

## 完整候选及唯一机制替换

- temp_strategy/20261004-opposing-pressure-extreme-rejection/00-opposing-pressure-extreme-rejection-family.json：SHAc7bdd15bdf692f0138c6c98f95f7fc60a0ec88d2e8a7cf08115f3bb12ff1e672，versionb7a63f8c9393b8e52c8b4c994a0e56dc62e33ea21ad7b9c600f717ae33a33e35。
- 同目录01-v29c-opposing-pressure-extreme-rejection.json：SHAbdf5485f178c2e909993ebc476f9774afbda7198e4312e72a14fc786f3c0fd9a，versionbf556e3117b99573ac753d1e401cb49b1ca48459b28a41d1e1cd3d75f6aeef88。

族4/组合6启用规则。从RG7只替换两弱趋势补充全程序为一个完整反向压力极值拒绝机制，不同时改退出/风险或按币挑条件。原9技术配置、AF0基础开仓整对象、全部统一RG4关闭整对象保持。新补充名rg10_opposing_pressure_extreme_rejection_long/short，不携带RG9 RSI55/45条件或RG8强趋势loss。

LONG：闭合4h ADX<20，过去8个闭合小时[2:10]有效quote加权主动卖出多数、Close[2]<Open[9]；闭合1h Low[1]严格跌破此前8小时最低Low至少0.10 ATR，又Close[1]严格回到该边界上方0.10 ATR，body上涨0.15..2 ATR。该小时quote严格大于过去8小时均值，主动卖出仍多数而价格上涨。SHORT镜像为先上涨/主动买入、上方极值失败、价格下跌而主动买入仍多数。闭合日线ADX>=20且DI反向的排除保持；形成价格[0]仅限制相对闭合Close[1]的-0.15..+0.35 ATR保持并要求观察成交quote>0。持续压力和价格方向是机制的一部分，未单独搜阈值。

20、8小时、0.10/0.15/0.35/2 ATR和50%均继承先前明示尺度，不做收益参数网格。新补充不要求4h EMA方向（区间拒绝不是追随4h EMA），但原配置仍由基础/关闭复用，不新增指标/周期。强4h趋势由原AF0判断，弱区由新拒绝判断，不要求所有状态都成交。有意Data[0]保留，不新增forming taker0或opening hash依赖。

完整只读temp_strategy+strategy_templates共412旧portable/905同侧完整入口，无精确去空白重复，没有读取对应收益；非语义新颖或alpha证明。

## 合成失败证据与纠正

29341 actual terminal exit1：3314 passed/168 failed，结果20261004-rg10-expr-checks.json SHAfef43b703bf632a79d1289461d671200f151584f0fb448acc19897e9447e03a1。原因在仓库外checker：旧prior-flow预期仍LONG买多数/SHORT卖多数；weighted_opposition又意外保持与support相同fixture。168包含同一案例的程序和本地有序计数/首规则重复，不是168市场样本。策略JSON和原运行时不修改。

另存rg10_expr_checks_fixturefix.go只纠正上述预期与相反加权fixture，7970 actual terminal exit0：3482/0，结果20261004-rg10-expr-checks-fixturefix.json SHAa0cdba223ff0395f0d31f8179c156a9ad28146054f048c4a1fb887e52f5b36a2。原失败helper/输出保留。严格sweep/reclaim等值、实际新低/新高、反向主动quote合法性/严格多数、前8小时加权压力及价格反向、ADX20/日线DI边界、forming和无关EMA/ATR2/QPS独立、body/live边界/多空镜像/基础可达/全部旧关闭矩阵通过。整对象断言配置/基础/完整关闭保持，两个补充与事前声明全程序一致。非private/cached/API/live/forward/独立指标数学或盈利证明。

## 完整撮合与资格

AF0/RG7/RG10×BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT，共12完整run，各初始1000；UTC2022-09-01含至2026-10-01不含，1491天/213周/每币至少192笔。每币>=0.9次/周、真实成本净正、四年稳定、跨币泛化联合门槛不放宽。四完整Sep–Aug年、额外2026-09、日历2023/24/25/2026 Jan–Sep均报告；45月exit cohort仅归因，不是独立初始化收益。开发49月已观察，非时间holdout；AAVE/ATOM/ETC/LINK收益仍未观察、资格待核，开发失败不读验证收益。

原backtest_engine_v7/standard_1m、观察分钟close/下分钟Open、当前现金10%保证金/8倍、每侧fee0.0005/slip5bps、真实funding率/时点及原缺mark分钟Close回退、外部profit/loss5/5与AutoStopOrder=false不变。外部5/5只是调用gate；普通ROI-only不平，ROI<=-20为唯一灾难例外。正常盈利/亏损必须独立市场确认；原RG4弱闭合ADX<20+小时价格反向破上一闭合Low/High+ROI±5外正常分支完整保持，统一全部入口/不按hash路由。

真实funding mark完整性仍缺：RG9全4280应用/1092回退、自身1578/405；新回测另单列应用/回退。RG8的固定12原始HTTP空mark例仅是样例证据，不补零/造mark/改成本或声称精确资格完成。

来源public-canonical-repaired-v2-funding-tail-v1，全public-archive/public-archive及verified-archive20261003-v2；四币数据hash、原replay/data helper SHA、suffix54/overlap7 manifest SHA与RG9冻结协议同。每次全source/CRC/hash/prefix/tail重核，不改递推种子/默认warmup/容差。

## 预声明核验、保护和恢复

主输出results/20261004-rg10-development4-canonical-repaired-v2-funding-tail-v1.json。主后核8个AF0/RG7旧控制逐笔/metrics/年度/source完整重现、全会计/方向/年度/族/集中度；实际补充入口canonical闭合OHLC/quote/taker、过去8小时加权压力与价格相反、严格极值拒绝/body/live、日线反向排除、原函数实际200input/199closed ATR/ADX种子和完整Expr逐笔核对，不把无关EMA/ATR2字段变成入口门槛。正常关闭仍原Position/gross mark-price-denominator ROI/outergate及完整统一RG4审计；forced-end单列；全原复利quantity/fills/fees/真实funding/缺mark回退和分钟活动另核。不把任何静态分组/未来小时归因当可实现收益。

新只读ARM62288 actual terminal exit0，截止北京时间07:04:21.904/23.659/25.402，三库模板17/17/17、结果222/219/8，v29摘要相同/go_binance parsed全导出与RG8一致。元数据不是449条新forward内容分析。配置/engine/environment/实际indicator_cache.go/SKILL SHA实际不变，无App/DB写入/分配/启用/下单/生产/前端/conf/app.conf/新仓库测试文件。所有临时Go/大输出/overlay仓库外，virtual诊断源不得实际创建；完整候选失败留temp_strategy。

正式技能1.0.8/trusted:false与原pending候选不变。阶段总结新增独立评审副本仅结构valid、四行为案例pending/aggregate null，没有委派/strictwin/promote；不影响安全研究继续。实际启动保存真实handle，只poll同handle，不因timeout重启；下次第一条先阶段总结再核真实goal/进程，paused停止不自行恢复，阶段完成不自动complete/paused/blocked。
