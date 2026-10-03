# PV2 即时主动成交恢复：冻结协议

运行后状态：12次回测完成，但独立当前成交比例审计发现178/641笔补充入口违反预期信号，已定位优化缓存未刷新Taker[0]。原收益仅作为未修复运行记录保留，不代表预期PV2策略效果；待授权限定修复及同条件重跑。详见2026-10-03-pullback-active-flow-summary.md。

## 单变量与研究边界

2026-10-03 北京时间13:55左右、观察PV2收益之前冻结。开发结果PV1已经观察，因此本轮是开发期机制检验，不是未见样本验证。此前216个完整JSON、472个long/short程序只读扫描无TakerBuyRatio[0]入口；v42用它作平仓反向冲量确认，不等于本轮回调恢复入口。

- 保留AF0/v29C基础入口、PV1补充族完整条件、9个技术指标参数、所有区间与统一AF0平仓对象。
- PV2补充long仅追加 `kline_1h.TakerBuyRatio[0] >= 0.55`，short仅追加 `kline_1h.TakerBuyRatio[0] <= 0.45`；不扫描其他阈值、不改变等待时间/成交额下限/回调深度/风险/方向。
- PV1已有 `Amount[0]>0` 与闭合小时Qps[1]/[2]>0检查，以及两小时逆势实体、继续回调、Qps[1]<Qps[2]、4h趋势/日线反向过滤、资金费与0.35ATR追价上限。PV2不修改它们。
- 比例为主动买入报价成交额/总报价成交额；LONG至少55%主动买入，SHORT至多45%主动买入（对应至少55%主动卖出）。总量衰减与方向占比不是相同信息，但成交占比不是订单簿失衡/因果驱动证明，也可能在小时早期低成交量时噪声较大。
- `GetLineFloatValues`及回测klinePriceSeries按同一报价口径计算；BuildMinuteClose的形成小时overlay只累积截至观察分钟的quote/taker quote，不使用未来闭合小时占比。闭合[1]/[2]与实时[0]语义不变。
- 前端代码编辑器可以发送任意字符串expr，但当前autocomplete没有TakerBuyRatio建议；本轮不扩展前端或指标实现，不把缺少提示误称为后端不支持。
- helper入口顺序保持base LONG → supplement LONG → base SHORT → supplement SHORT，明确 `-exit-source base`，不使用hash绑定或当前状态退出守卫。独立族与组合close对象必须逐字段等于AF0；基础入口/technology也保持一致。完整重撮合，禁止对旧账本静态过滤后计算“策略盈利”。

## 固定数据与判定

- 开发BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT。AF0、PV1和PV2各四币共12次完整重跑，以前两组逐笔账本复现作为控制。
- UTC2022-09-01..2026-08-31（1461日）。每币≥0.9次/周（至少188笔）、每币总净收益>0、四个Sep–Aug年度都>0、跨币验证全部通过；逐币列费用/资金费/PF/回撤/方向集中度。不得只选盈利币或放宽原门槛。
- 未观察收益验证名单仍AAVEUSDT/ATOMUSDT/ETCUSDT/LINKUSDT，完整日期数据资格待核查；开发失败不请求它们的收益。
- 相同canonical repaired v2不可变数据；backtest_engine_v7、standard_1m；初始1000USDT/币、保证金当前可用现金10%复利、8倍、双边各0.0005费率、双边各5bps滑点、实际历史资金费、外部profit/loss5/5；分钟收盘观察、下一分钟开盘成交。
- 统一平仓：ROI≥16且趋势失败或动量与反向冲量联合确认；ROI≤-12且趋势/动量失败；ROI≥28且动量/反向冲量确认；ROI≤-20为灾难止损例外。外部门槛5/5只是资格，不能称为正负5%自动止盈止损。
- 校验实际JSON/hash、编译及LONG/SHORT可达性、比例边界与独立性、零成交/缺少资金费、平仓矩阵及真实/空/错误入口hash不影响uniform closes。局部有序selector验证不是私有引擎/API/前向端到端验证。
- 完整账本会计 `net=gross-fees+funding`、年度/方向/数量相符、同源hash、实际entry/exit分钟活动检查。标准引擎未模拟订单簿深度及全部交易所步长，分钟活动干净不是实盘成交保证。

两个完整新JSON立即保留在temp_strategy/20261003-pullback-active-flow/；AF0/PV1对照复用已冻结完整文件、不覆盖。结果与检查器在研究缓存。禁止App、配置/数据库/生产/前端修改、策略启用及仓库测试文件。无发布策略前goal保持active。

## 回测前文件冻结

2026-10-03北京时间13:55–13:57生成并保存，尚未观察PV2回报：

| 文件 | SHA256 |
| --- | --- |
| 00-live-active-flow-family.json | 4cbef8c3a8ee5e435c818604bbe58dd51b652ebc89edd3261bd05374e667df11 |
| 01-v29c-live-active-flow-pullback.json | a5ab4f9197158bc56c6673674f6e343ff76f460a40680b6b0919f23f8265a6ca |

AF0基础文件实际为20261003-adverse-flow/01-v29c-observed-activity-control.json，SHA b27cb840b0ee92bc165b64fed7b8e5574946d50cf7922f4e419b31aa78e6d04c；PV1组合SHA4a5f680f1fad1a453d4947b98fe76e4dea4d809c8f083762aeb20ede9f6d01ee。首次combine命令误写00序号后失败，核对旧账本path修正为01，文件未被覆盖。protocol一次无实际变化的patch定位失败，没有影响已保存冻结内容。

新增信息维度不代表与价格统计独立。比例为自小时开始累计报价成交，不是突破瞬间流量或资金净流入；低流量/单筆成交不稳定及各小时内阶段不同仍未限制。官方USD-M归档字段说明见https://github.com/binance/binance-public-data#klines-1；实际字段/计算路径以当前项目源码为准。
