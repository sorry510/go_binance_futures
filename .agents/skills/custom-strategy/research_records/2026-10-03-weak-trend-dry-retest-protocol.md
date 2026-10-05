# RG0：弱趋势缩量二次试探（收益读取前冻结）

2026-10-03北京时间20:09:03生成并保存两完整JSON，20:10:11.259核对并冻结SHA；RG0收益尚未计算/读取。此前PV5已完成24次，不能把其失败静态筛单当新回测，也不重复启动已terminal句柄。上一轮是progress：新候选、完整回测、真实资金费数据与接收者/会计审计及恢复检查点均完成。

## 单一机制与旧试验区别

原AF0 v29C趋势基础保留，另加一对4h闭合ADX<20的弱趋势入口。LONG：倒数第二个闭合1h先向下刷新之前8小时低点，随后1h以更少quote成交额再次试探同一/更低低点，却收盘转强并改善闭合主动买入占比；当前已观察价格收复后者高点0.10 ATR、且不超过0.35 ATR才买。SHORT完全方向对称。

这只是假设：较低的总成交活动与两侧主动成交占比改善可能标记二次试探后的恢复，不把它直接等同真正卖压/买压衰竭或盈利。改善占比并不要求越过0.5，也不声称主动成交额本身增加。低ADX只是趋势强度弱，不保证价格在稳定区间；ADX20、8个闭合小时及0.10/0.35 ATR沿用已有合同窗口/阈值，无参数搜索。

已检查v31/v41/v64/v65/v69/v72/v80/v90/v125：它们分别有单根wick/sweep、两根先收盘突破再回收、climax midpoint、三小时price/flow或OBV背离、1-2-3结构；RG0采用“二次更低/更高试探+闭合缩量+闭合占比改善+live恢复”的组合，不是仅给PV5再追加一个过滤器。385个旧portable入口去空白全程序扫描无精确匹配；粗略二次极值+缩量原子也未命中。不是任意语义等价去重或alpha新颖性证明。

## 完整文件与身份

| 文件（temp_strategy/20261003-weak-trend-dry-retest/） | SHA-256 | snapshot version |
| --- | --- | --- |
| 00-weak-trend-dry-retest-family.json | d5de21244c66abee25565721b79734843efcee1abcfc6a3d5e4f8f1d4101a919 | 57c739ae6bdb98be293b69cfc1975f24311794a25673ca351bdbbf88e01dd9aa |
| 01-v29c-weak-trend-dry-retest.json | a3e8981f88aea6c04acfe48c6fe70c756b82c7f983c4c744936662435b8aa07d | 60674b2d08f0a5d2c89b5b5176a66a3e00108ab168bdec2a39c5d239923a5ac0 |

族4规则、组合6规则，均含LONG/SHORT与两侧确认平仓。原9指标、基础两侧入口和两个统一关闭对象逐字段一致；组合顺序base LONG / supplement LONG / base SHORT / supplement SHORT。不新增指标/周期或生产变量，不读forming taker[0]，不绑定OpenStrategyHash。闭合ratio[1]/[2]与QPS[1]/[2]源必须独立核对；价格与总量[0]有意保留。

真实Go/Expr临时验证690项0失败，包括弱ADX闭合边界、初始极值/二次试探/收盘恢复、缩量/零量/ratio方向、forming ratio不影响入口、live缓冲和不追价边界、价格缩放、禁用/局部有序匹配、原基础与关闭对象身份和LONG/SHORT ROI/hash矩阵。结果`results/20261003-rg0-expr-checks.json`；helper `verification/rg0_expr_checks.go` SHA f7b1680b5e42ec179a9d2d63dc6fcca9d40cabe138396a4fe622725a015d4067。不是真实private selector、整段顺序cache、API/live/forward或盈利证明。

## 完整研究、成本和门槛

- AF0、PV5、RG0三个组合 × BTC/ETH/SOL/XRP，各从头完整49月重新撮合，共12次。族文件是完整便携机制对照，保留但本轮没有单独回测；不得冒称三个组合之外另有族收益。
- UTC2022-09-01T00:00Z（含）至2026-10-01T00:00Z（不含）；1491天/213周，每币至少192笔。保留四个完整Sep–Aug年度及新增2026-09，各日历2023/24/25/2026 Jan–Sep单列；45月trade cohort不等于从1000重新开跑独立收益率。
- 初始1000 USDT/币，当前现金10%保证金复利，8倍，双边各0.0005手续费与5bps滑点、真实历史资金费，外部profit/loss5/5。普通+5/-5及+16/+28/-12到价alone仍false；正常退出必须市场确认，唯一ROI-only例外为-20灾难止损。原退出不因当前regime或开仓族悄然更换。
- 要同时通过每币>=0.9次/周、净正、四完整年稳定与跨币泛化。开发失败不获取/挑选未见验证币收益；冻结AAVE/ATOM/ETC/LINK，完整历史资格仍待核对。开发年份已多轮观察，不能冒称未触碰时间holdout；仅参数在本候选本轮收益前冻结。
- 两个已冻结控制必须与PV5上一轮49月完整逐笔/metrics/年度/数据source hash重现，不能静态删除/相加被补充入口改变的单仓复利路径。

## 来源和执行保护

使用上轮已独立验证全部旧bar/资金费前缀一致的`public-canonical-repaired-v2-funding-tail-v1`，仍原standard引擎v7/1m。真实资金费suffix固定manifest SHA78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547；隔离data helper SHAa340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63，replay复制与原文件同SHA21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8。仅研究数据采集身份，不修引擎current-taker缺口，不更新生产/DB/config，缓存命中仍校验完整身份/coverage。所有三组同源同引擎。

四币预期hash BTC c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457 / ETH 7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371 / SOL 64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818 / XRP 8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。若变化停止解释对照，不把新hash结果和旧控制混合。

官方USD-M K线档案包含quote asset volume与taker buy quote volume，checksum校验文件完整性，不保证每值正确；现有独立canonical成交量差异/已证明v2价格纠错政策全部保留。[Binance官方档案说明](https://github.com/binance/binance-public-data/blob/master/README.md?plain=1)。本轮官方Kline REST旧文档路径未能获取，不引用其不可读内容；直接以可读官方档案和当前本地转换/receiver为准。

三库本轮仅程序只读元数据/v29，截止北京时间20:03:32.068 /20:03:34.744 /20:03:36.783，模板17/17/17，结果221/217/6；v29 technology/strategy与上轮完全一致。相同count不证明所有结果行原样，此轮不是444行forward新全量分析。配置/原指标缓存/环境/live转换SHA与上轮一致；不修改app.conf、前端/生产、不操作App、不写库/分配/交易启用，不新增仓库测试文件。

## 审计与恢复

所有实际RG0补充入口核对标准信号时间entry_time−1、闭合[1]/[2]和前8根[3:11]canonical极值、quote/ratio、弱4hADX接收与live价格ATR边界，原9指标fresh BuildMinuteClose完整程序必须true。独立只聚合信号分钟前当前小时quote，核对当前Amount，保留canonical闭合值；fresh不代表整段顺序cache/私有selector/live端到端。会计、年度、side/族集中度与实际entry/exit分钟非零量审计，分钟活动不等于订单簿容量/成交保证。

完整结果/辅助程序仍在仓库外缓存results/verification，所有失败JSON在temp_strategy原位保留；主输出`20261003-rg0-development4-canonical-repaired-v2-funding-tail-v1.json`。quota或恢复前先读resume-checkpoint的最新阶段并用真实句柄/进程核对；live observation timeout不等于终止，不重复启动活回测。未满足完整联合门槛不发布、不标goal complete/blocked/自行paused。
