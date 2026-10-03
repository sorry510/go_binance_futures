# ID121 no-no-chase — Fresh-Symbol Formal Validation

Purpose: formally validate the only ID121 ablation simplification that remained close to canonical strength on the original 15-symbol cohort: removing the no-chase condition while changing nothing else.

Frozen source strategy:
- archived `no_no_chase.json`;
- SHA-256 `6398522cecdddc67acc2926034f79db3bdae7c44e1ba3c950310fcfa204d632e`;
- canonical leverage=4, TP8/SL6, fee/slippage, single-position semantics.

## Outcome-independent fresh-universe attempt

The initial fresh set was frozen before returns:
LINK/BCH/ETC/TRX/XLM/AAVE.

With `Repository(nil)`, all six were blocked before Engine execution because the local DB/cache has no corresponding long 1m history. **No no-no-chase return, trade, PF or PnL was read.**

A read-only inventory of `market_klines_1m_chunks` was then performed to see whether an alternative untouched set already exists locally.

Non-source cached symbols include:
- ALGO, INJ, LDO: long coverage;
- LIT: long coverage but known special historical gap;
- PENDLE, PYTH: later-starting coverage;
- several much newer contracts.

However, ALGO/INJ/LDO/PENDLE/PYTH were already used in the canonical ID121 fresh-symbol holdout, whose outcomes are known. Reusing them for the no-no-chase simplification would not be an untouched validation and would violate the family freeze against retuning to that holdout. LIT alone cannot form a cross-symbol validation set.

## Decision

**Fresh-symbol validation unavailable / blocked before returns.**

Do not:
- import or REST-repair new symbols merely to enable this test;
- reuse ALGO/INJ/LDO/PENDLE/PYTH as supposedly untouched validation;
- use original 15 source symbols;
- lower the required cross-symbol validation breadth;
- inspect partial results.

The valid next path remains a genuinely untouched temporal holdout once its local historical coverage is complete.
