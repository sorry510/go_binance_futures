ALTER TABLE config
ADD COLUMN future_test_fee_rate DECIMAL(12,8) NOT NULL DEFAULT 0.0005;

ALTER TABLE test_strategy_results
ADD COLUMN open_fee_rate DECIMAL(12,8) NOT NULL DEFAULT 0;

ALTER TABLE test_strategy_results
ADD COLUMN close_fee_rate DECIMAL(12,8) NOT NULL DEFAULT 0;
