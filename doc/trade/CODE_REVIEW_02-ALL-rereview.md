# 方案 02 整体 Code Review · 修复与复审（02ALL R1）

审查日期：2026-11-10  
来源：[`CODE_REVIEW_02-ALL.md`](CODE_REVIEW_02-ALL.md)，**保留原文，不改写原审查证据**。  
状态：**可安全精简项已修复并离线验收；真实成交、授权、历史资产对账和生产部署仍 BLOCKED；方案 02 不视为完成。**

## 一、按 Finding 逐项处理

| Finding | 判定 | 本次实际处理 / 不处理的理由 |
| --- | --- | --- |
| **02ALL-F1 P1** — 成交账本运行闭环断裂 | **仍 BLOCKED** | `canAttestLiveFills=false`、`spotRealOrderReleaseApproved=false` 保留，不能从 SDK BaseURL、Fake 或未核实的 REST JSON 推导 `rest_verified=1`。响应级请求-订单-tradeId-手续费身份校验、可信证据持久化和独立可撤销授权尚未完成。**禁止为了“复刻合约”而直接把两个 false 改成 true。** |
| **02ALL-F2 P1（设计冲突）** — “完全无 Account”与手动库存保护矛盾 | **已澄清边界，表设计不动** | 普通 `spot_symbols` 列表/配置与模板均无 Account/Owner；**资金账本中的 `main/auto_strategy` 只是内部归属标签，不是前端多账户选项**。保留三张资金表、幂等键、`active_slot` 和手动库存隔离，不能取消后再卖出未知资产。没有进行 v26 模型迁移。 |
| **02ALL-F3 P2** — 账户页 mode/版本握手与重复 scope | **已修复 UI/API 耦合；保留必要 SQL 归属** | `spot/account.vue` 移除 `getSpotStrategyMode()`、`contract_version===4` 检查及 `main/auto_strategy` 固定标识；直接读取现有 Holding/Orders/Fills。前端 `spotStrategyScope`、`scope()` 与无消费者的 `getSpotStrategyMode` 调用删除。服务端仍按固定 AccountMain/OwnerAuto 查询并校验 JWT；无 scope 参数请求可以正常读取，显式非法 account/owner、部分 scope 仍 HTTP 400。**不可移除 SQL 层内部 Owner 隔离**。旧 `signal_key` 哈希包含归属标签属于已存订单幂等身份，改它会改变历史重试键，故保留。 |
| **02ALL-F4 P2** — 冗余预检和运行代码复杂度 | **部分修复** | `s3_legacy_preflight.go` 与对应测试加 `//go:build spot_legacy_doctor`，从默认 Go 运行构建中排除 286 行**零生产消费者**的只读诊断，仍用明确 tag 运行原 9 项回归。**没有新增 doctor 命令、后台扫描或真实账户访问**。旧现货下单源防护 `AuthorizeLegacyOrder`、未知状态、CAS、防手动库存误卖、Fill 来源防伪及订单状态机均保留；拆分到 `orderstate` 子包属于后续重构，不在没有等价安全证明时删除。 |
| **02ALL-F5 P2** — 二进制和 static 半发布风险 | **仍 BLOCKED；未热发布** | 本地只读探测无 `go_binance_futures` 进程，TCP 3333 无监听；但硬盘已有二进制 SHA-256 `6c76017514c1df9c7c36bc4e21baf04e9f01520abf247cd6f5b16223828daf2f`，本次临时编译 SHA-256 `3b8f7e3521e988a538c2530bb4b8fee5fdb336523daece0d6b7f0294b783677d`，两者不同。新 `dist` 与旧 `static` 均为 119 文件，**哈希未全部一致**。为避免“新前端旧 Go”重演，本次**没有覆盖 static、替换主二进制或重启服务**；待授权成套发布。 |
| **02ALL-F6 P2** — 文档范围语义冲突 | **已修复** | 总方案开头和最低安全原则明确：Owner/Account **仅用于下单、Holding、Fill 与资金隔离**，不用于现货币种列表、策略配置或前端参数；更新 S3-P0、S4、R5、部署 SOP、中英文 README。历史报告/阶段记录保留并由最新章节覆盖。 |
| **02ALL-F7 P3** — 全量 `go test ./...` 红点 | **未处理原有非 02 缺陷** | 本次没有声称或执行“全仓测试通过”。原报告列出 controllers 策略模板相关 fixture 红点，以及 `tests` 依赖不在 Git 内的 `conf/app.conf`；本轮仅复测已列出的目标包/Controller 正反用例。不得复制生产配置来强行压绿。 |
| **02ALL-F8 P3** — 独立的 `spot_closed_env.go` | **保留** | 这是已收盘 K 线筛选、禁止 Futures FundingRate/ROI 混入 Spot 求值的边界，与合约直接共用环境语义不同；简单移除会改变交易信号行为。 |

## 二、代码调整明细

1. `src/views/spot/account.vue`：去掉 mode API 前置请求与版本不匹配禁用逻辑；本地账户和受控交易直接走只读服务，显示资金未核实与交易未授权提示。获取失败会清理当前页过期数据，避免以前一次页面缓存推断无风险；资产 tab 使用 Holding 状态筛选，而不是订单状态。
2. `src/api/spotStrategy.ts`：删除 `spotStrategyScope`、`scope()`、前端私有 mode API 包装；只读请求只传币种、订单/仓位 ID、状态和分页等查询字段。
3. `controllers/spot_strategy_s4.go`：在 `s4ReadScope()` 只要求 JWT，缺省范围由服务器硬编码；旧调用可传正确范围，但非法覆盖和只传一半范围仍拒绝。**真正 SQL 仍使用 AccountMain/OwnerAuto**，没有放开跨 Owner 读取。
4. `controllers/spot_strategy_s4_positive_test.go`：新增 `TestS4AccountEndpointsDefaultServerOwnedScope`，对 Holding/Orders/Fills 检查无 scope 的 HTTP 200，并核实 `lead`、`manual` 和不完整范围的 HTTP 400。
5. `service/spotstrategy/s3_legacy_preflight.go`、`s3_legacy_preflight_test.go`：编译标签 `spot_legacy_doctor`。普通构建不可访问预检函数，但可以在隔离测试中显式加入标签运行；这**不是**新增一个生产可执行 `doctor` 命令。
6. `scripts/spot-strategy-browser-e2e.mjs`：增加 mode 接口模拟 503 时本地账户/受控交易仍可加载的场景，断言不再向任何只读 GET 传 Account/Owner，并保留模板、Quote 分页、只读拦截和中英文回归。

## 三、本地测试结果与命令

| 检查 | 实测结果 |
| --- | --- |
| `go test -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./feature/strategy/line ./command` | **5/5 PASS** |
| `go test -count=1 ./controllers -run '^Test(S1SpotStrategy\|S4)'` | **PASS** |
| `go test -count=1 -tags spot_legacy_doctor ./service/spotstrategy -run '^TestS3Preflight'` | **PASS**，原 9 项按需诊断证据仍存在 |
| `go test -race -count=1`（上述 5 包） | **5/5 PASS**；只有非致命 macOS `ld` 报警 |
| `go vet`（5 目标包和 controllers） | **PASS** |
| `go build -o /tmp/.../go_binance_futures .` | **PASS**，临时二进制 38,974,546 B，临时目录已清理；**未替换项目二进制** |
| `pnpm typecheck` / `pnpm build` | **PASS** |
| `pnpm test:s4-browser` | **14 项业务/交互场景 PASS**，164 次隔离本机 HTTP GET、0 POST/PUT/DELETE；其中新增账户页面 mode=503 时仍可读取 |
| `git diff --check` / 文档相对链接检查 | **PASS** |
| `go test ./...` | **SKIPPED**，不得声称通过；旧报告 F7 需独立修复 |

以上均是本地代码与独立 SQLite/Fake/Headless Chrome 测试，没有使用线上交易 API，也没有证明真实 REST 响应受信、真实账户中存在可证明归属的 Holding。

## 四、当前安全边界和后续 Gate

- **状态继续 BLOCKED**：真实订单发送/撤单；授权审计（可撤销、过期、操作员绑定）；交易所 `myTrades` 响应级来源认证；旧订单、余额、人工持仓清点和退出义务；完整 BUY→Holding→SELL→settled/PnL；生产 v25 和前后端成套部署。
- **不改 v25、不删除资金 Owner 字段、不修改信号幂等历史键**。过去的 `s4ReadScope` 账户校验已由“服务端内部固有查询范围 + 可选显式范围验证”替代，不能理解为跨账户读取许可。
- **F5 不虚报消除**：本地磁盘二进制与新源码不是同一构建，`dist` / `static` 未匹配。本轮仅在前端仓库创建新 `dist`，没有向仍有旧二进制的后端 `static` 热复制。上线前必须在授权维护窗口构建/核验同一候选包的 Go 二进制和前端资源，以及 JWT 正反测试、真实 DB Schema 与回滚证据。
- `CODE_REVIEW_02-ALL.md` 为原始审查；本文件仅记录**已实施整改、保留风险和不予放行的决定**。不能将 F1/F2/F5 的 BLOCKED 改写成“02 已完成”。


## 五、02ALL R2：归属参数与重复校验下沉（2026-11-10）

继续处理 **02ALL-F3 / F4 中无业务价值的重复参数**，本次只调整现货 **S4 只读 Service 层**，没有触碰实际 POST/DELETE 委托或存量仓位数据库：

- `service/spotstrategy/s4_read.go` 的 `ListHoldings(ctx, db, req)`、`ListOrders(ctx, db, holdingID, req)`、`ListOrderFills(ctx, db, localOrderID, req)` **不再接收 account/owner**。旧实现虽然接收参数，实际 SQL 已始终使用 `AccountMain/OwnerAuto`，参数只触发重复的 `ValidateScope`；现在直接把“仅本地策略归属”固化在唯一 SQL 读取路径。
- 控制器 `controllers/spot_strategy_s4.go` 改为按上述新签名调用；入口仍验证 JWT，若旧客户端显式传入非法 Account/Owner，继续 HTTP 400。去掉额外的 `ValidateS4Scope` 包装，直接复用 `ValidateScope`；删除未调用的 `ValidateS4Symbol`。保留 `account_id=? AND owner=?`、`holding_id` 归属验证和本地 `order_id` 与 Fill 的身份关联。
- `service/spotstrategy/s4_read_test.go` 原测试“给 Service 传 `owner=manual` 会拒绝”已不再对应新签名；新增更强的 `TestS4FixedInternalScopeRejectsForeignInventoryAndFills`：向独立 SQLite 测试库插入 `main/manual` 的 Holding、订单及已标记 verified 的 Fill，按正常读接口查询只能得到**零条本地策略数据**，并且不能用其 Holding ID / 订单 ID 绕过归属查询。
- 既有 SQL 验证、未知状态/待核对可见性、签名错误脱敏、未核实成交不计算收益等测试保留。**这不是删除账户归属，而是去掉永远没被使用的调用参数**。

### R2 离线结果

`go test -count=1 ./service/spotstrategy ./controllers -run '^Test(S1SpotStrategy|S4)'` **PASS**；`go test -race -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./feature/strategy/line ./command` **5/5 PASS**，`go vet` 及 `git diff --check` **PASS**。macOS Race 链接器仍有非致命 LC_DYSYMTAB 警告。

**保持 BLOCKED**：F1 真实来源认证与 Holding→结算链、F2 真正删除数据库 Owner 所需的产品取舍、F5 二进制/static 成套发布，以及全部实盘权限。Schema v25 不变，`spotRealOrderReleaseApproved=false` 与 `canAttestLiveFills=false` 未修改。


## 六、2026-11-10：S4 现货搜索与配置开关功能修复

S4 现货币种列表原使用精确 `symbol=?`，现改为经字母数字白名单验证的包含查询 `LIKE ?` 并在 SQL 分页前过滤。Vue 增加防抖查询。S4 配置开关改为最小字段 PUT，避免提交未改变的策略/指标 JSON；保留正数预算、已启用 LONG/CLOSE_LONG 策略和真实 Expr 有效性校验。最小字段关停即便旧配置损坏仍可恢复；不能将 UI 启用等同于真实交易放行。测试记录见 [S4-R5 §9](开发方案-02-S4-R5-币种列表去账户绑定.md)。本次新 Go/Vue 构建未部署到运行进程。
