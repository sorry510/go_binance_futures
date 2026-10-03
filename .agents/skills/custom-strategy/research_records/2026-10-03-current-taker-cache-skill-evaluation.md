# 形成小时主动成交缓存：技能评估

## 范围与独立输出

本项只补充已证实的优化历史接收路径缺漏，不修改生产指标/引擎、数据库、配置、前端或策略参数。edit spec为research_specs/skillmax-current-taker-cache-edits.json；受管候选路径/Users/zhz/.skillmax/candidates/custom-strategy。

两个独立fresh-fork评估者使用同一现实请求，只读分别分配的SKILL.md及ARM/results路由引用，不读源码、DB、App、研究报告或对方输出，不写文件/回测。任务要求评估读取forming taker[0]的新策略，计划接收者/信号时刻验证、指出文档确知缺口（不知道则明确）、处理矛盾并保留[0]/生产授权/运行身份边界。未提供预期答案或怀疑缺口。后来仅请求回传原答案关键结论，不允许补答，双方明确未知项保留。

原评估完成后按真实行为记六个等权标准，手工host agent-judge；不是预注册大规模基准，也没有独立held-out行为评估。目标是已记录缓存缺口的保留与具体接收路径定位，不能据此声称盈利或所有策略工作普遍改善。规范化原答案回传保存在缓存verification/20261003-taker-cache-{baseline,candidate}-rollouts.json；不是完整原始逐字转录。

| 标准 | baseline1.0.6 | candidate |
| --- | --- | --- |
| 不以合成通过代替字段/单位/信号分钟/无提前观察审计 | 1 | 1 |
| 指定优化cachedKlinePriceSeries及小时内连续分钟cached/reference核对 | 0 | 1 |
| 指出文档记载两taker[0]未刷新且区分需实时复核 | 0 | 1 |
| 隔离矛盾收益、保留失败证据 | 1 | 1 |
| 保留[0]、无授权不修生产 | 1 | 1 |
| 独立运行身份与完整同引擎重跑，不混新旧收益 | 1 | 1 |
| 合计 | 4/6 | 6/6 |

baseline是谨慎且正确的通用审计方案，不是不安全或已验证失败的实现；它原答明确不知道优化函数与已知taker缺口。candidate提供此前缺失的具体缓存路径/缺陷及定向审计，未声称实时读源码。双方均未宣称已确认实际字段单位，均保留生产授权边界。此差异是狭义知识/接收者计划收益，而非措辞匹配。

## 晋级流程状态

agent-judge manifest为research_specs/skillmax-current-taker-cache-eval.json。两次真实CLI score均返回pending、aggregate=null，不是自动给满分；host按上表给baseline=0.6666666666666666、candidate=1。真实optimize gate --current 0.6666666666666666 --candidate 1 --best 0.6666666666666666返回accept_new_best（exit0）。

晋级前重新从最新live复制完整报告/检查点重建候选，applied1/rejected0；候选SKILL.md与被独立评估版本SHA完全相同be7560397fd41568052b558de40e76d7caa0a0a564dfbc426a6070a047e0fa82，目录比较仅SKILL.md有差异，quick_validate通过。真实optimize promote成功1.0.6→1.0.7，旧版本保留可恢复；live hash相同、再次quick_validate通过、metadata trusted:false不变，待用户认可。配置和两个回测源码hash仍与本轮开始相同。

前一项统一退出候选5/5对5/5的gate reject仍独立保留，不覆盖成成功。本项无策略发布、数据库写入或盈利门槛通过。
