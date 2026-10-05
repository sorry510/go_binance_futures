# RG16 收缩区间反向扫边收回：完整阶段总结

## 当前结论与真实状态

**invalidated**。12次完整49月撮合与所有主后程序实际结束；四币频率通过，但BTC/ETH/SOL净亏、四币均有负完整年度。成本/执行审计还有1笔零成交活动分钟填单，明确失败，不可将算术一致当作可执行证明。没有满足每币≥0.9次/周、真实成本、四年稳定、跨币泛化的策略，没有发布或写库。

北京时间2026-10-04 13:44:52.581主63780启动，13:52:02实际poll terminal exit0。开31463、关43883实际terminal exit0，成本54594实际terminal exit1，源诊断98787在14:00:41实际poll terminal exit0。会计、自然归因、显式失败报告均actualexit0；报告能输出不等于执行gate通过。限定pgrep无本轮残留进程。实际goal active，阶段结束不是完整目标完成；下次先阶段总结，再核goal和进程。

## 实验与不变合同

只将RG15补充的同向越界突破换为反向严格扫边并闭合收回，首次按前次自身扫边/收回判断，位移由极值回收代替强制同侧实体。保留相反主动quote严格多数与放量、闭合4h weak OR aligned strong、日线强反向排除、原九配置、基础和完整uniform RG4关闭。不使用forming taker[0]、MarketCondition或entry-hash路由。完整协议已在收益前冻结。

原backtest_engine_v7/standard_1m，UTC2022-09-01inclusive..2026-10-01exclusive、1491天/213周/每币最低192笔；现金10%保证金、8倍、双边fee0.0005、各侧不利5bps、真实资金费及原缺mark分钟Close回退、外部5/5门控、AutoStop=false。原200input/199closed种子、54真实资金费尾部/7 overlap和四币data hash不变，不减少费用或更改年界。

## 原引擎组合诊断结果

不是已证明可执行的真实收益；SOL包含下面保留的零活动填单。

| 币 | 笔数 | 次/周 | 毛额 | 费用 | 资金费PnL | 净额USDT | PF | 回撤 | 平均持有小时 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| BTC | 343 | 1.610 | 123.427 | 234.684 | -20.814 | -132.071 | 0.941 | 32.88% | 24.96 |
| ETH | 409 | 1.920 | -301.736 | 192.329 | -11.961 | -506.027 | 0.752 | 59.15% | 14.28 |
| SOL | 430 | 2.019 | -177.872 | 284.777 | -27.746 | -490.395 | 0.863 | 59.57% | 10.09 |
| XRP | 403 | 1.892 | 553.480 | 347.669 | -9.372 | 196.438 | 1.053 | 51.15% | 11.27 |

完整四Sep–Aug年及额外Sep2026净額如下，均为退出归因，不是分别重置本金的年度收益率：

| 币 | Sep2022–Aug2023 | Sep2023–Aug2024 | Sep2024–Aug2025 | Sep2025–Aug2026 | 额外Sep2026 |
| --- | ---: | ---: | ---: | ---: | ---: |
| BTC | -260.107 | -9.948 | 94.766 | 102.295 | -59.077 |
| ETH | -362.441 | -142.958 | 6.805 | 32.224 | -39.657 |
| SOL | -139.560 | 4.716 | -162.329 | -167.889 | -25.333 |
| XRP | 414.638 | -536.541 | 220.763 | -32.432 | 130.011 |

日历退出归因2023/2024/2025/Jan–Sep2026净額：

- BTC -92.490 / 25.695 / -42.391 / 50.916。
- ETH -136.765 / -154.782 / -4.192 / 46.251。
- SOL 83.473 / -341.999 / -203.759 / -43.094。
- XRP 196.928 / 17.897 / -247.834 / 237.122。

四币都存在负完整年；XRP去最佳5笔净-665.456。AAVE/ATOM/ETC/LINK收益未读取，开发门槛已失败；精确结算mark和真实订单簿成本仍未证明。

## 新补充没有共同收益优势

真实补充1219笔、组合基础366笔，不能用1585减独立AF0的429来算补充。组合持仓占用和复利改变后续基础机会。下列仅原组合账本归因；独立族本轮没有单独收益研究，静态剔除不产生新PnL。

| 币 | 补充笔数 | 补充毛额 | 补充净額 | LONG净額 | SHORT净額 |
| --- | ---: | ---: | ---: | ---: | ---: |
| BTC | 265 | -365.085 | -555.671 | -241.362 | -314.308 |
| ETH | 314 | -504.214 | -659.476 | -343.604 | -315.872 |
| SOL | 308 | -312.385 | -527.821 | -393.138 | -134.684 |
| XRP | 332 | 55.193 | -237.989 | 240.363 | -478.352 |

BTC组合毛额正但成本后亏；ETH/SOL补充毛额本身负，不是仅费用问题。XRP补充整体仍亏，原基础两侧+434.428支撑组合净正，不能挑XRP LONG或反转SHORT宣称盈利。

预先指定已有ADX20 OR边界weak/strong自然归因净額：BTC -239.130/-316.540、ETH -262.619/-396.857、SOL -216.837/-310.984、XRP +150.913/-388.902。唯一正弱XRP去最佳5单转-517.990，无共同稳定组。全部1219与原开/关证据匹配，只描述，不删组或改阈值。

## 通过和失败必须区分

- 实际Go/Expr 4158/0；独立数值oracle、明确两侧扫边/收回、严格多数/合法quote、首次抑制、ADX/daily、recovery/live、未用forming/ratio字段和旧完整关闭矩阵/基础原九整对象通过。
- 当前前端TypeScript validator隔离VM两issue=null、9配置、四类型/shape有效；限定305旧portable/691enabled入口0精确重复。不是语义/alpha新颖、API/live/forward或盈利证明。
- 会计12run/2689账目、8AF0-RG15完整共享控制逐笔/metrics/年度/source/snapshot重现、errors0，路径只按已证等价resolve/realpath归一化，其余字段严格。
- 1219新开仓canonical闭合97520字段、4h6095、daily3657、ATR2438、range7314、actual200input/199closed、独立相反quote和新几何/fullExpr0失败。最小回收0.20568ATR，0.15下限由扫边+收回蕴含；fresh原函数不是全sequential/private/live或独立指标数学证明。
- 1585正常关闭、0forced end、原944/新增654/both13/added-only641，原Position/gross ROI/outer gate/whole RG4与独立弱结构0失败。参数化原RG13close helper旧Scope名称不代表旧候选，本轮input/SHA/version/portable均核对。
- 全5370资金费应用/1528缺mark分钟回退；自身2853/910。最大qty误差4.1e-12、net2.3e-13，在原容差内；**零活动填单1/failed1/成本程序exit1**。原输出“arithmetic failed”包含零活动判定，不能误报为金额算术错误，亦不能改成0。

正常关闭矩阵保持，ROI为原mark价分母gross口径：

| 场景 | LONG/SHORT预期 |
| --- | --- |
| 普通ROI门槛到了但独立信号不支持 | false |
| ROI≥5或≤−5、闭合4h ADX<20、LONG跌破前Low/SHORT越过前High | true |
| ROI≥16且trend失败或momentum+反向放量冲量 | true |
| ROI≤−12且trend或momentum失败 | true |
| ROI≥28且momentum或反向冲量支持锁利 | true |
| ROI≤−20灾难例外，无信号也可 | true |

ROI在(-5,5)外部门控不运行关闭；普通ROI-only拒绝，灾难是明确例外。

## 执行异常的源证据

位置：SOL第244笔SHORT，UTC2025-01-14T15:01:00、北京时间23:01，原分钟Open187.13/quote0/TradeCount0。原引擎仍以187.13×(1−0.0005)=187.036435填单。净-9.618686保留，不静态删除或加回；改占仓和未来数量必须新全撮合。

原月ZIP在cache archives/a368e821fbab57c4-SOLUSDT-1m-2025-01.zip，SHA b56b792f9fc9cd63dc0fef4841741692dc2e2cab3282faab66e9d0dc01ce58a1。三邻接CSV与canonical OHLC/quote/trade/taker字段完全一致：15:00有11笔/quote4303.77；15:01全零；15:02有151笔/quote239299.91。不是新增指标或缓存制造零行。

当前原engine pending只检查正fill/notional，未验证TradeCount/QuoteVolume，SHA ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad未变。历史无成交不证明当时订单簿为空，亦不能证明该模拟填单真实可执行。

源诊断98787 actualexit0。当前观察quote4303.77 / 前8closed[1:9]均值123110255.063375 ≈0.0000349587，即0.0035%。现有AF0的90%累计活跃条件会拒绝这一已观察信号，无未来字段；不保证下一分钟活动/capacity/成本/盈利。QPS分母名义3599.999秒，不是elapsed-minute-rate。未修引擎、价格、成本或账本。

## 完整文件、SHA与恢复

两完整JSON保存在temp_strategy/20261004-contracted-range-opposed-quote-reclaim/。族SHA a647f88bb45736e45c869791fa66f29a7b0da48fd98db62730f345d4d916b547/version9d2a2ffbe3955a29001e71bd1154c06493735d2376691d52e4f81751bfbe4d26；组合SHA2e574fb5580d80d3efdce2a2416cc9212c7972ad2c91fdfa7b0cbd25bbe12312/version8478b5940bed703f70698cfd74ef39316770dfc623c341855c4645fed02572ce。完整收益前protocol在.agents/skills/custom-strategy/research_records/2026-10-04-contracted-range-opposed-quote-reclaim-protocol.md。

缓存根/Users/zhz/Library/Caches/go-binance-strategy-research/：全部raw/checks在results，隔离helpers在verification。关键结果SHA：

- 主7b7f822953c89448b8b9baa7d6f6a33727173154f95315dfa069d8f4eb3e0a7b。
- 会计daeec5ded1a4a0ac79091cfb6a8fc841d147939bc17cbfd38becd61f8740664a。
- 自测440255faf23e48e5e47eb720458cdb2466eb32efd8123c00614093443004a1ba。
- 开89951c5c5a9afbef29c792c823ace434b27b39b7a5b6ef84488c8c82bf4c6b65。
- 关7271a12b168785e71a2388198a124252bf15c29ccaef368e852d1bc804bf295e。
- 失败成本2f9de202eca527b8a509e00955928677d14183b2d1797cc4091f18ae510c9915。
- 自然归因e7ee6899bca9b0ca8a809e06b8123adc56df1f11538ec4606ae02d0fbe3319d7。
- 原零活动诊断df580ea40027b447605b2b72c82a4ff2a8639e62b97ab24a89c3f72eda3745bc。
- 失败执行专用只读report helper beaff133412ad2cd8a091f221e801291a65435be609408da4326dda27223b1ce，明确invalidated/execution FAILED/failed row。原成功路径report保留，不删failed判定冒称通过。

没有App操作、DB写/分配/启用/下单/production/前端/conf/app.conf修改、仓库测试文件或材料删除。正式SKILL SHA270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd、conf SHA7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa、三个原生产源码hash保持。用户metrics CSV规则与dirty研究保留。SkillMax comparison评审副本apply5/结构有效，7行为例pending，未晋级。

下一安全动作：先读PV5完整累计量门槛配置和现有QPS接收合同，再单维检验扫边机制加入同一个已有90%观察累计quote活跃条件，完整候选/oracle/收益前协议后四币和完整对照重撮合。它不是下一分钟流动性/盈利保证。RG17尚未生成或读取收益，不挑弱XRP/方向/年度，不修生产引擎；原全部门槛与未阅验证币保持。下次首条先本阶段总结，再实核goal/进程；不复用终止句柄，不将阶段结束mark complete/paused/blocked。
