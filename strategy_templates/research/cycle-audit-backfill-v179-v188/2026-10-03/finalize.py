import json
from pathlib import Path
from datetime import date
ROOT=Path("strategy_templates/research/cycle-audit-backfill-v179-v188/2026-10-03")
BASE=Path("strategy_templates/research")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
OLD={
"v179":"return-energy-concentration-regime/2026-10-03-early-gate/results/events.json",
"v180":"range-occupancy-regime-cross/2026-10-03-early-gate/results/events.json",
"v181":"activity-volatility-coupling-regime/2026-10-03-early-gate/results/events.json",
"v182":"per-trade-volatility-impact-regime/2026-10-03-early-gate/results/events.json",
"v183":"price-monotonicity-regime-cross/2026-10-03-early-gate/results/events.json",
"v184":"notional-vs-trade-arrival-concentration/2026-10-03-early-gate/results/events.json",
"v185":"participation-ticket-size-coupling-regime/2026-10-03-early-gate/results/events.json",
"v186":"rolling-4h-extreme-order-continuation/2026-10-03-early-gate/results/events.json",
"v187":"taker-global-account-skew/2026-10-03-early-gate/results/events.json",
"v188":"corwin-schultz-liquidity-stress-reversal/2026-10-03-early-gate/results/events.json"}
def mean(xs):return sum(xs)/len(xs) if xs else 0.0
def sm(es):
 return {"n":len(es),"mean_r1":mean([e["r1"] for e in es]),"mean_r4":mean([e["r4"] for e in es]),
 "mean_r12":mean([e["r12"] for e in es]),"win_r12":mean([1 if e["r12"]>0 else 0 for e in es]),
 "long":sum(e["side"]=="LONG" for e in es),"short":sum(e["side"]=="SHORT" for e in es)}
def summary(v,es):
 byy={str(y):sm([e for e in es if int(e["year"])==y]) for y in (2023,2024,2025,2026)}
 bys={s:sm([e for e in es if e["symbol"]==s]) for s in SYMS}
 byside={q:sm([e for e in es if e["side"]==q]) for q in ("LONG","SHORT")}
 weeks=((date(2026,10,1)-date(2023,1,1)).days/7)*len(SYMS)
 return {"version":v,"events":len(es),"overall":sm(es),"by_year":byy,"by_symbol":bys,"by_side":byside,
 "positive_symbols":f'{sum(bys[s]["mean_r12"]>0 for s in SYMS)}/6',
 "positive_years":f'{sum(byy[str(y)]["mean_r12"]>0 for y in (2023,2024,2025,2026))}/4',
 "frequency_per_symbol_week":len(es)/weeks}
out={}
for v,rel in OLD.items():
 old=json.loads((BASE/rel).read_text())
 later=json.loads((ROOT/v/"results/events.json").read_text())
 if any(int(e["year"]) not in (2023,2024) for e in old):raise RuntimeError(v+" old contamination")
 if any(int(e["year"]) not in (2025,2026) for e in later):raise RuntimeError(v+" later contamination")
 keys={(e["symbol"],int(e["signal_time"]),e["side"]) for e in old}
 if any((e["symbol"],int(e["signal_time"]),e["side"]) in keys for e in later):raise RuntimeError(v+" overlap")
 combined=old+later
 (ROOT/"results"/f"{v}_events.json").write_text(json.dumps(combined,indent=2))
 out[v]=summary(v,combined)
(ROOT/"results/summary.json").write_text(json.dumps(out,indent=2))
print(json.dumps({v:{"events":s["events"],"r12":s["overall"]["mean_r12"],
 "positive_symbols":s["positive_symbols"],"positive_years":s["positive_years"],
 "freq":s["frequency_per_symbol_week"],
 "years":{y:s["by_year"][y]["mean_r12"] for y in ("2023","2024","2025","2026")}}
 for v,s in out.items()},indent=2))
