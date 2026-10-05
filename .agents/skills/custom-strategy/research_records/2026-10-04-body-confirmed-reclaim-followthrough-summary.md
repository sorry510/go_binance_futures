# RG21完整阶段总结：闭合实体确认不能满足频率与跨币年度稳定

## 当前结论

`invalidated`，不发布、不入库。原AF0/RG20/RG21×四开发币12完整49月顺序回测、8共享完整控制和全部审计结束：主92570、opening3005、closing54067、cost18826均actualterminal0；会计/natural/phase脚本actualexit0。四币净额正，但每币频率全部低于0.9，ETH/SOL/XRP还有负完整年度，ETH/XRP负日历年。BTC四完整年正不能代替所有币联合门槛。

原UTC2022-09-01inclusive至2026-10-01exclusive、1491天213周、最低每币192笔、8x/outer5与5/完整uniformRG4确认关闭、两侧fee0.0005/不利slip5bps/当前cash10%margin和真实funding及原分钟mark回退均保持。各年/侧/规则为全顺序原交易退出归因，不是独立逐年初始化收益。

## 四币完整结果

|币|笔数|每周频率|毛收益|手续费|资金费PnL|净收益|PF|最大回撤%|均持小时|
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
|BTCUSDT|148|0.694836|517.438610|131.749143|-20.969194|364.720273|1.271157|14.665779|30.582657|
|ETHUSDT|155|0.727700|490.556657|126.621501|-14.251686|349.683470|1.234711|17.167518|19.579247|
|SOLUSDT|179|0.840376|1051.042662|196.627842|-28.827883|825.586937|1.317664|23.770791|12.732588|
|XRPUSDT|151|0.708920|931.982430|156.568786|-6.847027|768.566617|1.428112|22.425993|13.350441|

|币|2022-09至2023-08|2023-09至2024-08|2024-09至2025-08|2025-09至2026-08|额外2026-09|去最佳5描述净额|
|---|---:|---:|---:|---:|---:|---:|
|BTCUSDT|7.589649|43.259039|12.087978|381.032978|-79.249372|63.909136|
|ETHUSDT|-105.478591|29.234646|192.359925|250.141688|-16.574198|-11.072828|
|SOLUSDT|-117.476553|498.075883|178.579163|278.364084|-11.955640|346.115761|
|XRPUSDT|277.373371|-198.710300|260.231795|395.849339|33.822413|272.190685|

|币|日历2023|日历2024|日历2025|2026Jan–Sep|
|---|---:|---:|---:|---:|
|BTCUSDT|131.328975|7.344246|69.984426|209.248902|
|ETHUSDT|46.608070|-13.285336|130.772102|317.974150|
|SOLUSDT|211.986426|166.711315|6.657672|397.158494|
|XRPUSDT|199.860634|203.077515|-187.731391|540.811967|

|币|RG20笔数→RG21|RG20净→RG21|联合结论|
|---|---|---|---|
|BTCUSDT|213→148|274.767523→364.720273|频率失败|
|ETHUSDT|226→155|936.049907→349.683470|频率失败、完整年度失败|
|SOLUSDT|224→179|988.100297→825.586937|频率失败、完整年度失败|
|XRPUSDT|205→151|444.565999→768.566617|频率失败、完整年度失败|

新增body条件不一定提高所有币收益：BTC/XRP改善而ETH/SOL净额下降。不是根据单币/少量最佳单选择保留该过滤，亦不按币或侧取消门槛。去最佳5/年度/侧/规则都是描述，不是删单后重撮合收益。

## 全实际逻辑与自然市场分组

RG21自身633=223补充+410组合内基础，不用633减独立AF0 429当补充。补充各币52/52/52/67笔，净BTC−180.036023、ETH−23.874590、SOL+820.937664、XRP+102.379876；BTC两侧毛净皆负，ETH合计毛+21.252441不足覆盖fee41.091751及funding−4.035280。不能称绿色/红色body就证明被动吸收有效。

自然weak/strong全部223actual paired：BTC weak−116.888732/strong−63.147291，ETH−12.045855/−11.828735，SOL+101.918785/+719.018879，XRP−71.045877/+173.425753。没有四币共同稳定盈利组；SOL strong四完整年及去最佳5描述正，XRP strong四完整年正但去最佳5负，仍不是通用release或筛组反事实。

RG20原477快照body同向219不是RG21的223实际补充：顺序占仓、基础机会、现金和退出会改变后续交易；必须完整重跑，而不是静态过滤历史219单并宣布收益。

## 真实核验与尚未完成门槛

- 12run/1930全部账目/8共享AF0-RG20控制snapshot与全部ledger/metrics/datasource身份exact，会计errors0，四原canonical数据hash及fundtailmanifest/54append/7overlap不变。
- Expr5654/0，真实frontend VM两issue=null/9enabled/四type，限定316portable723entry0exactdups；只证明合成与形状，不是全private-selector/live/API/forward或盈利。
- 实际223补充/17840闭合字段/+1115四小时/+669日线/+446ATR/+1338range/+669activity/+223strict，wholeprogram/canonical/original200input199closed/全原价量/新body0失败。223全部closedbody确认和receiver parity，最小signedbodyATR0.0057417178159178805；最低activity0.9000472889994341，最大Extreme锚advance0.3483429569843932。range收缩118/非105，旧CloseCap通过93/拒绝130仅描述，不用它们重建旧收益。
- 原正常关闭633/0forced-end、old505/added128/both0/added-only128，全部Position/ROI/outergate/whole程序0失败。普通ROI5与−5单独不平仓，确认分支有效；ROI<=−20唯一更深灾难例外。没有改成ROI-only。
- all12成本1930笔/4542fundingsettlement/1371原分钟mark回退/0零活动/0算术失败；自身633/1467结算/436回退/0失败，控制及自身失败原行均保留（当前0），最大qty误差4.093e−12/净误差3.411e−13是浮点级。
- 精确venue结算mark、历史order-book容量/实际slip和未阅AAVE/ATOM/ETC/LINK仍pending；原模型核验通过不是全部真实成本或泛化通过。开发联合失败，不提前消耗未阅验证币。

## 完整保留文件与身份

两完整候选temp_strategy/20261004-body-confirmed-reclaim-followthrough/：familySHAa071dab86f01d76f099e28309acdb5c2733bc1a98dc0d422a899c0a0ffb20da4/version2b83defff791cdd916b3eccea18aed304fa7a8527d9fd6c2ab966c7b33491786，comboSHAf4daf7caa9256fdb34326daf05503047895cee01b97b80ea371d3e194bf4647f/version77e817fbe930cd3772ae4a6b27b3497e87761a7680791029285a2e0785ace2fe。收益前protocolSHA4646f51f9d5e1e64379b077d5ac1d5421748596d05eefa0d529728d278e3d7d5。

结果缓存根/Users/zhz/Library/Caches/go-binance-strategy-research/results/：

|证据|文件|SHA256|
|---|---|---|
|study|20261004-rg21-development4-canonical-repaired-v2-funding-tail-v1.json|478d310c3ff400bb0fa35d3696afb687f2036c7c0e5f80b0bd97ba2c52d7914a|
|accounting|20261004-rg21-accounting-summary.json|5c1546b9f6a4c558a4fa9f6c973ce2476fe6a6f140382ae44dc5ce32dab02ce7|
|expr|20261004-rg21-expr-checks.json|15c6a3a201b78f00ab8ec1f976dfcb76112a0d9fa2a4810c16da6c34274f55d6|
|opening|20261004-rg21-canonical-actual-seed-open-signal-audit.json|5229ce05ea3e7b8e33af25e8775a26996a61e0b8db650908cfbc90f2e503256d|
|closing|20261004-rg21-original-position-close-signal-audit.json|7f94c02cea4ef479d351cc1a3e910cce1f7eb62ebd8639ae1ec2a1b5d08ba72b|
|costs|20261004-rg21-execution-original-cost-fallback-audit.json|6faf0ac9e96aabf895d1084601eed3c8014bc061e49cb9aa54db89e8eaf901a7|
|natural|20261004-rg21-natural-entry-regime-attribution.json|ca55a9bed40f7149611a3704c34edc3f266538577e1fd06d6946d7b037e19e93|

phase文件20261004-rg21-phase-evidence-summary.json。原code/source/conf/正式skill与所有旧失败、用户dirty/metrics修订、待评审技能草案保留，不覆盖、不删除或晋级。图谱InitParseEnv片段旧行号259..335返回MFI/OBV而非函数，已按不足回退实际756起文件核对；没据旧片段改系统。

## 下一研究维度与权限

下一条RG22尚未生成、冻结/校验、主启动或读收益。拟只替换闭合主动成交量多数方向：LONG buyQuote*2>quote，SHORT buyQuote*2<quote，保留交易方向/闭合body/原strict sweep-reclaim-currentcross/cap/activity/基础/wholeclose/9指标/所有风险和门槛。这是主动成交与价格同向共振的独立备选假设，不是把亏损交易反向；历史净亏不证明另一flow谓词盈利，也不把所有市场状态强制开仓。先完整配置防精确重复、独立numeric合法quote与body/cap/regime组合矩阵、旧wholeprogram仅flow替换和收益前协议，再原完整撮合。若频率/年/成本/币不达标仍淘汰，不按币挑参数。

本地v29 ID114最新只读语义exact，RG18 ID124前一独立写库授权已fresh回查。RG21研究无新增DB写、分配/启用/下单、App/生产/前端/conf修改、新仓库_test.go、材料删除或新委派。Skill指导完整原合同与审计，现有成本/代数窄SkillMax草案仍trusted:false/pending而非formal提升，不为每轮重复建技能。

当前目标未完成，实际goal active。本turn完成完整新候选/全12回测/审计与总结，属于progress，不自行complete/paused/blocked；下次用量恢复前先阶段总结再核实际状态，所有本轮句柄已结束不要再poll。

