const data = [
  {
    symbol: 'BTCUSDT', // 交易对
    initialMargin: '0', // 当前所需起始保证金(基于最新标记价格)
    maintMargin: '0', //维持保证金
    unrealizedProfit: '0.00000000', // 持仓未实现盈亏
    positionInitialMargin: '0', // 持仓所需起始保证金(基于最新标记价格)
    openOrderInitialMargin: '0', // 当前挂单所需起始保证金(基于最新标记价格)
    leverage: '100', // 杠杆倍率
    isolated: true, // 是否是逐仓模式
    entryPrice: '0.00000', // 持仓成本价
    maxNotional: '250000', // 当前杠杆下用户可用的最大名义价值
    positionSide: 'BOTH', // 持仓方向
    positionAmt: '0', // 持仓数量
    updateTime: 0, // 更新时间(订单交易成功时的时间，毫秒时间戳)
  },
  {
    symbol: 'XTZUSDT',
    positionAmt: '23.5', // 持仓数量
    entryPrice: '5.09', // 买入价格
    markPrice: '5.07920469',
    unRealizedProfit: '-0.25368978', // 当前收益
    liquidationPrice: '0.45161396',
    leverage: '10',
    maxNotionalValue: '1000000',
    marginType: 'cross',
    isolatedMargin: '0.00000000',
    isAutoAddMargin: 'false',
    positionSide: 'LONG',
    notional: '119.36131021', // 当前持仓usdt，原始等于 notional - unRealizedProfit
    isolatedWallet: '0',
    updateTime: 1638199621922,
  },
  {
    symbol: 'XTZUSDT',
    positionAmt: '0.0',
    entryPrice: '0.0',
    markPrice: '5.07920469',
    unRealizedProfit: '0.00000000',
    liquidationPrice: '0',
    leverage: '10',
    maxNotionalValue: '1000000',
    marginType: 'cross',
    isolatedMargin: '0.00000000',
    isAutoAddMargin: 'false',
    positionSide: 'SHORT',
    notional: '0',
    isolatedWallet: '0',
    updateTime: 0,
  },
]
