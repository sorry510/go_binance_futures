# RG24：EMA20回踩收回实体极值确认，收益读取前冻结协议

## 当前证据及唯一改动

冻结时间：北京时间 2026-10-04 23:54:49（2026-10-04 15:54:49 UTC）。前一原合同研究轮 RG23 已全部结束，未通过频率和年度稳定性门槛；本轮 RG24 收益尚未生成或读取，没有真实回测进程。上个独立用户请求 RG18 入本地库已完成并回读 ID124，不重复插入，不构成 RG24 写库或启用授权。

本轮以 **RG21整个对象**为父，不以RG23为父。唯一逻辑维度是闭合小时回踩收回的结构参照：

- LONG 当前 closed[1] 的 Low/Close 双侧0.10ATR缓冲参照，由最近四小时 low 改为已启用 `ema_1h_20.Data[1]`；previous closed[2] 的 Low/Close 双侧缓冲参照由历史四小时 low 改为 `ema_1h_20.Data[2]`。
- SHORT 对称把当前/previous High/Close 双侧缓冲参照由历史 range high 改为同一闭合 EMA20[1]/[2]。

保留 RG21 实时 LONG Close[0]>High[1] / SHORT Close[0]<Low[1] 的严格整根极值后续确认、原极值+.35ATR/- .35ATR上限、闭合实体同向、闭合quote被动/对手方多数、Qps当前累计量≥闭合均值90%、原previous抑制及所有range合法性/日线/四小时条件。保留AF0基础全部对象和顺序、完整统一RG4确认退出、原九指标与参数。没有新增指标/周期、MarketCondition、形成中taker[0]、止损数字或方向反转。EMA20回踩再收回是假设，不是从损失推导的反向alpha或更高收益证明；没有按币/年选参或阈值搜索。

## 完整候选与父身份

两完整portable JSON已在helper生成前保存，包含全部long/short/close_long/close_short；永久保留失败试验。

- family：`temp_strategy/20261004-ema-reclaim-body-followthrough/00-ema-reclaim-body-followthrough-family.json`
  - SHA `9be6bc35aa34cfceccb1a0bf8093a0f7a746fab7b9ca7faba563a5944baba8f4`
  - 项目version `b91a1e6ded068c23d028ab4c61468a9a3df371bfcb84b46ca0acc9832e69d9e1`
- combo：`temp_strategy/20261004-ema-reclaim-body-followthrough/01-v29c-ema-reclaim-body-followthrough.json`
  - SHA `b49157e0691a2760ae4e160de13eb643a8e2a42f002133f27232c0803b6484de`
  - 项目version `39c0d43307d57131346829c9705c9656d0f9fa87886a7080be3c93cfd3559623`
- RG21父combo SHA `f4daf7caa9256fdb34326daf05503047895cee01b97b80ea371d3e194bf4647f` / version `77e817fbe930cd3772ae4a6b27b3497e87761a7680791029285a2e0785ace2fe`。
- AF0 SHA `b27cb840b0ee92bc165b64fed7b8e5574946d50cf7922f4e419b31aa78e6d04c` / version `59287750973dad3fa07f76f7e991f7b788a62960fc185b838cc49ad23ffddb68`。
- spec：`.agents/skills/custom-strategy/research_specs/20261004-rg24-ema-reclaim-body-followthrough.json`。
- 本地基础只读repeatable-read快照ID114、name唯一、0写，as_of_ms1791128094607，与冻结ARM v29语义exact。technology SHA `0ae3b681d78d49884c11165fc70684ca8db52c1db28a4bd6eb77d554f06bfa00`；strategy SHA `3d43aea89d4dc56b29aeb3890bdf92028ab0bac54777a9d9238df46cda4f00e2`。快照 `results/20261004-rg24-local-v29-snapshot.json` SHA `b89d707f4c69616f58d164dfb2d2318c1374bc88083f18e4f3af58085a55aa81`。

## 预检及保留的失败证据

真实Go/Expr首版实际退出1，5914通过/4失败，仅family/combo两侧price_scaled_declared_expectation场景：价格及ATR×1000但新EMA价格参照未同步缩放；独立数值oracle与表达式均false，没有模型与表达式不一致。没有读任何RG24收益后修策略。

- 首版源 `verification/rg24_expr_checks.go` SHA `14749ea8eafe269f8a8cd2091049d1f1631ce5f5c04acfec170c8efeb8462919`。
- 首版结果 `results/20261004-rg24-expr-checks.json` SHA `f41584e6d16c0096f138f2431aebb4eb01ab1cd11d26832ff50fa67dafa97034`。
- v2仅在price_scaled测试数据同步全部EMA20×1000；没有删检查、放宽预期或修改候选。
- v2源 `verification/rg24_expr_checks_v2.go` SHA `d3795dae529e85b9f2814d7cd2c918f8d465fcfefe25dd73cb1c99bdce98a57c`。
- v2实际退出0，5918通过/0失败，结果 `results/20261004-rg24-expr-checks-v2.json` SHA `7fcf3831c3e5a968b1e8fde12b67e6441c7cf614d77fb1657e65336dabb30278`。

包括整个父程序仅替换参照、独立数值oracle、严格0.10ATR等号两侧、EMA-only与旧range-only正交可达、previous闭合EMA新鲜性抑制、未用形成中/远端EMA隔离、原body/current activity/所有其余条件及完整退出矩阵。合成/本地有序模型不是实际指标数学、完整顺序private receiver或live/forward及盈利证明。

真实前端TypeScript隔离VM：两对象issue=null、9enabled、四types、shape通过；结果文件 `results/20261004-rg24-frontend-contract-checks.json`。限定322portable/741entry配置去空白身份扫描0精确重复；`results/20261004-rg24-portable-entry-identity-scan.json` SHA `c4028d3a751bb7b90c8863ece8f722b127d32e268e13f0cf6cdf91fc30dfc917`，不称全局语义或alpha新颖。

实际程序化本地HTTP六启用规则逐条code200/passfalse，已退出0且名称/类型与候选exact；`results/20261004-rg24-api-rule-validation.json` SHA `a86ca50dbe53d8a5a759594faf663425e988f58b3119c4f9ad408fa9a9df613b`。固定mock只是当前编译/运行，不证明真实LONG/SHORT仓位、8倍历史/private/forward或收益。不使用App UI或新仓库测试文件。

退出矩阵（两侧及空/错/实际entry hash）：

| 条件 | 决策 |
| --- | --- |
| 仅ROI跨5/-5/16/28/-12，无独立信号 | false |
| ROI≥16且趋势失效，或动量失效并反向活跃实体 | true |
| ROI≤-12且趋势或动量失效 | true |
| ROI≥28且动量失效或反向活跃实体 | true |
| ROI跨外部±5且闭合4hADX<20，并当前反向突破闭合小时极值 | true |
| ROI≤-20，无普通信号 | true，唯一无信号灾难例外 |
| ROI处在外部5/5内部 | 不获正常关闭评估资格 |

## 风险、时间、币与原门槛

原UTC2022-09-01含至2026-10-01不含，49月1491天213周；每币≥0.9次/周即至少192笔，四个完整Sep-Aug年度各净正，并报告日历2023/2024/2025、2026Jan-Sep和额外2026-09。不用额外月或年度cohort替代完整合同，不称年度独立初始化收益。

开发BTC/ETH/SOL/XRP全部，AF0/RG21/RG24组合各完整一次，共12run。未阅验证币AAVE/ATOM/ETC/LINK不读；开发联合过线后仍必须冻结参数、确认新币历史真实资金费tail/源完整性并完成跨币验证，不能直接发布。

初始1000USDT/币；每次当前可用cash10%保证金×8、outer5/5、双侧费率0.0005、不利滑点双侧5bps、真实历史资金费率及原缺mark的分钟Close回退。精确结算mark和历史订单簿/容量滑点仍pending，算术0失败不能宣告“真实成本”全部过线。原8x/仓位/退出/费用/日期/周频/年/币范围不调整。

## 原运行身份与诊断隔离

正式engine SHA `ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad`；environment `b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5`；indicator_cache `1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0`；feature/strategy/line/technology.go `aef51d7cadaa5b561b4fdb893c4da3a2e089c3b69ecd47263c8553a0b35804b8`。
app.conf SHA `7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa`；正式skill `270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd`，均不改。

原funding-tail main-copy SHA `21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8`，data-copy `a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63`。主撮合不使用任何诊断overlay。

新opening桥只新增导出已经计算的闭合hourlyEMA1/2，没有变更生产计算/缓存/evaluator；虚拟Go overlay仍仅两个原诊断桥路径，仓库无实际新桥文件。桥SHA `3ef3dbe5abbe292405ab835835e355bb744495e494730b4065b41e175d7b3d34`；overlay SHA `de75f4431457ac2e0e7c4e95046adf494472bb32a2766ac899d0565b67cd13bd`；opening源 `eced6e419b7b5c08046fd68c7d1f7d640546ad71a3f29e4aac69e1d78ccdb371`；actual build0 binary `f10e1947f3271e996abe255e85e42499aec4ee2067cff41fa3b690a091287117`。以canonical原199closed1h输入用原EMA函数复算，与真实receiver EMA1/2逐笔比对，再判断current/previous reclaim。此审计不是EMA独立数学或完整顺序private/live/forward证明。

完整12run结束后：全会计/八AF0-RG21共享控制整账目、指标/候选/源hash、全部实际补充闭合字段/EMA1/2与price/previous/body/currentactivity/其余条件、全部实际正常关闭、all12成本/失败行/零活动检查并单独归属，之后natural弱强4h、range收缩与旧close-cap描述cohort及完整年度分析。补充归属用新组合自身账目，绝不直接扣AF0账目或删cohort宣称新收益。

固定后审计源SHA：accounting `e975905f3ab572002fd982c4d1a03ce50ad4cf81c33f94141b65d51773210ebb`；natural `67d12de09f972ab7c9fe9c410828d3d528a57c4ac9e8bda5683a7a2ba9f14477`；phase `c59b41db8b9d4ed02671fc65c2564ab8dc19c6f0ef09f507daf87b2f06fba89c`。保留all costs及所有失败，不拿focus通过代替全控制通过。

## 完整主命令与等待纪律

```text
rtk proxy go run /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go -conf conf/app.conf -strategy-files temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json,temp_strategy/20261004-body-confirmed-reclaim-followthrough/01-v29c-body-confirmed-reclaim-followthrough.json,temp_strategy/20261004-ema-reclaim-body-followthrough/01-v29c-ema-reclaim-body-followthrough.json -symbols BTCUSDT,ETHUSDT,SOLUSDT,XRPUSDT -start 2022-09-01 -end 2026-09-30 -execution-source public-archive -indicator-source public-archive -minute-repair verified-archive -cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1 -archive-cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2/archives -output /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261004-rg24-development4-canonical-repaired-v2-funding-tail-v1.json
```

只poll新真实live句柄；partial、观测timeout或临时poll错误不是重启理由。旧RG22/RG23与本轮Expr/API/build/local预检均已actual terminal，不重poll。先核不存在主输出及同名活进程，再启动一次。保持所有dirty文件、failed candidates与pending trusted:false技能副本，不删/晋级/新委派；无App/新DB写/分配/activation/order/production/frontend/conf/仓库test修改。目标维持active，没有任何合格发布。

路径中verification/results相对于 /Users/zhz/Library/Caches/go-binance-strategy-research。

