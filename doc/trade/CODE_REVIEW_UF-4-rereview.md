# CODE_REVIEW_UF-4 · 复审报告

> 日期：2026-11-19。复审对象：当前工作区中的 UF-4「为什么没有开仓」实现；不覆盖实施方 `CODE_REVIEW_UF-4.md`。
>
> 开发契约：[UF · U 本位合约交易六项精简优化开发方案](开发方案-UF-U本位合约交易六项精简优化.md) §六、§九；前置边界：[UF-3 复审](CODE_REVIEW_UF-3-rereview.md)。
>
> **复审结论：UF-4 在「当前源码 + 离线 Mock/SQLite + 前后端构建」范围内 ✅ 通过，允许进入 UF-5 代码开发。**
>
> **真实 MySQL 8 / ARM Linux / Testnet / Demo / Mainnet / 生产发布 Gate 仍为 ⛔ BLOCKED。**
> 本轮未发现新的 P0/P1 Finding。UF-4 是进程内只读诊断，不产生交易信号，不改变开平仓条件。

---

## 一、复审基线与范围

### 1.1 基线

| 项目 | 值 |
| --- | --- |
| 后端仓库 | `/Users/zhz/work/binance/go_binance_futures` |
| HEAD | `80ad478` (`master`, `origin/master`) |
| 工作区总改动 | 178 项：82 D + 7 M + 89 ?? |
| 非静态 UF-4 改动 | 6 M + 5 ??：后端诊断/控制器/路由、README、方案状态、自评与测试 |
| 静态产物改动 | 82 删除 + 1 修改 + 84 新增，为 UI 全量重建同步结果 |
| 前端仓库 | `/Users/zhz/work/binance/go_binance_futrues_new_ui`，HEAD `b2508ab` |
| 前端工作区 | 9 项：UF-4 新增 3 项 + 累积 UF-3 UI 改动 6 项 |

前端工作区保留了尚未在前端仓库提交的 UF-3 UI 改动，因此本轮 `dist → static` 重建产物同时包含 UF-3 与 UF-4 UI。后端 `static/` 与当前前端 `dist` 精确一致。

本轮未修改业务代码、未 commit/push、未改 `conf/app.conf`、未执行 `sync db`、未访问真实 Binance、未重启服务、未触碰仓位。

### 1.2 隔离副本

| 检查 | 结果 |
| --- | --- |
| 后端副本 | `/tmp/phase_audit_uf4` |
| 前端副本 | `/tmp/phase_audit_uf4_ui` |
| 期望文件 | 1192 |
| 实际文件 | 1192 |
| Go 文件 | 783 |
| static 文件 | 124 |
| SHA256 差异 / 缺失 / 多余 | 0 / 0 / 0 |
| 删除确认 | 82/82 |
| `conf/app.conf` in copy | false |
| `conf/app.conf.example` in copy | true |

前次副本构建因用户中断只同步了部分静态文件（缺 46 项）；已完全删除并重建，最终校验为 0 差异。该中断不属于代码 Finding。

---

## 二、UF-4 后端静态审查

### 2.1 诊断存储

`feature/entry_diagnostics_uf4.go` 实现进程内单例：

```go
type mainEntryDiagnosticStore struct {
    sync.RWMutex
    generation    uint64
    globalReason  string
    observedAt    int64
    rows          map[string]MainEntryDiagnostic
    lastSubmitted map[string]int64
    now           func() time.Time
}
```

关键约束：

- `mainEntryDiagnosticsCap = 200`，`rows` 与 `lastSubmitted` 均有 200 条硬上限；
- `RWMutex` 保护并发写入与快照读取；
- `generation` 在每轮 `begin()` 递增，旧轮次回调通过 `s.generation != generation` 直接丢弃；
- `lastSubmitted` 跨轮次保留，因此“上次委托提交确认时间”不会被下一轮策略未命中覆盖；
- 快照固定 `account_id='main'`、`scope='current_process_last_main_cycle'`；未启动时 `global_reason='not_started'`。

### 2.2 观察器与账户隔离

```go
func beginMainEntryObservation(r *accountTradeRunner) mainEntryObserver {
    if r == nil || r.AccountID != binance.MainAccountID {
        return mainEntryObserver{}
    }
    return mainEntryObserver{generation: mainEntryDiagnostics.begin(r.AllowNewOpens), enabled: r.AllowNewOpens}
}
```

判定：

- Lead runner 返回零值观察器，`record/global/both` 均不写 Main 快照；
- Main `AllowNewOpens=false` 时只发布 `global_off`，不写逐币决策；
- `StartTrade` 在总开关 OFF 且本地无 Main 工作时直接 `mainEntryDiagnostics.begin(false)`，不会为了诊断唤醒 Binance 或加载币种；
- 已有 Managed CLOSE 仍走 UF-1 原路径，诊断不参与交易判断。

### 2.3 原因真实性与错误边界

观察点只复用本轮已经发生的判断：

- 全局：`symbols_unavailable`、`positions_unavailable`、`open_orders_unavailable`、`ownership_sync_failed`、`selector_unavailable`、`no_candidate`、`max_losing_positions`、`max_positions`、`direction_off`；
- 逐币：`symbol_off`、`not_evaluated`、`existing_position`、`pending_order`、`strategy_not_matched`、`depth_unavailable`、`invalid_quantity`、`trade_config_failed`、`order_submit_failed`、`order_submitted`。

其中：

- `no_candidate` 仅表示启用币种未进入本轮实际 `SelectCoins` 结果；不猜测 `volume_low`、`market_data_stale` 或 `cooldown`；
- `trade_config_failed` 来自 UF-2 已发生的 `EnsureConfig` 错误；
- `order_submitted` 只表示 `SubmitOpen` 返回无错误，即委托 ACK，**不代表 Fill 或成交**；
- 原始异常文本不进入 API 快照；探针确认 mock 错误中的 `AUDIT_SECRET_TOKEN_8848` 不出现在 JSON 响应中。原错误仍保留在既有日志中，属于原行为，不是 UF-4 新增 API 泄漏。

### 2.4 热路径不改交易行为

`feature/feature.go` 的差异均为在既有分支追加 `observer.record/global/both` 调用，或把原循环中已存在的持仓/挂单检查提前到 `EvaluateEntry` 之后用于观察；没有新增：

- `SubmitOpen` / `SubmitClose`；
- 选币或策略二次计算；
- Binance REST/WS 调用；
- ORM 写入或数据库表；
- 对原 `continue`、`break`、`return`、`Sleep` 顺序的改写。

### 2.5 接口与路由

```text
GET /futures/entry-diagnostics
```

```go
web.Router("/futures/entry-diagnostics", &controllers.MainEntryDiagnosticsController{}, "get:Get")
```

控制器行为：

- JWT claims 缺失或 `Username` 为空 → 401 `entry_diagnostics_auth_required`；
- `symbol` 仅允许大写字母/数字且长度 ≤ 32；`position_side` 仅允许空 / `LONG` / `SHORT`；否则 400；
- 成功只返回 `feature.MainEntryDiagnostics(symbol, side)` 的内存快照；
- 控制器与快照函数不含 `orm.`、`binance.`、`.Raw(`、`.Exec(`、`SelectCoins`、`EvaluateEntry`、`SubmitOpen`、`SubmitClose`。

---

## 三、UF-4 前端审查

- `src/api/trade.ts`：新增只读 `getFuturesEntryDiagnostics`，使用 `skipProgress` 避免轮询闪动全局进度条。
- `src/views/futures/EntryDiagnostics.vue`：默认收起；展开后首次读取并每 15 秒轮询，收起或卸载时 `clearInterval`；带 `serial` 防止旧响应覆盖；超过 30 秒显示过期提示。
- `src/views/futures/symbol.vue`：仅在现有合约交易币种页嵌入组件，无新顶级菜单。
- `locales/zh-CN.yaml` / `en.yaml`：双语原因码、范围说明与 ACK 非 Fill 提示。
- 未发现下单、撤单、认领、配置修改或修复类请求。

---

## 四、Gate 结果

| Gate | 结果 |
| --- | --- |
| 工作区 `git diff --check` | ✅ exit 0 |
| `go vet`（88 包） | ✅ exit 0 |
| `go build -o /tmp/go_binance_futures_uf4_audit .` | ✅ exit 0 |
| 全量 `go test -count=1`（88 包） | ✅ 75 ok + 13 no test + 0 FAIL |
| UF-4 专项 `go test -count=5 -run '^TestUF4'` | ✅ exit 0 |
| 5 个关键包 `go test -count=5` | ✅ exit 0 |
| 5 个关键包 `go test -race -count=2` | ✅ exit 0 |
| 前端隔离副本 `pnpm typecheck` | ✅ exit 0 |
| 前端隔离副本 `pnpm build` | ✅ exit 0，4.29 MB |
| 重建 `dist` 与后端 `static/` | ✅ 124/124 文件一致 |

说明：race 中 macOS `LC_DYSYMTAB` linker warning 非致命；日志中的 `COIN-M DAPI configured ...` 为初始化打印，不是审计期间真实 DAPI 请求。

---

## 五、隔离探针

| 探针 | 覆盖点 | 结果 |
| --- | --- | --- |
| `TestAuditUF4DiagnosticNeverLeaksRawErrorText` | 选币与委托失败原始错误文本不进入 API 快照 | ✅ PASS |
| `TestAuditUF4SnapshotDoesNotTriggerExtraTradingCalls` | 1000 次快照读取不增加 OPEN/CLOSE/配置调用 | ✅ PASS |
| `TestAuditUF4MemoryAndGenerationAreStrictlyBounded` | 350 条写入后行数与 ACK 时间均 ≤200；旧 generation 不污染新轮次 | ✅ PASS |
| `TestAuditUF4LeadObserverCannotOpenOrOverwriteMain` | Lead 无法取得 Main 观察器或改写快照 | ✅ PASS |
| `TestAuditUF4EntryDiagnosticsRouteIsGetOnlyAndReadOnly` | 路由仅 GET；控制器无 DB/Binance/策略/交易 token | ✅ PASS |
| `TestAuditUF4UnauthorizedEntryDiagnosticsReturnsNoPayload` | 无 JWT 401 且无快照字段；授权后返回 main scope | ✅ PASS |

6 个运行节点全部 PASS；`go test -count=5 -run '^TestAuditUF4' ./feature ./controllers` exit 0。探针只写在隔离副本，未污染工作区。

审计探针曾因前置条件缺失（未先写入 `order_submitted` 就检查跨轮次保留）失败一次；已在隔离副本修正后 PASS，不属于代码 Finding。

---

## 六、观察项与未通过 Gate

### 6.1 观察项（不阻断本阶段）

1. **O1**：前端工作区含累积 UF-3 UI 改动；本轮静态产物是 UF-3+UF-4 一次性重建，不能表述为“仅 UF-4 产物”。
2. **O2**：诊断仅存在当前进程内存，重启丢失，非历史事件或多实例统一视图。
3. **O3**：`invalid_quantity` 是观察标签，不改变原交易路径；若后续提交成功，最终原因会变为 `order_submitted`。
4. **O4**：未入选候选币种只记录 `no_candidate`，不推断成交量/行情过期等未经实际执行的细粒度原因。
5. **O5**：原始错误文本仍写入既有日志；API 快照不携带，但日志访问控制需由既有运维边界负责。
6. **O6**：`order_submitted` 是委托 ACK，页面文案已明确“不代表已成交”，不得用于 Fill/收益统计。

### 6.2 仍然 BLOCKED

| Gate | 状态 |
| --- | --- |
| 真实 MySQL 8 账本、权限与 P95 | ⛔ 未验 |
| ARM Linux 部署包、`static` 成套发布、回滚 | ⛔ 未验 |
| 真实 Binance FAPI / 私有 WS / 账户数据 | ⛔ 未验 |
| Testnet / Demo / Mainnet 实际开仓与诊断一致性 | ⛔ 未验 |
| 多实例部署下诊断来源定位 | ⛔ 未验 |
| 生产持久化授权、审计、回滚演练 | ⛔ 未验 |

历史未决事项不因本轮通过而关闭：现货扫描 F1、06-7 F1、CM-7 F8/P1、UF 真实账户与 MySQL 8 Gate。

---

## 七、最终判定

```text
UF-4：✅ 源码 / 离线 Mock / 前后端构建 Gate 通过

允许：进入 UF-5（实盘策略效果统计）代码开发。
禁止：把进程内诊断视为账户真相、成交证明、多实例监控、生产发布授权或真实 Fill 统计数据源。
```

本复审不修改历史报告，不替代真实交易所与生产发布 Gate。若后续让诊断参与交易决策、接入真实账户数据、改为持久化或多实例聚合，应另起复审并记录证据。
