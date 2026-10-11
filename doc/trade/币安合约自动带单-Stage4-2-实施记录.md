# Binance Futures 自动带单 — Stage 4-2 Lead 账户开仓风控

日期：2026-10-10

**阶段结论：Lead 专属开仓风险核验器已完成离线实现与共享循环 Mock 接口接线；无生产 Lead 启动工厂/真实订单入口。真正的资金/盈亏采集和跨请求订单预留仍待 Stage 4-3/Stage 5 账户级执行链完成后验证。**

## 1. 代码实现

- 新增 `service/leadaccount/risk.go`：`RiskController`、`RiskLimits`、`RiskOpenOrder`、`RiskSnapshot`、`RiskDecision`；专属于 `account_id=lead`。只读取显式传入的 Lead 风控证据，不能从 Main 获取资金/仓位，也不读取或写入数据库。
- `NewRiskController()` 固定从未配置、关闭且暂停开仓开始，只有 `SetLimits` 和 `Pause`；**Stage 4-2 未提供生产 Enable/Resume 功能**。设置额度本身不能启用交易；未通过身份、真实 Portfolio 绑定、WS、Stage7 明确授权等安全 Gate 始终拒绝。
- 账户资金：强制钱包/可用资金/权益有效并与 Lead 身份匹配，按名义额/杠杆计算初始保证金并留 **2% 安全垫**；保证金不足拒绝新开仓。
- 交易金额：未配置额度（0）拒绝；校验单笔保守名义额、Lead 总名义额（包括全账户所有受控/人工仓位、LONG/SHORT、挂单**剩余未成交数量**）、币安订单最小/最大名义、杠杆限制，以及币种原共享 `Usdt*leverage` 上限。
- MARKET 使用保守的 **0.5% 参考价偏移**计算 Lead 风险额度及保证金；共享币种原金额上限按**实际下单计划名义额**核验，避免所有正常 MARKET 订单被误拒。LIMIT 不加此报价缓冲；数量/价格合法性及过滤规则由未来账户执行适配器提供已验证证据。
- 仓位控制：账户已有仓位与剩余开仓挂单一起计算持仓槽位、同币同方向冲突；亏损仓位沿用 Main 的 `FuturesLeveragedROI<-0.1` 百分比口径。所有账户持仓都计入（包括人工），但只读取 Lead，不跨账户。
- 当日亏损：`DailyNetRealizedPNLUSDT` 必须是完整 Lead 来源的**实际已实现净盈亏（含手续费与资金费）**；按 UTC 00:00–24:00 划日，要求完成标记 `PNLComplete`、正确 `PNLDayStartUTC`、覆盖到本次风险快照时刻的 `PNLAsOf`。无法证明完整时 **fail-closed**；亏损达到设定阈值时阻断开仓。
- 最大回撤可选，启用时须有可证实的账户权益高水位 `PeakEquityVerified`；无可信基线不能绕过阈值。
- 快照必须是该账户已完成仓位/挂单/PnL 对账，无 unknown orders，默认 **10 秒新鲜度**；请求中断、挂单/数量不合法、白名单失效一律拒绝新开仓，返回固定 `blocking_reasons`。

## 2. Stage 3 共享交易循环接线

- `feature/account_trade_cycle.go` 加入账户级 `PreflightOpen` 注入式风险检查；Lead Runner 缺失该依赖时 `validate()` 明确失败，Main 完全不依赖它。
- `feature/feature.go` 的 LONG/SHORT 分支分别在计算本次精度后的价格/数量、设置交易所账户杠杆/保证金以及 `SubmitOpen` **之前**执行 Lead 风控；被拒方向跳过新开仓，但另一方向照常评估。主账户仍返回 True，不额外调用风控服务或 Binance API。
- Lead Mock 工厂只存在 `_test.go`，这次测试专用 Hook 允许模拟不同风控结果；**真实** Lead Runner/执行工厂尚未创建，亦未启动任何后台任务。
- 只限制新开仓：原先平仓、Ownership 对账、受控订单超时撤销和通知路径不通过 `PreflightOpen`，不会因为达到“单笔/日亏损”阈值而被自动强制平仓。

## 3. 测试与边界

- `service/leadaccount/risk_test.go`：默认关闭；完整 Mock 风控证据；账户隔离、异常/过期/未对账快照、白名单与身份、资金/名义/杠杆/交易所精度、日亏损与 UTC 跨日、LONG/SHORT、未成交挂单敞口、余额不足、最大持仓/亏损数、可选回撤、MARKET 缓冲与配置金额基线。
- `feature/account_trade_cycle_stage4_2_test.go`：默认风控拒绝不会发单；LONG 风控拒绝时 SHORT 可执行；Main 不访问 Lead 预检查；Lead 缺失 Hook 拒绝。
- 继续运行 Stage 1–4-1 及 Stage 3 原有 Mock / Golden 测试，要求 Main 开平仓历史与策略计算行为不变。

## 4. 尚未解除的安全门禁（后续明确处理）

1. **真实 Lead 资金证据采集**：Stage 4-3/5 必须从明确绑定的 Portfolio 专用 Key 获取最新资产、所有活动仓位、所有开仓未成交委托与真实已实现净 PnL；不能用 Main 历史订单/不完整页面数据或只读 `userStatus` 替代。资金/手续费/资金费数据要可靠分页并覆盖 UTC 日起点至快照时刻。
2. **交易前并发保留额度**：`CheckOpen` 本阶段是单次只读预检查，**不能替代** Stage 4-3 的账户级锁、订单 Claim/幂等/unknown-submit 阻断与确认成交后的实际风险对账；若跨实例运行，必须依赖数据库事务/唯一键，不得将 Go Mutex 当作分布式锁。
3. **可操作启停**：只允许 Stage 6 强鉴权管理界面结合 Portfolio 人工核对及 Stage 7 小额授权解除门禁，不能通过修改配置布尔值直接绕过。
4. **动态止盈/止损与保护订单**仍使用共享策略和 Ownership 已有逻辑，本阶段不复制策略也不执行真实 Binance mutation。私有 User Data WS、账户健康/限流与警报在 Stage 5。

## 5. 约定

不修改 `conf/app.conf`；不执行 `sync db` 或真实账户写请求；不自动提交/推送；不修改策略模板研究文件；测试进程结束后不保留后台进程或监听端口。实施不需要新的 schema / DB 版本。
