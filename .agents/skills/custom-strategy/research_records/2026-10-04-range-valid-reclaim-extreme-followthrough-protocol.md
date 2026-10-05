# RG19：取消相对收缩限制的扫边收回极值确认 — 原合同回测前协议

## 本地基础、收益知情范围与唯一变更

2026-10-04北京时间20:46保存本协议时，原8倍/49个月/完整信号确认关闭合同的RG19主回测尚未启动，其收益尚未读取。上一个goal阶段取得实际进展：RG18全部完成但invalidated，RG19两完整JSON已经保存；本轮已核真实goal active，Expr88475实际terminal0，限定进程查找只有查询自身，无原合同RG19主残留。原RG18仅依用户单独授权入库本地ID124，不能把该授权扩展为新候选入库。

本轮程序直连conf/app.conf活动本地127.0.0.1:3306/go_bn_test，无App/应用初始化，以repeatable-read只读事务核验v29 ID114：
- 名称：日线DMI方向确认+Funding非拥挤4h趋势加速1h有效突破双向 v29。
- technology语义SHA0ae3b681d78d49884c11165fc70684ca8db52c1db28a4bd6eb77d554f06bfa00。
- strategy语义SHA3d43aea89d4dc56b29aeb3890bdf92028ab0bac54777a9d9238df46cda4f00e2。
- 与原ARM冻结v29完全语义相同，原portable rawSHAb511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0；本地一致快照results/20261004-rg19-local-v29-snapshot.json SHA778fad214e4b8e62161d40ef921b4eb13779821527e1c49e12a19a6b09828178。
AF0仍是有意研究修改的v29C+观察累计活跃对照，不冒称未经修改的rawDB v29。

必须披露：工作树另有research/range-valid-reclaim-extreme-followthrough的4倍、45个月、ROI-only关闭及不同频率/验证门槛诊断，其相关开发收益此前已被看到。它不满足原风险/时间/信号确认关闭合同，不能代替本轮证据，也不能把本轮假设称为全局从未看过收益。本轮补齐已固定RG19在原合同下的完整测试，不据另一合同收益再改参数；该工作树及结果保留。其六个额外验证币不等于本线程四个预设未阅币。

RG18净额四币均正仍四币频率全失败，且完整年/日历年存在亏损；补充去最佳5全负。此次唯一维度是去掉两侧补充入口的recent_width < older_width比较，保留recent_high > recent_low、older_width > 0。程序变量从contraction改名range_valid；不把非收缩组的静态归因当新增PnL。

保留：严格closed[1]扫边并收回、closed[2]先前几何首次抑制、合法closed反向aggressor quote多数与量高于非负前8小时均值、closed4h弱势OR同侧EMA/DI强势、closed日线反向排除、recovery/live/no-chase边界、current price严格越closed[1]High/Low、观察当前累计QPS至少前8closed[1..8]均值90%。该累计量不是elapsed-rate、方向性flow或下一分钟容量。保留故意的[0]，不用forming taker[0]。

## 完整候选与收益前检查

temp_strategy/20261004-range-valid-reclaim-extreme-followthrough/：
- 00-range-valid-reclaim-extreme-followthrough-family.json，4规则，SHA0943023388c6b9c5b813fdfef4ba0f3cb55b42b87d79480215fee9587cfc2f66，version4e20ef722c3fb61dbd347499f1c6c7325d39c643cbf1c1e7fcfa585224987782。
- 01-v29c-range-valid-reclaim-extreme-followthrough.json，6规则，SHA99ccf758dc78e33596487f81268150918371bc8396301cb88f8b69b8a393b1ce，version7b3157b2dbc861867601057919739be19240325b50403d0afc82c66ed2851cd5。
原9指标及周期、AF0两个入口整对象及顺序、whole uniform RG4两侧关闭整对象相同；无新指标、MarketCondition或hash绑定关闭。

实际Go/Expr88475已terminal0：4862passed/0failed，结果SHAa52e06a0d0be15f3fecc735053cc51e1293f77237c2cfce4114426541517f45c。检验完整旧program唯一替换、独立numeric两侧oracle、recent/older width负/零/正与等于/更大边界、严格极值等号及90%活动联合、原闭合价量/ADX/日线/基础可达和完整关闭矩阵。不是历史完整cache/private selector/live/forward/盈利证明。
真实frontend技术指标validator隔离VM两文件issue=null、9enabled/四type/shape通过，结果SHA6294d0e8e5376feac6591e50b3f92b4c972d05a23f56c2e63016f21df010f8ac；无App/UI/build/API。限定311旧portable/709enabled entry/0精确重复，结果SHA0e51f5f9b32ce485b969f7af46dd0400dfdb40a02b15f2ad4ce58912b7cf7646；排除自身/research/diagnostic/audit/>128KiB，非全语义或alpha新颖。

新opening helper已build terminal0，sourceSHAe1022c1c06339b603da56b3aa8a298e6760d3ad5b7b15f58a452ea5d225579ea，binaryd629713581b80ac9b4a3e4a2439323f600fd90d6d6599b7460c37ad2f2b683fd。独立canonical审计以RangeValid入场，ContractionPass只作描述；保留所有价量、活动、strict price、真实200input199closed及原函数指标parity。旧RG13关闭binary6247af2b6c2473779616b880d4cfa1cafaaf2d3bc87ad6fcd0c6ee401c71cb78和成本binaryada03ce7f3b86c2196f4804442516e85d6b95add30cc2ea70391d558be68c24b不变。三个新会计/自然归因/范围精确report helper node --check均exit0，收益尚未读。

## 原完整关闭决定矩阵

不修改whole uniform RG4：普通profit16/loss12/profit28分支与原方向/动量/活跃体确认组合，新增outer正负5与closed4h ADX<20及当前反向越closed小时极值组合；唯一无确认例外为更深ROI<=-20灾难止损。

| LONG/SHORT场景 | 原判定 |
| --- | --- |
| 普通外部+5或-5，仅ROI越线，无有效确认 | false |
| +5或-5且closed4h ADX<20，LONG价严格破Low[1]/SHORT价严格破High[1] | true |
| 上述价格恰等于极值，且原其他确认分支为false | false |
| ROI+16/+28或-12，原对应方向/动量确认成立 | true |
| 原ROI区间条件满足但对应确认全无，尚未达灾难线 | false |
| ROI<=-20灾难例外，两侧 | true |
| ROI在(-5,+5) | 外层不调用，表达式不能实现区间内平仓 |

该矩阵已包含于4862合成检查；主后还必须以全部实际正常关闭的真实Position、ROI、outer gate和whole code核验，end_of_data独立截尾。

## 冻结回测与门槛

AF0/RG18/RG19组合×BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT，UTC2022-09-01inclusive到2026-10-01exclusive，12完整run、49个月、1491天/213周。每币>=0.9次/周最低192笔，不能换aggregate gate。报告四个Sep–Aug完整年、额外2026Sep、日历2023/2024/2025及Jan–Sep2026、side/family/最佳5集中度。年/45月切片仅退出时间归因，不是独立重置撮合收益。

原backtest_engine_v7 standard_1m、观测分钟close→next分钟open；初始cash1000、当前availablecash10%保证金×8、每侧fee0.0005、不利slip5bps、真实funding、原缺精确mark分钟Close回退、outer5/5、AutoStop=false和所有完整确认关闭保持。不放宽成本/0.9/四年/跨币、挑年或扫描新阈值。资金费率真实不意味着mark/深度/slip证据齐备。
四币数据SHA依次c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457 / 7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371 / 64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818 / 8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。真实fundtail54追加/7overlap/manifest78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547保持。
主输出results/20261004-rg19-development4-canonical-repaired-v2-funding-tail-v1.json自带逐run checkpoint；真实terminal前不因timeout/部分文件重启。12全部完成后全账本会计+8共享AF0/RG18完整控制exact；candidate path只准固定workspace/realpath等价，source仅cache_hit获取观察差异排除，不排除其余identity。

主后全部RG19补充opening独立几何/原receiver/QPS/当前极值，全部RG19正常closing和全12实际fills/currentcashqty/双边fee/funding/fallback/分钟成交活动独立核验。all-study与focus错误分别按真实行报告，所有失败保留，不能把旧控制失败称整个audit过关。标记精确mark、历史订单簿容量及实际滑点证据pending；未达到真实成本不能发布。

收益前预设closed4h ADX20 weak/strong及recentWidth<olderWidth的contracted/noncontracted自然归因，包含全部补充且逐笔配对真实开平仓，仅描述side/year/exit和集中度，不删组、不反向推alpha或称反事实收益。四开发币联合达标后才具备未阅验证资格；AAVE/ATOM/ETC/LINK收益仍不读。另一风险诊断的DOGE/LTC/AVAX/UNI/ZEC/ADA不称本线程未阅币。

## 源身份与权限

20:46实核engineSHAae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad、environmentb98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5、indicatorcache1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0、conf7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa、正式SKILL270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd不变。main/datacopy SHA21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8 / a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63。图谱/graph-augmented search均无对应代码结果，按不足回退已知当前源，不凭旧图谱推公式。
没有App、新DB写/分配/启用/下单/production/frontend/conf变化、仓库测试文件或材料删除。全部dirty、用户metrics timestamp-max规则、三个以上pending SkillMax草案保留；正式技能不晋级。当前goal active，尚无合格策略，不自行complete/paused/blocked。下次用量恢复前先阶段总结并核真实goal及进程。

