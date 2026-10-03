# v162 GitHub Core Release Cadence Acceleration — Protocol

1. Repository identity is copied unchanged from the frozen GitHub Core Release Catalyst family. No repo substitution or symbol expansion is permitted.
2. Use only GitHub Release objects with draft=false, prerelease=false and non-null published_at. Same-symbol same-UTC-date releases collapse to earliest published_at, matching the parent family's source semantics.
3. Release timestamps before 2023 may be used only as causal warmup for cadence intervals. Price outcomes before 2023 and after 2024 are not evaluated in discovery.
4. For release i, current_interval = published_i - published_(i-1); previous_interval = published_(i-1) - published_(i-2). score_i = log(previous_interval/current_interval). Positive score means cadence acceleration.
5. Signal only when score crosses zero relative to the preceding release's score: <=0 to >0 -> LONG; >=0 to <0 -> SHORT. No interval magnitude threshold or semver/title filtering.
6. Entry is the next complete Binance USD-M 1h open after published_at. Dynamic eligibility: same-name USD-M history >=730 days and trailing 24 complete 1h QuoteVolume >=5M USDT.
7. Measure signed 1h/4h/12h returns. 12h endpoint must remain in the signal calendar year.
8. Discovery gate: >=80 eligible signals, >=8 symbols, event-weighted 12h >=+0.25%, >=60% symbol means positive, and both 2023/2024 aggregate 12h means >0.
9. Failure freezes this family: do not scan interval ratios, use release-count windows, filter semver/major releases, delete one direction, or reverse the mapping.
10. This is distinct from the parent single-release catalyst, which treated every stable release as LONG. Here the state variable is change in the release-arrival interval and direction can be LONG or SHORT.
