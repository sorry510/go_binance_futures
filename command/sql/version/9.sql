ALTER TABLE `config` ADD ws_futures_fast_move_enable INTEGER DEFAULT (0);
ALTER TABLE `config` ADD ws_futures_fast_move_threshold INTEGER DEFAULT (20);
ALTER TABLE `config` ADD ws_futures_fast_move_recover INTEGER DEFAULT (18);
ALTER TABLE `config` ADD ws_futures_fast_move_cooldown_sec INTEGER DEFAULT (900);
ALTER TABLE `config` ADD ws_futures_fast_move_windows TEXT DEFAULT ('3m,5m,10m');
