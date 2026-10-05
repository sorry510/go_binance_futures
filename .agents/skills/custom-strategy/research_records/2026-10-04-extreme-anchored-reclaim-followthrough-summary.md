# RG20完整阶段总结：频率已过，年度与执行证据仍不合格

## 当前结论

`invalidated`，`release_qualified=false`。已完成AF0/RG19/RG20×BTC/ETH/SOL/XRP的12次原49月完整项目顺序回测，并完成全部后审计。四币频率和累计净收益通过，但四币都至少有一个亏损的完整Sep–Aug年度，BTC/XRP还存在负日历年。因此不发布、不写入数据库、不读取预设未阅验证币。不能用总体正收益或另一45月/4x/ROI-only诊断替代原目标。

窗口UTC2022-09-01inclusive至2026-10-01exclusive，1491天/213周，每币>=0.9次/周至少192笔。8x/outer5与5/完整uniformRG4技术确认关闭/当前cash10%margin/每侧fee0.0005及不利slip5bps/真实funding及原缺mark分钟价格回退保持。49月整体顺序撮合、各年按退出归因，并非独立逐年初始化收益。

## 四币完整结果

|币|笔数|每周频率|毛收益|手续费|资金费PnL|净收益|PF|最大回撤%|均持小时|
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
|BTCUSDT|213|1.000000|486.579512|187.701208|-24.110781|274.767523|1.148359|15.766476|28.184585|
|ETHUSDT|226|1.061033|1191.069800|235.483141|-19.536752|936.049907|1.392357|16.251167|19.934882|
|SOLUSDT|224|1.051643|1278.137310|257.313883|-32.723130|988.100297|1.315006|26.703528|12.409003|
|XRPUSDT|205|0.962441|645.870581|196.091736|-5.212846|444.565999|1.199634|34.422554|11.816260|

|币|2022-09至2023-08|2023-09至2024-08|2024-09至2025-08|2025-09至2026-08|额外2026-09|去最佳5描述净额|
|---|---:|---:|---:|---:|---:|---:|
|BTCUSDT|-13.961141|82.959794|-12.372918|248.802346|-30.660558|-41.938257|
|ETHUSDT|-17.894091|263.563346|269.170545|392.596792|28.613315|359.907378|
|SOLUSDT|-151.963837|619.691586|238.123721|262.092354|20.156473|481.483212|
|XRPUSDT|407.190440|-437.887141|203.491936|235.049091|36.721673|-20.179478|

|币|日历2023|日历2024|日历2025|2026Jan–Sep|
|---|---:|---:|---:|---:|
|BTCUSDT|109.022050|113.201829|-16.987204|140.335074|
|ETHUSDT|194.150767|189.893551|197.898519|486.355984|
|SOLUSDT|171.977608|271.795483|45.263305|464.350369|
|XRPUSDT|222.634472|-42.976764|-232.261590|404.849965|

去最佳5、侧、规则、趋势、年和cap分组均是原有交易描述，不是静态删除后重新撮合收益，不建立独立alpha或可交易反向。BTC和XRP去最佳5描述总净转负，利润集中风险仍明显。

## 实际入场逻辑与自然市场分组

RG20自身868=477补充+391组合内基础；不可用868减独立AF0 429当补充。477补充全部和实际开关证据逐笔配对。补充净BTC−282.115059、ETH+372.885196、SOL+897.870646、XRP−181.961269；BTC两侧毛净均亏，XRP补充毛合计亦负。四币共同稳定盈利的自然weak/strong组不存在；仅SOL strong去最佳5描述仍正，不足跨币泛化。

旧Close锚cap在当前实际快照通过124/拒绝353，新Extreme锚477全通过。旧cap拒绝组：BTC91笔净−357.927091、ETH107笔+257.212725、SOL70笔+821.612888、XRP85笔−124.988345；这不是RG19会交易这些快照或两策略顺序反事实。新cap确实扩大可达，但频率改善伴随部分币更差，不能继续放宽追价或挑币宣布成功。range收缩244/非收缩233同为描述，全477正宽度合格。

## 全核验与真实成本边界

- main81791已实际terminal0；opening93223/closing70528/cost29788本次真实poll全部exit0，不再poll/重启。会计12run/1894全部账目/8共享AF0-RG19完整控制exact/errors0。自然归因与phase脚本实际exit0，所有输入SHA/全部配对核验。
- Expr5486通过/0失败；原200input/199closed，477补充/38160closed-field/+2385四小时/+1431日线/+954ATR/+2862range/+1431累计activity/+477strict，0失败；最低activity0.9000338217046567，最大同侧极值advance0.3493773462287001。原函数parity并非独立指标数学、完整顺序cache/private-selector/API/live/forward证明。
- 868正常关闭/0forced-end，old601/added269/both2/added-only267，真实原Position/ROI/outergate/whole正常程序0失败。普通ROI越外门槛本身仍false，技术确认分支true，ROI<=−20为唯一深灾难例外。原关闭没有被换成ROI-only。
- all12成本1894笔/4525资金费结算/1373原minuteClose mark回退/0零活动填单/0算术失败；自身868/1982结算/667回退/0失败。全部控制和自身错误行保留，当前确为0，不能把其它旧研究失败删掉。最大数量误差2.274e−12、净误差1.706e−13，仅正常浮点。
- 原5bps不利滑点、两侧手续费和实际历史funding均计入，但精确venue结算mark和真实历史order-book容量/滑点证据仍pending。0零活动不等于保证填单。官方旧mark采集12个HTTP200+time/rate exact但mark空的样本只证明样本缺值，不是全覆盖，不以合成值、零funding或无成本替代。

## 保留文件及可审计身份

两完整失败候选保留temp_strategy/20261004-extreme-anchored-reclaim-followthrough/：family SHA83a0099f7c2393fe825e6f13ef4b0c13989db7f6a08fc503ad800a138945e51c，combo SHA12a1a4c924e7c9bed118337ed883d123de42734ad7d59451522b964568eb5f15 / version be0e12c702b6aa9c10d6995f4af3e701721de68bbbc328f353a343b824ed4486。收益前协议SHAeef7d6a673aef0f35b816e3f0933c4719b26a6ddcb154dd6898f592bba7e3b14。

缓存结果根：/Users/zhz/Library/Caches/go-binance-strategy-research/results/

|证据|文件|SHA256|
|---|---|---|
|study|20261004-rg20-development4-canonical-repaired-v2-funding-tail-v1.json|d89c1d6f6c306fdcea20d9e48444f74efc857f391fd486ee0b9c1df58012bbb7|
|accounting|20261004-rg20-accounting-summary.json|34117b2198494360065dc4bca269769057cd24aa636adce2d446f3a5bb65d6b5|
|expr|20261004-rg20-expr-checks.json|771fd2319b21afa024960c9c630a6ecf9115dc74949e788cc41ac8e48cbe802c|
|opening|20261004-rg20-canonical-actual-seed-open-signal-audit.json|e46f497738560f5d1742f1982b88736cfbdf639639dde532ff7d957d1673f936|
|closing|20261004-rg20-original-position-close-signal-audit.json|e7dca0919081216c7813436f7d46ec62311e7ba7da9ed34ed2e1505b22a66bd4|
|costs|20261004-rg20-execution-original-cost-fallback-audit.json|dba0602329a93a92b823694a4a83ebc6cadf18eec8f29ffff47b6d8e83f2bb92|
|natural|20261004-rg20-natural-entry-regime-attribution.json|358aec2a40fcb09ab57e4416adaab413c0ad0055f89b83a6e1bca86967925037|

phase文件20261004-rg20-phase-evidence-summary.json；所有原件保留，不删失败或覆盖旧控制。四数据hash/fundtail54append及7overlap/current main/data复制源与原协议一致。

## 下一步固定诊断与权限

下一条候选尚未冻结或测试。不从净亏推反向或静态删自然分组。先对477实际补充核验闭合信号小时的body方向：LONG Close[1]>Open[1]，SHORT Close[1]<Open[1]，相等不确认。这是没有额外阈值或指标的新价格响应质量假设；扫边收回加current极值越线并不必然意味着闭合信号body方向一致。先查原canonical OHLC与receiver parity及跨币存在性，保存全部行，不读取未阅币或网格挑阈值。若机制真实且独立，再冻结RG21的两侧同一个body维度完整JSON、独立oracle、边界、旧完整程序唯一追加与收益前协议，然后原12run全重撮合；频率可能下降应接受失败，不能改0.9。

程序本地v29 ID114语义与原冻结v29一致。本次最新用户RG18入库已回读本地ID124/唯一名/压缩JSON/原字节exact；本研究无新增DB写，不重复插入，不分配/激活/下单。没有App/生产/前端/conf变化、新仓库测试文件或删除，全部dirty/用户metrics timestamp-max规则保留。core/conf/正式SKILL实际SHA保持。

技能影响：现有custom-strategy指导完整成本范围/原合同/全配对审计，并保留严格price与cap交集的独立验证。custom-strategy-cost-algebra-20261004只是trusted:false待评审窄草案，结构有效不等于行为strictwin，原正式skill不晋级；不创建重复技能或新委派。

研究目标仍未完成，实际goal active。本turn获得完整审计和新机制诊断前协议，属于progress；下次用量恢复前先阶段总结并核实际进程，不能拿旧live描述重复启动。

