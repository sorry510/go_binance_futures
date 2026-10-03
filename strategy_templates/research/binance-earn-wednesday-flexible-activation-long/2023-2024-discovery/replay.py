import csv, datetime, io, json, statistics, time, urllib.request, urllib.error, zipfile
from functools import lru_cache
from pathlib import Path

UTC = datetime.timezone.utc
ROOT = Path("strategy_templates/research/binance-earn-wednesday-flexible-activation-long/2023-2024-discovery")
EVENTS = json.loads((ROOT/"inputs/eligible.json").read_text())
VISION = "https://data.binance.vision/data/futures/um/monthly/klines"
CACHE = Path("/tmp/binance-earn-wednesday-eligibility")
CACHE.mkdir(exist_ok=True)
UA = "Mozilla/5.0"
HOUR = 3600000

def get_bytes(url, tries=5):
    last = None
    for k in range(tries):
        try:
            req = urllib.request.Request(url, headers={"User-Agent":UA})
            with urllib.request.urlopen(req, timeout=25) as r:
                return r.read()
        except urllib.error.HTTPError as e:
            if e.code == 404:
                return None
            last = e
        except Exception as e:
            last = e
        time.sleep(.4*(k+1))
    raise last
def month_key(ms):
    return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime("%Y-%m")

def next_month(ym):
    y,m = map(int,ym.split("-"))
    if m == 12:
        return f"{y+1:04d}-01"
    return f"{y:04d}-{m+1:02d}"

@lru_cache(maxsize=None)
def load_month(symbol, ym):
    path = CACHE/f"{symbol}-1h-{ym}.zip"
    miss = CACHE/f"{symbol}-1h-{ym}.missing"
    if miss.exists():
        return {}
    if not path.exists():
        url = f"{VISION}/{symbol}/1h/{symbol}-1h-{ym}.zip"
        b = get_bytes(url)
        if b is None:
            miss.write_text("404")
            return {}
        path.write_bytes(b)
    out = {}
    with zipfile.ZipFile(path) as z:
        with z.open(z.namelist()[0]) as fh:
            for r in csv.reader(io.TextIOWrapper(fh)):
                if not r or not r[0].isdigit() or len(r) < 5:
                    continue
                t = int(r[0]); t = t//1000 if t > 10**15 else t
                out[t] = (float(r[1]),float(r[4]))
    return out
rows = []
for e in EVENTS:
    pub = int(e["publish_ms"])
    entry_t = (pub//HOUR + 1)*HOUR
    ym = month_key(entry_t)
    bars = load_month(e["symbol"],ym)
    bars.update(load_month(e["symbol"],next_month(ym)))
    if entry_t not in bars or bars[entry_t][0] <= 0:
        raise RuntimeError(f"missing entry bar {e['symbol']} {entry_t}")
    entry = bars[entry_t][0]
    rs = {}
    for h in (1,4,12):
        t = entry_t + (h-1)*HOUR
        if t not in bars or bars[t][1] <= 0:
            raise RuntimeError(f"missing endpoint {e['symbol']} {h}h {t}")
        rs[h] = __import__("math").log(bars[t][1]/entry)
    rows.append({
        **e,
        "entry_ms":entry_t,
        "entry":datetime.datetime.fromtimestamp(entry_t/1000,UTC).isoformat(),
        "r1":rs[1],"r4":rs[4],"r12":rs[12]
    })

rows.sort(key=lambda x:(x["publish_ms"],x["asset"]))
cols = ["publish","article_code","asset","symbol","side","quote_volume_24h","entry","r1","r4","r12"]
with (ROOT/"results/events.csv").open("w",newline="") as f:
    w = csv.DictWriter(f,fieldnames=cols,extrasaction="ignore")
    w.writeheader(); w.writerows(rows)
def mean(xs):
    return statistics.fmean(xs) if xs else 0.0

def stats(xs):
    return {
        "n":len(xs),
        "mean_r1":mean([x["r1"] for x in xs]),
        "mean_r4":mean([x["r4"] for x in xs]),
        "mean_r12":mean([x["r12"] for x in xs]),
        "win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in xs]),
    }

by_token = {}
by_batch = {}
by_year = {}
for x in rows:
    by_token.setdefault(x["asset"],[]).append(x)
    by_batch.setdefault(x["article_code"],[]).append(x)
    by_year.setdefault(x["publish"][:4],[]).append(x)

token_stats = {k:stats(v) for k,v in sorted(by_token.items())}
batch_stats = {k:{**stats(v),"title":v[0]["article_title"],"publish":v[0]["publish"]} for k,v in sorted(by_batch.items())}
year_stats = {k:stats(v) for k,v in sorted(by_year.items())}
overall = stats(rows)
positive_tokens = sum(1 for s in token_stats.values() if s["mean_r12"] > 0)
positive_batches = sum(1 for s in batch_stats.values() if s["mean_r12"] > 0)
batch_equal = mean([s["mean_r12"] for s in batch_stats.values()])
token_fraction = positive_tokens/len(token_stats) if token_stats else 0
batch_fraction = positive_batches/len(batch_stats) if batch_stats else 0

gate = (
    overall["mean_r12"] >= 0.005
    and batch_equal >= 0.005
    and token_fraction >= 0.60
    and batch_fraction >= 0.60
    and year_stats.get("2023",{}).get("mean_r12",0) > 0
    and year_stats.get("2024",{}).get("mean_r12",0) > 0
)

summary = {
    "status":"promote_exact" if gate else "frozen_failed_discovery",
    "gate_pass":gate,
    "events":len(rows),
    "unique_tokens":len(token_stats),
    "independent_batches":len(batch_stats),
    "overall":overall,
    "batch_equal_mean_r12":batch_equal,
    "positive_tokens":f"{positive_tokens}/{len(token_stats)}",
    "positive_token_fraction":token_fraction,
    "positive_batches":f"{positive_batches}/{len(batch_stats)}",
    "positive_batch_fraction":batch_fraction,
    "by_year":year_stats,
    "by_token":token_stats,
    "by_batch":batch_stats,
    "oos_2025_plus_evaluated":False,
    "exact_tp_sl_run":False
}
(ROOT/"results/summary.json").write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps({
    "events":len(rows),"unique_tokens":len(token_stats),"batches":len(batch_stats),
    "mean_r1":overall["mean_r1"],"mean_r4":overall["mean_r4"],"mean_r12":overall["mean_r12"],
    "batch_equal_r12":batch_equal,"positive_tokens":summary["positive_tokens"],
    "positive_batches":summary["positive_batches"],"by_year":year_stats,"gate_pass":gate
},ensure_ascii=False,indent=2))
