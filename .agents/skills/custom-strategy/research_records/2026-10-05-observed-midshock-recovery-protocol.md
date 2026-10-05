# RG28 反向冲击观测中点回收：收益前协议

冻结时间 2026-10-05 08:54:52 UTC。主输出尚不存在，限定活进程查询exit1，所有预检真实终止，未读RG28收益。前轮RG27原12run完整结束invalidated及其自身XRP零活动exit失败保留，不重复旧研究或删除失败证据。

## 当前用户目标与明确验收口径

用户本轮明确更新为每币≥0.8次/周，真实成本、四年稳定、跨币泛化，无需强求每币盈利。旧研究的0.9/每币净正和每币逐年净正仅为历史口径，不覆盖当前用户目标。213周每币至少171笔，不使用活跃周或四舍五入。逐币所有年/侧/费用/亏损完整披露，但逐币盈利不是硬门槛。

稳定性预声明为固定四币各1000USDT独立账户的等初始资金组合总净及四个完整Sep–Aug年度净均正，完整日历2023/24/25同样披露并核验。只相加已记录各账户净交易，不假设资金池重新复利；聚合已实现平仓顺序回撤仅为已实现代理，不声称完整组合mark-to-market回撤。最大贡献币剔除后净正作为收益集中/泛化约束，不能事后选亏损币删去。固定开发BTC/ETH/SOL/XRP，不挑币/年度/单边阈值网格。原未阅AAVE/ATOM/ETC/LINK不读收益，只有开发数值及信号/成本门槛通过才同冻结参数完整泛化验证，未阅组不要求每币盈利，但每币频率与固定组合年度稳定仍需完整检查。

完整真实成本仍需精确venue结算mark与历史book容量/滑点证据；模型原mark fallback和5bps假设完整标记而非冒充证明。没有达标前不复制发布模板、写库、分配、启用、下单。

## 唯一新家庭与固定范围

本地v29程序化只读快照实际exit0：活动[database] go_bn_test ID114，唯一精确name、两JSON语义与冻结原v29一致，database_writes0。整体whole RG27为父，仅替换两补充开仓对象和顶层name；原AF0基准开仓完整对象和顺序、九指标及统一完整RG4信号平仓不变。

LONG：最新closed1相对closed2下跌且反向实体为ATR1*.15..2、合法卖方quote严格多数、quote高于此前八closed2:10均值。当前观察上涨body、Low0>Low1，Close0严格收回closed1 BODY midpoint且不超midpoint+ATR1*.35；正当前累计quote<closedshockquote，同时保持原literalQPS0≥mean(QPS1:9)*.90。SHORT相反。当前quote未闭合，不称整小时缩量/当前主动方向确认；有意[0]保留，不读forming taker或MarketCondition。更早信号形成只是冻结研究假设，不能由旧关闭PnL反推出优势。整体路径重撮合，不静态反向或筛单。

family SHA f6899268718a0ffe0d7c90a112a91e7241fd3dcc0d4c613886cc72cb4347020b，project version d83bff0ce2516fb83b13b99fb1b1ef8216c9d41d492881adb408dfba6aef07d3。
combo SHA db7577065577596bd97bddc0b31ccfe0960fe6cfa4a813ee06664bf9112549e6，project version 09f142498db4feb6be445217aeff6a7fe3314e8f9704916c2688bf30a32687de。

## 真实预检及接收者证据要求

Go/Expr43131真实terminal0，12474 passed/0failed：独立numeric observed/closedshock oracle、legalOHLC、body/quote/多数/严格midpoint/原cap与literalQPS边界、观测body与反侧极值、无taker0和无taker2/ATR2依赖、全价量缩放、高周期闭合regime/日线veto、whole父对象/order/base9/完整关闭矩阵。原QPS-only浮点反例单独保留，不认为新rawquote90%必相等，不添加epsilon或改策略。新家庭和原RG27合成接受集合不同只证明非等价，不证明利润。

新opening审计16980真实build0。对全部实际supplement，通过entry_time-1独立归集仅已观察分钟Open/High/Low/Close/quote/QPS，完整canonicalclosed1..10每bar八字段、closed2:10均值和原199closed指标种子，closed1body/flow/quote及观察时点body/反侧极值/严格midpoint和cap/activity逐项配对。使用现有诊断only overlay导出原接收者，没有修改源、数据或填单；MAIN绝不使用overlay。Fresh接收者不是整个private/sequential/live/forward证明；原函数指标一致性不是独立数学验证。

Frontend实际exit0：两个候选issue=null/9/四type/shape，限定330portable/765enabledentries/0 whitespace-exact重复；排除own/research/diagnostic/>128KiB，不声称全局semantic/alpha新颖性。当前前端sourceSHA与这次结果完全一致。

API18876真实terminal0：六规则逐条code200/passfalse。固定mock只证明各表达式当前能运行，不证实际position方向、forward传播或盈利；服务实际可用，未自行启动或操作App。三个Node会计/自然归因/证据汇总syntaxexit0。汇总必须按当前组合门槛，不再要求每币盈利；每币亏损仍完整披露。

## 原风险、成本、日期与控制

UTC2022-09-01 inclusive至2026-10-01 exclusive，49月1491日213周；AF0/RG27/RG28×四币12完整standard1m/backtest_engine_v7，初始1000/币、current cash10% margin、8x、outer5/5、每侧fee.0005/不利5bps/实际funding，缺精确mark原分钟Close回退不变逐笔计数。

完整AF0/RG27八共享控制相对RG27父main要求whole快照、fullledger、指标、datahash/source/config/日期完全复现，仅source.cache_hit可观测变化。全12成本账必须包含RG27已知XRP零activity，不因新focus clean说all clean。按每条Passed谓词区分numeric和activityfailure，不依据generic“arithmetic failed”console推断公式错；失败交易/净值不删、无加回亏损/延迟成交信用/时间黑名单。0prints不是无orderbook的证明，但当时缺可执行证据；不授权静默修改fill/数据。任何不同撮合模型需标识、取得适当权限并完整重跑控制，不能单笔修补来过门槛。

## LONG/SHORT统一平仓矩阵

|条件|预期|
|---|---|
|outer(-5,5)|普通平仓不评估|
|ROI5/-5/16/28/-12 alone，无信号|false|
|ROI≥16 AND (trendfail OR momentumfail AND opposing price-volume impulse)|true|
|ROI≤-12 AND (trendfail OR momentumfail)|true|
|ROI≥28 AND (momentumfail OR opposing impulse)|true|
|(ROI≥5 OR≤-5) AND closed4hADX<20 AND live破closed反侧极值|true|
|ROI≤-20 catastrophe|true；唯一无信号确认例外|

统一关闭不依赖entryhash，AutoStop=false。先完整主actualterminal，再会计/八控制、全部新实际开仓、whole正常关闭、all12成本全部结束，再自然side/regime/year归因及阶段总结。任何partial/timeout不重启，exclusive新审计输出不覆盖旧证据。

## 唯一主命令

```text
rtk proxy go run /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go -conf conf/app.conf -strategy-files temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json,temp_strategy/20261005-counter-shock-flow-recovery/01-v29c-counter-shock-flow-recovery.json,temp_strategy/20261005-observed-midshock-recovery/01-v29c-observed-midshock-recovery.json -symbols BTCUSDT,ETHUSDT,SOLUSDT,XRPUSDT -start 2022-09-01 -end 2026-09-30 -execution-source public-archive -indicator-source public-archive -minute-repair verified-archive -cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1 -archive-cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2/archives -output /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-development4-canonical-repaired-v2-funding-tail-v1.json
```

## 固定数据身份

BTC c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457；
ETH7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371；
SOL64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818；
XRP8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。
fundtailmanifest78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547/真实54追加7overlap不变，canonical各interval量保留。全新cache/readsource沿用冻结verified-archive修复身份，不覆盖原数据或ARM。

## 收益前完整文件SHA

```text
7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa  conf/app.conf
ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad  service/backtest/engine.go
b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5  service/backtest/environment.go
1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0  service/backtest/indicator_cache.go
aef51d7cadaa5b561b4fdb893c4da3a2e089c3b69ecd47263c8553a0b35804b8  feature/strategy/line/technology.go
270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd  .agents/skills/custom-strategy/SKILL.md
41964ee1ab5cf1724730e13bc92f6f5d10b275460e60f8b2f4f3c5a2c944c4ce  .agents/skills/custom-strategy/research_specs/20261005-rg28-observed-midshock-recovery.json
b27cb840b0ee92bc165b64fed7b8e5574946d50cf7922f4e419b31aa78e6d04c  temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json
51f5055a310c5520a51783d0a4c2edb9af9cc269aa9b3db794fa5c6da5166db8  temp_strategy/20261005-counter-shock-flow-recovery/01-v29c-counter-shock-flow-recovery.json
f6899268718a0ffe0d7c90a112a91e7241fd3dcc0d4c613886cc72cb4347020b  temp_strategy/20261005-observed-midshock-recovery/00-observed-midshock-recovery-family.json
db7577065577596bd97bddc0b31ccfe0960fe6cfa4a813ee06664bf9112549e6  temp_strategy/20261005-observed-midshock-recovery/01-v29c-observed-midshock-recovery.json
21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go
a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go
672b6ddceb426c49e9dd50f02a6a3c59cd8b22491dd17b9004ea503332a3507e  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg28_pattern_expr_checks.go
8fd43094ae276edc9a80ab2bf9c8e939415f2dd1f6f316499262ea436c96aeb9  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg28_closed_signal_audit.go
9d5b55dc05c772165d3f265c03c38cf2da9b0a5a3de8695ec80e10f4acbe6ebf  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg28_closed_signal_audit
31b53b62d02df2cd5491cfde55badb290bbf4cbc61f0b442783be8a7eaa4e093  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg28_accounting_audit.mjs
e3634dc16d4954188eeed727e3d4d4a7b449181be1f1505d69a68a49dcafa83d  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg28_regime_attribution.cjs
48f5838ed3fff0fb87c31128910279a156bd429e96b9fafefe6b8294fcee5bd8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg28_report_evidence.cjs
3ef3dbe5abbe292405ab835835e355bb744495e494730b4065b41e175d7b3d34  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg24_signal_receiver_bridge.go
de75f4431457ac2e0e7c4e95046adf494472bb32a2766ac899d0565b67cd13bd  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg24_receiver_overlay.json
6247af2b6c2473779616b880d4cfa1cafaaf2d3bc87ad6fcd0c6ee401c71cb78  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg13_close_signal_audit
ada03ce7f3b86c2196f4804442516e85d6b95add30cc2ea70391d558be68c24b  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg10_execution_cost_audit
9f0cfbbca09c86c294881047f9c0c62b428ffc24c97d465a2e57d8e9854355e4  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg28_local_v29_snapshot.go
5664518e0dec3ed8ad8c21224831e928abe2d18f5d7c876f60d5d09b1804eeee  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg28_portable_preflight.cjs
702020e1b5667c23d0667d70d9fe68b307e0360d1d2062777876e5d0188d4764  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-expr-checks.json
67d3c5e795eb709a88f5caba48d5ce7a45ed41e0e967d8f96cb4076fdd7a5731  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-local-v29-snapshot.json
91f7f6ff8c435142eb76636ae0fd1db84be4a01a72fb609ac9051dc305543b60  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-api-rule-validation.json
29c34008ff1ad92afd5795686301bb5ee3ffe189e63e01bba31670d2de7c9616  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-frontend-contract-checks.json
f70ff322b13747d8013404cc0ec2caee7cd5a95a426e9547d3ce00d429c06943  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-portable-entry-identity-scan.json
0ac3a03d2561494ce9ec096b4427a0f6c15f15398d36eb7401668f5cae5b1ea1  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg27-development4-canonical-repaired-v2-funding-tail-v1.json
9c94aeb1210e4e4757b5b05a1193d0e868a5791e27457451d7200e02eb6957ee  /Users/zhz/work/binance/go_binance_futrues_new_ui/src/utils/technology.ts
```

conf/app.conf、生产/evaluator/前端/正式skill不改，未审批skill草案不得promote；不加仓库_test.go、不新增委派，不写globalmemory。Goal继续active，无结果前不称可用，不把阶段完成当目标完成。
