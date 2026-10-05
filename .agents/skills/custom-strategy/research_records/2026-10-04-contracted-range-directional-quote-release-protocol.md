# RG13 相邻闭合四小时范围收缩后主动 quote 同向释放：收益前冻结协议

冻结时间：2026-10-04 北京时间09:14:43之后、启动本轮完整收益回测之前。RG13收益尚未读取。协议和完整JSON先保存；之后不根据逐币/方向/年份收益改参数。

## 当前结论和单维机制

RG12强趋势回撤量价交接失败：XRP频率0.854次/周、SOL净亏、四币有负完整Sep–Aug年。538笔旧补充入口的closed EMA20位置诊断也不支持简单追加恢复过滤：两种自然分组均四币负；这些只是已观察账目归因，不是新初始1000策略收益。

本轮把两条RG12补充完整开仓程序整体替换为一个预声明的价格区间收缩释放机制，不调整基础入口、指标、关闭、风险、日期、币种、成本和资格门槛。不是对失败分组反向/删侧或阈值搜索。旧v11/v89/压缩假突破v2/v54配置已只读，仅作语法和机制比较，未读其收益；不宣称压缩突破概念从未存在。

- recent四闭合小时[2:6]最高最低范围严格大于0并小于等长older四小时[6:10]范围；older也必须大于0。相等拒绝。
- latest closed1报价成交额高于之前八小时[2:10]均值，均值正且逐根非负；closed1主动买quote在0..总quote内。LONG严格主动买多数，SHORT严格主动卖多数，50%等值拒绝。
- closed1收盘严格突破recent高/低边缘0.10×ATR1；closed2此前不得已突破它自身[3:7]四小时边缘0.10×ATR2。ATR1/2均正，首次释放而不是反复追涨杀跌。
- closed4h ADX<20允许局部弱趋势突破；ADX>=20必须同向closed EMA20/50和DI严格一致。强势相等或不一致拒绝。
- 原closed daily ADX>=20且DI逆向排除、同向body0.15..2.0ATR、forming价格[-0.15,+0.35]ATR范围及forming总quote正保持。
- 不使用forming TakerBuyAmount/Ratio[0]、过去主动quote累计/压力交接、过去八小时净价差、外部市场标签、新指标/间隔、币种名称或入口hash路由。
- 原AF0基础两个完整入口、原9启用配置和whole统一RG4 close_long/close_short对象完全保持。独立入口族4规则和组合6规则都完整保存；本轮历史收益测试的是组合，不是独立入口族的独立收益。

## 冻结文件身份

完整JSON路径：temp_strategy/20261004-contracted-range-directional-quote-release/
- 00-contracted-range-directional-quote-release-family.json：SHA256 4258b05982c8d2703f9d03373b9007da3b0ecce1e9c20efad966de5a71262ac7；version 611e2fa6b5469f6291e31ba81d49eebce61cbf2a00febf756ff1fb2c25e44c41。
- 01-v29c-contracted-range-directional-quote-release.json：SHA256 a1eb5a1e0080a8e75e9c9da199a4e055d8ab759904ab34875f2316dd49b3cae5；version bea53f9acfc843d4007effa54d358ad77a3c825df6ff659f9e4111aa40ff5f1f。
- LONG全规则SHA256 6d5070fecf9c0bc998163c0c77c79403c0c03c6c447197f94cd8ae5af8c2f471；SHORT a1b0d7dbbb9547812caae1fc08c75c58eacba615985a84da20b1f3660d92b517。

AF0：temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json，SHA b27cb840b0ee92bc165b64fed7b8e5574946d50cf7922f4e419b31aa78e6d04c，version 59287750973dad3fa07f76f7e991f7b788a62960fc185b838cc49ad23ffddb68。
RG12：temp_strategy/20261004-strong-trend-pullback-pressure-handoff/01-v29c-strong-trend-pullback-pressure-handoff.json，SHA ffb7b20784944fb14f38e835d11fa01827c1c1b9f3b4e0f7e60ed4bb3971d5fb，version 22ec73f30484962cbbedd3a43d232645ffa27039dda82c894d67f41e65e0d13e。
旧完整RG12结果用于复现这两个共享对照，不重做独立旧研究。

## 日期、数据、风险和成本固定

UTC2022-09-01含至2026-10-01不含，1491天/213周/49月。按BTCUSDT、ETHUSDT、SOLUSDT、XRPUSDT依次运行AF0/RG12/RG13，12完整run。四币是已观察开发集；AAVE/ATOM/ETC/LINK的收益未读，不能把开发结果当独立holdout。

原backtest_engine_v7、standard_1m，observed分钟Close决策→next分钟Open，双向每次5bps不利滑点；双边费率0.0005，真实资金费率及结算时间；初始1000，当前现金max(cash,0)×0.1保证金×8杠杆复利；外部门槛profit/loss5/5，AutoStopOrder=false。原gross/current-mark分母ROI，不能等同净保证金收益。
原full AF0关闭的16/-12/28/-20和统一RG4弱趋势ROI>=5或<=-5、当前价反穿closed1低/高继续同时应用所有入口，不改为ROI-only，不删除正常关闭。

独立canonical repaired-v2 funding-tail-v1缓存：
/Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1
共享已验证archives：
/Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2/archives
execution/indicator都public-archive，minute-repair verified-archive。
数据SHA：BTC c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457；ETH 7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371；SOL 64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818；XRP 8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。
funding tail原54新增/7重叠一致，manifest SHA78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547。
原缺结算mark用该分钟Close回退必须逐笔统计；即使算术全通过，仍不是精确资金费结算成本通过，资格pending，不造价/删资金费/降成本。

联合发布门槛不变：每币>=0.9次/周（本窗口至少192次）、真正可验证成本、四完整Sep–Aug年稳定且逐币跨币泛化；额外分列完整日历年/2026部分/额外Sep。任何关键失败即不发布，不挑侧/币/年份/参数。best-five剔除仅静态归因，不是删除后的反事实重回测收益。

## 已实际完成的收益前检查

- rg13_expr_checks.go实际terminal exit0：4958 passed/0 failed。实际Go结构和Expr编译执行，两侧强弱ADX×EMA×DI/forming反向、4vs4相对收缩/equality、latest/prior释放边缘、合法quote及无forming/history taker依赖、ATR1/2、body/live、disabled入口、AF0基础/whole统一RG4关闭和hash无路由等矩阵。是合成及本地规则顺序模型，不是private selector/历史全cache/API/live/forward/独立数学/盈利证明。
  输出SHA9cf5a104f6af0cca1f60c986bd418811aa75cdc2deaa56e0d4525a24563731a7。
- 当前真实前端utils/technology.ts用已安装TypeScript转译后隔离Node VM调用，两JSON issue=null/9启用/rule shape和四类型均通过。输出SHA7af13d4aaee1da3e33ffff437f5537ed9f344d844b176008080b18c716e930c5；无UI/build/API或前端编辑。
- 配置限定完整入口whitespace-normalized身份扫描300文件/677启用入口/0重复；排除research路径、诊断名字、>128KiB文件，自身两文件不算重复，不是语义/alpha/global证明。没有读其它研究收益。输出SHA2966727f729673634fd400952a96e0b2937712fc0f45b8723cf098cdea6ab602。
- 开/关audit helper已实际go build exit0，收益输出还未运行审计，不能把build称审计完成。

## 收益后预声明核验

1. rg13_accounting_audit.mjs重建12run逐笔quantity/fill双fee/净=毛−fee+funding、规则版本/四币dataSHA/tail/config/年度/side；8共享AF0/RG12完整trades/metrics/source除cache_hit必须重现。
2. rg13_closed_signal_audit.go针对所有RG13真实补充：entry−1 fresh original200input199closed环境；canonical[1..10]全部80字段、ATR1/2、4h ADX/DI/EMA20/50、dailyADX/DI；实际recent/older/previous六边缘、相对收缩/首次释放、closed合法quote多数/volume、weak-or-strong regime、daily/body/live全部核验。不沿用旧RG12八小时反向/price pressure/handoff逻辑。无selector/full顺序/live/forward结论。
3. rg13_close_signal_audit.go针对所有自己正常关闭：原Position/当前mark分母gross ROI/outergate，full原close和added RG4，两分支独立标记，强制end单列，不按entry hash筛掉关闭。
4. 原通用rg10_execution_cost_audit实际执行独立验证12run current cash复利/next-open滑点/双fee/真实资金费事件/mark回退和无零活动撮合；自身和全对照覆盖分列。算术正确不自动消除精确mark缺失。
5. 完整失败JSON/协议/输出保存，不写入发布strategy_templates目录/数据库、不分配、不启用或下单，不为了过门槛修改数据或参数。

## ARM来源更新和保护

本轮通过选定app.conf注释arm配置、whitelist、read-only repeatable-read程序完成元数据快照22785 actualterminal exit0：
go_binance as_of1791076078917，17templates/222results/v29单一；oracle1 as_of1791076080875，17/219；oracle2 as_of1791076082660，17/8。每库不同截止，不是同步分布式快照，也不是449条新forward内容分析。
新v29导出SHA b511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0，byteexact前轮；元数据SHA6bbda11694d76d4800706254500ebd2174805e937c48ee6d0d292b0ee9d7462d。

保护SHA：conf/app.conf 7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa；engine ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad；environment b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5；indicator_cache 1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0；正式custom-strategy b36ba1e915711e54aab4e5ca3bb3567be95e737336d1c2af6e27e12b54a82f77。
所有验证桥只通过外部overlay虚拟加入，生产目录不放新文件；无新仓库测试、App/UI、生产/前端/config/DB变更。其他dirty研究资料保持。

辅助源码SHA已在启动前工具输出记录：Expr d4f2472775302294b9358c7fec9ee80bfe97c8955e94fcdbfef896cd59427b43；开41b637aab8d03bda755a9d4d9b98bf86246cccc5223c441249e8ed1229a2e190；bridge8aa504d7eb079a0cc7f73542bc626866c1a0026213e50493e3598a8af51315c9；关ad4f2f4363a16dc1ffc8cebbed2cb641ce673da468337e3b8fd9ac2228f2d87d；account6b0ae836df9c776b99c074ab146b83b1c0f1d12c9a21ca0e81d1b75e7254333a；overlayb705eeebf7649c01a05d025027b55c50cd04b38955b98f0a47a71861acf804d2；preflight588db53a7b2eba920f6157e7d1449e22007e078833a5a604d6d229feee2c24da。
主/数据copy仍21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8 / a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63。这里只读固定原实现，没有修forming主动quote0缺陷；新策略不依赖该字段。
