# CODE_REVIEW_CM-6-rereview-2 · F7/P1 修复后独立复审

> 日期：2026-11-18（北京时间）｜分支：`feat/m-coin-futures`｜HEAD：`bd804e2 feat: cm-5`（未提交 CM-6 工作区）。
>
> **此文保留 [首次复审](CODE_REVIEW_CM-6-rereview.md) 的失败事实与 P1 判定，不覆盖原始报告。** CM-6 的源码与 Fake/Mock 通过不等于 Demo/Testnet 或主网实盘可用。

## 一、F7 原因与修复

首次复审 F7/P1 证明：`service/coinmexecution/runner_cm6.go::tryEntry` 在读取旧配置后，先检查 `coinMAutoProtectionReady(coin)`，随后运行 DAPI 策略。如果在信号计算过程中修改逐币 STOP/TP，最后读取 `cfg` 时只比较 `Strategy/Technology`，仍会用过时的有效保护配置 POST 开仓，从而留下无法布设 STOP/TP 的 Managed 仓位。

**本轮最小修复**（不更改其他产品的交易或全局控制逻辑）：

- 保留原有开仓前 `coinMAutoProtectionReady(coin)`，防止旧配置已经不合法时开始策略计算。
- 策略信号计算完成并重新查询 `delivery_symbols` 后，除 `Strategy` / `Technology` 外，也逐项比较 `Profit`、`Loss`、`TickSize`、`Leverage`。**任何修改，包括改成另一个仍然有效的数值，都视为旧信号失效，当前 tick 直接跳过。**
- 针对重新读取的 `cfg` 再次执行 `coinMAutoProtectionReady(cfg)`，确保保护参数未被禁用、缺失或变成不可执行状态，才继续 CM-5 `Processor.Open`。
- 不改 `config.coin_m_quant_enable` 与 `delivery_symbols.enable` 双开关；不增加额外的数据库表、配置开关或审批系统。CM-5 `Processor.Open` 仍独立检查开关、交易设置与实际账户/远端持仓状态。

## 二、永久回归证据

新增 `service/coinmexecution/runner_protection_race_cm6_test.go`：

| 场景 | 在 Fake 策略回调内修改的逐币字段 | 修复后断言 |
| --- | --- | --- |
| 1 | `profit=0, loss=0, tickSize=0` | **不 POST / 不生成 OPEN 或 Managed 周期** |
| 2 | `loss=0` | 同上 |
| 3 | `profit=0` | 同上 |
| 4 | `tickSize=0` | 同上 |
| 5 | `leverage=0` | 同上 |
| 6 | `profit=9`（仍为有效保护） | 旧信号失效，不 POST |
| 7 | `loss=7`（仍为有效保护） | 旧信号失效，不 POST |
| 8 | `tickSize=0.2`（仍为有效价格步长） | 旧信号失效，不 POST |
| 9 | `leverage=5`（仍为有效杠杆） | 旧信号失效，不 POST |
| 10 | 保持已禁用的保护参数，再次执行 tick | fail-closed，仍不 POST |

每个场景均基于隔离 SQLite + Fake Gateway；对于已发生变更的场景，恢复有效配置后重新 tick，验证新的策略可以正常产生**唯一**新的开仓意图，而非永久阻断交易。永久回归连续 `-count=5`：**PASS**。

这覆盖了原审计隔离探针 `TestAuditCM6ProtectionFieldsEditedDuringSignalStillOpen` 的不安全复现路径；修复后的永久测试以 **POST=0、OPEN=0、Managed 周期=0** 为明确通过条件，避免将“复现漏洞成功”误判为“业务正确”。

## 三、复审后 Gate

| Gate | 本轮结果 |
| --- | --- |
| F7 两项永久测试 `-count=5` | **PASS** |
| 全部 CM-6 定向测试 `go test -count=5 -p=2 -run '^TestCM6' ./service/coinmexecution ./feature/api/binance` | **PASS** |
| 四个核心包 `go test -count=5 -p=2` | **PASS** |
| 88 个非历史研究 Go 包单次回归 | **PASS** |
| 88 包 `go vet` | **PASS** |
| 四核心包 `go test -race -count=2 -p=2` | **PASS**；macOS linker `LC_DYSYMTAB` warning 非致命 |
| `go build -o /tmp/go_binance_futures_cm6_rereview_v27 .` | **PASS** |
| `gofmt` / `git diff --check` | **PASS** |

原仓库 `strategy_templates/research/` 含历史重复函数定义，因此 88 包回归按明确的 `go list ./...` 非研究列表执行，不把无法直接构建的历史研究目录虚报为通过。

## 四、残留边界及发布禁令

- 本轮消除了**策略计算期间**修改保护配置引起的已证实 F7 竞态，未验证 MySQL 8 上配置变更与交易下单的事务级原子性。其他在最终读库以后到交易所 POST 之前发生的极窄并发窗口尚待 CM-7/真实环境进一步核查；不宣称跨数据库网络请求具备全局原子性。
- 没有真实 COIN-M DAPI 签名下单、Algo 触发、部分成交私有 WS、Testnet/Demo/主网或 MySQL 8 迁移验收。
- CM-5 遗留 `userTrades` 恰好达到 1000 笔时继续 fail-closed，未实现自动翻页。
- 没有修改 `conf/app.conf`、生产数据库、运行服务或本地开关状态，也没有提交 Git。

**复审判断**：F7/P1 在源码和隔离回归范围内已修复。完整 Gate 通过后允许进入 CM-7 代码开发；**不代表可以发布到真实交易环境或开启自动实盘**。
