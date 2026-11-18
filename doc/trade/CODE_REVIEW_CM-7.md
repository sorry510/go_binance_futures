# CODE_REVIEW_CM-7 · COIN-M 四页面、文档、完整离线链路及发布前检查

> 日期：2026-11-18。分支：`feat/m-coin-futures`；基线 HEAD：`63efbe8 feat: cm-6`。
>
> **阶段结论：CM-7 代码、前端构建和离线验收通过；真实交易/数据库迁移/发布 Gate 仍未完成。** 本报告不是实盘验收或上线许可。
>
> 安全约束：未操作 Binance 真正账户、未读取生产余额、未发送签名 DAPI 交易请求、未执行 `sync db`、未更改 `conf/app.conf`、未重启服务或提交 Git。

## 一、CM-7 交付范围

| 验收要求 | 源码与结果 |
| --- | --- |
| 四个 COIN-M 菜单 | `src/router/modules/coinM.ts`，置于 U 本位与现货之间：币种、币本位本地账户、策略模板、仓位与受控交易 |
| 独立币种及策略模板 | `src/views/coinM/symbol.vue`、`strategyTemplate.vue`；独立 `/coin-m/...` API，四方向策略，USD 等值预算与合约张数，字段不复用 Spot/FAPI 交易请求 |
| 独立本地账户页 | `src/views/coinM/account.vue`：周期、订单、Fill 明细、按 `margin_asset` 分资产 PnL 与手续费、可选参考 USD、DAPI 只读钱包。修复订单 `filled_contracts` 错用“剩余张数”标题及不明确的证据状态文字 |
| 仓位与受控交易页 | `src/views/coinM/controlledTrade.vue`：新增本地运行状态、已启用币数、未结算订单及 Managed 周期、错误提示与刷新；复用本地持仓/订单明细。**不暴露手动真实下单、撤单、平仓、归属接口**，因为真实 Testnet 尚未验收 |
| 本地只读状态 API | `service/coinmexecution/status_cm7.go`、`controllers/coin_m_runtime_cm7.go`、`routers/router.go`：`GET /coin-m/runtime-status` 需有效 JWT；查询 `config`、`delivery_symbols` 和本地 CM 账本，**完全不调用 DAPI/Ws/订单写入** |
| 中英文提示准确性 | `locales/zh-CN.yaml` / `en.yaml`：不再错误宣称 CM-6 自动策略“永远不会自动下单”；区分“保存配置不立即下单”、“已部署+开关及门禁允许后可能真实下单”、“页面手动交易操作未开放” |
| 中英文 README | 根 `README.md`、`README.EN.md` 增加独立 COIN-M 入口；新增 `doc/readme/zh-CN/10-币本位合约交易.md` 与 `doc/readme/en/10-coin-m-futures.md`；双语系统概况新增四页面 |
| 端到端 Fake/SQLite | `service/coinmexecution/e2e_cm7_test.go`：Runner → Open → 真实 Fill 幂等 → STOP + TP → 两开关 OFF 后策略退出 → 撤销各自 Algo → CLOSE → Fill → 已平仓汇总；LONG/SHORT 独立子测试 |
| 后端权限及读接口回归 | `status_cm7_test.go`、`controllers/coin_m_runtime_cm7_test.go`：默认本地只读、OFF 后仍显示待恢复订单、无本地 Config 失败关闭、无 JWT 返回 401，不请求交易所 |
| 静态产物同步 | 前端 `pnpm typecheck`、`pnpm build`、`rsync -a --delete dist/ backend/static/`、`diff -qr dist/ backend/static/` 全部通过 |

## 二、离线 E2E 的关键证据

- **两种方向均通过**：BTCUSD_PERP 的 LONG 与 SHORT 各独立建立 Managed 周期，并使用对应 BUY/SELL 开平仓方向；Fake DAPI 只有两个普通交易意图（OPEN 和 CLOSE）。
- **真实 Fill 归属**：Fill 根据 exchange order ID / trade ID 归属到正确订单与周期，重复提交相同 Fill 不会被写入第二次。
- **保护先于退出**：STOP_MARKET、TAKE_PROFIT_MARKET 各自生成唯一 Algo；按照实际 2 秒 tick 与原本 15 秒保护单核对节奏补齐，不绕开 Runner 恢复机制。策略出场前必须将两个已拥有的保护单核实为 CANCELED。
- **两个开仓开关 OFF 也能关闭原有仓位**：关闭仅阻止新建 OPEN；完整 CLOSE Fill 到达后周期为 `CLOSED`、`remaining_contracts=0`，本地总收益证据为 `verified`。
- **退出以后不重复开仓**：后续 tick 继续遵守 OFF 开关，不产生第三笔订单。
- **本地运行摘要没有伪造私有 WS 健康状态**：本地只有开关/账本数量；JSON 中 `private_ws_verified=false`、`wallet_verified=false` 明确表示**未检查**，并不是推断断线或余额为 0。

测试**没有**连接真实 Binance REST/WS，不验证订单提交、Algo 触发的真实网络时序，也不代表数据库在 MySQL 8 上的精确并发行为已经通过。

## 三、执行结果

| Gate | 本机结果 |
| --- | --- |
| `go test -count=5 -p=2 -run '^TestCM7' ./service/coinmexecution ./controllers` | **PASS** |
| 四个核心包 `go test -count=5 -p=2` | **PASS** |
| 除已知独立研究目录 `strategy_templates/research/` 外的全部 **88** 个 Go 包 `go test -count=1 -p=2` | **PASS** |
| 同一 88 包 `go vet` | **PASS** |
| 四核心包 `go test -race -count=2 -p=2` | **PASS**；macOS linker 的 `LC_DYSYMTAB` warning 不影响返回码 |
| `go build -o /tmp/go_binance_futures_cm7_v27 .` | **PASS**；约 40 MB，本机隔离编译产物，未覆盖正在运行的程序 |
| 前端 `pnpm typecheck` | **PASS** |
| 前端 `pnpm build` | **PASS**；构建输出约 4.26 MB |
| `dist → backend/static`，树级 `diff -qr` | **PASS**，差异为 0 |
| 前后端 `git diff --check` | **PASS** |

历史研究目录中存在多套独立实验文件的重复函数定义，不把 `go test ./...` 虚报为无条件全绿；保持历史研究文件原样。

## 四、真实环境与发布仍然 BLOCKED

| 待验收 Gate | 目前结论 |
| --- | --- |
| 正式 MySQL 8 v26→v27 迁移、备份和重复迁移 | **未执行** |
| COIN-M Demo/Testnet 专用凭证、REST/WS host、listenKey、保活与掉线恢复 | **未现场执行** |
| 真正 LONG/SHORT 开仓、部分 Fill、CLOSE、真实 STOP/TP Algo 父子订单 | **未现场执行** |
| 币本位保证金币钱包、张数过滤器、杠杆、逐/全仓，以及 UM 共用持仓模式 | **未现场验收** |
| 主网小额交易、ARM 服务器部署、后台进程重启和生产发布 | **未授权、未执行** |
| 真实手动受控下单/撤单/认领入口 | **未开放**；当前页面故意保持只读，不能以“完整手动交易能力”对外宣称 |
| 其他前端功能的并行未提交改动 | **未隔离提交**；本轮全量前端构建包括工作区现有 Lead/登录等改动，不能把 `static` 同步视为这些功能通过单独 Code Review |

真实交易发布前，必须沿用 [中英双语使用文档](../readme/zh-CN/10-币本位合约交易.md) 的检查清单，由用户另行决定数据库升级和网络验收。**不得**因为编译、Fake 测试或双开关默认 OFF 而跳过真实 DAPI 校验；双开关若被主动打开，在新源码部署后可能发送真实订单。

## 五、最终结论

**CM-7 在“独立四页面 + 运行只读观察 + 中英文 README + 离线端到端测试 + 前后端构建与静态同步”的源码范围内完成，可以结束 CM 开发阶段并进入独立的 Demo/Testnet/发布前核查。**

该结论不等于手动交易页面全部写操作完成，更不是生产交易系统可以直接使用的许可。本阶段没有新增第三层开关、审批系统或复杂交易框架。


---

## 六、CM-7 复审 F8/P1 整改追踪（2026-11-18）

[首次复审](CODE_REVIEW_CM-7-rereview.md) 在 `Processor.Open` 的 DAPI 预检期间修改 `Profit/Loss/TickSize/Leverage`，复现无 STOP/TP 的开仓，按 P1 阻断。现已在 Processor 内补齐预检结束后的配置与保护单二次校验，以及本地持久化成功、交易所 POST 前的最终校验。新增 60 个竞态注入场景和确定未发送意图的本地结案测试。

[第二轮复审](CODE_REVIEW_CM-7-rereview-2.md) 通过 88 个非历史研究 Go 包、CM-7 和四个核心包各五次重复测试、Race、Vet 及 Build。**CM-7 源码与隔离离线测试恢复为通过；跨系统配置与 HTTP 请求不具备严格事务原子性，真实环境发布门禁仍全部保留。**
