INSERT INTO strategy_templates (name,technology,strategy,createTime,updateTime) VALUES
	 ('templete1','{"ma":[],"ema":[{"name":"ema_4h_3","kline_interval":"4h","period":3,"enable":true},{"name":"ema_4h_7","kline_interval":"4h","period":7,"enable":true}],"rsi":[{"name":"rsi_4h_14","kline_interval":"4h","period":14,"enable":true}],"kc":[],"boll":[],"atr":[]}','[{"name":"custom_long1","type":"close_long","code":"let rule1 = ema_4h_3.Data[0] > ema_4h_7.Data[0] && ema_4h_3.Data[1] < ema_4h_7.Data[1];\nlet rule2 = rsi_4h_14.Data[0] < 20;\n\nrule1 && rule2","fullScreen":false,"enable":true},{"name":"custom_short1","type":"close_short","code":"rsi_4h_14.Data[0] > 90","fullScreen":false,"enable":true}]',0,0),
	 ('kc 边轨','{"ma":[],"ema":[{"name":"ema_12h_14","kline_interval":"12h","period":14,"enable":true}],"rsi":[{"name":"rsi_4h_14","kline_interval":"4h","period":14,"enable":true},{"name":"rsi_1d_14","kline_interval":"1d","period":14,"enable":true}],"kc":[{"name":"kc_4h_50_2","kline_interval":"4h","period":50,"multiplier":2.75,"enable":false},{"name":"kc_4h_50_3","kline_interval":"4h","period":50,"multiplier":3.75,"enable":false},{"name":"kc_1d_50_2","kline_interval":"1d","period":50,"multiplier":2.75,"enable":true},{"name":"kc_1d_50_3","kline_interval":"1d","period":50,"multiplier":3.75,"enable":true}],"boll":[],"atr":[]}','[{"name":"kc_long_from_low","type":"long","code":"// 当前价格 > 窄通道(低轨) && 上个k线价格 < 窄通道(低轨), 含义就是价格反弹突破了窄通道的低轨\nlet rule1 = kline_4h.Close[0] > kc_1d_50_2.Low[0] && kline_4h.Close[1] < kc_1d_50_2.Low[1];\n\n// 在之前k线的 2-12 条记录中可以找到，最低价低于过宽通道(低轨)\nlet rule2 = any((2..12), kline_4h.Low[#] < kc_1d_50_3.Low[#]);\n\n// 在 12h 的k线中最近 2 条线上涨的，意思是大级别也是看涨\nlet rule3 = IsDesc(kline_12h.Close[0:2]);\n\nrule1 && rule2 && rule3\n","fullScreen":false,"enable":true},{"name":"kc_short_form_high","type":"short","code":"// 当前价格 < 窄通道(高轨) && 上个k线价格 > 窄通道(高轨), 含义就是价格反弹跌破了窄通道的高轨\nlet rule1 = kline_4h.Close[0] < kc_1d_50_2.High[0] && kline_4h.Close[1] > kc_1d_50_2.High[1];\n\n// 在之前k线的 2-12 条记录中可以找到，最高价 > 过宽通道(高轨)\nlet rule2 = any((2..12), kline_4h.High[#] < kc_1d_50_3.High[#]);\n\n// 在 12h 的k线中最近 2 条线下跌的，意思是大级别也是看跌\nlet rule3 = IsAsc(kline_12h.Close[0:2]);\n\nrule1 && rule2 && rule3\n","fullScreen":false,"enable":true},{"name":"rsi_short","type":"short","code":"rsi_4h_14.Data[0] > 90 && rsi_1d_14.Data[0] > 82","fullScreen":false,"enable":false}]',0,0),
	 ('kc 中轨','{"ma":[{"name":"ma_5m_14","kline_interval":"5m","period":14,"enable":true}],"ema":[],"rsi":[{"name":"rsi_4h_14","kline_interval":"4h","period":14,"enable":true},{"name":"rsi_1d_14","kline_interval":"1d","period":14,"enable":true}],"kc":[{"name":"kc_4h_50_2","kline_interval":"4h","period":50,"multiplier":2.75,"enable":true},{"name":"kc_4h_50_3","kline_interval":"4h","period":50,"multiplier":3.75,"enable":true}],"boll":[{"name":"boll_1h_144_5","kline_interval":"1h","period":144,"std_dev_multiplier":0.5,"enable":true},{"name":"boll_1h_144_8","kline_interval":"1h","period":144,"std_dev_multiplier":0.85,"enable":true}],"atr":[{"name":"atr_1h_5","kline_interval":"1h","period":5,"enable":true}]}','[{"name":"kc_long_mid","type":"long","code":"// 12个k线内最低价跌破过宽通道中轨，然后最新价格刚反弹突破窄通道中轨\nlet rule1 = any((2..12), kline_4h.Low[#] < kc_4h_50_3.Mid[#]) && kline_4h.Close[1] < kc_4h_50_2.Mid[1] && kline_4h.Close[0] > kc_4h_50_2.Mid[0];\n\n// 放量上涨\nlet rule2 = kline_4h.Qps[0] > kline_4h.Qps[1];\n\nrule1 && rule2","fullScreen":false,"enable":true},{"name":"close_long_profit","type":"close_long","code":"// 盈利率 > 10%\nlet rule1 = ROI >= 10;\n\n// 当前价格 < 上个k线的收盘价\nlet rule2 = kline_5m.Close[0] < kline_5m.Close[1];\n\n\nrule1 && rule2","fullScreen":false,"enable":true},{"name":"close_long_loss","type":"close_long","code":"// 当前标记价格 < 买入价格 - 3 * atr, 进行止损\nlet rule1 = float(Position.MarkPrice) < float(Position.EntryPrice) - atr_1h_5.Data[0] * 3;\n\n// boll 宽度\nlet bbw = (boll_1h_144_8.High[0] - boll_1h_144_8.Low[0]) / boll_1h_144_8.Mid[0] * 100;\n\n// 之前价格在bool0.5上面，当前价格 < bool0.5 的上轨，boll0.85宽度 >= 3, 进行止损\nlet rule2 = kline_1h.Close[0] < boll_1h_144_5.High[0] && bbw >= 3;\n\n\nrule1 || rule2","fullScreen":false,"enable":true}]',0,0),
	 ('rsi 超卖','{"ma":[],"ema":[],"rsi":[{"name":"rsi_4h_14","kline_interval":"4h","period":14,"enable":true},{"name":"rsi_1d_14","kline_interval":"1d","period":14,"enable":true}],"kc":[],"boll":[],"atr":[]}','[{"name":"rsi_long","type":"long","code":"rsi_4h_14.Data[0] <= 20 || rsi_1d_14.Data[0] <= 20","fullScreen":false,"enable":true},{"name":"rsi_short","type":"short","code":"rsi_4h_14.Data[0] >= 90 || rsi_1d_14.Data[0] >= 90","fullScreen":false,"enable":true}]',0,0),
	 ('boll 1h','{"ma":[{"name":"ma_3m_14","kline_interval":"3m","period":14,"enable":true}],"ema":[{"name":"ema_1h_5","kline_interval":"1h","period":5,"enable":true},{"name":"ema_1h_12","kline_interval":"1h","period":12,"enable":true}],"rsi":[{"name":"rsi_1h_14","kline_interval":"1h","period":14,"enable":true}],"kc":[],"boll":[{"name":"boll_1h_144_5","kline_interval":"1h","period":144,"std_dev_multiplier":0.5,"enable":true},{"name":"boll_1h_144_8","kline_interval":"1h","period":144,"std_dev_multiplier":0.85,"enable":true}],"atr":[{"name":"atr_1h_5","kline_interval":"1h","period":5,"enable":true},{"name":"atr_1h_12","kline_interval":"1h","period":12,"enable":true}]}','[{"name":"bool1_long","type":"long","code":"// 当前收盘价 > EMA5, EMA5 >大于EMA12, 当前收盘价 > 10个小时内的前高点\nlet rule1 = kline_1h.Close[0] > ema_1h_5.Data[0] && ema_1h_5.Data[0] > ema_1h_12.Data[0] && all(kline_1h.High[1:8], # <= kline_1h.Close[0]);\n\n// 前一个EMA5 < 0.85布林带的上轨, 当前EMA5>0.85布林带的上轨\n// let rule2 = ema_1h_5.Data[1] < boll_1h_144_8.High[1] && ema_1h_5.Data[0] > boll_1h_144_8.High[0];\n\n// 前一个收盘价 < 0.85布林带的上轨, 当前EMA5>0.85布林带的上轨\nlet rule2 = kline_1h.Close[1] < boll_1h_144_8.High[1] && kline_1h.Close[0] > boll_1h_144_8.High[0];\n\n// 基本盘不能跌超过 2%, 系统启动后1小时才运行策略\nlet rule3 = BasicTrend > -2 && (NowTime - SystemStartTime) / 1000 > 600;\n\n// 增量上涨, 没有超买\n// let rule4 = kline_1h.Qps[0] > kline_1h.Qps[1] && rsi_1h_14.Data[0] < 80;\n\nrule1 && rule2 && rule3\n","fullScreen":false,"enable":true},{"name":"bool1_long_close_loss","type":"close_long","code":"// 当前标记价格 < 买入价格 - 3 * atr, 进行止损\nlet rule1 = float(Position.MarkPrice) < float(Position.EntryPrice) - atr_1h_5.Data[0] * 3;\n\n// boll 宽度\nlet bbw = (boll_1h_144_8.High[0] - boll_1h_144_8.Low[0]) / boll_1h_144_8.Mid[0] * 100;\n\n// 之前价格在bool0.5上面，当前价格 < bool0.5 的上轨，boll0.85宽度 >= 3, 进行止损\nlet rule2 = kline_1h.Close[0] < boll_1h_144_5.High[0] && bbw >= 3;\n\n\nrule1 || rule2","fullScreen":false,"enable":true},{"name":"bool1_long_close_profit","type":"close_long","code":"// 盈利率 > 10%\n// let rule1 = ROI >= 10;\n\n// 当前价格 < 上个k线的收盘价\nlet rule2 = kline_3m.Close[0] < kline_3m.Close[1];\n\nlet rule3 = Position.MarkPrice > Position.EntryPrice + atr_1h_5.Data[0] * 2;\n\nrule2 && rule3","fullScreen":false,"enable":true},{"name":"bool1_short","type":"short","code":"// 当前收盘价 < EMA5, EMA5 < EMA12, 当前收盘价 < 8个小时内的前低点\nlet rule1 = kline_1h.Close[0] < ema_1h_5.Data[0] && ema_1h_5.Data[0] < ema_1h_12.Data[0] && all(kline_1h.Low[1:8], # >= kline_1h.Close[0]);\n\n// 前一个收盘价 > 0.85布林带的下轨, 当前EMA5<0.85布林带的下轨\nlet rule2 = kline_1h.Close[1] > boll_1h_144_8.Low[1] && kline_1h.Close[0] < boll_1h_144_8.Low[0];\n\n// 基本盘不能涨超过 -1%, 系统启动后10min才运行策略\nlet rule3 = BasicTrend < -1 && (NowTime - SystemStartTime) / 1000 > 600;\n\n// 增量下跌, 没有超卖\n// let rule4 = kline_1h.Qps[0] > kline_1h.Qps[1] && rsi_1h_14.Data[0] > 30;\n\nrule1 && rule2 && rule3\n","fullScreen":false,"enable":true},{"name":"bool1_short_close_loss","type":"close_short","code":"// 当前标记价格 > 买入价格 - 3 * atr, 进行止损\nlet rule1 = float(Position.MarkPrice) > (float(Position.EntryPrice) + atr_1h_5.Data[0] * 3);\n\n// boll 宽度\nlet bbw = (boll_1h_144_8.High[0] - boll_1h_144_8.Low[0]) / boll_1h_144_8.Mid[0] * 100;\n\n// 之前价格在bool0.5上面，当前价格 > bool0.5 的上轨，boll0.85宽度 >= 3, 进行止损\nlet rule2 = kline_1h.Close[0] > boll_1h_144_5.Low[0] && bbw >= 3;\n\n\nrule1 || rule2","fullScreen":false,"enable":true},{"name":"bool1_short_close_profit","type":"close_short","code":"// 盈利率 > 10%\n// let rule1 = ROI >= 10;\n\n// 当前价格 < 上个k线的收盘价\nlet rule2 = kline_3m.Close[0] > kline_3m.Close[1];\n\nlet rule3 = Position.MarkPrice < Position.EntryPrice - atr_1h_5.Data[0] * 2;\n\n\nrule2 && rule3","fullScreen":false,"enable":true}]',0,0),
	 ('kc 1h(适用于止跌回涨时)','{"ma":[{"name":"ma_5m_14","kline_interval":"5m","period":14,"enable":true},{"name":"ma_4h_14","kline_interval":"4h","period":14,"enable":true},{"name":"ma_3m_14","kline_interval":"3m","period":14,"enable":true}],"ema":[],"rsi":[{"name":"rsi_1h_14","kline_interval":"1h","period":14,"enable":true}],"kc":[{"name":"kc_1h_50_2","kline_interval":"1h","period":50,"multiplier":2.75,"enable":true},{"name":"kc_1h_50_3","kline_interval":"1h","period":50,"multiplier":3.75,"enable":true}],"boll":[{"name":"boll_1h_144_5","kline_interval":"1h","period":144,"std_dev_multiplier":0.5,"enable":false},{"name":"boll_1h_144_8","kline_interval":"1h","period":144,"std_dev_multiplier":0.85,"enable":false}],"atr":[{"name":"atr_1h_5","kline_interval":"1h","period":5,"enable":true}]}','[{"name":"long_open","type":"long","code":"// 12个k线内最低价跌破过宽通道低轨，然后最新价格刚反弹突破窄通道低轨\n// let rule1 = one((2..12), kline_1h.Low[#] < kc_1h_50_3.Low[#]) && kline_1h.Close[1] < kc_1h_50_2.Low[1] && kline_1h.Close[0] > kc_1h_50_2.Low[0];\n\n// 24个k线内跌破过窄通道的低轨，最新价格刚反弹突破窄通道中轨\nlet rule1 = any((2..24), kline_1h.Low[#] < kc_1h_50_2.Low[#]) && kline_1h.Close[1] < kc_1h_50_2.Mid[1] && kline_1h.Close[0] > kc_1h_50_2.Mid[0];\n\n// 在大级别的k线中最近 2 条线上涨的，意思是大级别也是看涨\nlet rule2 = kline_4h.Close[1] > kline_4h.Open[1] && kline_1h.Close[0] > kline_4h.Open[0];\n\n// 基本盘不能跌超过 2%, 系统启动后1小时才运行策略\nlet rule3 = BasicTrend >= 0 && (NowTime - SystemStartTime) / 1000 > 600;\n\n// 增量上涨, 没有超买\nlet rule4 = kline_1h.Qps[0] > kline_1h.Qps[1] && rsi_1h_14.Data[0] < 80;\n\n\nrule1 && rule2 && rule3 && rule4\n","fullScreen":false,"enable":true},{"name":"short_open","type":"short","code":"// 12个k线内最高价突破过宽通道高轨，然后最新价格刚跌破窄通道高轨\nlet rule1 = any((2..24), kline_1h.High[#] > kc_1h_50_3.High[#]) && kline_1h.Close[1] > kc_1h_50_2.High[1] && kline_1h.Close[0] < kc_1h_50_2.High[0];\n\n// 在大级别的k线中最近 2 条线下跌的，意思是大级别也是看跌\nlet rule2 = kline_4h.Close[1] < kline_4h.Open[1] && kline_1h.Close[0] < kline_4h.Open[0];\n\n// 基本盘不能涨超过 -1%, 系统启动后10min才运行策略\nlet rule3 = BasicTrend < 0 && (NowTime - SystemStartTime) / 1000 > 600;\n\n// 增量下跌, 没有超卖\nlet rule4 = kline_1h.Qps[0] > kline_1h.Qps[1] && rsi_1h_14.Data[0] > 30;\n\nrule1 && rule2 && rule3 && rule4\n","fullScreen":false,"enable":true},{"name":"long_close","type":"close_long","code":"// 盈利率 > 10% and 当前处于下跌期\nlet is_profit = (ROI >= 10) && float(Position.MarkPrice) > (float(Position.EntryPrice) + atr_1h_5.Data[0] * 2.2) && (kline_3m.Close[0] < kline_3m.Close[1]);\n\n// 当前标记价格 < 买入价格 - 3 * atr, 进行止损\nlet is_loss = (ROI <= -10) && float(Position.MarkPrice) < (float(Position.EntryPrice) - atr_1h_5.Data[0] * 2.5) && (kline_3m.Close[0] > kline_3m.Close[1]);\n\nis_profit || is_loss || (ROI < -20)\n\n\n// let is_profit = ROI >= 10 && kline_3m.Close[0] < kline_3m.Close[1];\n// let is_loss = ROI <= -10 && kline_3m.Close[0] > kline_3m.Close[1];\n// is_profit || is_loss","fullScreen":false,"enable":true},{"name":"short_close","type":"close_short","code":"// 盈利率 > 10%，当前处于上涨期\nlet is_porfit = (ROI >= 10) && float(Position.MarkPrice) < (float(Position.EntryPrice) - atr_1h_5.Data[0] * 2.2) && kline_3m.Close[0] > kline_3m.Close[1];\n\n// 当前标记价格 > 买入价格 + 3 * atr, 进行止损\nlet is_loss =  (ROI <= -10) && float(Position.MarkPrice) > (float(Position.EntryPrice) + atr_1h_5.Data[0] * 2.5) && kline_3m.Close[0] < kline_3m.Close[1];\n\n\nis_porfit || is_loss || (ROI < -20)\n\n\n//let is_profit = ROI >= 10 && kline_3m.Close[0] > kline_3m.Close[1];\n//let is_loss = ROI <= -10 && kline_3m.Close[0] < kline_3m.Close[1];\n\n//is_profit || is_loss","fullScreen":false,"enable":true}]',0,0);
