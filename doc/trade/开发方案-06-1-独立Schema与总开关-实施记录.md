# 06-1：独立 Schema 与总开关——实施与验收记录

> 执行日期：2026-11-11  
> 工作分支：`feat/trade_spot`；源基线：`master 8f9a70b`、Schema v24。  
> 结果：**06-1 已完成（本地实验库 `go_bn_test`）**；后续 06-2～06-6 尚未执行。  
> 约束：**未连接生产数据库、未执行 Binance 真实订单、未修改 `app.conf`、未修改前端未提交文件、未 commit/push**。

## 一、已完成的改动

| 文件 | 变更 |
| --- | --- |
| `appversion/version.go` | Schema 需求版本从 v24 升为 **v25** |
| `models/tableStruct.go` | 新增 `Config.SpotQuantEnable` → `config.spot_quant_enable`，默认 0 |
| `models/spot_symbols.go` | 新增 `SpotSymbols.SpotQuantEnabled` → `spot_symbols.spot_quant_enabled`，默认 0；不复用旧 `enable` |
| `command/db_update.go` | v25 专属迁移入口，以及 MySQL 开关 `TINYINT` 类型校验 |
| `command/spot_quant_migration.go` | 只新建四张现货业务表，确保索引和字段；对 MySQL/SQLite 提供对应自增主键 |
| `models/spot_quant.go` | 四种 Spot 记录的 Go DTO；资产金额以字符串传递，禁止 ORM 自动同步财务 DECIMAL 表 |
| `controllers/index.go` | `GET /service/config` 新增 `data.spotQuantEnable`，`PUT /service/config` 接收 `{"spot_quant_enable":0/1}` 并拒绝非法值 |
| `controllers/index_service_config_test.go` | 新增总开关校验、与旧 Spot/抢新/Futures 配置隔离的测试 |
| `command/db_update_test.go` | v24→v25 SQLite 测试：开关默认 OFF、业务表、DECIMAL、关联/唯一索引、写入/重复 Fill、重复迁移、旧 Spot/Futures 数据保持不变 |

**实现方式**：两处开关继续由现有 Beego ORM 管理；MySQL ORM 默认生成 `INT(11)`，随后以 `ensureV25SpotQuantFlagTypes` 校正为 `TINYINT NOT NULL DEFAULT 0`。四张业务表使用显式 DDL，避免 ORM 把金额字符串列从 `DECIMAL(36,18)` 改成 `VARCHAR`。

## 二、v25 新增数据结构

| 表 | 目的 | 关键约束 |
| --- | --- | --- |
| `spot_managed_positions` | 一次 BUY→SELL 策略持仓周期 | `id` 主键；`symbol,status` 联合索引；数量/成本/已实现盈亏均为 `DECIMAL(36,18)` |
| `spot_managed_orders` | 策略委托与持仓 `position_id` 关联 | `client_order_id` 唯一、`symbol+exchange_order_id` 唯一；`position_id` 索引；委托/成交数量用 DECIMAL |
| `spot_trade_fills` | 逐笔实际成交及手续费 | `symbol+exchange_order_id+trade_id` 唯一、`position_id` 与本地 `order_id` 索引；数量、价格、报价额、手续费使用 DECIMAL |
| `spot_strategy_templates` | 现货独立策略模板库 | 独立 `id,name,technology,strategy,create_time,update_time`，不使用 Futures 模板表 |

所有四表不增加多账户、Owner、审核/认领/模式状态机；与 Futures 表没有共享写入。现阶段只建立物理数据基础，**不创建新下单入口、自动扫描、账户页面或策略模板 CRUD**；这些分别属于 06-2～06-5。

## 三、数据库升级及差异核查（仅实验库）

在实际升级前使用只读 SQL 核对了目标数据库名称为 `go_bn_test`、Schema v24，四张新表与两处新列均不存在。新代码编译到独立临时可执行文件，在包含当前 `conf` 与迁移目录的隔离工作目录里执行项目标准命令 `./go_binance_futures sync db`。**未替换项目原有二进制文件**，避免干扰正在运行的其他工作。

- 首次升级日志：`database sync start, target version: 25` → `24 -> 25` → `database sync completed`。
- 随后再次执行 `sync db`，结果为 `database version is already up to date: 25`。
- MySQL 专门校验与类型修正后，两处开关列均为 `tinyint(4) NOT NULL DEFAULT 0`（即 MySQL `TINYINT`），再执行一次迁移，结构/行数完全一致。
- 新表数量 **4**，其中 **10 个财务金额/数量字段**为 `DECIMAL(36,18)`；客户端订单 ID 与成交标识唯一性已验证。
- 不含新增字段的旧表 Schema 未发生变化；原有旧索引 46 条元数据记录不变；未删除或修改已有 Futures 表字段。

| 既有表 | v24 行数 | v25 行数 |
| --- | ---: | ---: |
| `config` | 1 | 1 |
| `spot_symbols` | 809 | 809 |
| `strategy_templates` | 59 | 59 |
| `futures_managed_positions` | 0 | 0 |
| `futures_managed_orders` | 0 | 0 |
| `futures_trade_fills` | 0 | 0 |

说明：这些数字仅针对 **`go_bn_test`**；该实验库 Futures 业务表原本为空，生产真实仓位未访问。迁移前后旧表结构及行数一致，不等同于跨库的全面数据审计。

## 四、测试结果

| 验收项 | 结果 |
| --- | --- |
| `go test ./command ./controllers ./models -count=1` | **PASS** |
| 临时 SQLite 顺序升级至 v25、重复执行 | **PASS** |
| SQLite 自动关闭、旧 `spot_symbols.enable` 不变、旧 Futures 种子数据不变 | **PASS** |
| SQLite Spot 持仓/订单/成交插入、唯一约束与重复升级 | **PASS**（临时数据库） |
| MySQL `go_bn_test` v24→v25 实际迁移 | **PASS** |
| MySQL 资产数量和手续费高精度 DECIMAL 插入/读取 | **PASS**：18 位小数原样保留 |
| MySQL 重复 ClientOrderID 与重复 Fill | **PASS**：数据库均拒绝 |
| MySQL 实际插入事务回滚 | **PASS**：四张新表无测试残留行 |
| MySQL 升级后重复 `sync db` 幂等 | **PASS**：v25 与表结构、行数保持一致 |
| `go vet ./command ./controllers ./models` | **PASS** |
| `git diff --check` | **PASS** |
| 应用代码 Go 编译检查（排除历史策略研究目录） | **PASS**：83 个应用包，`go test <应用包列表> -run '^$' -count=1` |
| 全仓库 `go test ./... -run '^$' -count=1` | **未通过**：历史 `strategy_templates/research/...` 中原有重复符号/构建报错，与 06-1 代码无关，未擅自改动 |

MySQL 测试通过事务内插入后 `ROLLBACK`，未向业务表留测试订单；**没有调用 Binance REST/WS 下单接口**。

## 五、范围冻结与后续

- 现货量化配置总开关和每币开关初始均为 **0**。此时系统**没有**新的 Spot 策略扫描器或交易执行器，故不存在通过新增开关直接触发下单的路径。
- `GET /service/config` 暴露 `spotQuantEnable`；`PUT /service/config` 可独立写 `spot_quant_enable`（仅 0 或 1）。现货交易页单币种启停与模板接口属于 **06-2**，配置中心的独立卡片属于 **06-5**。
- `spot_symbols.enable`、`config.spot_enable`、`config.spot_new_enable` 等历史功能不受本次新增开关影响。
- 06-2 从新现货表读取币种与策略，严禁复活旧方案 02 或调用 Futures 下单接口。
- 06-3 使用本阶段新表建立仓位/订单/Fill 生命周期；业务事务、部分成交、重启恢复和 PnL 均**尚未实现**，不能把 Schema 完成当作实盘可用。
- 生产库升级、前端构建/部署、任何真实小额现货买卖，均需单独安排并核验，不属于本次实施。

**回滚提示**：项目迁移是向前升级，MySQL DDL 不是事务性的；当前没有自动降级脚本。需要回退时必须先备份并人工核对 `go_bn_test` v25 记录及新表是否有真实数据，不能在后续阶段数据落库后直接 DROP。不得将实验库的变更误用到生产。

## 六、2026-11-11 Code Review 整改补充

根据 [CODE_REVIEW_06-0_06-1.md](CODE_REVIEW_06-0_06-1.md) 的 06-F1 阻塞反馈，本阶段新增了 **v25 Schema 只读验证与拒绝启动门禁**，详见 [整改复评](CODE_REVIEW_06-0_06-1-rereview.md)。

- 对已标记 v25 的库，`sync db` 必须确认 4 张独立业务表、两处开关列及 10 个精确 DECIMAL 字段存在；缺失则明确失败，不能以版本号代替物理完整性，不静默新建空成交表。
- 正常进程启动前也执行只读 Schema 检查，阻止旧方案 02 同号 v25 库进入运行态。
- 配置 GET/PUT 使用一致的 `spot_quant_enable`；错误字段名及混合修改明确拒绝；专用更新仅修改该列，GET/PUT 返回 JSON 而不受 `Accept` 协商影响。
- 补缺表永久回归、SQLite 数据持久化隔离测试与多种 `Accept` 测试；3 包 Go 单测、race、vet、主程序 build、`git diff --check` 全部通过；`go_bn_test` 只读验证 PASS。
- 原有全仓库策略研究代码红点、MySQL 8 生产库权限/迁移预检、06-3 写入层 `side/status` 校验未在本阶段提前处理。

以上整改未改 `app.conf`，未执行新的真实业务库升级，不碰生产仓位，也未进行 commit/push。
