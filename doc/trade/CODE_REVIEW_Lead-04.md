# CODE_REVIEW_Lead-04：真实资格、双账户限额与发布验收

> **状态：Lead-04 进行中。本轮完成 SAPI/Futures API 权重分类、10/30/60 币白名单筛选的本地回归，以及复用已有的 Lead 只读验证能力；真实 Lead Portfolio 与交易所成交未验收。**
>
> 个人使用：无第二套策略/订单系统、无新 UI、无额外配置。不修改 `app.conf`、生产 MySQL、服务或实盘 Gate。

## 1. 核对的官方规则

- Copy Trading 两个只读端点：`GET /sapi/v1/copyTrading/futures/userStatus`、`GET /sapi/v1/copyTrading/futures/leadSymbol`，签名请求，IP Weight 1。`isLeadTrader=true` 仅验证带单交易员身份，**不返回 Portfolio ID**，不能证明标签对应特定 Portfolio。官方：https://developers.binance.com/en/docs/catalog/advanced-trading-copy-trading/api/rest-api/future-copy-trading
- 带单 Portfolio 专属 API Key 需在币安 My Lead 组合中创建，启用 Futures/IP 限制。只支持 USDT；不支持 Multi-Assets；Lead API 订单限额 20 次/10 秒。官方：https://www.binance.com/en/support/faq/detail/2bec848b904b422197ce121d0925f20b
- SAPI IP 权重头 `X-SAPI-USED-IP-WEIGHT-1M` 与 Futures `X-MBX-USED-WEIGHT-1M` 属于不同限额池。SAPI UID Weight 不能当作 Futures Order Count。官方：https://developers.binance.com/en/docs/products/advanced-earn/general-info

## 2. 本轮变更

| 文件 | 说明 |
| --- | --- |
| `service/binanceapiusage/transport.go` | 同一个 Collector/BudgetCoordinator 根据 `/sapi/` URL 分类为 `Product=sapi`；读取对应 SAPI IP 响应头。`/fapi/` 保持 Futures 原逻辑；SAPI 429 不错误阻断 Futures；不记录签名或 Key |
| `service/binanceapiusage/collector.go` | SAPI IP 参考上限 12000/分钟，用作**保守的统一 IP 预算参考**，实际以对应端点和交易所响应为准 |
| `service/leadaccount/metrics.go` | 原 Lead Usage 新增 Futures/SAPI 各自请求量和估算权重，旧字段保留兼容，但跨限额池的合计不能当作单一配额 |
| `service/binanceapiusage/transport_lead04_test.go` | Mock HTTP 证实不同响应头正确分组、SAPI 429 只影响 SAPI 预算、不会泄漏签名；**Main+Lead 共用 Futures IP 预算**，Main 高权重后 Lead 普通行情请求被延期 |
| `feature/lead_stage04_limit_test.go` | 10/30/60 白名单候选币，分别重复 100 次筛选，SAPI Loader 各仅请求 1 次；Main Client 无法冒充 Lead |
| `service/leadaccount/metrics_stage04_test.go` | Lead Futures、Lead SAPI 与 Main 请求在指标中正确归属 |

已有 `POST /futures/lead/verify` 调用 `Manager.VerifyReadOnly`，执行独立 Lead 签名 SAPI、账户 CanTrade、USDT、挂单、仓位和 Hedge Mode 检查；没有必要重新开发接口。`PortfolioBindingConfirmed=false` 仍是事实：两个 SAPI 端点不提供 Portfolio ID，必须与 Binance 带单组合 API Key 来源核对，不能依靠用户输入的标签。

## 3. 已获得证据 vs 尚缺证据

**本地证据：** SAPI/Futures 分类及退避 Mock 测试通过；10/30/60 币各 100 轮本地筛选只触发首次一次白名单加载；Stage 1–3 的 Main/Lead 账户、订单、Fill 隔离测试保留。此筛选时间仅为 CPU 局部时间，**不是完整交易循环 P50/P95，也不是交易所延迟**。

**真实环境缺口：** 这个开发终端的 `BINANCE_LEAD_CREDENTIAL_KEY`、`BINANCE_LEAD_CREDENTIAL_FILE`、`BINANCE_LEAD_CONFIG_FILE`、`BINANCE_LEAD_OBSERVER_MODE` 均未设置，故没有真实 Lead Portfolio 的签名 REST/WS 结果。不能推断部署服务也没有凭证，更不能回退 Main Key。尚缺 Main 单独/Lead 单独/同时运行时的真实 1m/5m 权重、真实 429/418、主网 Fill、MySQL 8 EXPLAIN 与服务重启证据。

## 4. 本地验收命令

- `go test ./service/binanceapiusage ./service/leadaccount ./feature -run '^TestLead04' -count=1 -p=2`：**EXIT=0**。
- `go test ./feature -run '^TestLead04' -count=1 -p=2 -v`：10/30/60 币各 100 轮，`sapi_requests=1`。
- `./scripts/test-go.sh -p=2 -timeout=120s`：**EXIT=0**，完整 Go 应用包回归通过。
- `go test -race ./feature ./service/leadaccount ./service/binanceapiusage ./feature/api/binance -run '^TestLead04|^TestLead03|^TestLeadMainParity|^TestStage54Lead|^TestStage7' -count=1 -p=2`：**EXIT=0**，无 Data Race（macOS 链接器有非阻断警告）。
- `go vet ./feature ./service/leadaccount ./service/binanceapiusage ./feature/api/binance ./service/futuresownership`：**EXIT=0**。
- `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/go_binance_futures_lead04_arm64 .`：**EXIT=0**，产物核验为 ELF aarch64 静态可执行文件，**只编译，未部署**。
- `git diff --check` 与本轮新增 Go 文件 `gofmt -l`：通过。

## 5. 下一步必须真实验收

1. 在实际服务原有 Lead 管理页保存**该组合专属** API Key 后执行“验证”。比对币安组合页面的 API Key、IP 白名单及 Enable Futures，不输出 Secret。
2. 原有只读 WS 检查健康、仓位、订单和账户隔离。如果没有真实 Fill，则明确标记未验证，不能用普通 Testnet 账户替代 Lead。
3. 现有 V4-5 面板记录 Main/Lead 单独与并行的 1m/5m 请求、SAPI/Futures 响应头、延迟与 429/418；MySQL 仅 SELECT/EXPLAIN。
4. 所有真实安全 Gate 通过且用户**另行明确授权**后才能考虑小额实单。`LeadDurableOpenGate`、`/futures/lead/start`、`allow_new_opens=false` 目前均保持锁定；本次构建不自动开启交易。

**结论：Lead-04 第一轮本地开发完成，整体仍进行中，真实实盘验收未完成。**
