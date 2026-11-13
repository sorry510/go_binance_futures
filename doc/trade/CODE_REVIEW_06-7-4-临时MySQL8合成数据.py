#!/usr/bin/env python3
"""Generate data ONLY for an isolated, disposable MySQL 8 verification database.

No database connection, file mutation, network access or credentials.
Never pipe the generated INSERT statements into a real user/production schema.
"""
import sys
import hashlib


def base36(value: int) -> str:
    digits = "0123456789abcdefghijklmnopqrstuvwxyz"
    result = ""
    while value:
        value, rem = divmod(value, 36)
        result = digits[rem] + result
    return result.zfill(7)


if __name__ == "__main__":
    if sys.argv[1:] != ["--isolated-test-only"]:
        raise SystemExit("requires --isolated-test-only; output contains INSERTs")
    print("START TRANSACTION;")
    for i in range(1, 501):
        symbol = "BTCUSDT" if i % 2 else "WBTCUSDT"
        base = "BTC" if i % 2 else "WBTC"
        print(
            "INSERT INTO spot_managed_positions"
            "(id,symbol,base_asset,quote_asset,bought_qty,remaining_qty,avg_cost_quote,realized_pnl,status,created_at,updated_at,closed_at)"
            f" VALUES({i},'{symbol}','{base}','USDT',0.01,0,100,0.1,'CLOSED',1760000000000,1760000000000,1760000000000);"
        )
        for side, oid, price, quote in (
            ("BUY", i * 2 - 1, "100", "1"),
            ("SELL", i * 2, "110", "1.1"),
        ):
            if side == "BUY":
                event_unix = 1760000000 + i * 60
                digest = hashlib.sha256(
                    f"spot_quant_0673_v4|{symbol}|BUY|0|1m|{event_unix}".encode()
                ).hexdigest()[:22]
                client_id = "sq4" + "01" + base36(event_unix) + digest
            else:
                digest = hashlib.sha256(
                    f"spot_quant_0674_exit_v1|{symbol}|{i}".encode()
                ).hexdigest()[:30]
                client_id = "sqe" + digest
            assert len(client_id) in (33, 34)
            print(
                "INSERT INTO spot_managed_orders"
                "(id,position_id,symbol,side,order_type,client_order_id,exchange_order_id,requested_qty,filled_qty,status,created_at,updated_at)"
                f" VALUES({oid},{i},'{symbol}','{side}','MARKET','{client_id}','{700000 + oid}',"
                "0.01,0.01,'FILLED',1760000000000,1760000000000);"
            )
            print(
                "INSERT INTO spot_trade_fills"
                "(position_id,order_id,exchange_order_id,trade_id,symbol,side,qty,price,quote_qty,commission,commission_asset,trade_time)"
                f" VALUES({i},{oid},'{700000 + oid}','{800000 + oid}',"
                f"'{symbol}','{side}',0.01,{price},{quote},0,'USDT',1760000000000);"
            )
    print("COMMIT;")
