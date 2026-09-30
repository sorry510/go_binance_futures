# Protocol

- Source: DeFiLlama chain stablecoin charts, fields totalBridgedToUSD and totalCirculatingUSD.
- bridged_share = bridged USD / circulating USD.
- Signal is the sign change of the 7-day share difference; positive cross LONG, negative cross SHORT.
- Entry next UTC daily open.
- Dynamic eligibility: Binance USD-M history >=2 years and signal-day QuoteVolume >=5m USDT.
- Discovery 2023-2024 only; OOS untouched unless gate passes.
- Gate: n>=80, symbols>=8, mean7>=+0.25%, breadth>=60%, both discovery years positive.
- No post-hoc reversal or window/threshold sweep.
