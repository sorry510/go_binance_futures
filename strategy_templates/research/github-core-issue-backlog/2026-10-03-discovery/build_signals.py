import datetime,json
from collections import Counter
from pathlib import Path

ROOT=Path("strategy_templates/research/github-core-issue-backlog/2026-10-03-discovery")
IN=ROOT/"inputs"
MAP=json.loads((IN/"source_map.json").read_text())
START=datetime.date(2022,1,1)
END=datetime.date(2024,12,31)
DISC_START=datetime.date(2023,1,1)
DISC_END=datetime.date(2024,12,31)

all_signals=[]
coverage={}
for asset,repo in MAP.items():
    p=IN/f"{asset}_issues.json"
    if not p.exists():
        raise SystemExit(f"missing {p}")
    rows=json.loads(p.read_text())
    cnt=Counter(r["created_at"][:10] for r in rows)
    dates=[]; d=START
    while d<=END:
        dates.append(d); d+=datetime.timedelta(days=1)
    vals=[cnt.get(x.isoformat(),0) for x in dates]
    score={}
    for i,x in enumerate(dates):
        if i<34: continue
        cur=sum(vals[i-6:i+1])/7.0
        base=sum(vals[i-34:i-6])/28.0
        score[x]=cur-base
    n=0
    for x in dates:
        if x<DISC_START or x>DISC_END: continue
        prev=x-datetime.timedelta(days=1)
        if x not in score or prev not in score: continue
        a=score[prev]; b=score[x]
        if a<=0<b: side="SHORT"
        elif a>=0>b: side="LONG"
        else: continue
        all_signals.append({
          "asset":asset,"symbol":asset+"USDT","repo":repo,
          "signal_date":x.isoformat(),
          "signal_complete_at":(x+datetime.timedelta(days=1)).isoformat()+"T00:00:00Z",
          "side":side,"score_prev":a,"score_now":b,
          "issue_count_signal_day":cnt.get(x.isoformat(),0)
        }); n+=1
    coverage[asset]={"repo":repo,"issue_rows":len(rows),
                     "y2022":sum(r["created_at"].startswith("2022") for r in rows),
                     "y2023":sum(r["created_at"].startswith("2023") for r in rows),
                     "y2024":sum(r["created_at"].startswith("2024") for r in rows),
                     "raw_signals":n}
    print("SIGNALS",asset,n,coverage[asset],flush=True)

summary={"raw_signals":len(all_signals),"raw_symbols":len({x["asset"] for x in all_signals}),
         "long":sum(x["side"]=="LONG" for x in all_signals),
         "short":sum(x["side"]=="SHORT" for x in all_signals),
         "source_coverage":coverage,"post_token_returns_read":False}
(IN/"signals_raw.json").write_text(json.dumps(all_signals,indent=2))
(ROOT/"results/signal_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2),flush=True)
