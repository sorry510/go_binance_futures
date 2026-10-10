# 币安合约自动带单 — Stage 4-1 账户配置与只读身份验证

日期：2026-10-10

**范围：完成账户配置底座、独立凭证加密保存及显式只读验证。未接入真实 Lead 下单、后台任务、WS 或前端 API。任何凭证保存/只读验证都不能打开真实交易。**

## 1. 实施文件

- `service/leadaccount/credentials.go`：独立 Lead 加密凭证仓库，AES-256-GCM + 随机 Nonce + 账户/版本 AAD，原子写入（临时文件后 rename）；保存文件权限 `0600`、目录须预先设为 `0700`。禁止相对路径、目录权限过宽和符号链接目标。任何密钥加载失败都 fail-closed，数据不进入普通 JSON 输出。
- `service/leadaccount/verify.go`：Lead 专属 `Manager`，账户 `lead` 固定。保存凭证仅在显式调用时发生；不存在注册到 Main 启动程序的默认连接、轮询或交易入口。保存/轮换后旧验证记录立即失效，同步验证通过 generation 拒绝旧凭证的晚到响应。
- `feature/api/binance/account_client.go`：新增基于明确 `AccountClient` 的签名只读 `GetPositionModeContext`，不写仓位模式。继续复用 Stage 1 的 `LeadTraderStatus`、`LeadTradingSymbols`、余额、仓位及挂单只读方法。
- 新增 `service/leadaccount/credentials_test.go`、`verify_test.go` 与 `feature/api/binance/account_client_stage4_1_test.go` 回归测试。

## 2. 凭证操作与部署前提

- 只能在**用户显式配置**时通过 `NewManagerFromEnv` 构造管理对象，服务启动并不自动调用。
- 外部环境变量 `BINANCE_LEAD_CREDENTIAL_KEY`：**32 字节随机值的标准 Base64 编码**，必须保存在可信的系统 Secret Manager / 服务启动环境，绝不放在 `app.conf`、Git 或日志。
- 外部环境变量 `BINANCE_LEAD_CREDENTIAL_FILE`：指向独立私有目录内的绝对路径（例如私有持久目录的 `lead.enc`）；目录须先由用户/部署程序创建并设为 `0700`，不是项目源码目录。保存只生成加密文件 `0600`，不自动建目录、不生成新主密钥。丢失主密钥后原文件不能解密；备份须连同密钥使用单独可信渠道管理。
- `SaveCredentials` 需要完整新 Key、Secret；轮换后 `KeyHint` 仅展示遮蔽后的后四位；PortfolioLabel 仅展示，不作为身份验证依据。
- Stage 6 再新增强鉴权、幂等、审计的 HTTP 接口，录入页面、校验按钮；本阶段没有新增路由、数据库表或数据库迁移。

## 3. 只读验证流程

手动调用 `VerifyReadOnly(ctx)` 才触发一次完整验证，45 秒整体超时：

1. 使用 Lead Portfolio 专用 Key 调用 SAPI `userStatus`，必须为 Lead Trader，否则停止进一步的账户访问。
2. 读取 `leadSymbol` USDT 白名单并去重，不在筛选循环内重复请求。
3. 使用同一 Lead Key 查询 USD-M Futures 账户 `CanTrade`、USDT 余额与可用余额，校验数据合法以及非 Multi-Asset Margin。
4. Fresh 模式查询账户仓位与所有未成交订单快照，不取 Main 的私有缓存。
5. 签名 GET `/fapi/v1/positionSide/dual` 验证 Hedge Mode 为开启。
6. 返回不含原始 Key、Secret、签名请求、原始交易所错误消息的 `VerificationReport`；失败只以固定 `blocking_reasons` 描述。

**重要边界**：SAPI 的 `IsLeadTrader` 和成功的 `/fapi` 读取，并不能保证 `/fapi` 数据来自用户选定的那一个真实 Lead Portfolio。真正的 Portfolio 身份和资金绑定必须与 Binance 官方 Portfolio 页面进行人工/证据校对，目前 `PortfolioBindingConfirmed=false`。只读检查全部成功时，仅允许 `ReadOnlyChecksPassed=true`；`TradingReady` **永远为 false**。

状态 `Enabled=false`、`AllowNewOpens=false` 为结构级默认关闭且本阶段无解除入口；还会固定提示 `portfolio_binding_not_confirmed`、`risk_gate_not_implemented`、`ws_not_ready`、`stage7_live_authorization_required`。

## 4. 测试及拒绝策略

- 凭证：AES-GCM 密文与轮换、错误主密钥、密文篡改、空 Key、路径权限、符号链接、未配置密钥、公开 JSON 不泄露 Key/Secret。
- 验证：正确 Lead 身份与六类 GET 快照、非 Lead 立即停止、每个请求阶段失败均阻断、非 Hedge Mode、NaN 余额、旧凭证中途轮换、再次验证失败不能沿用旧成功状态。
- SDK 客户端：`GetPositionModeContext` 使用明确的 Lead API Key 签名 GET，只发一次请求，无 POST/PUT/DELETE。
- 附注：Mock HTTP 不是 Binance Lead Portfolio 真实授权证明；系统不会在 Stage 4-1 读真实私有 API 或执行真实下单。

## 4-A. CODE_REVIEW_Stage4-1.md 跟进（2026-10-10）

原审计文档保持不变。审计结论为无 P0/P1，一项 P3 接口契约问题及后续阶段观察。本轮完成以下处理：

| 发现 | 状态 | 处理 |
|---|---|---|
| F1 · P3：截止时间/取消时 `VerifyReadOnly` 的 error 为 nil | **已修复** | 返回 `context.DeadlineExceeded` 或 `context.Canceled`；报告附固定 `verification_deadline_exceeded` / `verification_canceled`，`ReadOnlyChecksPassed=false` 且保存该失败报告。不信任取消后晚返回的 Reader，即使给出有效数据也禁止通过；进入验证前 Context 已失效则不发 API 请求。 |
| F2 · P3：真实 Portfolio 资金/持仓身份对应关系 | 保留 Gate 0-LIVE/Stage 6 | 需对照币安 Portfolio 页面与真实只读数据，不能依靠 `userStatus` 或展示名称自动确认，不修改现有 `PortfolioBindingConfirmed=false`。 |
| F3 · P3：删除凭证/持久审计时间或指纹 | Stage 6 | 需要强鉴权、告警与阻断未知单后的安全删除流程；本阶段不增加默认路由或删除入口。 |
| F4 · P3：单次验证 API 用量/耗时 | Stage 5/6 | 由既有 `lead_trading_sapi` 和 `lead_trading` 观测源接入界面，不在本阶段额外发起 API 请求。 |
| F5 · P3：45s 为整体预算 | **文档澄清** | 使用调用方 Context 与 45s 整体 Context，单个 SDK HTTP 读请求无另设子超时；先到的 Context 取消/超时终止后续可取消的请求。底层 Reader 如违反取消约定，返回结果仍强制拒绝。 |

**调用方契约**：`VerifyReadOnly(ctx)` 返回的 **`nil error` 只代表调用正常完成**；仍需同时检查 `ReadOnlyChecksPassed` 和 `BlockingReasons`，它不表示拥有 Lead 下单资格。超时/取消则必须返回非 nil 的对应 Context 错误。无论成功、失败或超时，`TradingReady=false` 且独立账户真实交易入口未开放。

新增永久回归测试：
- `verify_review_test.go`：真实 Context 截止时间、预先取消不请求、忽略取消的 Reader 晚返回、失败报告不能继承原成功状态。
- `credentials_review_test.go`：父目录 symlink、文件可执行位、非法长度/空值不能覆盖旧密文、无遗留临时文件及错误信封版本拒绝。
- `verify_review_edges_test.go`：报告 JSON 脱敏与防御性拷贝、USDT 白名单去重和错误资产模式拦截。

测试限于离线 Fake/Mock 和测试专用临时目录；未调用真实 Binance 私有接口，未修改用户凭证/业务 DB/生产配置。

## 5. 后续

- **4-2**：Lead 账户专属总/单笔名义额、每日已实现亏损、保证金及熔断 Gate。禁用状态保持。
- **4-3/4-4/4-5**：以明确 `lead` 账户绑定的 Executor、订单 unknown 查询/对账、暂停/熔断等为新开仓前置条件。
- **Stage 5**：独立 User Data WS、重启恢复、账户 API 预算与告警。
- **Stage 6**：安全凭证设置页面及 Portfolio 读写门禁/身份对照界面。
- **Stage 7**：经过明示授权再做小额实盘验证；不能将 Stage 4-1 只读通过当成交易授权。

不修改 `conf/app.conf`，不操作业务数据库，不自动 commit/push，不启停现有 3333 端口服务。
