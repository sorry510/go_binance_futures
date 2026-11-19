# CODE_REVIEW_UF-2 · Main 合约杠杆/保证金配置失败阻止开仓

> 日期：2026-11-18；后端分支 `master`。依据：[UF 六项精简优化方案](开发方案-UF-U本位合约交易六项精简优化.md)、[UF-1 Code Review](CODE_REVIEW_UF-1.md)。
>
> **结论：源码修复与隔离测试已完成；真实 Binance Main 账户和正式环境未验收。**
> 本阶段不修改数据库 schema、不执行迁移、不连接真实交易所下单、不重启服务、不提交 Git。

## 一、原始故障与根因

UF-1 后，Main 的 `feature/account_trade_cycle.go::mainTradeRunner.EnsureConfig` 仍将 `feature.UpdateSymbolTradeInfoContext` 的结果固定包装为 `nil`。后者又使用 `_ = binance.EnsureTradeConfigContext(...)` 忽略 Binance 配置接口错误。

因此，当 Binance 拒绝保证金模式变更、杠杆设置失败、出现网络错误/超时或账户设置无权修改时，执行循环可能仍进入 `SubmitOpen`。账户实际杠杆/保证金配置不一定等于策略预算计算时使用的值。

## 二、代码变更

| 文件 | 处理 |
| --- | --- |
| `feature/feature.go` | `UpdateSymbolTradeInfoContext` 改为返回 `error`；拒绝 nil、无效保证金模式或非正数杠杆。原本遗留的 `UpdateSymbolTradeInfo` 无返回值包装函数保留，失败时记日志而不伪称成功 |
| `feature/account_trade_cycle.go` | Main `EnsureConfig` 改用 `ensureMainOpenConfiguration(ctx, coin, cfg)`，真实错误直达共享开仓循环。原有 LONG/SHORT `EnsureConfig` 失败 `continue` 分支执行，没有提交 OPEN |
| `feature/main_open_config_uf2.go` | Main 新开仓专用的两段校验：先校验总/币种开关、保证金模式/杠杆，再执行既有 Binance 配置服务；完成后**重新读数据库**确认 `config` 与 `symbols` 的交易参数仍与本轮快照一致 |
| `feature/api/binance/trade_config_cache.go` | 保留现有 **5 秒成功缓存 + singleflight**，但不再仅把 HTTP 无错误视为杠杆已生效；要求 `SymbolLeverage` 回传的 Symbol 和 Leverage 均与请求匹配。错误/空响应不缓存成功 |
| `feature/api/binance/trade_config_cache_test.go` | 既有配置变化缓存测试根据请求返回对应杠杆（4→5），继续验证实际成功后方可缓存 |
| `feature/main_open_config_uf2_test.go` | Main 真实 Runner 接线测试 + 纯 Mock/内存快照边界、LONG/SHORT 失败不 POST、已有 Managed CLOSE 不依赖开仓设置 |
| `feature/api/binance/trade_config_uf2_test.go` | 仅对本地 `httptest.Server` 模拟的 FAPI endpoint 进行签名请求，验证 -4046、保证金拒绝、杠杆拒绝、杠杆 ACK 不一致、无效响应、超时和成功缓存 |

**独立作用范围：** U 本位 Main 新 OPEN。没有修改已有 Managed CLOSE、Lead 专用 AccountClient、COIN-M DAPI、Spot 交易 API、保护单/真实 Fill 记录逻辑或共享 2 秒调度。

## 三、最终开仓门禁执行顺序

1. Main 当前开启新开仓，逐币仍启用、币种/保证金类型/杠杆有效。
2. 只对新 OPEN 执行 `EnsureTradeConfigContext`：保证金模式返回 `-4046`（交易所确认已是目标模式）可接受；真正 HTTP/API 权限或网络错误阻断，随后设置杠杆。
3. 杠杆响应必须包含正确 `symbol` 与 `leverage`。只有**两个配置步骤均明确成功**才缓存 5 秒，减少重复 Binance 请求。不能用错误响应创建成功缓存。
4. 远端 RPC 完成后重新读取 `symbols` 与 `config`；若当前总开关或逐币开关 OFF，或者杠杆、保证金模式、USD 预算、精度、策略、技术指标、止盈/止损、风险/方向设置发生变化，**丢弃本轮旧信号**，等待下一次策略计算。
5. 两段确认成功后才由原 Main Runner 调用 `SubmitOpen`。原 `SubmitOpen` 内部订单归属、远端订单唯一 ID 和账户风险控制维持不变。

**不是跨系统严格原子事务：** 这是提交 OPEN **之前**的两次门禁，并未在数据库更改与 HTTP POST 之间建立分布式锁；最终读库到实际 POST 之间的极小竞态不宣称彻底消失。原 Binance 成功缓存 TTL=5s 也不是交易所账户状态永久证明。

## 四、永久测试

- `TestUF2MainConfigureFailureFailsClosedAndNeverReadsStaleSnapshot`：margin、leverage、429、timeout、权限等错误原样传播；失败后不继续信号/提交阶段。
- `TestUF2MainFinalConfigSnapshotRejectsAllTradeCriticalEdits`：20 种本轮配置变化（总/逐币开关、杠杆/模式、预算/精度、策略/技术指标、止盈/止损、交易方向、订单类型与风险上限）均丢弃旧信号。
- `TestUF2MainConfigPositiveCacheOrValidMarketUpdatesStillAllow`：合法配置且行情 WS 更新并不会无故拒绝交易。
- `TestUF2MainInvalidConfigStopsBeforeRPC`、`TestUF2MainConfigSnapshotUnavailableBlocks`：无效币种/开关或数据库失败时不冒险提交。
- `TestUF2MainLongShortRPCFailureNeverSubmitsOpen`：LONG 与 SHORT 均验证配置失败导致 OPEN=0。
- `TestUF2FailedEntryConfigDoesNotBlockManagedExit`：已有 Managed CLOSE 继续按 UF-1 原流程执行。
- `TestUF2ProductionMainRunnerDoesNotDiscardPreflightError`：真正 Main Runner 实例不吞掉前置校验错误。
- `TestUF2ExchangeMarginLeverageFailuresNeverGetSuccessCache`：本地 FAPI 模拟服务器校验各种 Binance 回复（含 `-4046`、错误 symbol/leverage、空响应）和 5 秒缓存。
- `TestUF2CanceledTradeConfigFailsWithoutSuccessfulCache`：取消请求不会缓存成功。

## 五、测试结果

| Gate | 结果 |
| --- | --- |
| UF-2 专项 `go test -count=5 -p=2 -run '^TestUF2' ./feature ./feature/api/binance` | **PASS** |
| 主链路、Main/Lead Ownership、FAPI、订单历史和系统健康 5 包 `go test -count=5 -p=2` | **PASS** |
| 除历史 `strategy_templates/research/` 外的所有可部署 Go 包 | **PASS** |
| 上述工程 Go 包 `go vet` | **PASS** |
| 五个关键包 `go test -race -count=2 -p=2` | **PASS** |
| `go build -o /tmp/go_binance_futures_uf2 .` | **PASS** |
| `gofmt`、`git diff --check` | **PASS** |

## 六、UF-1 前端改动的清理

按用户此前意见，已将 UF-1 额外新增的配置中心提示与其 zh/en 语言文件恢复到前端 `HEAD`，并将 UF-1 当时整体前端构建带来的后端 `static` 已暂存差异恢复到后端 `HEAD`。仅恢复这些确认为 UF-1 的源文件和构建输出，其他 Lead、Spot、COIN-M 源码及后端 UF-1 功能保留不变。

本轮 **没有执行新的前端 build，也没有重新同步 `dist` 到 `static`**。如未来要发布包含其他前端模块变更的版本，应单独审查并构建。

## 七、待完成与下一阶段

- 真实 Main 币安账户测试网/小额实盘、配置失败后持仓状态、额外持仓模式/权限变化场景未现场验证。
- 本阶段未操作 `conf/app.conf`、数据库、真实下单、生产服务或 Git 提交。
- UF-2 源码与离线 Gate 通过后，可进入 **UF-3（异常订单定位）**。上线前应独立对交易所签名账户、真实余额、自动开仓与平仓执行门禁验收。
