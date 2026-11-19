# UF · U 本位合约交易六项精简优化开发方案

> 日期：2026-11-18。审查基线：后端 `master`。当前进度：**UF-1、UF-2 均已完成后端源码与隔离测试**；UF-3～UF-6 尚未实施。按用户要求已撤回 UF-1 的额外前端提示及当次 `static` 构建产物；UF-2 不修改前端、生产数据库、运行服务或生产配置。
>
> 范围只涉及 **Main U 本位合约**；保留 Lead 的账户隔离、币本位 COIN-M、现货及既有订单归属。个人自用、最少改动、分阶段实现并复审。编号统一为 **UF-1～UF-6**。

## 一、实施顺序

| 阶段 | 对应需求 | 优先级 | 实施先决条件 |
| --- | --- | --- | --- |
| **UF-1（后端/离线已完成）** | 总开关关闭后，已有 Main auto_strategy Managed 仓位仍可平仓 | P1 | [Code Review](CODE_REVIEW_UF-1.md)：88 包、五次重复、Race、Vet、Build 通过；后续依要求撤回额外前端提示及对应 `static` 构建，实盘未验收 |
| **UF-2（后端/离线已完成）** | 杠杆与保证金模式设置失败必须阻止新开仓 | P1 | [Code Review](CODE_REVIEW_UF-2.md)：错误不再丢失，交易所响应、配置变更二次检查、5 秒成功缓存；88 包、5 次回归、Race、Vet、Build 全部通过；真实交易未验收 |
| **UF-3** | 异常订单一键定位 | P2 | 使用现有 Managed/Fill/系统健康事实 |
| **UF-4** | 为什么没有开仓 | P2 | UF-1/2 的阻断原因与 UF-3 定位入口已稳定 |
| **UF-5** | 实盘策略效果统计 | P2 | 按 `settled` + REST 核实 Fill 计算，不依赖历史估值 |
| **UF-6** | 订单历史性能优化 | P2 | 先记录 UF-5 前后的真实基准，精确比对计算结果 |

严格按此顺序提交，每个 UF 单独做 Code Review，通过之后进入下一阶段。允许独立写测试，但不能直接把六项耦合成一次改造。

## 二、统一技术边界

1. **不新建顶级菜单**；诊断放在「合约交易」页面，异常定位放在现有「合约订单 / 合约仓位 / 系统看板」，策略效果统计作为「合约订单」的二级 Tab 或卡片。
2. 默认**不新建数据库表、不增加全局交易开关、不引入队列/新调度器/复杂审计**；必要时先有实测数据和不可替代的证据再讨论索引迁移。
3. 行情、策略、账户和网络错误不得伪装成“策略没命中”。Order ACK 不是 Fill；本地订单/人工标注不等于已核实收益。
4. 只读统计不得驱动交易操作；前端状态与操作必须严格区分。**历史人工状态/完整性覆盖只影响展示，不可影响统计纳入资格或风控**。
5. 仅保留“Main”的原有单账户热路径和既有 2 秒调度；不改 `app.conf` 连接、不写生产数据库、不修改真实账户，不私自执行 `sync db`。
6. 不覆盖任何同时进行的 Lead、Spot、COIN-M 工作区文件；最终前端 `pnpm build` 生成 `dist` 再同步到后端 `static`，提交时按功能隔离审查。

## 三、UF-1：总开关仅阻止新仓，已有 Managed 持仓继续退出

**目前的已确认行为**：`feature/account_trade_cycle.go::StartTrade` 仅当 `FutureEnable==1` 才运行 Main 的 `runAccountTradeCycle`。虽然 `feature/feature.go` 的交易循环已经用 `runner.AllowNewOpens` 在持仓处理后阻断开仓，但 Main Runner 被硬编码为 `true`，而启动入口在关闭开关时不执行任何 Main 周期。

**修改范围**：`feature/account_trade_cycle.go`、`feature/feature.go`、必要的 `feature/ownership.go` 回归测试；配置中心开关下添加简短说明，原字段不变。

**具体做法**：
- Main Runner 的 `AllowNewOpens` 改为与最新 `FutureEnable==1` 挂钩；`StartTrade` 即使 OFF，也应在**本地仍有 Main auto_strategy 活跃 Managed 仓位或未终结订单**时运行退出/恢复分支。OFF 且确无此类任务时不额外发 Binance 轮询。
- 将 `runner.SelectCoins` 的计算移至持仓同步、策略退出之后。否则候选选币失败或候选为空会提前 `return`，把已有仓位平仓一起阻断。出场依赖已载入的 `allCoins` 与 Managed 仓位；如对应币种配置缺失，明确告警且不得冒险补猜策略参数。
- OFF 时继续核实已有订单、取消属于自动策略的过期 OPEN 挂单、执行原自动平仓/止盈止损、刷新真实 Fill、维持现有恢复处理。**OFF 不是立即强平，也不是取消全部保护单**。
- 禁止新 OPEN，且不允许因为已有仓位、Owner 或当前币种被关闭而把外部/手动仓位冒认成 auto_strategy。
- 避免启动 Main 和 Lead 双循环重复执行同一账户；Lead 保持现有独立逻辑。

**必测**：OFF+无仓位→0 新单/0 不必要远端调用；OFF+LONG/SHORT Managed 达到出场条件→各产生唯一 CLOSE；OFF+不满足出场条件→继续持有；OFF+超时 OPEN→仅取消属于本系统的 OPEN；OFF+未知 Owner→不碰外部仓位；选币返回空或报错仍可退出已有 Managed；ON+同样输入回归原先开仓；Lead 与 Spot/CM 行为不变。

**验收**：关闭总开关后，开仓计数为 0，已有合法 Managed 的退出、结算和异常恢复仍能正常推进；无法安全退出时显示清楚的原因。

## 四、UF-2：杠杆/保证金配置失败，开仓 fail-closed

**已确认行为**：`feature/feature.go::UpdateSymbolTradeInfoContext` 丢弃 `binance.EnsureTradeConfigContext` 错误；`feature/account_trade_cycle.go::mainTradeRunner.EnsureConfig` 无条件返回 `nil`。下游在计算出入场数量后仍可能尝试提交订单。

**修改范围**：`feature/feature.go`、`feature/account_trade_cycle.go`、`feature/api/binance/trade_config_cache.go` 的测试、`feature/*test.go`。

**具体做法**：
- 让配置函数提供可传播的 `error` 返回值，Main 的 `EnsureConfig` 原样返回；失败直接跳过当前币种和方向，不执行 `SubmitOpen`。不再无条件声称已设置杠杆或保证金模式。
- 复用现有 `EnsureTradeConfigContext`：已知 `-4046`（保证金模式已经相同）可接受；真正网络失败、权限失败、杠杆变更失败、非法参数必须阻断。成功配置缓存继续保留，避免 2 秒重复请求。
- 开仓前二次读取/比较 `symbols.enable`、`leverage`、`marginType`、`usdt`、`strategy` 和已使用的精度等关键字段，若在远端配置 RPC 期间被修改，丢弃旧信号交给下一轮，不用旧数量发单。
- 此检查**只作用于新 OPEN**，不阻断已持仓 CLOSE 或恢复。错误原因进入 UF-4 诊断，不暴露 API key、签名 URL 或敏感响应体。

**必测**：设置保证金失败、设置杠杆失败、网络超时、429、非法杠杆、配置编辑竞态时 `SubmitOpen=0`；既有配置/成功缓存不重复调用；`-4046` 后杠杆设置正常可开仓；LONG/SHORT 两方向；已有仓位的 CLOSE 无须因此被阻断。

## 五、UF-3：异常订单定位（先于“没有开仓”诊断）

**目前已有**：`service/systemhealth/service.go` 的 `TradeCheck` 汇总 `reconcile_required`、`fill_verification_pending`，以及 `agent_trade_proposals` 的 `execution_uncertain` 和 `protection_failed`。**后两项是 Agent proposal 的状态，不能直接视为所有 Main Managed Orders 数量**。合约订单列表按持仓周期展示，某些未产生 `position_id` 的异常 OPEN 订单不会出现在周期主表中。

**修改范围**：`service/futuresownership` 的只读查询、`service/orderhistory` 的异常列表或薄适配层、`controllers`、`routers/router.go`、`src/views/order/index.vue`、`src/views/ai/controlledTrade.vue`、系统看板相关页面。

**具体做法**：
- 建立同一套**只读异常分类规则**：Managed 订单待核对/超时未知、Managed 仓位对账异常、已关闭未 `settled` 且需补齐数据、REST Fill 待核实；Agent proposal 的 `execution_uncertain/protection_failed` 单独按来源展示。
- 新增一个需 JWT 的轻量只读查询，例如 `GET /futures/order-anomalies?account_id=main&type=...&page=...&limit=...`，结果给出 `account_id`、`owner`、`symbol`、`position_side`、`position_id`（可空）、`client_order_id`、原始状态/原因/更新时间和导航目标；不在页面点击时触发自动修复。
- 系统看板点击异常数字跳到「合约订单」已有过滤上下文。能定位 `position_id` 的自动展开子订单及 Fill；无 `position_id` 的未成交 OPEN，展示专用小型订单明细面板（不创建新的“订单历史主表”）。
- 与现有 `systemhealth` 统计保持口径统一：显示「系统健康汇总」和「当前可定位记录」，差异与账户/Owner/历史范围的过滤含义明确说明。
- 未验证的交割/结算不能标记为成功；人工历史覆盖不可消灭真实异常告警；手动认领仍沿用既有确认与归属约束。

**必测**：异常记录有持仓/无持仓两种路径、Agent proposal 不误指向普通 Managed、Main/Lead 不串账户、错误详情不暴露凭证、无 JWT 拒绝、历史人工标注不影响查询、无异常时空列表准确。

## 六、UF-4：为什么没有开仓（诊断而非重新计算策略）

**目前已有**：`feature/trade_selection_diagnostics.go` 可记录 `smart_local_v2` 无候选原因，但主要在日志；`feature/feature.go` 的热路径包含总/方向开关、持仓数量、亏损数量、选币、持仓与挂单占用、策略信号、行情、交易精度、账户配置及订单提交等门槛。

**修改范围**：`feature/feature.go` 与 `account_trade_cycle.go` 的只读观察器、`scanner` 现有选币诊断接口（仅需时）、Controller + Router、`src/views/futures/symbol.vue` 及 i18n。

**具体做法**：
- 在现有 2 秒热路径内采集**真实执行过的判定结果**，不要为页面额外执行策略或再次调用 Binance。每个本轮候选币种/方向记录 `symbol`、`LONG/SHORT`、`reason_code`、可理解的简短文本、`observed_at`、上次成功进入开仓流程时间。
- 原因按阶段展示：`global_off`、`symbol_off`、`direction_off`、`no_candidate`、`market_data_stale`、`volume_low`（必须有真实过滤证据）、`cooldown`、`max_positions`、`max_losing_positions`、`existing_position`、`pending_order`、`strategy_not_matched`、`invalid_quantity`、`depth_unavailable`、`trade_config_failed`、`order_submit_failed`、`not_evaluated`。不要把“没入选选币池”凭猜测细化为成交量不足。
- 单进程内存保存最近一次或有限条状态，保留最近更新时间，限制最大记录数；重启显示“暂无本轮诊断”，不需要新数据库表、不写每 tick 日志或发送通知。
- JWT 只读接口如 `GET /futures/entry-diagnostics`；合约交易页面增加「未开仓原因」轻量查看区，可按币种/方向查询；每 10～15 秒按需刷新，用户只见必要说明。
- 错误与“策略没触发”分开呈现；页面观察接口不改变开关或策略。

**必测**：总关/方向关、无候选、策略未命中、挂单/已有仓位、风险数量上限、网络故障、UF-2 配置失败。状态对并发请求安全、内存有界、含时间戳、无额外 DAPI REST 调用、不改变交易结果。

## 七、UF-5：已核实的实盘策略效果统计

**数据源**：`futures_managed_positions` → `futures_managed_orders` → `futures_trade_fills`；优先复用 `service/orderhistory` 和 `service/futuresownership/settlement.go` 的身份、账户、成交和结算验证。只统计 `account_id=main`、默认 `owner=auto_strategy`、真实 `settlement_status=settled`、完整 REST 核实、同一结算资产的周期。人工覆盖的“verified”不算实盘认证。

**修改范围**：新建轻量只读 `service/orderhistory/performance.go`（或 `service/outcomereview` 内现有合适方法）、只读 Controller / Router、`src/views/order/index.vue` 的「策略表现」区域（无需新增顶级菜单）、中英文 i18n。

**统计口径**：
- 默认近 **7 天/30 天**（按真实 `closed_at`，不是最新列表页/开仓日），可按币种、LONG/SHORT、`source_ref` 中的策略规则 Hash 筛选；没有稳定策略版本名时仅显示 Hash/未知版本，不伪装成当前模板名。
- 已结算交易数 `N`；胜率 = 净盈利周期数 / N，净零盈亏单独标记；净收益 = 真实已实现 `realized_pnl - commission`，手续费币种不同且无可靠换算时该周期**不纳入可合并净收益**。
- PF = 正净收益总和 / 负净收益绝对值总和；当没有亏损交易时返回 `null`/“暂无亏损样本”，不伪造“∞盈利能力”。
- 展示资金回撤时，基于**结算周期累计净利润**计算金额的峰谷回撤，名称写为“累计已结算净收益回撤”；**不能当成账户权益最大回撤百分比**，缺少账户权益历史就不声称有。
- 展示平均单笔净收益、LONG/SHORT 分布、各币种汇总、样本数；跨 USDT/USDC 分资产输出，不经兑换直接相加。
- 若范围内有 `incomplete`、`legacy`、手续费币种不明或 Fill 缺失，另外提供“排除样本 N”的原因计数；未完成结算的周期不得以当前市价估算混入胜率/PF。

**必测**：真实多次部分 Fill、手续费、LONG/SHORT、零收益、全胜/全亏/无样本、不同币种不同结算资产、人工手改历史状态、缺 Fill、相同策略 Hash 不同币、时间边界、旧周期；测试期望与实际结算权威结果对齐。

## 八、UF-6：订单历史页面性能（最后处理，结果一致优先）

**已确认实现**：`service/orderhistory/history.go::List` 当前对所筛选的全部周期逐 100 行批量 `project`，生成 `filtered_pnl`。分页操作也会重新计算全范围收益；这是**重复全范围扫描**而不是原先“只能统计当前页”的功能缺失。

**修改范围**：`service/orderhistory/history.go`、`pnl.go`、Controller / Router、`src/views/order/index.vue` 与接口定义、性能测试。只读且不改变原始 Managed/Fill 数据。

**具体做法**：
1. 先建立 SQLite/测试 MySQL 8 的 **100/1,000/10,000 周期**性能基准：列表响应时间、SQL 次数、扫描行数、内存、正确的筛选范围总收益；记录对应 DB 版本和数据量。
2. 将“当前页列表”与“筛选范围总收益”拆分为独立可缓存的只读查询（例如 `GET /orders/summary`），保持兼容旧 `GET /orders` 契约或以可选 `include_summary` 参数平滑迁移。
3. 前端分页翻页**只请求当前页数据**，仅当账户、Owner、币种、方向、状态、结算或时间筛选条件改变时重新请求汇总；适度短缓存并提供「刷新」强制重算，明确估值时间。
4. 保留每批读取 orders/fills 的批量关联，不改成逐行 N+1 查询。使用 `EXPLAIN` 证明确有瓶颈后再考虑索引；**不先建冗余汇总表，不做预计算异步队列**。
5. 原逻辑包含未结算持仓的估算盈亏与 `symbols.close` 快照；优化后必须逐资产、逐筛选条件对比相同结果，`null` 与未知值不能变成 0，不能把人工审核状态当成已核实结算。行情价时间戳继续返回。

**必测**：分页 1→2→3 不重复触发全量汇总；改变筛选立即失效缓存；空集、超出范围页、跨 USDT/USDC、存在异常/手工标注、实时新 Fill、超时取消；100/1000/10000 对照数据**逐项精确一致**，在指定测试环境至少证明查询数量或端到端耗时明显下降，不为性能牺牲结果正确性。

## 九、每阶段完成标准（统一）

- 每个阶段文件审查范围和变更说明写到 `doc/trade/CODE_REVIEW_UF-N.md`，复审不覆盖原报告。
- 在隔离环境跑对应单测、`go test -count=5 -p=2` 关键包、`go test -race`、`go vet`、`go build`；功能涉及 UI 时再跑 `pnpm typecheck`、`pnpm build` 并核对 `dist → backend/static`。
- 必须保证 Main/Lead 账户隔离、未知委托结果 fail-closed、真实订单唯一性以及真实 Fill/账户结算语义；保留历史人工状态仅展示的约束。
- 若真实 MySQL、Demo/Testnet 或网络集成未执行，在报告中明确“离线已通过 / 实盘未验证”，不能宣称生产安全已验收。
- 不修改 `conf/app.conf`、不写生产 DB、不启动真实开平仓、不随意清理旧订单记录；只在用户授权时进行部署或迁移。

## 十、开始方式

先做 **UF-1**。完成并复审后依次 **UF-2 → UF-3 → UF-4 → UF-5 → UF-6**。优先修正交易安全边界，不为诊断和统计添加额外风险开关或新菜单。
