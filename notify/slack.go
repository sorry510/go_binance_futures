package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/beego/beego/v2/core/config"
)

var slack_token, _ = config.String("dingding::dingding_token")
var slack_word, _ = config.String("dingding::dingding_word")

type Slack struct {
}

type SlackData struct {
	Msgtype string `json:"msgtype"`
	Markdown SlackApiMarkDownData `json:"markdown"`
}

type SlackApiMarkDownData struct {
	Title string `json:"title"`
	Text string `json:"text"`
}

func SlackApi(content string) {
	// 放到单独执行，避免主进程阻塞(未知原因突然不能在 goroutine 中执行了)
	// go func () {
		url := "https://oapi.dingtalk.com/robot/send?access_token=" + dingding_token
	
		requestBody := SlackData{
			Msgtype: "markdown",
			Markdown: SlackApiMarkDownData {
				Title: dingding_word,
				Text: content,
			},
		}
		jsonData, _ := json.Marshal(requestBody)
		fmt.Println(string(jsonData))
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("Error creating request:", err)
			return
		}
		// 设置请求头，比如 Content-Type
		req.Header.Set("Content-Type", "application/json")
		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error making POST request:", err)
			return
		}
		defer resp.Body.Close()
		// 读取响应体
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		// 打印响应体
		fmt.Println(string(bodyBytes))

		// 检查响应状态码
		if resp.StatusCode != http.StatusOK {
			fmt.Println("Unexpected status code:", resp.StatusCode)
		}
	// }()
}

func (pusher Slack) BuyOrderSuccess(symbol string, quantity float64, price float64, side string) {
	text := `
## %s交易通知
#### **币种**：%s
#### **类型**：<font color="#008000">买单</font>
#### **买单价格**：<font color="#008000">%f</font>
#### **方向**：<font color="#008000">%s</font>
#### **买单数量**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	SlackApi(fmt.Sprintf(text, symbol, symbol, price, side, quantity, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) BuyOrderFail(symbol string, quantity float64, price float64, side string, info string) {
	text := `
## %s交易通知
#### **币种**：%s
#### **方向**：<font color="#008000">%s</font>
#### **买单价格**：<font color="#008000">%f</font>
#### **买单数量**：<font color="#008000">%f</font>
#### **类型**：<font color="#ff0000">挂买单失败</font>
>%s
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	SlackApi(fmt.Sprintf(text, symbol, symbol, side, price, quantity, info, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) SellOrderSuccess(symbol string, orderType string, profit float64) {
	text := `
## %s交易通知
#### **币种**：%s
#### **类型**：<font color="#ff0000">卖单</font>
#### **子类型**：<font color="#ff0000">%s</font>
#### **预计收益**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	SlackApi(fmt.Sprintf(text, symbol, symbol, orderType, profit, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) SellOrderFail(symbol string, info string) {
	text := `
## %s交易通知
#### **币种**：%s
#### **类型**：<font color="#ff0000">卖单失败</font>
>%s
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	SlackApi(fmt.Sprintf(text, symbol, symbol, info, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) RushOrderSuccess(symbol string, quantity float64, price float64, side string) {
	text := `
## %s交易通知
#### **币种**：%s
#### **类型**：<font color="#008000">买单</font>
#### **买单价格**：<font color="#008000">%f</font>
#### **方向**：<font color="#008000">%s</font>
#### **买单数量**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	SlackApi(fmt.Sprintf(text, symbol, symbol, price, side, quantity, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) NoticeFutureCoin(symbol string, side string, price string, autoOrder string) {
	text := `
## %s合约通知
#### **币种**：%s
#### **买卖类型**：<font color="#008000">%s</font>
#### **通知时价格**：<font color="#008000">%s</font>
#### **是否自动下单**：<font color="#008000">%s</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	SlackApi(fmt.Sprintf(text, symbol, symbol, side, price, autoOrder, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) ListenFutureCoin(symbol string, listenType string, percent float64, price float64) {
	text := `
## %s合约k线监控通知
#### **币种**：%s
#### **类型**：<font color="#008000">%s</font>
#### **当前变化率**：<font color="#008000">%f</font>
#### **当前价格**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	SlackApi(fmt.Sprintf(text, symbol, symbol, listenType, percent, price, nowTime()))
}

func (pusher Slack) ListenFutureCoinKlineKc(symbol string, listenType string, nowPrice, lossPrice, profitPrice1, profitPrice2, profitPrice3 float64) {
	text := `
## %s合约肯纳特通道监控通知
#### **币种**：%s
#### **类型**：<font color="#008000">%s</font>
#### **当前价格**：<font color="#008000">%f</font>
#### **推荐止损价格**：<font color="#008000">%f</font>
#### **推荐半仓止盈价格**：<font color="#008000">%f</font>
#### **推荐全部止盈价格**：<font color="#008000">%f</font>
#### **展望止盈价格**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	SlackApi(fmt.Sprintf(text, symbol, symbol, listenType, nowPrice, lossPrice, profitPrice1, profitPrice2, profitPrice3, nowTime()))
}

func (pusher Slack) ListenFutureCoinFundingRate(symbol string, listenType string, rate float64, price string) {
	text := `
## %s合约资金费率
#### **币种**：%s
#### **类型**：<font color="#008000">%s</font>
#### **当前资金费率**：<font color="#008000">%.2f%%</font>
#### **当前价格**：<font color="#008000">%s</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	SlackApi(fmt.Sprintf(text, symbol, symbol, listenType, rate, price, nowTime()))
}

func (pusher Slack) RushSpotBuyOrderSuccess(symbol string, quantity float64, price float64) {
	text := `
## %s新币抢购交易通知
#### **币种**：%s
#### **类型**：<font color="#008000">币币交易新币上市抢购</font>
#### **买单预计价格**：<font color="#008000">%f</font>
#### **买单数量**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDingApi(fmt.Sprintf(text, symbol, symbol, price, quantity, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) RushSpotSellOrderSuccess(symbol string, quantity float64, price float64) {
	text := `
## %s挖矿抢卖交易通知
#### **币种**：%s
#### **类型**：<font color="#008000">币币交易新币上市抢卖</font>
#### **卖单预计价格**：<font color="#008000">%f</font>
#### **卖单数量**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDingApi(fmt.Sprintf(text, symbol, symbol, price, quantity, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) BuySpotOrderSuccess(symbol string, quantity float64, price float64) {
	text := `
## %s现货交易通知
#### **币种**：%s
#### **类型**：<font color="#008000">买入</font>
#### **买单预计价格**：<font color="#008000">%f</font>
#### **买单数量**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDingApi(fmt.Sprintf(text, symbol, symbol, price, quantity, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) SellSpotOrderSuccess(symbol string, quantity float64, price float64) {
	text := `
## %s现货交易通知
#### **币种**：%s
#### **类型**：<font color="#008000">卖出</font>
#### **卖单预计价格**：<font color="#008000">%f</font>
#### **卖单数量**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDingApi(fmt.Sprintf(text, symbol, symbol, price, quantity, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) NoticeSpotCoin(symbol string, side string, price string, autoOrder string) {
	text := `
## %s币币通知
#### **币种**：%s
#### **买卖类型**：<font color="#008000">%s</font>
#### **通知时价格**：<font color="#008000">%s</font>
#### **是否自动下单**：<font color="#008000">%s</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDingApi(fmt.Sprintf(text, symbol, symbol, side, price, autoOrder, time.Now().Format("2006-01-02 15:04:05")))
}

func (pusher Slack) ListenSpotCoin(symbol string, listenType string, percent float64, price float64) {
	text := `
## %s现货k线监控通知
#### **币种**：%s
#### **类型**：<font color="#008000">%s</font>
#### **当前变化率**：<font color="#008000">%f</font>
#### **当前价格**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDingApi(fmt.Sprintf(text, symbol, symbol, listenType, percent, price, time.Now().Format("2006-01-02 15:04:05")))
}