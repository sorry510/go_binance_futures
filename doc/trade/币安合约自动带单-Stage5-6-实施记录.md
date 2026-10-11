# 币安合约自动带单 — Stage 5-6 多账户并发回归与 Gate 5 离线验收

日期：2026-10-10 至 2026-10-11（按本地测试环境）
状态：**离线实现、生产尚未接线；Mock 测试通过。真实 Binance/Testnet 环境及 Gate 5 未验收**。

## 实现与验收

- 联合 Stage 4-6 Main/Lead 两循环 Mock 与 Stage 5-1～5-5 的专用测试：重复/乱序 WS、ListenKey 到期与重连、启动时 DB Pending/Unknown、32 并发原子订单 Claim、429/418、快照启动缺口、API 统计、同账户通知取消与 UI/镜像并发观测。
- `service/leadaccount/stage56_integration_test.go` 对多线程状态读取、镜像变更、通知 Worker、启动对账与 WS 断连做加速并发模拟；测试使用固定 Mock 数据和逻辑加速循环，**不等于真实生产两账户每 2 秒网络负载测试**。
- Stage 4 的生产依赖闭包守卫和公开方法白名单继续有效；全仓扫描只对 `service/leadaccount/supervisor.go` 内部一个经过审计的 inert 构造调用作显式例外，生产根包依旧不能引入该模块。
- **结论口径**：Stage 5-1～5-6 已完成离线开发及 Mock 联合回归；Gate 5 的生产实测项（真实 Binance 私有 WS、测试网断线/ListenKey/Algo、双账户真实 2s 负载、MySQL 8 恢复与 API 预算证据）尚未执行，不能标为生产上线通过。
- 后续 Stage 6/7 仍需 Portfolio 身份/资产归属、受控交易前端/后端强鉴权及最小金额真实权限验收；当前 Main 生产程序仍没有新 Lead 执行任务。

## 安全约束

未修改 `app.conf`、未执行 `sync db`、未访问真实 Lead API Key 或实际下单、未自动 commit/push。Main 原有行情与交易仍为独立代码路径；这份离线记录不构成 Stage 7 实盘授权。


## 实际自动化测试汇总（2026-10-10～11）

| 检查 | 结果 |
|---|---|
| `go test -race -count=1`：`service/leadaccount`、`feature/api/binance`、`service/futuresownership`、`feature`、`feature/strategy/line`、`scanner`、`service/backtest`、`command`、`controllers` | **全部 9 包通过，RACE_EXIT=0** |
| `go vet`：同 9 包 | **通过，VET_EXIT=0** |
| 根项目 Go 生产 `go build` | **通过，BUILD_EXIT=0** |
| `git diff --check` | **通过，DIFF_EXIT=0** |
| `go list -deps .` 的 `service/leadaccount` 依赖 | **0 个**；生产根包仍无 Lead 执行入口 |
| 端口 3333 遗留监听进程 | **无** |
| Testnet 实际 WS、真实 Portfolio 与 MySQL 8 schema 迁移 | **本次未执行** |

回归沿用已有约定，只跳过与本功能无关的 `TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals`，并保留已知 macOS 非致命链接器 `LC_DYSYMTAB` warning。测试中发现的旧 Stage 1 限流测试预期已同步新策略：16 个普通开仓槽位 + 2 个安全退出/撤单预留，新增 Stage 5 测试覆盖该预算；原有 IP 全局限流未关闭。

**离线总结：5-1～5-6 均已实现并完成 Mock/相关模块自动化测试；Gate 5 真实环境验收未通过，必须在获得授权且具备真实 Lead Portfolio/Testnet、WS 和 MySQL 8 测试条件后补验。** 此状态不能作为开放 Lead 实盘交易的依据。

**Stage 5 代码审计 F1 修复**：`TestStage5ReviewSupervisorPublicSurfaceLocked` 固定监督器仅 `Run`、`Status` 两个公开方法；并沿用既有 `go list -deps .` 生产入口守卫，防止仅因为 Stage 5 实现完成就自动启用私有 WS 或实盘 Broker。离线测试通过不代表真实 Gate 5 或 Stage 7 授权。
