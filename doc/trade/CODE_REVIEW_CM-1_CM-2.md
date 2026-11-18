# CODE_REVIEW_CM-1_CM-2 · 币本位独立模型 / DAPI 环境隔离与行情链路复审

> 日期：2026-11-17。分支：`feat/m-coin-futures`；HEAD `72cf42a feat: cm-1`（父 `fdcc261 feat: cm-0`）；master `0b3fa34 (v2.2.3)`。
>
> 审计对象：① 已提交的 CM-1（12 文件，+802 −35）；② 工作区未提交的 CM-2（13 项：6 M + 7 ??，已跟踪部分 +423 −124）。
>
> 本报告为**新增复审**，不覆盖 `CODE_REVIEW_CM-1.md` / `CODE_REVIEW_CM-2.md`。审计期间未修改用户工作区业务代码、未自动提交、未改 `conf/app.conf`、未访问真实 Binance、未对真实数据库执行迁移、未重启服务或触碰仓位。

## 一、结论摘要

| 项目 | 结论 |
| --- | --- |
| CM-1 源码实现（v27 / 唯一全局开关 / 四独立账本 / API 隔离） | ✅ 与方案和既有报告一致 |
| CM-2 源码实现（环境隔离 / DAPI 只读 / 精度过滤 / 元数据同步 / 市场 WS / Weight 隔离） | ✅ 与方案和既有报告一致，未发现真实下单入口 |
| 隔离副本离线 Gate | ✅ 通过：等价校验、`diff-check`、全量 `count=1`、`vet`、build、5 包 race、CM-2 专项 `count=5`、3 组隔离探针 |
| 重复运行稳定性 | ⚠️ 不通过：`controllers` 固定 ORM alias 导致 `-count>1` 失败；既有 CM-1 与 Spot Quant 测试均受影响 |
| 实盘 / 生产 Gate | ⛔ 全部 BLOCKED：真实 MySQL 8 v27、备份回滚、签名 WS、Demo/Testnet、真实下单、ARM Linux 成套发布均未执行 |
| 阶段意见 | **⚠️ 有条件通过（研发基线）**：可作为进入 CM-3 的代码基线；不得表述为已完成上线或实盘验收。F1 修复或收窄声明后方可宣称「`-count=5` 全绿」 |

**未发现 P1 级代码缺陷。** CM-1 / CM-2 的自评结论「可进入下一阶段」在研发静态与离线范围内成立；但 CM-2 报告中「定向 Go 测试 `-count=5 -p=2` PASS」需要限定为 `-run 'CM2'` 的三个包，不能泛指包含控制器 CM-1 测试的重复运行。

## 二、基线与等价性

```text
branch        feat/m-coin-futures
HEAD          72cf42a feat: cm-1
parent        fdcc261 feat: cm-0
master        0b3fa34 tag v2.2.3
worktree      13 项（6 M + 7 ??）
tracked diff  423 insertions(+), 124 deletions(-)
```

隔离副本：`git archive HEAD | tar -x -C /tmp/phase_audit_cm12`，再按 `git status --porcelain=v1 -z` 同步全部 M/?? 文件；副本内 `conf/app.conf` 由 `app.conf.example` 兜底。等价校验使用 `git ls-files '*.go'` + 工作区未跟踪 Go 文件清单逐文件 SHA256：

```text
SYNCED_CHANGES=13
GO_FILES_CHECKED=722
GO_SHA256_MATCHED=722
GO_SHA256_DIFF=0
GO_SHA256_MISSING=0
APP_CONF_PRESENT=True
```

未使用全仓 `os.walk` / `grep -r`，避免历史 research 目录 SIGTERM(137)。

## 三、CM-1 复审

### 3.1 方案契约对照

| 要求 | 代码证据 | 判断 |
| --- | --- | --- |
| Schema v27 且版本唯一真源 | `appversion/version.go:4` = 27；`command/db_update.go:377-380` 调 `ensureV27CoinMSchema` | PASS |
| 迁移不覆盖旧业务数据 | `command/coin_m_migration.go:13-97` 仅 ADD COLUMN / CREATE TABLE IF NOT EXISTS / CREATE INDEX；不改旧 U 本位、Spot、Lead 数据 | PASS |
| 已声明 v27 但结构损坏要 fail closed | `command/coin_m_migration.go:187-266` 检查列、`DECIMAL(36,18)`、索引、唯一性和开关值；`command/db_update.go:496-499,516-519` 每次 sync 校验 | PASS |
| 配置中心唯一新增全局开关 | `models/tableStruct.go:9` `CoinMQuantEnable`；`controllers/index.go:77` 只读输出；`150-160` 只允许单独 0/1 patch；`202-205` 只更新该列 | PASS |
| 每币开关复用 `delivery_symbols.enable` | 未新增每币开关字段；`models/delivery_symbols.go:11` 保留 `Enable` | PASS |
| 老 `usdt` 不作为新预算 | `budget_usd DECIMAL(36,18) NOT NULL DEFAULT 0`；DTO 不注册 ORM；`models/coin_m_quant.go:66-72` 明确 raw SQL 读取 | PASS |
| 四张独立业务表 | `coin_m_managed_positions / orders / trade_fills / strategy_templates`，见 `command/coin_m_migration.go:36-62` | PASS |
| 周期归属与 Fill 去重 | `position_id`、`client_order_id` 唯一、`(symbol, exchange_order_id, trade_id)` 唯一，见 `69-95` | PASS |
| API 单字段 0/1 校验与字段隔离 | `controllers/index.go:139-187,193-208` | PASS |
| 不接真实交易执行器 | 全仓 Grep 未发现 delivery SDK 的 `NewCreateOrderService`、条件单或改双向模式接口 | PASS |

### 3.2 CM-1 离线验证

- 探针 `TestReviewCM12V27SchemaFailClosedProbe`：SQLite 模拟 v26 → 执行 `ensureV27CoinMSchema` → 断言旧 `enable=1` / `usdt=10` / 全局开关 0 不变、`budget_usd=0`、四表空、重复 Fill 被唯一索引拒绝；随后人为删表，`VerifyV27CoinMSchema` 返回错误。**PASS**。
- 该探针只验证 Go 迁移逻辑；**不等于真实 MySQL 8 的 DDL、字符集、锁表、索引或回滚验证**。

## 四、CM-2 复审

### 4.1 环境隔离与 REST guard

`feature/api/binance/coin_m_environment.go:28-56` 从固定白名单解析 mainnet / testnet / demo：

- `binance.testnet=false` 只能选 mainnet；`true` 只能选 testnet/demo；非法值 fail closed。
- 凭证按环境选择：主网凭证、U 本位测试网凭证、独立 Demo 环境变量，无跨环境回落。
- `feature/api/binance/index.go:176-215` 在包初始化时构造 `deliveryClient`，`newCoinMRESTGuard` 拒绝非当前主机和非 `/dapi/`、`/futures/data/` 路径；`http.Client` 重定向同样经过 guard。
- `coin_m_environment.go:108-129` 校验 SDK 全局 WS 模式与选定 REST 环境一致；私有读 `134-145` 在 setup 失败、client 为空、凭证缺失或端点不匹配时拒绝。

探针 `TestReviewCM12EnvironmentFailClosedProbe` / `TestReviewCM12RESTGuardProbe` / `TestReviewCM12PrivateAndWSReadinessProbe` 用假 Transport 验证：合法三环境选择正确；四类非法组合被拒；FAPI / Spot / SAPI / 跨主机请求 0 次出网；setup 失败时公开读也拒绝；无凭证私有读拒绝；市场 WS 未收到有效 ticker 时 `DeliveryMarketWSFresh()=false`。**PASS**。

### 4.2 DAPI 只读与 Weight 隔离

- `feature/api/binance/delivery.go:24-52` 仅暴露账户、余额、持仓、持仓模式四个签名 GET，全部前置 `coinMPrivateReadiness`；没有 CM 下单、撤单、改模式接口。
- `58-72` ExchangeInfo 为公开 GET，失败时返回错误；`75-99` 按真实 `environment` 写入 `delivery` 产品的 weight / order limits，不再硬编码 `mainnet`。
- `service/binanceapiusage/coin_m_cm2_test.go` 证明 Demo 418 与 Testnet 429 分别计入 `delivery/demo` 与 `delivery/testnet`，不互相污染。
- 全仓 Grep：`dapi/v1/order` 仅出现在 guard 的负向测试 URL，无真实调用。**PASS**。

### 4.3 合约元数据与精度

- `feature/delivery.go:100-149` 只接受 `PERPETUAL + TRADING + quoteAsset=USD + _PERP`，要求 `contractSize`、`marginAsset` 和 PRICE_FILTER / LOT_SIZE / MARKET_LOT_SIZE 完整。
- `154-193` 用 `math/big.Rat` 校验张数 step/min/max 与 LIMIT 价格 tick/min/max；market 单走 MARKET_LOT_SIZE；所有校验为纯函数。
- `199-260` 先完整解析/校验整份 exchangeInfo，再在单事务内写库；新币 `enable=0`、`budget_usd` 默认 0；已有币只改元数据，不碰 `enable/usdt/strategy/technology/leverage`。

探针 `TestReviewCM12MetadataRefreshDoesNotTouchOperatorConfigProbe`：已有币 `enable=1, budget_usd=1.25, usdt=10, strategy/technology/leverage` 刷新后保持不变；新币默认 OFF / 0；季度合约不入库。已有 `TestCM2ExchangeInfoRefreshPreservesLocalTradingConfiguration` 还证明坏元数据在任何写库前被拒。**PASS**。

### 4.4 市场 WS 与生产接线

- `feature/api/binance/coin_m_ticker.go:15-44` 只 UPDATE 已知 `contract_type='PERPETUAL'` 的行情列，SQL 不含 `enable / budget_usd / strategy / technology`；无效价格、非 PERP、未知 symbol 均跳过。
- `feature/api/binance/delivery.go:115-154` 使用独立市场 WS 循环，5 秒重连；`DeliveryMarketWSFresh` 明确注释不能替代私有用户流授权。
- `main.go:417-431` 在既有 12 小时任务内加入 `UpdateDeliverySymbolsTradePrecision()`，并启动一条 `UpdateDeliveryCoinByWs()`；没有创建第二套调度中心。
- 静态检查：CM-2 没有 DAPI listenKey/User Data WS，也没有把市场 WS 当成交证据。**PASS（CM-2 范围内）**。

## 五、离线 Gate 实测

| Gate | 结果 |
| --- | --- |
| 用户工作区 `git diff --check` | PASS（无输出，EXIT=0） |
| 隔离副本 722 Go 文件 SHA256 | PASS（0 diff / 0 missing） |
| `go vet ./...` | PASS（EXIT=0） |
| `go build -o /tmp/go_binance_futures-cm12-audit .` | PASS（EXIT=0） |
| `go test -count=1 -p=2 ./...` | PASS：73 ok + 13 no test files + 0 FAIL（86 包） |
| `go test -race -count=1 -p=2` 五包 | PASS：command / controllers / feature / feature/api/binance / service/binanceapiusage；macOS `LC_DYSYMTAB` linker warning 非致命 |
| `go test -count=5 -p=2 -run 'CM2'` 三包 | PASS：feature / feature/api/binance / service/binanceapiusage |
| `go test -count=5 -p=2 -run 'CM1\|CM2'` 五包 | **FAIL**：CM-1 控制器持久化测试第 2~5 次失败 |
| `go test -count=5 -p=2` 五包全量 | **FAIL**：除上项外，既有 Spot Quant 控制器测试同样因固定 alias 失败 |
| 三组新增隔离探针 | PASS：环境 fail-closed / v27 fail-closed / 元数据与开关隔离 |
| 真实网络 / 实单 opt-in | 未运行；环境变量已清空，标准回归不联网 |

## 六、Findings

### F1 · P2 · 固定 ORM alias 使 `-count>1` 不可重复

`controllers/index_coin_m_config_test.go:75-78` 每次调用 `TestCM1GlobalTogglePersistOnlyOneColumn` 都执行：

```go
const alias = "controller_cm1_config_patch"
if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "cm1.db")); err != nil {
    t.Fatal(err)
}
```

`-count=5` 时同一进程内第二次起即报：

```text
[ORM] DataBase alias name `controller_cm1_config_patch` already registered, cannot reuse
--- FAIL: TestCM1GlobalTogglePersistOnlyOneColumn
```

同包既有 `index_spot_quant_review_test.go` 的 `controller_spot_quant_patch_test` 有同样问题。影响范围是**测试可重复性与报告准确性**，不是 CM-1 业务代码运行时行为；`count=1`、race 与全量测试均通过。建议改为 `sync.Once` + 每次清空表，或按 `t.Name()`/随机数生成唯一 alias，并同步修正 CM-2 报告中的 `-count=5` 表述。修复前不得宣称「重复运行全绿」。

### F2 · P3 · 全局开关 OFF 仍启动公开行情链路

`main.go:423,431` 无条件启动 12 小时 DAPI exchangeInfo 刷新和 COIN-M 市场 WS，未读取 `config.coin_m_quant_enable` 或任何每币开关。CM-2 阶段没有交易执行器，因此这不是「OFF 仍下单」；但它确实让默认关闭的模块持续访问外部 DAPI 并写 `delivery_symbols` 行情。方案 §3.4.3 只承诺停止「不必要的高频账户请求」，当前没有账户轮询，所以不构成契约违背。建议 CM-3/README 明确写出「OFF 仍同步公开行情、仍占用 DAPI 权重」，并给出完全停止的运维方式。

### F3 · P3 · 公开 REST 外部证据不可由本次审计独立复核

`CODE_REVIEW_CM-2.md` 记载 2026-11-17 公共 `https://dapi.binance.com/dapi/v1/exchangeInfo` HTTP 200、30 合约、20 个有效 USD 永续、`BTCUSD_PERP contractSize=100` 等。本次审计遵守「探针不联网」边界，未重放该请求，仓库内也未见脱敏原始响应文件。因此该结论仍视为实施者声明，不能升级为独立验收证据。CM-5 前建议保留响应级脱敏证据（时间、URL、HTTP 状态、关键字段）。

### F4 · P3 · MySQL 索引检查未按表限定（低风险观察）

`command/coin_m_migration.go:103-105` 在 MySQL 下用 `information_schema.statistics` 仅按 `schema + index_name` 判断索引存在，未带 `table_name`；`coinMUniqueIndexExists` 则带了表名。当前索引名均有 `coin_m_` 前缀，冲突概率低；但严格来说，同名索引落在别表会使 `coinMIndexExists` 误判。可在生产 v27 检查单中增加按表核对。

## 七、后续 Gate（仍 BLOCKED）

1. **先修 F1 或收窄声明**，再重新执行 `-count=5` 并留证。
2. **CM-3 准入**：UI 必须只有一个配置中心全局开关 + 每币列表 `enable`；不得新增第三开关、审批状态机或 CM 订单写页面；页面文案须区分合约张数、USD 预算与保证金资产。
3. **真实数据库**：生产 MySQL 8 先备份，再在维护窗口人工 `sync db` 到 v27；执行只读检查单（2 开关 / `delivery_symbols` 新列 / 4 表 / DECIMAL / 唯一索引 / 旧数据不变），并做 ARM Linux 二进制 + static + schema 成套发布与回滚演练。
4. **真实交易所**：Demo/Testnet 签名账户 GET、DAPI User Data WS、真实成交、429/418、长稳、多 `marginAsset`、Algo 条件单均未验证；市场 WS 不得替代私有流授权。
5. **实盘授权**：CM-5 之前不得以任何方式产生真实 CM 订单；之后仍需独立书面授权、环境前缀与 append-only 审计，禁止靠翻业务开关越过环境隔离。

## 八、最终判定

**CM-1 / CM-2 的代码与离线证据支持进入 CM-3 的前端骨架与账本页面开发；不认可「已可上线」或「已完成实盘验收」。**

F1 是本次唯一需要尽快修正的可重复性问题；F2~F4 为边界/证据改进项。既有两份 CM 报告的阶段方向判断成立，但 CM-2 的重复测试声明必须收窄。生产与真实交易 Gate 保持全部 BLOCKED。

---

## 九、复审后修复追踪（2026-11-17，补充）

以下内容为本报告原始审查完成后追加的代码修复结果；**保留原 Finding 描述作为历史审计事实**，以此处状态为准。

| Finding | 处理状态 | 已执行变更与验证 |
| --- | --- | --- |
| **F1 / P2** | **已修复** | `controllers/index_coin_m_config_test.go` 与 `index_spot_quant_review_test.go` 使用原子序列生成唯一 ORM alias，每轮仍使用自己的 `t.TempDir()` SQLite 文件；重复 `-count=5` 不再出现已注册 alias 冲突。 |
| **F2 / P3** | **接受现有设计，文档说明** | 已在主开发方案 §3.4.3 和 `CODE_REVIEW_CM-2.md` 明确：**两个交易开关仅限制新开仓，不限制公开 ExchangeInfo/行情 WS 的运行**；OFF 期间仍有公开流量和行情列更新，但无签名账户轮询或下单。不新增第三开关。 |
| **F3 / P3** | **缓存证据补齐，二次在线复核仍未完成** | 本机留存的 CM-2 首次公开响应 JSON 已核对有效字段和 SHA256，并生成 [CM-2 公共 DAPI 响应证据](CM-2-公开DAPI响应证据.md)。首次请求 HTTP 200 已有实施时终端记录；此次尝试重新访问因 TLS 连接故障未成功，**不得写成独立二次实时联网验收**。 |
| **F4 / P3** | **已修复** | `command/coin_m_migration.go` 的 `coinMIndexExists(o,table,name)` 在 MySQL、PostgreSQL、SQLite 均增加对应表名过滤；v27 索引建立和验证全部传入正确表名。另加「同名索引不能匹配错误表」回归测试。 |

**验证范围与生产限制**：F1、F4 修复均只触及本地代码、测试与文档；未更改 v27 表定义和交易逻辑，没有运行数据库迁移或实盘下单。真实 MySQL 8、ARM Linux 部署、COIN-M 私有 WS 与订单链路仍遵循原报告的阻断条件。

### 修复后的实际测试 Gate

| Gate | 本次结果 |
| --- | --- |
| `go test -count=5 -p=2 ./command ./controllers ./feature ./feature/api/binance ./service/binanceapiusage` | **PASS**：5/5 包，复现并消除了原 F1 失败 |
| `go test ./command -run '^TestSyncDatabaseInitializesAndIsIdempotent$' -count=5 -p=2` | **PASS**：包括新增的错误表名索引测试（F4） |
| `go test -count=1 -p=2`（排除 `strategy_templates/research` 的 86 个包） | **PASS**：与原审计隔离副本 86 包范围一致 |
| `go vet`（同一 86 包） | **PASS** |
| `go test -race -count=1 -p=2`（上述 5 个关键包） | **PASS**；macOS linker `LC_DYSYMTAB` 警告不影响退出码 |
| `go build -o /tmp/go_binance_futures_cm12_review_fixed .` | **PASS**；仅生成 `/tmp` 二进制，没有替换运行程序 |
| `git diff --check` | **PASS** |
| 当前工作目录原样 `go test ./...` | **FAIL（非 CM 回归）**：包含 337 个包，其中历史 `strategy_templates/research` 下有 8 个包因重复函数定义无法编译；这些研究文件没有被本次修改，未私自清理。 |
| 本机再次访问公共 DAPI | **未成功**：直接 TLS 超时，代理 TLS 连接失败；仅核验先前 HTTP 200 的缓存响应哈希与字段 |

**完整目录与原报告的差异**：本机 `go list ./...` 返回 337 包，排除历史 `strategy_templates/research` 后是 86 包；原报告在隔离副本上使用 86 包，因此不能把本机 `go test ./...` 的失败说成 CM 代码不通过，也不能宣称本机完整 337 包全绿。与业务相关的 86 包和 Race Gate 已通过。

**最终裁定（代码/离线范围）**：F1 和 F4 已修复；F2 按既定两级交易开关设计处理并完善文档；F3 已保留原始缓存摘要，但独立二次实时访问证据仍未取得。**允许进入 CM-3 开发，不代表已具备生产部署或真实订单执行条件**。
