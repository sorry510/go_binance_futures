# RG28 反向冲击观测中点回收：完整开发结论

结论：`invalidated`，`release_qualified=false`。当前每币≥0.8次/周全部通过，但固定四币组合净负、前三个完整Sep–Aug年负、完整日历2023/24/25负、剔除最大贡献币后负，不能发布。判定不是要求每个币盈利。全部候选、亏损交易和控制失败保留；当前没有合格新策略。

## 范围与恢复完整性

UTC2022-09-01含至2026-10-01不含，49月1491日213周。AF0/RG27/RG28 × 固定BTC/ETH/SOL/XRP，12完整标准1分钟项目撮合。各1000USDT独立账户，当前cash10%保证金、8x、outer profit/loss5/5、每侧fee0.0005、不利滑点5bps、真实资金费时间/率、原缺mark分钟Close回退不变。不是重新资金池复利。

前次paused后，在明确恢复且实际goal active时核对32冻结SHA、原11run checkpoint和当前本地v29只读语义。主97136于北京时间23:19:28.158启动，只重做中断XRP/RG28，日志确认原11次already recorded；23:20:42实际terminal0。旧42234已terminated、新97136及全部审计均终止，不再poll或重启。恢复详细记录：`2026-10-05-rg28-resume1-audit.md`。配置、数据、策略参数和风险成本未改。

本地活动[database] `go_bn_test.strategy_templates` ID114的v29两JSON语义与冻结基线一致，新的只读snapshot75165 actualterminal0/database_writes0。未再次插入已完成的RG18（ID124）；该独立授权不覆盖RG28或下一候选。

## 逐币完整结果

单位USDT；资金费为净现金流，净=毛−双边费+资金费。滑点已进入成交价，不能再从净额扣一次。逐币回撤沿用项目账户模型，与下面组合已实现代理区别。

| 币 | 笔数 | 次/周 | 毛收益 | 双边费 | 资金费 | 净收益 | PF | 账户回撤% | 平均持仓h |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BTCUSDT | 407 | 1.910798 | 240.255771 | 254.232090 | -20.265967 | -34.242286 | 0.985705 | 46.312450 | 22.895413 |
| ETHUSDT | 463 | 2.173709 | 473.146896 | 350.255639 | -13.167270 | 109.723987 | 1.031007 | 37.738164 | 15.744456 |
| SOLUSDT | 465 | 2.183099 | 270.388401 | 400.493411 | -34.280238 | -164.385248 | 0.966976 | 43.066725 | 10.479211 |
| XRPUSDT | 462 | 2.169014 | 17.941588 | 289.998502 | -2.639441 | -274.696356 | 0.911327 | 49.159470 | 12.544697 |

每币需至少ceil(213*.8)=171笔，全四币通过；不以活跃周缩短分母或四舍五入过门槛。固定初始组合4000USDT，净-363.599904（-9.089998%），剔除最大贡献ETH后净-473.323890。这只是集中度描述，不是重平衡可执行反事实。按平仓顺序的组合已实现代理回撤33.905701%，不是完整mark-to-market权益回撤。

## 四个完整年与额外月

| 币/组合 | 2022-09至2023-08 | 2023-09至2024-08 | 2024-09至2025-08 | 2025-09至2026-08 | 额外2026-09 |
| --- | --- | --- | --- | --- | --- |
| BTCUSDT | -165.163139 | -266.076318 | 124.892295 | 280.799861 | -8.694985 |
| ETHUSDT | -84.202391 | -64.852170 | 67.107656 | 172.211885 | 19.459006 |
| SOLUSDT | 51.036776 | 168.393055 | -217.800947 | -134.427629 | -31.586503 |
| XRPUSDT | -6.678105 | -298.683500 | -62.001900 | 87.210672 | 5.456477 |
| 固定四币组合 | -205.006858 | -461.218933 | -87.802897 | 405.794789 | -15.366005 |

所有年为同一完整复利账户的交易归因，不是各年重新初始化的独立年化回测。

| 币/组合 | 2023 | 2024 | 2025 | 2026年1–9月 |
| --- | --- | --- | --- | --- |
| BTCUSDT | -124.875487 | -159.580136 | 125.557905 | 207.357322 |
| ETHUSDT | -64.618608 | -96.557561 | 168.070341 | 151.548745 |
| SOLUSDT | 156.959757 | -274.539410 | -164.398765 | -91.898458 |
| XRPUSDT | -30.354965 | 3.601012 | -258.835630 | 125.796825 |
| 固定四币组合 | -62.889303 | -527.076095 | -129.606148 | 392.804434 |

2026年1–9月为未完整日历年，不替代四个完整Sep–Aug周期。固定组合前三完整年/三个完整日历年均负；即使允许单币亏损，当前组合稳定性仍明确失败。

## 实际补充入场归因

1430补充+367本候选路径内base=1797。单独AF0对照429笔，不能静态相减或把被占用机会的原收益加回。组合顺序、持仓占用和cash复利均被完整重撮合。

| 币 | 补充笔数 | 毛收益 | 双边费 | 资金费 | 净收益 | 弱趋势笔/净 | 强趋势笔/净 | 剔除最好5笔净 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BTCUSDT | 327 | -173.124412 | 203.581732 | -10.068419 | -386.774563 | 161/78.580320 | 166/-465.354883 | -573.759644 |
| ETHUSDT | 376 | 129.039241 | 283.799691 | -4.142077 | -158.902526 | 191/-67.468270 | 185/-91.434256 | -501.859899 |
| SOLUSDT | 344 | 117.012751 | 296.111271 | -17.316892 | -196.415412 | 159/-183.513911 | 185/-12.901501 | -636.083445 |
| XRPUSDT | 383 | -226.538988 | 240.644340 | 0.335059 | -466.848269 | 201/-231.258424 | 182/-235.589845 | -794.596053 |

| 币 | 补充LONG笔/净 | 补充SHORT笔/净 | 整组剔除最好5笔净 |
| --- | --- | --- | --- |
| BTCUSDT | 157/-251.104724 | 170/-135.669839 | -276.751859 |
| ETHUSDT | 170/-138.033002 | 206/-20.869524 | -242.832315 |
| SOLUSDT | 167/159.904386 | 177/-356.319799 | -672.603047 |
| XRPUSDT | 194/-565.331708 | 189/98.483439 | -672.479195 |

弱/强分组完全采用实际信号时闭合4h ADX，不按收益选阈值；没有共同跨币稳定优势。SOL补充long、XRP补充short或BTC弱趋势的局部正收益不能授权事后选币/选方向/删组，更不能由净亏推断反向策略可盈利。BTC/XRP补充毛收益已负，ETH/SOL正毛收益不足覆盖原费用；问题不是漏扣费用。

同次AF0固定组合净+1724.235077，四完整年+2.152974/+88.678643/+573.325217/+1152.586986，剔除最高贡献XRP后+1112.522765，但各币频率约.484/.488/.624/.418均不足.8。RG27净+1278.060057，前三中的前两个完整年负，且新频率门槛仅SOL通过。对照暴露的是质量/频率权衡，不是任何版本已经完整可用。

## 全部接收者、会计和成本复核

- 会计actualexit0：12run/2864笔/8个共享AF0-RG27控制完整快照、ledger、metrics、data/source/config和原日期exact；errors=[]。不是只核总净。
- 开仓56439 actualterminal0：全部1430实际supplement、114400 canonical闭合字段、7150四小时/4290日线/2860 ATR/2860小时EMA/11440 priorquote/4290活动/1430中点续行复核，0失败。逐笔只聚合entry_time−1的已观察分钟OHLC/quote/QPS，保留原canonical区间量和200输入/199闭合指标种子。
- 最低literal原QPS比.9000228633008532；中点推进最大.3499661000496371ATR；shock实体.15027661317295535至1.9732689140635327ATR；最低收回实体比例.5000000000000044；最大观察quote/shockquote .9972954996340896。观察中累计quote不是已闭合小时缩量，也不是实时主动买卖方向；未读forming taker0。
- 平仓74147 actualterminal0：1797正常/0期末强制，旧分支1077/added728/共同8/added-only720，全部原关闭程序与真实position配对0失败。正常ROI-only仍false，只有ROI≤−20灾难例外；不依赖OpenStrategyHash。
- 全成本86260 actualterminal1：2864笔、6057实际funding应用、1772原mark回退；1零活动填单，1 Passed=false，all_passed=false。不是数值公式错误：最大quantity差2.7285e−12、gross/net差1.7053e−13、双边fee差2.5726e−15、funding差2.6645e−15，fill差0。
- 唯一失败严格归属旧RG27控制XRP seq59：entry1736858820000 / exit1736861520000（UTC2025-01-14 13:32退出，北京时间21:32），entryTrades2143/quote2348749.08897，exitTrades0/quote0。原失败单和来源context均保留；不改成交时间、不删单、不加回亏损，且0打印不等于不存在订单簿。
- RG28自身1797笔/3465资金费应用/1077原mark回退，0零活动/0数值失败；自身原模型算术与分钟活动检查通过，但精确交易所结算mark、历史盘口容量和真实滑点仍未证明。全-study失败不能因focus clean改称通过。
- Natural及phase汇总actualexit0，全1430配对，固定组合判定invalidated；汇总进程成功不改变成本审计失败状态。Fresh receiver/parity不是整个private/sequential/live/forward或独立指标数学证明。

## 原统一平仓决策矩阵

|场景（LONG/SHORT方向对应）|预期|
|---|---|
|outer ROI在(-5,5)|普通关闭不评估|
|ROI 5/-5/16/28/-12但无确认|false|
|ROI≥16且trendfail，或momentumfail与反向价量impulse同时|true|
|ROI≤−12且trendfail或momentumfail|true|
|ROI≥28且momentumfail或反向impulse|true|
|ROI跨±5且closed4h ADX<20、live破closed反侧极值|true|
|ROI≤−20|true，唯一无信号灾难例外|

AutoStop=false；正常信号退出只能在outer eligibility后评估，不能当盘中独立止损订单。未分配模板、未改symbol strategy_type或启用交易。

## 下一研究假设（尚未生成/读取收益）

停止以更早反向冲击回收强行增加频率。先核对是否已存在同一表达式，再用v29本身的方向/ADX加速/RSI实体/实时不追价构建补充入口，仅把新鲜突破的参考范围改为前4个闭合小时（一个4h方向周期）。原12小时base对象和顺序、全九指标、whole RG4关闭、日期/成本/风险/门槛不变。新家庭不额外用weak-regime反转，弱趋势不新开仓仍能按原结构退出。

这是一项新的入场家庭假设，不是将RG28亏损静态反向、筛赢家或调成本；更短结构也可能增加噪声，尚无盈利判断。不做lookback网格或重选年度；只有完整收益前预检/冻结协议后才在同次AF0/RG28控制下重新全49月撮合。开发门槛未过，不读取原未阅AAVE/ATOM/ETC/LINK。

## 保留文件与SHA

候选：`temp_strategy/20261005-observed-midshock-recovery/00-observed-midshock-recovery-family.json`（f6899268718a0ffe0d7c90a112a91e7241fd3dcc0d4c613886cc72cb4347020b），`01-v29c-observed-midshock-recovery.json`（db7577065577596bd97bddc0b31ccfe0960fe6cfa4a813ee06664bf9112549e6），combo项目version09f142498db4feb6be445217aeff6a7fe3314e8f9704916c2688bf30a32687de。

收益前协议`2026-10-05-observed-midshock-recovery-protocol.md` SHA d50896a7b217e3735fa5944f41d56ad844af44014d5cbfe59d660260d69c8c1a。大结果根为`/Users/zhz/Library/Caches/go-binance-strategy-research/results/`：

```text
10b73bd90b01eac334bbd518b3fa6903f85ee36addb62d19f484f46a8791bd69  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-accounting-summary.json
a59a37b7efba74e3f2d10696aff43f16462faa7c4c86ebc43afd7f41141e2beb  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-canonical-actual-pattern-open-signal-audit.json
d635e4021f20e8940f8eb2aed8dcbd1b2b8643deffc8fd3ce14defaeb661737f  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-original-position-close-signal-audit.json
cc662e0eb7280ed36213250ebc75ecbfab7e9571d2585fc5589baa79cd4cde56  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-execution-original-cost-fallback-audit.json
87049b986ed9fe703e6caea38bd59d429ddfb1b969860786e38db54f94c5a51a  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-natural-entry-regime-attribution.json
035f34b65acb5bf11fe394aa909afff052c48bb53f5e4ab4d4050d1ede1486e2  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-phase-evidence-summary.json
f7f216b0c6a17d0673edd4ed29e8c952af03fa9b6008bdc82b3a382e8c7a2f26  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-development4-canonical-repaired-v2-funding-tail-v1.json
```

全四canonical数据hash/fundtail54真实追加7精确重叠、保护conf/production/frontend/正式skill源hash与原protocol一致。最终完成限定pgrep exit1，无本轮主或审计活进程。不写数据库、不改配置/生产/前端/正式skill、不新增仓库_test.go、不操作App、不委派或更新globalmemory；goal保持active，阶段完成不等于目标完成。
