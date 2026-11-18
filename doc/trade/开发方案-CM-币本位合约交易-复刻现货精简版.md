# 开发方案 CM · 币本位合约交易（复刻 06 现货最终精简版）

> 方案日期：2026-11-17。当前状态（2026-11-18）：**CM-0～CM-5 已完成源码与离线测试**（CM-5 DAPI 执行原语、真实 Fill 归属、私有 WS 与 Algo 保护单均通过 Fake/Mock），**CM-6～CM-7 尚未实施**；实际数据库未升级，没有真实下单或自动策略运行。
>
> 参考：已落地的 Spot 06-0～06-7 自动交易及四页面结构；现有 U 本位合约策略、持仓、订单与成交链路。源码数据库版本已升级至 v27（CM-1）；实际部署数据库需要后续备份、维护窗口和人工执行 `sync db`，未触及生产库。

## 一、目标与原则

1. **个人使用，保持简单**。完全复刻现货最终方案的四个入口；**配置中心只增加一个全局开关**。每币开关放在币本位币种列表，交互复刻现有 U 本位合约。
2. **币本位独立业务账本**。COIN-M（DAPI）与 U 本位（FAPI）、Spot 在交易产品、合约张数、保证金币种、盈亏计价上严格分开。只复用成熟的公共能力，不混用订单/Fill 记录。
3. **仅做币本位永续**。第一期只接入 exchangeInfo 中 contractType=PERPETUAL、contractStatus=TRADING 的 USD 反向合约，例如 BTCUSD_PERP；暂不支持季度交割合约、交割和移仓。
4. **支持完整合约方向**：LONG、SHORT、CLOSE_LONG、CLOSE_SHORT，以及 U 本位现有的止盈止损、杠杆、逐仓/全仓、交易对选币、技术指标与策略规则。
5. **两级启停，仅控制新的开仓**：全局开关 `config.coin_m_quant_enable` 在配置中心；每币开关复用 `delivery_symbols.enable`，在「币本位交易」币种列表单独配置，与现有合约交易一致。关闭开关不再新开仓，但 Managed 仓位继续检查平仓和订单状态。不增加第三个开关或复杂治理。
6. **安全隔离**：不复用 U 本位的 FAPI 交易执行器和 USD 线性盈亏计算器；明确 DAPI 的账号/钱包/行情/WS 端点与请求限流。所有新配置、开关均默认关闭。禁止自动切换 Binance UM/CM 共用的双向持仓模式。
7. **不迁移旧订单**。复用当前 delivery_symbols 的行情记录但不沿用旧数据作为开启交易的证据。新账本从零建立，真实成交以 DAPI 用户成交及签名用户流为准。
8. **不动 app.conf 现有连接、不动生产数据库**。通过现有 ./go_binance_futures sync db 实施版本化迁移；开发期间仅允许独立测试数据库和 Fake / 离线断言。

## 二、与现货 06 最终版的对应关系

| 现货最终版 | 本方案 CM | 备注 |
| --- | --- | --- |
| 顶级「现货交易」 | 顶级「币本位交易」 | 放在「合约交易」下方、现货上方 |
| 现货交易 / 本地账户 / 策略模板 / 仓位与受控交易 | 币本位交易 / 币本位本地账户 / 币本位策略模板 / 仓位与受控交易 | UI 尽量原样复刻 U 本位合约，数据源换为 CM |
| config.spot_quant_enable | config.coin_m_quant_enable | **配置中心唯一新增开关**，默认 0 |
| spot_symbols.spot_quant_enabled | delivery_symbols.enable | 复用现有币种启停字段，默认 0；在币种列表逐币配置 |
| spot_managed_positions | coin_m_managed_positions | 一次开平仓周期一行，带 position_id |
| spot_managed_orders | coin_m_managed_orders | 开/平仓订单及归属 |
| spot_trade_fills | coin_m_trade_fills | DAPI 真实成交，保存张数、实际计价币 |
| spot_strategy_templates | coin_m_strategy_templates | 独立模板记录，规则语法沿用合约 |
| 现货 BUY / SELL | COIN-M LONG / SHORT / CLOSE_LONG / CLOSE_SHORT | 支持双向仓位和合约止盈止损 |
| 2 秒现货扫描 | 复用现有 2 秒策略调度 | 不创建第二调度中心 |

配置中心独立卡片名称：「币本位合约交易」，**卡片内只提供一个「启用币本位自动交易」全局开关**。杠杆、保证金模式、预算、止盈止损和策略均在「币本位交易」页面逐币配置。侧边菜单使用「币本位交易」，子菜单按上述四项；不新增单独的「币本位订单」顶级菜单。账户页承载订单历史和子表格成交明细。

## 三、必须单独处理的币本位差异

### 3.1 合约张数与预算

币本位合约 quantity 是**合约张数（contract number）**，不是 BTC 数量、也不是 USDT/USDC 数量。必须从实时 exchangeInfo 读取 contractSize、marginAsset、tickSize、stepSize、minQty、maxQty 和相关过滤器；**不得**把 U 本位 quantity 直接转给 DAPI。

建议每币使用「保证金预算（USD 等值）」输入，而不是现货的“投入 USDT”，避免误认为账户持有 USDT。新建 budget_usd 字段（默认 0），不迁移 delivery_symbols 旧 usdt 数值；开仓需由用户显式保存非零预算。

示例，仅用于公式说明：contractSize=100 USD/张、预算=250 USD、杠杆=4 → 理论名义额=1000 USD → 10 张。实际还必须检查币本位钱包可用 BTC、交易所限额、手续费预留及精度，不满足则拒绝开仓，不能偷偷扩大预算。

不引入多资产自动转换或自动划转保证金功能。BTCUSD_PERP 需要 BTC 保证金，ETHUSD_PERP 需要 ETH 保证金；无可用余额就不开新仓。

### 3.2 反向合约盈亏（inverse PnL）

设 N 为张数、C 为每张 USD 面值、Pe 为开仓价、Px 为当前标记价格或实际平仓成交价格。

- 多仓币本位盈亏（保证金币种） = N × C × (1 / Pe − 1 / Px)
- 空仓币本位盈亏（保证金币种） = N × C × (1 / Px − 1 / Pe)
- 用于前端说明的 USD 估值 = 该币盈亏 × 对应时点币/USD 价格，**只是换算展示，不是交易所实际结算 USD**。
- ROI 应以本币盈亏 / 本币初始保证金计算，按 Binance 实际杠杆/保证金口径核对，不能照搬线性合约价格涨幅×杠杆公式。
- 实现中保留币本位计价精度（字符串 Decimal），不能把本币 PnL 当成 USDT PnL。

例：BTCUSD_PERP，N=10，C=100 USD，Pe=50000 USD，Px=55000 USD；多仓未实现盈亏为约 0.00181818 BTC。按 55000 USD/BTC 的展示参考价，约 100 USD（并非账户收到 100 USDT）。

已实现盈亏、未实现盈亏、手续费、资金费分别标明结算资产；资金费若需要精确到持仓周期，必须使用真实交易所收入记录核实后再计入，不编造交易 Fill。统计默认按币种分组；不同币种不直接加总为「总 BTC」。

### 3.3 COIN-M API、持仓模式及 2026 更新

- 交易/行情采用 DAPI（生产 REST https://dapi.binance.com；新订单 POST /dapi/v1/order；ExchangeInfo GET /dapi/v1/exchangeInfo；成交 GET /dapi/v1/userTrades）。
- 自身 User Data Stream 使用 DAPI listenKey 和对应 COIN-M WS 流（生产 wss://dstream.binance.com），禁止用 FAPI/现货的用户流误记成交。
- 当前币安说明：UM 与 CM **共用 dualSidePosition 持仓模式**。第一期读取并验证现有模式；**不得为启用币本位自动调用修改模式接口**，防止影响 U 本位在途仓位。双向仓的 LONG/SHORT 请求须使用正确 positionSide。
- 2026 年 COIN-M 条件单迁移之后，STOP、TAKE_PROFIT 等不能继续当作普通订单交给 /dapi/v1/order；若实现交易所条件保护，须走当前支持的 /dapi/v1/algoOrder 及相应查询/取消逻辑。不能用旧 SDK 方法“能编译”推断接口仍有效。
- 用户订单仍按现有交易规则进行：API/WS 正常响应后维护订单状态，真实成交由 DAPI Fill 证据确认，重试使用稳定 clientOrderId，不引入额外业务审批。
- 币本位 Demo/Testnet **必须与主网隔离**。当前项目 deliveryClient 由主网 api_key/api_secret 创建，即使全局 testnet=true 也仍指向主网；正式接单前必须消除此缺陷。Demo endpoint 或凭证缺失时应拒绝真实发单，绝不能静默回落主网。
- 复用现有 REST Weight 统计/429、418 退避，但单独登记 delivery/COIN-M；市场行情可复用既有 delivery_symbols/WS 数据，禁止用 Spot/FAPI 行情冒充 CM 价格。

### 3.4 开关和退出行为

1. 配置中心的全局 `config.coin_m_quant_enable=1` 且币本位列表中该币种的 `delivery_symbols.enable=1`，才可能产生新开仓；同时需规则命中、DAPI WS 已认证、合法张数与可用保证金币、没有重复开仓。
2. 关闭任何开仓开关，只禁止新开仓；已有 Managed 仓位继续按 LONG/SHORT 出场规则及止盈止损检查是否需平仓，未结算订单继续对账。关闭开关本身**不强平、不撤销已有保护单**。
3. 所有开关关闭且没有 Managed 仓位/待处理订单时，停止不必要的高频**私有账户请求**；新开仓启动需同时满足全部门槛。**全局/每币交易开关仅控制开仓，不控制公开行情采集**：CM-2 已接入的公开 DAPI 合约信息（启动时、此后每 12 小时）与 COIN-M 行情 WS 在开关 OFF 时仍运行、占用公开行情请求权重；不使用签名账户接口。若运维要求完全停止 COIN-M 外联，当前版本需要停止应用或调整启动任务，尚无仅停采集的独立运维机制；不向配置中心添加第三个开关。
4. 不能单凭「本地无仓位」把交易所手动持仓归为 Managed；不能因为重启漏单而重复提交新开仓。

## 四、数据库（版本化、独立于 U 本位及 Spot）

CM-1 已实现 **源码 Schema v27**：基于 v26，使用独立 Go 迁移方法 `ensureV27CoinMSchema`，并通过 SQLite 物理 v26→v27 模拟测试；**未对实际运行数据库执行迁移**。

**复用 delivery_symbols** 保存币本位产品信息与每币策略配置：直接使用现有 `enable` 作为每币交易开关（默认 0）；新增必要的 contract_size、margin_asset、contract_type、budget_usd（默认 0）等字段。保留现有策略及技术指标字段；旧 `usdt` 数值不直接作为新交易预算。**不新增独立的每币开关字段或第三层开关。**

新增以下四张独立业务表：

| 新表 | 用途 | 关键字段与约束 |
| --- | --- | --- |
| coin_m_managed_positions | 本系统一次持仓周期一行 | id/position_id、symbol、LONG/SHORT、margin_asset、contract_size、entry_price、opened_contracts、remaining_contracts、已实现盈亏币种、状态 |
| coin_m_managed_orders | 与 position_id 绑定的开仓/平仓订单 | client_order_id 唯一、exchange_order_id、BUY/SELL、position_side、OPEN/CLOSE、requested_contracts、filled_contracts、状态 |
| coin_m_trade_fills | 真实逐笔成交、收益与手续费证据 | (symbol, exchange_order_id, trade_id) 唯一、order_id、position_id、quantity_contracts、price、realized_pnl、commission、commission_asset、trade_time |
| coin_m_strategy_templates | 独立策略模板和技术指标 | name、technology、strategy、create/update 时间 |

无需新增审批、事件溯源、虚拟账户、第三套通用订单框架、复杂人工认领审核或独立 Task Center。必要的唯一索引、持仓归属与 Fill 去重不属于可删除的“过度设计”，它们直接防止重复真实下单。

默认不迁移旧 delivery_symbols 里的订单/策略账户数据。新建表为零记录；首次安装或升级不会自动开新仓。新增表和列必须通过 ./go_binance_futures sync db 生成，且新增 Schema 不覆盖 v26 数据。

## 五、页面（完全照 U 本位的现有 UI）

顶级菜单「币本位交易」，置于「合约交易」与「现货交易」之间。

1. **币本位交易**：交易对列表、模糊查询、**每币 `enable` 启停开关（与 U 本位合约一致）**、批量启停、杠杆、逐/全仓、合约张数精度、预算（USD 等值）、技术指标、LONG/SHORT/CLOSE_* 策略；只展示 COIN-M PERPETUAL；金额文案不得写成“投入 USDT”。
2. **币本位本地账户**：独立币本位钱包余额与 Managed 持仓/订单；一行一个开平仓周期，可展开订单与真实 Fill；收益栏显示本币、可选择参考 USD 换算，支持筛选后的总计（按结算资产分类）。
3. **币本位策略模板**：复刻 U 本位技术指标和四种策略规则；模板数据隔离，允许复制规则内容而非混用数据库模板 ID。
4. **仓位与受控交易**：复刻 U 本位 Managed 生命周期、查看异常、必要的手动受控操作，所有写操作只作用于 COIN-M 订单，不允许触碰 Spot 或 USDT-M 仓位。

配置中心增加独立「币本位合约交易」卡片，**只新增一个全局开关 `config.coin_m_quant_enable`（默认关闭）**；币种开关直接在「币本位交易」列表配置。其他交易参数全部逐币设置，配置中心不重复展示。

## 六、实现阶段（按顺序执行，每阶段结束 Code Review）

| 阶段 | 任务 | 完成标准 |
| --- | --- | --- |
| CM-0（已完成） | 现状映射和边界确认：对比 06 现货与 U 本位合约链路，列出差异；核对 SDK v2.8.12 是否支持当前 DAPI / Algo | [静态分析报告](CM-0-币本位交易现状映射与开发边界.md) + [Code Review](CODE_REVIEW_CM-0.md)；未改真实交易逻辑 |
| CM-1（已完成） | 独立数据模型、v27 migration、全局开关默认关闭、复用已有币种 `enable`、测试库迁移 | [Code Review](CODE_REVIEW_CM-1.md)；仅新增一个全局开关、不新增每币开关字段；离线 v26→v27 和重复升级通过，未部署 |
| CM-2（已完成） | COIN-M 行情、合约信息与客户端：DAPI / WS / Testnet 隔离、精度过滤器、Weight | [Code Review](CODE_REVIEW_CM-2.md)：真实公开 DAPI exchangeInfo 200；定向离线测试与 12 包回归通过，签名 WS/真实订单未接入 |
| CM-3（已完成） | 币种列表启停、独立模板 CRUD、配置中心单开关、前端四页面骨架与本地账本只读 API | [Code Review](CODE_REVIEW_CM-3.md)：前后端构建、87 个非历史研究 Go 包、CM-3 五次重复测试与 Race 检测通过；dist 已同步 static，真实交易与完整 PnL 留待 CM-4～CM-6 |
| CM-4（已完成） | 独立持仓周期、订单/Fill 只读汇总，按币本位 mark 计算 inverse PnL，参考 USD 估值、结算资产分类汇总、DAPI 钱包只读 | [Code Review](CODE_REVIEW_CM-4.md)；[首次复审 F1](CODE_REVIEW_CM-4-rereview.md) 已在 [第二轮复审](CODE_REVIEW_CM-4-rereview-2.md) 中修复并通过 11 组证据异常探针与 87 包离线回归；Fill 实际采集、签名 WS、下单留待 CM-5，真实 MySQL 与实盘验收尚未执行 |
| CM-5（源码/离线已完成） | 独立 COIN-M DAPI 普通订单、真实 Fill、私有 User WS、双 STOP/TP Algo 保护单、部分成交与不确定响应恢复 | [Code Review](CODE_REVIEW_CM-5.md)、[独立复审](CODE_REVIEW_CM-5-rereview.md)、[整改复审](CODE_REVIEW_CM-5-rereview-2.md)：修复 O2 满页证据状态、O3 保护单类型校验，88 包 Go 测试、4 包 5 次重复 Gate、Race Detector、Vet、构建通过；尚未连接自动调度、没有在真实 Testnet/Mainnet 发单或执行迁移 |
| CM-6 | 接入现有每 2 秒共享调度；全局与每币开关、出场优先、API 节流与错误恢复 | 两级启停控制新开仓，关闭任一开关后已有 Managed 持仓仍按规则平仓 |
| CM-7 | 完整页面、README 中英双语、端到端测试、发布前检查 | Build→dist→static；Go 测试、离线回归；明确真实环境验收仍未执行 |

验收重点：相同 signal 不重复提交；合约数量不误当币数量；不同 marginAsset 的盈亏不混加；错误价格/规则/余额时拒绝交易；数据库或签名 WS 故障时不发新开仓；已有 U 本位/Spot/Lead 行为不变；升级 Schema 不会自动开关。

建议推进次序：先 CM-0→CM-4 完成独立配置、账本与 UI，再 CM-5→CM-7 打通 DAPI 真实执行。实现完成并不表示已获授权或已通过实盘验收；真实账号联调、Demo/主网回归应另行进行。

## 七、已有代码可复用的起点

- 币本位行情/客户端：feature/api/binance/delivery.go、feature/api/binance/index.go、feature/delivery.go、models/delivery_symbols.go。**现有 deliveryClient 在 testnet=true 时仍然用主网账号/端点，只能复用基础行情结构，不可直接用于新交易执行。**
- 现货 06 全链路：feature/spot_quant_runner.go、spot/execution、spot/ledger、models/spot_quant.go、Vue src/views/spotQuant；复用两开关、2 秒调度、成交幂等、UI 布局的已验证模式。
- U 本位合约：feature/account_trade_cycle.go、futuresownership、tradefills 及 futures/symbol.vue / strategyTemplate.vue / account2.vue。复用策略 DSL、做多做空与止盈止损的业务语义，不直接共用订单写库接口。
- API 使用量：service/binanceapiusage；复用产品隔离的 REST 权重采集和 429/418 处理。

## 八、官方依据（2026-11-17 核对）

- Binance COIN-M Trade API： https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-coin-m-futures/api/rest-api/trade
- Binance COIN-M Market Data / exchangeInfo： https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-coin-m-futures/api/rest-api/market-data
- Binance COIN-M User Data Streams： https://developers.binance.com/en/docs/products/derivatives-trading-coin-futures/user-data-streams
- Binance COIN-M 更新日志（2026 UM/CM 整合）： https://developers.binance.com/zh-CN/docs/products/derivatives-trading-coin-futures/change-log

> 设计取舍：这是一套 **独立产品账本 + 重用现有 UI/策略/调度 + DAPI 专用执行适配层** 的最小完整交易闭环。不是重新造一套大型交易平台。
