import urllib.request,json,re,datetime,time,decimal
codes=[
('2024-10-24','e54c0b97f8c445058f419874d7d605d3'),('2024-09-17','2e11b3d512fa4027b221b7b518fafe74'),
('2024-07-15','dcd1e0044c944469bbbfe821ea8883e5'),('2024-06-20','eb8c059da7ce44ce8a30df7536045b2b'),
('2024-05-16','e285c92c01ef4d40954cbb50ce6e2de5'),('2024-04-15','9612f699890f4740bd0c731a9d9543c2'),
('2024-02-19','9f85f3d4f9d44c0fa16550e00fcfc72f'),('2024-01-12','56ab05eb7c334cdda7b7401c6e17e0f0'),
('2023-10-05','a621f30d706b4a84b0fe89cd04537f5a'),('2023-08-10','aa7335ef8d784218995a6c04c2512663'),
('2023-07-12','98fe5090addb4775bd53c7a1da12687a'),('2023-06-09','0d89c7051f964e37a71420d589f75cc2'),
('2023-05-11','b873155acb324bd3839136576f81367a'),('2023-04-11','18e8432047844f628ca9a78c42bc4ca0'),
('2023-03-09','674ef90dae164ce2b956a65d9e945945'),('2023-02-06','75b9608892024db785b46a06df721464'),
('2023-01-12','eec6f8e3de5b42ad901d2020a4e0a966')]
H={'User-Agent':'Mozilla/5.0','Accept':'application/json'}
def flat(n):
 if isinstance(n,dict):
  if n.get('node')=='text':return n.get('text','')
  s=''.join(flat(c) for c in n.get('child',[]) or [])
  return (' '+s+' ') if n.get('tag') in {'p','div','td','th','tr','table','li','ul','ol','br'} else s
 if isinstance(n,list):return ' '.join(flat(x) for x in n)
 return ''
events=[]
for batch,code in codes:
 u='https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query?articleCode='+code
 for k in range(5):
  try:
   o=json.loads(urllib.request.urlopen(urllib.request.Request(u,headers=H),timeout=20).read());break
  except Exception:
   time.sleep(.5*(k+1))
 else:
  print('FETCH_FAIL',code);continue
 d=o.get('data') or {};body=d.get('body','{}')
 try:root=json.loads(body) if isinstance(body,str) else body
 except:root={}
 txt=re.sub(r'\s+',' ',flat(root)).strip()
 # Split by explicit "By date time (UTC)" sections; if absent, use first deadline in intro.
 marks=list(re.finditer(r'By\s+(20\d\d-\d\d-\d\d)\s+(\d\d:\d\d)\s*\(UTC\)\s*:',txt,re.I))
 segments=[]
 if marks:
  for i,m in enumerate(marks):
   segments.append((m.group(1),m.group(2),txt[m.end():marks[i+1].start() if i+1<len(marks) else len(txt)]))
 else:
  m=re.search(r'by\s+(20\d\d-\d\d-\d\d)\s+(\d\d:\d\d)\s*\(UTC\)',txt,re.I)
  if not m:
   print('NO_EFFECTIVE',code,d.get('title'));continue
  segments=[(m.group(1),m.group(2),txt)]
 n=0
 for day,hm,seg in segments:
  et=int(datetime.datetime.fromisoformat(day+'T'+hm+':00+00:00').timestamp()*1000)
  for pair,old,new in re.findall(r'\b([A-Z0-9]+/USDT)\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9]+(?:\.[0-9]+)?)\b',seg):
   a,b=decimal.Decimal(old),decimal.Decimal(new)
   if a==b:continue
   base=pair.split('/')[0]
   if base.endswith('UP') or base.endswith('DOWN'):continue
   events.append({'batch':batch,'code':code,'title':d.get('title'),'effective_ms':et,'effective_utc':day+'T'+hm+':00Z','spot_pair':pair,'base':base,'old_tick':old,'new_tick':new,'direction':'LONG' if b<a else 'SHORT'})
   n+=1
 print(batch,d.get('title'),'USDT_EVENTS',n)
print('EVENTS',len(events),'LONG',sum(x['direction']=='LONG' for x in events),'SHORT',sum(x['direction']=='SHORT' for x in events),'BATCHES',len(set(x['effective_utc'] for x in events)))
for x in events:print('EV',x['effective_utc'],x['spot_pair'],x['old_tick'],'->',x['new_tick'],x['direction'])
open('/tmp/spot_tick_size_events.json','w').write(json.dumps(events,ensure_ascii=False,indent=2))
