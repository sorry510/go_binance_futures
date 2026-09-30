# Volume Profile / Value Area Breakout

Fixed auction-market definition: use the previous 24 completed 1h bars, assign each bar's QuoteVolume to its typical price (H+L+C)/3, and define the central 70% value area by weighted 15th/85th price quantiles. First close above VAH is LONG; first close below VAL is SHORT; a close back inside the value area rearms. Entry proxy is the next 1h open.

True Binance Vision contract-history audit confirms all ten fixed old symbols were already more than two years old by 2023.

2023-2024 discovery: 21,303 events; 12h signed mean -0.00215%, 6/10 symbols positive. 2025 OOS1: +0.0576%, 6/10 positive. 2026 OOS2: +0.1208%, 8/10 positive.

Decision: freeze. Discovery is economically zero/negative, so later improvement cannot be used to promote the family. No exact 1m replay and no post-hoc reversal or alternative value-area percentages.
