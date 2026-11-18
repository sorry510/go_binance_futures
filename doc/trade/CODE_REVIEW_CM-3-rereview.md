# CODE_REVIEW_CM-3-rereview · 币本位配置、模板、四页面骨架与本地账本只读 API 复审

> 日期：2026-11-18（北京时间）
>
> 分支：`feat/m-coin-futures`；HEAD `1d5be90 feat: cm-2`；父提交 `72cf42a feat: cm-1`；master `0b3fa34 (tag v2.2.3)`。
>
> 本报告是**新增复审**，不覆盖 `doc/trade/CODE_REVIEW_CM-3.md`。本次审查对象为当前工作区未提交的 CM-3 改动，后端共 166 项状态变更（含静态资源重建），前端仓库 11 项状态变更。
>
> 审计期间未修改业务源码、未提交代码、未修改 `conf/app.conf`、未访问真实 Binance、未执行 `sync db`、未重启服务、未触碰真实仓位。所有临时探针均写在隔离副本中，审计结束删除。

---

## 一、复审结论

| 项目 | 结论 |
| --- | --- |
| CM-3 方案契约（币种列表启停 / 独立模板 CRUD / 配置中心单开关 / 四页面骨架 / 本地账本只读 API） | ✅ 通过 |
| 后端 `/coin-m/` 路由、控制器、服务层与账本读接口隔离 | ✅ 通过 |
| 前端独立菜单、独立 API、四页面骨架、i18n、`dist` → `static` 一致性 | ✅ 通过（按骨架阶段通过） |
| 后端离线 Gate（diff-check / vet / build / 全量测试 / 重复测试 / race） | ✅ 通过 |
| 前端离线 Gate（typecheck / build / 产物一致性） | ✅ 通过 |
| 隔离探针（JWT 越权、跨产品写入、字段白名单、规则方向、ledger 注入） | ✅ 通过 |
| 前轮 CM-1/CM-2 F1：固定 ORM alias 导致 `-count>1` 失败 | ✅ 已闭环 |
| 真实 DAPI / 签名 User WS / 真实成交 / MySQL 8 v27 / 生产发布 / 回滚 | ⛔ 全部 BLOCKED |
| 阶段意见 | **✅ 有条件通过：可作为进入 CM-4 的研发基线；不得表述为最终页面、实盘或上线完成** |

**核心判定：** CM-3 自评报告《CODE_REVIEW_CM-3.md》中「在配置与页面骨架范围内完成，并通过本地构建及离线 Go 测试，具备进入 CM-4 的研发条件」这一结论，在本次独立复审中成立。

但必须保留三个边界：

1. 本次通过的是**CM-3 的骨架与只读账本契约**，不是方案 §五中 CM-4 才要求的完整 inverse PnL、未实现盈亏、按结算资产汇总、真实钱包展示。
2. 「仓位与受控交易」页面仍是只读骨架；真实下单、撤单、认领、手动平仓和 DAPI 对账均留待 CM-5，本阶段不得宣传为已开放受控交易。
3. 所有离线、隔离副本、SQLite、Mock 和构建通过，**均不构成真实 Testnet / Demo / Mainnet、MySQL 8 迁移、ARM Linux 发布、回滚或生产授权 Gate 通过**。

本次未发现 P0 / P1 级阻塞代码缺陷；列出 3 项非阻塞观察项，详见第五节。

---

## 二、基线与审计方法

### 2.1 基线锁定

```text
branch        feat/m-coin-futures
HEAD          1d5be90 feat: cm-2
parent        72cf42a feat: cm-1
master        0b3fa34 tag v2.2.3
audit date    2026-11-18
worktree      166 项状态变更（含 static 重建）
frontend      11 项状态变更
```

本次 CM-3 主要改动：

- 后端新增：`controllers/coin_m_quant.go`、`controllers/coin_m_quant_read.go`、`service/coinmquant/*`；
- 后端修改：`routers/router.go`、`models/coin_m_quant.go`；
- 文档：主方案状态更新为 CM-0～CM-3 已完成；新增实施方自评 `doc/trade/CODE_REVIEW_CM-3.md`；
- 静态资源：前端重建后大量旧 hashed JS/CSS 删除、新 hashed JS/CSS 新增、`static/index.html` 更新；
- 前端新增 CM-3 页面、API 与菜单，同时夹带少量非 CM-3 改动（见 O1）。

### 2.2 审计输入

已阅读并交叉核对：

- `doc/trade/开发方案-CM-币本位合约交易-复刻现货精简版.md`，重点是 §五页面与 §六 CM-3 完成标准；
- `doc/trade/CODE_REVIEW_CM-3.md`；
- `doc/trade/CODE_REVIEW_CM-1_CM-2.md`；
- `doc/trade/CM-2-公开DAPI响应证据.md`；
- 后端控制器、服务、路由、DTO、账本读模型；
- 前端路由、API、四个 `coinM` 页面、配置中心与多语言文件。

### 2.3 隔离副本与等价校验

为避免污染用户工作区，使用 `git archive HEAD` 展开 HEAD，再按 `git status --porcelain=v1 -z` 同步修改、新增和删除；未跟踪目录按文件清单展开。等价校验使用逐文件 SHA256，避免全仓 `os.walk` / `grep -r` 触发历史 research 目录的 SIGTERM(137)。

后端副本：

```text
copy          /tmp/phase_audit_cm3
SYNCED_CHANGES=166
FILES_CHECKED=1190
GO_FILES=728
STATIC_FILES=195
SHA256_MATCHED=1111
SHA256_DIFF=0
SHA256_MISSING=0
DELETED_CONFIRMED=79
APP_CONF_PRESENT=True
```

前端副本：

```text
copy          /tmp/phase_audit_cm3_ui
SYNCED_CHANGES=11
FILES_CHECKED=14
SHA256_MATCHED=14
SHA256_DIFF=0
```

`conf/app.conf` 不来自用户真实配置，副本内使用 `app.conf.example` 兜底；不把任何真实凭证带入 Gate。

---

## 三、CM-3 完成标准逐项对照

方案 §六对 CM-3 的完成标准为：

> 币种列表启停、独立模板 CRUD、配置中心单开关、前端四页面骨架与本地账本只读 API。

### 3.1 币种列表启停：PASS

证据：`service/coinmquant/repository.go`、`controllers/coin_m_quant.go`、`routers/router.go`。

已核对：

- `/coin-m/symbols` 读取 COIN-M 币种列表，只使用 `delivery_symbols` 的现有 `enable`，未新增每币开关字段；
- 预算使用 `budget_usd`，未复用 U 本位旧 `usdt` 字段；
- patch 采用白名单，拒绝 `usdt`、`contract_size`、`margin_asset`、`exchange_order_id`、`position_id` 等不属于本阶段配置的字段；
- 单币启用要求正预算、已启用的 LONG 或 SHORT 规则、合约元数据可用；关闭币种不等于强平；
- 批量更新先整体校验，再单事务写入；
- 市场行情只用于展示和合约列表，不改变 `enable`、预算、策略或交易参数。

**判断：** 与 CM-3 契约一致，未越界到真实交易参数修改。

### 3.2 独立模板 CRUD：PASS

证据：`service/coinmquant/repository.go`、`controllers/coin_m_quant.go`、`models/coin_m_quant.go`。

已核对：

- 模板写入独立的 `coin_m_strategy_templates`，未复用 U 本位 / 现货模板表；
- 规则方向仅允许 `long`、`short`、`close_long`、`close_short`，spot 风格或未知方向被拒绝；
- `RuleTest` 仅语法校验，不执行回测、不下单、不产生真实订单；
- 模板 CRUD 全部走 `/coin-m/` 路由，并有 JWT 用户校验。

**判断：** 模板数据隔离和规则边界符合 CM-3 要求；完整策略语义验证留 CM-4/CM-5，不在本阶段扩大解释。

### 3.3 配置中心单开关：PASS

证据：`src/views/dashboard/configShow.vue`、`models/tableStruct.go`、`controllers/index.go`。

已核对：

- 配置中心新增独立「币本位合约交易」折叠卡片；
- 卡片内只有一个 `coin_m_quant_enable` 开关；
- 默认关闭；保存调用 `saveField('coin_m_quant_enable', value)`；
- 后端仍保持 CM-1 的单字段 0/1 patch 与持久化校验；
- 每币启停只在币本位交易列表配置，配置中心不重复展示。

静态扫描结果：

```text
CONFIG_COIN_M_SWITCH_COUNT=1
CONFIG_COIN_M_SWITCH_HAS_SAVE=1
```

**判断：** 未发现第二/第三个全局开关；与「唯一全局开关」契约一致。

### 3.4 前端四页面骨架：PASS（按骨架阶段）

证据：`src/router/modules/coinM.ts`、`src/api/coinM.ts`、`src/views/coinM/*`、`locales/zh-CN.yaml`、`locales/en.yaml`。

已核对：

- 新增 `/coin-m` 顶级菜单，`rank=2.05`，位于 U 本位合约和现货之间；
- 四个子页面齐全：
  - `/coin-m/symbols`
  - `/coin-m/account`
  - `/coin-m/strategy-templates`
  - `/coin-m/controlled-trade`
- 前端 API 模块独立使用 `coin-m/...` 前缀，未调用 Spot / FAPI 配置和订单接口；
- 文案明确区分：
  - 「只配置币本位策略，不会自动下单」；
  - 「CM-3 本地账本只读预览」；
  - 「真实下单、撤单、平仓和归属操作在 CM-5 才开放」；
  - 合约张数、USD 等值预算、保证金币种表述正确，不写「投入 USDT」。

阶段边界：

- 方案 §五第 2 点的完整钱包余额、参考 USD 换算、筛选后按结算资产总计、inverse PnL、未实现盈亏属于 CM-4；
- 方案 §五第 4 点「必要的手动受控操作」在 CM-3 中只做骨架；本页复用只读账本视图，不提供交易按钮。

**判断：** 四页面骨架完成；不得把 CM-3 页面骨架当作方案 §五最终页面能力已完成。

### 3.5 本地账本只读 API：PASS

证据：`controllers/coin_m_quant_read.go`、`service/coinmquant/ledger_read.go`。

已核对：

- 只提供 `ListPositions`、`ListOrders`、`GetPositionDetail`；
- 注释和实现均保持 read model，不做钱包查询、不做订单/成交对账写回、不调用 Binance 交易 API；
- 详情中可读取本地 Fill，但没有自行计算未实现盈亏、不合并不同 `marginAsset` 的 PnL；
- symbol/status 过滤器拒绝 `%`、空格与注入串，避免模糊匹配和 SQL 注入。

**判断：** 只读账本 API 符合 CM-3；完整周期汇总和 PnL 口径留 CM-4。

---

## 四、安全边界与交易隔离

### 4.1 路由与认证

后端新增 12 条 `/coin-m/` 路由，全部指向 `CoinMQuantController`。静态扫描：

```text
CM_ROUTES=12
```

所有 handler 首先执行 `auth()`：

- 从 JWT 用户 claim 中取 `Username`；
- 要求 claim 存在、非空且类型为 `string`；
- 不满足则返回 401 `coin_m_auth_required`。

隔离探针覆盖了控制器的 15 个 handler，在无 JWT claims 的直接请求下均返回：

```text
http=401
body contains coin_m_auth_required
```

**PASS。**

### 4.2 无真实下单入口

静态扫描：

```text
CM_LIVE_ORDER_TOKENS=0
CM_CROSS_PRODUCT_API_IMPORTS=0
```

已核对：

- 未出现 delivery SDK 的 create order、cancel、Algo、改 dualSidePosition 等真实交易调用；
- 前端 `src/api/coinM.ts` 未调用 FAPI、Spot、U 本位订单接口；
- CM-3 页面没有下单、撤单、手动认领、手动平仓按钮或 API；
- 币种配置与模板写入只作用于 `delivery_symbols` 白名单列和 `coin_m_strategy_templates`；
- 账本读接口只读取 `coin_m_managed_*` / `coin_m_trade_fills`，不写现货、U 本位或 Lead 账本。

**判断：** 跨产品污染和提前解锁真实交易的风险在 CM-3 代码范围内被有效阻断。

### 4.3 DTO 与 DECIMAL 隔离

`models/coin_m_quant.go` 当前给 DTO 字段补充了 `orm:"column(...)"` tag，但这些 DTO 没有注册到 Beego ORM；原注释仍明确要求不要注册，避免 string 字段被映射为 VARCHAR 而破坏 `DECIMAL(36,18)` 语义。

运行时读写仍走 Raw SQL / 显式字段白名单。SQLite 单测只能验证逻辑，不等同于 MySQL 8 严格 DECIMAL 精度验收。

**判断：** 当前不构成缺陷；列见 O2。

---

## 五、Findings / 观察项

本次未发现 P0 / P1 级阻塞问题。以下为不影响进入 CM-4、但建议记录或收窄声明的观察项。

### O1 · P2 · 前端 diff 夹带非 CM-3 改动

观察到前端工作区除 CM-3 页面/API/菜单外，还有与本阶段无直接关系的改动：

- `src/api/leadTrading.ts`
- `src/views/futures/leadTrading.vue`
- `src/router/modules/futures.ts`（lead 菜单 `showLink: false`）
- `src/views/login/index.vue`（429 登录限速提示）

这些改动不影响 CM-3 的后端契约、隔离 Gate 或 `/coin-m` 页面通过判定，但会造成阶段归属不清和回滚边界模糊。

建议：提交 CM-3 时拆分变更，或在提交说明中逐项解释为何同批发布；后续复审不得把这些改动默认归入 CM-3 验收。

### O2 · P3 · DTO `orm` tag 可能误导后续维护

`models/coin_m_quant.go` 中 DTO 字段新增 `orm:"column(...)"`，但这些 DTO 未注册 ORM，且代码注释要求保持不注册。当前运行行为正确，但 tag 容易被后续维护者误解为可注册 ORM，从而引入 VARCHAR / DECIMAL 精度漂移。

建议：移除无实际用途的 `orm` tag，或在注释中明确「tag 仅用于文档，不得调用 `orm.RegisterModel`」。

### O3 · P2 · CM-3 页面能力必须按「骨架 + 只读」表述

方案 §五最终页面包含 inverse PnL、参考 USD、按结算资产筛选汇总、真实 Fill 展开和受控操作；CM-3 实际只完成骨架与本地账本只读。账户详情虽可读取本地 Fill，但没有 CM-4 的完整周期汇总和收益核算；受控交易页没有真实写操作。

建议：CM-3 验收与版本说明中固定表述为「四页面骨架 + 本地账本只读预览」，不得据此提前关闭 CM-4 / CM-5。

---

## 六、离线 Gate 与隔离探针

### 6.1 后端 Gate

执行时清空真实/测试网相关 env，包括 `BINANCE_TESTNET_*`、`SPOT_QUANT_TESTNET_*`、`SPOT_QUANT_VALIDATE_*`、`SPOT_LEDGER_MYSQL_DSN`、`SPOT_STRATEGY_LIVE_TRADING_APPROVAL`、`BINANCE_COIN_M_ENVIRONMENT`、`BINANCE_COIN_M_DEMO_API_KEY`、`BINANCE_COIN_M_DEMO_API_SECRET` 等，确保 Gate 不联网、不下单。

```text
git diff --check                                          PASS
go vet ./...                                              PASS
go build -o /tmp/go_binance_futures_cm3_audit .           PASS
go test -count=1 -p=2 ./...                               PASS
    87 packages: 74 ok + 13 no test + 0 FAIL
go test -count=5 -p=2 -run '^TestCM3' ./service/coinmquant PASS
go test -count=5 -p=2 ./service/coinmquant ./controllers  PASS
go test -race -count=2 -p=2 ./service/coinmquant ./controllers PASS
```

说明：

- 全量测试排除了历史 `strategy_templates/research/**` 独立研究程序；该范围与 CM-1/CM-2 复审一致；
- `-count=5` 包级补充验证确认前轮 CM-1/CM-2 F1（固定 ORM alias 导致重复运行失败）已经闭环；
- `-race` 中出现 macOS linker `LC_DYSYMTAB` warning，历史已确认非致命，退出码为 0，按 PASS 处理。

### 6.2 前端 Gate

```text
pnpm typecheck    PASS
pnpm build        PASS
```

构建输出中的 Browserslist / `caniuse-lite` 过期提示为非致命 warning，不影响退出码。

### 6.3 静态产物一致性

```text
frontend/dist      vs backend/static      DIFF_EXIT=0
rebuilt dist       vs frontend/dist       A=0
rebuilt dist       vs backend/static      B=0
files per tree     120
```

说明：前后端静态资源一致，重建产物与仓库中的 `dist` / `static` 均一致；但这只证明本地构建产物一致，不代表服务器已发布。

### 6.4 隔离探针

探针均写在 `/tmp/phase_audit_cm3` 隔离副本中，审计结束时删除。

探针一：`controllers/review_cm3_probe_test.go`

- 直接构造无 JWT claims 的请求，遍历 15 个 `CoinMQuantController` handler；
- 断言全部 401，且错误体包含 `coin_m_auth_required`；
- 结果：`ok go_binance_futures/controllers`。

探针二：`service/coinmquant/review_cm3_probe_test.go`

- 验证 Spot / USDT-M sentinel 不被 CM patch 修改；
- 验证白名单拒绝 `usdt`、`contract_size`、`margin_asset`、`exchange_order_id`、`position_id`；
- 验证四方向规则接受、spot 规则拒绝；
- 验证 ledger 的 symbol/status 过滤阻止 `%` 与 `OPEN OR 1=1` 注入；
- 结果：`ok go_binance_futures/service/coinmquant`。

---

## 七、未通过 / 仍 BLOCKED 的 Gate

以下项目本次**没有执行**，且不得因 CM-3 离线通过而视为完成：

1. 真实 MySQL 8 上执行 v27 迁移、备份、锁表时间、索引、字符集、重复升级、回滚演练；
2. 真实 Binance COIN-M DAPI 签名私有端点读取，包括账户、余额、持仓、持仓模式；
3. 签名 User Data WS、断线重连、事件去重、时序与恢复；
4. Testnet / Demo 下的真实开仓、平仓、部分成交、重试、撤单、Algo、429/418；
5. Mainnet 小额实盘；
6. ARM Linux + 静态资源 + schema 成套发布、灰度、监控和回滚；
7. 持久化部署授权、撤销与审计链；
8. CM-4 完整 inverse PnL、未实现盈亏、按 `marginAsset` 分类汇总、钱包只读展示；
9. CM-5 真实 DAPI 交易链路、CM-6 调度接入、CM-7 端到端发布前检查。

---

## 八、最终判定

**CM-3 有条件通过。**

可以作为进入 CM-4 的研发基线，原因是：

- 方案 §六 CM-3 完成标准逐项满足；
- 后端路由、控制器、服务层和账本读接口边界清楚；
- 前端页面骨架、独立 API、多语言和静态产物一致；
- 离线 Gate、重复测试、race、静态扫描和两个隔离探针均通过；
- 前轮 CM-1/CM-2 F1 固定 alias 问题已闭环。

同时明确不认可以下表述：

- 「CM 页面已经最终完成」；
- 「本地账本只读已完成完整 PnL / 钱包 / 受控交易」；
- 「已通过真实 Testnet / Demo / Mainnet」；
- 「v27 已可在生产 MySQL 8 安全升级」；
- 「可以发布或已具备生产授权」。

**下阶段准入：** 允许进入 CM-4，完成独立持仓、订单、真实 Fill 周期汇总与反向 PnL；在 CM-5 之前，任何真实下单、撤单、改持仓模式、手动认领或仓位写回均不得接线。

---

## 九、复审后处理追踪（2026-11-18）

保留以上复审报告原始 Findings，新增本节记录实施方核实与本地修正结果。

| 项目 | 处理状态 | 本次核实及操作 |
| --- | --- | --- |
| **O1 / P2：前端夹带非 CM-3 的 Lead / 登录改动** | **已确认，作为提交隔离事项保留** | 核对前端 `src/api/leadTrading.ts`、`src/views/futures/leadTrading.vue`、`src/router/modules/futures.ts`、`src/views/futures/symbol.vue`、`src/views/login/index.vue` 的工作区状态。这些改动不属于 CM-3，未被本次修改、暂存或重置；后续 CM-3 提交须单独挑选文件，不能混入其他 AI 正在进行的工作。 |
| **O2 / P3：DTO 的 `orm` 标签可能误导** | **已澄清并补充回归测试；保留标签** | 查看 Beego v2.1.0 的 `client/orm/orm_raw.go`：`Raw.QueryRows` 对**未注册的 struct** 会调用 `parseStructTag(fe.Tag.Get(defaultStructTagName))`，优先读取 `column(...)` 进行字段映射。因而这些标签并非装饰或无实际用途，不能按原建议随意移除。已在 `models/coin_m_quant.go` 顶部标注“仅服务 Raw QueryRows，不得 RegisterModel”；新增 `service/coinmquant/dto_mapping_test.go`，检查五个 CM DTO 的列映射标签完整性。**未改变表结构、SQL DECIMAL 类型或模型注册。** |
| **O3 / P2：CM-3 只属于页面骨架** | **已满足，无须修改业务代码** | `CODE_REVIEW_CM-3.md` 和主开发方案均清楚区分：四页面骨架、本地只读账本属于 CM-3；inverse PnL、账户汇总和钱包属于 CM-4；真实 DAPI 下单、撤单及受控交易属于 CM-5，不提前宣称完成。 |

### 本次补充 Gate

```text
go test -count=5 -p=2 ./service/coinmquant ./controllers          PASS
go test -count=5 -run '^TestCM3RawSQLDTOColumnMapping$' ./service/coinmquant   PASS
go vet ./service/coinmquant ./controllers ./models               PASS
git diff --check                                                 PASS
```

**修正后的结论仍为：CM-3 有条件通过，可进入 CM-4。** O2 的原始观察是合理的维护风险提醒，但“移除 `orm` 标签”不适用于当前 Beego Raw SQL 读模型。O1 的 Git 提交归属边界应在实际提交时处理，不属于修改交易业务逻辑的缺陷。本轮未运行迁移、未修改生产数据库或配置、未操作 Binance 账户、未提交 Git。

**方案边界澄清**：第七节第 7 项提及的「持久化部署授权、撤销与审计链」不属于已约定的个人使用精简 CM 开发范围，不能据此新增审批系统、独立审计链或把它设为 CM-4 的前置条件。继续沿用现有全局交易开关、每币开关、账户隔离和必要风控；真实环境的凭证权限、部署及交易验证仍需按既定阶段检查。
