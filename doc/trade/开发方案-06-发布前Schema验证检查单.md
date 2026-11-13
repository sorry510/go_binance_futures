# 方案 06：发布前 Schema 验证与成套部署检查单

创建日期：2026-11-11（06-2 Code Review F1/F3 整改）

## 1. 运行态检查的实际边界

用户已明确要求移除 `main.go` 的 `dbVersion >= 25` 运行态 Schema 检查，因此 **正常服务启动只校验 `config.version` 与代码所需版本**。启动流程**不会调用** `command.VerifyV25SpotQuantSchema`，不得据此声称同号 v25 历史旧表结构已被自动拦截。

`command.SyncDatabase(25)` 会执行 `VerifyV25SpotQuantSchema`，在已标记 v25 而不兼容的数据库上拒绝继续，避免误补建空 Fill 表而掩盖数据缺失。但 **`sync db` 是可能执行 DDL 的升级命令，不是纯只读检查**。务必由用户在已授权的目标环境中显式执行，生产环境不由 AI 自动操作。

**发布硬性要求：** 在任何 06-3/06-4 本地账本写路径投入生产之前，必须先核实并记录 v25 物理 Schema 校验结果。无法确认则停止发布或回滚，而不是依赖版本号放行。

## 2. 执行步骤（所有生产变更需用户另外授权）

- [ ] 记录目标环境、Git commit、二进制版本、当前 `config.version`、配置所在数据库和备份状态；确认**不是**已废弃方案 02 的同号 v25 物理表结构。
- [ ] 先在用户指定的目标库完成**只读**表/字段检查；MySQL 可查询 `information_schema.tables/columns`，确认存在 `config.spot_quant_enable`、`spot_symbols.spot_quant_enabled`、`spot_managed_positions`、`spot_managed_orders`、`spot_trade_fills`、`spot_strategy_templates`，并且 10 个财务字段是 `DECIMAL(36,18)`。这一步不能仅查 `config.version`。
- [ ] 生产数据库备份、维护窗口和权限由用户确认后，**只使用项目统一命令** `./go_binance_futures sync db`；升级命令可能写表或索引。若发现 v25 旧结构冲突或校验失败，立即停止；不得擅自改 `config.version`、清空/重建 Fill 表或写生产库。
- [ ] 留存 `sync db` 的退出码、日志、版本号与表结构验证证据；在启动完整新版本前再用只读 SQL 核对预期表、字段、索引与开关默认值。
- [ ] 全局 `config.spot_quant_enable` 保持 OFF；任一真实 Spot 自动单的最终放行必须同时满足**全局开关 ON、单币 `spot_quant_enabled=1`、账户与资金/精度/幂等检查通过**。全局 OFF 时可以保留单币偏好但**不能发新自动单**；由 06-4 交易执行代码和测试强制实现。
- [ ] 06-5/06-6 完成后，统一验证前端 `dist`、后端 `static`、新后端可执行文件、数据库 Schema/接口版本一一匹配，避免“页面已更新而 API 不存在”或反向情况。
- [ ] 在 Fake/Testnet 和只读账户确认无误后，才在用户单独批准下考虑真实环境发布；任何异常停止交易并恢复成套旧二进制/前端构建及合适的 DB 备份，数据库回滚必须先确认不会丢失成交账本数据。

## 3. 06-2 当前状态

06-2 只实现 Spot 配置、币种管理、模板 CRUD 与模拟预览。**现有服务安装包没有同步此阶段的隐藏前端页面/API**；本阶段没有自动订单执行接口和资金风险。`go_bn_test` 早前完成的 v25 验证不代表任何生产库验收。

**Gate：研发/离线 Gate 可继续，生产/真实下单/成套部署 Gate 保持 BLOCKED，直至以上检查和后续阶段验收全部完成。**


## 4. 06-6 本地只读实测与生产 Gate（2026-11-12）

详细证据：[06-6 实施记录](开发方案-06-6-完整联调与交付-实施记录.md)。

本轮**只在开发机 `go_bn_test`** 运行只读事务（MySQL 5.6.51，`config.version=25`），4 张 Spot Quant 表全部存在，10 个 DECIMAL 字段精度 `(36,18)`，订单/Fill 唯一索引和 position/order 查询索引均存在；总开关 `spot_quant_enable=0`，809 个 Spot symbols 全部 `spot_quant_enabled=0`，Spot Managed Position/Order/Fill 皆为空。只读 EXPLAIN 显示精确 symbol 使用组合索引，订单与 Fill 使用对应关联索引；**不能据此推断 MySQL 8 P95**。

前端 `pnpm build` 后完整同步至**本地后端工作区** `static/`，115/115 个相对路径与 SHA-256 全等；本次原资源备份在 `/tmp/spot066_static_backup.9TdpPs/static/`。**服务器生产二进制、静态文件和 Schema 尚未检查或更新**，本节没有勾选上面任何生产维护/升级项目。

Testnet public `exchangeInfo` 成功、签名账户 `-2015`（未授权/Key/IP 具体原因待确认）；主网仅执行签名 GET 账户核验成功（不披露余额，不发订单）。**未进行 Testnet 真实小额 BUY/SELL、WS 长连或主网下单**。当前代码默认不注册 Spot Quant 自动交易循环、拒绝主网下单/撤单，相关安全开关保持关闭。生产和任何真实交易继续 **BLOCKED**。

若需要本地恢复原静态资源，操作前先确认没有其他 AI/开发人员继续修改 `static`，然后可手动执行：
```bash
rsync -a --delete /tmp/spot066_static_backup.9TdpPs/static/ /Users/zhz/work/binance/go_binance_futures/static/
```
这只回滚**本地文件**，不触及生产或数据库；本次未执行。
