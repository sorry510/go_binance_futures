# CODE_REVIEW_CM-7-rereview · COIN-M 四页面、运行状态只读 API、E2E 与发布前文档复审

> 日期：2026-11-18（北京时间）
>
> 分支：`feat/m-coin-futures`；HEAD `63efbe8 feat: cm-6`（父 `bd804e2 feat: cm-5`）。
>
> 本报告是**新增复审**，不覆盖实施方 `doc/trade/CODE_REVIEW_CM-7.md`。审查对象为当前工作区 173 项改动：CM-7 只读运行状态 API、控制器与路由、LONG/SHORT Fake E2E、状态 API 测试、中英文 README 与页面文案、全部静态产物重建。
>
> 审计期间未修改业务源码、未提交代码、未修改 `conf/app.conf`、未访问真实 Binance、未执行 `sync db`、未重启服务、未触碰真实仓位。临时探针只写在隔离副本中。

---

## 一、复审结论

| 项目 | 结论 |
| --- | --- |
| 四个 COIN-M 独立页面与菜单顺序 | ✅ 通过 |
| `/coin-m/runtime-status` JWT 鉴权与本地只读语义 | ✅ 通过 |
| 页面不暴露手动下单/撤单/平仓/归属操作 | ✅ 通过 |
| 中英文 README 与运行前检查清单表述 | ✅ 通过 |
| LONG/SHORT Fake 开仓→保护→出场→结算 E2E | ✅ 通过 |
| CM-6 F7 在“策略求值期”修复 | ✅ 通过 |
| 新增 F8：CM-5 Processor 签名预检期仍可编辑保护配置并自动开仓 | ⛔ P1，未闭环 |
| 后端离线 Gate | ✅ 通过 |
| 前端 typecheck 与 dist/static 一致性 | ✅ 通过 |
| 真实 DAPI / Demo / Testnet / Mainnet / MySQL 8 / 发布 | ⛔ 全部 BLOCKED |
| 阶段意见 | **❌ 不通过，禁止宣告 CM 开发完成；修复 F8 后须重新复审** |

**核心判定：** CM-7 的页面、文档、只读状态 API 与离线 E2E 主体完成，CM-6 F7 的原复现路径也已修复。但 CM-6 二轮复审明确遗留的“最终读库以后到交易所 POST 之前”并发窗口，在本轮隔离探针中可稳定复现：保护参数在 CM-5 `Processor.Open` 的 `Spec` 网络预检阶段被改为禁用后，OPEN 仍会提交；成交后无法生成 STOP/TP。该结果与 F7 属同类资金安全后果，等级 P1，不能按“源码/离线通过”结案。

---

## 二、基线与隔离副本

### 2.1 基线

```text
branch        feat/m-coin-futures
HEAD          63efbe8 feat: cm-6
parent        bd804e2 feat: cm-5
master        0b3fa34 tag v2.2.3
worktree      173 项（后端源码/文档 13 项 + 静态 160 项）
```

非静态改动：

```text
M  README.EN.md
M  README.md
M  doc/readme/en/00-system-overview.md
M  doc/readme/zh-CN/00-系统概览.md
M  doc/trade/开发方案-CM-币本位合约交易-复刻现货精简版.md
M  routers/router.go
?? controllers/coin_m_runtime_cm7.go
?? controllers/coin_m_runtime_cm7_test.go
?? doc/readme/en/10-coin-m-futures.md
?? doc/readme/zh-CN/10-币本位合约交易.md
?? doc/trade/CODE_REVIEW_CM-7.md
?? service/coinmexecution/e2e_cm7_test.go
?? service/coinmexecution/status_cm7.go
?? service/coinmexecution/status_cm7_test.go
```

静态目录为 79 个旧哈希 JS 删除、79 个新哈希 JS 新增、`static/index.html` 修改。

前端仓库仍有 11 项工作区改动；其中 CM-7 相关为 `locales/{zh-CN,en}.yaml` 文案与 `src/views/coinM/controlledTrade.vue`、状态 API 接线；其他 Lead/登录等改动属于并行工作区，不视为本轮 CM-7 已单独审查的内容。

### 2.2 等价校验

```text
copy          /tmp/phase_audit_cm7
SYNCED_CHANGES=173
FILES_CHECKED=1167
GO_FILES=771
STATIC_FILES=120
SHA256_MATCHED=1167
SHA256_DIFF=0
SHA256_MISSING=0
DELETED_CONFIRMED=79
APP_CONF_PRESENT=True
```

首次发现 `/tmp/phase_audit_cm7` 为非空残留目录，已完全删除后重建，再执行等价校验。

---

## 三、静态审查

### 3.1 运行状态只读 API

`service/coinmexecution/status_cm7.go` 的 `RuntimeSummary` 只查询本地 `config`、`delivery_symbols`、`coin_m_managed_positions`、`coin_m_managed_orders`：

- `PrivateWSVerified`、`WalletVerified`、`TradeActionsAvailable` 固定为 `false`；
- `ReadOnly` 固定为 `true`；
- 不持有 Processor/Gateway，不调用 DAPI、私有 WS 或签名接口；
- 缺少 `config` 表或查询失败时返回 503，不把异常解释为“开关关闭”。

`controllers/coin_m_runtime_cm7.go` 在调用前执行 `c.auth()`。`auth()` 要求 JWT `Username` 为非空字符串，否则返回 `coin_m_auth_required`。`routers/router.go:126` 新增 `GET /coin-m/runtime-status`。全部 `/coin-m/*` 路由中没有下单、撤单、改仓或认领入口。

隔离探针验证匿名请求不能读到 `global_entry_enabled` 等本地状态。

### 3.2 页面与文案

`src/views/coinM/controlledTrade.vue` 只有状态刷新按钮和只读描述，复用 `CoinMAccount controlled`；`account.vue` 在 `controlled=true` 时不加载钱包。Zeitgeist 意义上的关键边界是：状态页明确显示“页面手动操作未开放”，不会从本地开关推断 WS 连通或余额充足。

`locales/zh-CN.yaml` / `en.yaml` 已删除“永远不会自动下单”的绝对表述，改为：

- 保存配置本身不立即下单；
- 已部署且双开关及门禁允许后可能真实下单；
- Demo/Testnet 未验收前不要开启；
- 关闭新开仓不等于强平或撤保护单。

四项文案与实际源码行为一致，未发现页面手动交易写接口。

### 3.3 README 与发布前清单

新增中英文 `10-币本位合约交易.md`，明确：

- 源码完成不等于实盘可用；
- 部署后手动打开双开关可能发送真实订单；
- v26→v27 迁移、Demo/Testnet、真实 Algo 父子、MySQL 8、ARM 部署、主网小额均未执行；
- 生产批准前保持 `coin_m_quant_enable=0` 且逐币 `enable=0`。

未发现 README 把离线 Fake 测试表述为实盘验收。

### 3.4 CM-6 F7 修复核对

当前 HEAD `runner_cm6.go:313-319` 已包含：

```go
if cfg.Strategy != coin.Strategy || cfg.Technology != coin.Technology ||
    cfg.Profit != coin.Profit || cfg.Loss != coin.Loss ||
    cfg.TickSize != coin.TickSize || cfg.Leverage != coin.Leverage {
    return nil
}
if err := coinMAutoProtectionReady(cfg); err != nil {
    return err
}
```

`runner_protection_race_cm6_test.go` 覆盖 9 种配置变更与禁用态下一 tick fail-closed。该修复对“策略求值期间编辑”的根因有效，本轮未推翻。

---

## 四、新增发现 F8（P1）：签名预检期保护配置竞态

### 4.1 定位

`runner_cm6.go:297-320` 的最终比较发生在 `r.processor.Open(...)` 之前。进入 `Processor.Open`（`processor.go`）后，配置只在第 117 行 `configForEntry` 读取一次，随后依次执行：

```text
121  API.Spec
136  preflight / requireReady
139  API.TradeSettings
142  API.HasOpenOrders
149  API.OpenAlgos
156  API.Position
166  API.Mark
174  API.Available
209  API.Query
237  本地落库
246+ DAPI POST
```

这些步骤包含多次网络往返。若在 `configForEntry` 之后、POST 之前编辑 `Profit`/`Loss`/`TickSize`/`Leverage`，Runner 已经完成最终快照比对，Processor 也不会再次执行 CM-6 的保护单可执行性校验。

### 4.2 隔离探针实证

探针：`TestAuditCM7ProtectionEditedDuringSignedPreflightStillOpens`

方法：用包装 Gateway 覆写 `Spec`，在第一次签名/网络预检时执行：

```sql
UPDATE delivery_symbols SET profit='0', loss='0', tickSize='0' WHERE symbol='BTCUSD_PERP'
```

结果：

```text
CM-7 Runner 最终配置比对通过
Processor.configForEntry 读取有效配置
Spec 预检期间配置被改为禁用保护
OPEN 仍发出 1 次 POST
Fill 写入成功
下一 tick 无法生成 STOP/TP，openAlgos=0
```

`-count=5` 连续 5 次均可复现。探针 PASS 表示不安全行为被复现，不代表业务正确。

### 4.3 与 CM-6 F7 的关系

| 维度 | F7（已修复） | F8（本轮新增） |
| --- | --- | --- |
| 编辑时机 | `strategy.Entry` 信号求值期间 | `Processor.Open` 的签名/网络预检期间 |
| 原拦截点 | Runner 最终快照比对 + 刷新后 `coinMAutoProtectionReady` | 同一个比对已执行完，之后无保护配置复检 |
| 后果 | 无保护自动开仓 | 无保护自动开仓 |
| 等级 | P1 | P1 |

CM-6 二轮复审中已明确记录“最终读库以后到交易所 POST 之前的极窄并发窗口尚待 CM-7/真实环境进一步核查”。本轮核查结论是：窗口仍存在，且由于 Processor 在 POST 前执行多次网络调用，并非仅微秒级竞态。

### 4.4 最小修复与再审条件

1. 将 STOP/TP 可执行性判断下沉到 `Processor.Open` 的最终 POST 前，或由 Runner 将经过校验的配置快照显式传入并在发单前复验；
2. 在 `Processor.Open` 越过签名预检后再次读取逐币保护字段，任何变化都标记旧信号失效、停止本次 POST 并记录本地失败；
3. 增加永久回归测试：在 `Spec`/`TradeSettings`/`OpenAlgos` 等回调中修改保护配置，断言 POST=0、OPEN=0、Managed 周期=0；
4. 修复后重跑 CM-7 全量 Gate 与独立复审，才可重新讨论“CM 阶段完成”。

---

## 五、离线 Gate

执行时清空 Testnet、实盘、Demo、审批与数据库 DSN 等 env；所有 HTTP Mock 均在隔离副本内完成。

### 5.1 后端

```text
git diff --check（工作区）                                  PASS
go vet ./...                                               PASS
go build -o /tmp/go_binance_futures_cm7_audit .            PASS
go test -count=1 -p=2（88 包显式列表）                     PASS
    OK=75 NO_TEST=13 FAIL=0 TOTAL=88
go test -count=5 -p=2 -run '^TestCM7' ./service/coinmexecution \
    ./controllers                                          PASS
go test -count=5 -p=2 ./service/coinmexecution \
    ./service/coinmquant ./feature/api/binance ./controllers PASS
go test -race -count=2 -p=2 ./service/coinmexecution \
    ./service/coinmquant ./feature/api/binance ./controllers PASS
```

macOS linker `LC_DYSYMTAB` warning 非致命，退出码为 0。

### 5.2 隔离探针

```text
TestAuditCM7ProtectionEditedDuringSignedPreflightStillOpens  PASS（复现 F8）
TestAuditCM7RuntimeSummaryNeverClaimsRemoteOrWrites           PASS
TestAuditCM7RuntimeStatusDeniesAnonymousRead                  PASS
```

合计 3/3 PASS；F8 探针 `-count=5` PASS。

### 5.3 前端与产物

```text
pnpm typecheck                       PASS
frontend/dist vs backend/static      DIFF_EXIT=0
files per tree                       120 / 120
coinM live-action APIs               0
```

本轮未重新执行 `pnpm build`：CM-7 没有新增未同步的前端源码，审计前已核对产物与后端 `static/` 完全一致；typecheck 通过，避免重复构建改写 79 个哈希文件名。实施方自评中的 build PASS 作为历史证据保留，本复审不把它升级为实盘 Gate。

---

## 六、未通过 / 仍 BLOCKED 的 Gate

1. **F8 修复 Gate**：签名预检期保护配置竞态未闭环，禁止宣告 CM 阶段完成。
2. 真实 MySQL 8 v26→v27 迁移、备份、重复升级、唯一索引与 DECIMAL 精度未验证。
3. COIN-M Demo/Testnet 凭证、REST/WS host、listenKey、保活、断线恢复未现场执行。
4. 真实 LONG/SHORT 普通单、部分 Fill、STOP/TP Algo 父子触发、撤单与平仓未执行。
5. 币本位钱包、张数过滤器、杠杆、逐/全仓及 UM 共用持仓模式未真实核对。
6. ARM Linux 部署、后台进程重启、持久化授权、主网小额交易未执行。
7. 页面手动受控写操作未开放，不能表述为“完整手动交易能力”。
8. 其他前端并行未提交改动未单独通过 Code Review。

---

## 七、审评意见

CM-7 在可读状态、文档准确性、页面边界和离线 E2E 方面完成了可验证交付；只读状态 API 明确不冒充远程账户健康，页面也未提供手动写操作。后端 88 包、专项 5 次、race、build、vet、typecheck 与产物一致性均通过。

但 CM-6 二轮复审遗留的 TOCTOU 窗口在本轮被探针稳定复现，后果与 F7 相同：自动开仓后没有可执行保护单。依据资金安全边界，不能因窗口发生在网络预检期就降级为普通观察项。

**最终判定：CM-7 ❌ 不通过；禁止宣告 CM 开发完成或进入真实部署；修复 F8、补充永久回归测试并重审前，不得打开真实交易开关。**


---

## 八、F8 后续整改与第二轮复审（2026-11-18）

本报告中的 **F8/P1 失败复现与“不通过”结论是修复前的原始审计结果，保留作为历史证据**。实施方随后在 `Processor.Open` 的 DAPI 预检完成后增加末端配置与 STOP/TP 校验，并在 POST 前再次检查。新增 4 个预检期注入位置 × 15 种配置变化，共 60 组永久回归，验证 POST=0、OPEN=0、Managed 周期=0；另测试了已经持久化但确认未发单的本地意图安全结案。

**修复后的最新结论：[CODE_REVIEW_CM-7-rereview-2.md](CODE_REVIEW_CM-7-rereview-2.md)**。88 个非历史研究 Go 包、四核心包各五次测试、Race Detector、Vet 和 Build 均已通过。此结论只针对源码及隔离离线测试；真实 MySQL 8、Demo/Testnet、Algo/WS 和实盘部署依旧未验收。
