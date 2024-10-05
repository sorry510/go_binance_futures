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

var dingding_token, _ = config.String("dingding::dingding_token")
var dingding_word, _ = config.String("dingding::dingding_word")

type DingDingData struct {
	Msgtype string `json:"msgtype"`
	Markdown MarkDownData `json:"markdown"`
}

type MarkDownData struct {
	Title string `json:"title"`
	Text string `json:"text"`
}

func DingDing(content string) {
	// 放到单独执行，避免住进程报错
	go func () {
		url := "https://oapi.dingtalk.com/robot/send?access_token=" + dingding_token
	
		requestBody := DingDingData{
			Msgtype: "markdown",
			Markdown: MarkDownData {
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
	} ()
}

func nowTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func BuyOrderSuccess(symbol string, quantity float64, price float64, side string) {
	text := `
## %s交易通知
#### **币种**：%s
#### **类型**：<font color="#008000">买单</font>
#### **买单价格**：<font color="#008000">%f</font>
#### **方向**：<font color="#008000">%s</font>
#### **买单数量**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDing(fmt.Sprintf(text, symbol, symbol, price, side, quantity, time.Now().Format("2006-01-02 15:04:05")))
}

func BuyOrderFail(symbol string, quantity float64, price float64, side string, info string) {
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
	DingDing(fmt.Sprintf(text, symbol, symbol, side, price, quantity, info, time.Now().Format("2006-01-02 15:04:05")))
}

func SellOrderSuccess(symbol string, orderType string, profit float64) {
	text := `
## %s交易通知
#### **币种**：%s
#### **类型**：<font color="#ff0000">卖单</font>
#### **子类型**：<font color="#ff0000">%s</font>
#### **预计收益**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDing(fmt.Sprintf(text, symbol, symbol, orderType, profit, time.Now().Format("2006-01-02 15:04:05")))
}

func SellOrderFail(symbol string, info string) {
	text := `
## %s交易通知
#### **币种**：%s
#### **类型**：<font color="#ff0000">卖单失败</font>
>%s
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDing(fmt.Sprintf(text, symbol, symbol, info, time.Now().Format("2006-01-02 15:04:05")))
}

func RushOrderSuccess(symbol string, quantity float64, price float64, side string) {
	text := `
## %s交易通知
#### **币种**：%s
#### **类型**：<font color="#008000">买单</font>
#### **买单价格**：<font color="#008000">%f</font>
#### **方向**：<font color="#008000">%s</font>
#### **买单数量**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDing(fmt.Sprintf(text, symbol, symbol, price, side, quantity, time.Now().Format("2006-01-02 15:04:05")))
}

func NoticeFutureCoin(symbol string, side string, price string, autoOrder string) {
	text := `
## %s合约通知
#### **币种**：%s
#### **买卖类型**：<font color="#008000">%s</font>
#### **通知时价格**：<font color="#008000">%s</font>
#### **是否自动下单**：<font color="#008000">%s</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDing(fmt.Sprintf(text, symbol, symbol, side, price, autoOrder, time.Now().Format("2006-01-02 15:04:05")))
}

func ListenFutureCoin(symbol string, listenType string, percent float64, price float64) {
	text := `
## %s合约k线监控通知
#### **币种**：%s
#### **类型**：<font color="#008000">%s</font>
#### **当前变化率**：<font color="#008000">%f</font>
#### **当前价格**：<font color="#008000">%f</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDing(fmt.Sprintf(text, symbol, symbol, listenType, percent, price, nowTime()))
}

func ListenFutureCoinKlineKc(symbol string, listenType string, nowPrice, lossPrice, profitPrice1, profitPrice2, profitPrice3 float64) {
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
	DingDing(fmt.Sprintf(text, symbol, symbol, listenType, nowPrice, lossPrice, profitPrice1, profitPrice2, profitPrice3, nowTime()))
}