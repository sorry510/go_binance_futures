import hashlib
import json
import math
from collections import Counter
from pathlib import Path


ROOT = Path(__file__).resolve().parent


def load(path):
    return json.loads((ROOT / path).read_text())


events = load("inputs/events.json")
eligibility = load("inputs/eligibility.json")
trades = load("results/trades.json")
summary = load("results/summary.json")

key = lambda x: (x["code"], x["base"], x["publish_ms"])
event_keys = {key(x) for x in events}
eligibility_keys = {key(x) for x in eligibility}
eligible_keys = {key(x) for x in eligibility if x["eligible"]}
trade_keys = {key(x) for x in trades}

assert len(event_keys) == len(events), "duplicate event keys"
assert event_keys == eligibility_keys, "events and eligibility universe differ"
assert eligible_keys == trade_keys, "eligible events and trades differ"
assert all(
    t["entry_ts"] == ((t["publish_ms"] // 60000) + 1) * 60000 for t in trades
), "entry timestamp violates next-complete-minute rule"

net = sum(t["net_pct"] for t in trades)
funding = sum(t["funding_pct"] for t in trades)
fees = sum(t["fees_pct"] for t in trades)
gross_profit = sum(t["net_pct"] for t in trades if t["net_pct"] > 0)
gross_loss = -sum(t["net_pct"] for t in trades if t["net_pct"] < 0)
pf = gross_profit / gross_loss
triggers = Counter(t["trigger"] for t in trades)

assert summary["token_events"] == len(events)
assert summary["eligible_events"] == len(eligible_keys)
assert summary["trades"] == len(trades)
assert summary["wins"] == sum(t["net_pct"] > 0 for t in trades)
assert summary["losses"] == sum(t["net_pct"] <= 0 for t in trades)
assert summary["triggers"] == {k: triggers[k] for k in ["TP", "SL", "TIME", "EOF"]}
assert math.isclose(summary["net_pct_sum"], net, rel_tol=1e-12)
assert math.isclose(summary["funding_pct_sum"], funding, rel_tol=1e-12)
assert math.isclose(summary["fees_pct_sum"], fees, rel_tol=1e-12)
assert math.isclose(summary["pf"], pf, rel_tol=1e-12)

manifest = ROOT / "manifest.sha256"
if manifest.exists():
    for line in manifest.read_text().splitlines():
        expected, rel = line.split("  ", 1)
        actual = hashlib.sha256((ROOT / rel).read_bytes()).hexdigest()
        assert actual == expected, f"hash mismatch: {rel}"

print(
    json.dumps(
        {
            "ok": True,
            "events": len(events),
            "eligible": len(eligible_keys),
            "trades": len(trades),
            "pf": pf,
            "net_pct_sum": net,
        },
        sort_keys=True,
    )
)
