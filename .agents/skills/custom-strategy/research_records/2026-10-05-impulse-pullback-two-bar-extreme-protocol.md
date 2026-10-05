# RG26 推进回撤两bar完整极值续行：收益前协议

冻结时间 2026-10-05 07:20:47 UTC（北京时间UTC+8）。当前 RG26 主输出不存在，限定查询没有主回测/RG26审计活进程；尚未读取 RG26 收益。父RG25已经12run/全部后审计完成invalidated，所有失败资料保留。

## 单一假说与范围

本地 go_bn_test v29 ID114 只读查询24876实际terminal0，精确名称唯一及technology/strategy语义仍等于frozen原v29，database_writes=0。继承whole RG25，仅改变两补充实时trigger与0.35ATR cap的共同锚：

- LONG：level=max(High[1],High[2])；Close[0]>level 且 <=level+ATR[1]*0.35。
- SHORT：level=min(Low[1],Low[2])；Close[0]<level 且 >=level-ATR[1]*0.35。
- Expr以三元比较实现level，独立oracle以numeric max/min实现，不从表达式反向生成oracle。
- 所有闭合放量推进/同向主动quote多数/反向实体缩量浅回撤/半实体及反向极值保留、0.90 current字面QPS、4h weak或同向strong、daily反向veto原样；原base对象/顺序、9指标参数和whole uniformRG4 closes逐对象精确保持。故意形成中[0]保留，不读形成中taker。

原补充407笔的完整canonical几何诊断actual exit0，不读取NetPnL/收益分组：131笔未过推进极值，243笔两bar完整锚不同，57笔在原cap内无法严格越完整极值。SHA da3aaeaeeb0d17827cbcf7551e0f6bb5083c44e24a018dfd6cfe0872aaad7b1b。新锚不是等价修复，不仅删除早入场：新cap可允许另一批更晚入场，必须完整路径重撮合。没有参数grid、按币/单边选择或收益选择过滤，几何不证明alpha。

两个完整候选已先保存于temp_strategy/20261005-impulse-pullback-two-bar-extreme/：
family SHA1d139c9a6acb525b281183859d1fbdf8400474f25aff152e96df387effb3e937/version286710821f989c80bfa19b53bbd19fdb6f66bf5e7ac2d89720b0343a31fe42d1；
combo SHA16e0b90f1467551f44232a257e7be4c62dde9ad234483583b04730c528d0a1d0/versionc747e01529534be1944333a4749892dc93fb3ccf6f9f8295d7a54b5fc3440972。

## 真实预检与当前限制

实际Go/Expr76112 terminal0，6610 passed/0 failed：独立合法OHLC长短二极值/strict equality拒绝/cap等号和上下边界、旧true新false的仅局部cross、旧false新true的移锚晚cross、全部parent剩余conjunction、精确base/order/9和whole close、原literalQPS浮点fixture及全引用维度缩放。仅synthetic/局部顺序模型，不是canonical全receiver/sequential/private/live/forward或盈利。

新opening73088 actual build0，原overlay仅导出receiver供审计，主不使用。新canonical审计独立记录两bar双方极值/level/pivot gap和literal Float64 QPS、原199closed指标种子与所有原条件。三Node后审计syntax0：eight shared AF0/RG25控制必须对父RG25完整study严格复现，只允许source.cache_hit观测变化；canonical pattern/全部normal close/all12成本后再natural年/方向/原weak-strong归因，禁止静态删交易宣称利润。

Frontend真实validator VM两issue=null/9启用/四types/shapevalid；该次限定326portable/753entries，0 exact whitespace duplicates（明确排除own/research/diagnostic/>128KiB），不是全项目语义或alpha新颖性证明。

本地HTTP actual exit7：127.0.0.1:3333 connection refused，lsof未发现listener，API status=blocked_local_service_unavailable/rules=[]。没有HTTP200结论。依custom-strategy实际Go/Expr回退可继续独立历史回测；不自动启动服务，不操作App，不修改conf。HTTP/live/forward仍待服务恢复，阻止无依据发布。

## 原完整风险/成本/门槛全部保留

UTC2022-09-01 inclusive→2026-10-01 exclusive，49月1491日213周。AF0/RG25/RG26组合×BTC/ETH/SOL/XRP，12独立完整standard1m/backtest_engine_v7 run。初始1000USDT/币/currentcash10% margin/8x，每侧fee0.0005/不利slip5bps/实际funding，原缺失精确settlement mark的observed minuteClose回退保留并逐笔报告。原source/repair/datahash/tailmanifest不变。

outer5/5只是可评估gate；wholeRG4所有正常profit/loss分支均需要方向相应price/trend/momentum/volume确认，唯一无确认灾难例外ROI<=-20。ROI>=16与trend或joint确认、ROI<=-12与trend/momentum、ROI>=28与momentum/impulse、weak4h ADX<20且ROI>=5或<=-5并破闭合反侧极值原样。AutoStop=false。此风险/平仓矩阵与前协议2026-10-05-volume-impulse-shallow-pullback-protocol.md（SHAa7e8137974fde7258f8734e4c6fb00f8fbc6be8493669c557cb68018cad9bc8b）的“固定执行与判定门槛”完全一致，不继承其旧局部开仓锚或“API成功”状态。

每币>=0.9/完整周，至少192笔；每币净正和四完整Sep-Aug年净正，完整日历2023/24/25及2026JanSep/额外Sep单列；报告gross/fees/funding/PF/DD/side/holding/集中度。年度及自然cohort只是按平仓归因，不是独立初始化收益或删交易反事实。开发全部原gate通过前不读未阅AAVE/ATOM/ETC/LINK；精确venue settlement marks、历史深度/容量/滑点、未阅验证仍pending。算术绿灯/分钟有成交不替代真实成本。

## 冻结唯一主命令

```text
rtk proxy go run /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go -conf conf/app.conf -strategy-files temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json,temp_strategy/20261005-volume-impulse-shallow-pullback/01-v29c-volume-impulse-shallow-pullback.json,temp_strategy/20261005-impulse-pullback-two-bar-extreme/01-v29c-impulse-pullback-two-bar-extreme.json -symbols BTCUSDT,ETHUSDT,SOLUSDT,XRPUSDT -start 2022-09-01 -end 2026-09-30 -execution-source public-archive -indicator-source public-archive -minute-repair verified-archive -cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1 -archive-cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2/archives -output /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg26-development4-canonical-repaired-v2-funding-tail-v1.json
```

主只启动一次，保存真实句柄，观察实际terminal后再运行全部后审计；partial checkpoint/观察超时不能结束或重启。所有新输出wx/O_EXCL，失败保留。没有写库、分配、激活、订单、App、生产/前端/conf/新仓库_test文件、删除或新委派。

数据hash仍BTC c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457；
ETH7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371；
SOL64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818；
XRP8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。
fundtailmanifest78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547，54actual additions/7exact overlap。interval canonical volume仍不以minute sum替换。

## 当前实际 SHA-256

```text
7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa  conf/app.conf
ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad  service/backtest/engine.go
b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5  service/backtest/environment.go
1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0  service/backtest/indicator_cache.go
aef51d7cadaa5b561b4fdb893c4da3a2e089c3b69ecd47263c8553a0b35804b8  feature/strategy/line/technology.go
270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd  .agents/skills/custom-strategy/SKILL.md
1d139c9a6acb525b281183859d1fbdf8400474f25aff152e96df387effb3e937  /Users/zhz/work/binance/go_binance_futures/temp_strategy/20261005-impulse-pullback-two-bar-extreme/00-impulse-pullback-two-bar-extreme-family.json
16e0b90f1467551f44232a257e7be4c62dde9ad234483583b04730c528d0a1d0  /Users/zhz/work/binance/go_binance_futures/temp_strategy/20261005-impulse-pullback-two-bar-extreme/01-v29c-impulse-pullback-two-bar-extreme.json
b27cb840b0ee92bc165b64fed7b8e5574946d50cf7922f4e419b31aa78e6d04c  temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json
d1cec654ad2a0e5dc6556e383805ddce64598a73f09d1bbd3074f4d396a30b06  temp_strategy/20261005-volume-impulse-shallow-pullback/01-v29c-volume-impulse-shallow-pullback.json
21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go
a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go
26a780772480013ab674de99095ed3460a40dc0bcd66dbe9e73a7f3d5582a980  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg26_closed_signal_audit.go
16929e945cfc97caab77fbfa66fc4ac264580af969b8350a500d4642e19cf77a  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg26_closed_signal_audit
bcae130040ec0716db4a23962e5f74c12230394ddf86efbc8129d9bcb83bd95c  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg26_pattern_expr_checks.go
c26dd85d4f31f463e2e5199d93d0bfd6c17624571f7e051dc951e15f378d515b  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg26_accounting_audit.mjs
fd05cc773cc2b015a2eff0dec8c4a9869ab075fae857ba21ce76d08e34048e79  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg26_regime_attribution.cjs
77cb1a98b1ec0bc5580ed525acfa5a5c8956709604acc65f4d9ef6f955bb9141  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg26_report_evidence.cjs
3ef3dbe5abbe292405ab835835e355bb744495e494730b4065b41e175d7b3d34  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg24_signal_receiver_bridge.go
de75f4431457ac2e0e7c4e95046adf494472bb32a2766ac899d0565b67cd13bd  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg24_receiver_overlay.json
6247af2b6c2473779616b880d4cfa1cafaaf2d3bc87ad6fcd0c6ee401c71cb78  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg13_close_signal_audit
ada03ce7f3b86c2196f4804442516e85d6b95add30cc2ea70391d558be68c24b  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg10_execution_cost_audit
c6a19fd8ef3ea4812c7087ba7753f81222ae6ca0b37eb13ee46c32a3a5ddbabf  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg26-local-v29-snapshot.json
6730033b7151d3976e89f880490800b3ce578f6bf1c98768cb767faa96c7e093  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg26-expr-checks.json
7fb56175f5422d773dd53c83eded6b09868e4eaf7553bef2849ce936a6ba0ef1  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg26-api-rule-validation.json
ac4789a4faa32fe9b5c532f56dd80578fad8e8fe3fc4cd26ff7960dbd94fc5fe  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg26-frontend-contract-checks.json
fd8312ae00cd9603e8eb56d905d007ccfe2eb12ef1f1daccb427dc2428ed8cf0  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg26-portable-entry-identity-scan.json
da3aaeaeeb0d17827cbcf7551e0f6bb5083c44e24a018dfd6cfe0872aaad7b1b  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg25-two-bar-extreme-geometry.json
6db2f7bf047ca6de90e6be041932fd2ba428c4d9625db108023716aae1368faf  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg25-development4-canonical-repaired-v2-funding-tail-v1.json
```

正式技能不变；本轮QPS fixture经验仅未审批managed草案，未独立behavioral score/gate/strict-win/promote。阶段完成不等于研究目标完成，不因预检通过称策略可用。

