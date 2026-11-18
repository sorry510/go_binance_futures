# CODE_REVIEW_CM-6-rereview · COIN-M 自动策略调度与 2 秒旁路接线复审

> 日期：2026-11-18（北京时间）
>
> 分支：`feat/m-coin-futures`；HEAD `bd804e2 feat: cm-5`（父 `8664375 feat: cm-4`）。
>
> 本报告是**新增复审**，不覆盖实施方 `doc/trade/CODE_REVIEW_CM-6.md`。审查对象为当前工作区 15 项 CM-6 改动：主计划/自评更新、`main.go` 调度旁路、`protection.go` F2 修复，以及 12 个新增 Go 源码/测试文件。
>
> 审计期间未修改业务源码、未提交代码、未修改 `conf/app.conf`、未访问真实 Binance、未执行 `sync db`、未重启服务、未触碰真实仓位。临时探针只写在隔离副本中。

---

## 一、复审结论

| 项目 | 结论 |
| --- | --- |
| 复用 Spot 2 秒调度、不新建调度中心 | ✅ 通过 |
| CM-5 执行原语被 Runner 显式调用 | ✅ 通过 |
| 先恢复未结算订单、再处理 Managed 出场、最后评估新开仓 | ✅ 通过 |
| 双开关、私有 WS 就绪、稳定 ID、10s/8s/15s 节流、TryLock、256 事件队列 | ✅ 通过 |
| 关闭新开仓开关不等于强平；已有仓位仍可退出 | ✅ 通过 |
| 独立 DAPI Kline、跨环境/跨产品隔离、公开请求不发 API key | ✅ 通过 |
| 自评 F1～F6 永久回归 | ✅ 全部闭环 |
| 后端离线 Gate（vet/build/test/race/diff-check） | ✅ 通过 |
| 前端 CM-6 源码改动 | ✅ 无；`dist` 与后端 `static` 120 文件一致 |
| 新增发现 F7：STOP/TP 配置在信号计算中被编辑后仍允许自动开仓 | ⛔ P1，未闭环 |
| 真实 DAPI / Demo / Testnet / Mainnet / MySQL 8 / 生产发布 | ⛔ 全部 BLOCKED |
| 阶段意见 | **❌ 不通过，禁止进入 CM-7；修复 F7 后须重新复审** |

**核心判定：** CM-6 的主体接线、顺序保证和 F1～F6 修复均有源码与测试支撑，但本轮隔离探针发现一个新的 P1 资金安全缺口：`tryEntry` 在信号计算前使用旧配置做 STOP/TP 可执行性预检，信号计算后重新读取配置时只比对 `Strategy`/`Technology`，未重新校验 `Profit`/`Loss`/`TickSize`/`Leverage`。因此配置在策略求值期间被改为禁用保护后，仍会自动 POST 开仓并形成无 STOP/TP 的 Managed 仓位。此问题直接违反方案“自动开仓前保护单可执行性前置校验”的要求，不能按“有条件通过”处理。

---

## 二、基线与隔离副本

### 2.1 基线

```text
branch        feat/m-coin-futures
HEAD          bd804e2 feat: cm-5
parent        8664375 feat: cm-4
master        0b3fa34 tag v2.2.3
worktree      15 项变更（3 M + 12 ??）
frontend      11 项变更（历史/并行改动，本轮无 CM-6 源码改动）
```

CM-6 工作区改动：

```text
M  doc/trade/开发方案-CM-币本位合约交易-复刻现货精简版.md
M  main.go
M  service/coinmexecution/protection.go
?? doc/trade/CODE_REVIEW_CM-6.md
?? feature/api/binance/coin_m_klines.go
?? feature/api/binance/coin_m_klines_cm6_test.go
?? service/coinmexecution/cm6_safety_review_test.go
?? service/coinmexecution/runner_cm6.go
?? service/coinmexecution/runner_cm6_test.go
?? service/coinmexecution/runner_overflow_cm6_test.go
?? service/coinmexecution/runner_protection_cm6_test.go
?? service/coinmexecution/runner_protection_preflight_cm6_test.go
?? service/coinmexecution/runtime_cm6.go
?? service/coinmexecution/signal_identity_cm6.go
?? service/coinmexecution/strategy_cm6.go
```

### 2.2 等价校验

```text
copy          /tmp/phase_audit_cm6
SYNCED_CHANGES=15
FILES_CHECKED=1156
GO_FILES=765
STATIC_FILES=120
SHA256_MATCHED=1156
SHA256_DIFF=0
SHA256_MISSING=0
DELETED_CONFIRMED=0
APP_CONF_PRESENT=True
```

`git archive HEAD` 输出中 WorkBuddy 的 tar 提示为非致命包装信息；等价校验逐文件 SHA256 为 0 diff、0 missing。

---

## 三、静态审查

### 3.1 `main.go` 共享调度旁路

`main.go:460-473` 仍在原 Spot 2 秒 `loopRun` 中。`main.go:463-466` 新增：

```go
go coinmexecution.RunCoinMProtected(func() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    coinmexecution.ReportCoinMScheduler(coinmexecution.StartCoinMTrade(ctx))
})
```

审查结论：

- 未创建第二个调度中心，也未改动 Spot Quant 回调本身；
- CM 回调用独立 goroutine 旁路，慢 DAPI 不阻塞 Spot；
- `runtime_cm6.go:83-86` 的 `processing.TryLock()` 保证重叠 tick 直接跳过，不会堆积等待同一 Ormer 的 goroutine；
- `runtime_cm6.go:218-225` 的 `RunCoinMProtected` 捕获 panic，`ReportCoinMScheduler` 去重并脱敏日志。

隔离探针 `TestAuditCM6PanicDoesNotEscapeSchedulerGuard` 验证 panic 不外泄。

### 3.2 运行态与 WS 生命周期

`runtime_cm6.go` 的关键门禁：

```text
StartCoinMTrade
  → ctx/DB 检查
  → CoinMNeedsRuntime(orm.NewOrm())
  → 无工作直接返回（不做签名请求）
  → CoinMEnvironment() 为空则报错
  → sync.Once 初始化 runtime 并启动 streamLoop
  → runtime.Tick(ctx)
```

`CoinMNeedsRuntime`（`runner_cm6.go:58-86`）在以下任一情况才要求运行态：

1. 全局开关 ON 且至少一个 `PERPETUAL` 币种 enable；
2. 存在未平 Managed 仓位；
3. 存在待提交/未知/新/部分成交/待确认/触发中/待撤或证据不足的订单。

私有 User WS 通过 `streamLoop` 按 15 秒重新评估工作需求；事件队列为 256 条有界 channel，满时报错并要求 REST 对账，不静默丢事件。

### 3.3 Runner 顺序与资金安全

`runner_cm6.go:132-242` 的 `Tick` 顺序与方案一致：

1. `CompareAndSwap` 防重入，ctx 取消即返回；
2. 读全局开关；
3. `unsettled()` 先扫描所有不确定订单；
4. 对每个待恢复对象做 `Reconcile` / `ReconcileProtection`，每 tick 最多处理一个后返回；
5. 只有全部未结算项都是 ACTIVE PROTECT 时，才允许继续评估已有仓位；
6. 已有 Managed 仓位优先于新开仓；多个活动周期、非 OPEN 状态、未验证 Fill/mark 均 fail closed；
7. 无仓位时才检查双开关与私有 WS 就绪，按分页 round-robin 取一个币种做 entry。

`unsettled()` 读取 `LIMIT 101`，超过 100 条直接报错，避免把截断结果当成完整账本。隔离探针用 101 条 UNKNOWN 订单复验 fail closed。

### 3.4 双保护单与关闭开关边界

`managePosition`（`runner_cm6.go:352-444`）：

- 已有仓位在双开关 OFF 时仍进入出场判断；
- 只有在 `CoinMNeedsRuntime` 认定有 Managed 工作时 WS 才保持；
- 显式出场前逐个撤销本周期 ACTIVE PROTECT，并依赖 CM-5 的二次核实；失败/未知不继续 CLOSE；
- 未出场时保留 STOP/TP，不重复下保护；一个 tick 最多补一张 Algo；
- 保护价按 inverse 合约与 tickSize 分方向取整（`runner_cm6.go:450-500`）。

`protection.go` 的 F2 修复加入远端 `OrderType` 与本地 `AlgoID` 校验；`cm6_safety_review_test.go` 证明不匹配时标记 `UNKNOWN` 并阻止 CLOSE。

### 3.5 COIN-M Kline 与策略语义

`feature/api/binance/coin_m_klines.go:18-76` 只用 `deliveryClient.NewKlinesService()`，校验：

- `USD_PERP` 符号、合法 interval/limit；
- OHLC 正数、有限值、时间线递增；
- `deliveryClient.BaseURL == coinMRuntime.RESTURL`；
- 最新 Kline 新鲜度在允许窗口内。

探针确认公开 Kline 不发 `X-MBX-APIKEY`，非法 symbol/interval/limit 不发出请求，切换环境后不回落到另一 host。

`strategy_cm6.go` 中：

- ROI 阈值先于 close DSL（`strategy_cm6.go:144-154`），`close_long=true` 不能绕过 8%/6%；
- 无显式 close 规则时用 ±3% 中性带与两根 DAPI 5m Kline 确认；
- 所有启用指标必须有完整、有限的实际序列，缺失不发信号；
- LONG/SHORT 同时命中拒绝开仓。

`signal_identity_cm6.go` 的 ID 含环境、interval、base36 openTime 与 hash，探针确认跨环境、反向持仓独立且同/重叠 Kline 不重复开仓。

---

## 四、自评 F1～F6 闭环核对

| 编号 | 自评声明 | 复审证据 | 结论 |
| --- | --- | --- | --- |
| F1 | close DSL 不绕过 ROI | `cm6_safety_review_test.go`；本轮独立探针 `TestAuditCM6CloseDSLStillNeedsConfiguredROI` | ✅ 闭环 |
| F2 | 撤销保护校验远端类型与 AlgoID | `cm6_safety_review_test.go` 两组错位场景 | ✅ 闭环 |
| F3 | 101 条未结算订单 fail closed | `runner_overflow_cm6_test.go`；本轮 101 条独立探针 | ✅ 闭环 |
| F4 | 策略/指标计算中被编辑不发单 | `TestCM6RuleOrIndicatorEditedDuringEntryInvalidatesSignal` | ✅ 对 Strategy/Technology 闭环；见 F7 扩展缺口 |
| F5 | 默认 close 需 5m 确认 | `TestCM6StrategySignalsUseCorrectHedgeDirection` | ✅ 闭环 |
| F6 | 自动开仓前 STOP/TP 可执行性预检 | `runner_protection_preflight_cm6_test.go` 七组；本轮独立探针 | ✅ 对静态配置闭环；见 F7 竞态缺口 |

---

## 五、新增发现 F7（P1）：保护配置竞态仍可自动开仓

### 5.1 定位

`service/coinmexecution/runner_cm6.go:263-319`：

```go
func (r *CoinMRunner) tryEntry(ctx context.Context, coin models.DeliverySymbols) error {
    ...
    if err := coinMAutoProtectionReady(coin); err != nil { ... }   // 第 267 行，使用旧快照
    side, err := r.strategy.Entry(ctx, coin)                       // 第 270 行，期间可发生配置编辑
    ...
    cfg, err := r.symbol(coin.Symbol)                              // 第 301 行，重新读取
    ...
    if cfg.Strategy != coin.Strategy || cfg.Technology != coin.Technology {
        return nil                                                // 第 311-313 行，只比对这两项
    }
    ...
    _, err = r.processor.Open(...)                                 // 第 318 行，仍可 POST
}
```

`coinMAutoProtectionReady`（`runner_cm6.go:247-262`）检查 `Leverage`、`TickSize`、`Profit`、`Loss`。但在信号求值后重新读取 `cfg` 时，没有再次调用该函数，也没有比较这些字段。`Processor.configForEntry` 会重新读取全局开关、enable、面值、杠杆、预算与 entry 规则，但不检查 CM-6 自动开仓所需的 STOP/TP 可执行性。

### 5.2 隔离探针实证

探针：`TestAuditCM6ProtectionFieldsEditedDuringSignalStillOpen`

步骤：

1. 使用 CM-6 fixture，双开关 ON、私有 WS Fake ready；
2. fake strategy 的 `onEntry` 回调在信号计算期间执行：
   ```sql
   UPDATE delivery_symbols SET profit='0', loss='0', tickSize='0' WHERE symbol='BTCUSD_PERP'
   ```
3. `Strategy` 与 `Technology` 保持不变，因此 F4 的快照检查不会拦截；
4. `r.Tick` 仍产生 1 次 DAPI 普通单 POST；
5. 对该订单写入真实 Fill 后再 tick，新的 STOP/TP 配置无法生成触发价，`openAlgos=0`。

结果：

```text
TestAuditCM6ProtectionFieldsEditedDuringSignalStillOpen  PASS（即成功复现不安全行为）
-count=5 连续 5 次均可复现
```

### 5.3 影响与等级

- **P1**：自动开仓可在 STOP/TP 已被禁用或不可执行后发生，形成无保护的 COIN-M Managed 仓位；
- 与方案第 5 条“新开仓前再次读取两个开关、策略和技术指标快照”及自评 F6“缺失或禁用保护的配置不允许自动开仓”冲突；
- 这不是真实 DAPI 问题，而是本地调度门禁问题；Fake Gateway 已足以证明逻辑缺口。

### 5.4 最小修复与再审条件

1. 在 `tryEntry` 重新读取 `cfg` 后，用刷新后的配置再次执行 `coinMAutoProtectionReady`；
2. 将 `Leverage`/`Profit`/`Loss`/`TickSize` 与信号计算时使用的快照逐项比对，或直接用刷新后的配置作为最终 POST 依据；
3. 增加永久回归测试：信号求值期间编辑 STOP/TP 字段必须跳过本轮，不发 POST、不落 OPEN 记录；
4. 修复后重新执行完整 Gate 与 CM-6 复审，才可重新讨论 CM-7 准入。

---

## 六、离线 Gate

执行时清空 Testnet、实盘、Demo、审批与数据库 DSN 等 env；所有 HTTP Mock 均在隔离副本内完成。

### 6.1 后端

```text
git diff --check（工作区）                                  PASS
go vet ./...                                               PASS
go build -o /tmp/go_binance_futures_cm6_audit .            PASS
go test -count=1 -p=2（88 包显式列表）                     PASS
    OK=75 NO_TEST=13 FAIL=0 TOTAL=88
go test -count=5 -p=2 -run '^TestCM6' ./service/coinmexecution \
    ./service/coinmquant ./feature/api/binance ./controllers  PASS
go test -count=5 -p=2 ./service/coinmexecution \
    ./service/coinmquant ./feature/api/binance ./controllers  PASS
go test -race -count=2 -p=2 ./service/coinmexecution \
    ./service/coinmquant ./feature/api/binance ./controllers  PASS
```

macOS linker `LC_DYSYMTAB` warning 非致命；race 退出码为 0。

审计过程说明：首次全量测试因 zsh 未对未加引号的包列表变量做单词拆分，把 88 个包名当成单个参数而失败；这是审计命令错误，已改用 zsh 数组显式展开后重跑 PASS，不作为代码 Finding。

### 6.2 隔离探针

新增 10 个仅存在于隔离副本的审计探针：

```text
TestAuditCM6IdleNoRuntimeAndNoSignedPost                    PASS
TestAuditCM6UnsettledReconciliationPrecedesExit             PASS
TestAuditCM6HundredAndFirstUnsettledFailsClosed             PASS
TestAuditCM6ManagedExitStillRunsWithBothEntrySwitchesOff    PASS
TestAuditCM6ProtectionFieldsEditedDuringSignalStillOpen     PASS（复现 F7）
TestAuditCM6CloseDSLStillNeedsConfiguredROI                 PASS
TestAuditCM6InvalidProtectionPreflightBlocksSignal          PASS
TestAuditCM6StableEntryIdentityAndOverlap                   PASS
TestAuditCM6PanicDoesNotEscapeSchedulerGuard                PASS
TestAuditCM6DAPICandlesNeverCrossEnvironment                PASS
```

合计：`PROBE_RUN=10 PROBE_PASS=10 PROBE_FAIL=0`。其中 F7 探针的 PASS 表示“不安全行为被成功复现”，不是代码合格。

### 6.3 前端与静态产物

```text
frontend UI changes       11（历史/并行改动，非本轮 CM-6 源码）
frontend/dist vs static   DIFF_EXIT=0
files per tree            120 / 120
coinM live-action APIs    0（src/api/coinM.ts 仅配置/模板/只读账本）
```

本轮未重新执行前端 typecheck/build：CM-6 没有前端源码、路由或 i18n 改动，产物沿用先前已验证的一致性；`coin-m` API 模块没有下单、撤单或改仓入口。

---

## 七、未通过 / 仍 BLOCKED 的 Gate

1. **F7 修复 Gate**：STOP/TP 配置编辑竞态未闭环，禁止进入 CM-7。
2. 真实签名 DAPI：普通单、Algo、userTrades、私有 User WS、429/418、父子 Algo 触发均未现场验证。
3. Demo / Testnet / Mainnet 隔离：环境配置、host guard、真实 WS listenKey 与 keepalive 未实跑。
4. MySQL 8 v27：迁移、备份、回滚、EXPLAIN、P95 与并发 Ormer 行为未验证。
5. ARM Linux + static + schema 成套部署、持久化授权、主网小额实盘均未执行。
6. CM-5 遗留限制：`userTrades` 满 1000 笔仍 fail-closed，未实现分页；Algo 真实 OrderType、TradeSettings 等仍需真实环境确认。
7. 方案外历史 Gate 不变：现货扫描 F1、06-7 F1 与生产统一 Gate 仍需单独裁决。

---

## 八、审评意见

CM-6 的主体工程接线质量较高：没有新建调度中心，恢复/出场/开仓顺序可证，双开关不会误伤已有仓位，DAPI Kline 与 UM/Spot 隔离，F1～F6 均有永久回归测试支撑。离线 Gate 全绿。

但 F7 是一个可复现的 P1 资金安全缺口：保护配置在策略求值期间被禁用后，系统仍会自动开仓，随后无法生成 STOP/TP。依据本轮审计标准，不能把“离线全绿”等同于阶段通过。

**最终判定：CM-6 ❌ 不通过；禁止进入 CM-7；修复 F7 并完成重新复审前，不得打开真实交易开关、不得部署、不得宣称生产可用。**


---

## 九、后续复审追踪（2026-11-18）

本报告中的 **F7/P1 失败复现与“禁止进入 CM-7”判定属于修复前的原始审计结果，原样保留**。实施方随后在 `tryEntry` 最终读库处补齐 `Profit/Loss/TickSize/Leverage` 快照比对及刷新配置的保护单校验，并添加 9 种竞态注入与下一轮恢复的永久回归测试；F7 测试重复 5 次、CM-6 整体 5 次、88 个非历史研究 Go 包、Race Detector、Vet 与 Build 均已通过。

**修复后的最新结论请见 [CODE_REVIEW_CM-6-rereview-2.md](CODE_REVIEW_CM-6-rereview-2.md)**。这仅解除 F7 的源码与离线测试阻塞，仍不代表通过 MySQL 8、Demo/Testnet、私有 WS 或实盘验收。
