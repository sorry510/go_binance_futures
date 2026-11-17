# CODE_REVIEW_CM-0 · 币本位交易现状映射与开发边界

> 2026-11-17；阶段：CM-0；仅设计和静态代码审查。  
> 主文档：[CM-0-币本位交易现状映射与开发边界.md](CM-0-币本位交易现状映射与开发边界.md)  
> 依据：[开发方案-CM-币本位合约交易-复刻现货精简版.md](开发方案-CM-币本位合约交易-复刻现货精简版.md)

## 一、审查范围

- 已读取本地 Go 代码中的 `main.go`、`feature/account_trade_cycle.go`、`feature/spot_quant_runner.go`、`feature/delivery.go`、`feature/api/binance/index.go` 与 `delivery.go`。
- 已读取 `models/delivery_symbols.go`、`models/tableStruct.go`、现货 v25 迁移及业务表、FAPI Managed Position/Order/Fill 模型。
- 已核对前端 Futures/Spot Quant 路由及 `dashboard/configShow.vue` 单开关形式。
- 已在本地依赖 `github.com/adshao/go-binance/v2@v2.8.12/delivery` 中核对 API 方法、User Data WS 消息结构。
- 官方 Binance 2026 年 COIN-M Trade、Market Data、Change Log、User Data Streams 作为外部接口约束。
- **未启动服务、未连主网签名接口、未修改数据库或交易业务代码。**

## 二、Code Review 结果

| 关键项 | 结论 | 必须遵守的开发边界 |
| --- | --- | --- |
| 06 精简复刻范围 | PASS | 复用 4 个页面入口、每 2 秒调度、独立 Fill 账本，不制造第三套框架 |
| 币本位独立产品边界 | PASS | 不把 CM 数据写入 FAPI Main/Lead、Spot 的订单及持仓账本 |
| 全局开关唯一性 | PASS | 配置中心只加 `coin_m_quant_enable` 一个新开关，默认 0 |
| 每币开关 | PASS | 只用 `delivery_symbols.enable`，不新增 `coin_m_quant_enabled` |
| 历史配置安全 | PASS（已识别风险） | 既有 `delivery_enable` 值未知，不能当新开仓许可；不能用旧 `usdt=10` 作为预算 |
| 现有 CM 市场数据 | PARTIAL | `UpdateDeliveryCoinByWs` 尚未启动，精度发现定时调用被注释 |
| 现有 CM 主网/测试网隔离 | BLOCKER（CM-2） | 当前 `deliveryClient` 总是用主网凭证和 mainnet Transport；必须先隔离再允许任何实盘写接口 |
| SDK DAPI 基础功能 | PASS | ExchangeInfo、普通订单、查撤单、账户、持仓、WS 可用 |
| SDK 真实 Fill / Algo | BLOCKER（CM-5） | v2.8.12 缺 userTrades 和新版 Algo 完整封装；不能旧 STOP 接口冒充新 API |
| 逆向合约数量与盈亏 | BLOCKER（CM-4/CM-5） | ContractSize、保证金币种、反向盈亏和真实 Fill 独立计算 |
| Shared position mode | BLOCKER（CM-5 前确认） | 不调用 UM/CM 的账户级持仓模式修改接口 |
| Schema 与生产风险 | PASS | 当前 v26，CM-1 暂定 v27；只有用户明确部署时运行 `sync db` |
| 测试报告真实性 | PASS | 本阶段只做静态分析；**未宣称**已通过实盘、WS 或数据库集成测试 |

## 三、结论

**CM-0 可接受，允许进入 CM-1 的实现阶段。**

批准范围仅限「新增独立数据库结构、唯一 CM 全局开关、复用币种 enable、保持默认关闭、测试用例及迁移 Review」。**CM-1 不可包含真实 COIN-M 订单发送、账户级持仓模式变更、生产数据库迁移或服务重启。**

后续阻断项：CM-2 解决专用签名客户端与 WS 环境隔离；CM-4 完成真实成交/币本位盈亏模型；CM-5 才能接入 DAPI 普通订单、Algo、userTrades 的完整链路并进行离线回归。

目前不触发真实资金交易。
