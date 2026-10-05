# RG21：闭合实体同向的扫边收回极值确认 — 收益前协议

## 当前基础和冻结时点

本协议保存时RG21主回测尚未启动、RG21历史收益未读。RG20主81791/开93223/关70528/成本29788全部实poll terminal0，自然和phase actualexit0；12原49月run/1894账目/8全共享控制exact。RG20四币频率过而四币均有负完整年度，invalidated，不把累计净正当合格。

22:03:44前实际本地v29只读consistent snapshot3641 terminal0：活动127.0.0.1:3306/go_bn_test ID114，name唯一，technology语义0ae3b681d78d49884c11165fc70684ca8db52c1db28a4bd6eb77d554f06bfa00 / strategy语义3d43aea89d4dc56b29aeb3890bdf92028ab0bac54777a9d9238df46cda4f00e2，与原frozenARMv29 b511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0一致；新local快照SHA28d2937f47e378254a285b0893d71886b7922044be536d80c9ad2d94aec065d2，无App/应用初始化/DB写。AF0是有意变体而非DB原raw字节。

## 唯一机制：closed body响应确认

只在RG20两个补充完整程序尾部追加一个同方向闭合实体判定：LONG kline_1h.Close[1]>kline_1h.Open[1]，SHORT Close[1]<Open[1]。严格相等拒绝，无新数值阈值或指标。previous sweep/reclaim与forming当前strict High/Low cross不蕴含closed body响应：收益分组前冻结的完整477canonical/receiver/ledger诊断SHA9c90bb2fcb42c2b6bb073f41fcf34ed5961350abf93e1f4f94f2cc5a1131f43c，仅查计数不算PnL，BTC/ETH/SOL/XRP同向51/50/52/66、反向71/82/47/52、平0/0/1/5。

这是价格反应质量假设，不是因果吸收证明或从亏损推inverse edge。附加过滤可能显著降低频率，接受失败；不改每币0.9、不按币/侧/年挑条件、不用静态删除原单称新收益。R20发展时期已见，不称全局未阅市场；AAVE/ATOM/ETC/LINK历史收益仍未读。

全部其他条件exact：原Extreme锚0.35ATR cap/strict currentprice cross，闭合反向扫边及收回、previous几何firstness、positive两个range、合法闭合反向aggressor多数/放量、closed4h weak OR同侧strong EMA/DI、closed日线反向排除、recovery、90%forming累计QPS、正quote；基础整对象/顺序/9指标/三个周期/whole统一RG4完整关闭原样。故意[0]保留，不用forming taker[0]/MarketCondition/hash-bound退出。

## 完整候选与实际校验

- /Users/zhz/work/binance/go_binance_futures/temp_strategy/20261004-body-confirmed-reclaim-followthrough/00-body-confirmed-reclaim-followthrough-family.json，四规则，SHAa071dab86f01d76f099e28309acdb5c2733bc1a98dc0d422a899c0a0ffb20da4，version2b83defff791cdd916b3eccea18aed304fa7a8527d9fd6c2ab966c7b33491786。
- /Users/zhz/work/binance/go_binance_futures/temp_strategy/20261004-body-confirmed-reclaim-followthrough/01-v29c-body-confirmed-reclaim-followthrough.json，六规则，SHAf4daf7caa9256fdb34326daf05503047895cee01b97b80ea371d3e194bf4647f，version77e817fbe930cd3772ae4a6b27b3497e87761a7680791029285a2e0785ace2fe。
- Spec research_specs/20261004-rg21-body-confirmed-reclaim-followthrough.json 保存parentSHA/diagnosticSHA/唯一追加/所有门槛；两完整JSON均保留，旧失败资料不覆盖。
- Go/Expr70173实际terminal0：5654/0，证据SHA15c6a3a201b78f00ab8ec1f976dfcb76112a0d9fa2a4810c16da6c34274f55d6；sourceSHA37ee92d9c61fab15441aecb80a9d8934d50b302491246e01d7a12b0e8360eb45。whole previous程序仅追加严格body、独立两侧numeric oracle、legalOHLC body正/零/负与旧整程序、Open[0/2/9]synthetic索引隔离、原wide-wick/advance/cap边界与活动/flow/日线/强反向conjunction全保留；原关闭矩阵、base可达和whole对象差异全核。仅合成/local顺序模型，不是private selector/完整cache/API/live/forward/盈利证明。
- 真实frontend TypeScript validator隔离VM两issue=null/9enabled/四types/shape，证据SHA5a1ea8473e05f6e3911142e8a985e9b85b8c920306c9b2a714d28377b8118b8a；限定316portable723entry/0精确重复，SHA93569f207973f3c0cc58d3441c5ceb63a01c449990a480b3d37d332a22d18485，排除自身/research/diagnostic/audit/>128KiB，非全语义/alpha新颖。无App/UI/build/API。
- 新opening已actual buildexit0：sourceSHAdec8bdadf4c23acdb322f26bde2f3bd120ce6f9e12c1ab4d47afe05527669bf8 / binarya0ea5acc328a3ba0d633cacabd98d2f04d4ea10c3c2842b7ed0847acf9d64fb5；在原全部canonical/open/prog证据上增加closedbodystrict与receiver signedbody parity。主后将执行所有实际补充，不把synthetic替代历史。

## 原完整关闭矩阵与风险固定

|两侧情景|原决定|
|---|---|
|普通+5/-5只越ROI门槛、全部技术确认无|false|
|+5/-5且closed4h ADX<20和LONG严格破Low1/SHORT严格破High1|true|
|只等于闭合极值、其它原分支无|false|
|原16/28利润或−12损失区域且对应方向/动量/活跃body确认成立|true|
|未达灾难线且技术确认均无|false|
|ROI<=−20明确灾难例外|true|
|ROI在(-5,+5)|外层不调用|

以上已实际包含于5654自测，主后全正常关闭原Position/ROI/outergate/whole program另核。AutoStop=false，只有深灾难例外无技术确认，未换ROI-only。

## 主回测与完整后审计

AF0/RG20/RG21组合×BTC/ETH/SOL/XRP，UTC2022-09-01inclusive至2026-10-01exclusive，12完整49月run、1491天/213周；每币0.9最低192笔，四Sep–Aug完整年、extraSep、日历2023/24/25和2026Jan–Sep/侧/实际规则/集中度全报。参数在RG21任何收益前固定，不scan新阈值或丢旧年。

原engine_v7 standard_1m，已观察minuteclose→下一minuteopen，cash1000起、当前cash10%margin×8，每侧fee0.0005/不利slip5bps、实际历史funding/原缺markminuteClose回退、outer5/5保持。真实成本精确mark和历史order-book容量/实际滑点仍pending，不把原算术0失败称精确执行。原零活动检查不删单或改善PnL。

主结果results/20261004-rg21-development4-canonical-repaired-v2-funding-tail-v1.json自带逐run checkpoint；必须真实terminal才能启动后全审计，不因timeout/partial文件重启。会计12全run/8共享AF0-RG20完整控制exact，path仅固定workspace realpath等价，source只允许cache_hit获取观察差别，其余snapshot/ledger/config/data等全比较。

后审计所有actual RG21补充closed OHLC/200input199closed/ATR/regime/range/closedflow/strict/cap/QPS及新body；全normal关闭；all12 currentcashqty/fill/双费/actualfunding/回退/分钟活动，所有控制失败行保留、all和focus分别报告。自然ADX20 weak/strong、range收缩/非收缩和旧Close cap当前快照归因均全配对实际开关，仅描述而非独立年初始化或静态删除反事实PnL。全部开发联合门槛过才取得AAVE/ATOM/ETC/LINK未读资格。

数据四hash和fundtail54append/7overlap/manifest78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547固定，完整公共canonical repairedv2与tailv1不混ARM旧收益。mainSHA21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8 / dataSHAa340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63，generic wholeclosebinary6247af2b6c2473779616b880d4cfa1cafaaf2d3bc87ad6fcd0c6ee401c71cb78/costada03ce7f3b86c2196f4804442516e85d6b95add30cc2ea70391d558be68c24b均current实核不变。

## 权限与恢复

没有App、新DB写、assignment/启用/下单、生产/前端/conf修改、新仓库_test.go或材料删除。RG18只是前一用户独立授权已入本地124并fresh只读verified，不扩大RG21写库权限。引擎/environment/indicatorcache/conf/正式SKILL SHA保持，所有dirty/用户metrics修订与失败材料/pending技能副本保留，无新委派或晋级。

goal实际active，当前无满足全部门槛策略；本阶段继续实质研究，不自complete/paused/blocked。下次用量恢复前先阶段性总结并核真实句柄，旧70173/3641/build等已terminal不重poll；主启动后追加真实句柄。

