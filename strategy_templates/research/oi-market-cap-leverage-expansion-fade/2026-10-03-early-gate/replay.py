import csv,datetime,io,json,math,time,urllib.parse,urllib.request,zipfile
from collections import defaultdict
from pathlib import Path

ROOT=Path("strategy_templates/research/oi-market-cap-leverage-expansion-fade/2026-10-03-early-gate")
SYMS={"SOLUSDT":"sol","DOGEUSDT":"doge","LTCUSDT":"ltc","AVAXUSDT":"avax","UNIUSDT":"uni","ZECUSDT":"zec"}
METRICS_CACHE=Path("/tmp/v147-binance-metrics")
KLINE_CACHE=Path("/tmp/v161_basis_momentum")
CM="https://community-api.coinmetrics.io/v4/timeseries/asset-metrics"
UTC=datetime.timezone.utc

def fetch_caps():
    params={"assets":",".join(SYMS.values()),"metrics":"CapMrktEstUSD","frequency":"1d",
            "start_time":"2022-11-30","end_time":"2024-12-31","page_size":10000}
    u=CM+"?"+urllib.parse.urlencode(params); rows=[]
    while u:
        req=urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0","Accept":"application/json"})
        o=json.loads(urllib.request.urlopen(req,timeout=30).read())
        rows.extend(o.get("data",[]))
        u=o.get("next_page_url")
        if u and u.startswith("https://api.coinmetrics.io"):
            u=u.replace("https://api.coinmetrics.io","https://community-api.coinmetrics.io",1)
        if u: time.sleep(.5)
    (ROOT/"inputs/coinmetrics_cap_rows.json").write_text(json.dumps(rows))
    by=defaultdict(dict)
    for r in rows:
        try: v=float(r["CapMrktEstUSD"])
        except: continue
        if v>0: by[r["asset"]][r["time"][:10]]=v
    return by

def read_daily_oi(sym):
    out={}
    for y in (2022,2023,2024):
        start=datetime.date(y,12,1) if y==2022 else datetime.date(y,1,1)
        end=datetime.date(y,12,31)
        d=start
        while d<=end:
            if y==2022 and d<datetime.date(2022,12,1):
                d+=datetime.timedelta(days=1); continue
            p=METRICS_CACHE/f"{sym}-{d.isoformat()}.zip"
            if p.exists():
                z=zipfile.ZipFile(p); last=None
                with z.open(z.namelist()[0]) as f:
                    rr=list(csv.reader(io.TextIOWrapper(f)))
                for r in rr:
                    if not r or r[0]=="create_time" or len(r)<4: continue
                    try: val=float(r[3])
                    except: continue
                    last=val
                if last and last>0: out[d.isoformat()]=last
            d+=datetime.timedelta(days=1)
    return out

def read_daily_price(sym):
    hours={}
    months=["2022-12"]+[f"{y}-{m:02d}" for y in (2023,2024) for m in range(1,13)]
    for m in months:
        p=KLINE_CACHE/f"fut-{sym}-{m}.zip"
        if not p.exists(): continue
        z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as f:
            for r in csv.reader(io.TextIOWrapper(f)):
                if not r or not r[0].isdigit() or len(r)<5: continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                dt=datetime.datetime.fromtimestamp(t/1000,UTC)
                hours[t]=(dt.date(),float(r[1]),float(r[4]))
    by=defaultdict(list)
    for t,(day,o,c) in sorted(hours.items()): by[day.isoformat()].append((t,o,c))
    out={}
    for day,rows in by.items():
        rows=sorted(rows)
        if len(rows)==24 and rows[-1][0]-rows[0][0]==23*3600000:
            out[day]={"open":rows[0][1],"close":rows[-1][2]}
    return out
def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(rows):
    return {
      "n":len(rows),
      "mean_r1":mean([x["r1"] for x in rows]),
      "mean_r3":mean([x["r3"] for x in rows]),
      "mean_r7":mean([x["r7"] for x in rows]),
      "win_r7":mean([1.0 if x["r7"]>0 else 0.0 for x in rows]),
      "long":sum(x["side"]=="LONG" for x in rows),
      "short":sum(x["side"]=="SHORT" for x in rows)
    }

caps=fetch_caps()
events=[]; source={}; errors=[]
for sym,asset in SYMS.items():
    oi=read_daily_oi(sym); px=read_daily_price(sym); cap=caps.get(asset,{})
    days=sorted(set(oi)&set(px)&set(cap))
    source[sym]={"oi_days":len(oi),"price_days":len(px),"cap_days":len(cap),"aligned_days":len(days)}
    ratio={d:oi[d]/cap[d] for d in days if oi[d]>0 and cap[d]>0}
    n=0
    for dstr in days:
        d=datetime.date.fromisoformat(dstr)
        if d<datetime.date(2023,1,1) or d>datetime.date(2024,12,31): continue

        need=[(d-datetime.timedelta(days=k)).isoformat() for k in range(0,32)]
        if any(x not in ratio for x in need): continue

        baseline=mean([ratio[(d-datetime.timedelta(days=k)).isoformat()] for k in range(1,31)])
        prev_day=d-datetime.timedelta(days=1)
        prev_base=mean([ratio[(d-datetime.timedelta(days=k)).isoformat()] for k in range(2,32)])
        if baseline<=0 or prev_base<=0: continue
        now=math.log(ratio[dstr]/baseline)
        prev=math.log(ratio[prev_day.isoformat()]/prev_base)
        if not (prev<=0<now): continue

        d7=(d-datetime.timedelta(days=7)).isoformat()
        if d7 not in px or dstr not in px or px[d7]["close"]<=0 or px[dstr]["close"]<=0: continue
        trend=math.log(px[dstr]["close"]/px[d7]["close"])
        if trend==0: continue
        side=-1.0 if trend>0 else 1.0

        entry_day=d+datetime.timedelta(days=1)
        if entry_day.isoformat() not in px: continue
        if (d+datetime.timedelta(days=7)).year!=d.year: continue
        entry=px[entry_day.isoformat()]["open"]
        if entry<=0: continue

        vals={}; ok=True
        for h in (1,3,7):
            ed=(d+datetime.timedelta(days=h)).isoformat()
            if ed not in px or px[ed]["close"]<=0: ok=False; break
            vals[h]=side*math.log(px[ed]["close"]/entry)
        if not ok: continue

        events.append({
          "symbol":sym,"asset":asset,"signal_date":dstr,"year":d.year,
          "side":"LONG" if side>0 else "SHORT",
          "leverage_ratio":ratio[dstr],"baseline_ratio":baseline,
          "score_prev":prev,"score_now":now,"ret7_trailing":trend,
          "r1":vals[1],"r3":vals[3],"r7":vals[7]
        }); n+=1
    print("SYMBOL",sym,"events",n,"source",source[sym],flush=True)

by_sym={s:sm([e for e in events if e["symbol"]==s]) for s in SYMS}
by_year={str(y):sm([e for e in events if e["year"]==y]) for y in (2023,2024)}
by_side={q:sm([e for e in events if e["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(events)
positive=sum(by_sym[s]["mean_r7"]>0 for s in SYMS)
weeks=((datetime.date(2025,1,1)-datetime.date(2023,1,1)).days/7)*len(SYMS)
freq=len(events)/weeks
gate=(overall["mean_r7"]>=.0025 and positive>=4 and freq>=.30 and
      by_year["2023"]["mean_r7"]>0 and by_year["2024"]["mean_r7"]>0)
summary={
  "version":"v167","status":"promote_oos" if gate else "frozen_failed_early_gate",
  "gate_pass":gate,"events":len(events),"positive_symbols":f"{positive}/6",
  "frequency_per_symbol_week":freq,"overall":overall,
  "by_symbol":by_sym,"by_year":by_year,"by_side":by_side,
  "source_meta":source,"oos_evaluated":False,"strict_engine_run":False
}
(ROOT/"results/events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({
  "events":len(events),"mean_r1":overall["mean_r1"],"mean_r3":overall["mean_r3"],
  "mean_r7":overall["mean_r7"],"positive_symbols":summary["positive_symbols"],
  "freq":freq,"2023":by_year["2023"]["mean_r7"],"2024":by_year["2024"]["mean_r7"],
  "long_r7":by_side["LONG"]["mean_r7"],"short_r7":by_side["SHORT"]["mean_r7"],
  "gate_pass":gate
},indent=2),flush=True)
