import csv,io,zipfile,json,math,datetime
from pathlib import Path
UTC=datetime.timezone.utc
ROOT=Path('/tmp/chain_stablecoin_bridged_share_out'); (ROOT/'results').mkdir(parents=True,exist_ok=True); (ROOT/'inputs').mkdir(exist_ok=True)
PRICE=Path('/tmp/chain_dex_residual_cache'); STABLE=Path('/tmp/chain_stablecoin_growth_cache')
UNIV=json.load(open('/Users/zhz/work/binance/go_binance_futures/strategy_templates/research/defillama-chain-stablecoin-supply-growth/2023-2026/inputs/universe.json'))
eligible={x['symbol']:datetime.date.fromisoformat(x['eligible_from']) for x in UNIV if x.get('status')=='ok'}
chain={x['symbol']:x['chain'] for x in UNIV if x.get('status')=='ok'}
def months(a,b):
 d=datetime.date.fromisoformat(a+'-01');e=datetime.date.fromisoformat(b+'-01');out=[]
 while d<=e:
  out.append(d.strftime('%Y-%m'));d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
 return out
def price(sym):
 out={}
 for mo in months('2022-12','2025-01'):
  p=PRICE/f'{sym}-{mo}.zip'
  if not p.exists(): p=STABLE/f'{sym}-{mo}.zip'
  if not p.exists() or p.stat().st_size==0: continue
  try:z=zipfile.ZipFile(p)
  except:continue
  for r in csv.reader(io.TextIOWrapper(z.open(z.namelist()[0]))):
   if not r or not r[0].isdigit(): continue
   t=int(r[0]); t=t//1000 if t>10**15 else t
   d=datetime.datetime.fromtimestamp(t/1000,UTC).date()
   out[d]=(float(r[1]),float(r[4]),float(r[7]))
 return out
def shares(ch):
 p=STABLE/('stable-'+ch.replace(' ','_')+'.json')
 if not p.exists(): return {}
 try:o=json.load(open(p))
 except:return {}
 out={}
 for x in o if isinstance(o,list) else []:
  try:
   d=datetime.datetime.fromtimestamp(int(x['date']),UTC).date()
   total=float(((x.get('totalCirculatingUSD') or {}).get('peggedUSD')) or 0)
   bridged=float(((x.get('totalBridgedToUSD') or {}).get('peggedUSD')) or 0)
   if total>0 and bridged>=0: out[d]=max(0.0,min(1.5,bridged/total))
  except: pass
 return out
def mean(a): return sum(a)/len(a) if a else 0.0
events=[]; coverage=[]
for sym,ch in chain.items():
 px=price(sym); sh=shares(ch)
 ds=sorted(set(px)&set(sh)); rows=[(d,*px[d],sh[d]) for d in ds]
 n=0
 for i in range(8,len(rows)-8):
  d=rows[i][0]
  if d.year not in (2023,2024) or d<eligible[sym] or rows[i][3]<5_000_000: continue
  if not all((rows[j][0]-rows[j-1][0]).days==1 for j in range(i-7,i+8)): continue
  delta=rows[i][4]-rows[i-7][4]; prev=rows[i-1][4]-rows[i-8][4]
  direction=0
  if prev<=0 and delta>0: direction=1
  elif prev>=0 and delta<0: direction=-1
  else: continue
  entry=rows[i+1][1]
  if entry<=0: continue
  events.append({'symbol':sym,'chain':ch,'signal_date':d.isoformat(),'year':d.year,
                 'direction':'LONG' if direction>0 else 'SHORT','share':rows[i][4],
                 'delta7':delta,'r1':direction*math.log(rows[i+1][2]/entry),
                 'r3':direction*math.log(rows[i+3][2]/entry),
                 'r7':direction*math.log(rows[i+7][2]/entry)})
  n+=1
 coverage.append({'symbol':sym,'chain':ch,'aligned_days':len(rows),'events':n,
                  'share_nonzero_days':sum(1 for r in rows if r[4]>0),
                  'share_first':rows[0][0].isoformat() if rows else None})
 print(sym,ch,'days',len(rows),'nonzero',sum(1 for r in rows if r[4]>0),'events',n,flush=True)
def summary(es):
 sy=sorted(set(x['symbol'] for x in es)); by={};pos=0
 for s in sy:
  q=[x for x in es if x['symbol']==s]; m7=mean([x['r7'] for x in q])
  by[s]={'n':len(q),'mean1':mean([x['r1'] for x in q]),'mean3':mean([x['r3'] for x in q]),'mean7':m7,'win7':mean([1 if x['r7']>0 else 0 for x in q])}
  if m7>0:pos+=1
 return {'n':len(es),'symbols':len(sy),'mean1':mean([x['r1'] for x in es]),'mean3':mean([x['r3'] for x in es]),'mean7':mean([x['r7'] for x in es]),'win7':mean([1 if x['r7']>0 else 0 for x in es]),'positive_symbols7':pos,'breadth7':pos/len(sy) if sy else 0,'by_symbol':by}
s=summary(events); y23=summary([x for x in events if x['year']==2023]); y24=summary([x for x in events if x['year']==2024])
gate=s['n']>=80 and s['symbols']>=8 and s['mean7']>=0.0025 and s['breadth7']>=0.60 and y23.get('mean7',0)>0 and y24.get('mean7',0)>0
out={'discovery_2023_2024':s,'2023':y23,'2024':y24,'gate_pass':gate}
(ROOT/'inputs'/'coverage.json').write_text(json.dumps(coverage,indent=2))
(ROOT/'results'/'events.json').write_text(json.dumps(events,indent=2))
(ROOT/'results'/'summary.json').write_text(json.dumps(out,indent=2))
print('SUMMARY',json.dumps(out,indent=2),flush=True)
