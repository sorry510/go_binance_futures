# Protocol

1. Source: DeFiLlama stablecoincharts/{chain}, using historical totalCirculating.peggedUSD and totalCirculatingUSD.peggedUSD.
2. Historical weighted peg ratio = totalCirculatingUSD / totalCirculating.
3. First ratio <=0.995 after a ratio >=0.999 re-arm is an under-peg stress event.
4. Fixed direction is SHORT the mapped chain native token; if coverage passes, entry would be next UTC daily open.
5. Before any return inspection require event-day nominal chain peggedUSD supply >=50m USD, corresponding Binance USD-M contract history >=2 years, >=8 eligible symbols, and >=8 distinct UTC event dates.
6. Same-date multi-chain events are retained for coverage but distinct-date count is separately required to prevent one broad stablecoin depeg from being treated as many independent episodes.
7. If coverage fails, do not inspect returns, loosen the supply gate, add immature symbols, or create a recovery/LONG mirror.
