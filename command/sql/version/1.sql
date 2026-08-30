-- Use CREATE UNIQUE INDEX instead of ALTER TABLE ADD CONSTRAINT,
-- because SQLite does not support adding constraints to an existing table.
CREATE UNIQUE INDEX `uk_symbols_symbol` ON `symbols` (`symbol`);
CREATE UNIQUE INDEX `uk_spot_symbols_symbol` ON `spot_symbols` (`symbol`);
