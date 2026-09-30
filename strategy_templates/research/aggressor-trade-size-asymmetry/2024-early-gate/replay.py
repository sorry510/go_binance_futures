import csv,datetime,io,json,math,os,statistics,subprocess,urllib.error,urllib.request,zipfile
from collections import defaultdict
from pathlib import Path

UTC=datetime.timezone.utc
ROOT=Path(__file__).resolve().parent
INPUTS=ROOT/'inputs'; RESULTS=ROOT/'results'
INPUTS.mkdir(exist_ok=True); RESULTS.mkdir(exist_ok=True)
TMP=Path('/tmp/aggressor_trade_size_asym'); TMP.mkdir(exist_ok=True)
SYMS=['TRXUSDT','ATOMUSDT','LTCUSDT','UNIUSDT']
MONTHS=['2023-12']+[f'2024-{m:02d}' for m in range(1,13)]
KMONTHS=['2023-12']+[f'2024-{m:02d}' for m in range(1,13)]+['2025-01']
BASE='https://data.binance.vision/data/futures/um/monthly'
LOOKBACK=720
MIN_QV24=5_000_000.0

def ts_hour(ms):
    if ms>10**15: ms//=1000
    return (ms//3_600_000)*3_600_000
def download(url,path):
    if path.exists() and path.stat().st_size>0:
        return
    cmd=['curl','--http1.1','-f','-L','--retry','4','--retry-all-errors']
    cmd += ['--connect-timeout','10','--max-time','900','-sS']
    cmd += ['-o',str(path),url]
    subprocess.run(cmd,check=True)

def aggregate_month(sym,mo):
    outp=TMP/f'{sym}-agg-{mo}.json'
    if outp.exists() and outp.stat().st_size>20:
        return json.load(open(outp))
    zpath=TMP/f'{sym}-agg-{mo}.zip'
    url=f'{BASE}/aggTrades/{sym}/{sym}-aggTrades-{mo}.zip'
    print('DOWNLOAD_AGG',sym,mo,flush=True)
    download(url,zpath)
    agg=defaultdict(lambda:[0.0,0,0.0,0])
    z=zipfile.ZipFile(zpath)
    fn=z.namelist()[0]
    with io.TextIOWrapper(z.open(fn)) as fh:
        for r in csv.reader(fh):
            if not r or not r[0].isdigit():
                continue
            price=float(r[1]); qty=float(r[2])
            first_id=int(r[3]); last_id=int(r[4])
            t=int(r[5]); buyer_maker=str(r[6]).lower()=='true'
            n=max(1,last_id-first_id+1)
            h=ts_hour(t); notion=price*qty
            if buyer_maker:
                agg[h][2]+=notion; agg[h][3]+=n
            else:
                agg[h][0]+=notion; agg[h][1]+=n
    out={str(h):v for h,v in sorted(agg.items())}
    outp.write_text(json.dumps(out,separators=(',',':')))
    try:zpath.unlink()
    except FileNotFoundError:pass
    print('AGG_DONE',sym,mo,len(out),flush=True)
    return out

def load_agg(sym):
    merged={}
    for mo in MONTHS:
        rows=aggregate_month(sym,mo)
        for h,v in rows.items():
            merged[int(h)]=v
    return merged
def load_klines(sym):
    saved=INPUTS/f'kline_{sym}.json'
    if saved.exists() and saved.stat().st_size>20:
        raw=json.load(open(saved))
        return {int(k):tuple(v) for k,v in raw.items()}
    out={}
    for mo in KMONTHS:
        zpath=TMP/f'{sym}-1h-{mo}.zip'
        url=f'{BASE}/klines/{sym}/1h/{sym}-1h-{mo}.zip'
        download(url,zpath)
        z=zipfile.ZipFile(zpath); fn=z.namelist()[0]
        with io.TextIOWrapper(z.open(fn)) as fh:
            for r in csv.reader(fh):
                if not r or not r[0].isdigit():
                    continue
                t=int(r[0])
                if t>10**15:t//=1000
                h=ts_hour(t)
                out[h]=(float(r[1]),float(r[4]),float(r[7]))
        try:zpath.unlink()
        except FileNotFoundError:pass
    saved.write_text(json.dumps(out,separators=(',',':')))
    return out

def load_hourly_agg(sym):
    saved=INPUTS/f'hourly_{sym}.json'
    if saved.exists() and saved.stat().st_size>20:
        raw=json.load(open(saved))
        return {int(k):v for k,v in raw.items()}
    out=load_agg(sym)
    saved.write_text(json.dumps(out,separators=(',',':')))
    return out

def series_from_agg(agg):
    out={}
    for h,(bn,bc,sn,sc) in agg.items():
        if bc>0 and sc>0 and bn>0 and sn>0:
            out[h]=math.log((bn/bc)/(sn/sc))
    return out
HOUR=3_600_000

def build_events(sym,agg,kl):
    s=series_from_agg(agg)
    hours=sorted(s)
    events=[]
    armed=True
    for i,h in enumerate(hours):
        if i<LOOKBACK:
            continue
        if h-hours[i-LOOKBACK] != LOOKBACK*HOUR:
            continue
        hist=[s[x] for x in hours[i-LOOKBACK:i]]
        sd=statistics.pstdev(hist)
        if sd<=0:
            continue
        z=(s[h]-statistics.fmean(hist))/sd
        if not armed:
            if abs(z)<1:
                armed=True
            continue
        if abs(z)<3:
            continue
        direction=1 if z>0 else -1
        armed=False
        dt=datetime.datetime.fromtimestamp(h/1000,UTC)
        if dt.year!=2024:
            continue
        qhours=[h-k*HOUR for k in range(24)]
        if any(x not in kl for x in qhours):
            continue
        qv24=sum(kl[x][2] for x in qhours)
        if qv24<MIN_QV24:
            continue
        entry_h=h+HOUR
        close4_h=h+4*HOUR
        close12_h=h+12*HOUR
        if entry_h not in kl or close4_h not in kl or close12_h not in kl:
            continue
        entry=kl[entry_h][0]
        if entry<=0:
            continue
        events.append({
            'symbol':sym,
            'signal_hour':dt.isoformat(),
            'direction':'LONG' if direction>0 else 'SHORT',
            'z':z,'asymmetry':s[h],'qv24':qv24,
            'r1':direction*math.log(kl[entry_h][1]/entry),
            'r4':direction*math.log(kl[close4_h][1]/entry),
            'r12':direction*math.log(kl[close12_h][1]/entry)
        })
    return events
def summarize(events):
    if not events:
        return {'n':0}
    syms=sorted({e['symbol'] for e in events})
    by={}
    positive=0
    for s in syms:
        rows=[e for e in events if e['symbol']==s]
        m12=statistics.fmean(e['r12'] for e in rows)
        by[s]={
            'n':len(rows),
            'mean1':statistics.fmean(e['r1'] for e in rows),
            'mean4':statistics.fmean(e['r4'] for e in rows),
            'mean12':m12,
            'win12':sum(e['r12']>0 for e in rows)/len(rows)
        }
        if m12>0:
            positive+=1
    return {
        'n':len(events),'symbols':len(syms),
        'mean1':statistics.fmean(e['r1'] for e in events),
        'mean4':statistics.fmean(e['r4'] for e in events),
        'mean12':statistics.fmean(e['r12'] for e in events),
        'win12':sum(e['r12']>0 for e in events)/len(events),
        'positive_symbols12':positive,'by_symbol':by
    }
def main():
    all_events=[]
    for sym in SYMS:
        print('SYMBOL_START',sym,flush=True)
        agg=load_hourly_agg(sym)
        kl=load_klines(sym)
        events=build_events(sym,agg,kl)
        all_events.extend(events)
        print('SYMBOL_RESULT',sym,json.dumps(summarize(events)),flush=True)
    summary=summarize(all_events)
    summary['gate_pass']=(
        summary.get('n',0)>=20 and
        summary.get('mean12',-999)>=0.001 and
        summary.get('positive_symbols12',0)>=3
    )
    (RESULTS/'events.json').write_text(json.dumps(all_events,indent=2))
    (RESULTS/'summary.json').write_text(json.dumps(summary,indent=2))
    print('SUMMARY',json.dumps(summary,indent=2),flush=True)

if __name__=='__main__':
    main()
