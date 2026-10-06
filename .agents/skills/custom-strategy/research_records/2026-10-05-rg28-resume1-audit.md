# RG28 第一次显式恢复核验

2026-10-05 15:19:27 UTC：用户继续active thread goal，实际get_goal为active；上轮真实暂停终止属于完成预检/11run的progress，不是no-progress或verifiedwait。无遗留活进程，不poll旧42234。

收益前协议SHA d50896a7b217e3735fa5944f41d56ad844af44014d5cbfe59d660260d69c8c1a和32个冻结文件刚刚逐字节验证全部exact。当前partial main SHA191dbfc60484fc71c2744361e5a2ec05009a4e82109a617cdb9f8ba1c180211b、11唯一run/3完整candidate/source/config/49月dates身份完整，唯一未完成XRP/RG28 projectversion09f142498db4feb6be445217aeff6a7fe3314e8f9704916c2688bf30a32687de；未覆盖或删除结果。

当前程序化只读local v29 snapshot75165实际terminal0：go_bn_test ID114两JSON语义与冻结原v29完全一致，database_writes0。补充哈希：
```text
9c34b2e24e5d050eac2e80e3649f7677d8a62e647cbedd59be6b3b2e7eabb430  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg28_resume1_local_v29_snapshot.go
085e322a1825571c7aafb19094c29f704836c93a00020eb36c98af8181e328ee  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-resume1-local-v29-snapshot.json
```
配置、不变候选、生产源、成本/数据/风险/频率≥.8和不强求逐币盈利的固定组合稳定/未阅组均保持。仅第一次启动outputabsent历史条件替换为以上严格现有checkpoint校验；原harness会跳过完成11run、完整重做被终止的XRP RG28。启动后保存工具真实句柄，timeout不重启，actualterminal之后才全部后审计。

唯一恢复命令：
```text
rtk proxy go run /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go -conf conf/app.conf -strategy-files temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json,temp_strategy/20261005-counter-shock-flow-recovery/01-v29c-counter-shock-flow-recovery.json,temp_strategy/20261005-observed-midshock-recovery/01-v29c-observed-midshock-recovery.json -symbols BTCUSDT,ETHUSDT,SOLUSDT,XRPUSDT -start 2022-09-01 -end 2026-09-30 -execution-source public-archive -indicator-source public-archive -minute-repair verified-archive -cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1 -archive-cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2/archives -output /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261005-rg28-development4-canonical-repaired-v2-funding-tail-v1.json
```
