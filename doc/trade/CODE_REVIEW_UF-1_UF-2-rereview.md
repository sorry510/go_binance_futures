# CODE_REVIEW_UF-1 / UF-2 · 复审报告

> 日期：2026-11-19。复审对象：**当前工作区中的 UF-1 与 UF-2**；不覆盖实施方自评 `CODE_REVIEW_UF-1.md`、`CODE_REVIEW_UF-2.md`。
>
> 开发契约：[UF · U 本位合约交易六项精简优化开发方案](开发方案-UF-U本位合约交易六项精简优化.md)。
>
> **复审结论：UF-1 / UF-2 在「当前源码 + 离线 Mock/SQLite + 工程构建」范围内 ✅ 通过，允许进入 UF-3 代码开发。**
>
> **实盘 / Testnet / 主网 / 真实 MySQL 8 / ARM Linux / 生产发布 Gate 仍为 ⛔ BLOCKED。**
> 本轮未发现新的 P0/P1 Finding；既有自评中已声明的「最终读库到真实 POST 之间非原子竞态」属于已知边界，不作为本轮阻断项。

---

## 一、复审基线与范围

### 1.1 基线

| 项目 | 值 |
| --- | --- |
| 后端仓库 | `/Users/zhz/work/binance/go_binance_futures` |
| HEAD | `5c31391` (`master`, `v2.2.4`) |
| 工作区改动 | 14 项（UF 后端源码/测试/文档；另有历史 staged 状态） |
| 前端仓库 | `/Users/zhz/work/binance/go_binance_futrues_new_ui` |
| 前端 HEAD | `b2508ab` |
| 前端工作区 | `git status --short` 为空 |
| 后端 `static/` | `git status --short -- static` 为空 |
| 隔离副本 | `/tmp/phase_audit_uf` |
| 隔离编译产物 | `/tmp/go_binance_futures_uf_audit` |

当前 UF 改动包含：

- `feature/account_trade_cycle.go`
- `feature/feature.go`
- `feature/main_open_config_uf2.go`
- `feature/api/binance/trade_config_cache.go`
- `feature/uf1_main_exit_test.go`
- `feature/main_open_config_uf2_test.go`
- `feature/api/binance/trade_config_uf2_test.go`
- 相关既有测试与中英文文档、自评文档

本轮复审**没有修改业务代码、没有 commit/push、没有改 `conf/app.conf`、没有改 `static/` 或 `strategy_templates/`、没有执行数据库迁移、没有重启服务、没有连接真实 Binance 下单**。

### 1.2 隔离副本等价性

隔离副本按 `git archive HEAD` + 同步 14 项 `M/A/??` 工作区改动建立，并逐文件 SHA256 比对：

| 检查 | 结果 |
| --- | --- |
| 同步改动数 | 14 |
| 检查文件数 | 1177 |
| Go 文件数 | 776 |
| static 文件数 | 120 |
| SHA256 一致 | 1177 |
| SHA256 差异 | 0 |
| 缺失文件 | 0 |
| 已删除文件确认 | 0 |
| `app.conf` 存在性 | 由 `app.conf.example` 兜底（未使用真实配置） |

结论：隔离副本中的测试结论可代表当前工作区源码状态。

### 1.3 前端与静态产物边界

- 前端工作区 0 改动，后端 `static/` 0 改动。
- `diff -qr frontend/dist backend/static` 仍有历史差异；复核确认这是 master/CM 合并后的基线产物差异，**不是 UF-1/UF-2 引入**。
- 因此本轮不重新执行 `pnpm typecheck` / `pnpm build`，也不执行 `dist → static` 同步；UF-1 自评中记录的前端改动已按用户要求撤回。

---

## 二、UF-1 复审：总开关仅阻止新仓，已有 Managed 仓位继续退出

### 2.1 验收契约

方案要求：

1. Main `FutureEnable` 关闭后不得新开仓；
2. 已有 Main `auto_strategy` Managed 仓位 / 未终结订单仍可恢复、核实 Fill、按原规则退出；
3. OFF 且确无本地 Main auto_strategy 工作时，不额外发 DAPI 账户轮询；
4. 本地账本读取失败必须 fail-closed，不得推断为「无仓位」；
5. 选币失败或候选为空不得阻断已有仓位退出；
6. 不得接管 Lead、`new_coin_rush`、`funding_rate` 或手动/未知 Owner 仓位；
7. OFF 不是强平，也不是取消全部保护单。

### 2.2 实现证据

`mainTradeRunner` 将开关仅作为 entry gate，并把 OFF 时同步范围压缩到 `auto_strategy`：

```go
entryEnabled := cfg != nil && cfg.FutureEnable == 1
syncPositions := syncStrategyExitPositions
if !entryEnabled {
    syncPositions = syncAutoStrategyOwnership
}
return &accountTradeRunner{
    AccountID: binance.MainAccountID, Config: cfg, AllowNewOpens: entryEnabled,
    ...
}
```

本地唤醒门禁只查 Main `auto_strategy` 的本地账本：

```go
SELECT EXISTS(SELECT 1 FROM futures_managed_positions
  WHERE account_id=? AND owner=?
    AND managed_qty>0.000000000001
    AND status IN ('active','suspended','reconcile_required'))
```

```go
SELECT EXISTS(SELECT 1 FROM futures_managed_orders
  WHERE account_id=? AND owner=?
    AND status IN ('pending','submitted','partially_filled','reconcile_required'))
```

且查询失败时返回 error：

```go
if err != nil {
    logs.Warning("Main futures exit/recovery work gate unavailable: ", err)
}
if runMain && err == nil {
    runAccountTradeCycle(mainTradeRunner(systemConfig))
}
```

共享热路径把候选选币移到 Managed 出场与超时订单处理之后，并在 OFF 时直接退出：

```go
// Exit/reconcile (and owned timeout-order cleanup) is always allowed
// even while Main/Lead NEW entries are paused.
if !runner.AllowNewOpens {
    return
}
coins, selectionErr := runner.SelectCoins(systemConfig, allCoins)
```

### 2.3 判定

| 验收点 | 判定 | 证据 |
| --- | --- | --- |
| `FutureEnable` 成为 entry-only gate | ✅ | `AllowNewOpens: entryEnabled`，主循环在 `!AllowNewOpens` 时 return |
| OFF 只唤醒 Main `auto_strategy` 工作 | ✅ | 两条 `EXISTS` 均限定 `account_id='main'` 与 `owner='auto_strategy'` |
| DB 失败 fail-closed | ✅ | `mainAutoStrategyWorkNeeded` 返回 error，`StartTrade` 不进入交易循环并告警 |
| 主动退出不依赖选币 | ✅ | `SelectCoins` 位于出场/超时订单清理之后；OFF 时根本不调用 |
| LONG/SHORT CLOSE 唯一 | ✅ | 永久测试与隔离探针均覆盖 |
| 未知/手动仓位不可冒认 | ✅ | 查询固定 owner；同步范围 `syncAutoStrategyOwnership` |
| Lead/Spot/COIN-M 不受影响 | ✅ | 未改 Lead AccountClient、Spot 与 DAPI 路径；Lead parity 测试保留 |
| OFF 不等于强平 | ✅ | 仍走原 `EvaluateExit` / `SubmitClose`，未新增强制平仓或全量撤单 |

**UF-1 判定：✅ 源码与离线范围通过。**

---

## 三、UF-2 复审：配置 RPC 失败必须阻止新开仓

### 3.1 验收契约

1. `EnsureTradeConfigContext` 错误不得再被吞掉；
2. margin/leverage 网络、权限、超时、429、非法参数均 fail-closed；
3. 成功缓存只能在明确成功后写入；
4. 远端 RPC 期间配置被编辑时不得用旧信号发单；
5. 只阻断新 OPEN，不阻断已有 Managed CLOSE / 恢复；
6. 错误文本不得泄漏 API key、签名 URL 或敏感响应体。

### 3.2 错误传播链

配置函数改为返回 error：

```go
func UpdateSymbolTradeInfoContext(ctx context.Context, symbols *models.Symbols) error {
    ...
    if err := binance.EnsureTradeConfigContext(ctx, symbols.Symbol, marginType, int(symbols.Leverage)); err != nil {
        return fmt.Errorf("Main futures margin/leverage not confirmed for %s: %w", symbols.Symbol, err)
    }
    return nil
}
```

Main Runner 不再把错误固定包成 nil：

```go
EnsureConfig: func(ctx context.Context, coin *models.Symbols) error {
    return ensureMainOpenConfiguration(ctx, coin, cfg)
},
```

共享循环在 `SubmitOpen` 前检查：

```go
if err := runner.EnsureConfig(ctx, coin); err != nil {
    logs.Error("account trade configuration rejected:", err)
    continue
}
_, err := runner.SubmitOpen(...)
```

LONG 与 SHORT 两个分支均有该前置校验；失败时 `continue`，不会进入对应 `SubmitOpen`。

### 3.3 5 秒缓存写入条件

原 HTTP 200 即成功的问题已修复；现在要求杠杆 ACK 明确匹配请求：

```go
leverageAck, leverageErr := SetLeverageContext(ctx, symbol, leverage)
if leverageErr != nil {
    return nil, leverageErr
}
if leverageAck == nil || leverageAck.Leverage != leverage ||
    !strings.EqualFold(leverageAck.Symbol, symbol) {
    return nil, fmt.Errorf("futures leverage confirmation mismatch for %s", symbol)
}
runtimeTradeConfigCache.items[symbol] = runtimeTradeConfigEntry{...}
```

`-4046` 仍作为「保证金模式已经相同」接受；其它 margin/leverage 错误直接返回；失败和取消请求不进入成功缓存。缓存 TTL 仍为 5 秒，singleflight 保留。

### 3.4 RPC 后最终快照复检

`ensureMainOpenConfigurationWith` 先校验总/逐币开关与 margin/leverage，再调用交易所配置服务，RPC 成功后重新读库：

```go
current, latestConfig, err := read(ctx, coin.Symbol)
if err != nil {
    return fmt.Errorf("Main futures final entry settings unavailable: %w", err)
}
return compareMainOpenConfigSnapshot(coin, current, config, latestConfig)
```

复检覆盖：

- `symbols`：`Symbol`、`ID`、`Enable`、`MarginType`、`Leverage`、`Usdt`、`TickSize`、`StepSize`、`Strategy`、`Technology`、`Profit`、`Loss`、`Type`、`KlineInterval`、`StrategyTemplateID`；
- `config`：`FutureEnable`、`FutureAllowLong`、`FutureAllowShort`、`FutureOrderType`、`FutureMaxCount`、`LossMaxCount`；
- 再次确认最新 `MarginType` / `Leverage` 合法。

快速变动的 WS 行情字段不进入比较，因此正常行情更新不会误杀信号。该设计符合「不要增加复杂限定逻辑」的约束。

### 3.5 判定

| 验收点 | 判定 | 证据 |
| --- | --- | --- |
| 配置错误从 FAPI 传到 Main Runner | ✅ | `EnsureTradeConfigContext` → `UpdateSymbolTradeInfoContext` → `EnsureConfig` → `continue` |
| margin/leverage 无效在 RPC 前阻断 | ✅ | `validateMainMarginLeverage` 拒绝非 `ISOLATED/CROSSED` 与非正杠杆 |
| `-4046` 可接受 | ✅ | `isMarginTypeAlreadyConfigured` 仅匹配 API error code |
| ACK 不一致/空响应不缓存 | ✅ | 显式比较 `LeverageAck.Leverage` 与 `Symbol` |
| 5 秒成功缓存保留 | ✅ | 仅成功后写 TTL；`hasRuntimeTradeConfig` 匹配 symbol/margin/leverage |
| RPC 期配置编辑丢弃旧信号 | ✅ | `compareMainOpenConfigSnapshot` 覆盖 20 类交易关键字段；隔离探针实证 |
| 不阻断已有退出 | ✅ | `EnsureConfig` 只在 OPEN 分支调用；UF-2 永久测试 `FailedEntryConfigDoesNotBlockManagedExit` |
| Lead/COIN-M/Spot/手动仓位隔离 | ✅ | 改动限定在 Main FAPI 新开仓前置路径；无共享授权/订单归属改写 |

**UF-2 判定：✅ 源码与离线范围通过。**

---

## 四、Gate 执行结果

所有 Go Gate 均在 `/tmp/phase_audit_uf` 隔离副本内执行；命令显式清空相关 `BINANCE_*`、`SPOT_*`、`COIN-M` 环境，避免审计期间联网或触发真实交易。

| Gate | 结果 |
| --- | --- |
| `git diff --check` | ✅ exit 0 |
| `go vet`（88 个可部署工程包） | ✅ exit 0 |
| `go build -o /tmp/go_binance_futures_uf_audit .` | ✅ exit 0 |
| 全量 `go test -count=1`（88 包） | ✅ 75 ok + 13 no test files + 0 FAIL |
| UF 专项 `go test -count=5 ./feature ./feature/api/binance` | ✅ exit 0 |
| 核心 5 包 `go test -count=5` | ✅ exit 0 |
| 核心 5 包 `go test -race -count=2` | ✅ exit 0 |
| 隔离 UF 探针 `go test -count=5 -run '^TestAuditUF' ./feature` | ✅ exit 0 |

说明：

- 全量测试排除历史 `strategy_templates/research/` 实验包；这不是 UF 引入的排除项。
- race 日志含 macOS `LC_DYSYMTAB` linker warning；退出码为 0，判定为非致命告警。
- 探针日志中的 `COIN-M DAPI configured: environment= mainnet ...` 是配置初始化打印，**不是审计期间真实 DAPI 请求或下单证据**；实际探针使用 SQLite/Mock/内存快照。

---

## 五、隔离探针

探针只写在隔离副本 `/tmp/phase_audit_uf/feature/audit_uf_probe_test.go`，未污染用户工作区。

| 探针 | 覆盖点 | 结果 |
| --- | --- | --- |
| `TestAuditUF1OnlyMainOwnedWorkWakesOFFCycle` | 空闲不唤醒、Main-owned 仓位唤醒、缺表 fail-closed、ON 不查 OFF gate | ✅ PASS |
| `TestAuditUF1OFFExitDoesNotWaitForCandidateSelection` | OFF 下 LONG/SHORT 各唯一 CLOSE，选币 0 次、OPEN 0 次、超时订单清理仍执行 | ✅ PASS |
| `TestAuditUF2ConfigEditsDuringRPCDiscardOldSignal` | RPC 期间杠杆与总开关被改，旧信号被丢弃 | ✅ PASS |
| `TestAuditUF2InvalidMarginOrLeverageStopsBeforeExchangeRPC` | nil/非法 margin/0/负杠杆在配置 RPC 前拒绝 | ✅ PASS |

运行输出：

```text
--- PASS: TestAuditUF1OnlyMainOwnedWorkWakesOFFCycle
--- PASS: TestAuditUF1OFFExitDoesNotWaitForCandidateSelection
    --- PASS: .../LONG
    --- PASS: .../SHORT
--- PASS: TestAuditUF2ConfigEditsDuringRPCDiscardOldSignal
--- PASS: TestAuditUF2InvalidMarginOrLeverageStopsBeforeExchangeRPC
ok  go_binance_futures/feature
```

4 个测试函数、6 个运行节点全部 PASS；`-count=5` 重复验证同样 PASS。

---

## 六、观察项与未通过 Gate

### 6.1 已知边界（不阻断本阶段）

1. **非严格原子性**：UF-2 的最终读库发生在真实交易所 POST 之前，但不持有跨数据库/HTTP 的分布式锁；用户在极窄窗口内再次改配置时，不宣称绝对不可能使用旧快照。实施方自评已明确该边界，方案也禁止引入复杂锁/队列，本复审不新增 P1。
2. **5 秒缓存是成功提示，不是账户永久证明**：User WS、其它路径或交易所侧外部变更需依赖 `InvalidateTradeConfig` 或下一次 RPC；TTL 期内不是永久强一致。
3. **本地账本不可读时 OFF 暂停 Main 循环**：这是刻意 fail-closed，不会冒认仓位，但真实生产若有 schema/权限故障，需要系统告警能及时发现；该告警日志已存在。

### 6.2 仍然 BLOCKED

| Gate | 状态 |
| --- | --- |
| 真实 Binance Main Testnet / Demo 账户 | ⛔ 未验 |
| 真实主网小额 OPEN/CLOSE、部分成交、未明委托 | ⛔ 未验 |
| 真实 FAPI margin/leverage 权限、429、超时、`-4046` 响应 | ⛔ 仅 Mock 验证 |
| MySQL 8 真实账本、索引、P95、备份/回滚 | ⛔ 未验 |
| ARM Linux 部署包与 `static` 成套发布 | ⛔ 未验 |
| 生产持久化授权、配置变更审计、回滚演练 | ⛔ 未验 |
| 真实账户中 Lead / COIN-M / Spot / 手动仓位并行隔离 | ⛔ 未验 |

历史未决事项不因本轮通过而关闭：现货扫描 F1、06-7 F1、CM-7 F8/P1 仍按既有报告处理。

---

## 七、最终判定

```text
UF-1：✅ 源码 / 离线 / 工程 Gate 通过
UF-2：✅ 源码 / 离线 / 工程 Gate 通过

允许：进入 UF-3（异常订单定位）代码开发。
禁止：宣告实盘安全、打开真实自动交易开关、部署生产、在主网账户验证前把 Mock/SQLite 结果当作实盘结果。
```

本复审不修改历史报告，不替代真实交易所与生产发布 Gate。若后续出现真实 FAPI 响应差异、MySQL 8 行为差异或新增前端改动，应另起复审并重新记录证据。
