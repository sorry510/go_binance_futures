# Dynamic Momentum Cycle — Faithful Replication Source Audit

Purpose: revisit Oliver Borgards (2021), *Dynamic time series momentum of cryptocurrencies*, without repeating the previously frozen SMA5 turning-point approximation.

## What is specified by the paper family

The public paper and follow-up implementations consistently specify:
- a positive/negative momentum cycle requires two consecutive peaks and two consecutive troughs to both rise/fall;
- the first valid four-turning-point cycle is the formation period;
- later valid chained cycles form the momentum period;
- turning points come from a moving-average smoothing filter;
- sensitivity parameter kappa controls the number/relevance of identified turning points;
- Borgards (2021) uses kappa=5 and refers the filter implementation back to Borgards & Czudaj (2020).

## Source audit

Sources checked:
1. Borgards, O. (2021), *Dynamic time series momentum of cryptocurrencies*, North American Journal of Economics and Finance 57, 101428, DOI 10.1016/j.najef.2021.101428.
2. Borgards, O. & Czudaj, R.L. (2020), *The prevalence of price overreactions in the cryptocurrency market*, JIFMIM 65, 101194, DOI 10.1016/j.intfin.2020.101194.
3. Borgards, O., Czudaj, R.L. & Hoang, T.H.V. (2021), *Price overreactions in the commodity futures market*, Resources Policy 71, 101966.
4. Oliver Borgards doctoral dissertation metadata/archive, *Speculation driven overreaction and momentum effects in cryptocurrency and commodity markets* (2021), Qucosa/Monarch.
5. Author/research pages and searchable derivative literature reproducing the method description.

Across the publicly retrievable/searchable material, the exact moving-average smoothing-filter recursion / turning-point pseudocode was not recovered. The sources specify the role of kappa and the cycle construction after turning points exist, but that is insufficient to reproduce the original turning-point sequence uniquely.

The Qucosa archive exposes the dissertation PDF metadata, but the PDF attachment is not fetchable through the research browser used for the audit. No alternative author code repository or executable implementation was located.

## Decision

**SOURCE BLOCKED — not an alpha failure.**

Do not:
- reuse the previously frozen SMA-slope approximation;
- invent a centered/trailing SMA turning-point rule;
- infer the implementation from Fig. 1;
- tune SMA length or confirmation bars;
- claim a faithful Borgards replication.

Reopen only if one of the following becomes available:
- author/source code;
- an exact algorithm/pseudocode from the authors;
- a primary source that fully specifies the smoothing/filter recursion and causal turning-point confirmation.

No market returns were read in this audit, no DB write occurred, and no strategy candidate was created.
