# QPS定义与真实资金费尾部：狭义技能改进评估

本次只补两个已实际验证、可重复的方法缺口，不重写技能、风险/发布门槛或外部45月默认日期改动。新增文本在`research_specs/skillmax-observed-qps-funding-tail-edits.json`，当前生产/数据库/配置均不改。

## 实际独立评估

两个全新受限上下文agent只读取分配的完整SKILL.md、ARM/results两个引用和同一真实研究恢复请求；不读源码、其他技能版本、数据库/App/web、研究报告或另一答案，不修改/跑回测。请求先于两个答案发出，具体评分标准是在答案完成后由host按行为判断：不声称预注册大规模benchmark或独立held-out验证。

- `/root/pv5_skill_eval_a`读取最新live baseline（保留外部日期改动），SHA `3a5bf0e7d90fb92e62ca325a5ca8e3c33687a395715a18557d93a45d1e380027`。
- `/root/pv5_skill_eval_b`读取只添加两条补充的candidate，SHA `b36ba1e915711e54aab4e5ca3bb3567be95e737336d1c2af6e27e12b54a82f77`。
- 实际答案规范化摘要保存在缓存verification/20261003-qps-funding-{baseline,candidate}-rollouts.json，不是完整逐字转录，也没有在摘要中扩写新能力。两者均正确拒绝盈利/发布推断、保留授权与完整门槛；baseline对未说明QPS等事实谨慎标未知是合理行为，不把它说成不安全。

## Host agent-judge（八项等权）

| 可观察计划标准 | baseline | candidate |
| --- | ---: | ---: |
| 采集在收益前失败不是策略亏损/盈利证据 | 1 | 1 |
| 保留[0]、仅观察信号时间之前分钟、不用未来量 | 1 | 1 |
| 保留有日期的名义周期累计QPS定义，不称elapsed rate，并须复核当前runtime | 0 | 1 |
| 真timestamp/rate/结算mark与ARM重叠、重复/gap/end验证，不伪造资金费 | 1 | 1 |
| 具体定位loader只补前缀、不补尾部，以隔离真suffix和新cache恢复 | 0 | 1 |
| 全部原bar/资金费字段前缀exact校验，再完整重跑全部冻结控制 | 0 | 1 |
| 显式四年/全曝光每币频率/跨币优先，cohort不是重置账户收益 | 1 | 1 |
| 原成本/引擎/来源身份与生产/DB/配置授权边界 | 1 | 1 |
| 合计 | 5/8 = 0.625 | 8/8 = 1 |

baseline有通用源hash和完整对照计划，但未写明确全字段旧数据前缀相等，故不能把它计成该更具体标准已完成。新文本的胜出范围只是此请求的已知定义保留与具体采集恢复计划，不是盈利、普遍行为优越或真实运行验证；不能和此前不同rubric的score=1混为一谈。

## 实际工具状态

评估manifest为`research_specs/skillmax-observed-qps-funding-tail-eval.json`。两次真实CLI score均exit0并返回agent-judge pending、aggregate=null，不是自动判满分；host按上表完成实际答案评分baseline=0.625、candidate=1。同请求baseline是本次新rubric的初始best，不与此前不同评估数值混用。真实`optimize gate --current 0.625 --candidate 1 --best 0.625`返回accept_new_best、exit0。

晋级前已从最新live完整目录重新执行apply（budget4/applied1/rejected0），连同本轮summary/checkpoint和所有外部改动复制；目录比较仅SKILL.md不同，候选SHA与被评估版本完全相同，quick_validate通过。真实promote于北京时间19:56:40.447成功1.0.7→1.0.8，旧1.0.7保留可恢复；晋级后再次quick_validate通过，live SHA为b36ba1e915711e54aab4e5ca3bb3567be95e737336d1c2af6e27e12b54a82f77，外部45月默认与本轮记录均保留。

metadata与SkillMax state仍trusted:false，待用户认可；这里是文档方法可复用，不是用户已经认可、生产修复、数据库写入或盈利发布。保护配置/源码SHA保持。完整研究目标仍active，技能优化不能代替下一轮完整策略检验。
