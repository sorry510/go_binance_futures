from pathlib import Path
import csv, json, random, math
from datetime import datetime, timezone
from collections import defaultdict

src=Path('strategy_templates/research/id121-v54-exact-exit/2026-09-30-audit/results/trades.csv')
rows=[]
for r in csv.DictReader(src.open()):
    if r['strategy']!='ID121': continue
    x=float(r['norm_return'])
    ts=int(r['entry_time'])
    dt=datetime.fromtimestamp(ts/1000,timezone.utc)
    rows.append({'symbol':r['symbol'],'ts':ts,'dt':dt,'ret':x,'side':r['side']})
rows.sort(key=lambda x:x['ts'])

def agg(rs):
    gp=sum(x['ret'] for x in rs if x['ret']>0)
    gl=-sum(x['ret'] for x in rs if x['ret']<0)
    net=sum(x['ret'] for x in rs)
    pf=gp/gl if gl>0 else (999.0 if gp>0 else 0.0)
    return {'trades':len(rs),'pf':pf,'net':net,'wins':sum(x['ret']>0 for x in rs)}

overall=agg(rows)
symbols=sorted(set(x['symbol'] for x in rows))
by_symbol={s:agg([x for x in rows if x['symbol']==s]) for s in symbols}
loo={}
for s in symbols:
    loo[s]=agg([x for x in rows if x['symbol']!=s])

months=defaultdict(list)
quarters=defaultdict(list)
for x in rows:
    m=x['dt'].strftime('%Y-%m')
    q=f"{x['dt'].year}-Q{(x['dt'].month-1)//3+1}"
    months[m].append(x)
    quarters[q].append(x)
by_month={m:agg(months[m]) for m in sorted(months)}
by_quarter={q:agg(quarters[q]) for q in sorted(quarters)}

month_keys=sorted(months)
rng=random.Random(121)
boot=[]
for _ in range(10000):
    sample=[]
    for _j in range(len(month_keys)):
        k=rng.choice(month_keys)
        sample.extend(months[k])
    boot.append(agg(sample)['pf'])
boot.sort()
def quant(v,p):
    if not v:return None
    pos=(len(v)-1)*p
    lo=int(math.floor(pos)); hi=int(math.ceil(pos))
    if lo==hi:return v[lo]
    return v[lo]*(hi-pos)+v[hi]*(pos-lo)

pos_months=sum(v['net']>0 for v in by_month.values())
pf_months=sum(v['pf']>1 for v in by_month.values())
pos_quarters=sum(v['net']>0 for v in by_quarter.values())
pf_quarters=sum(v['pf']>1 for v in by_quarter.values())

net_sorted=sorted(((s,v['net']) for s,v in by_symbol.items()),key=lambda z:z[1],reverse=True)
top3=sum(max(0,x[1]) for x in net_sorted[:3])
positive_total=sum(max(0,v['net']) for v in by_symbol.values())

out={
 'source_trades':len(rows),
 'overall':overall,
 'symbols':by_symbol,
 'leave_one_symbol_out':loo,
 'leave_one_out_pf_min':min(v['pf'] for v in loo.values()),
 'leave_one_out_pf_max':max(v['pf'] for v in loo.values()),
 'leave_one_out_net_min':min(v['net'] for v in loo.values()),
 'leave_one_out_net_max':max(v['net'] for v in loo.values()),
 'symbol_net_ranking':net_sorted,
 'top3_share_of_positive_symbol_net': top3/positive_total if positive_total>0 else None,
 'months':by_month,
 'month_count':len(by_month),
 'positive_net_months':pos_months,
 'pf_gt_1_months':pf_months,
 'quarters':by_quarter,
 'quarter_count':len(by_quarter),
 'positive_net_quarters':pos_quarters,
 'pf_gt_1_quarters':pf_quarters,
 'monthly_block_bootstrap':{
   'replicates':10000,'seed':121,'blocks_per_sample':len(month_keys),
   'pf_q01':quant(boot,.01),'pf_q05':quant(boot,.05),'pf_q10':quant(boot,.10),
   'pf_median':quant(boot,.5),'pf_q90':quant(boot,.90),'pf_q95':quant(boot,.95),'pf_q99':quant(boot,.99),
   'fraction_pf_le_1':sum(x<=1 for x in boot)/len(boot),
   'fraction_pf_le_observed':sum(x<=overall['pf'] for x in boot)/len(boot)
 }
}
outp=Path('strategy_templates/research/id121-exact-exit-robustness/2026-10-01-audit/results/summary.json')
outp.write_text(json.dumps(out,indent=2)+'\n')
print('OVERALL',overall)
print('LOO PF min/max',out['leave_one_out_pf_min'],out['leave_one_out_pf_max'])
print('MONTHS',len(by_month),'positive net',pos_months,'PF>1',pf_months)
print('QUARTERS',len(by_quarter),'positive net',pos_quarters,'PF>1',pf_quarters)
print('TOP3 share positive net',out['top3_share_of_positive_symbol_net'])
print('BOOT',out['monthly_block_bootstrap'])
print('BOTTOM SYMBOLS',sorted(((s,v['pf'],v['net']) for s,v in by_symbol.items()),key=lambda z:z[2])[:5])
print('TOP SYMBOLS',sorted(((s,v['pf'],v['net']) for s,v in by_symbol.items()),key=lambda z:z[2],reverse=True)[:5])
print('MONTH PFs')
for m,v in by_month.items(): print(m,v['trades'],round(v['pf'],3),round(v['net'],4))
