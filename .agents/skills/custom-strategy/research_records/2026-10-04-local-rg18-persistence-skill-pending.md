# RG18本地入库与技能更新草案

## 已完成的授权操作

用户明确要求“把RG18写入本地数据库中”。读取conf/app.conf活动[database]，实际目标127.0.0.1:3306/go_bn_test，不使用注释ARM配置或App。仅插入strategy_templates，不更改现有模板、symbols、策略分配、交易启用或风险参数。

来源temp_strategy/20261004-contracted-reclaim-extreme-followthrough/01-v29c-contracted-reclaim-extreme-followthrough.json，SHA06807efede7175fd511f5edd68cc49e7ea2fbdc8e269d1535e5e0d392ebebef6；与4566/0 Expr核验中的原文件身份一致，9个指标、6条enabled规则，包含四种type。既有invalidated研究结论不因持久化改变，未启动RG19或恢复研究goal。

实际只读检查：InnoDB，id自增，name/technology/strategy/createTime/updateTime字段匹配，JSON列longtext，0触发器，同名0。两个原raw JSON独立json.Compact，保留成员顺序与转义，不重新序列化策略表达式。

首次事务在读回时因本地不支持JSON_VALID函数失败，未commit，defer rollback执行；随后实际只读确认同名仍0，才重试。修复仅另存临时helper v2，使用Go json.Valid解析真实读回字节，并保持精确名称、CR/LF缺失和逐字节核验，不升级数据库/改表/跳过核验。第一次helper保留。

最终实际exit0，已提交1条、ID124，名称“RG18 v29C与收缩扫边收回实时极值突破组合（研究候选）”。createTime=updateTime=1791116182437，technology765字节、strategy10802字节；事务内与commit后JSON有效/单行/byte-exact及name_count=1均通过。不是激活或盈利证据。

confSHA7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa，正式SKILL SHA270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd不变；没有修改生产、前端或仓库测试文件。临时helpers在/tmp/go-binance-rg18-persist.JBH7Ui/，不是未来技能的持久依赖。

## SkillMax待评审更新

使用skill-creator指导，更新现有custom-strategy而不是安装近似新技能。spec为research_specs/skillmax-local-persistence-edits.json，仅补充授权的程序化本地入库、raw compaction、事务/重复重查和JSON_VALID缺失兼容。

实际skillmaxxing optimize apply --skill custom-strategy-local-persistence-20261004 --skill-dir 当前custom-strategy --edits 上述spec --base6 --min1 --total1 --step0：applied1/rejected0，候选目录/Users/zhz/.skillmax/candidates/custom-strategy-local-persistence-20261004/，trusted:false。未修改正式技能或此前候选，没有score/strict-win/promotion或新委派。

结构检查使用系统skill-creator quick_validate.py；其结果不证明行为优于基线。行为比较仍pending：活动local与注释ARM目标隔离；JSON_VALID不可用时保持等价JSON/单行/字节核验且失败回滚；提交结果模糊时先查精确名称、不得重复插入。现有成功操作是原始程序执行证据，不伪装为独立baseline/candidate评估。
