import json, math
from pathlib import Path
from collections import defaultdict
from datetime import date

BASE=Path("strategy_templates/research")
ROOT=BASE/"multi-mechanism-reversal-consensus/2026-10-03-discovery"
SOURCES={
 "v190":BASE/"trading-invariant-stress-reversal/2026-10-03-early-gate/results/events.json",
 "v192":BASE/"intrahour-volatility-signature-noise-reversal/2026-10-03-early-gate/results/events.json",
 "v194":BASE/"three-bar-fair-value-gap-fill-reversal/2026-10-03-early-gate/results/events.json",
 "v196":BASE/"intrahour-open-reference-recross-reversal/2026-10-03-early-gate/results/events.json",
}
by=defaultdict(dict)
for fam,p in SOURCES.items():
    rows=json.loads(p.read_text())
    seen=set()
    for e in rows:
        key=(e["symbol"],int(e["signal_time"]))
        if key in seen:
            raise RuntimeError(f"duplicate {fam} {key}")
        seen.add(key)
        by[key][fam]=e

out=[]
mixed=0
return_mismatch=[]
vote_hist=defaultdict(int)
for (sym,t),famrows in sorted(by.items()):
    longf=sorted([f for f,e in famrows.items() if e["side"]=="LONG"])
    shortf=sorted([f for f,e in famrows.items() if e["side"]=="SHORT"])
    if longf and shortf:
        mixed+=1
        continue
    chosen=None
    fams=None
    if len(longf)>=2:
        chosen="LONG"; fams=longf
    elif len(shortf)>=2:
        chosen="SHORT"; fams=shortf
    else:
        continue
    vote_hist[(chosen,len(fams))]+=1
    rs=[famrows[f] for f in fams]
    for field in ("r1","r4","r12"):
        vals=[float(x[field]) for x in rs]
        if max(vals)-min(vals)>1e-12:
            return_mismatch.append({"symbol":sym,"signal_time":t,"field":field,"values":vals,"families":fams})
    ref=rs[0]
    out.append({
      "symbol":sym,"signal_time":t,"year":int(ref["year"]),"side":chosen,
      "families":fams,"votes":len(fams),
      "r1":float(ref["r1"]),"r4":float(ref["r4"]),"r12":float(ref["r12"])
    })

def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(rows):
    return {"n":len(rows),
      "mean_r1":mean([x["r1"] for x in rows]),
      "mean_r4":mean([x["r4"] for x in rows]),
      "mean_r12":mean([x["r12"] for x in rows]),
      "win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in rows]),
      "long":sum(x["side"]=="LONG" for x in rows),
      "short":sum(x["side"]=="SHORT" for x in rows)}

symbols=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
by_sym={s:sm([x for x in out if x["symbol"]==s]) for s in symbols}
by_year={str(y):sm([x for x in out if x["year"]==y]) for y in (2023,2024)}
by_side={q:sm([x for x in out if x["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(out)
pos=sum(by_sym[s]["mean_r12"]>0 for s in symbols)
weeks=((date(2025,1,1)-date(2023,1,1)).days/7)*len(symbols)
freq=len(out)/weeks
gate=(overall["mean_r12"]>=0.002 and pos>=4 and freq>=0.30 and
      by_year["2023"]["mean_r12"]>0 and by_year["2024"]["mean_r12"]>0 and
      not return_mismatch)
summary={
 "version":"v197","status":"promote_2025_oos" if gate else "frozen_failed_discovery",
 "gate_pass":gate,"events":len(out),"mixed_direction_hours_discarded":mixed,
 "return_mismatch_count":len(return_mismatch),
 "positive_symbols":f"{pos}/6","frequency_per_symbol_week":freq,
 "overall":overall,"by_symbol":by_sym,"by_year":by_year,"by_side":by_side,
 "vote_histogram":{f"{k[0]}_{k[1]}votes":v for k,v in sorted(vote_hist.items())},
 "oos_2025_evaluated":False,"db_write":False
}
(ROOT/"results/events.json").write_text(json.dumps(out,indent=2))
(ROOT/"results/return_mismatch.json").write_text(json.dumps(return_mismatch,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({
 "events":len(out),"mixed":mixed,"mismatch":len(return_mismatch),
 "freq":freq,"mean_r1":overall["mean_r1"],"mean_r4":overall["mean_r4"],"mean_r12":overall["mean_r12"],
 "positive_symbols":summary["positive_symbols"],
 "2023":by_year["2023"]["mean_r12"],"2024":by_year["2024"]["mean_r12"],
 "long_r12":by_side["LONG"]["mean_r12"],"short_r12":by_side["SHORT"]["mean_r12"],
 "votes":summary["vote_histogram"],"gate_pass":gate
},indent=2))
