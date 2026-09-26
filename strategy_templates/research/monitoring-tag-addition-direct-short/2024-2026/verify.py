#!/usr/bin/env python3
import hashlib
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parent


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


expected_hashes = {
    "2024-discovery.py": "5ef27b915de779dc2e9c79cd4a700c172bca8e2b360a14ed3bea0e3337db5c6c",
    "2025-oos.py": "be628d1227e6c9a45548f79d5f65b56da05fa6aaa27906c563536cd9868d70f0",
    "2026-oos.py": "37e4f4dbda22daf1626dcbf5842d35dad40325759691639f3cfbce4d0c495eea",
}
for name, expected in expected_hashes.items():
    path = ROOT / "replay" / name
    assert sha256(path) == expected
    text = path.read_text()
    assert "mark=float(x[3])" in text
    assert "expected calc_time,funding_interval_hours,last_funding_rate,mark_price" in text

events = json.loads((ROOT / "inputs/eligible_events.json").read_text())
assert len(events["2024_discovery"]) == 9
assert len(events["2025_oos"]) == 9
assert len(events["2026_second_oos"]) == 17
assert sum(len(group) for group in events.values()) == 35

summary = json.loads((ROOT / "results/summary.json").read_text())
assert summary["2024_discovery"]["eligible_events"] == 9
assert summary["2025_oos"]["eligible_events"] == 9
assert summary["2026_second_oos"]["eligible_events"] == 17
assert summary["combined_historical"]["eligible_events"] == 35
assert all("historical_unrevalidated_due_funding_parser" == summary[key]["status"] for key in ("2024_discovery", "2025_oos", "2026_second_oos", "combined_historical"))

print(json.dumps({
    "ok": True,
    "eligible_events": 35,
    "replay_files": 3,
    "known_funding_parser_bug_present": True,
    "historical_metrics_revalidated": False
}, sort_keys=True))
