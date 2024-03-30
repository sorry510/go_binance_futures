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
}

func BuyOrderSuccess(symbol string, quantity string, price string, side string) {
	const text = `
## %s交易通知
#### **币种**：%s
#### **类型**：<font color="#008000">买单</font>
#### **买单价格**：<font color="#008000">%s</font>
#### **方向**：<font color="#008000">%s</font>
#### **买单数量**：<font color="#008000">%s</font>
#### **时间**：%s

> author <sorry510sf@gmail.com>`
	DingDing(fmt.Sprintf(text, symbol, symbol, price, side, quantity, time.Now().Format("2006-01-02 15:04:05")))
}