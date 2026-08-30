# 08. 总体验收清单

## Runtime

- [ ] 所有 Skill 共享同一个 Runner。
- [ ] Runtime 不依赖具体 Controller。
- [ ] 支持 Tool/Final/Repair/Retry/Cancel/Timeout/MaxRounds。
- [ ] Task 之间上下文完全隔离。
- [ ] Tool 重复调用、超大结果和死循环有硬限制。

## Tool / Service

- [ ] Agent 与 MCP 共享 Domain Service，而不是内部 HTTP 自调用。
- [ ] Tool 有输入/输出校验、超时和结果大小限制。
- [ ] Skill 只能调用白名单 Tool。
- [ ] read/write/trade 权限边界明确。
- [ ] 第一版 Agent 没有直接真实下单权限。

## Market Regime

- [ ] 现有市场环境 API/前端契约保持兼容。
- [ ] LLM 不可用时算法 fallback 正常。
- [ ] 同一时间不会重复运行市场环境分析。
- [ ] 结果包含 source/confidence/reason。

## Strategy Builder

- [x] 原多轮工具调用和 Repair 行为迁移到 Runtime。
- [x] 策略 JSON 继续执行真实业务校验和 expr 编译检查。
- [x] “AI 生成成功”和“导入数据库”仍是两个动作。
- [ ] 续聊使用 Conversation 概念，不依赖内存 Task ID 语义。
## Symbol Analysis

- [x] 指定任意有效 USDT 合约可以启动分析。
- [x] 数据获取失败进入 `data_missing`，模型不会伪造。
- [x] 输出为版本化 TradingPlan Schema。
- [x] neutral 是合法结论，不强迫做多/做空。
- [x] 所有证据均可追溯到 Context/Tool。

## Event / Alert

- [x] WS Tick 不直接调用 LLM。
- [x] Event Bus 消费失败不会阻塞 WS 回调。
- [x] Signal Engine 完成去重、阈值、聚合和冷却。
- [x] 行情极端时有 Agent 并发/频率保护。
- [x] AI 关闭或失败时基础报警仍工作。

## Scheduler / Store

- [ ] AI 周期任务通过统一 Scheduler 触发。
- [ ] `skip_if_running` 防止任务重叠。
- [ ] Task 历史可持久化并在重启后查询。
- [ ] Task、Conversation、Memory 概念分离。

## 安全与运行

- [ ] 日志和 Task 数据不保存 API Key、Token、密码。
- [ ] LLM/Tool 故障不会影响核心交易循环和 WebSocket。
- [ ] 每个 Skill 有独立开关，可以快速回滚。
- [ ] 有成功率、耗时、Token、Tool/Validator 错误等指标。
- [x] FastMove / Liquidation 新旧路径完成灰度对比，legacy 检测与直接通知实现已删除。

## Definition of Done

整个 Agent 项目第一版完成的标准不是“AI 能回答问题”，而是：市场环境、策略生成、单币分析、事件报警四项能力已经共享统一 Runtime/Tool/Task 基础设施；任何一个 Skill 都可以独立关闭；LLM 异常不会破坏核心交易程序；新增第五个 Skill 时无需再复制 Agent Loop、Task Store、Retry、Tool 执行和进度管理代码。