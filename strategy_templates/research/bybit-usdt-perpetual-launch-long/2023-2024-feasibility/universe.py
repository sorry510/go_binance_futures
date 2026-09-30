import urllib.request,re,json,time,datetime
UTC=datetime.timezone.utc
UA='Mozilla/5.0'
BASE='https://announcements.bybit.com/en/'
def page(n):
    u=f'{BASE}?category=new_crypto&page={n}'
    last=None
    for k in range(5):
        try:
            h=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=25).read().decode('utf-8','ignore')
            m=re.search(r'<script id="__NEXT_DATA__" type="application/json">(.*?)</script>',h,re.S)
            o=json.loads(m.group(1))
            return o['props']['pageProps']['articleInitEntity']
        except Exception as e:
            last=e;time.sleep(.7*(k+1))
    raise last
ev=[]
seen=set()
for p in range(45,95):
    e=page(p);a=e['list']
    years=[]
    for x in a:
        ts=x.get('date_timestamp') or x.get('publish_time') or 0
        if not ts:continue
        y=datetime.datetime.fromtimestamp(ts,UTC).year;years.append(y)
        if y not in (2023,2024):continue
        title=x.get('title','');desc=x.get('description','')
        text=title+' '+desc
        if 'perpetual' not in text.lower():continue
        syms=re.findall(r'(?<![A-Z0-9])([0-9]*[A-Z][A-Z0-9]{1,20}USDT)(?![A-Z0-9])',text)
        if not syms:continue
        topics=x.get('topics') or []
        if not any('Deriv' in str(t) for t in topics) and 'perpetual contract' not in text.lower():
            continue
        for s in syms:
            key=(s,ts)
            if key in seen:continue
            seen.add(key)
            ev.append({'bybit_symbol':s,'event_ts':ts,'title':title,'description':desc,
                       'topics':topics,'url':x.get('url'),'objectID':x.get('objectID'),
                       'date_timestamp':x.get('date_timestamp'),'start_date_timestamp':x.get('start_date_timestamp'),
                       'publish_time':x.get('publish_time')})
    print('PAGE',p,'YEARS',min(years) if years else None,max(years) if years else None,'EVENTS',len(ev),flush=True)
    if years and min(years)<2023:break
    time.sleep(.15)
ev.sort(key=lambda x:(x['event_ts'],x['bybit_symbol']))
print('TOTAL_EVENTS',len(ev),'UNIQUE_SYMBOLS',len(set(x['bybit_symbol'] for x in ev)),flush=True)
from collections import Counter
print('YEARS',Counter(datetime.datetime.fromtimestamp(x['event_ts'],UTC).year for x in ev),flush=True)
for x in ev[:20]:print('EV',datetime.datetime.fromtimestamp(x['event_ts'],UTC).isoformat(),x['bybit_symbol'],x['title'],flush=True)
for x in ev[-20:]:print('EV',datetime.datetime.fromtimestamp(x['event_ts'],UTC).isoformat(),x['bybit_symbol'],x['title'],flush=True)
open('/tmp/bybit_usdt_launch_events.json','w').write(json.dumps(ev,ensure_ascii=False,indent=2))
