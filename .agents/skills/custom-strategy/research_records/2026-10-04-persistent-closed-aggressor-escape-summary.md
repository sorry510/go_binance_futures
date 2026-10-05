# RG5完整研究总结：累计闭合主动方向改善，但仍不满足联合门槛

2026-10-04北京时间01:57:23完成核对。主23128实际terminal exit0，开仓30002/成本67444/关闭72330全部实际terminal exit0；12完整49月run/3144执行账目（包括共享对照，不是全新独立交易）。目标仍active，没有可发布策略。

## 结果与硬门槛

| 币 | 笔数 | 每周 | 净USDT | PF | 最大回撤% | 四完整Sep–Aug净USDT |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| BTCUSDT | 269 | 1.263 | 500.908 | 1.265 | 18.303 | 7.613 / 48.469 / 72.758 / 460.347 |
| ETHUSDT | 303 | 1.423 | 105.641 | 1.051 | 33.175 | -143.401 / -112.377 / 213.871 / 198.410 |
| SOLUSDT | 308 | 1.446 | -353.847 | 0.851 | 41.994 | -285.524 / 91.417 / -35.798 / -82.737 |
| XRPUSDT | 258 | 1.211 | -43.513 | 0.977 | 29.526 | -14.805 / -196.772 / 151.485 / 33.912 |

四币频率均>=0.9，但SOL/XRP净亏，ETH/SOL/XRP各有亏损完整年。BTC的四Sep–Aug年虽全正，日历2024仍负，不把滚动年边界表现冒充所有年度稳定。各版本的实际正负方向与出入场路径都保留，不按币/方向删失败结果。

| 币 | 日历2023/2024/2025/2026 Jan–Sep净 | 追加2026-09净 | 去最佳5笔净 |
| --- | --- | ---: | ---: |
| BTCUSDT | 193.622 / -92.126 / 252.916 / 229.121 | -88.278 | 201.396 |
| ETHUSDT | -6.597 / -139.640 / 181.356 / 202.609 | -50.860 | -224.829 |
| SOLUSDT | -0.347 / -259.795 / -103.320 / 8.735 | -41.205 | -670.257 |
| XRPUSDT | -76.975 / 130.137 / -197.211 / 169.247 | -17.333 | -389.656 |

年度/月份/族分解按exit time归因，不是独立初始化收益；开发49月已被观察，不能称未观察时间验证。AAVE/ATOM/ETC/LINK未观察收益未读、资格仍待核，开发失败不进入验证。RG5相对RG4四币总净均改善，但该比较不证明跨币/真实成本/年度门槛已经通过。完整版本裁决：invalidated（针对全部联合发布门槛）；不是证明永远亏损或可交易反向。

## 失败归因与下一维度

| 币 | 补充LONG笔数/净 | 补充SHORT笔数/净 | 全部gross/fees/funding |
| --- | --- | --- | --- |
| BTCUSDT | 95 / -239.892 | 77 / 129.183 | 767.480 / 247.809 / -18.763 |
| ETHUSDT | 108 / 53.512 | 96 / -256.593 | 341.280 / 221.373 / -14.266 |
| SOLUSDT | 72 / -181.891 | 105 / -224.182 | -136.779 / 197.368 / -19.700 |
| XRPUSDT | 44 / -64.558 | 127 / -309.940 | 149.039 / 187.896 / -4.656 |

SOL未扣费用gross也负，不是只减手续费能解决；XRPgross正被成本耗尽，实际不能按零成本判盈利。BTC补充LONG负、ETH补充LONG正、两币SHORT差异显著，不能机械删某侧或静态把亏损取反。RG5只要求八小时quote方向持续，并没有验证这八小时价格是否沿同方向推进；持续主动额可能伴随对手盘吸收。下一轮仅增加同一闭合窗口价格进展：Close[2]与Open[9]沿同方向，严格等值拒绝；不从静态剔除账目推算反事实收益，先冻结完整RG6再全撮合。当前尚未生成/测试RG6，不预判盈利。

## 完整审计与成本资格

8个AF0/RG4控制逐笔/metrics/年度/source身份完全重现；仅允许已声明cache_hit采集元数据差异，会计errors=0。新增开仓724 /闭合字段57920 /failed0，两个[2:10]累计额另与canonical独立求和相等、各小时有效且严格方向多数。原ATR/ADX/EMA用fresh接收器实际200输入/199闭合原函数另算，不是独立数学或全顺序缓存/private selector/API/live/forward证明。

正常关闭1138 /forced-end0 /failed0；old pass517 /RG4弱小时分支627 /重叠6 /added-only621。按exit_time−1原Position/gross mark-price-denominator ROI/外部gate和闭合结构核对，不用net/margin替代ROI。分支同环境归因不是删除分支反事实收益。

成本3144笔/5023真实资金费应用/1329原观察分钟Close mark回退；zero activity=0、arith failed=0。RG5自身1138笔/1783结算/475回退。实际费率时间已覆盖，exact marks=false，精确交易所成本资格仍未补齐，不补零/伪造mark/修production。分钟有活动不是订单簿容量或实际费率等级证明。

原standard v7/1m、下一分钟Open不利5bps、各边0.0005费、现金10%/8倍/外部5与5、-20唯一灾难例外和全部正常信号确认保持。普通ROI单独为false；±5资格+弱ADX+方向结构破坏为true；等值/内区间新增分支false；原16/−12/28确认分支原样。AutoStopOrder=false，内区间不调用close。未做API/live/forward验证，研究文件不是模板激活。

## 身份、文件和保护

完整族/组合留temp_strategy/20261004-persistent-closed-aggressor-escape/；SHA/version见首次收益前protocol，3206 Expr检查0失败，失败helper说明引号源保留，不改策略/容差。新9配置/AF0基础/完整RG4关闭都通过对象相同断言。373旧portable/1614同侧全程序无精确重复只是程序去重；旧v88/coherence未读收益。三库元数据截止01:39:58.217/01:40:03.394/01:40:08.651，模板17/17/17、结果221/218/7、v29相同，不是446行全新前向复查。

- results/20261004-rg5-development4-canonical-repaired-v2-funding-tail-v1.json SHA d869aecd6f3ef4692c79944b9c355253f2b8e49dc1a16316b7f2154002ca0eda
- results/20261004-rg5-accounting-summary.json SHA f7052028655cbcfd21cf942880f0d8d3ec6bd8013f424b5c966a1b1809b5385b
- results/20261004-rg5-expr-checks.json SHA 601a2aa3218f491e9ba718af1c2e432df9ea7c2629eaeccc5d785fb12edbf215
- results/20261004-rg5-canonical-actual-seed-open-signal-audit.json SHA 8f2881b7eddc53283601c6d8c9b73bab86e411954599792383371f0129fd1a5a
- results/20261004-rg5-execution-original-cost-fallback-audit.json SHA 58cac10a4c6f898a81a80122684570318fd60abc47ed5a5f34b5e95ecded1996
- results/20261004-rg5-original-position-close-signal-audit.json SHA efde285c2ecbcc2d48c8483d0fdf00fafa6d3e8f1af9975e07c6428a6325ddf8

完整首次收益前协议见2026-10-04-persistent-closed-aggressor-escape-protocol.md。无App/DB写入/分配/启用、生产/前端/config/新仓库测试文件。正式技能1.0.8/trusted:false保护SHA未变，receiver-seed/funding-mark候选行为pending、未晋级。下一次先阶段总结、核对goal/真实进程，不重复已terminal主；全部原硬门槛保留。
