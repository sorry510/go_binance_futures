import datetime,json
from collections import Counter
from pathlib import Path

ROOT=Path("strategy_templates/research/github-core-issue-resolution-balance/2026-10-03-discovery")
CREATED_ROOT=Path("strategy_templates/research/github-core-issue-backlog/2026-10-03-discovery/inputs")
IN=ROOT/"inputs"
MAP=json.loads((IN/"source_map.json").read_text())
START=datetime.date(2022,1,1)
END=datetime.date(2024,12,31)
DSTART=datetime.date(2023,1,1)
DEND=datetime.date(2024,12,31)

signals=[]; source={}
for asset,repo in MAP.items():
    created=json.loads((CREATED_ROOT/f"{asset}_issues.json").read_text())
    closed=json.loads((IN/f"{asset}_closures.json").read_text())
    cc=Counter(x["created_at"][:10] for x in created)
    xc=Counter(x["closed_at"][:10] for x in closed)

    dates=[]; d=START
    while d<=END:
        dates.append(d); d+=datetime.timedelta(days=1)

    balance={}
    for i,d in enumerate(dates):
        if i<6: continue
        week=dates[i-6:i+1]
        balance[d]=sum(xc.get(x.isoformat(),0) for x in week)-sum(cc.get(x.isoformat(),0) for x in week)

    n=0
    for d in dates:
        if d<DSTART or d>DEND: continue
        prev=d-datetime.timedelta(days=1)
        if d not in balance or prev not in balance: continue
        a=float(balance[prev]); b=float(balance[d])
        if a<=0<b:
            side="LONG"
        elif a>=0>b:
            side="SHORT"
        else:
            continue
        signals.append({
          "asset":asset,"symbol":asset+"USDT","repo":repo,
          "signal_date":d.isoformat(),
          "signal_complete_at":(d+datetime.timedelta(days=1)).isoformat()+"T00:00:00Z",
          "side":side,"balance_prev":a,"balance_now":b,
          "created_7d":sum(cc.get(x.isoformat(),0) for x in dates[dates.index(d)-6:dates.index(d)+1]),
          "closed_7d":sum(xc.get(x.isoformat(),0) for x in dates[dates.index(d)-6:dates.index(d)+1])
        }); n+=1

    source[asset]={"repo":repo,"created_rows":len(created),"closed_rows":len(closed),"raw_signals":n}
    print("SIGNALS",asset,n,source[asset],flush=True)

summary={"version":"v177","raw_signals":len(signals),"raw_symbols":len({x["asset"] for x in signals}),
         "long":sum(x["side"]=="LONG" for x in signals),"short":sum(x["side"]=="SHORT" for x in signals),
         "source":source,"post_event_returns_read":False}
(IN/"signals_raw.json").write_text(json.dumps(signals,indent=2))
(ROOT/"results/signal_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2),flush=True)
