import csv, io, zipfile, urllib.request, urllib.parse, urllib.error
import datetime, json, math, time, subprocess
import xml.etree.ElementTree as ET
from pathlib import Path
from functools import lru_cache

UTC = datetime.timezone.utc
UA = "Mozilla/5.0"
ROOT = Path(__file__).resolve().parent
CACHE = Path("/tmp/chain_fee_tvl_yield_cache")
CACHE.mkdir(exist_ok=True)
DV = "https://data.binance.vision/data/futures/um/monthly/klines"
S3 = "https://s3-ap-northeast-1.amazonaws.com/data.binance.vision"

CHAIN_MAP = {
    "BTCUSDT":"Bitcoin", "ETHUSDT":"Ethereum", "BNBUSDT":"BSC",
    "SOLUSDT":"Solana", "AVAXUSDT":"Avalanche", "ADAUSDT":"Cardano",
    "NEARUSDT":"Near", "TRXUSDT":"Tron", "MATICUSDT":"Polygon",
    "FTMUSDT":"Fantom", "OPUSDT":"OP Mainnet", "APTUSDT":"Aptos",
    "ARBUSDT":"Arbitrum", "SUIUSDT":"Sui", "XRPUSDT":"XRPL"
}
TVL_MAP = dict(CHAIN_MAP)
TVL_MAP["OPUSDT"] = "Optimism"
TVL_MAP["XRPUSDT"] = "Ripple"

def add2(d):
    try:
        return d.replace(year=d.year+2)
    except ValueError:
        return d.replace(year=d.year+2, day=28)
FIRST_FUTURES = {
    "BTCUSDT":"2020-01-01T00:00:00+00:00","ETHUSDT":"2020-01-01T00:00:00+00:00",
    "BNBUSDT":"2020-02-10T00:00:00+00:00","SOLUSDT":"2020-09-14T00:00:00+00:00",
    "AVAXUSDT":"2020-09-23T00:00:00+00:00","ADAUSDT":"2020-01-31T00:00:00+00:00",
    "NEARUSDT":"2020-10-15T00:00:00+00:00","TRXUSDT":"2020-01-15T00:00:00+00:00",
    "MATICUSDT":"2020-10-22T00:00:00+00:00","FTMUSDT":"2020-09-24T00:00:00+00:00",
    "OPUSDT":"2022-06-01T00:00:00+00:00","APTUSDT":"2022-10-19T00:00:00+00:00",
    "ARBUSDT":"2023-03-23T00:00:00+00:00","SUIUSDT":"2023-05-03T00:00:00+00:00",
    "XRPUSDT":"2020-01-06T00:00:00+00:00"
}
@lru_cache(None)
def first_kline(sym):
    v=FIRST_FUTURES.get(sym)
    return datetime.datetime.fromisoformat(v) if v else None
def month_span(start_ym, end_ym):
    out=[]
    d=datetime.date.fromisoformat(start_ym+"-01")
    end=datetime.date.fromisoformat(end_ym+"-01")
    while d<=end:
        out.append(d.strftime("%Y-%m"))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out

def fetch_k(sym, mo):
    p=CACHE/f"{sym}-{mo}.zip"
    if p.exists():
        return p
    for d in ("/tmp/chain_stablecoin_growth_cache","/tmp/chain_dex_residual_cache"):
        shared=Path(d)/f"{sym}-{mo}.zip"
        if shared.exists():
            return shared
    url=f"{DV}/{sym}/1d/{sym}-1d-{mo}.zip"
    tmp=str(p)+".part"
    q=subprocess.run(["curl","--http1.1","-L","--retry","4","--retry-all-errors",
                      "--connect-timeout","8","--max-time","40","-sS","-w","%{http_code}",
                      "-o",tmp,url],capture_output=True,text=True)
    code=(q.stdout or "")[-3:]
    if q.returncode==0 and code=="200":
        Path(tmp).replace(p)
    elif code=="404":
        Path(tmp).unlink(missing_ok=True); p.write_bytes(b"")
    else:
        Path(tmp).unlink(missing_ok=True)
        raise RuntimeError(f"kline download failed {sym} {mo} rc={q.returncode} http={code}")
    return p

@lru_cache(None)
def price_daily_window(sym, start_ym, end_ym):
    out={}
    for mo in month_span(start_ym, end_ym):
        p=fetch_k(sym,mo)
        if not p.exists() or p.stat().st_size==0:
            continue
        try:
            z=zipfile.ZipFile(p)
        except Exception:
            continue
        for r in csv.reader(io.TextIOWrapper(z.open(z.namelist()[0]))):
            if not r or not r[0].isdigit():
                continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            d=datetime.datetime.fromtimestamp(t/1000,UTC).date()
            out[d]=(float(r[1]),float(r[4]),float(r[7]))
    return out
@lru_cache(None)
def fee_daily(chain):
    p=CACHE/("fee-"+chain.replace(" ","_")+".json")
    if not p.exists() or p.stat().st_size<100:
        url="https://api.llama.fi/overview/fees/"+urllib.parse.quote(chain)+"?excludeTotalDataChartBreakdown=true&dataType=dailyFees"
        q=subprocess.run(["curl","--http1.1","-L","--retry","2",
                          "--retry-all-errors","--connect-timeout","8",
                          "--max-time","30","-sS","-o",str(p),url])
        if q.returncode:
            return {}
    try:
        o=json.load(open(p))
    except Exception:
        return {}
    out={}
    if not isinstance(o,dict):
        return out
    for x in o.get("totalDataChart") or []:
        try:
            d=datetime.datetime.fromtimestamp(int(x[0]),UTC).date()
            v=float(x[1] or 0)
            if v>0:
                out[d]=v
        except Exception:
            pass
    return out

@lru_cache(None)
def tvl_daily(chain):
    p=CACHE/("tvl-"+chain.replace(" ","_")+".json")
    if not p.exists() or p.stat().st_size<100:
        url="https://api.llama.fi/v2/historicalChainTvl/"+urllib.parse.quote(chain)
        q=subprocess.run(["curl","--http1.1","-L","--retry","2",
                          "--retry-all-errors","--connect-timeout","8",
                          "--max-time","30","-sS","-o",str(p),url])
        if q.returncode:
            return {}
    try:
        o=json.load(open(p))
    except Exception:
        return {}
    out={}
    if not isinstance(o,list):
        return out
    for x in o:
        try:
            d=datetime.datetime.fromtimestamp(int(x["date"]),UTC).date()
            v=float(x.get("tvl") or 0)
            if v>0:
                out[d]=v
        except Exception:
            pass
    return out

def mean(xs):
    return sum(xs)/len(xs) if xs else 0.0
def summarize(events):
    if not events:
        return {"n":0}
    symbols=sorted(set(x["symbol"] for x in events))
    by_symbol={}
    positive=0
    for sym in symbols:
        q=[x for x in events if x["symbol"]==sym]
        m7=mean([x["r7"] for x in q])
        by_symbol[sym]={
            "n":len(q),
            "mean1":mean([x["r1"] for x in q]),
            "mean3":mean([x["r3"] for x in q]),
            "mean7":m7,
            "win7":mean([1.0 if x["r7"]>0 else 0.0 for x in q])
        }
        if m7>0:
            positive+=1
    return {
        "n":len(events), "symbols":len(symbols),
        "mean1":mean([x["r1"] for x in events]),
        "mean3":mean([x["r3"] for x in events]),
        "mean7":mean([x["r7"] for x in events]),
        "win7":mean([1.0 if x["r7"]>0 else 0.0 for x in events]),
        "positive_symbols7":positive,
        "positive_symbol_fraction7":positive/len(symbols),
        "by_symbol":by_symbol
    }

universe=json.loads((ROOT/"inputs"/"universe.json").read_text())
DISC_START=datetime.date(2023,1,1)
DISC_END=datetime.date(2024,12,31)

def summarize_events(events):
    if not events:
        return {"n":0}
    symbols=sorted(set(x["symbol"] for x in events))
    by_symbol={}
    positive=0
    for sym in symbols:
        q=[x for x in events if x["symbol"]==sym]
        m7=mean([x["r7"] for x in q])
        by_symbol[sym]={
            "n":len(q),
            "mean1":mean([x["r1"] for x in q]),
            "mean3":mean([x["r3"] for x in q]),
            "mean7":m7,
            "win7":mean([1.0 if x["r7"]>0 else 0.0 for x in q])
        }
        if m7>0:
            positive+=1
    return {
        "n":len(events),
        "symbols":len(symbols),
        "mean1":mean([x["r1"] for x in events]),
        "mean3":mean([x["r3"] for x in events]),
        "mean7":mean([x["r7"] for x in events]),
        "win7":mean([1.0 if x["r7"]>0 else 0.0 for x in events]),
        "positive_symbols7":positive,
        "positive_symbol_fraction7":positive/len(symbols),
        "by_symbol":by_symbol
    }

events=[]
source_meta={}
for row in universe:
    sym=row["symbol"]
    chain=TVL_MAP[sym]
    eligible=datetime.date.fromisoformat(row["eligible_from"])
    px=price_daily_window(sym,"2023-01","2024-12")
    tv={d:v for d,v in tvl_daily(chain).items() if d<=DISC_END}
    if not tv:
        source_meta[sym]={"chain":chain,"status":"no_tvl_history"}
        print("NO_TVL",sym,chain,flush=True)
        continue
    n=0
    d=DISC_START
    while d<=DISC_END:
        if d<eligible:
            d+=datetime.timedelta(days=1); continue
        need=[d-datetime.timedelta(days=k) for k in range(9)]
        if any(x not in tv or tv[x]<=0 for x in need):
            d+=datetime.timedelta(days=1); continue
        score=math.log(tv[d]/tv[d-datetime.timedelta(days=7)])
        prev=math.log(tv[d-datetime.timedelta(days=1)]/tv[d-datetime.timedelta(days=8)])
        direction=1 if prev<=0<score else (-1 if prev>=0>score else 0)
        if direction==0 or d not in px or px[d][2]<5_000_000:
            d+=datetime.timedelta(days=1); continue
        fwd=[d+datetime.timedelta(days=k) for k in range(1,8)]
        if fwd[-1].year!=d.year or any(x not in px for x in fwd):
            d+=datetime.timedelta(days=1); continue
        entry=px[fwd[0]][0]
        if entry<=0:
            d+=datetime.timedelta(days=1); continue
        e={
            "symbol":sym,"chain":row["chain"],"tvl_chain":chain,
            "signal_date":d.isoformat(),"year":d.year,
            "direction":"LONG" if direction>0 else "SHORT",
            "score_prev":prev,"score_now":score,
            "signal_tvl":tv[d],"quote_volume":px[d][2],
            "r1":direction*math.log(px[fwd[0]][1]/entry),
            "r3":direction*math.log(px[fwd[2]][1]/entry),
            "r7":direction*math.log(px[fwd[6]][1]/entry)
        }
        events.append(e); n+=1
        d+=datetime.timedelta(days=1)
    source_meta[sym]={
        "chain":row["chain"],"tvl_chain":chain,
        "eligible_from":eligible.isoformat(),
        "tvl_first":min(tv).isoformat(),"tvl_last":max(tv).isoformat(),
        "price_days":len(px),"events":n
    }
    print("SYMBOL",sym,chain,"events",n,flush=True)

disc=summarize_events(events)
y23=summarize_events([x for x in events if x["year"]==2023])
y24=summarize_events([x for x in events if x["year"]==2024])
triggered=disc.get("symbols",0)
weeks=((datetime.date(2025,1,1)-DISC_START).days/7.0)*triggered if triggered else 0
freq=len(events)/weeks if weeks else 0
gate=(
    disc.get("mean7",0)>=0.0025 and
    disc.get("positive_symbol_fraction7",0)>=0.60 and
    freq>=0.30 and
    y23.get("mean7",0)>0 and y24.get("mean7",0)>0
)
summary={
    "version":"v148",
    "status":"promote_oos" if gate else "frozen_failed_discovery",
    "gate_pass":gate,
    "discovery_2023_2024":disc,
    "2023":y23,
    "2024":y24,
    "frequency_per_symbol_week":freq,
    "source_meta":source_meta,
    "oos_evaluated":False
}
(ROOT/"results"/"events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results"/"summary.json").write_text(json.dumps(summary,indent=2))
print("SUMMARY",json.dumps(summary,indent=2),flush=True)
