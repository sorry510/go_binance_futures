-- 06-7-4 Spot Quant production preflight: READ ONLY, MySQL 8.
-- Execute only after the operator confirms the TARGET database and backup.
-- Every statement is SELECT or EXPLAIN (not EXPLAIN ANALYZE).
-- Do not run ./go_binance_futures sync db without a separate authorization.

SELECT DATABASE() AS selected_schema, VERSION() AS mysql_version;

-- Exactly one config row is expected; an arbitrary first row is unsafe.
SELECT IF(COUNT(*)=1 AND SUM(id=1)=1, 'PASS', 'FAIL') AS config_row_gate,
       COUNT(*) AS config_rows,
       MAX(CASE WHEN id=1 THEN version END) AS schema_version,
       MAX(CASE WHEN id=1 THEN spot_quant_enable END) AS spot_quant_global_enabled
  FROM config;

SELECT COUNT(*) AS spot_symbols_total,
       COALESCE(SUM(spot_quant_enabled=1),0) AS enabled_spot_quant_symbols
  FROM spot_symbols;

-- A missing expected table MUST produce an explicit FAIL row.
SELECT expected.table_name,
       IF(actual.table_name IS NOT NULL,'PASS','FAIL') AS table_gate
FROM (
  SELECT 'config' AS table_name UNION ALL
  SELECT 'spot_symbols' UNION ALL
  SELECT 'spot_managed_positions' UNION ALL
  SELECT 'spot_managed_orders' UNION ALL
  SELECT 'spot_trade_fills' UNION ALL
  SELECT 'spot_strategy_templates'
) AS expected
LEFT JOIN information_schema.tables AS actual
  ON actual.table_schema=DATABASE() AND actual.table_name=expected.table_name
ORDER BY expected.table_name;

-- Physical switches must exist too, regardless of schema version number.
SELECT IF(COUNT(*)=2,'PASS','FAIL') AS switch_columns_gate,COUNT(*) AS found
FROM information_schema.columns
WHERE table_schema=DATABASE()
  AND ((table_name='config' AND column_name='spot_quant_enable')
   OR (table_name='spot_symbols' AND column_name='spot_quant_enabled'));

-- Summary must be PASS only when exactly 10 known fields are DECIMAL(36,18).
SELECT IF(COUNT(*)=10 AND SUM(column_type='decimal(36,18)')=10,
          'PASS','FAIL') AS decimal_gate,
       COUNT(*) AS expected_fields_found,
       COALESCE(SUM(column_type='decimal(36,18)'),0) AS correct_decimal_fields
FROM information_schema.columns
WHERE table_schema=DATABASE()
  AND ((table_name='spot_managed_positions' AND column_name IN
           ('bought_qty','remaining_qty','avg_cost_quote','realized_pnl'))
    OR (table_name='spot_managed_orders' AND column_name IN
           ('requested_qty','filled_qty'))
    OR (table_name='spot_trade_fills' AND column_name IN
           ('qty','price','quote_qty','commission')));

-- Individual fields are retained for investigation when the summary is FAIL.
SELECT table_name,column_name,column_type
  FROM information_schema.columns
  WHERE table_schema=DATABASE()
    AND (
      (table_name='spot_managed_positions' AND column_name IN
        ('bought_qty','remaining_qty','avg_cost_quote','realized_pnl'))
      OR (table_name='spot_managed_orders' AND column_name IN
        ('requested_qty','filled_qty'))
      OR (table_name='spot_trade_fills' AND column_name IN
        ('qty','price','quote_qty','commission'))
    )
  ORDER BY table_name,column_name;

SELECT table_name,index_name,non_unique,
       GROUP_CONCAT(column_name ORDER BY seq_in_index) AS indexed_columns
  FROM information_schema.statistics
  WHERE table_schema=DATABASE()
    AND table_name IN ('spot_managed_positions','spot_managed_orders','spot_trade_fills')
  GROUP BY table_name,index_name,non_unique
  ORDER BY table_name,index_name;

SELECT (SELECT COUNT(*) FROM spot_managed_positions) AS positions_total,
       (SELECT COUNT(*) FROM spot_managed_orders) AS orders_total,
       (SELECT COUNT(*) FROM spot_trade_fills) AS fills_total;

-- Missing/overfilled FILLED orders are not safe to ignore.
-- Validate the complete v4 identity, including hash binding:
-- sq4 + 2-char interval + 7-char base36 UTC open + 22-char SHA256 prefix.
-- This mirrors Runner's spotQuantClientID(symbol,BUY,0,interval,open).
-- A syntactically valid forged ID must fail, not authorize automatic BUY.
-- No modification of order history occurs here.
SELECT
  IF(COALESCE(SUM(status='FILLED' AND filled_qty>requested_qty),0)=0,
     'PASS','FAIL') AS filled_over_qty_gate,
  IF(COALESCE(SUM(invalid_buy_id),0)=0,
     'PASS','FAIL') AS buy_client_id_gate,
  COALESCE(SUM(status='FILLED' AND filled_qty<requested_qty),0) AS filled_missing_fills,
  COALESCE(SUM(status='FILLED' AND filled_qty>requested_qty),0) AS filled_over_qty,
  COALESCE(SUM(status IN ('NEW','PARTIALLY_FILLED','PENDING_CANCEL')),0) AS awaiting_recovery,
  COALESCE(SUM(invalid_buy_id),0) AS invalid_or_legacy_buy_ids
FROM (
 SELECT parsed.*,
   CASE WHEN side='BUY' AND NOT (
      COALESCE(valid_shape,0)=1
      AND SUBSTRING(client_order_id,13,22)=LEFT(SHA2(CONCAT(
          'spot_quant_0673_v4|',symbol,'|BUY|0|',
          kline_interval,'|',event_unix),256),22)
   ) THEN 1 ELSE 0 END AS invalid_buy_id
 FROM (
   SELECT symbol,side,client_order_id,status,requested_qty,filled_qty,
      CAST(CONV(SUBSTRING(client_order_id,6,7),36,10) AS UNSIGNED) AS event_unix,
      (REGEXP_LIKE(client_order_id,
       '^sq4(01|03|05|15|30|H1|H2|H4|H6|H8|HC|D1|D3|W1|M1)[0-9a-z]{7}[0-9a-f]{22}$','c')
       AND CAST(CONV(SUBSTRING(client_order_id,6,7),36,10) AS UNSIGNED)>0
      ) AS valid_shape,
      CASE SUBSTRING(client_order_id,4,2)
        WHEN '01' THEN '1m' WHEN '03' THEN '3m'
        WHEN '05' THEN '5m' WHEN '15' THEN '15m' WHEN '30' THEN '30m'
        WHEN 'H1' THEN '1h' WHEN 'H2' THEN '2h' WHEN 'H4' THEN '4h'
        WHEN 'H6' THEN '6h' WHEN 'H8' THEN '8h' WHEN 'HC' THEN '12h'
        WHEN 'D1' THEN '1d' WHEN 'D3' THEN '3d'
        WHEN 'W1' THEN '1w' WHEN 'M1' THEN '1M'
      END AS kline_interval
   FROM spot_managed_orders
 ) AS parsed
) AS checked;

-- EXPLAIN doesn't execute the query, unlike EXPLAIN ANALYZE.
EXPLAIN SELECT id,position_id,client_order_id
  FROM spot_managed_orders WHERE symbol='BTCUSDT' AND side='BUY' ORDER BY id DESC;
EXPLAIN SELECT id,position_id,status FROM spot_managed_orders
  WHERE symbol='BTCUSDT'
    AND (status IN ('NEW','PARTIALLY_FILLED','PENDING_CANCEL')
      OR (status='FILLED' AND filled_qty<>requested_qty)) ORDER BY id;
EXPLAIN SELECT id,symbol,status FROM spot_managed_positions
  WHERE symbol='BTCUSDT' ORDER BY id DESC LIMIT 100;
EXPLAIN SELECT id,order_id,trade_id FROM spot_trade_fills
  WHERE position_id=1 ORDER BY id;

-- IMPORTANT: This script is diagnostic; successful SQL results are NOT
-- an authorization for Testnet or Mainnet order placement.
