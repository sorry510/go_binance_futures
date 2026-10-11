# 币安合约自动带单 — Stage 5 审计修复记录

日期：2026-10-11  
分支：`feat/lead-trading`  
审计来源：`doc/trade/CODE_REVIEW_Stage5.md`（原文件保留不修改）

## 总结

审计确认 **无 P0/P1**，Stage 5-1～5-6 的离线实现可保留。生产 Go 根包仍不链接 `service/leadaccount`、不会自动创建 Lead 私有 WS 或开通真实下单。**Gate 5 真实环境验收尚未完成、Stage 7 未授权**；下文测试不等同于真实环境验收。

## 审计项逐项处理

| ID | 等级 | 处理 | 依据与边界 |
|---|---|---|---|
| **F1** | P3 | **已修复** | 六份 Stage 5 实施记录统一标注“离线实现，生产未接线”；新增 `TestStage5ReviewSupervisorPublicSurfaceLocked` 以反射固定 `LeadSupervisor` 公开方法仅 `Run/Status`，构造不启用权限、不启动 WS。原生产入口 AST/依赖闭包安全测试保持启用；在总方案写明 Stage 6/7 接线授权步骤 |
| **F2** | P3 | **观测模型已修复；生产展示待 Stage 6** | `LeadAlertQueue.Snapshot()` 提供容量、深度、累计背压丢弃、严重级丢弃；`LeadUsageSnapshotWithAlerts` 和 `LeadSupervisor.Status()` 输出同源数据。首次发生严重/非严重背压时输出固定、无 Secret/原始 HTTP 数据的日志；队列仍非阻塞，通知失败不改变风控。由于生产 Lead 尚未接线，**System Dashboard 没有新增 Lead UI/路由** |
| **F3** | P3 | **代码与单测已修复** | `LeadRuleSource.exchangeInfo` 按 Lead 实例缓存成功公共 exchangeInfo 五分钟，`singleflight` 合并并发缺失；开仓 Verify 与受控退出 `verifyCloseRules` 共用快照；私有仓位与杠杆档位仍实时核对。到期刷新失败无 stale fallback、严格阻断；显式失效有缓存 generation 检查，旧请求不能回填已作废规则 |
| **F4** | P3（历史） | **后续 Stage 6/7 处理** | 涉及恢复证据持久审计流水、动态过滤器、部分 Algo 边界、`RuntimeStatus` 内部状态行为、真实双账户/DB/WS 环境验收。当前没有完整外部权威证据，不改动线上 Broker 或擅自将 Gate 5 标为通过；总方案已列接线清单与剩余风险 |

## 永久回归测试

`service/leadaccount/stage5_review_fixes_test.go`：

- `TestStage5ReviewLeadExchangeInfoTTLSharedOpenAndClose`：20 次开仓检查 + 1 次 CLOSE 共享 1 次公共 exchangeInfo，私有 Position 仍每次读取；独立规则实例不共享；TTL 到期且读取失败必须拒绝、恢复后新读与主动失效必须重新读。
- `TestStage5ReviewLeadExchangeInfoSingleflightAndCancellation`：48 个并发开仓 Verify 只读取 1 次公共 exchangeInfo、48 次私有仓位，已取消 Context 不可使用缓存。
- `TestStage5ReviewQueueLossVisibleInUsageAndSupervisor`：容量 1 时产生一次关键和一次普通告警丢弃，核对队列与 `LeadUsageMetrics`、监督器状态，同时保持实盘禁开仓。
- `TestStage5ReviewSupervisorPublicSurfaceLocked`：公开方法白名单与新建 Supervisor 默认为 disabled/未联网。

## 审计修复后最终验证（2026-10-11）

| 验证项目 | 结果 |
|---|---|
| `go test -count=1 -race`：`service/leadaccount`、`feature/api/binance`、`service/futuresownership`、`feature`、`feature/strategy/line`、`scanner`、`service/backtest`、`command`、`controllers` | **全部 9 包 PASS（RACE_EXIT=0）** |
| 同 9 包 `go vet` | **PASS（VET_EXIT=0）** |
| 生产根包 `go build` | **PASS（BUILD_EXIT=0）**；临时二进制已删除 |
| `git diff --check` 与修改文件 `gofmt -l` | **PASS**，无差异格式错误 |
| `go list -deps .` 中的 `service/leadaccount` 依赖 | **0**，没有生产 Lead 入口 |
| 3333 端口监听进程 | **无** |
| 已知例外 | 沿用既有约定排除与本阶段无关的静态 `TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals`；本机 macOS 链接器输出非致命 LC_DYSYMTAB 警告 |

**验收结论**：本轮 F1/F2/F3 的离线代码与文档修复已通过相关模块自动化回归；**Gate 5 真实环境验收仍未通过**。

## 后续环境验证清单（仍未通过）

生产 Lead 区块和强鉴权页面、独立持久恢复审计流水、真实 Lead Portfolio 身份、真实 WS/ListenKey/Testnet 断线、Algo 止损止盈、MySQL 8 持久对账、Main 在线 Gate、真实 2s 双账户负载/预算与通知发送时延均尚未在本次审计修复中执行。

**工作纪律**：没有修改 `app.conf`，没有调用真实 Binance 私有 API 或发送订单，没有执行 `sync db`、没有改动 `strategy_templates`，没有自动 Git commit/push；保留全部原有未提交文件。
