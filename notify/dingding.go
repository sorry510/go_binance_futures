package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go_binance_futures/lang"
	"io"
	"net/http"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
)

var g_dingding_token, _ = config.String("dingding::dingding_token")
var g_dingding_word, _ = config.String("dingding::dingding_word")

type DingDing struct {
  ModuleName string
}

type DingDingData struct {
	Msgtype string `json:"msgtype"`
	Markdown DingDingApiMarkDownData `json:"markdown"`
  At At `json:"at"`
}

type At struct {
  AtMobiles []string `json:"atMobiles,omitempty"`
  IsAtAll   bool     `json:"isAtAll"`
}

type DingDingApiMarkDownData struct {
	Title string  `json:"title"`
	Text string   `json:"text"`
 
}

// 钉钉通知, 频率限制 1分钟20次
// https://open.dingtalk.com/document/orgapp/the-robot-sends-a-group-message
func DingDingApi(content string, pusher Pusher) {
  dingding_token := g_dingding_token
  dingding_word := g_dingding_word
	// 放到单独执行，避免主进程阻塞(没有 block 程序时不会执行)
  notifyConfig := GetNotifyConfig(pusher)
  if notifyConfig.DingDingToken != "" {
    // 读取模块配置信息，覆盖全局
    dingding_token = notifyConfig.DingDingToken
    dingding_word = notifyConfig.DingDingWord
  }
  logs.Info("DingDingApi:", dingding_token, notifyConfig.ID)
  
	go func () {
		url := "https://oapi.dingtalk.com/robot/send?access_token=" + dingding_token
	
		requestBody := DingDingData{
			Msgtype: "markdown",
			Markdown: DingDingApiMarkDownData {
				Title: dingding_word,
				Text: content,
			},
      At: At {
        AtMobiles: []string{},
        IsAtAll: true,
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
	}()
}

func (pusher DingDing) SetModuleName(name string) Pusher {
  pusher.ModuleName = name
  return pusher
}

func (pusher DingDing) GetModuleName() string {
  return pusher.ModuleName
}

func (pusher DingDing) TestPusher() {
  text := `
## Test
#### push test success

> author <sorry510sf@gmail.com>`
  DingDingApi(text, pusher)
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
	DingDingApi(text, pusher)
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
	DingDingApi(text, pusher)
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
  DingDingApi(text, pusher)
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
  DingDingApi(text, pusher)
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
  DingDingApi(text, pusher)
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
  DingDingApi(text, pusher)
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
  DingDingApi(text, pusher)
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
  DingDingApi(text, pusher)
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
  DingDingApi(text, pusher)
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
  DingDingApi(text, pusher)
}

func (pusher DingDing) FuturesCustomStrategyTest(params FuturesTestParams) {
	text := `
## %s
#### **{futures.side}**：<font color="#008000">%s</font>
#### **{futures.position_side}**：<font color="#008000">%s</font>
#### **{futures.price}**：<font color="#008000">%f</font>
#### **{futures.close_price}**：<font color="#008000">%f</font>
#### **{futures.quantity}**：<font color="#008000">%f</font>
#### **{futures.leverage}**：<font color="#008000">%f</font>
#### **{futures.profit}**：<font color="#008000">%f</font>
#### **{futures.strategy_name}**：<font color="#008000">%s</font>
#### **{futures.time}**：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("futures." + params.Type),
    lang.Lang("futures." + params.PositionSide),
    params.Price,
    params.ClosePrice,
    params.Quantity,
    params.Leverage,
    params.Profit,
    params.StrategyName,
    nowTime(),
  )
  DingDingApi(text, pusher)
}

func (pusher DingDing) FuturesPositionConvert(params FuturesPositionConvertParams) {
  text := `
## %s
#### **{futures.status}**：<font color="#008000">%s</font>
#### **{futures.position_side}**：<font color="#008000">%s</font>
#### **{futures.now_price}**：<font color="#008000">%s</font>
#### **{futures.leverage}**：<font color="#008000">%s</font>
#### **{futures.profit}**：<font color="#008000">%s</font>
#### **{futures.time}**：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("futures." + params.Status),
    lang.Lang("futures." + params.PositionSide),
    params.Price,
    params.Leverage,
    params.UnRealizedProfit,
    nowTime(),
  )
  DingDingApi(text, pusher)
}
