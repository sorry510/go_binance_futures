# 24h 波动通知阈值删除审计

## 范围结论

本次删除对象是配置字段 `WsFuturesPriceChangeLimit` 对应的旧版 24h 涨跌幅通知运行逻辑。极速波动通知、24h 行情数据、市场波动日志查询能力和既有日志数据不属于删除范围。

前端项目 `/Users/zhz/work/binance/go_binance_futrues_new_ui` 已不存在 `WsFuturesPriceChangeLimit`、`wsFuturesPriceChangeLimit`、`ws_futures_price_change_limit` 或 `24h波动通知阈值` 引用。

## 配置中心 → 合约交易

| 编号 | 代码位置 | 现有作用 | 对应页面操作流程（保留原始菜单名/页面名） | 备注 |
|---|---|---|---|---|
| 1 | `feature/api/binance/index.go:588` `flushLatestWsTickers` | 每轮 WebSocket 行情落库后调用旧版 24h 涨跌幅通知 | `配置中心 → 合约交易 → 24h波动通知阈值(%)` | 删除调用；保留极速波动通知调用 |
| 2 | `feature/api/binance/index.go:812` `symbolPriceNoticeMap`、`priceChangeNotice` | 根据 24h 涨跌幅阈值发送通知、写入 `price_change/24h` 市场波动日志，并按币种冷却 4 小时 | `配置中心 → 合约交易 → 24h波动通知阈值(%) → 合约交易 → 市场波动日志` | 删除运行逻辑和专用缓存；历史日志数据保留 |
| 3 | `controllers/index.go:87` `GetServiceConfig` | 在配置中心接口中返回 `WsFuturesPriceChangeLimit` | `配置中心 → 合约交易 → 24h波动通知阈值(%)` | 删除显式接口字段；前端已删除对应控件 |

## 文档与通知文案

| 编号 | 代码位置 | 现有作用 | 对应页面操作流程（保留原始菜单名/页面名） | 备注 |
|---|---|---|---|---|
| 4 | `README.md:124` | 说明旧版 24h 阈值和极速波动通知 | `配置中心 → 合约交易` | 删除旧版说明，保留极速波动说明 |
| 5 | `README.EN.md:129` | 英文说明旧版 24h 阈值和极速波动通知 | `Configuration Center → Futures Trade` | 与中文 README 同步更新 |
| 6 | `lang/config/zh.json:53`、`lang/config/en.json:53` `futures.up_or_down` | 旧版 24h 涨跌幅通知标题 | 后台通知，无独立页面入口 | 唯一运行引用随 `priceChangeNotice` 删除，语言键一并删除 |

## 数据库保留项

| 编号 | 代码位置 | 现有作用 | 对应页面操作流程（保留原始菜单名/页面名） | 备注 |
|---|---|---|---|---|
| 7 | `models/tableStruct.go:26` `WsFuturesPriceChangeLimit` | 映射数据库列 `config.ws_futures_price_change_limit` | 无页面入口 | 按用户要求保留，不修改模型和数据库列 |
| 8 | `command/db_update.go:21` `createConfig` | 新数据库初始化时写入 `ws_futures_price_change_limit` 默认值 | 无页面入口 | 属于数据库初始化契约，按用户要求保留 |
| 9 | `models/futures_market_notice_log.go` 及市场波动日志接口 | 保存并查询既有市场波动日志 | `合约交易 → 市场波动日志` | 保留表、接口及历史 `price_change/24h` 数据；极速波动仍继续写日志 |

## 待确认事项

无。删除边界已明确，数据库字段、初始化 SQL、历史日志和极速波动功能均保留。
