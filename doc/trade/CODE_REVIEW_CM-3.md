# CODE_REVIEW_CM-3 · 币本位配置、策略模板与四页面骨架

> 检查日期：2026-11-18（北京时间，开发从 11 月 17 日延续至 18 日）。分支 `feat/m-coin-futures`，源码数据库 Schema 为 v27。
>
> **本阶段未连接 DAPI 下单/撤单、未开启策略自动调度、未修改运行数据库或 app.conf、未重启服务。**

## 一、范围与实现

| 项目 | 实现位置 | 结果 |
| --- | --- | --- |
| 新顶级「币本位交易」菜单 | 前端 `src/router/modules/coinM.ts` | **完成**；rank 2.05，置于 U 本位合约 rank 2 与现货 rank 2.1 之间 |
| 四个子菜单 | `src/views/coinM/{symbol,account,strategyTemplate,controlledTrade}.vue` | **完成**；独立入口及页面骨架 |
| 配置中心全局开关 | `src/views/dashboard/configShow.vue` | **完成**；仅新增一个 `coin_m_quant_enable`，默认关闭；CM-1 后端独立处理 |
| 每币启停 | `service/coinmquant/repository.go`、`controllers/coin_m_quant.go` | **完成**；只使用现有 `delivery_symbols.enable`，没有新币种开关或第三个总开关 |
| 永续合约列表与搜索 | `service/coinmquant/repository.go`、`/coin-m/symbols` | **完成**；只读 `contract_type=PERPETUAL`、收藏、模糊搜索、排序、分页 |
| 币本位预算 | `delivery_symbols.budget_usd`，独立 Raw SQL DECIMAL | **完成**；不复用旧 `usdt` 字段，UI 明确写「USD 等值预算」 |
| 单币交易参数 | `/coin-m/symbols/:id` | **完成**；杠杆、逐全仓、止盈止损、指标及规则仅写本地配置，不调用 Binance 调仓 API |
| 策略编辑与模板 | `src/views/coinM/{symbol,strategyTemplate}.vue`、`coin_m_strategy_templates` | **完成**；LONG/SHORT/CLOSE_LONG/CLOSE_SHORT，独立模板 CRUD、导入、导出、syntax_only 校验 |
| 批量配置 | `/coin-m/symbols/batch` | **完成**；仅选中 IDs，先整体验证，再在一个本地事务内更新 |
| 币本位本地账户 | `src/views/coinM/account.vue`、`service/coinmquant/ledger_read.go` | **只读骨架完成**；Managed Position/Order/Fill 的列表、筛选、分页与详情 |
| 仓位与受控交易 | `src/views/coinM/controlledTrade.vue` | **只读骨架完成**；复用本地账本视图，明确没有下单、撤单、手动认领或真实平仓 |
| 后端路由 | `routers/router.go`、`controllers/coin_m_quant*.go` | **完成**；所有 CM-3 接口使用 `/coin-m/` 路径，JWT 用户验证 |
| 前端 API 隔离 | `src/api/coinM.ts` | **完成**；独立路由，不调用现货/USDT-M 的配置和订单接口 |
| 多语言 | `locales/zh-CN.yaml`、`locales/en.yaml` | **完成**；菜单、开关、合约张数、保证金币种、模板和本地账户提示 |

## 二、代码边界与交易安全

1. **配置中心只有一个新开关**，即 `coin_m_quant_enable`；每币开关为历史模型已有的 `delivery_symbols.enable`。二者都不能单独导致真实订单发送：CM-5/CM-6 尚未实现执行器。
2. **禁止跨产品误写**：币种设置只更新 `delivery_symbols` 的白名单配置列，模板只写 `coin_m_strategy_templates`；仓位/订单/Fill 只读取 `coin_m_managed_*` 与 `coin_m_trade_fills`。现货、U 本位及 Lead 账户表未被使用。
3. **预算精度**：后台 `budget_usd` 校验十进制非负数，最多 18 位整数和 18 位小数；SQL `DECIMAL(36,18)` 仍由 CM-1 v27 保证。当前使用 SQLite 做接口单测，其 NUMERIC 亲和性不等同于 MySQL 严格 DECIMAL 精度验收。
4. **启用前校验**：首次由页面启用单币要求正预算、至少一条已启用的 LONG 或 SHORT 开仓规则和已识别合约面值与保证金币种；关闭币种不强平、不改变已有持仓。批量启用只选择已配置币种。
5. **管理界面不是实盘控制台**：账户只展示本地 Managed 周期、订单和 Fill，不把本地记录认作真实钱包余额；不合并不同保证金币种的 PnL，也不编造未实现盈亏。反向 PnL 与按结算资产汇总留在 CM-4。
6. **没有新增交易订单写接口**：本阶段仅新增配置/模板的写接口和账本读接口，没有 `delivery.NewCreateOrderService`、Algo、撤单、仓位模式修改或真实订单同步。
7. **开发/部署边界**：没有执行 `./go_binance_futures sync db`，没有修改 `conf/app.conf`，没有覆盖正在运行的后端可执行文件。前端依既定流程 `pnpm build` 产生 dist 并同步至后端 `static/`，不代表服务器已经发布。

## 三、本地回归测试

| 验证 | 实际结果 |
| --- | --- |
| `go test ./service/coinmquant -run '^TestCM3' -count=5 -p=2` | **PASS**：逐币配置、数量/预算/合约隔离、完整四方向策略、模板 CRUD、批量编辑与本地读模型 |
| 原 U 本位/现货/Lead 与 CM 相关包测试（排除历史 research） | **PASS**：87 个非历史研究 Go 包，`-count=1 -p=2` |
| `go vet ./controllers ./service/coinmquant ./routers ./command` | **PASS** |
| `go build -o /tmp/go_binance_futures_cm3_v27 .` | **PASS**：独立编译产物，不替换应用可执行文件 |
| 前端 `pnpm typecheck` | **PASS** |
| 前端 `pnpm build` | **PASS** |
| 前端 `dist/` → 后端 `static/` | **PASS**：rsync 成功，`index.html` 核对一致 |
| `git diff --check` | **PASS** |
| `go test -race -count=2 -p=2 ./service/coinmquant ./controllers` | **PASS**；macOS linker 的 `LC_DYSYMTAB` 警告不影响运行与退出码 |

**测试范围说明**：项目包含大量历史 `strategy_templates/research/` 独立研究程序，部分存在重复函数声明；本项目此前的 CM-1/CM-2 联合审计确认非研究包为 86 个，CM-3 新增 `service/coinmquant` 后为 87 个。上述测试不表示全部历史研究实验程序均能编译。

## 四、后续开发边界

- **CM-4**：完善 Position/Order/Fill 的准确真实周期汇总；增加 inverse PnL、未实现盈亏、按 marginAsset 分组统计与钱包只读展示。CM-3 的本地列表不应被宣传为完成这些核算。
- **CM-5**：签名私有 WS 与 DAPI Fill 查询；LONG/SHORT/CLOSE、条件单、异常恢复及 Testnet。当前这些功能都未实现或验收。
- **CM-6**：复用 2 秒调度，把全局＋每币开关连接到策略执行，并保证关闭开仓后仍能处理退出。
- **CM-7**：完整 UI、README、端到端联调与发布前回归；ARM Linux 与真实 MySQL 8 的 v27 迁移及生产投产仍需另外验收。

## 五、结论

**CM-3 在配置与页面骨架范围内完成，并通过本地构建及离线 Go 测试，具备进入 CM-4 的研发条件。** 没有对真实币本位账户执行任何交易，也未迁移实际数据库。未完成的真实交易及收益口径严格留在后续阶段。
