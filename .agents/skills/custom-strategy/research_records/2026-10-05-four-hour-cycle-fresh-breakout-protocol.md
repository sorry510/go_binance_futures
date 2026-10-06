# RG29 四小时方向周期新鲜突破：收益前协议

冻结时间2026-10-05 15:56:49 UTC。实际goal active；新主输出test存在性exit1、限定pgrep exit1无活主/审计。未运行或读取RG29回测，全部候选先保存在temp_strategy。RG28原12完整run及所有会计/开关/成本/natural/phase已结束invalidated，完整失败总结保留。当前没有合格发布策略。

## 当前目标与固定验收

每币至少0.8次/完整周、213周ceil=171笔；允许单币亏损，但全部币的亏损/费用/年/侧必须披露。固定四币各1000USDT独立钱包的等初始资本组合净、四完整Sep–Aug年及完整日历2023/24/25均净正，剔除最大贡献币后净正作为集中/泛化检查。剔除只作描述，不重平衡或删除账目；组合平仓顺序已实现回撤只是代理，不是完整mark-to-market权益回撤。旧0.9/每币净正/每币逐年正不是本轮要求。

开发固定BTC/ETH/SOL/XRP，不按结果换币、周期、方向或日期。原未阅AAVE/ATOM/ETC/LINK仍未读，仅开发数值和信号/成本门槛通过后同冻结参数全量跨币验证；不要求未阅每币赚钱，但每币频率和固定组合年度稳定需验。精确venue资金费结算mark和历史book容量/滑点仍待证明，不能以原fallback算术一致和分钟活动冒充真实成本完成。不复制发布、不写库/分配/激活或下单。

## 唯一入场家庭替换与机制

整体whole RG28为父，只换两补充开仓对象和顶层name；原12小时AF0/v29开仓完整对象和先base后supplement顺序、九技术指标参数、whole uniform RG4信号确认关闭均exact保留。

新两开仓来自原AF0完整v29侧逻辑，仅参考结构High/Low[2:14]改[2:6]、previous[3:15]改[3:7]，即前4个闭合小时和对应上个小时窗口，不扫4/6/8等参数网格。LONG仍为非拥挤funding≤.0001/日线EMA20非下降与PlusDI多数、closed4h close>EMA50/EMA20>50与EMA20非下降/ADX≥20且≥ADX[3]、PlusDI多数、closed1 RSI≥55/正实体.15..2ATR、新鲜4小时High突破+.1ATR；SHORT仍保留原更严格日线bearish EMA/ADX/红bar/收盘下降与closed4h空向ADX加速、RSI≤45、fresh四小时Low跌破（原SHORT不新增ATR breakout buffer）。

两侧当前观察Close0只落在closed1方向前进≤.35ATR或逆向≤.15ATR，当前累计quote>0是唯一新增入口的活动条件，**没有QPS≥.90入口条件**、没有主动量多数条件。原base和关闭中literalQPS未改；本候选用最新结构顺趋势扩充机会，不再用RG28反向冲击中点回收。弱4h ADX<20不新增开仓，已有仓仍按全uniform结构退出；不宣称弱趋势alpha已验证。

此为入场家庭更换，不能将RG28失败单静态逆转/删组或信用加回。本轮只是收益前假设：一个4h方向周期的局部小时结构可能更高频，也可能增加噪声，不能预设盈利。整体实际持仓占用和现金复利必须重撮合。

family SHA f1573b74f726baef57585280009faffab8f7f0b33f4ea678675a94377e01f7e5 / projectversion17d8c7db061fd510f7818824e65c4173dd1c16c9a73bb2f8f2e116a0ad31e990。
combo SHA723149a4e18016f2467f71f47032a5dcd4f2801b053934abd175455bc1a2abab / projectversionb45dd2bec4a70f5408e1d95c87818da409aa0e7af1b5bf8d7f06da4f7bce0930。

## 已完成预检与后审计准备

Go/Expr53340 actualterminal0：3562/0。真实Go结构与expr运行，独立numeric4h/12h入口模型、合法小时OHLC、current/previous fresh strict窗口、闭合ADX/EMA/daily/RSI/funding/body及as-of长度、原实时不追价与positive quote、单ULP严格边界（不加epsilon）、forming高周期数据和taker独立、价量缩放、整个base/order/九/wholeRG4/全部关闭矩阵。两种结构接受集合不同仅证明非等价，不证明盈利；local priority不是全部private/live/forward。

Frontendactualexit0：两份issue=null、9指标/allfourtypes/shape；限定332portable/771enabledentries，0 whitespace-exact重复。own/research/diagnostic/>128KiB排除；不称全局semantic/alpha新颖性。当前frontend源码hash与结果一致。

APIactualterminal7，127.0.0.1:3333无法连接，**没有规则响应，API未验证**。按skill实际Go/Exprfallback继续历史研究，未自动启动服务/操作App；可用时补查，固定mock依旧不证明仓位传播或收益。

新opening71333 actualbuildterminal0。新diagnostic-only overlay只导出原fresh BuildMinuteClose环境，源/私有逻辑未改；MAIN绝不使用overlay。全部实际supplement需：独立entry_time−1已观察分钟OHLC/quote/QPS；canonicalclosed1..14每bar八字段；原200输入/199closed seed的ATR/RSI/EMA/ADX、四小时ADX[1/3]/EMA[1/4]与closedClose、日线EMA/ADX/closedOpenClose、16条真实as-of funding时点/率；4h结构fresh/body/no-chase/quote正且原12h完整base拒绝，配合真实记录优先级。QPS活动比只作归因，不存在.90门槛。Fresh/parity不是全 sequential/private/live/forward或独立指标数学验证。

三个Node会计/natural/phase syntaxexit0。会计完整核全部账目与八AF0/RG28控制对原RG28main fullsnapshot/ledger/metrics/data/source/config/日期exact，仅source.cache_hit观察值可变。所有成本审计必须全12包括每个失败；按每条数值与zeroactivity谓词分类，不凭generic“arithmetic failed”console推断数学错误。已知RG27控制XRP59零活动失败留在历史RG28证据中，本轮不包含RG27 run，不能拿旧失败删除当前失败。所有失败不删、不延迟改单/改fill、不加回亏损/黑名单/新成本模型。

## 原风险、成本、日期与数据

UTC2022-09-01含至2026-10-01不含，49月1491日213周；AF0/RG28/RG29×四币12完整standard_1m/backtest_engine_v7。各初始1000/current cash10%margin/8x/outer5-5/每侧fee.0005/不利5bps/实际funding时点率及原缺mark分钟Close回退完全不变。原v29恢复只读snapshot75165 ID114两JSON语义exact/0writes，本轮未改变source库。

BTC c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457；
ETH7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371；
SOL64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818；
XRP8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。
fundtailmanifest78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547/54真实追加/7 exactoverlap，canonicalinterval量与verified-archive修复来源保持，hash/CRC由loader验证。

## 原LONG/SHORT关闭矩阵

|场景|预期|
|---|---|
|outer(-5,5)|不评估普通关闭|
|ROI5/-5/16/28/-12但无确认|false|
|ROI≥16 AND(trendfail OR momentumfail AND反向价量impulse)|true|
|ROI≤−12 AND(trendfail OR momentumfail)|true|
|ROI≥28 AND(momentumfail OR反向impulse)|true|
|ROI跨±5 AND closed4hADX<20 AND live破closed反侧极值|true|
|ROI≤−20 catastrophe|true；唯一无信号例外|

AutoStop=false，统一关闭不依赖entryhash，不改外部eligibility或生产传播。普通flat ROI-only不接受。主actualterminal后会计、开/关、all12成本全部终止，再全side/regime/year/固定组合汇总。若信号/会计证据不完整，保留并不能排名；数值组合全部门槛通过仍仅promising under tested conditions，真实成本/未阅/API仍pending。组合净或完整年负invalidated，仅频率/集中等不足needs optimization，不把阶段通过当发布资格。所有结果失败完整保留。

## 唯一主命令

```text
rtk proxy go run /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go -conf conf/app.conf -strategy-files temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json,temp_strategy/20261005-observed-midshock-recovery/01-v29c-observed-midshock-recovery.json,temp_strategy/20261005-four-hour-cycle-fresh-breakout/01-v29c-four-hour-cycle-fresh-breakout.json -symbols BTCUSDT,ETHUSDT,SOLUSDT,XRPUSDT -start 2022-09-01 -end 2026-09-30 -execution-source public-archive -indicator-source public-archive -minute-repair verified-archive -cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1 -archive-cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2/archives -output /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg29-development4-canonical-repaired-v2-funding-tail-v1.json
```

partial/checkpoint/timeout不等于实际终止，不重跑已完整run或并发新同输出；只追踪实际返回句柄。当前无新主句柄，启动后记入检查点。主输出独立且收益前不存在。

## 收益前完整32文件SHA

```text
7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa  conf/app.conf
ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad  service/backtest/engine.go
b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5  service/backtest/environment.go
1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0  service/backtest/indicator_cache.go
aef51d7cadaa5b561b4fdb893c4da3a2e089c3b69ecd47263c8553a0b35804b8  feature/strategy/line/technology.go
270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd  .agents/skills/custom-strategy/SKILL.md
9aa127a8be727189cdc27d0f4b5359931919184b6279bd20ccaab397e80e2582  .agents/skills/custom-strategy/research_specs/20261005-rg29-four-hour-cycle-fresh-breakout.json
b27cb840b0ee92bc165b64fed7b8e5574946d50cf7922f4e419b31aa78e6d04c  temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json
db7577065577596bd97bddc0b31ccfe0960fe6cfa4a813ee06664bf9112549e6  temp_strategy/20261005-observed-midshock-recovery/01-v29c-observed-midshock-recovery.json
f1573b74f726baef57585280009faffab8f7f0b33f4ea678675a94377e01f7e5  temp_strategy/20261005-four-hour-cycle-fresh-breakout/00-four-hour-cycle-fresh-breakout-family.json
723149a4e18016f2467f71f47032a5dcd4f2801b053934abd175455bc1a2abab  temp_strategy/20261005-four-hour-cycle-fresh-breakout/01-v29c-four-hour-cycle-fresh-breakout.json
21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go
a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go
aa4fa1a74752010506b275b58292ede0a61b903c25037897988bcc3b3f674100  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg29_pattern_expr_checks.go
d0333463f4448df1ac03d83e14002f54b0ef23fb605571f015f186e8c9feea8e  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg29_closed_signal_audit.go
442cd00e43d0cc791eb3e9bb7051ccf5d779cf48d187923b88e7b2597d954452  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg29_closed_signal_audit
b9ead0ee0fcd2b71f3081598975aa67ab7f95913f4ce350dd40c6a5a924baeab  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg29_signal_receiver_bridge.go
9f4a40666f3fc0f58a911fe9f62af939fedda77adc24ac4fc9a685777a61a846  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg29_receiver_overlay.json
ea500c66d43e5f2816909524ff5dc6d86bcefae9f631305a2fe0c3d92eb6be9b  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg29_accounting_audit.mjs
cbfb0b36f4f67491f4c2c928ad40bcb160974bcd5604757ba61d344ae53fd1d1  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg29_regime_attribution.cjs
0f645f2a0e7bd778ebbfb35f6d5081a072e6ddbbaa4eef3baf68cca53843994c  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg29_report_evidence.cjs
2f021863ed3704acf123ede081780ada87857e3e4fbce2dfd3f4bafb0d20ff78  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg29_portable_preflight.cjs
6247af2b6c2473779616b880d4cfa1cafaaf2d3bc87ad6fcd0c6ee401c71cb78  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg13_close_signal_audit
ada03ce7f3b86c2196f4804442516e85d6b95add30cc2ea70391d558be68c24b  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg10_execution_cost_audit
9c34b2e24e5d050eac2e80e3649f7677d8a62e647cbedd59be6b3b2e7eabb430  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg28_resume1_local_v29_snapshot.go
8fe0d0874850095b91425843d92a9776805781d5d75c07d3dcd6a2ddb9557f97  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg29-expr-checks.json
86a03d1d13505eea73aaab3d2707a9a06aa43a52aa53bfc33ae45b2fb501c591  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg29-frontend-contract-checks.json
e99fb63d463b26dcfbf981794e0e3f09d737ca463b4b877f7e9d0a0605181306  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg29-portable-entry-identity-scan.json
66bb020db003d0b7095ccca0c8ffdcd2138deeac2acdd8d885539f90a1a87c46  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg29-api-preflight-blocked.json
085e322a1825571c7aafb19094c29f704836c93a00020eb36c98af8181e328ee  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-resume1-local-v29-snapshot.json
f7f216b0c6a17d0673edd4ed29e8c952af03fa9b6008bdc82b3a382e8c7a2f26  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-development4-canonical-repaired-v2-funding-tail-v1.json
9c94aeb1210e4e4757b5b05a1193d0e868a5791e27457451d7200e02eb6957ee  /Users/zhz/work/binance/go_binance_futrues_new_ui/src/utils/technology.ts
```

conf/生产/evaluator/frontend/正式SKILL不变，无仓库_test.go/数据库写入/分配/启用/下单/App/新委派/globalmemory或技能promote。未审批技能草案trusted:false，正式技能不能凭synthetic分数晋级。
