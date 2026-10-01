#!/usr/bin/env python3
import csv
import io
import json
import math
import time
import urllib.request
import zipfile
from bisect import bisect_left, bisect_right
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import date, datetime, timedelta, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CFG = json.loads((ROOT / "config.json").read_text())
OUT = ROOT / "results"
OUT.mkdir(parents=True, exist_ok=True)
CACHE = Path("/tmp/oi_taker_metrics_early")
CACHE.mkdir(parents=True, exist_ok=True)

TARGET_DAYS = {d for block in CFG["target_blocks"] for d in block}
fetch_days = set()
for block in CFG["target_blocks"]:
    ds = [date.fromisoformat(x) for x in block]
    start, end = min(ds), max(ds)
    cur = start - timedelta(days=1)
    last = end + timedelta(days=1)
    while cur <= last:
        fetch_days.add(cur.isoformat())
        cur += timedelta(days=1)
FETCH_DAYS = sorted(fetch_days)

def fetch_one(task):
    sym, day = task
    p = CACHE / f"{sym}-metrics-{day}.csv"
    if p.exists():
        return {"symbol": sym, "day": day, "path": str(p), "cached": True, "error": None}
    url = f"https://data.binance.vision/data/futures/um/daily/metrics/{sym}/{sym}-metrics-{day}.zip"
    err = None
    for k in range(4):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
            with urllib.request.urlopen(req, timeout=25) as r:
                b = r.read()
            z = zipfile.ZipFile(io.BytesIO(b))
            p.write_bytes(z.read(z.namelist()[0]))
            return {"symbol": sym, "day": day, "path": str(p), "cached": False, "error": None}
        except Exception as e:
            err = repr(e)
            time.sleep(0.5 * (k + 1))
    return {"symbol": sym, "day": day, "path": None, "cached": False, "error": err}

def parse_ts(s):
    return datetime.strptime(s, "%Y-%m-%d %H:%M:%S").replace(tzinfo=timezone.utc).timestamp()

tasks = [(s, d) for s in CFG["symbols"] for d in FETCH_DAYS]
fetch_log = []
with ThreadPoolExecutor(max_workers=20) as ex:
    futures = [ex.submit(fetch_one, x) for x in tasks]
    for i, f in enumerate(as_completed(futures), 1):
        fetch_log.append(f.result())
        if i % 50 == 0:
            print("fetch", i, "/", len(tasks), flush=True)

(OUT / "fetch_log.json").write_text(json.dumps(sorted(fetch_log, key=lambda x:(x["symbol"],x["day"])), indent=2))
missing = [x for x in fetch_log if x["error"]]
if missing:
    print("missing_files", len(missing), flush=True)

def load(sym):
    rows = []
    for day in FETCH_DAYS:
        p = CACHE / f"{sym}-metrics-{day}.csv"
        if not p.exists():
            continue
        with p.open() as f:
            for r in csv.DictReader(f):
                try:
                    oi = float(r["sum_open_interest"])
                    val = float(r["sum_open_interest_value"])
                    tr = float(r["sum_taker_long_short_vol_ratio"])
                    if oi > 0 and val > 0 and tr > 0 and math.isfinite(tr):
                        rows.append((parse_ts(r["create_time"]), oi, val / oi, math.log(tr)))
                except Exception:
                    pass
    dedup = {x[0]: x[1:] for x in rows}
    return [(t, *dedup[t]) for t in sorted(dedup)]

def near_leq(times, rows, target, maxgap=1800):
    i = bisect_right(times, target) - 1
    if i < 0 or target - times[i] > maxgap:
        return None
    return i, rows[i]

def near_geq(times, rows, target, maxgap=1800):
    i = bisect_left(times, target)
    if i >= len(rows) or times[i] - target > maxgap:
        return None
    return i, rows[i]

def analyze(sym):
    rows = load(sym)
    times = [x[0] for x in rows]
    prefix = [0.0]
    for x in rows:
        prefix.append(prefix[-1] + x[3])

    out = []
    prev_oi4 = None
    last_t = None
    for i, (t, oi, px, log_taker) in enumerate(rows):
        if last_t is None or t - last_t > 3600:
            prev_oi4 = None
        last_t = t

        b = near_leq(times, rows, t - 4 * 3600, CFG["max_asof_gap_minutes"] * 60)
        if not b:
            prev_oi4 = None
            continue
        bi, br = b
        oi0 = br[1]
        oi4 = math.log(oi / oi0)

        event_day = datetime.fromtimestamp(t, timezone.utc).date().isoformat()
        if prev_oi4 is not None and event_day in TARGET_DAYS and oi4 > 0 and prev_oi4 <= 0:
            j = bisect_left(times, t - 4 * 3600)
            count = i - j + 1
            if count > 0:
                mean_log_taker = (prefix[i + 1] - prefix[j]) / count
                direction = 1 if mean_log_taker > 0 else (-1 if mean_log_taker < 0 else 0)
                if direction:
                    rec = {
                        "symbol": sym,
                        "time": datetime.fromtimestamp(t, timezone.utc).isoformat(),
                        "oi4": oi4,
                        "mean_log_taker4": mean_log_taker,
                        "direction": "LONG" if direction > 0 else "SHORT",
                    }
                    ok = True
                    for h in CFG["future_horizons_hours"]:
                        q = near_geq(times, rows, t + h * 3600, CFG["max_asof_gap_minutes"] * 60)
                        if not q:
                            ok = False
                            break
                        fpx = q[1][2]
                        rec[f"fwd{h}h"] = direction * math.log(fpx / px)
                    if ok:
                        out.append(rec)
        prev_oi4 = oi4
    return rows, out

events = []
quality = {}
for sym in CFG["symbols"]:
    rows, ev = analyze(sym)
    events.extend(ev)
    quality[sym] = {
        "rows": len(rows),
        "events": len(ev),
        "first_time": datetime.fromtimestamp(rows[0][0], timezone.utc).isoformat() if rows else None,
        "last_time": datetime.fromtimestamp(rows[-1][0], timezone.utc).isoformat() if rows else None,
    }
    print(sym, "rows", len(rows), "events", len(ev), flush=True)

if events:
    with (OUT / "events.csv").open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(events[0].keys()))
        w.writeheader()
        w.writerows(events)

summary = {
    "status": "early_gate_complete",
    "n": len(events),
    "sample_target_days": len(TARGET_DAYS),
    "by_symbol": {},
    "data_quality": quality,
}
for sym in CFG["symbols"]:
    q = [e for e in events if e["symbol"] == sym]
    z = {"n": len(q)}
    for h in CFG["future_horizons_hours"]:
        vals = [e[f"fwd{h}h"] for e in q]
        z[f"mean_{h}h"] = sum(vals) / len(vals) if vals else None
        z[f"win_{h}h"] = sum(v > 0 for v in vals) / len(vals) if vals else None
    summary["by_symbol"][sym] = z

for h in CFG["future_horizons_hours"]:
    vals = [e[f"fwd{h}h"] for e in events]
    summary[f"mean_{h}h"] = sum(vals) / len(vals) if vals else None
    summary[f"win_{h}h"] = sum(v > 0 for v in vals) / len(vals) if vals else None

positive = sum(
    1 for sym in CFG["symbols"]
    if summary["by_symbol"][sym]["mean_4h"] is not None and summary["by_symbol"][sym]["mean_4h"] > 0
)
summary["positive_symbols_4h"] = positive
summary["passes_early_gate"] = bool(
    summary["mean_4h"] is not None
    and summary["mean_4h"] >= CFG["early_gate"]["min_signed_mean_4h"]
    and positive >= CFG["early_gate"]["min_positive_symbols_4h"]
)
summary["full_discovery_evaluated"] = False
(OUT / "summary.json").write_text(json.dumps(summary, indent=2))
print(json.dumps(summary, indent=2))
