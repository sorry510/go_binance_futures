# RG18：扫边收回后的实时极值确认 — 完整阶段总结

## 用户询问阶段成果后的最新状态

2026-10-04北京时间20:05:56实核goal已为usageLimited，限定进程查询无残留，不沿用下方14:59的历史active字样继续研究或自行恢复。RG18结果不变。RG19只在temp_strategy/20261004-range-valid-reclaim-extreme-followthrough/保存了两完整配置，尚未校验、保存收益前协议或启动回测，也没有RG19运行句柄；不能当成可用策略或已完成的新实验。下次用户恢复授权且实际goal允许时先给本阶段总结，再核配置/进程和准备状态接续。

## 当前结论

2026-10-04北京时间14:59:54再次实核：goal active，RG18主回测及全部开仓、平仓、成本核验均已观察到实际 terminal exit0，限定进程查询无残留。RG18 **invalidated，尚无合格可发布策略**；不能因四币组合总净额为正或审计算术通过称为有效盈利策略。

本阶段 RG17、RG18 各完成 AF0/前一组合/新组合×四币的12个完整49月run，共24个run（包含重复共享控制，不是24个独立新候选）。RG17所有结果及其旧控制零活动失败原件保持。RG18保存了两个完整JSON、收益前协议、合成/前端/实际信号/会计/成本/自然归因全部证据，无App操作、数据库写入、分配、启用、下单、生产或前端修改，没有修改conf/app.conf或添加仓库测试文件。

## 唯一变化和冻结约束

在RG17两个补充开仓完整程序末尾增加：LONG当前已观察小时Close[0]严格高于闭合High[1]；SHORT严格低于闭合Low[1]。相等拒绝。保留90%当前累计QPS门槛、闭合扫边收回、前次未收回、相邻四小时范围收缩、合法闭合反向主动quote放量、4h弱趋势或同向强趋势、日线强反向排除及ATR幅度/追价限制。

原AF0开仓整对象和顺序、9指标、完整统一RG4平仓整对象、8倍杠杆、外部5/5资格、AutoStop=false与成本不变。严格极值确认蕴含原负向0.15ATR live_hold下限，不把冗余条件计为额外独立确认；正向0.35ATR追价上限仍有效。保留用户原[0]，未引入forming taker[0]或MarketCondition。

冻结UTC2022-09-01 inclusive..2026-10-01 exclusive，共49月、1491天、213周。每币≥0.9次/周至少192笔。cash1000起、当前available cash的10%保证金×8；原engine_v7已观察分钟close→下一分钟open，每侧fee0.0005、不利slip5bps，实际资金费率及原缺mark时分钟Close回退。不是固定正负5%立即止盈止损，也不是独立初始化年度收益。

## 完整组合表现

以下均为完整组合实际顺序回测，金额USDT；频率按全213周计算。

|币|交易数|次/周|毛收益|净收益|PF|最大回撤%|平均持仓小时|
|---|---:|---:|---:|---:|---:|---:|---:|
|BTC|126|0.592|744.249|601.478|1.489|12.029|33.785|
|ETH|128|0.601|619.040|490.311|1.350|15.747|19.413|
|SOL|151|0.709|289.284|144.306|1.079|25.725|13.920|
|XRP|117|0.549|716.421|597.366|1.397|26.610|15.231|

**四币频率全部失败**。自身522笔=106补充+416组合内基础交易；不能用522减独立AF0的429笔推算新增，因为共同持仓、资金和机会占用改变了组合中的基础交易。

完整Sep–Aug四年净归因及额外2026年9月：

|币|2022-09..2023-08|2023-09..2024-08|2024-09..2025-08|2025-09..2026-08|额外2026-09|
|---|---:|---:|---:|---:|---:|
|BTC|0.400|58.202|184.870|439.757|-81.750|
|ETH|9.938|31.511|197.187|263.360|-11.685|
|SOL|-199.461|153.290|67.793|129.997|-7.313|
|XRP|276.583|-292.490|280.069|302.655|30.548|

BTC/ETH四组正并非联合达标，BTC首年仅约0.4、额外9月负，仍有成本资格和频率缺口；SOL/XRP存在完整亏损年度。不能改变年界隐去亏损。

|币|日历2023|日历2024|日历2025|2026-01..09|
|---|---:|---:|---:|---:|
|BTC|167.298|72.215|180.524|246.415|
|ETH|142.002|-37.071|174.600|305.979|
|SOL|8.566|-89.020|-27.965|225.347|
|XRP|155.760|99.322|-156.254|479.992|

2022年9–12月单列，不当完整年；45个月日历/上述四年度均为原交易退出时点的描述性归因，不是重新初始化年度回测。组合扣除最佳5笔的描述性剩余净额BTC278.529/ETH113.784/SOL−203.585/XRP147.932，只用于集中度判断，不称新策略收益。

## 补充机制和自然趋势归因

|币|补充笔数|多/空笔数|补充毛收益|补充净收益|去最佳5笔剩余净额|
|---|---:|---:|---:|---:|---:|
|BTC|30|16/14|-11.297|-43.521|-246.055|
|ETH|25|11/14|70.100|48.945|-107.973|
|SOL|20|10/10|48.555|30.917|-126.841|
|XRP|31|20/11|-53.842|-82.385|-282.520|

这些是实际组合里补充交易的归因，未单独完整回测family；BTC/XRP补充毛净都负，ETH/SOL正净集中于少数大单，四币去最佳5笔均负。不得删除最佳交易/方向/组后当作新的现金顺序收益，也不从亏损推断反向策略盈利。

按收益前冻结的closed4h ADX20弱/强自然边界，全部106补充都与真实开/关核验配对，没有扫描分组阈值：

|币|弱笔数/净归因|强笔数/净归因|弱去最佳5|强去最佳5|
|---|---:|---:|---:|---:|
|BTC|14 / −92.845|16 / 49.324|-93.484|-153.210|
|ETH|16 / 30.460|9 / 18.485|-33.027|-69.866|
|SOL|9 / 1.747|11 / 29.170|-28.747|-98.455|
|XRP|18 / −6.698|13 / −75.687|-136.220|-191.190|

弱/强都没有共同跨币稳定盈利组，不能静态删弱组或换关闭来声称盈利。强ETH部分完整年没有补充交易，也不是年年有效的证据。

## 完整核验、退出和成本边界

- 主99929：14:50:33实际exit0；12完整run、1700交易账目、8个AF0/RG17完整共享控制exact、会计errors0。只规范化realpath等价路径，其余快照/交易字段不排除。
- Go/Expr28838：14:39:13实际exit0，4566通过/0失败。完整旧程序+唯一严格极值追加、独立numeric oracle、相等/上下、当前活动 conjunction、闭合几何/首次性/ADX/daily及原全关闭矩阵；合成检查不是历史、私人evaluator、live、forward或盈利证明。
- 前端真实TypeScript隔离VM两JSON issue=null，9enabled、四type、形状通过；限定309旧portable703enabled入口0精确重复，排除自身/research/diagnostic/audit/>128KiB。不是UI/build/API或全语义/alpha新颖证据。
- 开57814：14:53:36实际exit0；106补充、8480闭合字段+530四小时+318日线+212ATR+636范围+318当前累计活动+106严格极值，原200input/199closed，全部0失败；最小活动比0.9000472889994341。实际quote/QPS一致，但当时的累计活动不保证未来下一分钟容量。
- 关29873：14:53:36实际exit0；522正常、0强制期末；old464/added59/both1/added-only58，0失败，真实Position/ROI/外部门槛和完整统一RG4逐笔重算。
- 成本3221：14:53:36实际exit0；all1700/4073资金费应用/1158缺精确mark回退/0零活动/0失败。自身522/1307资金费/363回退/0失败；最大qty误差1.819e−12、最大净误差1.848e−13，原容差未改。

全关闭矩阵保持：ROI≥16且trend_fail或(momentum_fail且activebodyimpulse)；ROI≤−12且trend_fail或momentum_fail；ROI≥28且momentum_fail或impulse；统一新增(ROI≥5或≤−5)且闭合4h ADX<20、当前价严格越过前小时反向极值。唯一无需技术确认的紧急例外ROI≤−20。普通ROI达到门槛但技术确认不足仍false，整体受原外部资格控制，非hash路由。

RG18原scope文字有通用“not an all-study passed-audit report”，实际成本字段all_passed=true/failed0；这里明确**全部原模型核验0失败**，但不是完整真实成本或发布通过，保留不可变JSON不改写。真实资金费率不等于精确venue结算mark齐全；精确mark、历史订单簿容量与真实滑点仍pending。AAVE/ATOM/ETC/LINK收益未读，当前失败开发候选不进入验证币。

## 完整文件和可复核身份

候选目录temp_strategy/20261004-contracted-reclaim-extreme-followthrough/：

- family SHA a6eea19dad0758c8a94d774d45356df7273dec8a055ff90cbe2f855b175f52b1，version2704e2e4569650f63d2d00eaa9cf73b22222a40cbfe469acc4ad16c711d39abf。
- combo SHA06807efede7175fd511f5edd68cc49e7ea2fbdc8e269d1535e5e0d392ebebef6，version454647690d272a21ea396a044004e7aa845ae444e505b95e5d9380f512bb3c8a。

缓存结果根/Users/zhz/Library/Caches/go-binance-strategy-research/results/，全部前缀20261004-rg18-：

|文件后缀|SHA256|
|---|---|
|development4-canonical-repaired-v2-funding-tail-v1.json|d5955655a7fbc8dc45a48a0e27a6b47f47f034b92531ddd1bb9893d4c4c1d216|
|accounting-summary.json|9cefce33c949502ae80a50a13c42a55caee93fb41b34f441457bc0fc7f18b732|
|expr-checks.json|eb7fd13c122181614709bbb2020647b7cd58a9951d1b498499abfe7f0c9bc00f|
|canonical-actual-seed-open-signal-audit.json|037fc832aeff3ac33951f4cc42ed69253c482bc5ec6fb40c5fc926f08fb3c202|
|original-position-close-signal-audit.json|77f054a26ae113e86751808aa8e5b2ce08cb53d8e9aae7be077d37ad29c4b122|
|execution-original-cost-fallback-audit.json|9995ddbab2889ed031dd1883ab081a81236ea6c8932699b0216233a5b691c777|
|natural-entry-regime-attribution.json|b4051c2b160dc62941b161f093b7ea72d601ffd16e2cea00e384c7bb0e769730|
|phase-evidence-summary.json|9b772ac541616f2ced052939b5ce3921b628616424744d3b3b912bf4f3f069e3|
|frontend-contract-checks.json|47bf9a0095a5739582345135931c5db3db4f6de34ecba045632f1d2859760c87|
|portable-entry-identity-scan.json|40f20803674ed85bed4ebc9262828163b8253f127c59251fe0ee2628638e1640|

本轮沿用14:15 ARM只读元数据/v29快照，不声称重新统计三库所有forward记录。三库模板17/17/17、结果222/219/8，v29语义一致/rawSHA b511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0。AF0是有意研究v29C变体，不是原DB字节不变策略。

原canonical/funding tail和源身份见收益前protocol；14:59实核confSHA7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa、正式SKILL SHA270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd不变。保留所有dirty和用户metrics timestamp-max规则。

## 下轮单机制假设和恢复要求

下一步检验**已确认扫边收回是否仍必须有此前范围收缩**：只把补充入口的recent_width<older_width条件移除，保留recent_high>recent_low及older_width>0正宽度合法性，以及RG18其余整程序、实时严格极值、90%活动、AF0基础、完整关闭、成本、频率/四年/跨币门槛。

这是继承自突破模型的过滤条件消融假设，不是放宽验收门槛或宣称扩张区间可盈利。RG19尚未生成、冻结或读取收益。须先完整JSON、独立numeric oracle/范围等值及严格极值/活动矩阵、真实前端合同与收益前协议，再AF0/RG18/新组合×四币完整顺序撮合，重新核全部真实开/关、成本、共享控制和自然趋势。新增更多交易若亏损同样淘汰，不从静态分组收益生成反事实。

SkillMax成本范围副本trusted:false、9行为评估pending，结构有效不等于严格胜出；未score、promote或新委派，正式skill及原三个pending副本未替换。记录2026-10-04-cost-audit-scope-skill-pending.md。当前goal实际active，阶段结束不是完整目标，不自行complete/pause/block。下一次用量限制后有恢复授权时，第一条先阶段总结、实核goal/进程/文件，再继续；全部旧句柄已terminal，不重poll或重启RG18。
