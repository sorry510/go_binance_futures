import csv,io,zipfile,datetime,json
from pathlib import Path
ROOT=Path("strategy_templates/research/github-core-release-cadence/2026-10-02-discovery")
CACHE=Path("/tmp/github_core_release_price_cache")
FIRST={"BTCUSDT":"2020-01","ETHUSDT":"2020-01","BNBUSDT":"2020-02","XRPUSDT":"2020-01","ADAUSDT":"2020-01",
       "AVAXUSDT":"2020-09","NEARUSDT":"2020-10","TRXUSDT":"2020-01","OPUSDT":"2022-06","APTUSDT":"2022-10"}
H=3600000
events=json.loads((ROOT/"inputs/cadence_signals_raw.json").read_text())

def months(a,b="2025-01"):
 y,m=map(int,a.split("-")); ey,em=map(int,b.split("-"))
 while (y,m)<=(ey,em):
  yield f"{y:04d}-{m:02d}"; m+=1
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
    if not r or not r[0].isdigit():continue
    t=int(r[0]);t=t//1000 if t>10**15 else t
    out[t]=(float(r[1]),float(r[4]),float(r[7]))
 return out
def parse(s): return datetime.datetime.fromisoformat(s.replace("Z","+00:00"))
bars={s:load(s) for s in FIRST}
first={s:datetime.datetime.fromtimestamp(min(q)/1000,datetime.timezone.utc) for s,q in bars.items() if q}
eligible_from={s:first[s]+datetime.timedelta(days=730) for s in first}
eligible=[]; skipped=[]
for e in events:
 sym=e["symbol"];pub=parse(e["published_at"]);q=bars.get(sym,{})
 if sym not in eligible_from:
  skipped.append({**e,"reason":"no_price_data"});continue
 if pub<eligible_from[sym]:
  skipped.append({**e,"reason":"history_lt_730d","eligible_from":eligible_from[sym].isoformat()});continue
 ms=int(pub.timestamp()*1000); floor=(ms//H)*H
 hist=[floor-H*i for i in range(24,0,-1)]
 if any(t not in q for t in hist):
  skipped.append({**e,"reason":"missing_prior_24h"});continue
 qv=sum(q[t][2] for t in hist)
 if qv<5_000_000:
  skipped.append({**e,"reason":"qv_lt_5m","qv24":qv});continue
 eligible.append({**e,"qv24":qv,"eligible_from":eligible_from[sym].isoformat()})
summary={"raw_signals":len(events),"raw_symbols":len({x["symbol"] for x in events}),
         "eligible_signals":len(eligible),"eligible_symbols":len({x["symbol"] for x in eligible}),
         "eligible_long":sum(x["side"]=="LONG" for x in eligible),"eligible_short":sum(x["side"]=="SHORT" for x in eligible),
         "coverage_gate_pass":len(eligible)>=80 and len({x["symbol"] for x in eligible})>=8,
         "skipped_by_reason":{},"post_event_returns_read":False}
for x in skipped: summary["skipped_by_reason"][x["reason"]]=summary["skipped_by_reason"].get(x["reason"],0)+1
(ROOT/"inputs/eligible_signals.json").write_text(json.dumps(eligible,indent=2))
(ROOT/"inputs/eligibility_skipped.json").write_text(json.dumps(skipped,indent=2))
(ROOT/"results/eligibility_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
