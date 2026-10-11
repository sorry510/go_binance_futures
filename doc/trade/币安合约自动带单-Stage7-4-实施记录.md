# 币安合约自动带单 Stage 7-4 — 双账户 Mock 联合回归、Race/Vet/Build

> 日期：2026-10-11，分支 `feat/lead-trading`。**仅离线代码和 Mock 验收，不允许真实 Lead 交易。**

## 实施及实际证据

- 新增 `feature/account_trade_cycle_stage7_4_test.go`：同一批共享 LONG/SHORT 信号在 Main/Lead **四种开关组合**（均关、仅 Main、仅 Lead、均开）下进行每例 8 轮加速并行 Fake 测试，检验仅允许的账号发出对应 Mock 意图；Lead 风控故障模拟下 Main 继续运行 12 轮、信号数不变。
- Lead 实际生产 runner 仍没有 `stage3LeadMockPermit`，新测试显式检验无测试专用 permit 无法构造可运行的生产 Lead 交易循环；旧 Stage 3/4 golden、Ownership 幂等、成交/撤单、短暂 WS 中断及未知提交的完整 Mock 矩阵仍参与相关包完整回归。
- Stage 7 只读宿主的 `Run`/Stop、按账户隔离、凭证轮换旧连接取消、审计 fail-closed 和内存监控高并发均通过 Race。
- 最终命令：`go test -race -count=1 ./service/leadaccount ./feature/api/binance ./service/futuresownership ./feature ./feature/strategy/line ./scanner ./service/backtest ./command ./controllers ./middlewares ./routers -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'`：**10 个有测试包 PASS**，`routers` 无测试。 `go vet` 同模块：**PASS**；`go build` 生产根包：**PASS**；`git diff --check`：**PASS**；所有本轮修改源码均经过 `gofmt`。macOS 可能显示非致命 `LC_DYSYMTAB` ld warning。
- **限制**：此处的“8/12 轮”是**加速逻辑 Mock，不是实际 2 秒网络负载/交易流量**。未执行双账号真实 Binance 流量评估、真实 WS/Algo 触发、网络限速/429/418 压测、生产 MySQL 8 数据验证，也不代表 Gate 0-LIVE/5/6/7 通过。
- 当前本地已有后端进程监听 3333，本轮没有启动/停止该服务，不自动复制新的二进制或前端 static、不 commit/push、不执行生产 DB 迁移。

## 工作边界

没有修改 `app.conf`，没有运行 `sync db`、修改真实业务库或触发任何真实 Lead Submit/Cancel/REST/WS 测试；没有自动提交/推送 Git，不回退未提交的用户修改。
