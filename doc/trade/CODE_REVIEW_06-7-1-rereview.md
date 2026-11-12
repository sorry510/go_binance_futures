# CODE_REVIEW_06-7-1：整改复评

日期：2026-11-12  
后端：`feat/trade_spot`，HEAD `32d203b`；**全部本轮改动仍为本地未提交工作区**  
原始报告：[CODE_REVIEW_06-7-1.md](CODE_REVIEW_06-7-1.md)（保持不变）  
阶段记录：[06-7-1 现货策略规则评估实施记录](开发方案-06-7-1-现货策略规则评估-实施记录.md)

**复评结论：F2/F3/F5/F6/F7 代码问题已修复，F1 选择保留合约已有的 ROI 阈值语义并通过文档、回归明确约束，F4 安全偏离已记录；F8 的自动轮询幂等仍属于 06-7-2，F9 保留精简架构、F10 保留干净副本与工作区区别。06-7-1 离线研发 Gate PASS；06-7-2 自动轮询及 Mainnet/Testnet/生产 Gate 继续 BLOCKED。**

## 一、逐项整改

| 问题 | 级别 | 处理结果 | 永久回归/证据 |
| --- | --- | --- | --- |
| F1 默认 `profit/loss=100` 导致 `close_long` 很少被评估 | P2 | **保留合约语义，文档明确并固定回归，风险需使用者配置**。新币的 100/100 确切表示 +100%/-100% ROI 阈值，`close_long=true` 是门控许可，**不是立即卖出**。没有未经授权地改成 8/6，也没有修改已有配置。方案 §3.2 明确先判断 profit/loss 再评估 close_long；启用自动交易前必须在币种配置核实阈值 | `TestReview0671DefaultROIThresholdIsExplicitPermissionGate`：ROI +10% 不卖、+101% 卖；新增 `service/spotquant/repository.go` 的默认值注释 |
| F2 `EvaluateExit` 可接收任意伪造 Snapshot | **P2** | **已代码修复**。公开签名改为 `EvaluateExit(coin, *spotledger.Ledger, positionID)`，内部 **强制调用 `Ledger.Accounting(positionID, "")`** 读取不可变 Fill 投影；外部不能再注入物化仓位数字。仅私有 `evaluateExitSnapshot` 用于表达式纯单元测试，不是业务 API | 新增 `TestReview0671EvaluateExitCannotTrustCallerMaterializedValues`：真实 SQLite Ledger、真实 BUY Fill；篡改本地可修改物化仓位为 qty=999/avg=1，退出 ROI 仍是 **10%** 而非 10900%；nil Ledger / 非法 ID 被拒绝 |
| F3 不可达的财务检查分支与不真实的测试 | P3 | **已修复**。删除 `!FullyPriced && len(UnpricedFees)==0` 的不可能状态伪检查，删除相应人工 Snapshot 测试。真正的账本冲突通过 `Ledger.Accounting` 的 `ErrNeedsReview` fail-closed | `TestReview0671EvaluateExitFailsClosedOnFillInconsistency`：真实 DB 中故意制造 Fill Qty 与订单矛盾，`errors.Is(err, spotledger.ErrNeedsReview)` 且无 SELL |
| F4 与 Futures 存在安全偏离 | P3 | **已文档化，保留安全设计**。现货无行情时拒绝 SELL，而 Futures 原 fallback 可能 fail-open；现货空阈值禁用，Futures 空阈值可能解析为 0；现货规则编译错误终止本轮，Futures 继续其他规则。无必要照搬可能误卖的行为 | 已在方案 §3.4、实施记录「整改补充」写明 |
| F5 日/周/月 Kline 两周期宽限过大 | P3 | **已修复**。提取 `verifySpotKlineRows`，按**一个实际周期+5min** 判断最后一根 Kline 的 OpenTime，增加 CloseTime 与 OpenTime 的一致性检查，拒绝超过窗口的历史行情 | `TestReview0671SpotKlineFreshnessByInterval`：日线 20h 通过/26h 拒绝、周线 6d 通过/8d 拒绝、月线 20d 通过/60d 拒绝；无效 CloseTime 拒绝 |
| F6 指标样本检查过弱 | P3 | **已修复**。不再只要求某一个 slice 非空：MA/EMA/RSI 检查 Data；MACD、ADX、KDJ、Supertrend 等检查应有的多个完整向量长度、数值有限；根据配置的 Period/SlowPeriod/SignalPeriod 检查实际原始 Kline 数量达到最低需求 | `TestReview0671SpotIndicatorSampleCompleteness`：MA5 只有 4 根 K 线拒绝，MACD 仅 DIF 拒绝，波带缺中/下轨拒绝，NaN、长度不一致拒绝 |
| F7 `UnrealizedProfit` AST 属性例外不区分宿主 | P3 | **已修复**。仅当成员链根为经过 Spot 成交构造的 `Position` 或 `Positions` 时允许该属性；对 `Other.UnrealizedProfit` / `Other["UnrealizedProfit"]` 拒绝，继续拒绝 `Leverage` 等 | `TestReview0671SpotUnrealizedProfitHostScope` |
| F8 06-7-2 未接入双开关、仓位校验和稳定 client ID | **P2 / 下一阶段** | **边界明确，未冒充已修复运行时**。给 5m 兜底卖出增加固定规则身份 `spot_simple_close_5m_v1` 与非空 RuleHash。方案 §3.3/3.4 要求 06-7-2 采用**持久化仓位周期、币种、方向、K 线时间与旧订单**判断幂等，而不能只靠可修改的规则 hash；20 轮重复、改策略、重启、UNKNOWN、双开关与活动仓位必须专测 | 默认平仓 RuleHash 由 `TestStage0671SpotDefaultCloseMatchesFuturesPriorityButUsesSpotKlines` 断言。自动 Runner **尚不存在，F8 接线 Gate 继续 BLOCKED** |
| F9 指标层与 Spot Ledger / 配置层的耦合 | P3 | **有意暂不重构**。当前只多一处单向依赖、没有 import cycle，不为个人系统引入新包或新的抽象层。F2 通过强制读取 Ledger 投影消除了核心数据来源风险；有实际循环依赖或复用压力时再调整 | Go Vet / Build / 包单测回归 |
| F10 当前工作区全仓与干净副本测试混用 | P3 | **区分状态，不改历史文件**。项目用户工作区有私有 `strategy_templates/research`，其中可能有重复 Go 定义；应用脚本跳过归档。另建干净 `git archive HEAD` 加本轮未提交 Go 文件覆盖的临时副本，只在该副本运行 `go test ./...`；报告分别记载 | 测试结果见下节 |

## 二、实际执行的测试

| 测试 | 实际结果 |
| --- | --- |
| `go test ./feature/strategy/line ./service/spotquant -run 'TestReview0671\|TestStage0671\|TestStage062SpotRejectFuturesOnlyIdentifiers' -count=5 -p=2` | **PASS** |
| `./scripts/test-go.sh -short`（当前工作区，不包含私有研究归档） | **73 个有测试的包 PASS、0 FAIL** |
| `go test -race ./feature/strategy/line ./service/spotquant ./service/spotledger ./service/spotexecution ./feature ./controllers ./spot ./spot/api/binance -count=1` | **8/8 PASS**；macOS 老的 `LC_DYSYMTAB` 警告不影响退出码 |
| `go vet` 上述 8 包 | **PASS** |
| `go build -o /tmp/spot0671_review_final .` | **PASS**；未覆盖正在运行的程序 |
| `git diff --check` | **PASS** |
| 干净 `git archive HEAD` + 本轮 Go 文件覆盖，`go test ./... -count=1` | **PASS：73 个有测试的包、13 个无测试文件的包、0 FAIL**，隔离目录 `/tmp/spot0671_rereview_clean.bjex3l`；不与当前工作区结果混称 |
| Mainnet / Testnet 下单、生产数据迁移、部署、自动轮询 | **均未执行** |

## 三、06-7-2 开发准入条件（不能由 06-7-1 代替）

- **先检查 DB 双开关与现有订单**：`EvaluateEntry` 是纯信号层，只知道当前 symbol；`StartSpotQuantTrade` 必须先确认 `spot_quant_enable`、`spot_quant_enabled`、同币唯一活动仓位以及所有 NEW/UNKNOWN 委托。
- **只用已核对 Fill 生成 SELL 信号**：`EvaluateExit(coin, ledger, positionID)` 是唯一可用的公开签名；账本失败即跳过该币。
- **每个策略事件的订单必须稳定且不可重复**：规则 hash 用作诊断元数据，不应单独作为 clientOrderID；重复 2 秒轮询 20 次、改规则、重启、同根 K 线完整买卖后的新周期都要证实不重复 POST，不能绕过旧 UNKNOWN。
- **默认阈值应在启用前由用户确认**：当前 100/100 意味着涨跌幅 ±100% 才进入退出判断。不要在 06-7-2 悄悄改默认值。
- **真实交易安全硬 Gate 不变**：Mainnet adapter 禁止发送订单，Spot Testnet 账户曾 -2015；没有另外授权之前不打开主网、生产数据库或服务器部署。

## 四、最终边界

**06-7-1 修复审查暴露的可在本地解决的代码安全与数据校验问题，保持与 Futures 退出规则顺序一致；F8 等运行时门禁属于尚未开始的 06-7-2，方案 06 整体不能认定实盘完成。**

本轮未修改 `main.go`、`conf/app.conf`、版本 Schema、前端或 `static`，没有执行数据库迁移、真实下单、服务器部署、自动 Git commit/push；原始 `CODE_REVIEW_06-7-1.md` 保持不变。
