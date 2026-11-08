# 01-5 Code Review 整改记录（2026-11-08）

对应原始审查：[CODE_REVIEW_01-5.md](CODE_REVIEW_01-5.md)。本文件是审查后的增量整改记录，不覆盖原报告观察或实盘 Gate 结论。**当前只有本地代码/测试通过；生产 v23 DB、T0、真实 Binance 验证与双端部署均未获授权。**

## 按发现项逐一处置

| Finding | 整改内容 / 判断 | 状态 |
| --- | --- | --- |
| **015-F1 P1** | `controllers/orders.go` 的 `respondError` 和 `success` 从 `ctrl.Ctx.Resp(map[string]any{...})` 改为 **`ctrl.JSONResp(...)`**。不再根据 `Accept` 头分发到 YAML/XML/Protobuf。新增 `TestStage015HistoryResponseFixedJSONAcrossAcceptTypes` 和 `TestStage015SuccessResponseAlwaysJSON`，以 JSON、YAML、XML、text/xml、Protobuf `Accept` 验证 HTTP 400/410/200、`application/json` 和可 JSON 解析的 `code/data/msg`，无 panic。 | **代码修复并测试通过** |
| **015-F2 P1** | 前端已在自身 `dist` 构建，后端 `static` 和根目录主程序保持不动，原因是运行中进程会立即对外提供磁盘静态文件。把前后端版本绑定、产物 staging、协同发布和回滚检查列入下面正式部署 SOP。不得只替换新后端 API 或先覆盖旧进程的 `static`。 | **01-6 上线硬前置，尚未发布** |
| **015-F3 P2** | 当前 `futures_trade_fills.position_id` 索引已存在，无需重复建；`RecentClosedSymbols` 的 `trade_time` 过滤没有辅助索引，可能拖慢高频选币。01-6 必须在目标 MySQL 8 上以只读 `EXPLAIN`、P95 查询时间、行数/分布、QPS 实测，若不达标走 **新 schema version + `sync db`** 加索引。没有凭空选择索引或擅改生产 v23。 | **数据库压测 Gate 未关闭** |
| **015-F4 P2** | 检查 `controllers/login.go`：当前单 `web::username` / `web::password` 生成带 `Username` claim 的 JWT，没有角色/账户级用户系统。本项目明确采用 **单一操作员读权限模型：唯一登录身份可只读 main 与 lead 两个账户**。路由仍经 `JwtMiddleware`；新增 `OrderController.authorized()` 防御性要求中间件填入有效 `Username` claims（否则 401），避免未来误漏路由过滤。main/lead 的 `position_id` 仍须严格绑定账户，否则 404。单操作员身份不能伪称有 RBAC；如未来引入多个独立登录用户，必须先实现账户级 RBAC，当前模型不能直接复用。 | **当前单账号策略已明确并加固；生产登录模型只读复核仍需 01-6** |
| **015-F5 P2** | 前端 `src/api/order.ts` 新增 `OrderHistoryApiResult<T>` 明确 `code/data/msg`；改为 `http.get<OrderHistoryApiResult<ManagedOrderPage>, unknown>` 和 `http.get<OrderHistoryApiResult<ManagedCycleDetail>, unknown>`。清除页面 `as ManagedOrderPage`、`as ManagedCycleDetail`，直接使用 `res.data`；避免误把第二泛型当作 query 参数类型。 | **代码修复，TS 类型检查通过** |
| **015-F6 P3** | 盈亏着色维持既有 `testOrder.vue` 与系统的 **盈利绿色 / 亏损红色** 风格，不在审计整改中单独改变此页面导致同系统不一致；如未来统一改为「红涨绿跌」，必须全站一起改。 | **确认保持现有风格** |
| **015-F7 P3** | 继续保留后端 `symbol__icontains` 模糊匹配（`BTC`/ `USDT` 均有效），前端中英文 placeholder 直接标记「模糊匹配 / contains match」。 | **前端修复** |
| **015-F8 P3** | Owner 下拉继续提示五种已知 Owner，但启用 `filterable + allow-create`，可输入符合后端正则的其它 Owner，无需更改可用交易 Owner 源码枚举，也不新增 DB 查询 API。 | **前端修复** |
| **015-F9 P3** | `service/orderhistory/history.go` 不再直接序列化整张 `FuturesManagedOrder` ORM 模型，改为显式只读 `ManagedOrderView` 白名单；`Position` 删除未在订单历史页面使用的 `source_ref` 返回。客户端订单号、Algo ID、真实订单号、逐笔价格/费用仍保留，**只允许在已认证单操作员的历史页面查看，不应当作公开 API 数据**。新增测试往真实模型 `source_ref` 写入敏感标识，确认列表/详情 JSON 不包含此字段，同时检查必需的 Fill/order IDs 未丢失。 | **代码修复并测试通过** |

## 01-6 部署发布 SOP（本次只记录，不执行）

1. 前置：目标生产数据库 Schema **v23 与代码契约一致**、所有需要的 Fill/Managed 字段存在；只读核查真实仓位/挂单、T0、Lead 登录身份、未归属仓位和完整结算/实盘 Gate。没核实或存在 blockers 则停止；不以 SQLite 单测替代 MySQL 8 性能 Gate。
2. 分别构建并校验后端目标架构二进制（ARM 服务器对应目标架构）和前端 `pnpm build` 生成的 **同版本** `dist`；记录 Git commit、前端产物哈希、二进制哈希、数据库 schema 版本及 UTC/北京时间发布记录。
3. **不要直接把 dist 覆盖正在运行旧服务的 static。** 将完整二进制 + dist 拷贝至独立 release staging 目录，校验文件与哈希；备份当前二进制和 static。需要在维护窗口先停止旧实例，按已核验的发布切换流程一起激活新二进制及完整 static 目录，不可出现旧页面请求新 API 或新页面请求旧 API 的半发布状态。
4. 冒烟：验证 JWT 认证、`GET /orders` 和 `GET /orders/:position_id/details` 分页、账户隔离、JSON `Accept` 兼容、正反数据完整性展示、只读 410 DELETE，及原有 OPEN/CLOSE/TP/SL 风控链；确认新 `static` 在应用启动前已进入同版本目录且资源路径不缺失。
5. 失败立即执行协调回滚（恢复**成套**的上版二进制和静态目录），勿为了回滚而恢复旧 `order` 自动猜配/删除行为或回退生产数据库 Schema。明确版本向前兼容性，不得假设旧版本可读新 DB v23。只有全部 Gate 达标且操作员另行授权才允许生产切换。
6. 01-6 只读性能 Gate：使用目标 DB `EXPLAIN SELECT DISTINCT f.symbol FROM futures_trade_fills f JOIN futures_managed_orders m ... WHERE f.market='USDM' AND f.account_id=? AND f.trade_time BETWEEN ? AND ? AND f.intent IN (...) AND f.qty>0`；对 `position_id IN (...) + account_id` 的分页详情投影分别收集扫描行数、计划索引、执行时间 P50/P95、数据量/活跃周期规模。若有慢查询，**先评估并确定**复合索引，例如 `(market,account_id,intent,trade_time)`（仅候选，实际顺序依据 EXPLAIN），须另开 schema 迁移版本，通过 `sync db` 测试后另行授权生产更新，不在 v23 下偷偷 DDL。

## 验证边界

- 新增的 GO Controller 固定 JSON（5 个非默认 Accept × 3 类返回）测试、401 未认证保护、订单模型 source_ref 不泄漏测试、原 01-5 完整性测试需全部通过；前端 ESLint、TypeScript 类型检查和构建通过。
- 继承 `controllers` 全包中既有策略模板 fixture 红点：`default` DB alias 重复注册、空指标校验及 `strategy_templates/research/config.json` 不支持的 `family` 字段。本轮不对其它 AI 的策略研究文件进行修复/删除。
- 本轮不执行 `sync db`、不接触生产 `go_binance`、不发 Binance 委托、不修改 `conf/app.conf`、不覆盖后端 static、不重启运行中的程序，也不自动 commit/push。
