package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go_binance_futures/lang"
	"io"
	"net/http"

	"github.com/beego/beego/v2/core/config"
)

var dingding_token, _ = config.String("dingding::dingding_token")
var dingding_word, _ = config.String("dingding::dingding_word")

type DingDing struct {
}

type DingDingData struct {
	Msgtype string `json:"msgtype"`
	Markdown DingDingApiMarkDownData `json:"markdown"`
}

type DingDingApiMarkDownData struct {
	Title string `json:"title"`
	Text string `json:"text"`
}

func DingDingApi(content string) {
	// 放到单独执行，避免主进程阻塞(未知原因突然不能在 goroutine 中执行了)
	// go func () {
		url := "https://oapi.dingtalk.com/robot/send?access_token=" + dingding_token
	
		requestBody := DingDingData{
			Msgtype: "markdown",
			Markdown: DingDingApiMarkDownData {
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

func (pusher DingDing) FuturesOpenOrder(params FuturesOrderParams) {
	text := `
## %s
#### **{futures.side}**：<font color="#008000">%s</font>
#### **{futures.position_side}**：<font color="#008000">%s</font>
#### **{futures.price}**：<font color="#008000">%f</font>
#### **{futures.quantity}**：<font color="#008000">%f</font>
#### **{futures.leverage}**：<font color="#008000">%f</font>
#### **{futures.status}**：<font color="%s">%s</font>
#### **{futures.error}**：<font color="#FF0000">%s</font>
#### **{futures.time}**：%s

> author <sorry510sf@gmail.com>`

	text = fmt.Sprintf(lang.LangMatch(text),
		params.Symbol + params.Title,
		lang.Lang("futures." + params.Side),
		lang.Lang("futures." + params.PositionSide),
    params.Price,
    params.Quantity,
    params.Leverage,
    getStatusColor(params.Status), lang.Lang("futures." + params.Status),
    params.Error,
		nowTime(),
	)
	DingDingApi(text)
}

func (pusher DingDing) FuturesCloseOrder(params FuturesOrderParams) {
	text := `
## %s
#### **{futures.side}**：<font color="#008000">%s</font>
#### **{futures.position_side}**：<font color="#008000">%s</font>
#### **{futures.price}**：<font color="#008000">%f</font>
#### **{futures.quantity}**：<font color="#008000">%f</font>
#### **{futures.leverage}**：<font color="#008000">%f</font>
#### **{futures.profit}**：<font color="#008000">%f</font>
#### **{futures.remarks}**：<font color="#FF0000">%s</font>
#### **{futures.status}**：<font color="%s">%s</font>
#### **{futures.error}**：<font color="#FF0000">%s</font>
#### **{futures.time}**：%s

> author <sorry510sf@gmail.com>`

	text = fmt.Sprintf(lang.LangMatch(text),
		params.Symbol + params.Title,
		lang.Lang("futures." + params.Side),
		lang.Lang("futures." + params.PositionSide),
    params.Price,
    params.Quantity,
    params.Leverage,
    params.Profit,
    params.Remarks,
    getStatusColor(params.Status), lang.Lang("futures." + params.Status),
    params.Error,
		nowTime(),
	)
	DingDingApi(text)
}

func (pusher DingDing) FuturesNotice(params FuturesNoticeParams) {
	text := `
## %s
#### **{futures.side}**：<font color="#008000">%s</font>
#### **{futures.position_side}**：<font color="#008000">%s</font>
#### **{futures.price}**：<font color="#008000">%f</font>
#### **{futures.auto_order}**：<font color="#008000">%s</font>
#### **{futures.time}**：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("futures." + params.Side),
    lang.Lang("futures." + params.PositionSide),
    params.Price,
    params.AutoOrder,
    nowTime(),
  )
  DingDingApi(text)
}

func (pusher DingDing) FuturesListenKlineBase(params FuturesListenParams) {
	text := `
## %s
#### **{futures.change_percent}**：<font color="#008000">%.6f</font>
#### **{futures.price}**：<font color="#008000">%f</font>
#### **{futures.remarks}**：<font color="#008000">%s</font>
#### **{futures.time}**：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    params.ChangePercent,
    params.Price,
    params.Remarks,
    nowTime(),
  )
  DingDingApi(text)
}

func (pusher DingDing) FuturesListenKlineKc(params FuturesListenParams) {
	text := `
## %s
#### **{futures.side}**：<font color="#008000">%s</font>
#### **{futures.now_price}**：<font color="#008000">%f</font>
#### **{futures.stop_loss_price}**：<font color="#008000">%f</font>
#### **{futures.target_half_profit_price}**：<font color="#008000">%f</font>
#### **{futures.target_all_profit_price}**：<font color="#008000">%f</font>
#### **{futures.desired_price}**：<font color="#008000">%f</font>
#### **{futures.time}**：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("futures." + params.PositionSide),
    params.NowPrice,
    params.StopLossPrice,
    params.TargetHalfProfitPrice,
    params.TargetAllProfitPrice,
    params.DesiredPrice,
    nowTime(),
  )
  DingDingApi(text)
}

func (pusher DingDing) FuturesListenKlineCustom(params FuturesListenParams) {
	text := `
## %s
#### **{futures.side}**：<font color="#008000">%s</font>
#### **{futures.now_price}**：<font color="#008000">%f</font>
#### **{futures.strategy_name}**：<font color="#008000">%s</font>
#### **{futures.time}**：%s

> <font color="#008000">%s</font>

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("futures." + params.PositionSide),
    params.NowPrice,
    params.StrategyName,
    nowTime(),
    params.Remarks,
  )
  DingDingApi(text)
}

func (pusher DingDing) FuturesListenFundingRate(params FuturesListenParams) {
	text := `
## %s
#### **{futures.side}**：<font color="#008000">%s</font>
#### **{futures.funding_rate}**：<font color="#008000">%.2f%%</font>
#### **{futures.price}**：<font color="#008000">%f</font>
#### **{futures.remarks}**：<font color="#008000">%s</font>
#### **{futures.time}**：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("futures." + params.PositionSide),
    params.FundingRate,
    params.Price,
    params.Remarks,
    nowTime(),
  )
  DingDingApi(text)
}

func (pusher DingDing) SpotOrder(params SpotOrderParams) {
	text := `
## %s
#### **{spot.side}**：<font color="#008000">%s</font>
#### **{spot.price}**：<font color="#008000">%f</font>
#### **{spot.quantity}**：<font color="#008000">%f</font>
#### **{spot.remarks}**：<font color="#FF0000">%s</font>
#### **{spot.status}**：<font color="%s">%s</font>
#### **{spot.error}**：<font color="#FF0000">%s</font>
#### **{spot.time}**：%s

> author <sorry510sf@gmail.com>`
  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("spot." + params.Side),
    params.Price,
    params.Quantity,
    params.Remarks,
    getStatusColor(params.Status), lang.Lang("spot." + params.Status),
    params.Error,
    nowTime(),
  )
  DingDingApi(text)
}

func (pusher DingDing) SpotNotice(params SpotNoticeParams) {
	text := `
## %s
#### **{spot.side}**：<font color="#008000">%s</font>
#### **{spot.price}**：<font color="#008000">%f</font>
#### **{spot.auto_order}**：<font color="#008000">%s</font>
#### **{spot.time}**：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("spot." + params.Side),
    params.Price,
    params.AutoOrder,
    nowTime(),
  )
  DingDingApi(text)
}

func (pusher DingDing) SpotListenKlineBase(params SpotListenParams) {
	text := `
## %s
#### **{spot.change_percent}**：<font color="#008000">%.6f</font>
#### **{spot.price}**：<font color="#008000">%f</font>
#### **{spot.remarks}**：<font color="#008000">%s</font>
#### **{spot.time}**：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    params.ChangePercent,
    params.Price,
    params.Remarks,
    nowTime(),
  )
  DingDingApi(text)
}
