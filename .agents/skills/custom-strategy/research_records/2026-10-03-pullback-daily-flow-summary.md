# PV4阶段总结：日线同向不能弥补补充入口弱优势

## 当前结论

12次AF0/PV3/PV4四年回测全部完成；8个AF0/PV3原控制的逐笔账本、metrics、官方年度、源hash与上一轮完整复现。PV4只是PV3两侧追加闭合日线DMI同向，全部其他条件/指标/原关闭对象保持冻结。四币频率均通过，但BTC/ETH仍净亏，四币均有亏损年度；未满足每币≥0.9次/周、真实成本、四年稳定、跨币泛化联合目标。没有合格发布策略，不观察冻结未见币收益、不发布/写库/启用。

## 1. 同条件结果

UTC2022-09-01..2026-08-31，1461日；standard v7/1m、同canonical repaired v2及四币hash。初始1000USDT/币、现金10%保证金复利、8倍、双边各0.0005与5bps、实际资金费、外部5/5；188笔/币频率门槛不改。全部金额USDT。

| 币 | 笔数 | 次/周 | 毛额 | 费用 | 资金费 | 净额 | 净PF | 最大回撤% |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| BTC | 252 | 1.207 | 31.399 | 170.637 | -16.601 | -155.838 | 0.929 | 32.48 |
| ETH | 272 | 1.303 | 160.775 | 188.204 | -14.397 | -41.827 | 0.984 | 40.16 |
| SOL | 304 | 1.457 | 746.161 | 317.789 | -32.445 | 395.928 | 1.087 | 36.81 |
| XRP | 240 | 1.150 | 529.574 | 191.219 | -2.518 | 335.837 | 1.126 | 43.42 |

年度按退出时刻归属Sep–Aug官方窗口；辅助entry年度归因单独标识，不混用：

| 币 | 2022–23 | 2023–24 | 2024–25 | 2025–26 | LONG净额 | SHORT净额 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| BTC | -139.571 | -71.826 | -34.219 | 89.778 | 40.998 | -196.837 |
| ETH | -134.757 | -30.231 | -32.237 | 155.398 | 175.612 | -217.439 |
| SOL | 92.274 | 420.808 | -118.486 | 1.332 | 238.334 | 157.593 |
| XRP | -60.881 | -250.724 | 359.026 | 288.416 | 315.029 | 20.809 |

相对完整PV3：BTC净额恶化28.950，ETH/SOL/XRP改善28.374/55.849/111.049；这仍不是跨币稳定优势。SOL/XRP去最佳5单转亏-96.312/-80.352，回撤不小；不得选这两币或合计收益替代原门槛。

描述性补充族LONG/SHORT净额：BTC-202.640/-323.333、ETH86.046/-328.272、SOL20.863/161.038、XRP-86.961/-177.691。日线同向只使总笔数270→252、285→272、325→304、255→240，未改变补充短侧在BTC/ETH的弱优势事实；同仓/复利与被阻塞基础机会也变，不能静态删除亏损补充单推导新盈利。日线DMI同向是原v29已有原子，不是新增优势证明。

## 2. 实现与信号核对

- 真实Go/Expr1472/0：PV3→PV4整个raw对象只有新名称与两追加条件，原technology/基础入口/关闭对象完整身份；闭合DI同向/反向/相等/0/epsilon，当前[0]相反不影响，[1]不要求额外ADX强度；既有closed ratio与0-index差异、零量拒绝、全部规则双向编译/运行、准确本地首条/匹配数/禁用、正常ROI-only false/确认close true/灾难true/hash无关。局部有序模型不是引擎私有selector或API/forward证明。
- 全账本755筆补充入口（BTC185/ETH199/SOL198/XRP173）独立验证portable规则hash与方向、entry_time-1 signal分钟、canonical闭合两小时连续/ratio改善/量衰减/逆势实体及价格恢复，0失败。
- 仓库外只读Go overlay桥接原newHistoricalEnvironment/BuildMinuteClose，每笔用原9个indicator配置重建真实历史输入，核对canonical闭合日线[1]的Open/Close/时间、日线DI同向，并编译/运行其精确完整portable开仓程序：755笔全部true。没有订单、收益重跑或生产改动；fresh snapshot不是完整顺序回测cache逐位等价/私有ordered selector/live/forward/订单簿证据。
- PV3真实3833分钟闭合taker receiver检查在两个保护源码/同dataset hash不变后沿用；current taker[0]缺口仍保留，不称已修复，不把本轮closed实验当PV2修复版。
- 全1068笔entry/exit分钟活动审计zero_liquidity_fill=0，仍不是交易所步长/订单簿可成交保证。
- 全12次2620笔会计恒等逐笔误差0；净额累计误差最大1.6201e-12，fee/funding/gross/官方年度/方向/净PF/全日历频率/计数相符；8原控制完整复现。

完整raw/检查/审计：缓存results/20261003-pv4-{development4-canonical-repaired-v2,expr-checks,closed-signal-receiver-audit,liquidity-audit,accounting-summary}.json，0600。临时程序及overlay只在verification；虚拟service/backtest/pv4_signal_receiver_diagnostic_bridge.go在真实仓库不存在，没有新增仓库测试文件。

诊断源码SHA：原environment.go=b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5，indicator_cache.go=1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0；桥接e70b905120dbd2a426de9d0500f343ce1aea08c9c758624a9a36aca0d5ddd918、overlay29cb422be7191d7b308b2218e210e616f86c04cf2c06915b5214635e5fe712bc、主audit298cbbe990856420e2ab203594353d5a2c00e030fb1e5d7d6b621b88add31b73、原loader副本a96e8d07cbf54846b435fbab672f91df7416bcf243e7e925620f1583d8f29f2d。

关闭矩阵两侧不变：普通ROI只跨5/-5/16/28/-12且无确认false；≥16与趋势失败或动量+反向冲量true；≤-12与趋势或动量失败true；≥28与动量或反向冲量true；只有≤-20灾难例外可独立true。外部5/5是资格，自定义AutoStopOrder仍false。

## 3. 文件、数据库与续接

两个完整候选在temp_strategy/20261003-pullback-daily-flow/，family SHAebe1cbb4ff9fec951301f5692be6ea871ed17e1df47429f53c1512f796c65fee、组合SHA02b9e4d38cc5c7116b0f4a396387a534a6b120eb30c35de0f0ae34b38df40cbd，versionc00b2a7a6e4e8a9dfeb9f12eb190d40f55e0165b7c8035dcd3c65463f9426d8a。新/旧失败JSON全部保留，未复制发布目录。

只读ARM元数据/v29复查截止北京时间18:39:09.368/18:39:11.227/18:39:13.024；17/17/17模板、221/217/6结果，v29 portable完整语义仍相同。不是完整结果内容重审计，相同数量不能证明所有行原样；上一份444行forward总结仍是较早快照、只支持insufficient evidence，不与本轮8倍/5/5历史净额合并。

本轮原回测6463、Expr67146、receiver88221、活动84367、snapshot95825均terminal exit0，无已知root活任务；下次核查进程，不复用旧句柄或无故重跑。两个前轮验证agent因用量失败，不再次授权/启动新agent，root自行完成。

下一轮停止叠加日线方向过滤，转向回调恢复时的已观察形成小时报价成交活跃度确认：在一组冻结对照只追加Qps[0]与既有闭合小时平均活跃度的比较，不读错误taker[0]，不变[0]实时语义、风险、成本、全门槛；新JSON/协议先保存，核对历史重复/可达性/实际接收/数据source不同量定义，再完整重撮合。这是待检验机制，不从本轮旧账本过滤推导预期盈利。全部goal仍未达成，保持active，不因困难或局部生产修复审批误标blocked。

现在可用的是两轮完整实验JSON、固定数据回测与实现/信号复現证据和续接报告；没有可发布盈利策略。未操作App/写库/启用，app.conf及保护生产源码hash不变，技能1.0.7/trusted:false保持。
