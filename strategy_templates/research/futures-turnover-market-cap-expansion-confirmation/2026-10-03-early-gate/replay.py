import csv,datetime,io,json,math,zipfile
from collections import defaultdict
from pathlib import Path

ROOT=Path("strategy_templates/research/futures-turnover-market-cap-expansion-confirmation/2026-10-03-early-gate")
SYMS={"SOLUSDT":"sol","DOGEUSDT":"doge","LTCUSDT":"ltc","AVAXUSDT":"avax","UNIUSDT":"uni","ZECUSDT":"zec"}
CACHE=Path("/tmp/v161_basis_momentum")
UTC=datetime.timezone.utc

raw=json.loads((ROOT/"inputs/coinmetrics_cap_rows.json").read_text())
caps=defaultdict(dict)
for r in raw:
    try:v=float(r["CapMrktEstUSD"])
    except:continue
    if v>0:caps[r["asset"]][r["time"][:10]]=v

def daily(sym):
    months=["2022-12"]+[f"{y}-{m:02d}" for y in (2023,2024) for m in range(1,13)]
    by=defaultdict(list)
    for m in months:
        p=CACHE/f"fut-{sym}-{m}.zip"
        if not p.exists():continue
        z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as f:
            for r in csv.reader(io.TextIOWrapper(f)):
                if not r or not r[0].isdigit() or len(r)<8:continue
                t=int(r[0]);t=t//1000 if t>10**15 else t
                dt=datetime.datetime.fromtimestamp(t/1000,UTC)
                by[dt.date().isoformat()].append((t,float(r[1]),float(r[4]),float(r[7])))
    out={}
    for day,rr in by.items():
        rr=sorted(rr)
        if len(rr)==24 and rr[-1][0]-rr[0][0]==23*3600000:
            out[day]={"open":rr[0][1],"close":rr[-1][2],"qv":sum(x[3] for x in rr)}
    return out

def mean(xs):return sum(xs)/len(xs) if xs else 0.0
def sm(rows):
    return {"n":len(rows),"mean_r1":mean([x["r1"] for x in rows]),"mean_r3":mean([x["r3"] for x in rows]),
            "mean_r7":mean([x["r7"] for x in rows]),"win_r7":mean([1.0 if x["r7"]>0 else 0.0 for x in rows]),
            "long":sum(x["side"]=="LONG" for x in rows),"short":sum(x["side"]=="SHORT" for x in rows)}

events=[];source={}
for sym,asset in SYMS.items():
    px=daily(sym);cap=caps[asset]
    days=sorted(set(px)&set(cap))
    turnover={d:px[d]["qv"]/cap[d] for d in days if px[d]["qv"]>0 and cap[d]>0}
    source[sym]={"price_days":len(px),"cap_days":len(cap),"aligned_days":len(days)}
    n=0
    for dstr in days:
        d=datetime.date.fromisoformat(dstr)
        if d<datetime.date(2023,1,1) or d>datetime.date(2024,12,31):continue
        need=[(d-datetime.timedelta(days=k)).isoformat() for k in range(0,32)]
        if any(x not in turnover for x in need):continue
        base=mean([turnover[(d-datetime.timedelta(days=k)).isoformat()] for k in range(1,31)])
        pbase=mean([turnover[(d-datetime.timedelta(days=k)).isoformat()] for k in range(2,32)])
        if min(base,pbase)<=0:continue
        prev_day=(d-datetime.timedelta(days=1)).isoformat()
        now=math.log(turnover[dstr]/base);prev=math.log(turnover[prev_day]/pbase)
        if not(prev<=0<now):continue
        d7=(d-datetime.timedelta(days=7)).isoformat()
        if d7 not in px:continue
        trend=math.log(px[dstr]["close"]/px[d7]["close"])
        if trend==0:continue
        side=1.0 if trend>0 else -1.0
        entry_day=d+datetime.timedelta(days=1)
        if entry_day.isoformat() not in px or (d+datetime.timedelta(days=7)).year!=d.year:continue
        entry=px[entry_day.isoformat()]["open"]
        vals={};ok=True
        for h in (1,3,7):
            ed=(d+datetime.timedelta(days=h)).isoformat()
            if ed not in px:ok=False;break
            vals[h]=side*math.log(px[ed]["close"]/entry)
        if not ok:continue
        events.append({"symbol":sym,"asset":asset,"signal_date":dstr,"year":d.year,
                       "side":"LONG" if side>0 else "SHORT","turnover":turnover[dstr],
                       "baseline":base,"score_prev":prev,"score_now":now,"ret7_trailing":trend,
                       "r1":vals[1],"r3":vals[3],"r7":vals[7]});n+=1
    print("SYMBOL",sym,"events",n,flush=True)

by_sym={s:sm([e for e in events if e["symbol"]==s]) for s in SYMS}
by_year={str(y):sm([e for e in events if e["year"]==y]) for y in (2023,2024)}
by_side={q:sm([e for e in events if e["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(events);positive=sum(by_sym[s]["mean_r7"]>0 for s in SYMS)
weeks=((datetime.date(2025,1,1)-datetime.date(2023,1,1)).days/7)*len(SYMS);freq=len(events)/weeks
gate=(overall["mean_r7"]>=.0025 and positive>=4 and freq>=.30 and by_year["2023"]["mean_r7"]>0 and by_year["2024"]["mean_r7"]>0)
summary={"version":"v168","status":"promote_oos" if gate else "frozen_failed_early_gate","gate_pass":gate,
         "events":len(events),"positive_symbols":f"{positive}/6","frequency_per_symbol_week":freq,
         "overall":overall,"by_symbol":by_sym,"by_year":by_year,"by_side":by_side,"source_meta":source,
         "oos_evaluated":False,"strict_engine_run":False}
(ROOT/"results/events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({"events":len(events),"mean_r1":overall["mean_r1"],"mean_r3":overall["mean_r3"],"mean_r7":overall["mean_r7"],
                  "positive_symbols":summary["positive_symbols"],"freq":freq,"2023":by_year["2023"]["mean_r7"],
                  "2024":by_year["2024"]["mean_r7"],"long_r7":by_side["LONG"]["mean_r7"],
                  "short_r7":by_side["SHORT"]["mean_r7"],"gate_pass":gate},indent=2),flush=True)
