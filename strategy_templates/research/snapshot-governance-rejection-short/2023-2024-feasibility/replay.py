import json,collections,datetime
from pathlib import Path
ROOT=Path(__file__).resolve().parent
p=ROOT/'inputs/all_proposals.json'
xs=json.load(open(p))
neg={'against','no','reject','rejected','do not approve','no action','oppose'}
pos={'for','yes','approve','approved','support'}
rows=[]
binary=0
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
    win=labels[0] if sc[0]>sc[1] else labels[1]
    if win<0:
        rows.append((q['asset'],q['symbol'],q['space'],q['id'],y,q['title'],ch,sc))
print('all',len(xs),'binary_clean',binary,'negative_wins',len(rows))
c=collections.Counter(r[0] for r in rows)
cy=collections.Counter(r[4] for r in rows)
print('assets',len(c),dict(c))
print('years',dict(cy))
for r in rows[:50]:
    print(r[4],r[0],r[5][:120],r[6],r[7])
