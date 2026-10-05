# RG25 放量推进—缩量浅回撤续行：收益读取前协议

冻结时间：2026-10-05 06:52:54 UTC（北京时间 UTC+8）。此时 RG25 主输出不存在，限定进程查询未发现主回测或 RG25 审计进程；尚未读取任何 RG25 收益。RG24 已完整结束并判定 invalidated，不覆盖其失败证据。

## 研究问题与唯一改动范围

以本地 go_bn_test 的 v29（ID 114）为基础，继承 AF0 开仓及整个 uniform RG4 信号确认平仓。父版本为 whole RG21，仅替换两条补充开仓对象，不修改基础对象、规则顺序、九个指标及参数、费用、仓位、日期、关闭逻辑或门槛。

新机制不是 RG24 的 EMA 收回，也不是调 reclaim 锚点：闭合 [2] 的同向实体推进且成交额高于此前八小时均值、主动成交额多数与价格方向一致；随后闭合 [1] 是反向实体的缩量浅回撤，保持推进实体至少一半且不破推进反向极值；当前 [0] 严格突破回撤 bar 的顺向极值。

LONG 的推进 Close[2]>Open[2] 且 >Close[3]，buy_quote*2>quote[2]；回撤 Close[1]<Open[1] 且 <Close[2]、Low[1]>Low[2]、Close[1]>=(Open[2]+Close[2])/2。SHORT 对称。推进实体必须处于 ATR[2]*0.15..2.0；quote[2]>mean(quote[3:11])>0，quote[1]>0 且 <quote[2]。保留原实时极值 cap 0.35*ATR[1]、正当前 Amount[0]、Qps[0]>=mean(Qps[1:9])*0.90、原 4h ADX<20 或同向强趋势、日线强反向 veto。仅使用闭合 TakerBuyAmount[2]；不读取形成中 taker[0]，保留故意实时 Close/Amount/Qps[0]。

这些结构是冻结假说而非盈利事实。复用 0.15..2 ATR 是用于不同的推进实体；半实体回撤是预先定义的结构条件，不进行参数网格、按币规则或根据后续收益选组。没有前一模式 guard：当前同向推进[2]与上一同向模式的反向回撤[2]互斥。

## 已完成的收益前检查及证据边界

- 本地 v29 仅程序化只读 repeatable-read 查询，无 App 初始化。ID 114、精确名称唯一；technology 语义 SHA 0ae3b681d78d49884c11165fc70684ca8db52c1db28a4bd6eb77d554f06bfa00、strategy 语义 SHA 3d43aea89d4dc56b29aeb3890bdf92028ab0bac54777a9d9238df46cda4f00e2 与冻结原始 ARM v29 一致，database_writes=0。
- 两个完整 JSON 已先保存于 temp_strategy/20261005-volume-impulse-shallow-pullback/。Family version c913b85524fadb103826f419579734f4135fee297d3445acc79626df6b543d3f；组合 version 33f9671f904d569ec26b0181078f54d5042317b25f8e9bcf9c2eee6d1c8f99eb。
- 首版实际 Expr 5930 passed/4 failed（真实 terminal 1），四处都是 raw quote 90% fixture 对原 Float64 QPS 运算顺序的错误等价假设。Expr 与独立 numeric oracle 都正确返回 false：差 -3.469446951953614e-18。首版源/结果完整保留。
- v2 仅修 fixture：原舍入 false ケース仍保留；以 3599.999 同步缩放全部 quote/QPS/taker quote 单位，使字面 QPS 等号可表示，并保留等号与上下边界。实际 5962 passed/0 failed/terminal 0。没有给策略或生产 QPS 增加 epsilon、改阈值或删失败证据。覆盖 LONG/SHORT 合法 OHLC、非法流量、全部新结构边界、整体维度缩放、基础对象/顺序/九指标不变和原完整平仓矩阵。局部顺序模型是 synthetic，不是私有/全顺序/live/forward。
- 前端真实 validator VM：两文件 issue=null、9 enabled、四类型。该次限定扫描 324 portable/747 enabled-entry、0 exact whitespace duplicates；不等于全项目语义新颖性或 alpha 证明。
- 本地程序化 HTTP 六条启用规则逐条 code200/passfalse，实际 terminal0。固定 mock 仅证明编译/运行，不证明真实方向仓位、8x、历史执行或盈利；无 App UI、模板插入、分配、激活或订单。
- 新 RG25 开仓接收者审计 actual build0；原诊断 bridge/overlay 仅供审计导出，不用于主回测。三份 Node 会计/自然归因/phase helper 已 syntax-check terminal0，尚未对新收益运行。
- 原主引擎、环境、indicator cache、指标源码、conf/app.conf、正式 SKILL 和 funding-tail 数据/回测副本 SHA 当前实核未变。

## 固定执行与判定门槛

UTC 2022-09-01 inclusive 至 2026-10-01 exclusive，共 49 月、1491 日、213 周；不得被技能默认 45 月替换。开发 BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT 全部执行。AF0/RG21/RG25 × 四币，共 12 完整独立顺序 run；共享控制必须与 RG21 原完整 study 八个 run 的整个账目、指标、策略快照、数据和 source（唯一允许 cache_hit 变化）完全复现。

初始 1000 USDT/币，当前可用现金 10% 作 margin、8x；每侧 fee=0.0005、每侧不利 slip=5bps、实际历史 funding。原缺失精确结算 mark 的分钟 close 回退原样保留并逐笔计数。outer profit/loss=5/5 是平仓可评估门槛，不是固定到价强平。正常关闭整个 uniform RG4：ROI>=16 的反转确认，ROI<=-12 的失效确认，ROI>=28 的动量/量价确认，以及 4h 弱趋势且当前破闭合反侧极值的 ROI>=5 或 <=-5 确认。唯一无确认紧急例外 ROI<=-20。AutoStopOrder=false。

| 两侧共同条件 | 预期平仓 |
| --- | --- |
| outer gate 内，非紧急 | 不评估/不平 |
| 仅跨普通 ROI gate，趋势/动量/量价/弱结构均不确认 | false |
| ROI>=16 且方向相应趋势反转，或动量衰退与反向量价同时确认 | true |
| ROI<=-12 且方向相应趋势或动量失效 | true |
| ROI>=28 且方向相应动量或量价反转 | true |
| ROI>=5 或 <=-5，闭合4h ADX<20且实时破闭合1h反侧极值 | true |
| ROI<=-20 明确紧急灾难止损 | true，唯一无信号确认例外 |

每币频率 >=0.9/完整周，至少 192 笔；每币净正、四个完整 Sep–Aug 年各净正，完整日历 2023/2024/2025 分别报告、2026 Jan–Sep及额外2026 Sep单列；每币及每侧集中度、PF、drawdown、gross/fee/funding、holding 时间全部披露。年度/家庭/自然组是按平仓时间归因，不是重新初始化年收益或静态删交易的反事实利润。

原未阅 AAVEUSDT/ATOMUSDT/ETCUSDT/LINKUSDT 保持未阅，只有开发全部原门槛通过才做冻结候选的跨币验证。精确 funding 结算 mark、历史深度/容量/滑点证据仍待补全；原模型算术正确、分钟有成交或开发盈利均不能替代真实成本/泛化。不得改费用、杠杆、风险、时间、币集合、频率或年度门槛以制造通过。

## 唯一主命令（收益前冻结）

```text
rtk proxy go run /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go -conf conf/app.conf -strategy-files temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json,temp_strategy/20261004-body-confirmed-reclaim-followthrough/01-v29c-body-confirmed-reclaim-followthrough.json,temp_strategy/20261005-volume-impulse-shallow-pullback/01-v29c-volume-impulse-shallow-pullback.json -symbols BTCUSDT,ETHUSDT,SOLUSDT,XRPUSDT -start 2022-09-01 -end 2026-09-30 -execution-source public-archive -indicator-source public-archive -minute-repair verified-archive -cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1 -archive-cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2/archives -output /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg25-development4-canonical-repaired-v2-funding-tail-v1.json
```

只启动一次，记录实际工具句柄并追踪至真实 terminal；partial checkpoint/观察超时不能用来重启。实际完成后才执行全会计、八控制、新补充接收者、全正常平仓及 all12 成本审计。每笔开仓核对 canonical closed1/2/3..10、独立 observed minute 0/原 QPS Float64 顺序、闭合推进/回撤/主动多数/极值 cap/所有原 regime、原种子 ATR/4h/daily parity。禁止继承 RG24 的 RangeValid/EMA收回/同向 body1/contraction/旧 close cap 结论。

自然归因只使用预先声明的原 ADX20 weak/strong、方向、四年及 normal exit attribution；全补充都包含。不按结果删组、反转交易或挑币。每个 helper 输出独占 wx/O_EXCL，失败仍保存。所有研究数据/账目/程序保留缓存，temp_strategy 仅完整 portable JSON；不发布或写库，不动生产、前端、配置或仓库测试文件。

## 冻结数据 identity

BTC c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457  
ETH 7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371  
SOL 64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818  
XRP 8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3  
funding-tail manifest 78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547（54 actual additions/7 exact overlap）。原 canonical interval volume 保持，不用分钟聚合替换闭合 interval amount。

## 当前实核 SHA-256

```text
7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa  conf/app.conf
ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad  service/backtest/engine.go
b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5  service/backtest/environment.go
1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0  service/backtest/indicator_cache.go
aef51d7cadaa5b561b4fdb893c4da3a2e089c3b69ecd47263c8553a0b35804b8  feature/strategy/line/technology.go
270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd  .agents/skills/custom-strategy/SKILL.md
c1b41c807c22ca1fcbbd618a3d0be1056c1d511e603f106768260acd863f21ba  temp_strategy/20261005-volume-impulse-shallow-pullback/00-volume-impulse-shallow-pullback-family.json
d1cec654ad2a0e5dc6556e383805ddce64598a73f09d1bbd3074f4d396a30b06  temp_strategy/20261005-volume-impulse-shallow-pullback/01-v29c-volume-impulse-shallow-pullback.json
21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go
a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go
fccca687b8f9b0c5d03e908395869da3b910d4665c311b6c86bd31a40245a68f  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg25_closed_signal_audit.go
7aa0df5ee54ce644a748f8234488a56f5c48425afc515ea1e7cab2977e4371f9  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg25_closed_signal_audit
f62c836895a34c222657b04a1cd1572181a7091425cc4bd031d40174be75e9fe  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg25_pattern_expr_checks_v2.go
4fc9ee692f8f36887973ad7d3c374edcab296918e365603341bb5119bcdc1237  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg25_accounting_audit.mjs
2e683f883338903d99f4e01d28c3bf75c79b3cf742b956e6c94674b118958ea8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg25_regime_attribution.cjs
35409b08a54d3d5b3948164dcbdb7d1e774aae7d9d810332ed2b5396b132a932  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg25_report_evidence.cjs
3ef3dbe5abbe292405ab835835e355bb744495e494730b4065b41e175d7b3d34  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg24_signal_receiver_bridge.go
de75f4431457ac2e0e7c4e95046adf494472bb32a2766ac899d0565b67cd13bd  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg24_receiver_overlay.json
6247af2b6c2473779616b880d4cfa1cafaaf2d3bc87ad6fcd0c6ee401c71cb78  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg13_close_signal_audit
ada03ce7f3b86c2196f4804442516e85d6b95add30cc2ea70391d558be68c24b  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg10_execution_cost_audit
9cc3e8dd9dcb873c66a8054ecb9505127f0f21a2618408936da8b89ed6df17ee  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg25-local-v29-snapshot.json
a9ddc556d22616ca4793481c151719a3baacf543c753c6a654dfbbc5d32b201e  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg25-expr-checks.json
2021e2eafdf0e95896852c4568795f2a0685401d56db35b63ddd915deb28ed70  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg25-expr-checks-v2.json
305baa839525031be9032310d648454c75573ba37b77efaeed47acde0c397bc3  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg25-api-rule-validation.json
7c7d2774bf94cadb2e2f2678c50b9422d83886e5d243235e61c3f52783875fc8  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg25-frontend-contract-checks.json
29ac88a5f785c2892a928831d1b2438ed7646c09a09018292dac545ae8316d37  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg25-portable-entry-identity-scan.json
b27cb840b0ee92bc165b64fed7b8e5574946d50cf7922f4e419b31aa78e6d04c  temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json
f4daf7caa9256fdb34326daf05503047895cee01b97b80ea371d3e194bf4647f  temp_strategy/20261004-body-confirmed-reclaim-followthrough/01-v29c-body-confirmed-reclaim-followthrough.json
478d310c3ff400bb0fa35d3696afb687f2036c7c0e5f80b0bd97ba2c52d7914a  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261004-rg21-development4-canonical-repaired-v2-funding-tail-v1.json
```

独立原始控制、首版浮点 fixture 失败、v2 修正及全部旧回测不覆盖。成功只以完整目标证据为准，不以阶段结束或全部 syntax-check 为准。

