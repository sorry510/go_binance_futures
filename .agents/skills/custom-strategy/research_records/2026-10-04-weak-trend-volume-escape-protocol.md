# RG2：弱趋势首次放量区间脱离（完整收益前协议）

2026-10-04北京时间00:04之后续接同一active目标，不重复已完成的RG1主研究。上一阶段12完整run、1897笔和全部会计/信号/原成本审计已完成，但RG1频率、SOL净额和年度稳定性失败。当前没有合格策略；用户要求在用量恢复前先给阶段总结，检查点继续维护。

## 单一机制与冻结身份

两个完整候选于2026-10-03 23:53:19前保存，23:57:52.599核对冻结身份。没有读取RG2历史利润；本协议在首次主回测前保存，不能称协议早于生成或合成检查。

| 文件（temp_strategy/20261003-weak-trend-volume-escape/） | SHA-256 | snapshot version |
| --- | --- | --- |
| 00-volume-escape-family.json | b54f762754e885207ddac1e9a3b86135fab689002ba6a242554c7328e776aa61 | ab3f1de81fc273853eb58b80ce7f388886e082b345e099a137a7c96a3c7658f6 |
| 01-v29c-volume-escape.json | b6c7f48af9f5acf2bd2221952914eb4c64a77d01783647dd049ca0243d5fb87e | 5328c9085269ccd6ef9522b2b3126a5b2d4a4a456581fdd87a73eef002ab0a73 |

族4/组合6规则，含两侧开仓和平仓。组合为base LONG / supplement LONG / base SHORT / supplement SHORT。原9指标和1h/4h/1d、基础两入口及完整统一关闭对象与AF0逐字段相同。不新增指标/系统变量，不改有意forming[0]，不读取forming taker[0]，不依赖OpenStrategyHash。

替换整个弱趋势补充入口机制，不再追加二次极值反转过滤器：4h闭合ADX<20时，最新闭合1h收盘首次超出之前8小时高/低极值0.10ATR，上一闭合1h未先超出其相应8小时极值；本小时quote成交额超过之前8小时平均，闭合主动买quote占比>0.5做多/<0.5做空并保护有效范围；目标方向实体0.15..2ATR，当前已观察价格仍在闭合收盘反向0.15ATR和同向0.35ATR以内且当前quote>0。

目标假设是成交活动支持的早期区间脱离延续，与RG1二次试探轮换反转不同。ADX弱不等于稳定震荡；taker占比不是资金净流入、订单簿吸收或独立盈利证明。保留既有8小时和ATR尺度阈值，无网格搜索或币别参数。

395旧portable、849同方向全程序去空白比较无精确重复。相关v31/v32/v36/v60/v65/v70/v75已读：存在趋势突破、极值失败反转、dry-up与effort/result结构，不宣称此次组合具有语义/alpha新颖性，也不把精确去重当有效性证明。

## 合成验证修复与统一关闭

仓库外verification/rg2_expr_checks.go真实Go/Expr首次1470项中6项失败，结果results/20261003-rg2-expr-checks.json和失败helper rg2_expr_checks_failed_base_overlap_20261003.go保留。失败均为组合LONG的ADX=20/25/50局部匹配数/第一规则预期：RG2正确为false，但未被隔离的原base LONG正确可达，检查器错误预期全无匹配。仅将rangeFixture的日线DI设为相等，隔离补充规则边界；原baseFixture单独继续证明基础入口可达。没有改策略、生产或容差，不把真实基础信号屏蔽写进候选。

新results/20261004-rg2-expr-checks-validated.json：1470项0失败，SHA dd4c3c6e807c65f3af6cc0f4fdb8a59e1f9402982d44a69224044d09d50024f5，37303真实terminal exit0。涵盖弱ADX/首次脱离/历史量均值/闭合主动额有效性与方向/实体与当前缓冲边界、价格和成交额比例缩放、forming taker独立、禁用规则、原入口/指标/退出完整身份及ROI/hash矩阵。含静态和重复合成案例，不是1470个独立市场样本，不是引擎私有selector、实际缓存接收者、API/live/forward或利润证据。

| LONG/SHORT情况 | 统一关闭结果 |
| --- | --- |
| ROI=+5/-5，或+16/+28/-12，仅ROI到达、确认信号不成立 | false |
| ROI>=16且趋势失败，或动量失败且反向累计量/实体冲击 | true |
| ROI<=-12且趋势或动量失败 | true |
| ROI>=28且动量失败或反向冲击 | true |
| ROI<=-20 | true，唯一ROI-only灾难止损例外 |

空/错误/实际开仓hash不改变统一关闭。外部profit/loss5/5只是调用资格，内区间不运行关闭；AutoStopOrder=false。5%不保证在该点成交；没有只看盈利阈值的普通止盈。

## 主对照与全部硬门槛

- AF0、RG1、RG2三个组合×BTC/ETH/SOL/XRP，各1000初始现金完整重新撮合，共12run。族单独收益没有本轮证据，不能从组合账本删/加成交当独立族利润。
- UTC2022-09-01T00:00（含）至2026-10-01T00:00（不含），1491天/213周、每币至少192笔。四完整Sep–Aug年、单独2026-09及日历2023/24/25/2026 Jan–Sep均报告。45月退出cohort只是归因，不是1000独立重跑收益率。
- 原标准engine v7/1m，观察分钟close决策、next-minute-open成交；当前可用现金10%保证金、8倍、双边各fee0.0005/slip5bps、真实资金费率/时间及原缺mark价格回退，外部5/5不变。
- 每币>=0.9次/周、净正、四完整年稳定、跨币泛化联合通过；不删失败币/年度、不改成本/频率、无事后胜者筛选。开发窗已经多轮观察，不是未观察时间验证；只称本候选参数在本轮利润前冻结。
- 未观察验证币AAVE/ATOM/ETC/LINK名单不变、历史资格待核对；开发失败不读取这些币收益。没有合格策略，不标目标complete/blocked或自行paused。

## 来源、成本限制和审计范围

复用缓存/Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1；execution/indicator均public-archive，minute-repair=verified-archive、repair_version=20261003-v2。canonical跨周期量差和独立纠错失败/成功证据保留，不强改canonical quote成分钟总和。真实资金费tail manifest SHA 78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547（每币54 suffix/7 overlap），前缀已在PV5以完整字段DeepEqual通过。

四币data_hash：BTC c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457；ETH 7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371；SOL 64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818；XRP 8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。缓存每次反序列化并重算hash/日期/周期/来源身份，不凭文件名跳过完整性。

原replay复制SHA 21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8，隔离data helper SHA a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63。只读解析commented arm/白名单库，原engine/生产/数据库/配置不变。

RG1对照1897笔原成本算术通过，但5030次资金费应用中1272次缺mark回退，RG1自身1630/416；一条旧官方BTC查询也mark为空。完整费率/时点不等于完整精确交易所结算mark，算术相符不等于成本资格发布通过。RG2必须另报原mark回退次数；不补零资金费、不制造精确mark、不改源码/runtime。安全同模型比较继续，不以局部数据限制停止无关研究。

主回测之后核对全部成交会计、8个AF0/RG1完整共享控制、年度/日历/方向/族/集中度，以及原现金复利仓位/fills/fee/funding时点与mark回退；所有实际RG2补充entry_time−1信号复核canonical閉合8小时极值/quote/taker/实体/首次条件、live仅观察时点价格和quote、原9指标fresh receiver和实际200输入/199闭合种子。fresh不是完整顺序cache、私有selector、API/live/forward或独立指标数学实现。分钟非零活动不代表订单簿成交保证。

## 保护与恢复

00:06配置SHA 7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa，engine ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad，environment b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5，indicator_cache 1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0；正式技能1.0.8 SHA b36ba1e915711e54aab4e5ca3bb3567be95e737336d1c2af6e27e12b54a82f77不变。

主输出results/20261004-rg2-development4-canonical-repaired-v2-funding-tail-v1.json，每run完成落盘。启动前pgrep无本轮回测/审计；启动后在检查点记实际handle/进度，先poll同handle，超时不是终止，不重复启动。

禁止App、DB写入/分配/启用交易、app.conf/生产/前端变更和新仓库测试文件。完整候选保留temp_strategy，辅助程序/overlay/大输出在仓库外缓存。技能receiver种子和mark覆盖候选已structurally valid但行为评估pending，未晋级；文档工作不代替策略推进。quota/中断恢复前先阶段总结，再核对实际goal/进程/冻结身份；若paused立即停止，不自行恢复。
