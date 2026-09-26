from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, as_completed
from functools import lru_cache
import urllib.request, urllib.error, zipfile, io, csv, datetime, time, json, re, statistics, threading

UTC=datetime.timezone.utc
CMS_LIST='https://www.binance.com/bapi/composite/v1/public/cms/article/list/query'
CMS_DETAIL='https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query'
VISION='https://data.binance.vision/data/futures/um/monthly'
UA='Mozilla/5.0'
LOCK=threading.Lock()

def get_bytes(url, tries=5, timeout=30):
    last=None
    for k in range(tries):
        try:
            req=urllib.request.Request(url, headers={'User-Agent':UA})
            with urllib.request.urlopen(req, timeout=timeout) as r:
                return r.read()
        except urllib.error.HTTPError as e:
            if e.code == 404:
                return None
            last=e
        except Exception as e:
            last=e
        time.sleep(.35*(k+1))
    raise last

def get_json(url):
    raw=get_bytes(url)
    if raw is None: return None
    return json.loads(raw)

def flatten(node):
    if isinstance(node, dict):
        if node.get('node')=='text':
            return node.get('text','')
        s=''.join(flatten(c) for c in node.get('child',[]) or [])
        # Preserve boundaries between structural cells/rows so adjacent table
        # values cannot become fake pairs such as PENDLE/USDCALT/USDC.
        if node.get('tag') in {'p','div','td','th','tr','table','li','ul','ol','br'}:
            return ' ' + s + ' '
        return s
    if isinstance(node, list):
        return ' '.join(flatten(x) for x in node)
    return ''

def paragraphs(body):
    root=json.loads(body) if isinstance(body,str) else body
    out=[]
    for c in (root or {}).get('child',[]) or []:
        if isinstance(c,dict) and c.get('tag')=='p':
            text=flatten(c).replace('\xa0',' ').strip()
            if text: out.append(text)
    return out

def fetch_2024_articles():
    found=[]
    seen_2024=False
    for page in range(1,120):
        url=f"{CMS_LIST}?type=1&pageNo={page}&pageSize=50&catalogId=48"
        obj=get_json(url)
        arts=((obj or {}).get('data') or {}).get('catalogs',[{}])[0].get('articles',[])
        if not arts: break
        dates=[]
        for a in arts:
            dt=datetime.datetime.fromtimestamp(a['releaseDate']/1000,tz=UTC)
            dates.append(dt)
            if dt.year==2024:
                seen_2024=True
                title=a.get('title','')
                if title.startswith('Binance Margin Adds'):
                    found.append(a)
        print('LIST_PAGE',page,'range',min(dates).date(),max(dates).date(),'target',len(found),flush=True)
        if seen_2024 and min(dates).year < 2024:
            break
    return found

def parse_article(a):
    code=a['code']
    obj=get_json(f"{CMS_DETAIL}?articleCode={code}")
    d=(obj or {}).get('data') or {}
    pub=datetime.datetime.fromtimestamp(d['publishDate']/1000,tz=UTC)
    root=json.loads(d.get('body','{}')) if isinstance(d.get('body'),str) else (d.get('body') or {})
    body_text=flatten(root).replace('\xa0',' ')
    core=body_text.split('Explore Binance Margin',1)[0]
    pairs=re.findall(r'\b([A-Z0-9]{2,24})/([A-Z0-9]{2,24})\b', core)
    bases={}
    for base,quote in pairs:
        bases.setdefault(base,[]).append(base+'/'+quote)
    # Include explicit new borrowable assets when the intro names tickers in parentheses.
    prefix=core.split('as new borrowable asset',1)[0] if 'as new borrowable asset' in core else ''
    for ticker in re.findall(r'\(([A-Z0-9]{2,20})\)', prefix):
        bases.setdefault(ticker,[]).append('borrowable_asset')
    if not bases:
        return {'error':'no_margin_assets','code':code,'title':d.get('title',a.get('title','')),'publish':pub.isoformat()}
    bases={k:sorted(set(v)) for k,v in bases.items()}
    return {
        'code':code,'title':d.get('title',a.get('title','')),
        'publish_ms':int(d['publishDate']),'publish':pub.strftime('%Y-%m-%d %H:%M:%S'),
        'opening':core[:3000],'bases':bases,
    }

def sub_years(dt,years=2):
    try: return dt.replace(year=dt.year-years)
    except ValueError: return dt.replace(year=dt.year-years,day=28)

def month_key(dt): return dt.strftime('%Y-%m')
def next_month(dt):
    if dt.month==12: return dt.replace(year=dt.year+1,month=1,day=1)
    return dt.replace(month=dt.month+1,day=1)

def zip_rows(url):
    raw=get_bytes(url)
    if raw is None: return None
    z=zipfile.ZipFile(io.BytesIO(raw))
    fn=z.namelist()[0]
    return z,fn

@lru_cache(maxsize=None)
def load_month(sym, month):
    url=f'{VISION}/klines/{sym}/1m/{sym}-1m-{month}.zip'
    got=zip_rows(url)
    if got is None: return []
    z,fn=got; out=[]
    with z.open(fn) as f:
        cr=csv.reader(io.TextIOWrapper(f))
        for x in cr:
            if not x or not x[0].isdigit() or len(x)<5: continue
            ts=int(x[0])
            if ts>10**15: ts//=1000
            out.append((ts,float(x[1]),float(x[4])))
    return out

@lru_cache(maxsize=None)
def load_funding_month(sym, month):
    url=f'{VISION}/fundingRate/{sym}/{sym}-fundingRate-{month}.zip'
    got=zip_rows(url)
    if got is None: return []
    z,fn=got; out=[]
    with z.open(fn) as f:
        cr=csv.reader(io.TextIOWrapper(f))
        hdr=next(cr,None)
        names=[str(v).strip().lower() for v in (hdr or [])]
        def idx(*cands):
            for c in cands:
                if c in names: return names.index(c)
            return None
        ti=idx('calc_time','funding_time')
        ri=idx('last_funding_rate','funding_rate')
        mi=idx('mark_price')
        for x in cr:
            try:
                tcol=ti if ti is not None else 0
                rcol=ri if ri is not None else 2
                if not x or tcol>=len(x) or not x[tcol].isdigit() or rcol>=len(x): continue
                ts=int(x[tcol])
                if ts>10**15: ts//=1000
                rate=float(x[rcol])
                mark=float(x[mi]) if mi is not None and mi<len(x) and x[mi] else 0.0
            except Exception:
                continue
            out.append((ts,rate,mark))
    return out

def eligibility(ev):
    pub=datetime.datetime.fromtimestamp(ev['publish_ms']/1000,tz=UTC)
    cutoff=sub_years(pub,2)
    sym=ev['base']+'USDT'
    hist=load_month(sym,month_key(cutoff))
    if not hist:
        return ev|{'eligible':False,'reason':'no_24m_month','symbol':sym}
    cutoff_ms=int(cutoff.timestamp()*1000)
    if hist[0][0] > cutoff_ms:
        return ev|{'eligible':False,'reason':'history_lt_2y','symbol':sym,'first_hist_ts':hist[0][0]}
    cur=load_month(sym,month_key(pub))
    if not cur:
        return ev|{'eligible':False,'reason':'no_event_month','symbol':sym}
    evms=ev['publish_ms']
    # contract must be trading at announcement and have a prompt next-minute bar
    before=any(b[0] <= evms for b in cur)
    after=next((b for b in cur if b[0] >= ((evms//60000)+1)*60000),None)
    if not before or after is None or after[0] > evms+5*60000:
        return ev|{'eligible':False,'reason':'not_trading_at_event','symbol':sym,'next_ts':after[0] if after else None}
    return ev|{'eligible':True,'reason':'ok','symbol':sym}

def slip(price,side,entering):
    rate=5/10000
    if (side=='LONG' and entering) or (side=='SHORT' and not entering): return price*(1+rate)
    return price*(1-rate)

def roi_long(entry, price):
    # Matches utils.FuturesLeveragedROI: unrealized/(qty*mark)*leverage*100.
    return (price-entry)/price*4*100

def replay(ev):
    pub=datetime.datetime.fromtimestamp(ev['publish_ms']/1000,tz=UTC)
    evms=ev['publish_ms']; horizon=evms+72*3600*1000
    sym=ev['symbol']
    end_dt=datetime.datetime.fromtimestamp((horizon+120000)/1000,tz=UTC)
    months=[]; cur=pub.replace(day=1,hour=0,minute=0,second=0,microsecond=0)
    endm=end_dt.replace(day=1,hour=0,minute=0,second=0,microsecond=0)
    while cur<=endm:
        months.append(month_key(cur)); cur=next_month(cur)
    bars=[]
    for m in months: bars.extend(load_month(sym,m))
    bars.sort(key=lambda x:x[0])
    target=((evms//60000)+1)*60000
    idx=next((i for i,b in enumerate(bars) if b[0]>=target),None)
    if idx is None: return ev|{'error':'no_entry_bar'}
    ent_ts,ent_open,_=bars[idx]
    if ent_ts>evms+5*60000: return ev|{'error':'entry_gap','entry_ts':ent_ts}
    entry=slip(ent_open,'LONG',True)
    qty=4.0/entry
    open_fee=qty*entry*0.0005
    funds=[]
    for m in months: funds.extend(load_funding_month(sym,m))
    funds.sort(key=lambda x:x[0])
    fi=0
    while fi<len(funds) and funds[fi][0]<ent_ts: fi+=1
    funding=0.0; trigger=None; exit_idx=None; last_valid=idx
    i=idx
    while i<len(bars):
        tsb,op,cl=bars[i]
        if tsb>horizon+60000: break
        last_valid=i
        while fi<len(funds) and funds[fi][0]<=tsb+59999:
            ft,rate,mark=funds[fi]
            if ft>=ent_ts and ft<=horizon:
                effective=mark if mark>0 else cl
                funding -= qty*effective*rate  # LONG pays positive funding
            fi+=1
        if tsb>=horizon:
            trigger='TIME'; exit_idx=i; break
        roi=roi_long(entry,cl)
        if roi<=-6:
            trigger='SL'; exit_idx=i+1 if i+1<len(bars) else i; break
        if roi>=8:
            trigger='TP'; exit_idx=i+1 if i+1<len(bars) else i; break
        i+=1
    if exit_idx is None:
        trigger='EOF'; exit_idx=last_valid
    ex_ts,ex_open,ex_close=bars[exit_idx]
    raw_exit=ex_open if trigger in ('TP','SL','TIME') else ex_close
    exitp=slip(raw_exit,'LONG',False)
    gross=(exitp-entry)*qty
    close_fee=qty*exitp*0.0005
    net=gross-open_fee-close_fee+funding
    return ev|{
        'entry_ts':ent_ts,'exit_ts':ex_ts,'trigger':trigger,
        'entry':entry,'exit':exitp,'gross_pct':gross*100,
        'fees_pct':(open_fee+close_fee)*100,'funding_pct':funding*100,
        'net_pct':net*100,'hold_h':(ex_ts-ent_ts)/3600000,
    }

arts=fetch_2024_articles()
Path('/tmp/margin_add_2024_articles.json').write_text(json.dumps(arts,ensure_ascii=False,indent=2))
print('ARTICLES',len(arts),flush=True)

parsed=[]
with ThreadPoolExecutor(max_workers=8) as ex:
    futs=[ex.submit(parse_article,a) for a in arts]
    for n,f in enumerate(as_completed(futs),1):
        r=f.result(); parsed.append(r)
        if n%10==0 or 'error' in r: print('DETAIL',n,'/',len(futs),r.get('error',''),flush=True)
parsed.sort(key=lambda r:r.get('publish_ms',0))
Path('/tmp/margin_add_2024_details.json').write_text(json.dumps(parsed,ensure_ascii=False,indent=2))
parse_err=[x for x in parsed if 'error' in x]
if parse_err: print('PARSE_ERRORS',json.dumps(parse_err,ensure_ascii=False),flush=True)

events=[]
for a in parsed:
    if 'error' in a: continue
    for base,pairs in a['bases'].items():
        events.append({
            'publish_ms':a['publish_ms'],'publish':a['publish'],'base':base,'pairs':','.join(pairs),
            'code':a['code'],'title':a['title'],
        })
events.sort(key=lambda x:(x['publish_ms'],x['base']))
Path('/tmp/margin_add_2024_events.json').write_text(json.dumps(events,ensure_ascii=False,indent=2))
print('TOKEN_EVENTS',len(events),'UNIQUE_BASES',len({x['base'] for x in events}),flush=True)

elig=[]
with ThreadPoolExecutor(max_workers=8) as ex:
    futs={ex.submit(eligibility,e):e for e in events}
    for n,f in enumerate(as_completed(futs),1):
        try:r=f.result()
        except Exception as e:r=futs[f]|{'eligible':False,'reason':'error:'+repr(e),'symbol':futs[f]['base']+'USDT'}
        elig.append(r)
        if n%20==0: print('ELIG',n,'/',len(futs),flush=True)

# Network errors are not eligibility failures. Retry only transient failures
# serially so TLS/rate pressure cannot silently shrink the discovery sample.
for i,r in enumerate(elig):
    if not str(r.get('reason','')).startswith('error:'):
        continue
    last=r
    for retry in range(3):
        try:
            last=eligibility(r)
            break
        except Exception as e:
            last=r|{'eligible':False,'reason':'error:'+repr(e),'symbol':r['base']+'USDT'}
            time.sleep(1.0*(retry+1))
    elig[i]=last
elig.sort(key=lambda x:(x['publish_ms'],x['base']))
Path('/tmp/margin_add_2024_eligibility.json').write_text(json.dumps(elig,ensure_ascii=False,indent=2))
eligible=[x for x in elig if x.get('eligible')]
reasons={}
for x in elig:
    if not x.get('eligible'): reasons[x['reason']]=reasons.get(x['reason'],0)+1
print('ELIGIBLE',len(eligible),'UNIQUE',len({x['base'] for x in eligible}),'INELIGIBLE_REASONS',json.dumps(reasons,sort_keys=True),flush=True)

# Preserve a compact eligibility input for reproducibility.
with Path('/tmp/margin_add_eligible_2024.tsv').open('w') as f:
    f.write('publish\tbase\tsymbol\tpairs\tcode\n')
    for x in eligible:
        f.write(f"{x['publish']}\t{x['base']}\t{x['symbol']}\t{x['pairs']}\t{x['code']}\n")

res=[]
with ThreadPoolExecutor(max_workers=8) as ex:
    futs={ex.submit(replay,e):e for e in eligible}
    for n,f in enumerate(as_completed(futs),1):
        try:r=f.result()
        except Exception as e:r=futs[f]|{'error':repr(e)}
        res.append(r)
        if n%20==0 or 'error' in r: print('REPLAY',n,'/',len(futs),r.get('symbol'),r.get('trigger'),r.get('error',''),flush=True)
res.sort(key=lambda x:(x['publish_ms'],x['base']))

# Enforce one active position per symbol: later overlapping event is skipped.
accepted=[]; skipped=[]; last_exit={}
for r in res:
    if 'error' in r:
        skipped.append(r|{'skip_reason':'replay_error'})
        continue
    prev=last_exit.get(r['symbol'])
    if prev is not None and r['entry_ts'] <= prev:
        skipped.append(r|{'skip_reason':'single_position_overlap'})
        continue
    accepted.append(r)
    last_exit[r['symbol']]=r['exit_ts']

Path('/tmp/margin_add_long_2024_results.json').write_text(json.dumps(accepted,ensure_ascii=False,indent=2))
Path('/tmp/margin_add_long_2024_skipped.json').write_text(json.dumps(skipped,ensure_ascii=False,indent=2))
cols=['publish','symbol','pairs','entry_ts','exit_ts','trigger','entry','exit','gross_pct','fees_pct','funding_pct','net_pct','hold_h']
with Path('/tmp/margin_add_long_2024_results.tsv').open('w') as f:
    f.write('\t'.join(cols)+'\n')
    for r in accepted:f.write('\t'.join(str(r[c]) for c in cols)+'\n')

ok=accepted
pos=sum(r['net_pct'] for r in ok if r['net_pct']>0)
neg=-sum(r['net_pct'] for r in ok if r['net_pct']<0)
months={}
for r in ok: months.setdefault(r['publish'][:7],[]).append(r['net_pct'])
by={}
for r in ok: by.setdefault(r['symbol'],[]).append(r['net_pct'])
summary={
    'articles':len(arts),'token_events':len(events),'unique_bases':len({x['base'] for x in events}),
    'eligible_events':len(eligible),'eligible_unique_bases':len({x['base'] for x in eligible}),
    'trades':len(ok),'skipped':len(skipped),
    'wins':sum(r['net_pct']>0 for r in ok),'losses':sum(r['net_pct']<=0 for r in ok),
    'pf':pos/neg if neg else None,'net_pct_sum':sum(r['net_pct'] for r in ok),
    'avg_net_pct':statistics.mean(r['net_pct'] for r in ok) if ok else None,
    'median_net_pct':statistics.median(r['net_pct'] for r in ok) if ok else None,
    'min_net_pct':min((r['net_pct'] for r in ok),default=None),
    'max_net_pct':max((r['net_pct'] for r in ok),default=None),
    'gross_profit':pos,'gross_loss':neg,
    'funding_pct_sum':sum(r['funding_pct'] for r in ok),
    'fees_pct_sum':sum(r['fees_pct'] for r in ok),
    'triggers':{k:sum(r['trigger']==k for r in ok) for k in ['TP','SL','TIME','EOF']},
    'months':{m:{'n':len(v),'net':sum(v),'wins':sum(x>0 for x in v)} for m,v in sorted(months.items())},
    'repeated_symbols':{s:{'n':len(v),'net':sum(v)} for s,v in sorted(by.items()) if len(v)>1},
    'ineligible_reasons':reasons,
    'single_position_overlap_skips':sum(x.get('skip_reason')=='single_position_overlap' for x in skipped),
}
Path('/tmp/margin_add_long_2024_summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print('SUMMARY',json.dumps(summary,ensure_ascii=False),flush=True)
