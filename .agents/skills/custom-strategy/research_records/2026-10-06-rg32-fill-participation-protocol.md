# RG32 成交名义金额/分钟成交活动：诊断前声明

冻结RG32 JSON、原49月四币/main/cost/八controls身份，不产生新策略或改变收益。本诊断只读取已完成的原完整12run/2215交易和成本审计2215行，逐行按symbol/version/sequence一对一匹配并核验entry/exit timestamp、dataset hash、价格和数量有效性；自身871及控制1344全部保留。数据身份需先与原phase引用SHA完全一致，缺行/重复/无效字段即停止，不能筛掉不利交易。

诊断名义金额=abs(actual ledger quantity×actual fill price)，分别除以对应原成本审计的canonical分钟quote；同报告trade count、p50/p90/p95/p99/max和最大占比前十行（排序只用于展示，不改账本）。按all/own/control和各coin/side报告范围，nearest-rank百分位预先固定。无通过阈值，不按结果增加gate/筛币/年/侧或声明该占比对应某个bp滑点。

分钟成交量是事后整分钟公共交易活动，不是下单瞬间可用盘口深度，也不是本策略挂单/市价单容量上界；该占比不能证明5bps成交、实际交易可复制、实际mark或完整真实成本。全部账目/净值保持原main，零活动/异常另报而非删除。无网络、DB、App、生产/front/conf/_test.go/globalmemory修改。

输出独立cache results/20261006-rg32-fill-participation-diagnostic.json；新只读配对helper verification/rg32_fill_participation_diagnostic.cjs。需要实际syntax/运行与输出SHA才可报告完成。主及所有旧审计已terminal，不能重新运行或poll旧句柄。本声明先于新占比数值计算；已有主收益不再作为未阅样本。

