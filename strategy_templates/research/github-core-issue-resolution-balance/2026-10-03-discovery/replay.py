import csv,datetime,io,json,math,zipfile
from collections import defaultdict
from pathlib import Path

ROOT=Path("strategy_templates/research/github-core-issue-resolution-balance/2026-10-03-discovery")
CACHE=Path("/tmp/github_core_release_price_cache")
FIRST={"BTCUSDT":"2020-01","ETHUSDT":"2020-01","BNBUSDT":"2020-02","XRPUSDT":"2020-01","ADAUSDT":"2020-01",
       "AVAXUSDT":"2020-09","NEARUSDT":"2020-10","TRXUSDT":"2020-01","OPUSDT":"2022-06","APTUSDT":"2022-10"}
UTC=datetime.timezone.utc
H=3600000
EVENTS=json.loads((ROOT/"inputs/eligible_signals.json").read_text())
ELIG=json.loads((ROOT/"results/eligibility_summary.json").read_text())

def months(a,b="2025-01"):
    y,m=map(int,a.split("-")); ey,em=map(int,b.split("-"))
    while (y,m)<=(ey,em):
        yield f"{y:04d}-{m:02d}"
        m+=1
        if m==13:y+=1;m=1

def load(sym):
    out={}
    for mo in months(FIRST[sym]):
        p=CACHE/f"{sym}-1h-{mo}.zip"
        if not p.exists() or p.stat().st_size==0: continue
        try:z=zipfile.ZipFile(p)
        except:continue
        with z.open(z.namelist()[0]) as f:
            for r in csv.reader(io.TextIOWrapper(f)):
                if not r or not r[0].isdigit() or len(r)<5:continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                out[t]=(float(r[1]),float(r[4]))
    return out

bars={s:load(s) for s in FIRST}
def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(rows):
    return {"n":len(rows),
            "mean_r1":mean([x["r1"] for x in rows]),
            "mean_r3":mean([x["r3"] for x in rows]),
            "mean_r7":mean([x["r7"] for x in rows]),
            "win_r7":mean([1.0 if x["r7"]>0 else 0.0 for x in rows]),
            "long":sum(x["side"]=="LONG" for x in rows),
            "short":sum(x["side"]=="SHORT" for x in rows)}

out=[]; errors=[]
for e in EVENTS:
    sym=e["symbol"]; q=bars.get(sym,{})
    side=1.0 if e["side"]=="LONG" else -1.0
    complete=datetime.datetime.fromisoformat(e["signal_complete_at"].replace("Z","+00:00"))
    entry=int(complete.timestamp()*1000)
    year=int(e["signal_date"][:4])
    end7=entry+(7*24-1)*H
    if datetime.datetime.fromtimestamp(end7/1000,UTC).year!=year:
        errors.append({**e,"reason":"7d_endpoint_crosses_signal_year"});continue
    if entry not in q:
        errors.append({**e,"reason":"missing_entry_bar"});continue
    px=q[entry][0]
    vals={}; good=True
    for days in (1,3,7):
        tt=entry+(days*24-1)*H
        if tt not in q or q[tt][1]<=0:
            errors.append({**e,"reason":f"missing_{days}d_endpoint"});good=False;break
        vals[days]=side*math.log(q[tt][1]/px)
    if not good:continue
    out.append({**e,"entry_time":entry,"entry_price":px,
                "r1":vals[1],"r3":vals[3],"r7":vals[7]})

by_sym=defaultdict(list);by_year=defaultdict(list);by_side=defaultdict(list);by_repo=defaultdict(list)
for x in out:
    by_sym[x["asset"]].append(x)
    by_year[x["signal_date"][:4]].append(x)
    by_side[x["side"]].append(x)
    by_repo[x["repo"]].append(x)

overall=sm(out)
sym_stats={k:sm(v) for k,v in sorted(by_sym.items())}
year_stats={k:sm(v) for k,v in sorted(by_year.items())}
side_stats={k:sm(v) for k,v in sorted(by_side.items())}
repo_stats={k:sm(v) for k,v in sorted(by_repo.items())}
positive=sum(v["mean_r7"]>0 for v in sym_stats.values())
freq=float(ELIG.get("frequency_per_eligible_symbol_week",0))
gate=(overall["mean_r7"]>=0.0025 and positive>=6 and freq>=0.30 and
      year_stats.get("2023",{}).get("mean_r7",0)>0 and
      year_stats.get("2024",{}).get("mean_r7",0)>0)

summary={"version":"v177","status":"promote_oos" if gate else "frozen_failed_discovery",
         "gate_pass":gate,"events":len(out),"symbols":len(sym_stats),
         "positive_symbols":f"{positive}/{len(sym_stats)}",
         "frequency_per_eligible_symbol_week":freq,
         "overall":overall,"by_year":year_stats,"by_side":side_stats,
         "by_symbol":sym_stats,"by_repo":repo_stats,
         "replay_errors":errors,"oos_2025_plus_evaluated":False,"strict_engine_run":False}
(ROOT/"results/events.json").write_text(json.dumps(out,indent=2))
(ROOT/"results/replay_errors.json").write_text(json.dumps(errors,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({
 "events":len(out),"symbols":len(sym_stats),"positive_symbols":summary["positive_symbols"],
 "mean_r1":overall["mean_r1"],"mean_r3":overall["mean_r3"],"mean_r7":overall["mean_r7"],
 "freq":freq,"2023":year_stats.get("2023",{}).get("mean_r7",0),
 "2024":year_stats.get("2024",{}).get("mean_r7",0),
 "long_r7":side_stats.get("LONG",{}).get("mean_r7",0),
 "short_r7":side_stats.get("SHORT",{}).get("mean_r7",0),
 "gate_pass":gate,"errors":len(errors)
},indent=2),flush=True)
