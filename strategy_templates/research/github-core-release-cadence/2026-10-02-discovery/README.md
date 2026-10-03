# v162 GitHub Core Release Cadence Acceleration — Discovery

Hypothesis: changes in the cadence of stable core-software releases can carry information distinct from a generic release catalyst. Repository identities are copied unchanged from the frozen GitHub Core Release Catalyst family.

Signal:
- use only GitHub Release objects with draft=false, prerelease=false and non-null published_at;
- same-symbol same-UTC-date releases collapse to the earliest published_at;
- current_interval = release_i - release_(i-1);
- previous_interval = release_(i-1) - release_(i-2);
- score = log(previous_interval / current_interval);
- zero up-cross -> LONG, zero down-cross -> SHORT;
- next complete Binance USD-M 1h open.

Dynamic eligibility is same-name USD-M history >=730 days plus trailing 24 complete 1h QuoteVolume >=5M USDT. Discovery is 2023-2024 only.

Coverage before outcomes:
- 285 raw signals / 10 symbols.
- 174 eligible signals / 10 symbols.
- LONG 86 / SHORT 88.
- 111 excluded for history <730 days.
- No discovery return was read before coverage passed.

## Final result

- **174 events / 10 symbols**.
- 1h signed mean: **-0.0863%**.
- 4h signed mean: **-0.1604%**.
- 12h signed mean: **+0.2268%**.
- Positive symbols: **6/10 (60%)**.
- 2023 12h: **-0.0866%**.
- 2024 12h: **+0.4279%**.
- LONG: 86 events, 12h **+0.3192%**.
- SHORT: 88 events, 12h **+0.1366%**.
- Replay errors: 0.

The preregistered +0.25% economic threshold is narrowly missed, and the mandatory annual sign-consistency gate fails because 2023 is negative. The side split is audit-only and cannot be used to alter the rule.

Decision: **freeze v162**. Do not scan release-interval ratios/windows, add semver/title filtering, delete a side, reverse the mapping, or inspect 2025+ for redesign. No strict TP8/SL6 replay and no DB import.
