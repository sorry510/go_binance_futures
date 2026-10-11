# 币安合约自动带单 — Stage 3 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage实施方案.md` 的 **Stage 3 — 提取并复用现有 StartTrade 交易流程（不复制策略）**（工作区未提交改动）。基线 `HEAD = 193a020`（`feat: stage2`，分支 `feat/lead-trading`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库与 `strategy_templates/` 未被改动）
- **审查类型**：review-only
- **结论**：**Stage 3 离线实现完成，且共享循环的重构是忠实的机械式抽取**——我逐段核对了 `feature.go` 的 407 行 diff，**未发现 main 交易语义漂移**；Gate 3 四项中"无重复策略实现""交易语义与旧实现一致""lead 仅 mock"三项通过，"原主账户全量回归"仍需用户真实运行确认（记录已声明）。未发现 P0/P1；发现 1 项 P2（用户侧回归）+ 4 项 P3。
- **审计日期**：2026-10-09

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| Gate 3「无重复交易策略实现」 | ✅ **实证**：`AutoStopOrder`/`CanOrderComplete` 全仓仅一份（`line_custom.go`），`GetCanLongOrShort` 与 `GetCanLongOrShortWithPositions` 共用同一 `evaluateEntryWithPositionLoader` 主体 ✓ |
| Gate 3「原主账户全量回归」 | ⏳ **用户侧待执行**（记录 §3/§4 已声明；本审计仅完成代码级等价性核对与自动化测试） |
| Gate 3「所有交易语义与旧实现一致」 | ✅ **代码 diff 核对 + golden 实证**：顺序/日志/算术/通知/30s 冷却逐段一致；退出优先级与 `canClose` 调用次数被 golden 锁定；MARKET/方向开关/LIMIT 均实证 |
| Gate 3「lead 仍只在 mock Broker 下运行」 | ✅ **实证**：`leadMockTradeRunner` 无生产调用方；`validate()` 强制 lead 必须 `Mock` + 具备白名单缓存；生产入口 `StartTrade` 只构造 main 运行器 |
| 是否新增 DB/配置 | ❌ 无新 Schema（仍 19）、无模型/配置中心/`app.conf` 改动 |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 计划条目逐项核对（Stage 3 §128-165）

| # | 计划要求 | 判定 | 证据 |
| --- | --- | --- | --- |
| 3.1.1 | `LoadSharedTradeConfig`：每周期读一次 Config/Symbols，保留既有选币参数 | ✅ | `runAccountTradeCycle` 从 `runner.Config` 取配置（`account_trade_cycle.go:35-42`）；`LoadSymbols` 每周期一次 ✓；`StartTrade` 仍是唯一调度入口且保留 `FutureEnable`/`flagFutures` 日志语义 ✓ |
| 3.1.2 | `SelectTradeCoins` 仍走 `selectConfiguredCoins(..., ModeTrade)`；同一快照支持两账户复用 | ✅ | main 运行器 `SelectCoins` = `selectConfiguredCoins(..., SmartLocalV2ModeTrade)` ✓；lead 走**独立** `selectLeadTradeCoins`（白名单优先的全市场池 → 再评分，非 Main Top60 交集 ✓，永久测试 `coin_selection_stage3_test.go` 用 61 币实证 X60 只对 Lead 可选 ✓） |
| 3.1.3 | `EvaluateEntry` 唯一实现 `GetCanLongOrShort`，沿用同一规则 JSON/hash/指标 | ✅ | `line_custom.go`：新 `GetCanLongOrShortWithPositions` 注入账户持仓，二者共用 `evaluateEntryWithPositionLoader` ✓（无第二份 DSL/hash 实现 ✓） |
| 3.1.4 | `EvaluateExit` 唯一实现 `AutoStopOrder` → 止损 + `CanOrderComplete` → 止盈，保持原优先级与收益率定义 | ✅ | `evaluateSharedTradeExit` / `evaluateTradeExitWithRules`（`account_trade_cycle.go:151-181`）优先级 auto-stop → loss → profit ✓；旧代码 `if nowProfit <= -loss` / `>= profit` 与内层 `CanOrderComplete` 语义等价（见 §3 diff 核对）✓；永久测试断言 `canClose` 调用次数（auto 时 0 次 ✓） |
| 3.1.5 | `AccountTradeLoop(AccountContext)`：每账户独立顺序（超时撤单 → 快照/对账 → 退出检查 → 开仓闸门 → 逐币信号 → 记录通知） | ✅ | `runAccountTradeCycle` 顺序与旧 `StartTrade` 逐段一致 ✓；`AllowNewOpens` 门禁位于**退出循环之后**（`feature.go:352-355`）✓ 与记录一致 |
| 3.1.6 | `TradeExecutionAdapter`：每账户独立 depth/精度/杠杆逐仓/下单/pending slot/ClientOrderID/超时撤单/成交同步 | ✅（以函数注入形式落地） | runner 的 `Depth/EnsureConfig/SubmitOpen/SubmitClose/CancelExpired/SyncPositions` 钩子 ✓；main 绑定旧实现（`GetDepthAvgPriceContext`/`UpdateSymbolTradeInfoContext`/`submitAutoStrategyOpen`/`submitManagedStrategyClose`/`cancelTimeoutOrder`/`syncStrategyExitPositions`）✓；真实 lead 适配器按计划留到 Stage 4 |
| 3.1.7 | `TradeHistoryWriter` + `NotifyAdapter` 带 account_id 记录 | ✅ | `RecordOpen`/`RecordClose` = main 的 `insertOpenOrder`/`insertCloseOrder`（Stage 2 已显式写 `account_id=main`）✓；`NotifyOpen/NotifyClose` 保持 `pusher.SetModuleName("futures")` 原样 ✓ |
| 3.2 | 10 行规则对应表（多空开关、MaxCount、LossMaxCount、超时撤单、订单类型、币配置、ROI、hash、历史、冷却） | ✅ | 逐项在 diff 中确认：方向开关独立门禁 ✓（探针实证）、`cycleAccount.AccountSlotCount()` 每账户 MaxCount ✓、`openBlockedSymbols` 本账户 ✓、`FutureBuyTimeout` 撤本账户超时单 ✓、MARKET/LIMIT 沿用同一配置不替换 ✓、`Usdt/Leverage/Profit/Loss` 读同一份币配置 ✓、hash 由 `openResult.Long/ShortStrategyHash` 传递 ✓、冷却改为**每运行器各自 sleep 30s** ✓（`feature.go:553-556`） |
| 3.3 | 一致性测试（old vs new、双账户同信号各自决策、暂停组合不误触发、backtest 不变） | 🔶 部分 | golden 锁定重构后语义（MARKET 长空各一笔 + 历史价 100.1/99.9 + 30s 冷却）✓；退出优先级纯函数表测试 ✓；暂停组合（`AllowNewOpens=false`）实测"只对账/撤单不开仓" ✓；**未做 old↔new 逐字节差分**（旧实现已被抽取，无法并行运行——由本次 diff 核对替代）⚠️；backtest 未运行（记录声明不改 backtest ✓，本阶段未触碰该包 ✓） |
| Lead 白名单缓存（本阶段新增，计划未单列） | ✅ | `lead_symbol_cache.go`：1 小时刷新 / 最长 2 小时有效 ✓、generation fencing（`Invalidate()` ++generation 使在途旧请求失效 ✓）、401/403 → `ErrLeadSAPIUnauthorized` 立即禁止新开仓 ✓、USDT 规范化（`HasSuffix("USDT") && quoteAsset ∈ {"" , "USDT"}`，`lead_symbol_cache.go:151-152`）✓、`Run(ctx)` 退出时 `defer Invalidate()` ✓；**无生产调度接线** ✓ |

---

## 3. main 语义等价性核对（逐段 diff 结论）

对 `feature/feature.go`（+407/−213，忽略空白）逐段比对旧 `StartTrade` 与新 `runAccountTradeCycle`：

1. **调用顺序完全一致**：找币 → 读持仓 → 读挂单 → `newTradeCycleAccountSnapshot` → `syncStrategyExitPositions` → 超时撤单 → 退出循环 → 开仓闸门 → 逐币 → 收尾 30s 冷却 ✓。
2. **退出分支等价**：旧代码是三个独立 `if`（auto-stop / `nowProfit <= -loss && CanOrderComplete` / `nowProfit >= profit && CanOrderComplete`），新代码是 `switch` 三个 case ✓；旧代码在阈值满足但 `CanOrderComplete` 未通过时会**落到下一个 if**，新 `evaluateTradeExitWithRules` 同样落到下一分支 ✓ **语义等价**。
3. **算术与精度未变**：市价多单历史价 `buyPrice*1.0012`、空单 `sellPrice*0.9988`，再经 `GetTradePrecision(..., TickSize)`；限价单直接用精度化后的 depth 价 ✓（探针实证 LIMIT 历史价 = 100 而非 100.1 ✓）。
4. **通知/日志/历史写入原样保留** ✓（`lang.Lang(...)` 标题、`pusher.SetModuleName("futures")` 模块名未变 ✓）。
5. **两处新增控制流对 main 不可达**（属 lead 预留的 fail-closed 路径）：`SelectCoins` 的 error 分支（main 的 `selectConfiguredCoins` 不会返回错误）与 `EnsureConfig` 的 error→`continue`（main 的钩子显式忽略 RPC 错误并返回 nil，代码注释已说明"legacy path deliberately ignores the Binance config RPC error; Stage 4 Lead must fail closed instead" ✓）。
6. **三处 lead 专属分支**：`AllowNewOpens` 门禁、开仓循环内的白名单 `Allows` 复检、`AccountID==lead` 才生效 ✓ → main 路径不变 ✓。
7. **冷却语义**：`time.Sleep(30s)` → `runner.Sleep(30s)`，注释改为"对当前账户单独冷却" ✓（符合计划 §3.2「每账户独立冷却」✓）。

---

## 4. 实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-Stage3-1** main LIMIT golden | 全 Fake main 运行器（depth=100、tickSize=0.1、stepSize=0.01、Usdt=100、leverage=4）跑一轮 | ✅ 意图 = `BUY/LONG/LIMIT/price=100/qty=4` + `SELL/SHORT/LIMIT/price=100/qty=4`；历史记录 = `LONG:100` / `SHORT:100`（**限价单记限价、不套用市价 1.0012/0.9988 调整** ✓） |
| **PR-Stage3-2** 方向开关 | `FutureAllowLong=0` 与 `FutureAllowShort=0` 两种配置 | ✅ 分别只开 SHORT / 只开 LONG（方向门禁独立 ✓） |
| **PR-Stage3-3** lead 选币在白名单不可用时 | 未刷新过的 `LeadSymbolCache` + `FutureStrategyCoin=smart_local_v2` | ✅ 返回**空列表且 error=nil**（新开仓 fail-closed，但**不会中断周期**，已有持仓仍会走到退出循环 ✓）；刷新后 `Allows` 正确放行/拒绝 ✓ |

> 说明：上述三条**均无永久回归测试**（现有永久测试覆盖 MARKET golden、退出优先级、方向外的其余规则、白名单首选项、冷却账户隔离、缓存生命周期）。建议把 LIMIT 路径、方向开关、白名单不可用三条固化进 `feature` 包测试。

---

## 5. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race ./feature ./scanner ./feature/api/binance ./service/futuresownership ./service/binanceapiusage -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` | ✅ 全部 ok（feature 1.65s、scanner 1.33s、api/binance 2.24s、ownership 1.92s、apiusage 2.79s） |
| `go vet ./feature ./scanner ./feature/api/binance` | ✅ 干净 |
| 根包 `go build -o /tmp/stage3_bin .` | ✅ 成功（未在仓库内留产物） |
| 探针 PR-Stage3-1/2/3 | ✅ 全部通过；副本去掉探针后 `./feature`、`./scanner` 原测试仍 ok |
| 全量 `go test ./...` | ⚠️ 仍红：未跟踪研究目录导致（与 Stage 1/2 报告的 F3 同一问题）+ 批量 setup 抖动；**与本阶段无关** |
| 真实 Binance / 真实 Lead API / 真实 DB | ❌ 全部未执行（记录 §3 已声明） |

---

## 6. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | **P2（用户侧回归）** | 共享循环重构虽经逐段 diff 与 golden 验证，但**尚未在真实 Main 业务上运行过**；这是本阶段唯一实质性外部依赖 | 建议回归清单：① 一轮真实开仓（MARKET 与 LIMIT 各一次）观察下单参数/历史价/通知；② 策略平仓与止盈/止损三种退出各触发一次；③ 确认 30s 冷却与 2s 调度节奏未变；④ 确认 `order` 历史与统计仍只记 main；⑤ 确认 lead 未启动（无第二条 WS、无 SAPI 定时请求） |
| F2 | P3（可观测性） | `runAccountTradeCycle` 的观测 source 固定为 `"start_trade"`（`account_trade_cycle.go:41`）→ 未来 lead 运行时其签名请求会计入 main 的来源标签 | 建议 Stage 5（接入 lead 时）按 `runner.AccountID` 设置 source（如 `lead_trading`），避免 per-account 归因失真 |
| F3 | P3（Stage 4 强化项） | `leadMockTradeRunner(cfg, cache, fake)` 直接复用调用方传入的 IO 钩子（`r := *fake`）→ "Mock=true" 只保证被 `validate()` 检查，**并不保证钩子本身是 fake**；若将来误把 main 运行器当 `fake` 传入，会得到"lead 身份 + main 真实 IO"的运行器 | 建议 Stage 4 改为由工厂**内部构造** fake/真实适配器（或引入显式的 mock 类型标记并在 `validate()` 中断言），彻底消除误用路径 |
| F4 | P3（测试缺口） | LIMIT 路径、`FutureAllowLong/Short=0` 方向开关、白名单不可用→空选择 三处无永久测试（本轮探针已实证） | 建议把 PR-Stage3-1/2/3 固化进 `feature/account_trade_cycle_stage3_test.go` |
| F5 | P3（文档） | 两处新增控制流（`SelectCoins` 错误、`EnsureConfig` 错误→`continue`）对 main 不可达，属 lead 预留 fail-closed 路径 | 建议在方案文档 §3.1 或记录中注明"这两条分支仅在 lead 适配器出错时生效"，便于后续回归时理解 |
| — | 正面 | ① 白名单**先于** Top60 评分（避免 Main Top60 之外的可带单币被遗漏）且不复制评分器 ✓；② 冷却按 `account_id` 隔离（scanner 层 SQL 过滤 + 未知账户报错 ✓）；③ 退出/对账不受"暂停新开仓/白名单失效"影响 ✓；④ `AutoStopOrder/CanOrderComplete` 仍唯一 ✓ |

---

## 7. 审计边界与未验证项

- **未验证**：真实 Main 业务回归、真实 Lead Portfolio Key 能力（Gate 0-LIVE）、真实资金/下单（Stage 7 授权前均不执行）。
- **未验证**：old↔new 逐字节差分（旧实现已被抽取，无法并行运行；由本次 diff 逐段核对 + golden 测试替代）。
- **未验证**：`service/backtest` 结果未跑（本阶段未触碰该包 ✓；记录声明不改 backtest ✓）。
- **未验证**：MySQL（本阶段无 Schema 变更）；前端（无改动）。
- 本审计未发起任何真实 Binance 请求、未改动用户仓库任何文件（探针仅存在于已删除的 `/tmp` 副本）。

---

## 8. 附：Stage 3 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `feature/account_trade_cycle.go`（新） | 181 | `accountTradeRunner`（账户 IO 钩子）、`mainTradeRunner`、`leadMockTradeRunner`、`validate`、唯一生产入口 `StartTrade`、共享退出判定 |
| `feature/feature.go`（改） | +407/−213 | 旧 `StartTrade` 主体抽取为 `runAccountTradeCycle(runner)`，业务逻辑原样保留、IO 改注入 |
| `feature/account_trade_cycle_stage3_test.go`（新） | 183 | Lead mock 白名单/暂停/自有持仓/不完整运行器拒绝 + **Main MARKET golden** + 白名单失效仍平仓 |
| `feature/coin_selection_stage3_test.go`（新） | 69 | 白名单先于 Top60（61 币实证）+ 退出优先级表测试（含调用次数） |
| `feature/coin_selection.go`（改） | +41 | `selectLeadTradeCoins`、`leadEligibleUniverse`（全市场 + 白名单优先，不触发 SAPI ✓） |
| `scanner/local_selector_v2.go`（改） | ±16 | `SmartLocalV2ModeLead` + `recentClosedSymbolsForAccountWithOrm(account_id)`（未知账户报错 ✓） |
| `feature/strategy/line/line_custom.go`（改） | ±13 | `GetCanLongOrShortWithPositions` + 共用主体（策略唯一 ✓） |
| `feature/api/binance/lead_symbol_cache.go`（新） | 222 | Lead 白名单 1h 刷新/2h 上限、generation fencing、退避、401/403 fail-closed、`Run(ctx)` 生命周期 |
| `feature/api/binance/lead_readonly.go`（改） | ±9 | 新增 `ErrLeadSAPIUnauthorized`（401/403 与业务失败统一哨兵） |
| `feature/api/binance/lead_symbol_cache_test.go`（新） | 160 | 冷启动 fail-closed、热路径零请求、每小时刷新、2h 过期、并发去重、旧请求晚返回失效、401/403 与空名单、恢复 |
| 文档 | — | `币安合约自动带单-Stage3-实施记录.md` + 方案文档 §实施记录更新 |

**建议下一步**：① 你侧执行 Main 真实运行回归（F1 清单）；② 把 F4 三条探针固化为永久测试；③ Stage 4 落实 F3 的"由工厂构造适配器/mock"加固，并实现真实 Lead 执行适配器（仍不得开真单）；④ 文档补 F2/F5 两条说明。
