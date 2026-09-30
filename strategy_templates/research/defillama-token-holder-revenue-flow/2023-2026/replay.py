import csv, io, json, math, subprocess, zipfile
import datetime, urllib.parse
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, as_completed

UTC = datetime.timezone.utc
ROOT = Path(__file__).resolve().parent
SOURCE = json.load(open(ROOT / "inputs/source_map.json"))
CACHE = Path("/tmp/holderrev_flow_cache")
CACHE.mkdir(exist_ok=True)
VISION = "https://data.binance.vision/data/futures/um/monthly/klines"

def add2(d):
    try:
        return d.replace(year=d.year + 2)
    except ValueError:
        return d.replace(year=d.year + 2, day=28)

def month_range():
    out = []
    d = datetime.date(2022, 12, 1)
    end = datetime.date(2025, 2, 1)
    while d < end:
        out.append(d.strftime("%Y-%m"))
        d = (d.replace(day=28) + datetime.timedelta(days=4)).replace(day=1)
    return out
def curl_file(url, path, allow_404=False):
    tmp = Path(str(path) + ".part")
    q = subprocess.run([
        "curl", "--http1.1", "-L", "--retry", "4", "--retry-all-errors",
        "--connect-timeout", "8", "--max-time", "40", "-sS",
        "-w", "%{http_code}", "-o", str(tmp), url
    ], capture_output=True, text=True)
    code = (q.stdout or "")[-3:]
    if q.returncode == 0 and code == "200":
        tmp.replace(path)
        return path
    tmp.unlink(missing_ok=True)
    if allow_404 and code == "404":
        path.write_bytes(b"")
        return path
    raise RuntimeError(f"download failed rc={q.returncode} http={code} url={url}")

def holder_file(slug):
    p = CACHE / f"holder-{slug.replace('/', '_')}.json"
    if p.exists() and p.stat().st_size > 50:
        return p
    url = (
        "https://api.llama.fi/summary/fees/" + urllib.parse.quote(slug) +
        "?excludeTotalDataChartBreakdown=true&dataType=dailyHoldersRevenue"
    )
    return curl_file(url, p)

def kline_file(symbol, month):
    p = CACHE / f"{symbol}-1d-{month}.zip"
    if p.exists():
        return p
    url = f"{VISION}/{symbol}/1d/{symbol}-1d-{month}.zip"
    return curl_file(url, p, allow_404=True)
def parse_kline_zip(path):
    out = {}
    if not path.exists() or path.stat().st_size == 0:
        return out
    try:
        z = zipfile.ZipFile(path)
    except zipfile.BadZipFile:
        return out
    fn = z.namelist()[0]
    for row in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not row or not row[0].isdigit():
            continue
        t = int(row[0])
        if t > 10**15:
            t //= 1000
        d = datetime.datetime.fromtimestamp(t / 1000, UTC).date()
        out[d] = {
            "open": float(row[1]),
            "close": float(row[4]),
            "quote_volume": float(row[7]),
        }
    return out

def first_futures_date(item):
    sym = item["futures_symbol"]
    p = kline_file(sym, item["first_month"])
    rows = parse_kline_zip(p)
    return min(rows) if rows else None

def price_daily(item):
    sym = item["futures_symbol"]
    out = {}
    for month in month_range():
        out.update(parse_kline_zip(kline_file(sym, month)))
    return out

def holder_daily(item):
    out = {}
    for slug in item["included_slugs"]:
        obj = json.load(open(holder_file(slug)))
        for row in obj.get("totalDataChart") or []:
            d = datetime.datetime.fromtimestamp(int(row[0]), UTC).date()
            out[d] = out.get(d, 0.0) + float(row[1] or 0)
    return out
def flow_at(rev, d):
    days = [d - datetime.timedelta(days=i) for i in range(13, -1, -1)]
    if any(x not in rev for x in days):
        return None
    prev = sum(rev[x] for x in days[:7])
    cur = sum(rev[x] for x in days[7:])
    if prev <= 0 or cur <= 0:
        return None
    return math.log(cur / prev), prev, cur

symbols = SOURCE["symbols"]
jobs = []
for item in symbols:
    for slug in item["included_slugs"]:
        jobs.append(("holder", slug, None))
    for month in month_range():
        jobs.append(("kline", item["futures_symbol"], month))
    jobs.append(("first", item["futures_symbol"], item["first_month"]))

def fetch_job(job):
    kind, name, month = job
    if kind == "holder":
        holder_file(name)
    else:
        kline_file(name, month)
    return job

print("DOWNLOAD_JOBS", len(jobs), flush=True)
with ThreadPoolExecutor(max_workers=20) as ex:
    fs = [ex.submit(fetch_job, j) for j in jobs]
    for i, f in enumerate(as_completed(fs), 1):
        f.result()
        if i % 100 == 0 or i == len(fs):
            print("DOWNLOAD", i, "/", len(fs), flush=True)
events = []
skipped = []
eligibility = {}

for item in symbols:
    sym = item["symbol"]
    fsym = item["futures_symbol"]
    first = first_futures_date(item)
    if first is None:
        eligibility[sym] = {"error": "no_first_futures_date"}
        continue
    eligible = add2(first)
    eligibility[sym] = {
        "futures_symbol": fsym,
        "first_futures_date": first.isoformat(),
        "eligible_from": eligible.isoformat(),
        "included_slugs": item["included_slugs"],
    }
    px = price_daily(item)
    rev = holder_daily(item)
    prev_flow = None
    for d in sorted(rev):
        f = flow_at(rev, d)
        if f is None:
            prev_flow = None
            continue
        flow, prev7, cur7 = f
        if prev_flow is None:
            prev_flow = flow
            continue
        direction = 0
        if prev_flow <= 0 < flow:
            direction = 1
        elif prev_flow >= 0 > flow:
            direction = -1
        prev_flow = flow
        if direction == 0 or d.year not in (2023, 2024):
            continue
        if d + datetime.timedelta(days=7) > datetime.date(2024, 12, 31):
            continue
        if d < eligible:
            skipped.append({"symbol": fsym, "signal_date": d.isoformat(), "reason": "history_lt_2y"})
            continue
        if d not in px:
            skipped.append({"symbol": fsym, "signal_date": d.isoformat(), "reason": "no_signal_day_price"})
            continue
        if px[d]["quote_volume"] < 5_000_000:
            skipped.append({"symbol": fsym, "signal_date": d.isoformat(), "reason": "quote_volume_lt_5m"})
            continue
        exit_days = {
            1: d + datetime.timedelta(days=1),
            3: d + datetime.timedelta(days=3),
            7: d + datetime.timedelta(days=7),
        }
        if any(x not in px for x in exit_days.values()):
            skipped.append({"symbol": fsym, "signal_date": d.isoformat(), "reason": "missing_forward_price"})
            continue
        entry_day = exit_days[1]
        entry = px[entry_day]["open"]
        if entry <= 0:
            continue
        ret = {}
        for h, ed in exit_days.items():
            ret[h] = direction * math.log(px[ed]["close"] / entry)
        events.append({
            "symbol": fsym,
            "signal_date": d.isoformat(),
            "year": d.year,
            "direction": "LONG" if direction > 0 else "SHORT",
            "flow7v7": flow,
            "previous_7d_holder_revenue": prev7,
            "current_7d_holder_revenue": cur7,
            "signal_day_quote_volume": px[d]["quote_volume"],
            "entry_date": entry_day.isoformat(),
            "entry_open": entry,
            "r1": ret[1],
            "r3": ret[3],
            "r7": ret[7],
        })
    print("SYMBOL", fsym, "events", sum(x["symbol"] == fsym for x in events), flush=True)
def summarize(es):
    if not es:
        return {"n": 0}
    syms = sorted(set(x["symbol"] for x in es))
    by_symbol = {}
    positive = 0
    for sym in syms:
        q = [x for x in es if x["symbol"] == sym]
        mean7 = sum(x["r7"] for x in q) / len(q)
        if mean7 > 0:
            positive += 1
        by_symbol[sym] = {
            "n": len(q),
            "mean1": sum(x["r1"] for x in q) / len(q),
            "mean3": sum(x["r3"] for x in q) / len(q),
            "mean7": mean7,
            "win7": sum(x["r7"] > 0 for x in q) / len(q),
        }
    return {
        "n": len(es),
        "symbols": len(syms),
        "mean1": sum(x["r1"] for x in es) / len(es),
        "mean3": sum(x["r3"] for x in es) / len(es),
        "mean7": sum(x["r7"] for x in es) / len(es),
        "win7": sum(x["r7"] > 0 for x in es) / len(es),
        "positive_symbols7": positive,
        "breadth7": positive / len(syms),
        "by_symbol": by_symbol,
    }

all_sum = summarize(events)
y2023 = summarize([x for x in events if x["year"] == 2023])
y2024 = summarize([x for x in events if x["year"] == 2024])
gate = (
    all_sum.get("n", 0) >= 80
    and all_sum.get("symbols", 0) >= 8
    and all_sum.get("mean7", -999) >= 0.0025
    and all_sum.get("breadth7", 0) >= 0.60
    and y2023.get("mean7", -999) > 0
    and y2024.get("mean7", -999) > 0
)
summary = {
    "discovery_2023_2024": all_sum,
    "2023": y2023,
    "2024": y2024,
    "gate_pass": gate,
    "gate": "n>=80, symbols>=8, mean7>=0.25%, breadth>=60%, 2023>0, 2024>0",
}
(ROOT / "results/events.json").write_text(json.dumps(events, indent=2))
(ROOT / "results/skipped.json").write_text(json.dumps(skipped, indent=2))
(ROOT / "results/summary.json").write_text(json.dumps(summary, indent=2))
(ROOT / "inputs/eligibility.json").write_text(json.dumps(eligibility, indent=2))
print("SUMMARY", json.dumps(summary, indent=2), flush=True)
