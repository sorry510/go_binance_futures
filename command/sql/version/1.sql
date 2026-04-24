ALTER TABLE `symbols` ADD CONSTRAINT `uk_symbols_symbol` UNIQUE (`symbol`);
ALTER TABLE `spot_symbols` ADD CONSTRAINT `uk_spot_symbols_symbol` UNIQUE (`symbol`);