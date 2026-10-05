# RG17 完整阶段总结：累计成交额改善组合结果，但频率与稳定性失败

## 当前结论与真实状态

2026-10-04北京时间14:32:44实核：goal active，无合格策略。主20497已14:28:57实际terminal exit0；新开48216、关50218 actualterminal0，成本36458 actualterminal1。限定pgrep没有本轮残留（exit1且无输出表示未匹配，不是任务报错）。全12完整49月回放完成，不重复启动。RG17 verdict invalidated，release_qualified=false；正总额不是四年稳定或可实盘证据。

本阶段实际progress：新完整候选/收益前验证/12回放/全账本控制/新增累计gate独立source核验/所有正常退出/全量成本失败明确归属/全量自然regime归因均完成。不是blocked或完整目标complete；没有用户暂停请求。

## 完整四币结果

UTC2022-09-01inclusive..2026-10-01exclusive，1491天/213周，每币最低192笔。现金1000起，每笔当前cash10%保证金×8倍；双边fee0.0005/各侧不利5bps/真实历史funding/原缺mark分钟Close回退。标准原backtest_engine_v7/standard_1m，观察分钟close→next minute open，outer5/5、AutoStop=false、wholeuniformRG4确认退出保持。

| 币 | 笔数 | 每周 | 净USDT | 毛USDT | PF | 最大回撤 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| BTC | 172 | 0.807512 | +272.164192 | +438.682650 | 1.188117 | 16.3638% |
| ETH | 195 | 0.915493 | +73.671788 | +225.791927 | 1.045981 | 28.2206% |
| SOL | 208 | 0.976526 | +376.322573 | +610.099322 | 1.141311 | 27.1288% |
| XRP | 174 | 0.816901 | +307.433228 | +463.753330 | 1.162040 | 32.7437% |

BTC/XRP不达0.9；平均持仓29.6083/17.6912/11.9979/13.2225小时。四币总额正，但无一四完整Sep–Aug年全正：
- BTC：−123.446606、+81.333833、+80.263254、+318.603016，额外Sep2026 −84.589305。
- ETH：−166.837373、−12.611525、+133.332965、+141.326856，额外Sep −21.539135。
- SOL：−62.314806、+274.215772、+179.812857、+3.073502，额外Sep −18.464750。
- XRP：+215.445484、−345.575675、+196.667736、+187.595914，额外Sep +53.299769。

完整日历2023/2024/2025与Jan–Sep2026：BTC +90.491331/+74.702120/+57.847288/+141.518778；ETH −8.467409/−59.098597/+94.596930/+193.614542；SOL +324.799744/−57.774490/−54.006528/+192.147174；XRP +98.882236/+0.325964/−89.557319/+302.281301。不能换年界隐藏亏损。最佳5笔剔除的描述剩余额为+7.367742/−239.002870/−44.510076/−110.152026；这是集中度诊断，不是新策略收益或反事实。

## 入口族与市场归因

真实RG17组合749笔=349补充+400本组合基础；不是749减AF0独立429。补充81/95/80/93笔，毛−99.514703/−92.916801/+262.335902/−281.618373，净−172.349718/−162.229409/+182.008226/−362.306021。三币补充毛也亏，不是只因费用。组合正额主要由基础支撑；独立族没有单独利润回放，不冒称其Standalone表现。

收益前预设closed4h ADX20边界的weak/strong全部349笔已和真实opening/normalclose配对：
| 币 | weak 笔/净 | strong 笔/净 |
| --- | --- | --- |
| BTC | 38 / −167.731714 | 43 / −4.618004 |
| ETH | 56 / −98.880325 | 39 / −63.349084 |
| SOL | 45 / +57.560483 | 35 / +124.447743 |
| XRP | 45 / −74.103419 | 48 / −288.202602 |

没有共同稳定盈利regime。SOL唯一正补充族去最佳5笔为−151.273431，weak/strong同样转负−153.183030/−177.383049，两组2025–26完整年均负。弱组中added-only退出36/53/42/42笔，净−179.689271/−81.555591/−48.008915/−106.611966，说明扫边收回后仍经常再次突破反向结构；不能据此删除弱组或平仓分支造盈利。strong也无跨币共同正额，不推亏损策略的inverse edge。

## 验证与成本

- Go/Expr合成26873 actualterminal0，4414/0。全两侧完整旧入口+唯一90%活动条件exact；独立数值oracle/90%等号与上下边界/字段index及clock/原完整开平仓矩阵验证。只证明合成与本地顺序模型，不是原private selector、历史顺序或live/forward盈利证明。
- 主12run/2763交易，完整会计0错误，8AF0/RG16共享控制按整snapshots、账本、metrics、年/频率、引擎、dataset及source（仅cache_hit观察除外）exact。path仅固定workspace/realpath等价，未删其他字段。
- 新开349/27920closed八字段/+1745四小时/+1047日线/+698ATR/+2094范围/+1047current活动field，实际原200input/199closed种子、newfullExpr/独立canonical已观察当前分钟累计quote与前8closed均值及QPS parity全部0失败。最小当前/闭合均值比0.900047289，阈值实核，非elapsed-rate与下一分钟容量证明。
- 正常退出749/0forced，old554、added199、both4、added-only195，完整原Position/ROI/outer gate/wholeRG4全0失败。旧helperScope里RG13是历史名标签，真实version/portable/study/hash按RG17配对，未换代码语义。
- 全量成本2763交易/5619资金费应用/1705缺精确mark分钟回退，failed1/zero_activity1，actualcostexit1。唯一失败是原RG16版本8478...的SOL seq244：2025-01-14T15:01Z/北京时间23:01 entryQuote0/TradeCount0；原失败、净额与实际source证据保留，不修源/engine、不删或加回。输出“arithmeticfailed”包含活动失败；最大净算术偏差仅1.8474e−13，不称发现真实货币算术错误。
- RG17自身749/1673资金费应用/527回退/0活动或算术失败，最大qty差1.8190e−12、净1.4211e−13。自身0失败不把all-study1改0，也不保证未来实时容量。精确结算价未完整，真实订单簿滑点/容量未建模，成本门槛仍pending。
- 本续接重查[Binance资金费历史官方定义](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data#Get-Funding-Rate-History)：markPrice是该资金费结算关联字段；文档支持字段含义，不保证我们的历史数据有完整字段。本轮未重新请求历史mark，先前12个实际空值样本不是全API永久不可获取证明。
- frontend实际TypeScript隔离VM两issue=null/9配置/四type/shape；限定307旧portable697entry0同侧整program精确重复，排除自身/research/diagnostic/audit/>128KiB。不是App/UI/build/API或语义/alpha新颖性证明。

## 关闭判定矩阵（同侧LONG/SHORT、原wholeRG4不变）

| 条件 | 正常决定 |
| --- | --- |
| ROI位于外部(−5,+5) | 不进入close evaluator；不保证零附近结构退出 |
| ROI普通±5/16/−12/28且没有确认信号 | false |
| ROI≥16且trend_fail或momentum_fail AND相应活跃实体冲击 | true |
| ROI≤−12且trend_fail或momentum_fail | true |
| ROI≥28且momentum_fail或相应活跃实体冲击 | true |
| ROI≥5或≤−5，closed4h ADX<20，LONG当前价严格破Low[1]/SHORT严格破High[1] | true |
| ROI≤−20 | true；唯一更深灾难例外，无普通确认要求 |

不同方向价格、EMA、DI、RSI和冲击条件对称定义；所有基础/补充统一退出，不做hash路由。Position时间退出未新增。CanOrderComplete受外部gate限制，AutoStopOrder保持false；模板插入/分配/启用均未进行。

## 保存身份与恢复

两完整portable均在temp_strategy/20261004-contracted-reclaim-cumulative-activity/：
族SHA6f77d651f4b64a97ec64ab88d636842ad21b3b9eb57e84c173a6209e3481d25d/version92edc06f2044523be89c92bf6d477d0a3bbccba676f6a22b38b3b2e2da7f9ce3；
组合SHA68e4a73b854cf2207d8b832085b78bdd10bc2ab3364fb077114863e4ae5cffd6/versionfb20107ca0e87d184f6c9e915a78456d05d37ab1a26eb279310f845324a1344c。全部失败candidate与raw成果保留。

研究results根=/Users/zhz/Library/Caches/go-binance-strategy-research/results/，20261004-rg17前缀：
- main SHA02e34fba947bd49861a4abd1f7d01d3fa0802135a036fed9ca73541a0a75fea2。
- accounting a211df1975ec8076e4350a4410699ed7bee7589677e16dd3cccfda2f6eef4fd1。
- expr7bfd2eef8c29f5fa8096c2398e715c7128affa75a77864b7e4d6e8d8864618d2。
- opening6da17a67483121da82b71acecc3f44c69af0fd15eefb8afff83ca263bcb2086e。
- closingea2116125162d355c6ca5d630a33ee26c8a00f79728170496b3c85c31ec79fff。
- failed-all-cost91d2d6e6a481ae787d5522cf63e2ece8890eab48fd6ebee66f3f2c30b5e3ae00。
- naturalb87bedab6a91c589020d0fd38dfc84b3ea2e9e5fb96cd7457c3012d802c9b0bc。
- phase-evidence-summaryb2f8228862000d406b1d91e62624d911a5eaa1d9c37c5a3381d82e506674b033。
- readonly ARM metadata/v29 snapshots49705 exit0，17模板/222、219、8结果；v29一致/raw SHA b511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0。只有元数据与v29导出，不声称全forward本轮重分析。

当前四币已开发，AAVE/ATOM/ETC/LINK收益仍未读；所有原9/风险/成本/频率/四年/跨币门槛保持。没操作App/DB写/分配/启用/订单/production/frontend/conf，未加仓库测试文件或删材料。conf与正式skill哈希7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa/270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd保留，外部metrics CSV规则及dirty成果不覆盖。三SkillMax草案行为pending，不晋级、不重复创建或委派。

下一具体研究维度：扫边收回已出现后，是否需要当前价格进一步严格越过该闭合信号的High[1]/Low[1]，而不是只在Close[1]附近容忍回落。只添加同侧follow-through一项，保留已有90%当前累计活动与所有其他条件；可能更稀疏，不能以放宽0.9来接受，先完整JSON/独立oracle/协议再全撮合，不因三币补充亏损反推可盈利逆向。RG18此时尚未生成或读收益。下次若限制后恢复，先此阶段总结、核实际goal/进程；不重复已完成R17、不自动pause/complete/block。

