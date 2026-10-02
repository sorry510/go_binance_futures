import csv, json, math, time, urllib.parse, urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import date, datetime, timedelta, timezone
from pathlib import Path

ROOT = Path("strategy_templates/research/defillama-protocol-tvl-momentum/2026-10-02-discovery")
UNIVERSE = json.loads((ROOT / "inputs/universe.json").read_text())
UA = "Mozilla/5.0"
DISC_START = date(2023, 1, 1)
DISC_END = date(2024, 12, 31)
BINANCE_END_MS = int(datetime(2025, 1, 1, tzinfo=timezone.utc).timestamp() * 1000) - 1
BINANCE_START_MS = int(datetime(2019, 1, 1, tzinfo=timezone.utc).timestamp() * 1000)

def get_json(url, retries=4):
    last = None
    for i in range(retries):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": UA, "Accept": "application/json"})
            with urllib.request.urlopen(req, timeout=25) as r:
                return json.loads(r.read())
        except Exception as e:
            last = e
            time.sleep(0.5 * (i + 1))
    raise last
def fetch_tvl(slug):
    obj = get_json("https://api.llama.fi/protocol/" + urllib.parse.quote(slug))
    by_day = {}
    for r in obj.get("tvl", []):
        ts = int(r.get("date", 0))
        v = float(r.get("totalLiquidityUSD") or 0)
        if ts <= 0 or v <= 0:
            continue
        d = datetime.fromtimestamp(ts, timezone.utc).date()
        if d <= DISC_END:
            by_day[d] = v
    return by_day

def fetch_binance(symbol):
    bars = {}
    start = BINANCE_START_MS
    url0 = "https://fapi.binance.com/fapi/v1/klines"
    while start <= BINANCE_END_MS:
        q = urllib.parse.urlencode({
            "symbol": symbol, "interval": "1d", "startTime": start,
            "endTime": BINANCE_END_MS, "limit": 1500
        })
        arr = get_json(url0 + "?" + q)
        if not arr:
            break
        for r in arr:
            dt = datetime.fromtimestamp(int(r[0]) / 1000, timezone.utc).date()
            bars[dt] = {"open": float(r[1]), "close": float(r[4]), "qv": float(r[7])}
        nxt = int(arr[-1][0]) + 86400000
        if nxt <= start:
            break
        start = nxt
        if len(arr) < 1500:
            break
        time.sleep(0.05)
    return bars
def load_asset(row):
    asset = row["asset"]
    slug = row["slug"]
    symbol = asset + "USDT"
    return asset, slug, symbol, fetch_tvl(slug), fetch_binance(symbol)

def signed_returns(side, bars, signal_day):
    entry_day = signal_day + timedelta(days=1)
    if entry_day not in bars:
        return None
    entry = bars[entry_day]["open"]
    if entry <= 0:
        return None
    out = {}
    for h in (1, 3, 7):
        end_day = signal_day + timedelta(days=h)
        if end_day.year != signal_day.year or end_day not in bars:
            return None
        close = bars[end_day]["close"]
        if close <= 0:
            return None
        out[h] = side * math.log(close / entry)
    return out

def mean(xs):
    return sum(xs) / len(xs) if xs else 0.0
events = []
failures = []
loaded = {}

with ThreadPoolExecutor(max_workers=6) as ex:
    futs = {ex.submit(load_asset, row): row for row in UNIVERSE}
    for fut in as_completed(futs):
        row = futs[fut]
        try:
            asset, slug, symbol, tvl, bars = fut.result()
            loaded[asset] = (slug, symbol, tvl, bars)
            print("LOADED", asset, "tvl_days", len(tvl), "bars", len(bars), flush=True)
        except Exception as e:
            failures.append({"asset": row["asset"], "slug": row["slug"], "error": str(e)})
            print("LOAD_FAIL", row["asset"], e, flush=True)

for row in UNIVERSE:
    asset = row["asset"]
    if asset not in loaded:
        continue
    slug, symbol, tvl, bars = loaded[asset]
    if not bars:
        continue
    first_bar = min(bars)
    d = DISC_START
    while d <= DISC_END:
        d7 = d - timedelta(days=7)
        d1 = d - timedelta(days=1)
        d8 = d - timedelta(days=8)
        if d in tvl and d7 in tvl and d1 in tvl and d8 in tvl:
            now = math.log(tvl[d] / tvl[d7])
            prev = math.log(tvl[d1] / tvl[d8])
            side = 1 if prev <= 0 < now else (-1 if prev >= 0 > now else 0)
            if side:
                if (d - first_bar).days >= 730 and d in bars and bars[d]["qv"] >= 5_000_000:
                    rs = signed_returns(side, bars, d)
                    if rs:
                        events.append({
                            "asset": asset, "slug": slug, "symbol": symbol,
                            "signal_date": d.isoformat(), "side": "LONG" if side > 0 else "SHORT",
                            "score_prev": prev, "score_now": now, "qv": bars[d]["qv"],
                            "r1": rs[1], "r3": rs[3], "r7": rs[7],
                        })
        d += timedelta(days=1)
events.sort(key=lambda x: (x["signal_date"], x["asset"], x["side"]))
with (ROOT / "results/events.csv").open("w", newline="") as f:
    cols = ["asset","slug","symbol","signal_date","side","score_prev","score_now","qv","r1","r3","r7"]
    w = csv.DictWriter(f, fieldnames=cols)
    w.writeheader()
    w.writerows(events)

by_symbol = {}
by_year = {}
by_side = {}
for e in events:
    by_symbol.setdefault(e["asset"], []).append(e)
    by_year.setdefault(e["signal_date"][:4], []).append(e)
    by_side.setdefault(e["side"], []).append(e)

def summarize(rows):
    return {
        "n": len(rows),
        "mean_r1": mean([x["r1"] for x in rows]),
        "mean_r3": mean([x["r3"] for x in rows]),
        "mean_r7": mean([x["r7"] for x in rows]),
        "win_r7": mean([1.0 if x["r7"] > 0 else 0.0 for x in rows]),
        "long": sum(1 for x in rows if x["side"] == "LONG"),
        "short": sum(1 for x in rows if x["side"] == "SHORT"),
    }

triggered = sorted(by_symbol)
positive = sum(1 for a in triggered if summarize(by_symbol[a])["mean_r7"] > 0)
weeks = ((date(2025,1,1) - DISC_START).days / 7.0) * len(triggered) if triggered else 0
freq = len(events) / weeks if weeks else 0
overall = summarize(events)
gate_pass = (
    overall["mean_r7"] >= 0.0025
    and triggered
    and positive / len(triggered) >= 0.60
    and freq >= 0.30
    and summarize(by_year.get("2023", []))["mean_r7"] > 0
    and summarize(by_year.get("2024", []))["mean_r7"] > 0
)

summary = {
    "status": "promote_oos" if gate_pass else "frozen_failed_discovery",
    "gate_pass": gate_pass,
    "events": len(events),
    "triggered_symbols": len(triggered),
    "positive_symbols": f"{positive}/{len(triggered)}",
    "frequency_per_symbol_week": freq,
    "overall": overall,
    "by_year": {k: summarize(v) for k, v in sorted(by_year.items())},
    "by_side": {k: summarize(v) for k, v in sorted(by_side.items())},
    "by_symbol": {k: summarize(v) for k, v in sorted(by_symbol.items())},
    "load_failures": failures,
    "oos_2025_2026_evaluated": False,
}
(ROOT / "results/summary.json").write_text(json.dumps(summary, indent=2))
(ROOT / "results/load_failures.json").write_text(json.dumps(failures, indent=2))

print(json.dumps({
    "events": len(events),
    "triggered_symbols": len(triggered),
    "positive_symbols": summary["positive_symbols"],
    "frequency_per_symbol_week": freq,
    "mean_r1": overall["mean_r1"],
    "mean_r3": overall["mean_r3"],
    "mean_r7": overall["mean_r7"],
    "2023_r7": summary["by_year"].get("2023", {}).get("mean_r7", 0),
    "2024_r7": summary["by_year"].get("2024", {}).get("mean_r7", 0),
    "gate_pass": gate_pass,
    "load_failures": len(failures),
}, indent=2))
