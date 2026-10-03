import json
from pathlib import Path
from collections import defaultdict

B=Path("strategy_templates/research")
R=B/"cycle-audit-backfill-v190-v197/2026-10-03/results"
old={
"v190":B/"trading-invariant-stress-reversal/2026-10-03-early-gate/results/events.json",
"v192":B/"intrahour-volatility-signature-noise-reversal/2026-10-03-early-gate/results/events.json",
"v194":B/"three-bar-fair-value-gap-fill-reversal/2026-10-03-early-gate/results/events.json",
"v196":B/"intrahour-open-reference-recross-reversal/2026-10-03-early-gate/results/events.json",
}
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(es):
 return {"n":len(es),"mean_r1":mean([e["r1"] for e in es]),"mean_r4":mean([e["r4"] for e in es]),
         "mean_r12":mean([e["r12"] for e in es]),"win_r12":mean([1.0 if e["r12"]>0 else 0.0 for e in es]),
         "long":sum(e["side"]=="LONG" for e in es),"short":sum(e["side"]=="SHORT" for e in es)}
def summarize(v,es):
 byy={str(y):sm([e for e in es if int(e["year"])==y]) for y in (2023,2024,2025,2026)}
 bys={s:sm([e for e in es if e["symbol"]==s]) for s in SYMS}
 return {"version":v,"events":len(es),"overall":sm(es),"by_year":byy,"by_symbol":bys,
         "positive_symbols":f"{sum(bys[s]['mean_r12']>0 for s in SYMS)}/6",
         "positive_years":f"{sum(byy[str(y)]['mean_r12']>0 for y in (2023,2024,2025,2026))}/4"}

combined={}
for v,p in old.items():
 oldes=json.loads(p.read_text())
 newes=json.loads((R/f"{v}_events.json").read_text())
 later=[e for e in newes if int(e["year"]) in (2025,2026)]
 combined[v]=oldes+later
 (R/f"{v}_strict_parity_events.json").write_text(json.dumps(combined[v],indent=2))

# v197 from strict-parity source sets
lookup=defaultdict(dict)
for fam,es in combined.items():
 for e in es: lookup[(e["symbol"],int(e["signal_time"]))][fam]=e
cons=[];mixed=0
for (sym,t),fr in sorted(lookup.items()):
 longs=sorted(f for f,e in fr.items() if e["side"]=="LONG")
 shorts=sorted(f for f,e in fr.items() if e["side"]=="SHORT")
 if longs and shorts: mixed+=1;continue
 fams=longs if len(longs)>=2 else (shorts if len(shorts)>=2 else [])
 if not fams: continue
 side="LONG" if longs else "SHORT"; ref=fr[fams[0]]
 if any(abs(fr[f]["r12"]-ref["r12"])>1e-12 for f in fams): raise RuntimeError("return mismatch")
 cons.append({"symbol":sym,"signal_time":t,"year":int(ref["year"]),"side":side,"families":fams,
              "r1":ref["r1"],"r4":ref["r4"],"r12":ref["r12"]})
(R/"v197_strict_parity_events.json").write_text(json.dumps(cons,indent=2))
combined["v197"]=cons

# verify old v197 exact keys for 2023-24
old197=json.loads((B/"multi-mechanism-reversal-consensus/2026-10-03-discovery/results/events.json").read_text())
a={(e["symbol"],int(e["signal_time"]),e["side"]) for e in old197}
b={(e["symbol"],int(e["signal_time"]),e["side"]) for e in cons if int(e["year"]) in (2023,2024)}
parity197={"old":len(a),"new":len(b),"missing":len(a-b),"extra":len(b-a),"exact":a==b}

summ={v:summarize(v,es) for v,es in combined.items()}
summ["v197"]["mixed_direction_hours_discarded"]=mixed
out={"mode":"strict_parity_2023_2024_plus_complete_2025_2026","v197_parity":parity197,"summaries":summ}
(R/"strict_parity_summary.json").write_text(json.dumps(out,indent=2))
print(json.dumps({"v197_parity":parity197,"summary":{v:{
 "events":s["events"],"r12":s["overall"]["mean_r12"],"positive_symbols":s["positive_symbols"],"positive_years":s["positive_years"],
 "years":{y:s["by_year"][y]["mean_r12"] for y in ("2023","2024","2025","2026")}
} for v,s in summ.items()}},indent=2))
