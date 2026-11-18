# CM-2 公共 DAPI 响应证据（本机缓存核对）

> 文件捕获时间（UTC，取本机文件修改时间）：2026-11-17T14:29:42+00:00  
> 本次哈希与字段核对时间（UTC）：2026-11-17T15:27:38+00:00  
> 来源：\`https://dapi.binance.com/dapi/v1/exchangeInfo\`，公开 GET，无签名与交易操作。

| 项目 | 已核对结果 |
| --- | --- |
| 首次原请求 HTTP | 200（依据首次实施时终端输出，非本次重新发起成功） |
| 本机原始 JSON 缓存 | \`/tmp/cm2_coinm_public_exchange.json\` |
| 原始响应字节数 | 35700 |
| SHA256 | \`fd11a913ba8314b0075bfd070c89e51587a8a52e432c408be4cda69c521693a4\` |
| JSON \`symbols\` 总数 | 30 |
| \`PERPETUAL + TRADING + USD + _PERP\` 数量 | 20 |
| BTCUSD_PERP 合约面值 | 100 USD/张 |
| BTCUSD_PERP 保证金币种 | BTC |
| BTCUSD_PERP PRICE_FILTER tickSize | \`0.1\` |
| BTCUSD_PERP LOT_SIZE min / max / step | \`1\` / \`1000000\` / \`1\` |
| BTCUSD_PERP MARKET_LOT_SIZE min / max / step | \`1\` / \`60000\` / \`1\` |

**证据等级与局限**：本次复核了在首次 CM-2 实施期间真实获得的原始公开响应缓存；这解决了“原始响应不可复查”的部分问题，但不是独立成功的第二次实时访问。此次尝试重访 \`dapi.binance.com\` 分别遇到 SSL 连接超时和代理 TLS 连接失败，不能声称重新联网成功。公开响应不是实时市场价格或签名交易验收证明。缓存为本机临时文件，不作为项目依赖。
