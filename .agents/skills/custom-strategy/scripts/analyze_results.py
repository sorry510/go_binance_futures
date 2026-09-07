#!/usr/bin/env python3
"""Review a frozen DBX result export with Decimal PnL; no database access or writes."""

import argparse
import json
from collections import defaultdict
from datetime import datetime, timedelta, timezone
from decimal import Decimal, InvalidOperation
from pathlib import Path

ZERO = Decimal(0)
EPS = Decimal("0.000000001")
ROUND_TOLERANCE = Decimal("0.000501")
HOUR_MS = Decimal(3600000)
TZ = timezone(timedelta(hours=8))
NUMERIC = (
    "price", "close_price", "position_amt", "usdt", "leverage", "profit", "loss",
    "close_profit", "open_fee_rate", "close_fee_rate", "createTime", "updateTime",
)


def quantile(values, percentile):
    if not values:
        return None
    values = sorted(values)
    position = Decimal(len(values) - 1) * Decimal(str(percentile))
    lower = int(position)
    upper = min(lower + 1, len(values) - 1)
    return values[lower] + (values[upper] - values[lower]) * (position - lower)


def prepare(row, as_of):
    result = dict(row)
    values = {key: Decimal(str(row[key]).strip()) for key in NUMERIC}
    if not all(value.is_finite() for value in values.values()):
        raise ValueError("non-finite numeric value")
    if (values["price"] <= 0 or values["close_price"] < 0 or
            values["position_amt"] == 0 or values["usdt"] <= 0 or
            values["leverage"] <= 0 or values["profit"] < 0 or values["loss"] < 0):
        raise ValueError("invalid price, quantity, stake, leverage or gates")
    if any(values[key] < 0 or values[key] > 1 for key in ("open_fee_rate", "close_fee_rate")):
        raise ValueError("fee rate outside [0, 1]")
    if not (0 < values["createTime"] <= values["updateTime"] <= as_of):
        raise ValueError("invalid or future timestamps")
    side = str(row["position_side"]).strip().upper()
    if side not in ("LONG", "SHORT") or (values["position_amt"] > 0) != (side == "LONG"):
        raise ValueError("side and signed quantity disagree")
    result.update(values)
    result["position_side"] = side
    result["closed"] = values["close_price"] > 0
    result["age_hours"] = (as_of - values["createTime"]) / HOUR_MS
    result["holding_hours"] = (values["updateTime"] - values["createTime"]) / HOUR_MS
    result["entry_day"] = datetime.fromtimestamp(float(values["createTime"] / 1000), TZ).date().isoformat()
    if result["closed"]:
        quantity = abs(values["position_amt"])
        gross = (values["close_price"] - values["price"]) * values["position_amt"]
        fees = quantity * (values["price"] * values["open_fee_rate"] +
                           values["close_price"] * values["close_fee_rate"])
        result.update(gross=gross, fees=fees, net=gross-fees,
                      margin_return_pct=(gross-fees) / values["usdt"] * 100,
                      pnl_difference=values["close_profit"] - (gross-fees))
    return result


def metrics(rows):
    closed = sorted((row for row in rows if row["closed"]), key=lambda row: (row["updateTime"], int(row["id"])))
    opened = [row for row in rows if not row["closed"]]
    n = len(closed)
    wins = [row for row in closed if row["net"] > EPS]
    losses = [row for row in closed if row["net"] < -EPS]
    positive = sum((row["net"] for row in wins), ZERO)
    negative = -sum((row["net"] for row in losses), ZERO)
    total = lambda field: sum((row[field] for row in closed), ZERO)
    equity = peak = drawdown = ZERO
    streak = longest = 0
    for row in closed:
        equity += row["net"]
        peak = max(peak, equity)
        drawdown = max(drawdown, peak-equity)
        streak = streak+1 if row["net"] < -EPS else 0
        longest = max(longest, streak)
    symbols = defaultdict(lambda: ZERO)
    for row in closed:
        symbols[row["symbol"]] += row["net"]
    best = max(symbols, key=symbols.get) if symbols else None
    return {
        "total": len(rows), "closed": n, "open": len(opened),
        "wins": len(wins), "losses": len(losses), "breakeven": n-len(wins)-len(losses),
        "win_rate_pct": Decimal(len(wins))*100/n if n else None,
        "gross_pnl": total("gross"), "fees": total("fees"), "net_pnl": total("net"),
        "positive_net_sum": positive, "absolute_negative_net_sum": negative,
        "net_profit_factor": positive/negative if negative else None,
        "expectancy_usdt": total("net")/n if n else None,
        "mean_margin_return_pct": total("margin_return_pct")/n if n else None,
        "median_margin_return_pct": quantile([r["margin_return_pct"] for r in closed], .5),
        "median_holding_hours": quantile([r["holding_hours"] for r in closed], .5),
        "p90_holding_hours": quantile([r["holding_hours"] for r in closed], .9),
        "max_closed_drawdown_usdt": drawdown, "max_loss_streak": longest,
        "closed_symbol_count": len(symbols), "best_symbol": best,
        "net_without_best_symbol": total("net")-symbols[best] if best else None,
        "net_without_best_trade": total("net")-max(r["net"] for r in closed) if n else None,
        "oldest_open_hours": max((r["age_hours"] for r in opened), default=None),
        "median_open_hours": quantile([r["age_hours"] for r in opened], .5),
        "p90_open_hours": quantile([r["age_hours"] for r in opened], .9),
        "zero_fee_closed": sum(r["open_fee_rate"] == r["close_fee_rate"] == 0 for r in closed),
        "stored_pnl_sum": total("close_profit"), "stored_minus_recomputed": total("pnl_difference"),
        "pnl_mismatch_ids": [r["id"] for r in closed if abs(r["pnl_difference"]) > ROUND_TOLERANCE],
    }


def grouped(rows, key):
    buckets = defaultdict(list)
    for row in rows:
        buckets[key(row)].append(row)
    return [{"key": label, **metrics(group)} for label, group in sorted(buckets.items())]


def analyze(snapshot, summary=False):
    output = {"accounting": "fee-adjusted simulated net PnL; no funding or extra exit slippage", "databases": {}}
    for database, capture in snapshot["databases"].items():
        as_of = Decimal(str(capture["as_of_ms"]))
        valid, invalid, seen = [], [], set()
        for row in capture["rows"]:
            if row["id"] in seen:
                raise ValueError(f"{database}: duplicate ID {row['id']}")
            seen.add(row["id"])
            try:
                valid.append(prepare(row, as_of))
            except (InvalidOperation, ValueError, KeyError, OverflowError) as exc:
                invalid.append({"id": row.get("id"), "error": str(exc)})
        versions = defaultdict(list)
        for row in valid:
            key = (str(row.get("strategy_template_id", 0)), str(row.get("strategy_snapshot_hash") or "missing"))
            versions[key].append(row)
        cohorts = []
        for (template_id, version_hash), rows in sorted(versions.items()):
            cohort = {"template_id": template_id, "snapshot_hash": version_hash, **metrics(rows)}
            if not summary:
                for label, key in {
                    "side": lambda r: r["position_side"],
                    "symbol": lambda r: r["symbol"],
                    "entry_day_shanghai": lambda r: r["entry_day"],
                    "fees": lambda r: str(r["open_fee_rate"])+"/"+str(r["close_fee_rate"]),
                    "settings_stake_leverage_profit_loss": lambda r: "/".join(str(r[k]) for k in ("usdt", "leverage", "profit", "loss")),
                    "entry_rule_hash": lambda r: str(r.get("open_strategy_hash") or "unknown"),
                }.items():
                    cohort["by_"+label] = grouped(rows, key)
                cohort["by_exit_rule"] = grouped([r for r in rows if r["closed"]],
                    lambda r: str(r.get("close_strategy_type") or "unknown")+"/"+str(r.get("close_strategy_hash") or "unknown"))
            cohorts.append(cohort)
        output["databases"][database] = {"as_of_ms": as_of, "raw_count": len(capture["rows"]),
            "valid_count": len(valid), "invalid_rows": invalid, "versions": cohorts}
    return output


def encode(value):
    if isinstance(value, Decimal):
        return float(round(value, 8))
    raise TypeError(type(value).__name__)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("snapshot", type=Path)
    parser.add_argument("--summary", action="store_true", help="omit nested breakdowns")
    args = parser.parse_args()
    with args.snapshot.open(encoding="utf-8") as handle:
        snapshot = json.load(handle)
    print(json.dumps(analyze(snapshot, args.summary), default=encode, ensure_ascii=False, indent=2, allow_nan=False))


if __name__ == "__main__":
    main()
