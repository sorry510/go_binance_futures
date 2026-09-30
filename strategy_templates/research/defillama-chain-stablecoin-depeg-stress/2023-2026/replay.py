import csv, io, zipfile, urllib.request, urllib.parse, urllib.error
import datetime, json, math, time, subprocess
import xml.etree.ElementTree as ET
from pathlib import Path
from functools import lru_cache

UTC = datetime.timezone.utc
UA = "Mozilla/5.0"
ROOT = Path(__file__).resolve().parent
CACHE = Path("/tmp/chain_stablecoin_depeg_cache")
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

def add2(d):
    try:
        return d.replace(year=d.year+2)
    except ValueError:
        return d.replace(year=d.year+2, day=28)
ELIGIBILITY_SOURCE = json.load(open(ROOT/"inputs"/"source_eligibility_universe.json"))
FIRST_FUTURES = {
    x["symbol"]: datetime.datetime.fromisoformat(x["first_futures"])
    for x in ELIGIBILITY_SOURCE if x.get("first_futures")
}
@lru_cache(None)
def first_kline(sym):
    return FIRST_FUTURES.get(sym)
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
    shared=Path("/tmp/chain_dex_residual_cache")/f"{sym}-{mo}.zip"
    if shared.exists():
        return shared
    url=f"{DV}/{sym}/1d/{sym}-1d-{mo}.zip"
    for attempt in range(4):
        try:
            req=urllib.request.Request(url, headers={"User-Agent":UA})
            p.write_bytes(urllib.request.urlopen(req, timeout=20).read())
            return p
        except urllib.error.HTTPError as e:
            if e.code==404:
                p.write_bytes(b"")
                return p
        except Exception:
            pass
        time.sleep(.4*(attempt+1))
    p.write_bytes(b"")
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
def stablecoin_daily(chain):
    p=CACHE/("stable-"+chain.replace(" ","_")+".json")
    if not p.exists() or p.stat().st_size<100:
        url="https://stablecoins.llama.fi/stablecoincharts/"+urllib.parse.quote(chain)
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
            units=float(((x.get("totalCirculating") or {}).get("peggedUSD")) or 0)
            usd=float(((x.get("totalCirculatingUSD") or {}).get("peggedUSD")) or 0)
            if units>0 and usd>0:
                peg=usd/units
                stress=abs(math.log(peg))
                out[d]=(peg,stress,units,usd)
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
universe=[]

@lru_cache(None)
def prepared_rows(sym, start_ym, end_ym):
    chain=CHAIN_MAP[sym]
    supply=stablecoin_daily(chain)
    if not supply:
        return None, None, [], "no_stablecoin_history"
    first=first_kline(sym)
    if not first:
        return None, None, [], "no_binance_history"
    eligible=add2(first).date()
    px=price_daily_window(sym,start_ym,end_ym)
    dates=sorted(set(px)&set(supply))
    rows=[]
    for d in dates:
        o,c,qv=px[d]
        peg,stress,units,usd=supply[d]
        rows.append((d,o,c,qv,peg,stress,units,usd))
    flow=[None]*len(rows)
    for i in range(13,len(rows)):
        if not all((rows[j][0]-rows[j-1][0]).days==1
                   for j in range(i-12,i+1)):
            continue
        cur=mean([rows[j][5] for j in range(i-6,i+1)])
        prev=mean([rows[j][5] for j in range(i-13,i-6)])
        if cur>0 and prev>0:
            flow[i]=math.log(cur/prev)
    return first, eligible, (rows,flow), "ok"
def generate(years, start_ym, end_ym, collect_universe=False):
    events=[]
    for sym in CHAIN_MAP:
        first,eligible,prepared,status=prepared_rows(sym,start_ym,end_ym)
        if status!="ok":
            if collect_universe:
                universe.append({"symbol":sym,"chain":CHAIN_MAP[sym],"status":status})
            print("PERIOD",sorted(years),sym,status,flush=True)
            continue
        rows,flow=prepared
        n=0
        for i in range(14,len(rows)-8):
            d=rows[i][0]
            if d.year not in years:
                continue
            if d<eligible or flow[i] is None or flow[i-1] is None:
                continue
            if rows[i][3] < 5_000_000:
                continue
            direction=0
            if flow[i-1]<=0 and flow[i]>0:
                direction=-1  # depeg stress rising => SHORT native token
            elif flow[i-1]>=0 and flow[i]<0:
                direction=1   # depeg stress easing => LONG native token
            else:
                continue
            if any((rows[i+k][0]-rows[i+k-1][0]).days!=1
                   for k in range(1,8)):
                continue
            if rows[i+7][0].year not in years:
                continue
            entry=rows[i+1][1]
            if entry<=0:
                continue
            events.append({
                "symbol":sym,
                "chain":CHAIN_MAP[sym],
                "signal_date":d.isoformat(),
                "year":d.year,
                "direction":"LONG" if direction>0 else "SHORT",
                "stress_flow_7v7":flow[i],
                "implicit_peg":rows[i][4],
                "peg_stress":rows[i][5],
                "stablecoin_units":rows[i][6],
                "stablecoin_usd_value":rows[i][7],
                "quote_volume":rows[i][3],
                "r1":direction*math.log(rows[i+1][2]/entry),
                "r3":direction*math.log(rows[i+3][2]/entry),
                "r7":direction*math.log(rows[i+7][2]/entry)
            })
            n+=1
        if collect_universe:
            universe.append({
                "symbol":sym,
                "chain":CHAIN_MAP[sym],
                "status":"ok",
                "first_futures":first.isoformat(),
                "eligible_from":eligible.isoformat(),
                "stablecoin_first":min(stablecoin_daily(CHAIN_MAP[sym])).isoformat(),
                "aligned_days":len(rows),
                "discovery_events":n
            })
        print("PERIOD",sorted(years),sym,"events",n,flush=True)
    return events
discovery=generate({2023,2024},"2022-12","2025-01",collect_universe=True)
disc_summary=summarize(discovery)
gate_pass=(
    disc_summary.get("mean7",0.0)>=0.0025 and
    disc_summary.get("positive_symbol_fraction7",0.0)>=0.60
)
summary={
    "discovery_2023_2024":disc_summary,
    "discovery_gate_pass":gate_pass,
    "oos_evaluated":False
}
all_events=list(discovery)

if gate_pass:
    oos1=generate({2025},"2024-12","2026-01")
    oos2=generate({2026},"2025-12","2026-09")
    summary["oos1_2025"]=summarize(oos1)
    summary["oos2_2026"]=summarize(oos2)
    summary["oos_evaluated"]=True
    all_events.extend(oos1)
    all_events.extend(oos2)

(ROOT/"inputs"/"universe.json").write_text(json.dumps(universe,indent=2))
(ROOT/"results"/"events.json").write_text(json.dumps(all_events,indent=2))
(ROOT/"results"/"summary.json").write_text(json.dumps(summary,indent=2))
print("SUMMARY",json.dumps(summary,indent=2),flush=True)
