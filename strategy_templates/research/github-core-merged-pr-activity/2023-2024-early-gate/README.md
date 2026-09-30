# GitHub Core Merged-PR Activity — 2023–2024 Early Gate

Hypothesis: sustained core-development activity in a token's canonical client/node repository may be a protocol-delivery signal distinct from release announcements.

Repository mapping is reused unchanged from the prior GitHub Core Release Catalyst: BTC bitcoin/bitcoin, ETH ethereum/go-ethereum, BNB bnb-chain/bsc, XRP XRPLF/rippled. GitHub Search merged-date qualifiers use server-side merge timestamps, avoiding commit-author timestamp look-ahead.

Monthly signal: activity = log(1 + current-month merged PR count) - mean(log(1 + count) over the previous 3 completed months). activity > 0 => LONG, activity < 0 => SHORT. Enter at 00:00 UTC on the first day of the next month. Signal month December 2024 is excluded so the complete 7d endpoint cannot leak into 2025.

Eligibility: trailing complete 24h USD-M QuoteVolume >=5m USDT; all four contracts have >2y history.

Frozen early gate: aggregate 7d signed mean >= +0.25%, >=3/4 symbols positive, and both 2023 and 2024 aggregate mean7 >0.

Result: 92 events. Mean 1d -0.0420%, 3d +0.1788%, 7d +0.3612%. Only ETH and XRP were positive (2/4). 2023 mean7 -0.3072%; 2024 +1.0903%.

Decision: breadth and cross-year gates fail. Freeze without expanding to the other six canonical repositories and without requesting 2025+ data.
