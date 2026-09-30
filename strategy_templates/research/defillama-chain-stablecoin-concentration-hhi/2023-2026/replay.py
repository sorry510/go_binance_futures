import csv, io, zipfile, urllib.request, urllib.parse, urllib.error
import datetime, json, math, time, subprocess, statistics
import xml.etree.ElementTree as ET
from pathlib import Path
from functools import lru_cache

UTC=datetime.timezone.utc
UA='Mozilla/5.0'
ROOT=Path(__file__).resolve().parent
CACHE=Path('/tmp/stablecoin_hhi_audit')
CACHE.mkdir(exist_ok=True)
DV='https://data.binance.vision/data/futures/um/monthly/klines'
S3='https://s3-ap-northeast-1.amazonaws.com/data.binance.vision'
CHAIN_MAP={
 'BTCUSDT':'Bitcoin','ETHUSDT':'Ethereum','BNBUSDT':'BSC','SOLUSDT':'Solana',
 'AVAXUSDT':'Avalanche','ADAUSDT':'Cardano','NEARUSDT':'Near','TRXUSDT':'Tron',
 'MATICUSDT':'Polygon','FTMUSDT':'Fantom','OPUSDT':'OP Mainnet','APTUSDT':'Aptos',
 'ARBUSDT':'Arbitrum','SUIUSDT':'Sui','XRPUSDT':'XRPL'
}

def get_bytes(url,tries=5):
    last=None
    for k in range(tries):
        try:
            return urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':UA}),timeout=30).read()
        except Exception as e:
            last=e; time.sleep(.5*(k+1))
    raise last

def add2(d):
    try:return d.replace(year=d.year+2)
    except ValueError:return d.replace(year=d.year+2,day=28)

@lru_cache(None)
def first_kline(sym):
    pref=f'data/futures/um/monthly/klines/{sym}/1d/'
    url=S3+'?'+urllib.parse.urlencode({'prefix':pref,'max-keys':'5'})
    ns={'s':'http://s3.amazonaws.com/doc/2006-03-01/'}
    root=ET.fromstring(get_bytes(url))
    keys=[x.text for x in root.findall('.//s:Contents/s:Key',ns) if x.text and x.text.endswith('.zip')]
    if not keys:return None
    raw=get_bytes('https://data.binance.vision/'+min(keys))
    z=zipfile.ZipFile(io.BytesIO(raw))
    for r in csv.reader(io.TextIOWrapper(z.open(z.namelist()[0]))):
        if r and r[0].isdigit():
            t=int(r[0]); t=t//1000 if t>10**15 else t
            return datetime.datetime.fromtimestamp(t/1000,UTC)
    return None

def month_span(a,b):
    out=[];d=datetime.date.fromisoformat(a+'-01');e=datetime.date.fromisoformat(b+'-01')
    while d<=e:
        out.append(d.strftime('%Y-%m'))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out

def fetch_k(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    for shared_root in ['/tmp/chain_stablecoin_growth_cache','/tmp/chain_dex_residual_cache']:
        q=Path(shared_root)/f'{sym}-{mo}.zip'
        if q.exists():return q
    url=f'{DV}/{sym}/1d/{sym}-1d-{mo}.zip'
    try:p.write_bytes(get_bytes(url,4))
    except urllib.error.HTTPError as e:
        if e.code==404:p.write_bytes(b'')
        else:raise
    except Exception:p.write_bytes(b'')
    return p

@lru_cache(None)
def price_daily_window(sym,a,b):
    out={}
    for mo in month_span(a,b):
        p=fetch_k(sym,mo)
        if not p.exists() or p.stat().st_size==0:continue
        try:z=zipfile.ZipFile(p)
        except Exception:continue
        for r in csv.reader(io.TextIOWrapper(z.open(z.namelist()[0]))):
            if not r or not r[0].isdigit():continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            d=datetime.datetime.fromtimestamp(t/1000,UTC).date()
            out[d]=(float(r[1]),float(r[4]),float(r[7]))
    return out

def usd_catalog():
    p=CACHE/'catalog.json'
    if p.exists() and p.stat().st_size>100:
        x=json.load(open(p))
        if isinstance(x,list) and x and x[0].get('pegType')=='peggedUSD':return x
    all_assets=json.loads(get_bytes('https://stablecoins.llama.fi/stablecoins?includePrices=true'))['peggedAssets']
    x=[a for a in all_assets if a.get('pegType')=='peggedUSD']
    p.write_text(json.dumps(x,ensure_ascii=False))
    return x

def asset_detail(aid):
    p=CACHE/f'{aid}.json'
    if not p.exists() or p.stat().st_size<50:
        p.write_bytes(get_bytes('https://stablecoins.llama.fi/stablecoin/'+str(aid)))
    return json.load(open(p))

_HHI_ALL=None
_QUALITY={}
def build_hhi_all():
    global _HHI_ALL,_QUALITY
    if _HHI_ALL is not None:return _HHI_ALL
    sums={ch:{} for ch in CHAIN_MAP.values()}; sqs={ch:{} for ch in CHAIN_MAP.values()}; cnts={ch:{} for ch in CHAIN_MAP.values()}
    cat=usd_catalog()
    for idx,a in enumerate(cat,1):
        try:o=asset_detail(a['id'])
        except Exception:continue
        cb=o.get('chainBalances') or {}
        for ch in sums:
            rows=(((cb.get(ch) or {}).get('tokens')) or [])
            for x in rows:
                try:
                    d=datetime.datetime.fromtimestamp(int(x['date']),UTC).date()
                    if d.year<2022 or d.year>2026:continue
                    v=float(((x.get('circulating') or {}).get('peggedUSD')) or 0)
                    if v<=0:continue
                    sums[ch][d]=sums[ch].get(d,0.0)+v
                    sqs[ch][d]=sqs[ch].get(d,0.0)+v*v
                    cnts[ch][d]=cnts[ch].get(d,0)+1
                except Exception:pass
        if idx%75==0:print('ASSET_HISTORY',idx,'/',len(cat),flush=True)
    out={}
    quality={}
    for ch in sums:
        try:
            agg=json.loads(get_bytes('https://stablecoins.llama.fi/stablecoincharts/'+urllib.parse.quote(ch)))
        except Exception:
            agg=[]
        den={}
        for x in agg if isinstance(agg,list) else []:
            try:
                d=datetime.datetime.fromtimestamp(int(x['date']),UTC).date()
                v=float(((x.get('totalCirculating') or {}).get('peggedUSD')) or 0)
                if v>0:den[d]=v
            except Exception:pass
        good={}; ratios=[]; counts=[]
        for day,sm in sums[ch].items():
            if day not in den or sm<=0:continue
            ratio=sm/den[day]
            n=cnts[ch].get(day,0)
            if 0.98<=ratio<=1.02 and n>=3:
                good[day]=(sqs[ch][day]/(sm*sm),n,ratio)
            if day.year in (2023,2024):
                ratios.append(ratio);counts.append(n)
        out[ch]=good
        quality[ch]={
            'valid_days':len(good),
            'discovery_ratio_min':min(ratios) if ratios else None,
            'discovery_ratio_median':statistics.median(ratios) if ratios else None,
            'discovery_ratio_max':max(ratios) if ratios else None,
            'discovery_active_assets_median':statistics.median(counts) if counts else None,
            'discovery_active_assets_min':min(counts) if counts else None
        }
    _HHI_ALL=out;_QUALITY=quality
    (ROOT/'inputs'/'data_quality.json').write_text(json.dumps(quality,indent=2))
    (ROOT/'inputs'/'catalog_snapshot.json').write_text(json.dumps([{'id':a.get('id'),'symbol':a.get('symbol'),'name':a.get('name'),'pegType':a.get('pegType')} for a in cat],ensure_ascii=False,indent=2))
    return out

def mean(xs):return sum(xs)/len(xs) if xs else 0.0

def summarize(events):
    if not events:return {'n':0}
    syms=sorted(set(x['symbol'] for x in events));by={};pos=0
    for s in syms:
        q=[x for x in events if x['symbol']==s];m7=mean([x['r7'] for x in q])
        by[s]={'n':len(q),'mean1':mean([x['r1'] for x in q]),'mean3':mean([x['r3'] for x in q]),'mean7':m7,'win7':mean([1.0 if x['r7']>0 else 0.0 for x in q])}
        if m7>0:pos+=1
    return {'n':len(events),'symbols':len(syms),'mean1':mean([x['r1'] for x in events]),'mean3':mean([x['r3'] for x in events]),'mean7':mean([x['r7'] for x in events]),'win7':mean([1.0 if x['r7']>0 else 0.0 for x in events]),'positive_symbols7':pos,'positive_symbol_fraction7':pos/len(syms),'by_symbol':by}

universe=[]
@lru_cache(None)
def prepared(sym,a,b):
    hhi=build_hhi_all().get(CHAIN_MAP[sym],{})
    if not hhi:return None,None,[], 'no_valid_hhi_history'
    first=first_kline(sym)
    if not first:return None,None,[],'no_binance_history'
    eligible=add2(first).date();px=price_daily_window(sym,a,b)
    dates=sorted(set(px)&set(hhi)); rows=[]
    for d in dates:
        o,c,qv=px[d];hh,n,ratio=hhi[d];rows.append((d,o,c,qv,hh,n,ratio))
    delta=[None]*len(rows)
    for i in range(7,len(rows)):
        if all((rows[j][0]-rows[j-1][0]).days==1 for j in range(i-6,i+1)) and (rows[i][0]-rows[i-7][0]).days==7:
            delta[i]=rows[i][4]-rows[i-7][4]
    return first,eligible,(rows,delta),'ok'

def generate(years,a,b,collect=False):
    events=[]
    for sym,ch in CHAIN_MAP.items():
        first,eligible,p,status=prepared(sym,a,b)
        if status!='ok':
            if collect:universe.append({'symbol':sym,'chain':ch,'status':status})
            print('PERIOD',sorted(years),sym,status,flush=True);continue
        rows,delta=p;n=0
        for i in range(8,len(rows)-8):
            day=rows[i][0]
            if day.year not in years or day<eligible or delta[i] is None or delta[i-1] is None:continue
            if rows[i][3]<5_000_000:continue
            if years=={2023,2024} and rows[i+7][0]>datetime.date(2024,12,31):continue
            direction=0
            if delta[i-1]<=0 and delta[i]>0:direction=-1
            elif delta[i-1]>=0 and delta[i]<0:direction=1
            else:continue
            if any((rows[i+k][0]-rows[i+k-1][0]).days!=1 for k in range(1,8)):continue
            entry=rows[i+1][1]
            if entry<=0:continue
            events.append({'symbol':sym,'chain':ch,'signal_date':day.isoformat(),'year':day.year,'direction':'LONG' if direction>0 else 'SHORT','hhi':rows[i][4],'hhi_change7':delta[i],'active_stablecoins':rows[i][5],'coverage_ratio':rows[i][6],'quote_volume':rows[i][3],'r1':direction*math.log(rows[i+1][2]/entry),'r3':direction*math.log(rows[i+3][2]/entry),'r7':direction*math.log(rows[i+7][2]/entry)})
            n+=1
        if collect:
            universe.append({'symbol':sym,'chain':ch,'status':'ok','first_futures':first.isoformat(),'eligible_from':eligible.isoformat(),'valid_hhi_days':len(build_hhi_all().get(ch,{})),'discovery_events':n})
        print('PERIOD',sorted(years),sym,'events',n,flush=True)
    return events

discovery=generate({2023,2024},'2022-12','2025-01',True)
disc=summarize(discovery)
gate=(disc.get('mean7',0)>=0.0025 and disc.get('positive_symbol_fraction7',0)>=0.60)
summary={'discovery_2023_2024':disc,'discovery_gate_pass':gate,'oos_evaluated':False}
all_events=list(discovery)
if gate:
    o1=generate({2025},'2024-12','2026-01');o2=generate({2026},'2025-12','2026-09')
    summary['oos1_2025']=summarize(o1);summary['oos2_2026']=summarize(o2);summary['oos_evaluated']=True;all_events+=o1+o2
(ROOT/'inputs'/'universe.json').write_text(json.dumps(universe,indent=2))
(ROOT/'results'/'events.json').write_text(json.dumps(all_events,indent=2))
(ROOT/'results'/'summary.json').write_text(json.dumps(summary,indent=2))
print('SUMMARY',json.dumps(summary,indent=2),flush=True)
