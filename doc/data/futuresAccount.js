// https://binance-docs.github.io/apidocs/futures/cn/#v2-user_data-2
const data = {
  "feeTier": 0,  // 手续费等级
  "canTrade": true,  // 是否可以交易
  "canDeposit": true,  // 是否可以入金
  "canWithdraw": true, // 是否可以出金
  "updateTime": 0,
  "totalInitialMargin": "0.00000000",  // 但前所需起始保证金总额(存在逐仓请忽略), 仅计算usdt资产
  "totalMaintMargin": "0.00000000",  // 维持保证金总额, 仅计算usdt资产
  "totalWalletBalance": "23.72469206",   // 账户总余额, 仅计算usdt资产
  "totalUnrealizedProfit": "0.00000000",  // 持仓未实现盈亏总额, 仅计算usdt资产
  "totalMarginBalance": "23.72469206",  // 保证金总余额, 仅计算usdt资产
  "totalPositionInitialMargin": "0.00000000",  // 持仓所需起始保证金(基于最新标记价格), 仅计算usdt资产
  "totalOpenOrderInitialMargin": "0.00000000",  // 当前挂单所需起始保证金(基于最新标记价格), 仅计算usdt资产
  "totalCrossWalletBalance": "23.72469206",  // 全仓账户余额, 仅计算usdt资产
  "totalCrossUnPnl": "0.00000000",    // 全仓持仓未实现盈亏总额, 仅计算usdt资产
  "availableBalance": "23.72469206",       // 可用余额, 仅计算usdt资产
  "maxWithdrawAmount": "23.72469206",     // 最大可转出余额, 仅计算usdt资产
  "assets": [ // 资产
      {
          "asset": "USDT",        //资产
          "walletBalance": "23.72469206",  //余额
          "unrealizedProfit": "0.00000000",  // 未实现盈亏
          "marginBalance": "23.72469206",  // 保证金余额
          "maintMargin": "0.00000000",    // 维持保证金
          "initialMargin": "0.00000000",  // 当前所需起始保证金
          "positionInitialMargin": "0.00000000",  // 持仓所需起始保证金(基于最新标记价格)
          "openOrderInitialMargin": "0.00000000", // 当前挂单所需起始保证金(基于最新标记价格)
          "crossWalletBalance": "23.72469206",  //全仓账户余额
          "crossUnPnl": "0.00000000" // 全仓持仓未实现盈亏
          "availableBalance": "23.72469206",       // 可用余额
          "maxWithdrawAmount": "23.72469206",     // 最大可转出余额
          "marginAvailable": true,   // 是否可用作联合保证金
          "updateTime": 1625474304765  //更新时间
      },
      {
          "asset": "BUSD",        //资产
          "walletBalance": "103.12345678",  //余额
          "unrealizedProfit": "0.00000000",  // 未实现盈亏
          "marginBalance": "103.12345678",  // 保证金余额
          "maintMargin": "0.00000000",    // 维持保证金
          "initialMargin": "0.00000000",  // 当前所需起始保证金
          "positionInitialMargin": "0.00000000",  // 持仓所需起始保证金(基于最新标记价格)
          "openOrderInitialMargin": "0.00000000", // 当前挂单所需起始保证金(基于最新标记价格)
          "crossWalletBalance": "103.12345678",  //全仓账户余额
          "crossUnPnl": "0.00000000" // 全仓持仓未实现盈亏
          "availableBalance": "103.12345678",       // 可用余额
          "maxWithdrawAmount": "103.12345678",     // 最大可转出余额
          "marginAvailable": true,   // 否可用作联合保证金
          "updateTime": 0  // 更新时间
         }
  ],
  "positions": [  // 头寸，将返回所有市场symbol。
      //根据用户持仓模式展示持仓方向，即单向模式下只返回BOTH持仓情况，双向模式下只返回 LONG 和 SHORT 持仓情况
      {
          "symbol": "BTCUSDT",  // 交易对
          "initialMargin": "0",   // 当前所需起始保证金(基于最新标记价格)
          "maintMargin": "0", //维持保证金
          "unrealizedProfit": "0.00000000",  // 持仓未实现盈亏
          "positionInitialMargin": "0",  // 持仓所需起始保证金(基于最新标记价格)
          "openOrderInitialMargin": "0",  // 当前挂单所需起始保证金(基于最新标记价格)
          "leverage": "100",  // 杠杆倍率
          "isolated": true,  // 是否是逐仓模式
          "entryPrice": "0.00000",  // 持仓成本价
          "maxNotional": "250000",  // 当前杠杆下用户可用的最大名义价值
          "positionSide": "BOTH",  // 持仓方向
          "positionAmt": "0",      // 持仓数量
          "updateTime": 0         // 更新时间 
      }
  ]
}