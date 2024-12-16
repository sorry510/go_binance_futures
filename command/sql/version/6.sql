ALTER TABLE futures_orders ADD average_price VARCHAR DEFAULT ('0');
ALTER TABLE futures_orders ADD stop_price VARCHAR DEFAULT ('0');
ALTER TABLE futures_orders ADD commission_asset VARCHAR DEFAULT ('USDT');
ALTER TABLE futures_orders ADD commission VARCHAR DEFAULT ('0');
ALTER TABLE futures_orders ADD realized_pnl VARCHAR DEFAULT ('0');
