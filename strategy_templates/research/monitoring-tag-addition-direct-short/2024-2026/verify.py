#!/usr/bin/env python3
import hashlib,json
from pathlib import Path
ROOT=Path(__file__).resolve().parent
def sha256(p): return hashlib.sha256(p.read_bytes()).hexdigest()
expected={
"2024-discovery.py":"5ef27b915de779dc2e9c79cd4a700c172bca8e2b360a14ed3bea0e3337db5c6c",
"2025-oos.py":"be628d1227e6c9a45548f79d5f65b56da05fa6aaa27906c563536cd9868d70f0",
"2026-oos.py":"37e4f4dbda22daf1626dcbf5842d35dad40325759691639f3cfbce4d0c495eea"}
for n,h in expected.items():
    p=ROOT/"replay"/n
    assert sha256(p)==h
    assert "mark=float(x[3])" in p.read_text()
    q=ROOT/"replay_corrected"/n
    t=q.read_text()
    assert "mark=float(x[3])" not in t
    assert "mark if mark is not None else cl" in t
events=json.loads((ROOT/"inputs/eligible_events.json").read_text())
assert sum(len(x) for x in events.values())==35
s=json.loads((ROOT/"results/summary.json").read_text())
for k in ("2024_discovery","2025_oos","2026_second_oos"):
    assert s[k]["status"]=="corrected_funding_parser_revalidated"
assert abs(s["combined_corrected"]["corrected_pf"]-1.0552149645214537)<1e-12
assert s["decision"].startswith("family frozen")
print(json.dumps({"ok":True,"eligible_events":35,"historical_helpers_preserved":True,"corrected_replay_validated":True,"combined_corrected_pf":s["combined_corrected"]["corrected_pf"]},sort_keys=True))
