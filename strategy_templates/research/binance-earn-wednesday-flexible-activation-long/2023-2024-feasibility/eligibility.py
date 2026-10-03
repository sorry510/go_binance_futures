import csv, datetime, io, json, time, urllib.request, urllib.error, zipfile
from functools import lru_cache
from pathlib import Path

UTC = datetime.timezone.utc
ROOT = Path("strategy_templates/research/binance-earn-wednesday-flexible-activation-long/2023-2024-feasibility")
VISION = "https://data.binance.vision/data/futures/um/monthly/klines"
UA = "Mozilla/5.0"
CACHE = Path("/tmp/binance-earn-wednesday-eligibility")
CACHE.mkdir(exist_ok=True)
EVENTS = json.loads((ROOT/"inputs/raw_events.json").read_text())
HOUR = 3600000
DAY = 86400000

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
    return datetime.datetime.fromtimestamp(ms/1000, UTC).strftime("%Y-%m")

def previous_month(ym):
    y,m = map(int,ym.split("-"))
    if m == 1:
        return f"{y-1:04d}-12"
    return f"{y:04d}-{m-1:02d}"

@lru_cache(maxsize=None)
def load_month(symbol, ym):
    path = CACHE/f"{symbol}-1h-{ym}.zip"
    miss = CACHE/f"{symbol}-1h-{ym}.missing"
    if miss.exists():
        return []
    if not path.exists():
        url = f"{VISION}/{symbol}/1h/{symbol}-1h-{ym}.zip"
        b = get_bytes(url)
        if b is None:
            miss.write_text("404")
            return []
        path.write_bytes(b)
    out = []
    with zipfile.ZipFile(path) as z:
        with z.open(z.namelist()[0]) as fh:
            for r in csv.reader(io.TextIOWrapper(fh)):
                if not r or not r[0].isdigit() or len(r) < 8:
                    continue
                t = int(r[0]); t = t//1000 if t > 10**15 else t
                out.append((t,float(r[7])))
    return out
eligible = []
excluded = []

for i,e in enumerate(EVENTS,1):
    sym = e["symbol"]
    event_ms = int(e["publish_ms"])
    cutoff_ms = event_ms - 730*DAY
    cutoff_ym = month_key(cutoff_ms)
    cutoff_prev = previous_month(cutoff_ym)

    history_rows = load_month(sym, cutoff_ym) + load_month(sym, cutoff_prev)
    has_history = any(t <= cutoff_ms for t,qv in history_rows)
    if not has_history:
        excluded.append({**e,"eligible":False,"reason":"history_lt_730d_or_missing"})
        print("EXCLUDED",i,len(EVENTS),e["asset"],"history",flush=True)
        continue

    event_ym = month_key(event_ms)
    prev_ym = previous_month(event_ym)
    rows = load_month(sym, prev_ym) + load_month(sym, event_ym)
    hour0 = (event_ms//HOUR)*HOUR
    prior = [(t,qv) for t,qv in rows if hour0-24*HOUR <= t < hour0]
    if len(prior) != 24:
        excluded.append({**e,"eligible":False,"reason":"missing_prior_24h","prior_bars":len(prior)})
        print("EXCLUDED",i,len(EVENTS),e["asset"],"prior_bars",len(prior),flush=True)
        continue

    qv24 = sum(qv for t,qv in prior)
    if qv24 < 5_000_000:
        excluded.append({**e,"eligible":False,"reason":"qv_lt_5m","quote_volume_24h":qv24})
        print("EXCLUDED",i,len(EVENTS),e["asset"],"qv",round(qv24,2),flush=True)
        continue

    eligible.append({**e,"eligible":True,"reason":"ok","quote_volume_24h":qv24})
    print("ELIGIBLE",i,len(EVENTS),e["asset"],round(qv24,2),flush=True)
reasons = {}
for x in excluded:
    reasons[x["reason"]] = reasons.get(x["reason"],0) + 1

summary = {
    "raw_events":len(EVENTS),
    "raw_unique_tokens":len({x["asset"] for x in EVENTS}),
    "raw_batches":len({x["article_code"] for x in EVENTS}),
    "eligible_events":len(eligible),
    "eligible_unique_tokens":len({x["asset"] for x in eligible}),
    "eligible_batches":len({x["article_code"] for x in eligible}),
    "excluded_by_reason":reasons,
    "coverage_gate_pass":(
        len(eligible) >= 8
        and len({x["asset"] for x in eligible}) >= 8
        and len({x["article_code"] for x in eligible}) >= 5
    ),
    "post_event_returns_read":False
}
(ROOT/"results/eligible.json").write_text(json.dumps(eligible,ensure_ascii=False,indent=2))
(ROOT/"results/excluded.json").write_text(json.dumps(excluded,ensure_ascii=False,indent=2))
(ROOT/"results/eligibility_summary.json").write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps(summary,ensure_ascii=False,indent=2))
print("ELIGIBLE_TOKENS",sorted({x["asset"] for x in eligible}))
