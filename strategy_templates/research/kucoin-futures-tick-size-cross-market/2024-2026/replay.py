import json,datetime,collections
from pathlib import Path
ROOT=Path(__file__).resolve().parent
R=json.load(open(ROOT/'results'/'discovery.json'))
def stat(xs,k):
 v=[x[k] for x in xs];return {'n':len(v),'mean':sum(v)/len(v) if v else None,'win':sum(x>0 for x in v)/len(v) if v else None}
print('r1',stat(R,'r1'));print('r4',stat(R,'r4'));print('r12',stat(R,'r12'))
by=collections.defaultdict(list)
for x in R:by[x['article_id']].append(x['r12'])
bm=[sum(v)/len(v) for v in by.values()]
print('batches',len(bm),'batch_equal_mean12',sum(bm)/len(bm),'positive_batches',sum(x>0 for x in bm))
