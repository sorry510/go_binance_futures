# Code Review：当前工作区改动（实盘交易周期与开仓冷却）

> 审查对象：当前工作区 9 个改动文件（基线 `HEAD dc295e2 feat: bt-3`）
> 关联文档：`doc/trade/回测自定义精度差异与性能修复方案.md`（未跟踪，状态「待开发」，其范围与本次实盘改动不一致 —— 见 F1）
> 审查方式：隔离副本 `/tmp/phase_audit_tradecycle`（803 Go，SHA256 全等）+ 全套离线 Gate + 独立行为探针
> 未改用户业务代码、未提交、未跑 `sync db`、未连真实 Binance、未写生产库或碰仓位。

---

## 一、结论

**当前改动：❌ 不通过 —— 存在 2 项 P1 治理阻断，禁止部署 / 实盘验证；代码层面的离线 Gate 与单测全部通过。**

| 级别 | 结论 |
|---|---|
| 编译 / 单测 / race / vet | ✅ 全绿，代码质量无缺陷 |
| 架构边界（退出不被阻塞、账户隔离） | ✅ 实现与测试正确 |
| **方案一致性 / 变更授权** | ❌ **P1：文档与实际改动范围根本性不一致** |
| **实盘策略行为** | ❌ **P1：重开间隔与调度频率无授权变更** |

**允许**：修正治理问题（补齐对应方案 / 授权）后重新复审。
**禁止**：以「单测通过」为依据部署、开启实盘或声称性能改善已验证。

---

## 二、改动范围实证

| 文件 | 实际改动 |
|---|---|
| `feature/trade_open_cooldown.go`（A） | 包级全局 `mainTradeOpenCooldown` / `leadTradeOpenCooldown`，硬编码 `3 * time.Minute`，`blockFrom`/`blocksAt` 均带 nil 保护 + Mutex |
| `feature/account_trade_cycle.go` | runner 增 `OpenCooldown` 字段；Main 构造器注入；新增 `submitCloseWithEntryCooldown`（成功平仓后 blockFrom）；注释改「1 second tick」 |
| `feature/feature.go` | 6 处 `SubmitClose` → `submitCloseWithEntryCooldown`；开仓段 `if OpenCooldown.blocksAt now return`；开仓成功的 `Sleep(30s)` → `OpenCooldown.blockFrom` |
| `feature/lead_trade_runtime.go` | Lead runner 注入独立 `leadTradeOpenCooldown` |
| `main.go` | `StartTrade` 调度 2s → 1s；`NoticeAllSymbolByStrategy` 与 `CheckTestResults` 1.5s → 1s |
| `feature/feature_test_strategy.go` | `FuturesTestNotice` int → `atomic.Int32`（CAS）；模拟平仓处理 `return` → `break` |
| 3 个测试文件 | cooldown 边界 / 账户隔离 / 退出不受阻 / 平仓失败不延长；cycle 轮询与模拟 break 回归；golden 测试适配 |

---

## 三、Findings

### F1 / P1 —— 方案文档与实际改动范围根本性不一致（治理阻断）

- **事实**：工作区唯一新增方案文档为 `doc/trade/回测自定义精度差异与性能修复方案.md`，主题为「解释 1m / 自定义精度交易差异 + 减少重复读取/解压/下载」，其第 1 节明确写「**不改实盘交易逻辑**」「不修改生产数据库」，第 4 节 P0～P2 全部是回测侧 Trades 内存复用、归档计数、前端阶段显示（且状态仍为「待开发 / 待 Code Review」，P0/P1/P2 均未见对应代码落地）。
- **实际改动却全部落在实盘交易路径**：`feature/` 下的开仓冷却、平仓触发、调度周期，`main.go` 的主循环频率。没有任何一处改动 `service/backtest` 或 `service/historicalmarket`。
- **风险**：以回测性能方案为凭据，夹带实盘交易行为变更；审查无法用方案验收标准衡量代码，变更也无法追溯授权。这是典型的「文档-代码范围漂移」。
- **修复**：①若这些实盘改动是真实意图，须另立独立方案文档并说明目标（降低轮询延迟？避免平仓后立即重开？）、风险、回滚方式，并在其中撤销「不改实盘」承诺或明确这是独立议题；②若不是，须从当前工作区剥离实盘改动，回到回测 P0 开发。**在补齐前禁止部署。**

### F2 / P1 —— 实盘策略行为的无授权变更（重开间隔 30s→3min、平仓后抑制新开仓）

- **原行为**：仅开仓成功后 `runner.Sleep(30 * time.Second)`，阻塞当前 runner（副作用：这段时间连退出扫描也被阻塞）；平仓后不抑制重开。
- **新行为**：硬编码 `tradeOpenCooldownDuration = 3 * time.Minute`；①开仓成功后非阻塞 block 3 分钟（**间隔延长 6 倍**）；②**任何成功平仓提交后也 block 3 分钟**（新增语义）。
- **实证**：独立探针 `TestProbeSuccessfulCloseSuppressesReentryForThreeMinutes` 验证：成功平仓后同周期立即再跑，候选选择被跳过；强制过期后恢复。开发方 `TestTradeOpenCooldownStillClosesAllPositionsAndOnlyBlocksNewEntries` 亦验证该行为。
- **风险**：对高频 / 短线策略，平仓后 3 分钟不可重开会显著改变交易频率与收益特征；该数值无配置项、无方案依据、无回测对比。原 30s 是硬编码，但本次把时长放大并扩展到平仓路径，属于策略语义变更而非「性能优化」。
- **修复**：①在方案中明确该行为变更并给出策略影响评估（至少用 1m / adaptive 回测对比重开间隔 30s vs 3min）；②将时长改为配置项或至少引用既有配置；③确认「平仓后也冷却」是否为真实需求（注释称避免 close→reopen churn，但需证据支持）。**在补齐前禁止开启实盘新开仓。**

### F3 / P2 —— 主循环 2s→1s 翻倍请求频率，无限流补偿或实测依据

- **事实**：`main.go` `StartTrade` 2s→1s，另两个策略测试循环 1.5s→1s。`loopRun` 用 CAS 单飞，不会并发叠加（已核实 `main.go:581-594`），故不会因慢轮次产生并发风暴；但快轮次下 REST 请求频率理论上翻倍。
- **风险**：原注释明确「REST 权重随 managed orders/WS 状态变化，不使用固定预算假设」；本改动未附任何权重测算、429 观测或 WS 命中比例证据。若私有 REST 请求（ReadPositions / ReadOpenOrders）主要靠 WS 状态缓存，翻倍的是本地循环频率而非外部请求，风险较低；但这需要实证，当前无。
- **修复**：在方案中补充 1s 轮询的 REST/WS 调用链分析与 429 观测计划；生产发布 Gate 加入权重监控与回退到 2s 的开关。

### F4 / P3 —— 文档状态与代码进度脱节

- **事实**：唯一新增方案文档「回测自定义精度差异与性能修复方案.md」状态仍为「待开发 / 待 Code Review」，其 P0/P1/P2 计划（分钟级 Trades 内存复用、归档计数、前端阶段显示）均无对应代码落地；而实际改动是文档范围之外的实盘交易行为变更。
- **风险**：文档状态与代码进度脱节，后续追溯与验收标准无法对齐。
- **修复**：明确实盘改动应归属的独立文档与其状态；回测方案则保持「待开发」直至 P0 真正落地。

---

## 四、离线 Gate（全部通过，但不构成放行依据）

| 检查 | 结果 |
|---|---|
| gofmt（9 文件） | ✅ 通过 |
| `go vet ./feature/...` | ✅ 通过 |
| `go build ./...`（隔离副本） | ✅ 退出码 0 |
| `go test -count=1` 全量 88 包 | ✅ 76 ok + 12 no test + 0 FAIL |
| `go test ./feature/... -count=1` | ✅ 4 包全 ok |
| `go test ./feature -count=5 -p=2` | ✅ 通过（7.884s） |
| `go test -race ./feature -count=2` | ✅ 通过（LC_DYSYMTAB warning 非致命） |

> 说明：用户工作区 `go build ./...` 仍因未跟踪的 `strategy_templates/research/**` 历史欠账失败，与本次改动无关（BT-3 审查已核实）。

---

## 五、架构边界验证（通过部分）

- **退出不受阻**：`blocksAt` 仅在 `!AllowNewOpens` 检查之后、`SelectCoins` 之前的开仓段调用（feature.go:363-371），所有 exit 分支在其之前执行；6 处平仓提交均经 `submitCloseWithEntryCooldown`，仅在 `err == nil` 时 block，失败不延长（测试验证）。
- **账户隔离**：Main / Lead 各自独立全局 cooldown 实例，测试 `TestTradeOpenCooldownSeparateMainAndLeadAccounts` 验证互不阻塞；生产两个构造器均注入正确实例。
- **nil 安全**：`blockFrom` / `blocksAt` 对 nil receiver 返回 no-op / false，测试构造 runner 未注入时等价于无冷却，不会 panic。
- **atomic 修复**：`FuturesTestNotice` 由裸 int 读写改为 `atomic.Int32` CAS，消除了调度频率提高后双循环并发访问的潜在数据竞争；race 全绿。
- **模拟平仓 break 修复**：`CheckTestResults` 中 `return` → `break` 使同一轮能处理多个模拟仓位的平仓，AST 回归测试锁定该结构，避免再次退化为单仓位。

以上均是正确的工程实现，问题集中在治理与策略语义，不在代码正确性。

---

## 六、放行 Gate

1. **先解 F1**：补齐与代码范围一致的方案文档 / 授权，明确这是实盘交易行为变更，或剥离实盘改动回到回测方案。
2. **再解 F2**：给出 3 分钟冷却的策略影响证据（回测对比）与配置化方案，确认平仓后冷却的业务依据。
3. **补 F3**：1s 轮询的 REST/WS 权重分析与 429 监控、2s 回退开关。
4. **生产 Gate**：真实 Binance Testnet 小额验证、MySQL 8 账本、ARM Linux + static 成套发布与回滚、持久化授权 / 审计 / 撤销，均未执行也不能由离线 Gate 推断。

**当前状态：源码与离线测试通过，但因 F1/F2 治理与语义阻断，判定不通过；修订后须另起 `*-rereview.md` 复审。**

> 审查探针仅写在隔离副本，报告落盘后清理；用户工作区保持交付时状态。
