# 币安合约自动带单 — Stage 4-6 Mock 全流程与 Gate 4 验收计划

> 状态：**已完成离线 Mock 验收（Gate 4 Offline PASS）**；2026-10-10。实际用例、修复、Race/Vet/Build 结果及未覆盖的真实交易 Gate 详见 [Stage 4-6 实施记录](币安合约自动带单-Stage4-6-实施记录.md)。

## 目标

串联 Stage 4-1～4-5 的账户身份、策略信号、开仓风控、Ownership/Broker、普通/Algo 恢复、暂停熔断与故障保护，证明 **Main 完全不受 Lead 引入影响**，且生产 Lead 写路径默认锁定。

## 验收矩阵

| 场景 | 关键断言 |
|---|---|
| LONG、SHORT 独立开仓 | 账户正确、策略一致、币种/白名单与额度独立 |
| CLOSE_LONG、CLOSE_SHORT、TP、SL | 优先级沿用 Main；受控仓位不超 `min(managed,live)`，人工单不被认领 |
| MARKET / LIMIT | 策略价格数量、Tick/Step/名义额/杠杆/保证金模式符合 Lead 合约过滤器 |
| 部分成交 → 撤单 → 重复 WS/REST 回报 | 成交量按累计值幂等，取消不代表 0 成交 |
| 拒单/429/418/网络超时/未知结果 | 查单不盲重试、账户级阻断状态与告警原因正确 |
| 普通单和 Algo STOP/TP 的恢复 | 仅绑定 Lead 查询；触发单必须核对真实子订单 |
| 凭证轮换、白名单失效、快照过期、缺失 WS | 不开启 Lead 新仓，不影响 Main；敏感数据不出现在日志 |
| Lead paused / risk_tripped / reconcile_required | 不开新仓，安全退出意图仍可判断，不确定写受阻 |
| Main 与 Lead 两套 Runner 模拟并行 | Main 结果对齐既有 Golden；Lead 故障不影响 Main |
| 默认生产编译与公开入口扫描 | 无可调用的未经授权的 Lead Submit/Cancel/Reset/Enable 路径 |
| 测试结束、DB、配置及 Git | 测试无遗留进程、无真实账户写调用、未执行主库 DB 迁移 |

## 完成规则

1. 定向和跨模块 Mock 测试、`go test -race`、`go vet`、生产 `go build` 通过。
2. 将每个验收场景的结果、Mock 数据来源、已知失败、与既有 Main 结果差异写入独立 **Stage 4-6 实施记录**。
3. 报告必须显式划定未覆盖的真实 Binance Lead Portfolio、真实 STOP/TP 权限、Stage 5 私有 WS/重启恢复、Stage 7 人工授权。
4. 只有 4-5 和 4-6 都完成且没有阻断缺陷，才将 Gate 4 标为 **离线通过**；不能标为实盘可用。


## 实施结果（2026-10-10）

- Stage 4-1～4-5 的相关测试全部执行；新增 `stage46_integration_test.go` 和 `account_trade_cycle_stage4_6_test.go` 联合 Mock、Main/Lead 双账户并行与 AST 生产入口扫描。
- 原验收矩阵已按 [实施记录](币安合约自动带单-Stage4-6-实施记录.md) 各项通过；交易所规则读取不可用时新增固定故障码 `lead_rules_unavailable`，不影响 Main。
- Gate 4 **只通过离线验收**；真实 Portfolio 账户归属、WS/重启恢复、实盘写权限、真实 Algo TP/SL 和 Stage 7 人工授权均尚未验证。严禁以此作为生产启用理由。
