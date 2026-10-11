# Stage 6-5 — Vue3 Lead 管理页面与中英文

> 2026-10-11；离线开发/Mock 验证，**真实 Binance 与 Stage 7 实盘未授权**。

## 实际交付

- 前端新增 `src/views/futures/leadTrading.vue`、`src/api/leadTrading.ts`、`/futures/lead` 菜单，以及 `locales/zh-CN.yaml` / `locales/en.yaml` 双语文案。
- 六 Card：Lead 状态、凭证、独立风控、自动交易、受控持仓、本地订单/诊断。Card 标题无歧义开关，禁止真实启用按钮可视化为 disabled。凭证字段使用密码输入，保存后清空；密钥不通过 GET 返回。
- 15 秒刷新只读状态、Lead 所有权列表及 5 分钟 API 指标；风险设置编辑期间不因轮询被覆盖。无本地配置时展示安全空态，不把 0 伪装为实测用量。
- 页面沿用现有 Element Plus 与 Vue3 布局，没有修改其他已有交易菜单，保留原本用户未提交的 locale 与 SmartLocalV2PreviewDialog 修改。

## 本阶段安全纪律

未修改 `app.conf`；没有执行 `sync db` 或生产数据库写入，不触发真实 Binance 下单/撤单/WS 启动；无自动 Git commit/push。历史用户未提交修改不回退。
