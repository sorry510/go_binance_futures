#!/usr/bin/env python3
import csv
import hashlib
import json
import math
from pathlib import Path

ROOT = Path(__file__).resolve().parent


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def agg(rows):
    values = [float(row["norm_return"]) for row in rows]
    gp = sum(value for value in values if value >= 0)
    gl = -sum(value for value in values if value < 0)
    return len(values), gp / gl if gl else math.inf, sum(values)


def close(actual, expected, tol=1e-10):
    if not math.isclose(actual, expected, rel_tol=tol, abs_tol=tol):
        raise AssertionError(f"expected {expected}, got {actual}")


summary = json.loads((ROOT / "results/summary.json").read_text())
id121 = json.loads((ROOT / "strategy/id121-db-template-121.json").read_text())
v33 = json.loads((ROOT / "strategy/v33-source.json").read_text())

assert id121["id"] == 121
assert id121["name"] == "v33 LONG + relaxed daily-ADX SHORT 正式候选"
assert id121["technology"] == v33["technology"]
diff_indexes = [i for i, (left, right) in enumerate(zip(v33["strategy"], id121["strategy"])) if left != right]
assert diff_indexes == [1]
assert id121["strategy"][1]["type"] == "short"
assert "adx_1d_14.ADX[1] >= adx_1d_14.ADX[3]" in v33["strategy"][1]["code"]
assert "adx_1d_14.ADX[1] >= adx_1d_14.ADX[3]" not in id121["strategy"][1]["code"]
assert "adx_4h_14.ADX[1] - adx_4h_14.ADX[3] >= 2" in v33["strategy"][1]["code"]
assert "adx_4h_14.ADX[1] - adx_4h_14.ADX[3] >= 2" not in id121["strategy"][1]["code"]

rows = list(csv.DictReader((ROOT / "results/paired_normalized.csv").open()))
by_strategy = {name: [row for row in rows if row["strategy"] == name] for name in ("ID121", "V54")}

for name in ("ID121", "V54"):
    n, pf, net = agg(by_strategy[name])
    expected = summary["fixed_notional_normalized"][name]
    assert n == expected["trades"]
    close(pf, expected["pf"])
    close(net, expected["net"])

base = {(r["symbol"], r["side"], r["entry_time"]): r for r in by_strategy["ID121"]}
v54 = {(r["symbol"], r["side"], r["entry_time"]): r for r in by_strategy["V54"]}
groups = {
    "COMMON": (set(base) & set(v54), base),
    "ID121_ONLY": (set(base) - set(v54), base),
    "V54_ONLY": (set(v54) - set(base), v54),
}
for name, (keys, source) in groups.items():
    n, pf, net = agg([source[key] for key in keys])
    expected = summary["fixed_notional_normalized"]["paired_attribution"][name]
    assert n == expected["trades"]
    close(pf, expected["pf"])
    close(net, expected["net"])

assert sha256(ROOT / "strategy/v33-source.json") == "f0d7596f2496dbbdd089705a4b53f261c8150a653b551c6d7433bb2c31c795c9"
assert sha256(ROOT / "strategy/v54.json") == "f40b7ea3ad0ab97651f3b8018ae2938e3261d4f4f73f493c92cea91639bc5f0b"
assert sha256(ROOT / "strategy/id121-db-template-121.json") == "5dd21dca486fefe90be671326efa6947220c2c7bfafce312fd157d0dced6b2a8"

print(json.dumps({
    "ok": True,
    "id121_trades": len(by_strategy["ID121"]),
    "v54_trades": len(by_strategy["V54"]),
    "common": len(groups["COMMON"][0]),
    "id121_only": len(groups["ID121_ONLY"][0]),
    "v54_only": len(groups["V54_ONLY"][0]),
    "id121_v33_diff_indexes": diff_indexes,
}, sort_keys=True))
