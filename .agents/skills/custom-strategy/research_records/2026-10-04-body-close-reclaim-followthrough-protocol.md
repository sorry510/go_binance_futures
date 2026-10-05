# RG23：闭合收盘恢复确认，收益读取前冻结协议

## 决策和唯一改动

2026-10-04 北京时间冻结；RG22完整12run及所有后审计已结束且invalidated。不把主动/被动方向、按币选择或静态删账目当跨币优势。以原RG21整个对象为父，唯一改动两补充入口的严格实时价格确认：

- LONG `kline_1h.Close[0] > kline_1h.High[1]` → `kline_1h.Close[0] > kline_1h.Close[1]`
- SHORT `kline_1h.Close[0] < kline_1h.Low[1]` → `kline_1h.Close[0] < kline_1h.Close[1]`

其余RG21整段价量/closed body/日线与四小时判断、原0.35ATR极值锚定上限、QPS90%累计量门槛、previous reclaim抑制、所有指标、AF0基础对象/顺序和whole统一RG4退出不变。没有同时改成交量方向，没有新增MarketCondition，没有读形成中taker[0]。新条件弱于整根极值突破，可能增加误入，不预设净收益或频率改善。

## 证据和候选

旧RG21全223补充的零阈值几何诊断219有正同侧影线，四币平均0.308956/0.275622/0.222126/0.227515ATR。诊断只复核已有canonical/receiver完整审计与原账目几何，没有按PnL分组、threshold grid、删除组或读取未阅币，不是新的raw数据审计或更早可成交/盈利证明。诊断SHA `f17c45fc59b937d65b5be9d06cd4e7b23b4c84a4311218774a777538a87430b3`。

两完整候选在helper生成前即保存并永久保留，包含全部long/short/close_long/close_short：

- family `temp_strategy/20261004-body-close-reclaim-followthrough/00-body-close-reclaim-followthrough-family.json`，SHA `dbe12701cf25e705e4b1a2d2b88858dd3a75e359cfa11f2a85b9f58f6f2f824d`，version `2bdd0cdfbfeeacb44cbd37a9c1f409b261fa078e99e82270630c261b1c2eb94f`
- combo `temp_strategy/20261004-body-close-reclaim-followthrough/01-v29c-body-close-reclaim-followthrough.json`，SHA `7bb7ed3e951b7230045a7a33e5c13481f0af8cdf48b74d233eb2bc3964968eac`，version `7a4edb349b072fa8cf6a99d5200a41e0a77f9ba0a9a657b41f544ef98525a3a3`
- RG21父组合SHA `f4daf7caa9256fdb34326daf05503047895cee01b97b80ea371d3e194bf4647f`；AF0 SHA `b27cb840b0ee92bc165b64fed7b8e5574946d50cf7922f4e419b31aa78e6d04c`

本地v29只读repeatable-read实际0写：go_bn_test ID114、唯一准确名称，technology/strategy语义hash `0ae3b681d78d49884c11165fc70684ca8db52c1db28a4bd6eb77d554f06bfa00`/`3d43aea89d4dc56b29aeb3890bdf92028ab0bac54777a9d9238df46cda4f00e2`；仍与原冻结v29一致。本次snapshot SHA `98cab3d3e838c4655cf186c25c6494c1a93670868268a8031bd7792b2172dcda`。

## 原门槛及范围

原UTC2022-09-01含至2026-10-01不含49月1491天213周；每币至少192笔，四个完整Sep–Aug年度各净正，报告日历2023/2024/2025及2026Jan–Sep和额外2026-09，不代替全年度合同或称年度重新初始化收益。开发BTC/ETH/SOL/XRP全部，AF0/RG21/RG23组合各完整一次，共12run。未阅验证币AAVE/ATOM/ETC/LINK不读取，只有开发联合过线才冻结后检查资格/真实资金费tail并验证。

初始1000USDT/币、原当前可用现金10%保证金、8倍、外部门槛5/5、每侧0.0005费率和5bps不利滑点、真实历史资金费率及原缺mark分钟Close回退。精确结算mark与历史订单簿滑点证据仍pending：算术审计0失败也不能宣布“真实成本”完整达标。不得调整风险/资金费/成本/日期/币范围/周频/年度门槛来过线。

## 前置校验

实际Go/Expr7890通过/0失败，包含整个父程序仅改锚点、独立数值模型、零影线等价、正影线较早价格区间可达、原极值上限与当前累计量的联合、闭合实体/其他原条件和全统一确认退出矩阵。合成/本地有序模型不是private sequential、live/forward或盈利证明。Expr证据SHA `6dcc61ae985d692ce537e1507621fe7b67863d8d0819ac082e72b303d72e1ca3`。

退出矩阵（LONG/SHORT均覆盖空/错误/实际entry hash）：

| 场景 | 决策 |
| --- | --- |
| 单独ROI达到5/−5/16/28/−12，无方向确认 | false |
| ROI16与趋势失效，或动量失效并有反向活跃实体 | true |
| ROI−12与趋势/动量失效 | true |
| ROI28与动量失效或反向活跃实体 | true |
| 外部门槛±5与闭合4h弱ADX和当前反向突破闭合小时极值 | true |
| ROI≤−20灾难止损，无普通信号 | true，明确唯一无信号例外 |
| ROI在外部5/5之内 | 不获正常退出评估资格 |

真实前端TypeScript isolated VM两对象 issue=null/9enabled/fourtypes/shape通过，SHA `04beb4d8ebb8ab2aa08b0508dccbbb45d05de5639fac4fadab937b692a6fbc76`；限定320portable/735enabled-entry扫描0去空白精确重复，SHA `78901d99c3038b775c5ec5d1a56d6f34c58ed98496e78ce4497cf60f9a406b34`，不称全局语义/alpha新颖。opening真实build0，源SHA `ed8b9025eb1aaa794062aa95bd12f7c0759ca45c9b6a23e9c78ceb3d8c25cc86`，binarySHA `c7369357ebedad22a589740531ca71faecb5c0e43c529fbf4c197a5e54874e60`。不新增仓库测试文件。

API逐条校验实际句柄69071已terminal0：六启用规则分别code200/passfalse，名称/类型与冻结候选逐条精确配对。证据在cache results/20261004-rg23-api-rule-validation.json；只是当前快照编译/运行，不用固定mock判定真实仓位/历史盈利。不重poll已结束句柄。

## 原运行身份及后审计

原engine SHA `ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad`；environment `b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5`；indicator_cache `1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0`，保护app.conf SHA `7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa`，正式skill SHA `270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd`，均未改。原隔离funding-tail-copy/source身份不变，仅使用已验证完整canonical与真实tail。

完成全部12run后核对实际新账目数量（不预设为父组或RG22数量）及全会计、八共享AF0/RG21控制完整身份、全部实际补充closed-Close+body+all原条件开仓、whole实际正常退出、all12真实原成本算术和失败归属、零活动成交；旧strict High/Low通过/拒绝只作为预声明描述cohort，不静态反事实。再natural完整weak/strong/range/oldcap与旧extreme描述/年度报告。每次poll真实同句柄，不因partial或timeout重复主进程。

运行命令：

```text
rtk proxy go run /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go -conf conf/app.conf -strategy-files temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json,temp_strategy/20261004-body-confirmed-reclaim-followthrough/01-v29c-body-confirmed-reclaim-followthrough.json,temp_strategy/20261004-body-close-reclaim-followthrough/01-v29c-body-close-reclaim-followthrough.json -symbols BTCUSDT,ETHUSDT,SOLUSDT,XRPUSDT -start 2022-09-01 -end 2026-09-30 -execution-source public-archive -indicator-source public-archive -minute-repair verified-archive -cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1 -archive-cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2/archives -output /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261004-rg23-development4-canonical-repaired-v2-funding-tail-v1.json
```

不读App UI、不新增数据库写库/绑定/激活/下单、不改生产/前端/config、原failed候选保留。RG18此前批准的ID124入库不构成RG23授权或盈利资格。
