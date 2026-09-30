import argparse,csv,datetime,io,json,math,re,subprocess,time,urllib.error,urllib.parse,urllib.request,xml.etree.ElementTree as ET,zipfile
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path

UTC=datetime.timezone.utc
ROOT=Path(__file__).resolve().parent
INPUTS=ROOT/'inputs'; RESULTS=ROOT/'results'
INPUTS.mkdir(exist_ok=True); RESULTS.mkdir(exist_ok=True)
CACHE=Path('/tmp/protocol_dex_share_cache'); CACHE.mkdir(exist_ok=True)
S3='https://s3-ap-northeast-1.amazonaws.com/data.binance.vision'
DV='https://data.binance.vision/data/futures/um/monthly/klines'
UA='Mozilla/5.0'
MIN_QV=5_000_000.0
DISCOVERY_YEARS={2023,2024}
CANDIDATES={'UNI','RAY','CRV','SUSHI','DODO','BAL','RUNE','WOO','KNC','INJ'}

def add2(d):
    try:return d.replace(year=d.year+2)
    except:return d.replace(year=d.year+2,day=28)
def curl_json(url,name):
    p=CACHE/name
    if not p.exists() or p.stat().st_size<100:
        subprocess.run(['curl','--http1.1','-L','--retry','2','--retry-all-errors',
                        '--connect-timeout','8','--max-time','40','-sS','-o',str(p),url],check=True)
    return json.load(open(p))

def first_kline(sym):
    pref=f'data/futures/um/monthly/klines/{sym}/1d/'
    u=S3+'?'+urllib.parse.urlencode({'prefix':pref,'max-keys':'5'})
    root=ET.fromstring(urllib.request.urlopen(u,timeout=20).read())
    ns={'s':'http://s3.amazonaws.com/doc/2006-03-01/'}
    keys=[x.text for x in root.findall('.//s:Contents/s:Key',ns) if x.text and x.text.endswith('.zip')]
    if not keys:return None
    key=min(keys)
    b=urllib.request.urlopen(urllib.request.Request('https://data.binance.vision/'+key,headers={'User-Agent':UA}),timeout=20).read()
    z=zipfile.ZipFile(io.BytesIO(b)); fn=z.namelist()[0]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if r and r[0].isdigit():
            t=int(r[0]); t=t//1000 if t>10**15 else t
            return datetime.datetime.fromtimestamp(t/1000,UTC)
    return None

def months():
    out=[]; d=datetime.date(2022,1,1); end=datetime.date(2026,9,1)
    while d<=end:
        out.append(d.strftime('%Y-%m'))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out
def fetch_month(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    u=f'{DV}/{sym}/1d/{sym}-1d-{mo}.zip'
    last=None
    for k in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
            p.write_bytes(b)
            return p
        except urllib.error.HTTPError as e:
            if e.code==404:
                p.write_bytes(b'')
                return p
            last=e
        except Exception as e:
            last=e
        time.sleep(0.5*(k+1))
    raise RuntimeError(f'failed monthly kline {sym} {mo}: {last}')

def price_daily(sym):
    out={}
    for mo in months():
        p=fetch_month(sym,mo)
        if not p.exists() or p.stat().st_size==0:continue
        try:z=zipfile.ZipFile(p)
        except:continue
        fn=z.namelist()[0]
        for r in csv.reader(io.TextIOWrapper(z.open(fn))):
            if not r or not r[0].isdigit():continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            d=datetime.datetime.fromtimestamp(t/1000,UTC).date()
            out[d]=(float(r[1]),float(r[4]),float(r[7]))
    return out

def chart_map(o):
    out={}
    for x in o.get('totalDataChart') or []:
        try:
            d=datetime.datetime.fromtimestamp(int(x[0]),UTC).date()
            v=float(x[1])
            if v>=0:out[d]=v
        except:pass
    return out
def load_dex_inputs():
    ov=curl_json('https://api.llama.fi/overview/dexs?excludeTotalDataChartBreakdown=true&dataType=dailyVolume','dex_overview_full.json')
    ps=curl_json('https://api.llama.fi/protocols','protocols.json')
    byid={str(p.get('id')):p for p in ps}
    comps=defaultdict(list)
    for d in ov.get('protocols') or []:
        p=byid.get(str(d.get('defillamaId')))
        if not p or p.get('category')!='Dexs':continue
        sym=(p.get('symbol') or '').upper().strip()
        if sym not in CANDIDATES:continue
        slug=d.get('slug')
        if slug:comps[sym].append(slug)
    global_daily=chart_map(ov)
    component_daily={}
    for sym in sorted(comps):
        for slug in comps[sym]:
            key=f'dex_{slug}.json'
            u='https://api.llama.fi/summary/dexs/'+urllib.parse.quote(slug)+'?excludeTotalDataChartBreakdown=true&dataType=dailyVolume'
            component_daily[slug]=chart_map(curl_json(u,key))
    snap={'components':dict(comps),'global_days':len(global_daily)}
    (INPUTS/'components.json').write_text(json.dumps(snap,indent=2))
    return global_daily,comps,component_daily

def aggregate_protocol_daily(comps,component_daily):
    out={}
    all_dates=set()
    for slug in comps:all_dates.update(component_daily.get(slug,{}))
    for d in all_dates:
        out[d]=sum(component_daily.get(slug,{}).get(d,0.0) for slug in comps)
    return out
def build_events(period):
    global_daily,comps,component_daily=load_dex_inputs()
    firsts={}
    with ThreadPoolExecutor(max_workers=8) as ex:
        fs={ex.submit(first_kline,s+'USDT'):s for s in sorted(CANDIDATES)}
        for f in as_completed(fs):
            s=fs[f]; firsts[s]=f.result()
            print('FIRST',s,firsts[s],flush=True)
    prices={}
    with ThreadPoolExecutor(max_workers=11) as ex:
        fs={ex.submit(price_daily,s+'USDT'):s for s in sorted(CANDIDATES)}
        for f in as_completed(fs):
            s=fs[f]; prices[s]=f.result()
            print('PRICE',s,len(prices[s]),flush=True)
    universe={}
    for s in sorted(CANDIDATES):
        first=firsts.get(s)
        universe[s]={'first_futures':first.isoformat() if first else None,
                     'eligible_from':add2(first).date().isoformat() if first else None,
                     'components':comps.get(s,[]),'price_days':len(prices.get(s,{}))}
    (INPUTS/'universe.json').write_text(json.dumps(universe,indent=2))
    years={'discovery':DISCOVERY_YEARS,'oos1':{2025},'oos2':{2026}}[period]
    events=[]
    for s in sorted(CANDIDATES):
        first=firsts.get(s)
        px=prices.get(s,{})
        if not first or not comps.get(s):
            continue
        proto=aggregate_protocol_daily(comps[s],component_daily)
        dates=sorted(global_daily)
        share7={}
        mom={}
        for d in dates:
            ds=[d-datetime.timedelta(days=k) for k in range(7)]
            gv=sum(global_daily.get(x,0.0) for x in ds)
            pv=sum(proto.get(x,0.0) for x in ds)
            if gv>0 and pv>0:
                share7[d]=pv/gv
        for d,v in share7.items():
            prev=d-datetime.timedelta(days=7)
            if prev in share7 and share7[prev]>0:
                mom[d]=math.log(v/share7[prev])
        eligible=add2(first).date()
        for d in sorted(mom):
            if d.year not in years or d<eligible:
                continue
            y=mom[d]
            yp=mom.get(d-datetime.timedelta(days=1))
            if yp is None:
                continue
            direction=0
            if yp<=0 and y>0:
                direction=1
            elif yp>=0 and y<0:
                direction=-1
            if direction==0:
                continue
            if d not in px or px[d][2]<MIN_QV:
                continue
            d1=d+datetime.timedelta(days=1)
            d3=d+datetime.timedelta(days=3)
            d7=d+datetime.timedelta(days=7)
            if d1 not in px or d3 not in px or d7 not in px:
                continue
            entry=px[d1][0]
            if entry<=0:
                continue
            events.append({
                'symbol':s,
                'signal_date':d.isoformat(),
                'year':d.year,
                'direction':'LONG' if direction>0 else 'SHORT',
                'share7':share7[d],
                'momentum7':y,
                'quote_volume':px[d][2],
                'r1':direction*math.log(px[d1][1]/entry),
                'r3':direction*math.log(px[d3][1]/entry),
                'r7':direction*math.log(px[d7][1]/entry)
            })
    return events,universe
def summarize(events):
    if not events:
        return {'n':0,'symbols':0}
    syms=sorted({e['symbol'] for e in events})
    by_symbol={}
    positive=0
    for s in syms:
        rows=[e for e in events if e['symbol']==s]
        mean7=sum(e['r7'] for e in rows)/len(rows)
        by_symbol[s]={
            'n':len(rows),
            'mean1':sum(e['r1'] for e in rows)/len(rows),
            'mean3':sum(e['r3'] for e in rows)/len(rows),
            'mean7':mean7,
            'win7':sum(e['r7']>0 for e in rows)/len(rows)
        }
        if mean7>0:
            positive+=1
    return {
        'n':len(events),
        'symbols':len(syms),
        'mean1':sum(e['r1'] for e in events)/len(events),
        'mean3':sum(e['r3'] for e in events)/len(events),
        'mean7':sum(e['r7'] for e in events)/len(events),
        'win7':sum(e['r7']>0 for e in events)/len(events),
        'positive_symbols7':positive,
        'breadth7':positive/len(syms),
        'by_symbol':by_symbol
    }
def main():
    ap=argparse.ArgumentParser()
    ap.add_argument('--period',choices=['discovery','oos1','oos2'],required=True)
    args=ap.parse_args()
    events,universe=build_events(args.period)
    summary=summarize(events)
    (RESULTS/(args.period+'_events.json')).write_text(json.dumps(events,indent=2))
    (RESULTS/(args.period+'_summary.json')).write_text(json.dumps(summary,indent=2))
    print('SUMMARY',json.dumps(summary,indent=2))

if __name__=='__main__':
    main()
