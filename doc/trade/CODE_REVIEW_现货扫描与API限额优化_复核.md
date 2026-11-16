# CODE_REVIEW_现货扫描与API限额优化：修复复核

对照原始 [CODE_REVIEW_现货扫描与API限额优化](CODE_REVIEW_现货扫描与API限额优化.md)，对 2026-11-16 本地工作区的 7 项 Finding 逐项核查、做安全范围内的修复和回归。原报告未覆盖或修改，保留原始审计判断。

## 处理结论

| Finding | 复核状态 | 已完成及剩余边界 |
| --- | --- | --- |
| F1 · P1 · 双开关不再禁止 Managed SELL | **产品语义按用户既定要求保留；生产 Gate 未闭环** | 用户此前明确要求关闭币种后仍能按策略平仓。本次没有恢复原闸门，也没有擅自新增 kill switch / 持久化授权 / append-only 审计。关闭开关不等于停机，已在中英文 03 文档说明；主网许可、独立紧急停机设计仍须另行处理 |
| F2 · P2 · 空闲 Runtime 与永久异常轮询 | **已缓解，运维入口未闭环** | 签名 WS 增加 15 秒工作检查；无启用币种、Managed 持仓与未结订单时关闭 WS 并暂停 serverTime/重连，仅低频查本地 DB；开关 OFF 且持续 ErrNeedsReview 时按 5→10→20→30 秒退避，仍保留后续恢复机会。异常订单需要人工检查，尚无独立修复 UI |
| F3 · P2 · WS 未 ready 无 REST-only SELL | **保持 fail-closed，补回归/文档** | 签名 WS 未确认时仍禁止自动 SELL；REST 只读对账不受影响。没有引入绕过 WS 的新下单路径。OFF+Managed+WS Down → 不下单，WS 恢复 → 可按验证仓位退出，新增测试通过；不存在额外的受控 REST-only 退出 UI |
| F4 · P2 · 时间源自证 | **已修复** | 快照 AsOf 改为 Binance ticker 的 CloseTime，主网 WS 最新 Price 不再将 24h OHLC 的 AsOf 改为本地当前时间；策略验证 now 独立取 time.Now()。补充模拟 24h 统计 4 分钟接受、6 分钟拒绝的策略集成测试 |
| F5 · P3 · 缓存细节 | **LRU 已修复，其余部分未改** | 快照/K线的缓存命中更新最近访问时间，超容量时优先淘汰过期记录、否则淘汰最久未使用记录；新增容量和热数据存活测试。singleflight 首调用者 ctx、旧只读构造函数的 endpoint 约束及每轮指标缺口均未改变 |
| F6 · P3 · 历史周期 DB 全量 Fill 投影 | **未解决，需真实负载分析** | 原实现故意依赖 Fill 证据而非可人工修改的状态缓存；不能为减少 DB 开销直接删去历史安全核对。仍需 MySQL 8 实际规模 EXPLAIN / P95 证据后再决定索引、范围查询或安全缓存 |
| F7 · P3 · 文档与回归缺口 | **部分闭环** | 两语种文档补充“OFF ≠ 紧急停机”、WS fail-closed、OFF 异常退避、缓存 LRU；新增 WS Down、OFF 异常恢复、空闲 WS、独立时间源及 LRU 专项测试。缺少实盘 weight / 多币 P95 基线、REST-only 审批恢复场景 |

## 具体实现

- `feature/spot_quant_runtime.go`：空闲时关闭签名 WS、停止无效的 serverTime REST / 重连；状态检查用独立 ORM 防止多个 goroutine 共享 Runner 的 Ormer；满足新工作条件后恢复 WS。
- `feature/spot_quant_runner.go`：仅当开仓开关 OFF 且恢复出现 `ErrNeedsReview` 时启动最多 30 秒的指数退避；成功或重新启用时清除记录。交易策略与 Fill 归属检查保持不变。
- `spot/api/binance/spot_quant_market.go` 和 `feature/spot_quant_market_strategy.go`：使用 Binance 实际 24h 统计时间与独立当前时间进行新鲜度校验，主网 WS 覆盖价格时不覆盖统计时间。
- `spot/api/binance/spot_quant_market_cache.go`：有界 LRU 替代随机淘汰，有并发保护；并发合并、TTL、跨周期和主/测试网缓存隔离规则保留。
- `feature/spot_quant_review_test.go`、`spot/api/binance/spot_quant_market_cache_test.go`：新增复核测试；已有按旧语义断言每次 OFF 都产生异常的集成测试，更新为允许限流冷却期，但仍检查订单/Fill 安全性。
- `doc/readme/zh-CN/03-现货量化交易.md` / `doc/readme/en/03-spot-quant.md`：同步新语义、风险与恢复说明。

## 测试证据

- `./scripts/test-go.sh -p=2 -timeout=120s`：**EXIT=0**，应用包无失败。
- `go test -race ./feature ./service/spotexecution ./spot/api/binance -run 'TestSpotReview|TestSpotQuantMarket|TestSpotQuant0673DisabledGate|TestSpotQuant0672Switches' -count=1 -p=2`：**EXIT=0**，未报告 DATA RACE（macOS 链接器存在非阻断 warning）。
- `go vet`（Spot / Strategy / Ledger 相关包）：**EXIT=0**。
- `git diff --check`：**EXIT=0**。
- 测试使用模拟 HTTP、临时 SQLite 与 fake executor；没有执行真实 Spot Testnet/Mainnet 下单。

## 发布与验收边界

本复核不替代原报告和 `CODE_REVIEW_06-7-ALL.md` 的发布要求，**不能据此宣称可直接在主网部署**。仍缺独立紧急停机控制、原有主网授权契约的明确更新、真实 MySQL 8 性能、ARM Linux 成套发布/回滚和真实多币/API 权重场景验证。主网真实资金不应以离线测试全绿作为唯一放行依据。

本次只修改本地源代码、测试与中英文模块文档；**未修改生产 DB、conf/app.conf、PM2、运行实例，也未发送真实交易订单、Git commit 或 push**。
