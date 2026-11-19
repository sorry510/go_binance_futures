# CODE_REVIEW_UF-1 · U 本位 Main 总开关与已有 Managed 出场解耦

> 日期：2026-11-18，后端 `master`。依据：[UF 六项精简优化开发方案](开发方案-UF-U本位合约交易六项精简优化.md)。
>
> **结论：UF-1 源码与离线 Mock/SQLite 测试通过。未连接真实 U 本位账户进行出场验证，尚未实盘验收。** 不包含 UF-2 杠杆/保证金错误传播修复。

## 一、变更与边界

| 文件 | UF-1 改动 |
| --- | --- |
| `feature/account_trade_cycle.go` | `mainTradeRunner.AllowNewOpens` 由 `Config.FutureEnable==1` 决定，OFF 也能执行 Main 退出循环；本地 `mainAutoStrategyWorkNeeded` 通过 SQLite/MySQL 兼容的 `EXISTS` 查询检查 Main `auto_strategy` 活跃且有正数张数的 Managed 仓位/未终结订单 |
| 同上 | OFF 且无本地 Main auto_strategy 工作 → 不发额外 DAPI 账户轮询；本地表出错则 fail-closed 并告警，不推断为“已无仓位”；ON 按原方式进入 Main 交易循环 |
| 同上 | OFF 时 `SyncPositions` 限定为 `syncAutoStrategyOwnership`，**不接管** `new_coin_rush` 和 `funding_rate` 等原本属于其他功能的 Owner；Lead 仍独立调用原共享 tick |
| `feature/feature.go` | 将 `SelectCoins` 移到 Managed 平仓和订单恢复之后，候选为空、选币失败不能阻断已有仓位；`!AllowNewOpens` 在候选策略/新订单前 return |
| `feature/lead_trade_parity_stage03_test.go` | 原 Lead 隔离测试按新语义更新 Main 开关 ON/OFF 期望，继续验证 Lead 不会与 Main 共用授权 |
| `feature/uf1_main_exit_test.go` | 新增 Main OFF/ON、Owner/账户隔离、LONG/SHORT、选币故障、无策略出场、缺失币种配置、离线唤醒与空闲测试 |
| `src/views/dashboard/configShow.vue`、`locales/{zh-CN,en}.yaml` | 总开关附近明确提示“关闭仅停止新开仓，已有 Managed 仓位仍按原规则退出；不是强平”及双语选币提示 |
| `doc/readme/{zh-CN,en}/02-*` | 中英双语操作手册同步总开关的新语义 |

## 二、处理次序与工作范围

1. `StartTrade` 每 2 秒沿用原 Main/Lead 共享调度，不增加一个新调度器。
2. **Main 总开关 ON**：使用原选币策略、原风险限制、原 LONG/SHORT 开平仓与通知链路。新增限制只在 Main Runner 构造时决定开仓权限。
3. **Main 总开关 OFF**：仅在本地 Main `auto_strategy` 活跃 `active/suspended/reconcile_required` Managed 周期、或 `pending/submitted/partially_filled/reconcile_required` 原订单存在时继续执行交易循环。查询只读、以 `account_id='main'`、`owner='auto_strategy'` 限制，不读取 Lead 或其他 Owner 的待处理任务。
4. 进入循环后仍读取原账户仓位/挂单，并通过现有 Ownership 校验、按原止盈止损/策略计算决定是否 CLOSE；**不把“关闭开关”理解为立即强平**，且不把交易所未知仓位自动认领为自己所有。
5. 现有 `cancelTimeoutAutoStrategyOrders`、原订单对账和真实 Fill 完成逻辑保持不变。原有币种信息读不到时明确告警并跳过该仓位，不能凭空计算退出参数。
6. 未结算订单查错与交易所网络故障仍按原逻辑记录，绝不因错误转向“再开一个单”。

**不是本阶段范围**：修复 `EnsureTradeConfigContext` 错误被 Main 忽略的问题（UF-2）；新建独立平仓机制；改变用户已创建的历史订单或手动状态。

## 三、永久测试与结果

| 测试 | 结果 |
| --- | --- |
| `TestUF1MainOFFOnlyWakesWithMainOwnedWork` | **PASS**：无仓位、Lead/其他 Owner、已完成订单和零张数仓位不唤醒；4 种未终结订单及 3 种正数仓位状态唤醒 |
| `TestUF1MainOFFDBFailureDoesNotInferSafeNoWork` | **PASS**：缺表不误判空闲；ON 不依赖 OFF 专用查询 |
| `TestUF1MainOFFLongShortExitBeforeSelection` | **PASS**：LONG 与 SHORT 在 OFF 下各唯一 CLOSE；不调用选币或开仓，只跑已归属超时订单清理 |
| `TestUF1MainONSelectionFailsButExitStillRuns` | **PASS**：选币异常不能阻断已有出场 |
| `TestUF1MainOFFHoldsWhenNoExitSignal` | **PASS**：未满足出场条件时继续持有 |
| `TestUF1MainOFFUnownedOrEmptyCannotTrade` | **PASS**：未知/外部持仓不可被自动认领或平仓 |
| `TestUF1MainONSelectionStillRunsAfterExit` | **PASS**：ON 情况正常计算新仓候选 |
| `TestUF1MainOFFUsesAutoStrategyOnlySync` | **PASS**：OFF 不自动扩展到其他 Owner；ON 不改变原同步范围 |
| `TestUF1MainOFFMissingSymbolCannotBlindlyClose` | **PASS**：原始币种配置缺失时不盲目提交 CLOSE |
| `TestLead03MainSharedRunnerRemainsUntouched` | **PASS**：Main 双开关都不借用 Lead 凭证/名单 |

| Gate | 离线结果 |
| --- | --- |
| UF-1 定向测试 `go test -count=5 -p=2` | **PASS** |
| `./feature ./service/futuresownership ./service/orderhistory ./service/systemhealth` `-count=5 -p=2` | **PASS** |
| 88 个非历史研究包 `go test -count=1 -p=2` | **PASS** |
| 四个关键包 `go test -race -count=2 -p=2` | **PASS**；macOS 链接器 LC_DYSYMTAB 警告不影响通过 |
| 同一 88 包 `go vet` | **PASS** |
| `go build -o /tmp/go_binance_futures_uf1 .` | **PASS**；约 40 MB，未部署 |
| 前端 `pnpm typecheck` | **PASS** |
| 前端 `pnpm build` | **PASS**；约 4.26 MB |
| `rsync dist → backend/static` + `diff -qr` | **PASS**；产物一致 |
| 前后端 `git diff --check` | **PASS** |

历史研究路径 `strategy_templates/research/` 包含已知的独立实验重名实现，不强行使用 `go test ./...` 声称全包通过；在 `go list ./...` 输出中排除该实验包后得 88 个可部署工程包。

## 四、仍需验证与下一阶段

- **真实环境**：尚未在 Demo/Testnet/主网使用实际持仓、部分成交、订单未明状态验证 OFF→CLOSE；源码 Mock 测试不等于成功实盘平仓。
- **安全边界**：本次只验证调度时传入的 `FutureEnable` 快照；在一个已开启的热循环与真正交易所 POST 之间发生的极短配置变更，不属于跨数据库/远端订单请求的严格原子性保障。建议 UF-2 在开仓最终校验时一并复核最新开关与交易配置。
- **原有系统故障**：全币种配置或账户行情快照不可用时仍不允许猜测平仓条件，日志记录原因；需确保现有后台告警可发现这些失败。
- **待处理的 UF-2**：Main 目前仍忽略杠杆/保证金模式设置失败；不能因为 UF-1 测试通过就把这一独立问题标记为已修复。
- 本次未操作 `conf/app.conf`、生产数据库、真实下单 API、同步迁移、服务重启或 Git 提交。

**UF-1 在源码、离线与前后端构建范围内通过。下一阶段 UF-2，单独修复配置 RPC 错误被吞掉的问题。**
