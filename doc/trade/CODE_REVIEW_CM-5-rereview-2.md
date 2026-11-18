# CODE_REVIEW_CM-5-rereview-2 · 复审观察项修正与回归

> 日期：2026-11-18。工作分支 `feat/m-coin-futures`。原始复审报告：[CODE_REVIEW_CM-5-rereview.md](CODE_REVIEW_CM-5-rereview.md)。
>
> 本文是原始复审之后的独立处理记录，**不改变原始 Findings 或其当时证据**。CM-5 的验收范围仍是源码、隔离 SQLite、HTTP Mock、Fake 与构建，不包含真实 Binance 交易或生产发布。

## 一、原复审判定

原审查结论为「**有条件通过，可进入 CM-6 的离线开发**」，没有 P0/P1 阻塞项。下列四项为观察项，不应把真实 Demo/Testnet 验收错认为本地测试已通过。

| Finding | 优先级 | 核对/实施结果 |
| --- | --- | --- |
| **O1：没有 HTTP/UI 手工下单入口** | P2 | **符合现有开发边界，不修改**。CM-5 只提供内部执行原语，未在 `main.go`、`routers` 或控制器接入自动或手工下单。CM-6 再研究共享调度接线。 |
| **O2：userTrades 每单只取最多 1000 笔** | P2 | **分页能力仍未实现**，保持 fail-closed。额外修复了本地状态：`Reconcile` 遇到满 1000 笔时先持久化 `AWAITING_FILLS`，再返回 `ErrUncertain`，不再保留看似已经结案的 `FILLED` 状态。未来需要分页时再按真实 API 语义设计。 |
| **O3：第二个保护单提交前，对远端已有 Algo 缺少类型比对** | P3 | **已修复**：`PlaceProtection` 的远端 `OpenAlgos` 归属检查新增 `remoteAlgo.OrderType == owned.OrderType`，与取消兄弟保护单的归属校验一致。错误类型、缺失类型都阻断提交新的 Algo。 |
| **O4：零持仓合约的 positionRisk / TradeSettings 返回完整性** | P2 | **保留为真实环境验证项**。当前校验无法证实杠杆及保证金模式时应拒绝新开仓，不因 API 结果为空而推断符合配置，也不擅自调用影响 U 本位/币本位共享设置的修改接口。 |

## 二、具体代码修改

1. `service/coinmexecution/protection.go`：对于远端已有保护单，同时匹配 clientAlgoId、algoId、symbol、positionSide、side 与 **OrderType**；任何不一致都报错，不发送第二个 Algo POST。
2. `service/coinmexecution/reconcile.go`：`userTrades` 恰好达到 1000 笔时，立即将本地订单置为 `AWAITING_FILLS`，返回 `ErrUncertain`，不将可能截断的一页当成全部成交，也不自动重新 POST。
3. 新增 `service/coinmexecution/protection_type_rereview_test.go`：真实已归属 STOP 作为基线，模拟远端类型被改为 TAKE_PROFIT 或空字符串，断言新 TAKE_PROFIT 不得提交、本地保护单保持原状态。
4. 新增 `service/coinmexecution/trades_page_rereview_test.go`：模拟交易所返回 `FILLED` 与整页 1000 笔 `userTrades`，断言结果为 `ErrUncertain`、本地 `AWAITING_FILLS` 已持久化、未新增或伪造 Fill、没有重复 POST。

**安全取舍**：不为个人系统新增审批、第三个开关、其他币种调参或复杂清算层。既有全局 `coin_m_quant_enable` 与每币 `delivery_symbols.enable` 控制开仓，已有仓位退出独立处理。

## 三、复审后回归 Gate

| 验证 | 本次执行结果 |
| --- | --- |
| 两个新增测试 `-count=5` | **PASS** |
| `go test -count=5 -p=2 ./service/coinmexecution ./service/coinmquant ./feature/api/binance ./controllers` | **PASS** |
| `go test -race -count=2 -p=2`（上述 4 包） | **PASS**（macOS linker `LC_DYSYMTAB` 非致命 warning） |
| 88 个非历史研究 Go 包 `go test -count=1 -p=2` | **PASS** |
| 同一 88 包 `go vet` | **PASS** |
| `go build -o /tmp/go_binance_futures_cm5_rereview_v27 .` | **PASS** |
| `git diff --check` | **PASS** |

此处「非历史研究包」排除项目中已知重复定义的 `strategy_templates/research/`。本轮没有前端源码变更，也不需重复构建前端；此前 CM-4 的 `dist/static` 状态不作本轮验收扩大解释。

## 四、仍未通过的真实环境 Gate

没有执行 v27 MySQL 数据库升级、真实 COIN-M 签名 DAPI、Demo/Testnet 订单、私有 User WS、Algo 子订单父子身份的现场验收，也没有验证零持仓账户实际杠杆/逐全仓接口行为。这些仍是 CM-6/CM-7 和发布前必须分别完成的工作。

**阶段结论：CM-5 仍仅限源码与离线验收有条件通过，允许推进 CM-6 研发，但不能直接发布真实交易。** 本轮没有执行 `sync db`、改动 `app.conf`、发送订单、重启后端或提交 Git。
