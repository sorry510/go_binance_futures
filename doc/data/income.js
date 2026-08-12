// incomeType 收益类型： TRANSFER 转账, WELCOME_BONUS 欢迎奖金, REALIZED_PNL 已实现盈亏, FUNDING_FEE 资金费用, COMMISSION 佣金, INSURANCE_CLEAR 强平, REFERRAL_KICKBACK 推荐人返佣, COMMISSION_REBATE 被推荐人返佣, API_REBATE API佣金回扣, CONTEST_REWARD 交易大赛奖金, CROSS_COLLATERAL_TRANSFER cc转账, OPTIONS_PREMIUM_FEE 期权购置手续费, OPTIONS_SETTLE_PROFIT 期权行权收益, INTERNAL_TRANSFER 内部账户，给普通用户划转, AUTO_EXCHANGE 自动兑换, DELIVERED_SETTELMENT 下架结算, COIN_SWAP_DEPOSIT 闪兑转入, COIN_SWAP_WITHDRAW 闪兑转出, POSITION_LIMIT_INCREASE_FEE 仓位限制上调费用

const data = [
	{
    	"symbol": "", // 交易对，仅针对涉及交易对的资金流
    	"incomeType": "TRANSFER",	// 资金流类型
    	"income": "-0.37500000", // 资金流数量，正数代表流入，负数代表流出
    	"asset": "USDT", // 资产内容
    	"info":"TRANSFER", // 备注信息，取决于流水类型
    	"time": 1570608000000, // 时间
    	"tranId":"9689322392",		// 划转ID
    	"tradeId":""					// 引起流水产生的原始交易ID
	},
	{
   		"symbol": "BTCUSDT",
    	"incomeType": "COMMISSION", 
    	"income": "-0.01000000",
    	"asset": "USDT",
    	"info":"COMMISSION",
    	"time": 1570636800000,
    	"tranId":9689322392,		
    	"tradeId":"2059192"					
	}
]
