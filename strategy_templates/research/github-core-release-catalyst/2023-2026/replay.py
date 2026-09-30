import csv,io,zipfile,urllib.request,urllib.error,datetime,json,math,time
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor,as_completed

UTC=datetime.timezone.utc
ROOT=Path(__file__).resolve().parent
CACHE=Path('/tmp/github_core_release_price_cache'); CACHE.mkdir(exist_ok=True)
DV='https://data.binance.vision/data/futures/um/monthly/klines'
UA='Mozilla/5.0'
FIRST_MONTH={'BTCUSDT':'2020-01','ETHUSDT':'2020-01','BNBUSDT':'2020-02','XRPUSDT':'2020-01',
             'ADAUSDT':'2020-01','AVAXUSDT':'2020-09','NEARUSDT':'2020-10','TRXUSDT':'2020-01',
             'OPUSDT':'2022-06','APTUSDT':'2022-10'}
END_MONTH='2025-01'

def months(start,end=END_MONTH):
 y,m=map(int,start.split('-')); ey,em=map(int,end.split('-'))
 while (y,m)<=(ey,em):
  yield f'{y:04d}-{m:02d}'
  m+=1
  if m==13:y+=1;m=1

def fetch(sym,mo):
 p=CACHE/f'{sym}-1h-{mo}.zip'
 if p.exists():return p
 u=f'{DV}/{sym}/1h/{sym}-1h-{mo}.zip'
 for k in range(4):
  try:
   req=urllib.request.Request(u,headers={'User-Agent':UA})
   p.write_bytes(urllib.request.urlopen(req,timeout=20).read());return p
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return p
   time.sleep(.4*(k+1))
  except Exception:time.sleep(.4*(k+1))
 p.write_bytes(b'');return p
def load_bars(sym):
 out={}
 for mo in months(FIRST_MONTH[sym]):
  p=fetch(sym,mo)
  if not p.exists() or p.stat().st_size==0:continue
  try:z=zipfile.ZipFile(p)
  except:continue
  fn=z.namelist()[0]
  for r in csv.reader(io.TextIOWrapper(z.open(fn))):
   if not r or not r[0].isdigit():continue
   t=int(r[0]); t=t//1000 if t>10**15 else t
   try:out[t]=(float(r[1]),float(r[4]),float(r[7]))
   except:pass
 return out

def parse_iso(s):
 return datetime.datetime.fromisoformat(s.replace('Z','+00:00')).astimezone(UTC)

def add2(dt):
 try:return dt.replace(year=dt.year+2)
 except:return dt.replace(year=dt.year+2,day=28)

events=json.load(open(ROOT/'inputs/discovery_universe.json'))
jobs=[(s,m) for s in FIRST_MONTH for m in months(FIRST_MONTH[s])]
print('PRICE_FILES',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=20) as ex:
 fs=[ex.submit(fetch,*j) for j in jobs]
 for i,f in enumerate(as_completed(fs),1):
  f.result()
  if i%100==0 or i==len(fs):print('PRICE_DOWNLOAD',i,'/',len(fs),flush=True)

bars={s:load_bars(s) for s in FIRST_MONTH}
first={s:datetime.datetime.fromtimestamp(min(q)/1000,UTC) for s,q in bars.items() if q}
eligible_from={s:add2(t) for s,t in first.items()}
eligible=[]; skipped=[]
H=3600000
for e in events:
 sym=e['symbol']; pub=parse_iso(e['published_at']); q=bars.get(sym,{})
 if sym not in eligible_from:
  skipped.append({**e,'reason':'no_price_data'});continue
 if pub<eligible_from[sym]:
  skipped.append({**e,'reason':'history_lt_2y','eligible_from':eligible_from[sym].isoformat()});continue
 pub_ms=int(pub.timestamp()*1000); floor_ms=(pub_ms//H)*H
 hist=[floor_ms-H*i for i in range(24,0,-1)]
 if any(t not in q for t in hist):
  skipped.append({**e,'reason':'missing_trailing_24h'});continue
 qv=sum(q[t][2] for t in hist)
 if qv<5_000_000:
  skipped.append({**e,'reason':'quote_volume_lt_5m','trailing_24h_qv':qv});continue
 entry_ms=(pub_ms//H+1)*H
 need=[entry_ms,entry_ms+3*H,entry_ms+11*H]
 if any(t not in q for t in need):
  skipped.append({**e,'reason':'missing_forward_bars'});continue
 entry=q[entry_ms][0]
 if entry<=0:continue
 row={**e,'eligible_from':eligible_from[sym].isoformat(),'trailing_24h_qv':qv,
      'entry_time':datetime.datetime.fromtimestamp(entry_ms/1000,UTC).isoformat(),
      'r1':math.log(q[entry_ms][1]/entry),'r4':math.log(q[entry_ms+3*H][1]/entry),
      'r12':math.log(q[entry_ms+11*H][1]/entry)}
 eligible.append(row)
def summ(es):
 if not es:return {'n':0,'symbols':0}
 sy=sorted(set(x['symbol'] for x in es));by={};pos=0
 for s in sy:
  a=[x for x in es if x['symbol']==s]
  d={'n':len(a),'mean1':sum(x['r1'] for x in a)/len(a),'mean4':sum(x['r4'] for x in a)/len(a),
     'mean12':sum(x['r12'] for x in a)/len(a),'win12':sum(x['r12']>0 for x in a)/len(a)}
  by[s]=d
  if d['mean12']>0:pos+=1
 return {'n':len(es),'symbols':len(sy),'positive_symbols12':pos,'breadth12':pos/len(sy),
         'mean1':sum(x['r1'] for x in es)/len(es),'mean4':sum(x['r4'] for x in es)/len(es),
         'mean12':sum(x['r12'] for x in es)/len(es),'win12':sum(x['r12']>0 for x in es)/len(es),
         'symbol_equal_mean12':sum(by[s]['mean12'] for s in sy)/len(sy),'by_symbol':by}

all_s=summ(eligible)
y23=summ([x for x in eligible if x['utc_date'].startswith('2023-')])
y24=summ([x for x in eligible if x['utc_date'].startswith('2024-')])
gate=(all_s.get('n',0)>=80 and all_s.get('symbols',0)>=8 and all_s.get('mean12',-9)>=0.0025
      and all_s.get('breadth12',0)>=0.60 and y23.get('mean12',-9)>0 and y24.get('mean12',-9)>0)
summary={'discovery_2023_2024':all_s,'2023':y23,'2024':y24,'gate_pass':gate,
         'gate':'n>=80, symbols>=8, mean12>=0.25%, breadth>=60%, 2023>0, 2024>0',
         'first_futures':{s:first[s].isoformat() for s in first},
         'eligible_from':{s:eligible_from[s].isoformat() for s in eligible_from},
         'skipped_count':len(skipped)}
json.dump(eligible,open(ROOT/'results/discovery_events.json','w'),indent=2)
json.dump(skipped,open(ROOT/'results/discovery_skipped.json','w'),indent=2)
json.dump(summary,open(ROOT/'results/discovery_summary.json','w'),indent=2)
print('SUMMARY',json.dumps(summary,indent=2),flush=True)
