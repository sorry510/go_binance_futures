# CODE_REVIEW Lead-04：Lead 独立账户手动交易开关

> 2026-11-17，本地代码改造。用户要求跳过独立真实 Lead Portfolio 凭证只读验证，检查与 Main 的账户隔离，并将永久硬锁改为默认关闭、人工开启。没有实际 Binance 下单。

## 修改内容

1. 仍通过原有加密凭证与独立配置文件（BINANCE_LEAD_CREDENTIAL_KEY、BINANCE_LEAD_CREDENTIAL_FILE、BINANCE_LEAD_CONFIG_FILE）保存 Lead API Key 与风险设置，不修改 app.conf。配置缺失、单笔/总金额上限不完整时不能开启。
2. Lead 页面自动开仓开关默认 false。用户主动打开并二次确认后，PUT /futures/lead/config 保存 allow_new_opens=true；关闭则恢复 false。保存前必须能够启动原有独立 Lead WS 观察器，不能把初始化失败的开关保存为成功。
3. 使用原 StartTrade 共享调度；Lead WS 健康且私有账户/币种缓存绑定一致时，ConfigureLiveSession 将持久化开关映射至 RiskController。无需单独点击 VerifyReadOnly；正常交易仍必须通过实时账户、资金、Hedge Mode、仓位、订单、交易规则和风控检查。
4. 新开仓关闭时不影响已管理仓位的 CLOSE_LONG/CLOSE_SHORT。原紧急暂停是独立的一次性进程闩锁，普通开关关闭后可以再次手动开启；紧急暂停须重启才能恢复。
5. 凭证轮换前先持久关闭旧新开仓授权，关闭旧 WS Host；更换 Key 不能沿用旧开关状态。

## Main / Lead 代码级检查

| 模块 | 结论 |
| --- | --- |
| EncryptedFileStore | 只从 Lead 独立加密文件读取，不从 app.conf 回退 Main Key |
| NewLeadAccountClient | 拒绝与当前配置 Main 正式或测试 Futures API Key 字符串相同的 Lead Key |
| CurrentLeadTradeSession | 使用同一个 Lead WS Host 的 Client，不使用 Main 的私有 Client |
| newLeadTradeIO / BindAccount | 仓位、挂单、Managed/Fill 均限定 account_id=lead |
| LeadExecutionAdapter | Broker 和专用 Lead Client 必须是同一实例，订单仍走 Ownership 及原幂等客户端 ID |
| runAccountTradeCycle | 只共享 Main 的策略/选币/公共市场数据；私有开平仓由 Lead ExecutionAdapter 提交 |
| 实际持仓退出 | 校验 Lead 真实账户仓位、Managed 归属、数量、WS 健康 |
| 原主合约 | 调度和私有账户分支未改变，不新增第二套 Scheduler |

**限制**：仅靠静态检查不能证明两个不同 API Key 一定属于两个不同币安账户，亦不能证明 Key 归属于某个指定 Lead Portfolio。已经拒绝相同 Main Key 与所有代码层 Main Client 回退；若用户输入另一把属于 Main 账户的有效 Key，不做真实交易所身份查询就无法判别。使用者必须自行核对。

## 本地测试证据

- 默认关闭、有效风险限制才能持久开启、关闭立即阻断；无 Key/WS 启动失败不保存启用状态。
- 使用 Mock Lead WS 镜像验证自动启用条件、普通关闭后受控平仓资格、断线阻断；Main 身份与 WS Client 不一致均拒绝。
- Main Futures API Key 与 Lead Key 相同的正式/测试账号均在客户端构造时拒绝。
- Controller ON/OFF/错误回滚用例在单独测试进程中通过，不连接币安。完整 Controller 测试中如已有旧测试永久触发紧急暂停，这两项测试跳过，不能擅自清除安全闩锁。
- 保留 Stage 4/6 AST 安全审查、订单幂等/WS 与 Main/Lead Ownership 隔离测试。

## 环境与发布

没有修改生产数据库、app.conf、main.go、Router；没有运行真实 Lead 订单，没有部署重启服务器，未提交/推送 Git。用户打开开关后可能产生真实订单；没有获得主网身份验证和真实成交证据。
