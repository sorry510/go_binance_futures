import json, datetime, collections
from pathlib import Path

ROOT=Path(__file__).resolve().parent
SRC=ROOT.parent.parent/'snapshot-governance-rejection-short'/'2023-2024-feasibility'/'inputs'/'all_proposals.json'
xs=json.load(open(SRC))
neg={'against','no','reject','rejected','do not approve','no action','oppose'}
pos={'for','yes','approve','approved','support'}
rows=[]; binary=0; negwins=0; ties=0
for q in xs:
    y=datetime.datetime.fromtimestamp(q['end'],datetime.timezone.utc).year
    if y not in (2023,2024):
        continue
    ch=[str(x).strip().lower() for x in q.get('choices',[])]
    sc=q.get('scores') or []
    if len(ch)!=2 or len(sc)!=2:
        continue
    labels=[]
    for c in ch:
        if c in pos or c.startswith('for '):
            labels.append(1)
        elif c in neg or c.startswith('against '):
            labels.append(-1)
        else:
            labels.append(0)
    if sorted(labels)!=[-1,1]:
        continue
    binary+=1
    if sc[0]==sc[1]:
        ties+=1
        continue
    win=labels[0] if sc[0]>sc[1] else labels[1]
    if win<0:
        negwins+=1
        continue
    rows.append({
        'asset':q['asset'],'symbol':q['symbol'],'space':q['space'],'proposal_id':q['id'],
        'end':q['end'],'year':y,'title':q['title'],'choices':q['choices'],'scores':sc
    })

(ROOT/'inputs'/'positive_wins.json').write_text(json.dumps(rows,ensure_ascii=False,indent=2))
summary={
    'all_proposals':len(xs),'binary_clean':binary,'positive_wins':len(rows),
    'negative_wins':negwins,'ties':ties,
    'assets':dict(collections.Counter(x['asset'] for x in rows)),
    'spaces':len(set(x['space'] for x in rows)),
    'years':dict(collections.Counter(str(x['year']) for x in rows)),
}
print(json.dumps(summary,ensure_ascii=False,indent=2))
for x in rows:
    print(x['year'],x['asset'],x['space'],x['end'],x['title'][:120])
