const data = {
  clientOrderId: 'testOrder', // 用户自定义的订单号
  cumQty: '0',
  cumQuote: '0', // 成交金额
  executedQty: '0', // 成交量
  orderId: 22542179, // 系统订单号
  avgPrice: '0.00000', // 平均成交价
  origQty: '10', // 原始委托数量
  price: '0', // 委托价格
  reduceOnly: false, // 仅减仓
  side: 'SELL', // 买卖方向
  positionSide: 'SHORT', // 持仓方向
  status: 'NEW', // 订单状态
  stopPrice: '0', // 触发价，对`TRAILING_STOP_MARKET`无效
  closePosition: false, // 是否条件全平仓
  symbol: 'BTCUSDT', // 交易对
  timeInForce: 'GTC', // 有效方法
  type: 'TRAILING_STOP_MARKET', // 订单类型
  origType: 'TRAILING_STOP_MARKET', // 触发前订单类型
  activatePrice: '9020', // 跟踪止损激活价格, 仅`TRAILING_STOP_MARKET` 订单返回此字段
  priceRate: '0.3', // 跟踪止损回调比例, 仅`TRAILING_STOP_MARKET` 订单返回此字段
  updateTime: 1566818724722, // 更新时间
  workingType: 'CONTRACT_PRICE', // 条件价格触发类型
  priceProtect: false, // 是否开启条件单触发保护
}
