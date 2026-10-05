# RG20：极值锚定追价限制的扫边收回确认 — 收益前协议

## 基础与冻结声明

2026-10-04本协议保存时，RG20主回测尚未启动，其历史收益尚未读取。RG19主/开/关/成本已全部实际terminal0，12完整49月run/1548账目/8共享AF0-RG18控制exact/errors0；四币频率全失败、ETH/SOL/XRP存在完整亏损年，invalidated。不能把净额全正或原模型成本算术通过称发布资格。

本轮程序再次直连活动本地127.0.0.1:3306/go_bn_test，repeatable-read只读事务，无App或应用初始化：v29 ID114与原冻结ARM portable rawSHAb511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0完全语义相同。technology语义SHA0ae3b681d78d49884c11165fc70684ca8db52c1db28a4bd6eb77d554f06bfa00、strategy语义SHA3d43aea89d4dc56b29aeb3890bdf92028ab0bac54777a9d9238df46cda4f00e2；results/20261004-rg20-local-v29-snapshot.json SHA58949504e51fd8d0216b63e718cefb6448a7534a8267360fa02337f0ce8d4ffc。AF0是有意研究v29C变体，不冒称rawDB字节原样。

之前看过RG19不同4x/45月/ROI-only风险诊断开发结果，本轮不是全局未见市场/假设；以已完成原合同RG19为前一对照，参数不据RG20未知收益变化。其文件/六额外验证币不修改或混入本轮。本线程预设AAVE/ATOM/ETC/LINK收益仍未读。

## 唯一逻辑维度与代数依据

只改变RG19两补充入口的追价cap锚点：
- LONG由currentClose<=closedClose[1]+0.35ATR[1]改为currentClose<=closedHigh[1]+0.35ATR[1]。
- SHORT由currentClose>=closedClose[1]-0.35ATR[1]改为currentClose>=closedLow[1]-0.35ATR[1]。
同一0.35ATR系数，仍strict current LONG>High[1]/SHORT<Low[1]，相等拒绝。旧cap+strictcross必然要求同侧信号wick<0.35ATR；较长合法wick因几何关系无法通过，不是技术指标缺失或已知拒绝机会的收益证明。新cap允许相对Close更大位移，可能追价亏损增加，不能宣称一定改善。

原lower live-hold边界仍保存但被canonical严格极值越线蕴含；不算第二独立确认。不改原闭合严格扫边/收回、previous几何首次性、两range正宽度、闭合合法反向aggressor quote多数及放量、closed4h weak OR同向EMA/DI strong、closed日线反向排除、recovery、90%当前累计QPS、基础整对象/顺序/原9指标/周期/whole uniform RG4关闭。保留故意[0]，不用forming taker[0]或MarketCondition。QPS门槛是累计活动，不是elapsed-rate、方向flow或next分钟执行容量。

## 完整候选和实际收益前校验

两完整JSON保存在temp_strategy/20261004-extreme-anchored-reclaim-followthrough/：
- 00-extreme-anchored-reclaim-followthrough-family.json，4规则，SHA83a0099f7c2393fe825e6f13ef4b0c13989db7f6a08fc503ad800a138945e51c，versionf3d0db0b47b8066a09f882b6c55322a2320c3b40d7008d304d83a0b379c463c5。
- 01-v29c-extreme-anchored-reclaim-followthrough.json，6规则，SHA12a1a4c924e7c9bed118337ed883d123de42734ad7d59451522b964568eb5f15，versionbe0e12c702b6aa9c10d6995f4af3e701721de68bbbc328f353a343b824ed4486。
单维replacement/冻结base hash规格在research_specs/20261004-rg20-extreme-anchored-reclaim-followthrough.json。没有新指标或hash绑定退出。

真实Go/Expr56485已terminal0：5486通过/0失败，SHA771fd2319b21afa024960c9c630a6ecf9115dc74949e788cc41ac8e48cbe802c。完整previous程序单处cap替换、独立numeric两侧oracle、合法wick×strictadvance矩阵含0/边界上下/长wick、新旧整程序对比、widewick不得覆盖activity/flow/daily/strong-opposition失败，原strictprice/QPS/range/close matrix不变。只证明合成可达及边界，不是全历史cache/private selector/live/forward/盈利。
helper生成第一次大写版本说明匹配失败在写出helper前发生，已修正精确大小写替换，不是策略失败，不改两候选或读其收益。

真实frontend TypeScript validator隔离VM：两issue=null/9enabled/四type/shape通过，SHA10b8c2e17fc72cd91af3ec4fdb1dc595189d0be900de489d7d34ac976e8f6be6；无App/UI/build/API。限定314portable717enabled entry/0精确重复，SHA1de92fa8b01564e1d8bbcf2dd274f0d04a974a9bcec4ddddae61cebf17921275；排除自身/research/diagnostic/audit/>128KiB，不称全语义/alpha新颖。

新opening build5609已实际terminal0，sourceSHA817ed68036d722ebdcacf760742f2825d2a11fb89575968a94937bb5917c01b8/binaryb06f898db75ae3de85ba7dee9d75b588880c2a588fb2aaff785557ef41193e82。所有RG20 actual supplement signal分钟/原新鲜receiver/canonical200input199closed字段和完整量价/strict/cap将独立核验。ExtremeAnchoredCapPass作必要判定，旧CloseCap及wick只作实际快照描述，不能当旧顺序反事实PnL。generic关闭/成本binary不变。三Node会计/自然归因/汇总helper node --check exit0。

## 原完整关闭矩阵与风险

两侧whole uniform RG4原profit16/loss12/profit28方向/动量/活跃体确认分支、新增outer正负5与closed4h ADX<20及当前反向越闭合小时极值的组合均exact；唯一无技术确认例外为更深ROI<=-20灾难止损。

|两侧场景|原决定|
|---|---|
|普通+5/-5仅ROI越线、技术确认全无|false|
|+5/-5且closed4h ADX<20、LONG严格破Low[1]/SHORT严格破High[1]|true|
|恰等于极值且其它原确认分支false|false|
|原16/28利润或-12损失分支的对应技术确认成立|true|
|未达灾难线、对应确认全无|false|
|ROI<=-20灾难例外|true|
|ROI在(-5,+5)|外层不调用|

已包含于5486实际Expr矩阵；主后还须全actual正常关闭Position/ROI/outer gate/whole program核验，end_of_data单独截尾。

AF0/RG19/RG20组合×BTC/ETH/SOL/XRP，UTC2022-09-01inclusive到2026-10-01exclusive，12完整49月run/1491天/213周。每币>=0.9次/周至少192笔；四Sep–Aug完整年、extraSep、日历2023/24/25和2026Jan–Sep及侧/基础-补充/最佳5集中度同时报告。不扫描新阈值或改变验收/日期/币/原成本。

原engine_v7 standard_1m，observed close→next open，cash1000起、当前availablecash10%margin×8，每侧fee0.0005/不利slip5bps、真实funding与原缺mark分钟Close回退、outer5/5和AutoStop=false保持。精确venue结算mark/历史深度/实际slip仍pending，算术通过不是完整真实执行证据；不补合成资金费或制造标记价。

## 数据、完整撮合和后核验

canonical数据四币SHA依次c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457 / 7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371 / 64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818 / 8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3；fundtail54追加/7overlap/manifest78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547不变。正式source/hash/minute repair版本保持。

主输出results/20261004-rg20-development4-canonical-repaired-v2-funding-tail-v1.json自带逐run checkpoint；真实terminal前不得因timeout/partial输出重启。主12全部后完整accounting/8AF0-RG19共享控制exact，path仅固定workspace realpath等价，source仅cache_hit获取观察差异允许，其余snapshot/交易 identity全比较。

全部RG20补充opening几何/QPS/strict/extremecap、全部normalclosing真实原Position/ROI/gate/whole code、all12 fills/currentcashqty/fee/funding/fallback及所有entry-exit分钟活动核验；all与focus失败按原行分别记录，旧控制错误不能冒称全audit过关或静态删除。原functionsparity不是独立指标数学或live/API/forward证明。

收益前预设实际closed4h ADX20 weak/strong、recentWidth<olderWidth收缩/非收缩，以及旧CloseCap同一快照pass/reject归因；包含全部补充并逐笔配对actual开关，仅描述年度/侧/退出/集中度，不称旧策略会交易/静态筛选后新PnL或新alpha。四开发币完整联合门槛过后才取得未阅AAVE/ATOM/ETC/LINK资格，不先消耗验证。

## 权限、当前身份与恢复要求

21:26实核engineSHAae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad/environmentb98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5/indicatorcache1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0/conf7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa/正式SKILL270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd保持。main/data helper仍SHA21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8 / a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63。
没有App/newDB writes/assignment/enabling/orders/production/frontend/conf变化、仓库测试文件或材料删除。旧RG18仅按前一独立用户授权已插入本地124，此次研究没有新入库。所有dirty/metrics timestamp-max/失败资料/SkillMax pending草案保留，不新委派或晋级。实际goal active，无达标发布策略，不自complete/paused/blocked。下次限制恢复前先阶段总结、核真实goal与句柄，不把旧状态当授权或live。

