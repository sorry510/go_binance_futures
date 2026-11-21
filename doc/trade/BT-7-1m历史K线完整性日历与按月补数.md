# BT-7｜1m K线完整性日历页面与按月补数

开发日期：2026-11-20（本地项目）

## 需求

- 原「历史回测」页面增加一个「1m K线完整性」按钮，跳转到独立页面 `/ai/backtest/1m-calendar`。
- 不新增侧边栏菜单；默认查询最近 12 个已完成的自然月份。
- 筛选字段**仅**：币种下拉和月份范围。
- 显示每月分钟 K 线覆盖率；点击月份显示每天覆盖状态。提供「补齐当月」与「补齐所选范围」按钮。
- 去掉原设计里的「最近补数任务」列表及多余筛选项。
- 不需要策略模板；只补充 1m 行情，不额外获取指标周期和资金费率。

## 后端

路由（复用现有 API 登录认证）：

- `GET /agents/historical-market/1m-coverage?symbol=BTCUSDT&from=2025-11&to=2026-10`：只读返回月份及每日 `expected/stored/percent/status`。
- `POST /agents/historical-market/1m-repair`：请求体 `{"symbol":"BTCUSDT","from":"2026-10","to":"2026-10"}`；可指定一个月或连续月份范围。
- 补数进度沿用 `GET /agents/backtests/prefetch/:jobId`，无新任务表。

代码：

- `service/historicalmarket/kline_1m_calendar.go`：UTC 月份校验（2019 年后、最多 60 个月、不允许未来月份），基于 `market_klines_1m_chunks` 的连续压缩月度元数据与 `market_klines_1m` 的分钟行实现**只读覆盖率**；重叠分钟不重复计数；当前月只统计截至最后已结束分钟，未来日期显示未开始。数据库查询不解析全量压缩 payload、不调用交易所。
- `service/backtest/kline_1m_calendar_repair.go`：独立于策略模板的补数任务。复用原有公共归档下载与 SHA-256 校验、最多 3 个 ZIP 同时下载、逐月串行入库；仍有缺口时通过币安 REST 补充，并在每月完成后检查。共用原有预取互斥锁，避免与回测预取同时写入相同行情。
- `controllers/agent_historical_market.go`、`routers/router.go`：增加 API。

预期：回测完整性查询只基于本地已保存数据；对于刚上市币种的上市前月份，由于没有本地可信上市时间元数据，未覆盖部分会显示缺失。补齐执行时会使用币安最早 K 线时间限制实际下载范围。补齐成功不代表上市前有 K 线。

## 前端

- `src/views/ai/Kline1mCalendar.vue`：双筛选条件、月度日历卡片、每天状态、选中月统计，以及补数按钮、确认框、进行中进度条。
- `src/views/ai/backtest.vue`：历史回测列表标题旁新增按钮。
- `src/router/modules/futures.ts`：隐藏路由，无侧边栏项目。
- `src/api/backtest.ts`：覆盖率、补数请求与类型。
- 不显示「最近补数任务」。进度条只显示当前页面启动的任务，任务在后端继续执行。

## 验证

- 前端 `pnpm exec vue-tsc --noEmit --skipLibCheck --pretty false`、`pnpm build` 通过。
- 后端 `go test ./service/historicalmarket ./service/backtest ./controllers ./routers -count=3 -p=2`、选择性 `go test -race`、`go vet`、`go build` 通过。
- 新增测试：自然月份输入校验、压缩月度与未压缩分钟行统计、当前月未来分钟排除、无数据月份、非法币种、补数不依赖策略模板、真实本地模拟 REST 补数成功。
- 正式 frontend `dist` 已同步到后端 `static`，原平台配置 `platform-config.json` 原样保留。
- 未直接清理或变更现有 MySQL 行情；未修改 `conf/app.conf`、未重启正在运行的后端。**后端需重新编译并重启才可使用新 API。**

