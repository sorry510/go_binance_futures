import csv,datetime,io,json,zipfile
from pathlib import Path
from collections import Counter

ROOT=Path("strategy_templates/research/github-core-issue-resolution-balance/2026-10-03-discovery")
CACHE=Path("/tmp/github_core_release_price_cache")
FIRST={"BTCUSDT":"2020-01","ETHUSDT":"2020-01","BNBUSDT":"2020-02","XRPUSDT":"2020-01","ADAUSDT":"2020-01",
       "AVAXUSDT":"2020-09","NEARUSDT":"2020-10","TRXUSDT":"2020-01","OPUSDT":"2022-06","APTUSDT":"2022-10"}
H=3600000
UTC=datetime.timezone.utc
END=datetime.datetime(2024,12,31,23,59,59,tzinfo=UTC)
signals=json.loads((ROOT/"inputs/signals_raw.json").read_text())

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
                if not r or not r[0].isdigit() or len(r)<8:continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                out[t]=(float(r[1]),float(r[4]),float(r[7]))
    return out

bars={s:load(s) for s in FIRST}
first={s:datetime.datetime.fromtimestamp(min(q)/1000,UTC) for s,q in bars.items() if q}
eligible_from={s:first[s]+datetime.timedelta(days=730) for s in first}

eligible=[]; skipped=[]
for e in signals:
    sym=e["symbol"]
    complete=datetime.datetime.fromisoformat(e["signal_complete_at"].replace("Z","+00:00"))
    q=bars.get(sym,{})
    if sym not in eligible_from:
        skipped.append({**e,"reason":"no_price_data"});continue
    if complete<eligible_from[sym]:
        skipped.append({**e,"reason":"history_lt_730d","eligible_from":eligible_from[sym].isoformat()});continue
    floor=int(complete.timestamp()*1000)
    hist=[floor-H*i for i in range(24,0,-1)]
    if any(t not in q for t in hist):
        skipped.append({**e,"reason":"missing_prior_24h"});continue
    qv=sum(q[t][2] for t in hist)
    if qv<5_000_000:
        skipped.append({**e,"reason":"qv_lt_5m","qv24":qv});continue
    eligible.append({**e,"qv24":qv,"eligible_from":eligible_from[sym].isoformat()})

disc_start=datetime.datetime(2023,1,1,tzinfo=UTC)
symbol_days={}; total_days=0
for sym in FIRST:
    st=max(disc_start,eligible_from.get(sym,END+datetime.timedelta(days=1)))
    days=(END.date()-st.date()).days+1 if st<=END else 0
    symbol_days[sym]=days; total_days+=days
freq=len(eligible)/(total_days/7.0) if total_days else 0.0
reasons=Counter(x["reason"] for x in skipped)
summary={
 "version":"v177",
 "raw_signals":len(signals),"raw_symbols":len({x["symbol"] for x in signals}),
 "eligible_signals":len(eligible),"eligible_symbols":len({x["symbol"] for x in eligible}),
 "eligible_long":sum(x["side"]=="LONG" for x in eligible),
 "eligible_short":sum(x["side"]=="SHORT" for x in eligible),
 "eligible_symbol_days":symbol_days,
 "frequency_per_eligible_symbol_week":freq,
 "frequency_gate_pass":freq>=0.30,
 "skipped_by_reason":dict(reasons),
 "post_event_returns_read":False
}
(ROOT/"inputs/eligible_signals.json").write_text(json.dumps(eligible,indent=2))
(ROOT/"inputs/eligibility_skipped.json").write_text(json.dumps(skipped,indent=2))
(ROOT/"results/eligibility_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2),flush=True)
