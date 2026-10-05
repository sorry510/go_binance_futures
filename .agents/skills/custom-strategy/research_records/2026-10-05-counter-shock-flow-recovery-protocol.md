# RG27 反向冲击失败与闭合主动成交回收：收益前协议

冻结时间 2026-10-05 08:06:05 UTC（北京时间16:06:05）。当前RG27主结果不存在，限定真实进程查询exit1，无主或后审计活进程；没有读取RG27收益。父RG26完整12run及全部后审计已结束invalidated，不重复旧研究、不删除失败资料。所有预检真实terminal0，HTTP为真实exit7。

## 唯一新家庭与冻结范围

本地活动[database]程序化只读快照27915实际terminal0：go_bn_test ID114，精确名称唯一，技术/策略语义与冻结原v29一致，database_writes=0。仅替换whole RG26中的两条supplement entry；整个基础对象、对象顺序、原9指标参数、统一完整信号确认RG4平仓、风险成本和用户门槛保持。两个完整portableJSON已先保存temp_strategy/20261005-counter-shock-flow-recovery/，不生成只有片段的策略。

LONG：closed2反向下跌实体/相对closed3下跌/卖方quote严格多数/quote高于prior8mean，closed1上涨实体和收复至少半冲击实体/Low1>Low2/买方quote严格多数/quote>prior8mean且<shockquote。随后liveClose0严格超过max(High1,High2)且不超该锚+ATR1*.35。SHORT方向相应反向冲击和价格/主动多数回收，High1<High2、实时跌破min(Low1,Low2)。两闭合quote/taker均要求0<=buy<=quote。原closed2 body ATR2*.15..2、4h weak或对齐strong、daily强反向veto、正live quote、原literalQPS累计90%保持。回收量>均值并<冲击量已蕴含冲击量>均值，不能把冗余条件算独立确认。故意live[0]保留，不读forming taker或MarketCondition。

这不是反转既有亏损交易，也不是挑币/时段/单边阈值网格。完整路径重撮合，不用静态筛除/反向交易声称利润。旧same-direction impulse与新opposing-shock家庭的synthetic接受集合不同，不据此推断利润。

family SHA609156f33e1e9408bffd125057f96f160739aec14ca157981c59ac99d99e403f；
project version849befb638ec71daa75e45f2c3a3b297ab87470b0edd03b587a3e127ffc8d4bf。
combo SHA51f5055a310c5520a51783d0a4c2edb9af9cc269aa9b3db794fa5c6da5166db8；
project versiondf95104b236f2507b25f6d11616dfc4f11c69f62d31f97e37067bf15764d3bee。

## 真实预检与边界

Go/Expr92721实际terminal0：10934 passed/0 failed。独立numeric observed-field oracle与合法OHLC双向fixtures、两闭合taker合法性及strict50%上下边界、意义回收quote均值/冲击上界、半实体inclusive、原shockATR范围、实时两bar max/min与cap、原regime/daily、全price/quote维度缩放、字面QPS90%。旧QPS近边界浮点反例单独保留为legacyQPS-only fixture，新的literalQPS equality采用可表示运行时单位；没有epsilon/production/策略参数修补。全parent对象/order及两entry唯一替换、wholebase/9/fullclose逐对象核对；normalROIalonefalse、方向确认true、唯一灾难ROI<=-20 true，空/wrong/真实hash一致统一关闭。局部ordered模型不冒充private/sequential/live/forward或历史盈利。

Opening审计98837实际build0；新closed1/2 dualflow、canonical字段及199closed实际指标种子、观察时点分钟累计量/原literalQPS独立核验，每条真实supplement必须配对。仅诊断overlay导出原函数，不改生产源；主回测绝不使用overlay。三个Node helper syntax0；accounting要求AF0/RG26八完整control和父RG26全study精确复现，唯一排除的观测字段是source.cache_hit。

Frontend刷新actualexit0：两issue=null/9/四types/shape；限定328portable759enabledentry，0 whitespace-exact duplicates，排除own/research/diagnostic/>128KiB，不是全局semantic/alpha新颖性。

HTTP actualexit7，127.0.0.1:3333拒绝连接，APIblocked/rules=[]，没有code200或pass结论。按custom-strategy真实Go/Expr回退可继续独立历史研究，HTTP/live/forward待完成；不自动启动服务或操作App。

## 全部原成本、风险与硬门槛

UTC2022-09-01 inclusive→2026-10-01 exclusive，49月1491日213周。AF0/RG26/RG27组合×BTC/ETH/SOL/XRP十二完整standard1m/backtest_engine_v7 run；初始1000USDT/币、current available cash10% margin、8x、outer5/5、每侧fee.0005/不利5bps/实际历史funding。原missing exact settlement mark→observed funding-minuteClose回退不改并逐笔统计；真实资金结算精确mark/订单簿容量/滑点还未证明，不以算术通过冒充完整realcost。

每币freq>=.9/全部日历周，至少192笔；每币净正、四完整Sep-Aug年都净正，日历2023/24/25和2026JanSep及额外Sep分别报告。gross/fees/funding/PF/DD/side/holding/最佳5笔集中度全部保留。完整年度和自然entry regimes属于exit时间cohort归因，不是独立初始化年度利润，也不是删除cohort后的可执行利润。

AAVE/ATOM/ETC/LINK原未阅验证币继续不读；所有开发门槛均通过才允许固定同版本进入泛化核验，精确venue marks与历史orderbook仍pending。失败保持invalidated，不降低门槛/挑币/宣称可用。

## 平仓决策矩阵（LONG/SHORT方向对应）

|普通资格/条件|结果|
|---|---|
|ROI在outer(-5,5)|不评估普通close|
|ROI>=5或<=-5以及16/28/-12区间，无任何信号|false|
|ROI>=16 AND (trend失败 OR momentum失败AND反向量价冲击)|true|
|ROI<=-12 AND (trend失败 OR momentum失败)|true|
|ROI>=28 AND (momentum失败 OR反向量价冲击)|true|
|(ROI>=5 OR ROI<=-5) AND closed4hADX<20 AND live破闭合反侧极值|true|
|ROI<=-20灾难止损|true：唯一无信号确认例外|

outergate仅是资格，AutoStop=false，入库/模板分配/启用/下单分别需要明确授权；本轮均不进行。

## 唯一主命令

```text
rtk proxy go run /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go -conf conf/app.conf -strategy-files temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json,temp_strategy/20261005-impulse-pullback-two-bar-extreme/01-v29c-impulse-pullback-two-bar-extreme.json,temp_strategy/20261005-counter-shock-flow-recovery/01-v29c-counter-shock-flow-recovery.json -symbols BTCUSDT,ETHUSDT,SOLUSDT,XRPUSDT -start 2022-09-01 -end 2026-09-30 -execution-source public-archive -indicator-source public-archive -minute-repair verified-archive -cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1 -archive-cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2/archives -output /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg27-development4-canonical-repaired-v2-funding-tail-v1.json
```

启动一次，保存真实句柄，实际terminal后才运行全会计/八control、全部开仓、全部正常关闭、全部12run成本与失败归属，再natural及完整年度/phase。观察timeout或partialcheckpoint不得重启。新产物wx/O_EXCL，不覆盖旧证据。

## 数据身份及当前SHA

BTC c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457；
ETH7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371；
SOL64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818；
XRP8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。
fundtailmanifest78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547，54真实追加/7exact overlap，canonical interval量不以分钟和替换。

```text
7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa  conf/app.conf
ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad  service/backtest/engine.go
b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5  service/backtest/environment.go
1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0  service/backtest/indicator_cache.go
aef51d7cadaa5b561b4fdb893c4da3a2e089c3b69ecd47263c8553a0b35804b8  feature/strategy/line/technology.go
270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd  .agents/skills/custom-strategy/SKILL.md
b27cb840b0ee92bc165b64fed7b8e5574946d50cf7922f4e419b31aa78e6d04c  temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json
16e0b90f1467551f44232a257e7be4c62dde9ad234483583b04730c528d0a1d0  temp_strategy/20261005-impulse-pullback-two-bar-extreme/01-v29c-impulse-pullback-two-bar-extreme.json
609156f33e1e9408bffd125057f96f160739aec14ca157981c59ac99d99e403f  temp_strategy/20261005-counter-shock-flow-recovery/00-counter-shock-flow-recovery-family.json
51f5055a310c5520a51783d0a4c2edb9af9cc269aa9b3db794fa5c6da5166db8  temp_strategy/20261005-counter-shock-flow-recovery/01-v29c-counter-shock-flow-recovery.json
21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go
a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go
3632cdc1d435a6b21278f96265a0034b19d84a36418c8f6167c678fda9b2835a  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg27_pattern_expr_checks.go
5da33be01232b00782c1be61d567c6535f6a77a626d56c59b0831c975b227198  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg27_closed_signal_audit.go
1882fa82743752d4ac011cfb629ecac7df927d0e16fdacc73673858f4a7b8001  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg27_closed_signal_audit
862bc858a1e63f9ff9efb11414125ed409349afaa1ae8a01fcad0da7e4d77bea  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg27_accounting_audit.mjs
0b35b823e51fbd168c26f97a5488c55e0ebc5088d8b046d2c4fa69a024ef5e4b  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg27_regime_attribution.cjs
0aad13826446bed577f7dba80a735b3ed92192e4c1561b6341609ac4e30cc90c  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg27_report_evidence.cjs
3ef3dbe5abbe292405ab835835e355bb744495e494730b4065b41e175d7b3d34  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg24_signal_receiver_bridge.go
de75f4431457ac2e0e7c4e95046adf494472bb32a2766ac899d0565b67cd13bd  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg24_receiver_overlay.json
6247af2b6c2473779616b880d4cfa1cafaaf2d3bc87ad6fcd0c6ee401c71cb78  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg13_close_signal_audit
ada03ce7f3b86c2196f4804442516e85d6b95add30cc2ea70391d558be68c24b  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg10_execution_cost_audit
20164f0fe5defd47c837cb5e4fff1fab800d2688f0485bcc872d4039872ecfbb  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg27_local_v29_snapshot.go
ae08e6f578b03dad1f408a7fccc5ca1eedd2961d0ec1fe2c7527aa35c275a50e  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg27_portable_refresh_preflight.cjs
cfcaca0c645f6b1652a05698d0ee7720715c16592730104501a00a00f0ee24f9  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg27-local-v29-snapshot.json
4f69faa48225ffaa6f86ba8b5f93d51d6c340a94e44aa0e446479cfeeda2d987  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg27-expr-checks.json
aab69be56351b1c9193f5e9e01511d3fc108a73fe25b292a7137550051dfce39  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg27-api-rule-validation.json
01fc7647d3a865a9b08f5a2b5ec8373f62ca8785f1e904d415368f9ca2f45877  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg27-frontend-refresh-contract-checks.json
096cc05830747e88199cd655ac3e89648c5f60f7ae714bf5ae926dd560334c88  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg27-portable-entry-refresh-identity-scan.json
ec3e1e225242b7d0927ae9028707634b0e7fbf6168163cde238098fffd347cf0  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg26-development4-canonical-repaired-v2-funding-tail-v1.json
```

正式skill/protectedconf/生产/前端不改，无新仓库_test文件、模板写/分配/激活/订单/App/删除/新委派或技能晋级。QPS及本地持久化经验已存在未审批技能草案，不伪造baseline/candidate行为评分。上一研究goal turn是完成RG25/RG26证据的progress，目标继续active，阶段结束不是目标完成。
