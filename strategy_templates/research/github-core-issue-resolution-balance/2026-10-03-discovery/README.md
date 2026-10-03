# v177 GitHub Core Issue-Resolution Balance — Discovery

Mechanism:
- daily issue creations and closures by UTC date;
- balance_t = closures over latest 7 complete UTC days - creations over same 7 days;
- fresh zero up-cross -> LONG (resolution overtakes arrival);
- fresh zero down-cross -> SHORT (backlog pressure resumes);
- enter next UTC-day Binance USD-M open.

No issue title/label/severity/NLP filters, repo normalization, closure-age weighting, price/funding/OI overlay or symbol-specific rule.

## Eligibility before outcomes

- 1,035 raw signals / 10 symbols.
- 883 eligible signals / 10 symbols.
- LONG 406 / SHORT 477.
- Frequency **1.0070 events/eligible-symbol/week**.
- 152 excluded only for <730d USD-M history.
- No token return was read before eligibility completed.

## 2023-2024 discovery

After excluding 18 preregistered year-end events whose 7d endpoint crossed the signal calendar year:

- **865 replayed events / 10 symbols**.
- 1d signed mean **+0.0666%**.
- 3d signed mean **-0.1924%**.
- 7d signed mean **-0.1927%**.
- Breadth **4/10 positive symbols**.
- 2023 7d **-0.4703%**.
- 2024 7d **+0.0854%**.
- LONG audit: 7d **+0.9695%**.
- SHORT audit: 7d **-1.1695%**.

The preregistered economic, breadth and annual sign-consistency gates fail. The LONG/SHORT split is strictly post-result attribution and cannot justify deleting SHORT.

Decision: **freeze v177**. Do not alter the 7d balance window, add issue-type filters, use close/open ratios, delete repos/sides, reverse the mapping, inspect 2025+ for rescue, or run strict TP8/SL6. No DB import.
