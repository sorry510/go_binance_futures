import urllib.request, zipfile, io, csv, datetime, math, time
UTC=datetime.timezone.utc
events=[
("FLOWUSDT","2026-01-02 10:59"),
("ATAUSDT","2026-03-13 09:00"),
("GTCUSDT","2026-03-13 09:00"),
("NTRNUSDT","2026-03-13 09:00"),
("PHBUSDT","2026-03-13 09:00"),
("RDNTUSDT","2026-03-13 09:00"),
("NFPUSDT","2026-04-30 07:00"),
("HFTUSDT","2026-05-22 08:00"),
("STORJUSDT","2026-05-22 08:00"),
("TLMUSDT","2026-05-22 08:00"),
("VANRYUSDT","2026-07-03 06:00"),
("LSKUSDT","2026-07-24 03:00"),
("STXUSDT","2026-07-24 03:00"),
("GLMRUSDT","2026-08-11 09:00"),
("ICXUSDT","2026-08-11 09:00"),
("MOVRUSDT","2026-08-11 09:00"),
("RAREUSDT","2026-08-11 09:00"),
]
def getzip(url,tries=3):
    last=None
    for k in range(tries):
        try:
            with urllib.request.urlopen(url,timeout=25) as r:
                raw=r.read()
            return zipfile.ZipFile(io.BytesIO(raw))
        except Exception as e:
            last=e; time.sleep(.25*(k+1))
    raise last
def load_day(sym,day):
    ds=day.strftime("%Y-%m-%d")
    u=f"https://data.binance.vision/data/futures/um/daily/klines/{sym}/1m/{sym}-1m-{ds}.zip"
    z=getzip(u); fn=z.namelist()[0]
    rows=[]
    with z.open(fn) as f:
        cr=csv.reader(io.TextIOWrapper(f))
        for x in cr:
            if not x or not x[0].isdigit() or len(x)<12: continue
            ts=int(x[0]); 
            if ts>10**15: ts//=1000
            rows.append((ts,float(x[1]),float(x[4])))
    return rows
def load_funding(sym,dt):
    m=dt.strftime("%Y-%m")
    u=f"https://data.binance.vision/data/futures/um/monthly/fundingRate/{sym}/{sym}-fundingRate-{m}.zip"
    try:z=getzip(u)
    except Exception:return []
    fn=z.namelist()[0]; out=[]
    with z.open(fn) as f:
        cr=csv.reader(io.TextIOWrapper(f))
        hdr=next(cr,None)
        # expected calc_time,funding_interval_hours,last_funding_rate,mark_price
        for x in cr:
            if not x or not x[0].isdigit(): continue
            ts=int(x[0]); 
            if ts>10**15: ts//=1000
            try:
                rate=float(x[2]); mark=float(x[3])
            except Exception:
                continue
            out.append((ts,rate,mark))
    return out
def slip(price,side,enter):
    rate=.0005
    if (side=="LONG" and enter) or (side=="SHORT" and not enter): return price*(1+rate)
    return price*(1-rate)
def gross_roi_short(entry,price):
    # mirror engine: gross / (qty*price) * leverage * 100
    return (entry-price)/price*4*100
def one(sym,ts):
    ev=datetime.datetime.strptime(ts,"%Y-%m-%d %H:%M").replace(tzinfo=UTC)
    bars=[]
    for k in range(4):
        day=(ev+datetime.timedelta(days=k)).date()
        try: bars.extend(load_day(sym,datetime.datetime.combine(day,datetime.time(),tzinfo=UTC)))
        except Exception as e:
            print("DAY_ERR",sym,day,e)
    bars.sort()
    if not bars: return None
    evms=int(ev.timestamp()*1000)
    # next 1m open after announcement timestamp
    idx=next((i for i,b in enumerate(bars) if b[0]>=evms+60000),None)
    if idx is None:return None
    ent_ts,ent_open,_=bars[idx]; entry=slip(ent_open,"SHORT",True)
    qty=4.0/entry
    open_fee=qty*entry*0.0005
    funding=0.0
    funds=load_funding(sym,ev)
    fi=0
    while fi<len(funds) and funds[fi][0]<ent_ts: fi+=1
    trigger=None; exit_idx=None
    horizon=evms+72*3600*1000
    i=idx
    while i<len(bars):
        tsb,op,cl=bars[i]
        while fi<len(funds) and funds[fi][0]<=tsb+59999:
            ft,rate,mark=funds[fi]
            if ft>=ent_ts and ft<=horizon:
                funding += qty*mark*rate # short receives positive funding
            fi+=1
        if tsb>=horizon:
            trigger="TIME"; exit_idx=i; break
        roi=gross_roi_short(entry,cl)
        if roi<=-6:
            trigger="SL"; exit_idx=min(i+1,len(bars)-1); break
        if roi>=8:
            trigger="TP"; exit_idx=min(i+1,len(bars)-1); break
        i+=1
    if exit_idx is None:
        exit_idx=len(bars)-1; trigger="EOF"
    ex_ts,ex_open,ex_close=bars[exit_idx]
    raw_exit=ex_open if trigger in ("TP","SL","TIME") else ex_close
    exitp=slip(raw_exit,"SHORT",False)
    gross=(entry-exitp)*qty
    close_fee=qty*exitp*0.0005
    net=gross-open_fee-close_fee+funding
    return dict(sym=sym,event=ts,entry_ts=ent_ts,exit_ts=ex_ts,trigger=trigger,net_pct=net*100,gross_pct=gross*100,funding_pct=funding*100)
res=[]
for sym,ts in events:
    try:r=one(sym,ts)
    except Exception as e:
        print("ERR",sym,e); r=None
    if r:
        res.append(r);print("{sym:9s} {trigger:4s} net={net_pct:+7.3f}% gross={gross_pct:+7.3f}% fund={funding_pct:+6.3f}%".format(**r))
gp=sum(max(r["net_pct"],0) for r in res); gl=-sum(min(r["net_pct"],0) for r in res)
print("SUMMARY n",len(res),"win",sum(r["net_pct"]>0 for r in res),"PF",gp/gl if gl else 999,"net",sum(r["net_pct"] for r in res),"avg",sum(r["net_pct"] for r in res)/len(res) if res else 0)
print("TRIGGERS",{k:sum(r["trigger"]==k for r in res) for k in ["TP","SL","TIME","EOF"]})
