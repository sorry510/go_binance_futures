# Protocol

Align Binance USD-M and Bybit linear 1h bars. d = Bybit hourly log return minus Binance hourly log return. 720h causal z-score, first |z|>=3, re-arm |z|<1. Trade Binance in sign(d) direction at next 1h open. No threshold/window/reversal search.
