import csv, datetime, io, json, time, urllib.request, urllib.error, zipfile
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed
from functools import lru_cache
from pathlib import Path

UTC=datetime.timezone.utc
UA="Mozilla/5.0"
BASE="https://data.binance.vision/data/futures/um/monthly"
ROOT=Path("strategy_templates/research/binance-portfolio-margin-collateral-ratio/2024-corrected-exact-discovery")
EVENTS=json.loads((ROOT/"inputs/eligible.json").read_text())
CACHE=Path("/tmp/collateral_ratio_corrected_exact"); CACHE.mkdir(exist_ok=True)

LEV=4.0
TP=8.0
SL=-6.0
FEE=0.0005
SLIP=0.0005
MAX_HOLD_MS=72*3600*1000

def month(dt): return dt.strftime("%Y-%m")
def next_month(dt):
    return (dt.replace(day=28)+datetime.timedelta(days=4)).replace(day=1,hour=0,minute=0,second=0,microsecond=0)

def get_bytes(url,key):
    p=CACHE/key
    miss=CACHE/(key+".missing")
    if miss.exists(): return None
    if p.exists(): return p.read_bytes()
    last=None
    for k in range(5):
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA})
            b=urllib.request.urlopen(req,timeout=35).read()
            zipfile.ZipFile(io.BytesIO(b))
            p.write_bytes(b)
            return b
        except urllib.error.HTTPError as e:
            if e.code==404:
                miss.write_text("404"); return None
            last=e
        except Exception as e:
            last=e
        time.sleep(.6*(k+1))
    raise last
def required_months(e):
    start=datetime.datetime.fromtimestamp(e["effective_ms"]/1000,UTC)
    # Entry is next minute; 72h horizon may cross at most one month boundary here.
    end=start+datetime.timedelta(hours=73)
    cur=start.replace(day=1,hour=0,minute=0,second=0,microsecond=0)
    last=end.replace(day=1,hour=0,minute=0,second=0,microsecond=0)
    out=[]
    while cur<=last:
        out.append(month(cur)); cur=next_month(cur)
    return out

def prefetch(sym,mo):
    get_bytes(f"{BASE}/klines/{sym}/1m/{sym}-1m-{mo}.zip",f"k-{sym}-{mo}.zip")
    get_bytes(f"{BASE}/fundingRate/{sym}/{sym}-fundingRate-{mo}.zip",f"f-{sym}-{mo}.zip")

jobs=sorted({(e["symbol"],m) for e in EVENTS for m in required_months(e)})
print("FILES",len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=8) as ex:
    fs=[ex.submit(prefetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%5==0 or i==len(fs): print("DOWNLOAD",i,"/",len(fs),flush=True)

@lru_cache(None)
def klines(sym,mo):
    b=get_bytes(f"{BASE}/klines/{sym}/1m/{sym}-1m-{mo}.zip",f"k-{sym}-{mo}.zip")
    if not b: return []
    z=zipfile.ZipFile(io.BytesIO(b)); out=[]
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit() or len(r)<5: continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out.append((t,float(r[1]),float(r[4])))
    return out

@lru_cache(None)
def funding(sym,mo):
    b=get_bytes(f"{BASE}/fundingRate/{sym}/{sym}-fundingRate-{mo}.zip",f"f-{sym}-{mo}.zip")
    if not b: return []
    z=zipfile.ZipFile(io.BytesIO(b))
    with z.open(z.namelist()[0]) as f:
        cr=csv.reader(io.TextIOWrapper(f))
        hdr=next(cr,None) or []
        names=[str(x).strip().lower() for x in hdr]
        ti=names.index("calc_time") if "calc_time" in names else 0
        ri=names.index("last_funding_rate") if "last_funding_rate" in names else 2
        mi=names.index("mark_price") if "mark_price" in names else None
        out=[]
        for r in cr:
            try:
                t=int(r[ti]); t=t//1000 if t>10**15 else t
                rate=float(r[ri])
                mark=float(r[mi]) if mi is not None and mi<len(r) and r[mi] else 0.0
                out.append((t,rate,mark))
            except Exception:
                pass
    return out
def slip(price,side,enter):
    adverse=((side=="LONG" and enter) or (side=="SHORT" and not enter))
    return price*(1+SLIP if adverse else 1-SLIP)

def gross_roi(entry,price,side):
    # Mirrors the audited project event-replay / Engine ROI denominator.
    if side=="LONG":
        return (price-entry)/price*LEV*100
    return (entry-price)/price*LEV*100

def replay(e):
    sym=e["symbol"]; side=e["direction"]; ev=int(e["effective_ms"])
    months=required_months(e)
    bs=[]
    for mo in months: bs.extend(klines(sym,mo))
    bs.sort()
    target=(ev//60000+1)*60000
    idx=next((i for i,b in enumerate(bs) if b[0]>=target),None)
    if idx is None: return {**e,"error":"no_entry"}
    if bs[idx][0]>target+5*60000:
        return {**e,"error":"entry_gap","target":target,"actual":bs[idx][0]}

    ent_t,ent_o,_=bs[idx]
    entry=slip(ent_o,side,True)
    qty=LEV/entry
    open_fee=qty*entry*FEE
    horizon=ent_t+MAX_HOLD_MS

    fs=[]
    for mo in months: fs.extend(funding(sym,mo))
    fs.sort()
    fi=0
    while fi<len(fs) and fs[fi][0]<ent_t: fi+=1

    funding_pnl=0.0
    trigger=None
    trigger_t=None
    exit_idx=None
    last_i=idx

    for i in range(idx,len(bs)):
        t,op,cl=bs[i]
        if t>horizon: break
        last_i=i

        # Position is held through this minute close. Funding records no later
        # than that close are included, matching the audited event replay.
        while fi<len(fs) and fs[fi][0]<=t+59999:
            ft,rate,mark=fs[fi]
            if ent_t<=ft<=horizon:
                px=mark if mark>0 else cl
                funding_pnl += (-1 if side=="LONG" else 1)*qty*px*rate
            fi+=1

        rr=gross_roi(entry,cl,side)
        if rr<=SL:
            trigger="SL"; trigger_t=t
            exit_idx=i+1 if i+1<len(bs) else i
            break
        if rr>=TP:
            trigger="TP"; trigger_t=t
            exit_idx=i+1 if i+1<len(bs) else i
            break
        if t>=horizon:
            trigger="TIME"; trigger_t=t
            exit_idx=i+1 if i+1<len(bs) else i
            break

    if exit_idx is None:
        trigger="EOF"; trigger_t=bs[last_i][0]; exit_idx=last_i

    xt,xo,xc=bs[exit_idx]
    raw_exit=xo if trigger in ("TP","SL","TIME") else xc
    exitp=slip(raw_exit,side,False)
    gross=((exitp-entry) if side=="LONG" else (entry-exitp))*qty
    close_fee=qty*exitp*FEE
    net=gross-open_fee-close_fee+funding_pnl

    return {**e,
        "entry_time":ent_t,"trigger_time":trigger_t,"exit_time":xt,
        "entry_price":entry,"exit_price":exitp,"quantity":qty,
        "trigger":trigger,"gross_pct":gross*100,
        "fees_pct":(open_fee+close_fee)*100,
        "funding_pct":funding_pnl*100,
        "net_pct":net*100,
        "hold_hours":(xt-ent_t)/3600000,
    }
def summarize(rows):
    good=[x for x in rows if not x.get("error")]
    gp=sum(max(x["net_pct"],0) for x in good)
    gl=-sum(min(x["net_pct"],0) for x in good)
    return {
        "n":len(good),
        "wins":sum(x["net_pct"]>0 for x in good),
        "losses":sum(x["net_pct"]<=0 for x in good),
        "tp":sum(x.get("trigger")=="TP" for x in good),
        "sl":sum(x.get("trigger")=="SL" for x in good),
        "time":sum(x.get("trigger")=="TIME" for x in good),
        "eof":sum(x.get("trigger")=="EOF" for x in good),
        "pf":gp/gl if gl>0 else None,
        "net_pct":sum(x["net_pct"] for x in good),
        "avg_pct":sum(x["net_pct"] for x in good)/len(good) if good else None,
        "gp":gp,"gl":gl,
    }

res=[]
for i,e in enumerate(EVENTS,1):
    x=replay(e); res.append(x)
    print("TRADE",i,"/",len(EVENTS),x["symbol"],x["direction"],
          x.get("trigger",x.get("error")),round(x.get("net_pct",0),4),flush=True)

good=[x for x in res if not x.get("error")]
by_article=defaultdict(list)
by_side=defaultdict(list)
by_year=defaultdict(list)
for x in good:
    by_article[x["article_code"]].append(x)
    by_side[x["direction"]].append(x)
    y=datetime.datetime.fromtimestamp(x["effective_ms"]/1000,UTC).year
    by_year[str(y)].append(x)

batch_means={k:sum(x["net_pct"] for x in q)/len(q) for k,q in by_article.items()}
positive_batches=sum(v>0 for v in batch_means.values())
all_summary=summarize(good)
gate=(
    all_summary["n"]==len(EVENTS)
    and all_summary["pf"] is not None and all_summary["pf"]>=1.15
    and all_summary["net_pct"]>0
    and positive_batches/max(1,len(batch_means))>=0.60
)

summary={
    "status":"untouched_exact_pass" if gate else "frozen_failed_untouched_exact",
    "gate_pass":gate,
    "all":all_summary,
    "by_side":{k:summarize(v) for k,v in sorted(by_side.items())},
    "by_year":{k:summarize(v) for k,v in sorted(by_year.items())},
    "by_article":{k:summarize(v)|{"mean_net_pct":batch_means[k]} for k,v in sorted(by_article.items())},
    "positive_batches":f"{positive_batches}/{len(batch_means)}",
    "positive_batch_fraction":positive_batches/max(1,len(batch_means)),
    "errors":[x for x in res if x.get("error")],
    "prior_event_returns_seen":False,
    "returns_were_unread_before_this_run":True,
}

(ROOT/"results/exact_trades.json").write_text(json.dumps(res,ensure_ascii=False,indent=2))
(ROOT/"results/exact_summary.json").write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print("SUMMARY",json.dumps(summary,indent=2),flush=True)
