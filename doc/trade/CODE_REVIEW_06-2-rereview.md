# CODE_REVIEW_06-2：整改复评

日期：2026-11-11  
原报告：[CODE_REVIEW_06-2.md](CODE_REVIEW_06-2.md)（保留原始记录，不覆盖）  
目标：复核 F1～F7，按方案 06 的个人使用简化原则修复明确问题，不进入真实交易。

## 逐项处理

| 问题 | 处理 | 依据 |
| --- | --- | --- |
| **06-2-F1 / P1** 运行态 v25 Schema 门禁与旧复评描述矛盾 | **按用户此前明确指示，保留 `main.go` 无 `dbVersion >= 25` 特殊启动检查的设计，不重加门禁。纠正旧文档、代码注释，发布前强制执行 CLI Schema 验证；生产部署 Gate 仍阻塞。** | `command/spot_quant_migration.go` 的 `VerifyV25SpotQuantSchema` 注释已说明仅用于显式 `sync db`。`CODE_REVIEW_06-0_06-1-rereview.md` 顶部新增勘误，说明原先的“运行态拒绝启动”说法在删除校验后已失效。新增 [发布前 Schema 验证检查单](开发方案-06-发布前Schema验证检查单.md)。当前运行态只检查版本号，**不能**宣称运行态主动拒绝旧结构。原报告允许以发布 SOP 代替运行态检查，此处使用替代方案。 |
| **06-2-F2 / P2** 基线已过期 | **已修复** | 总方案把 `4af6d73` / `8f9a70b` / v24 标为 06-0 历史快照，新增 2026-11-11 实测 `feat/trade_spot` HEAD `af07191`、本地 master `7ed6c93`、代码 Schema v25。 |
| **06-2-F3 / P2** 开发页面未成套部署 | **明确边界，不提前发布** | [发布前 Schema 验证检查单](开发方案-06-发布前Schema验证检查单.md) 明确 06-2 是研发基线，隐藏前端路由与后端新增 API 并未交付当前部署；`dist→static`、后端程序及 DB 成套升级按 06-6 完成。当前未拷贝 `static` 或替换二进制。 |
| **06-2-F4 / P3** Futures-only 变量可被 Spot 接受 | **已修复** | `service/spotquant/validation.go` 对 Expr AST 标识符作禁用名单检查：`ROI`、`FundingRate`、`Position/Positions`、杠杆/保证金等即使在禁用规则中也拒绝；模板保存、JSON 导入、单币策略保存和试算都复用验证。`controllers/spot_quant.go` 已从模拟 env 移除 `ROI`。不误判字符串常量中的名称；未知 Spot 技术指标仍可保留、仅在预览时报告 `syntax_only`。 |
| **06-2-F5 / P3** 同名模板导入覆盖无确认 | **已修复** | `src/views/spotQuant/strategyTemplate.vue` 提交导入前按名称分页查出所有匹配项，精确同名时显示中英双语确认框；取消不调用导入接口，非同名仍直接导入。保留原有同名更新后端行为，无复杂审计。 |
| **06-2-F6 / P3** 批量设置币级 enabled 不查看全局开关 | **设计语义已冻结；真实门控待 06-4 验证** | 现阶段 `spot_quant_enabled` 仅是单币配置偏好，不触发 Binance 订单；允许在总开关 OFF 时预先配置。06-4 新单发送需**同时**满足全局 `config.spot_quant_enable=1` 与单币 enabled=1，再做资金、精度、幂等校验。总开关 OFF 仅停止发新自动单，不清空单币偏好。检查单中明确要求永久测试，06-4 未完成前不接真实交易。 |
| **06-2-F7 / P3** 无控制器二次 `authorized()` | **待 06-5 统一检查** | `/spot-quant/*` 未被 JWT 排除，全局 `middlewares.JwtMiddleware` 已保护；原审查未发现认证绕过。06-5 正式菜单和权限联动时再统一二次鉴权，避免本阶段引入冗余安全状态机。 |

## 新增测试

- `TestStage062SpotRejectFuturesOnlyIdentifiers`：多种期货专属变量、启用/禁用规则、字符串常量、未知 Spot 指标、JSON 模板导入。
- `TestStage062SpotPreviewRejectsFuturesOnlyVariables`：控制器返回 400，不能把期货账户变量表现为合法模拟信号。
- 保留全部 06-2 正向场景及 06-1 v25 `SyncDatabase` 缺表 fail-closed 回归。
- 前端同名导入确认使用现有 `getList` 和 `ElMessageBox`；取消不会调用导入接口。无需新增 API / 表。

## 安全与阶段 Gate

- **06-2 功能代码 Gate**：目标测试与前端 TypeScript 构建通过后可作为研发基线；不表示页面已在当前服务上部署。
- **06-1/06-2 发布前置 Gate**：`main.go` 仍未验证表结构，因此需要明确完成 [发布前 Schema 检查单](开发方案-06-发布前Schema验证检查单.md) 并保存成功记录；在此之前生产 Gate 继续 BLOCKED。不得将发布 SOP 描述成已执行。
- **06-3 研发**：可以继续本地代码及 SQLite/Fake 测试，不得跳过 06-4 的总开关权威检查或提前执行真实现货订单。
- **不涉及**：生产数据库、真实 Binance、`conf/app.conf`、`dist→static`、项目后端二进制、Git commit/push。

## 实际验证（2026-11-11，当前工作区）

| 命令 / 检查 | 实际结果 |
| --- | --- |
| `go test ./command ./controllers ./models ./service/spotquant -count=1` | PASS，4/4 包 |
| `go test -race ./command ./controllers ./models ./service/spotquant -count=1` | PASS，4/4 包；macOS 链接器 `malformed LC_DYSYMTAB` 为非致命警告 |
| `go vet ./command ./controllers ./models ./service/spotquant` | PASS |
| `go build -o /tmp/spot_quant_code_review_062_bin .` | PASS，不覆盖项目二进制 |
| `git diff --check`（后端） | PASS |
| `pnpm exec prettier --check src/views/spotQuant/strategyTemplate.vue src/api/spotQuant.ts src/router/modules/spotQuant.ts` | PASS |
| `pnpm exec vue-tsc --noEmit --skipLibCheck --pretty false` | PASS |
| `pnpm exec vite build --outDir /tmp/spot_quant_review062_build --emptyOutDir` | PASS，隔离临时目录，未同步服务端静态文件 |
| `main.go` 中特殊 v25/Schema 运行态检查 | **0 处**，符合用户此前删除要求 |

**无关工作区注意事项**：前端全仓库 `git diff --check` 指出已有未提交的 `src/views/order/index.vue:659` 空白行带尾随空格。本次不更改该订单页，也不把其格式问题计入 Spot 06-2 的缺陷。历史全仓策略研究目录和依赖环境的测试问题，本次不声称已经消除。

验证无生产库连接、无 `sync db` 调用、无真实 Binance 订单、无现有服务重启；本次修改未提交/推送。
