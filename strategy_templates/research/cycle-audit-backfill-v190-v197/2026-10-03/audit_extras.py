import json
from pathlib import Path
B=Path("strategy_templates/research")
newroot=B/"cycle-audit-backfill-v190-v197/2026-10-03/results"
old={
"v190":B/"trading-invariant-stress-reversal/2026-10-03-early-gate/results/events.json",
"v192":B/"intrahour-volatility-signature-noise-reversal/2026-10-03-early-gate/results/events.json",
"v196":B/"intrahour-open-reference-recross-reversal/2026-10-03-early-gate/results/events.json"}
for k,p in old.items():
    a=json.loads(p.read_text())
    b=json.loads((newroot/f"{k}_events.json").read_text())
    ok={(e["symbol"],int(e["signal_time"]),e["side"]) for e in a}
    nb={(e["symbol"],int(e["signal_time"]),e["side"]) for e in b if int(e["year"]) in (2023,2024)}
    extra=sorted(nb-ok)
    print("\n",k,"extra",len(extra))
    for x in extra:
        import datetime
        print(x[0],datetime.datetime.fromtimestamp(x[1]/1000,datetime.timezone.utc).isoformat(),x[2])
