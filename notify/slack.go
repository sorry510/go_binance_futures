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

var slack_token, _ = config.String("slack::slack_token")
var slack_channel_id, _ = config.String("slack::slack_channel_id")

type Slack struct {
}

type SlackMessage struct {
  Channel string  `json:"channel"`
	Blocks  []Block `json:"blocks"`
  // Text    string `json:"text"`
	// Mrkdwn  bool   `json:"mrkdwn"`
}

type Block struct {
    Type string   `json:"type"`
    Text *Text    `json:"text,omitempty"`
    Fields []Text `json:"fields,omitempty"`
    Elements []Text `json:"elements,omitempty"`
}

type Text struct {
    Type string `json:"type"`
    Text string `json:"text"`
}

func SlackApi(content string) {
	// 放到单独执行，避免主进程阻塞(未知原因突然不能在 goroutine 中执行了)
	go func () {
		url := "https://slack.com/api/chat.postMessage"
		
		blocks := []Block{
			{
				Type: "context",
				Elements: []Text{
					{
						Type: "mrkdwn",
						Text: content,
					},
				},
			},
		}
	
		requestBody := SlackMessage{
			Channel: slack_channel_id,
			Blocks: blocks,
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
		req.Header.Set("Authorization", "Bearer " + slack_token)
		
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

func (pusher Slack) TestPusher() {
  text := `
> Test
> push test success
> author <sorry510sf@gmail.com>`
  DingDingApi(text)
}

func (pusher Slack) FuturesOpenOrder(params FuturesOrderParams) {
	text := `
>%s
>{futures.side}：%s
>{futures.position_side}：%s
>{futures.price}：%f
>{futures.quantity}：%f
>{futures.leverage}：%f
>{futures.status}：%s
>{futures.error}：%s
>{futures.time}：%s

> author <sorry510sf@gmail.com>`

	text = fmt.Sprintf(lang.LangMatch(text),
		params.Symbol + params.Title,
		lang.Lang("futures." + params.Side),
		lang.Lang("futures." + params.PositionSide),
    params.Price,
    params.Quantity,
    params.Leverage,
    lang.Lang("futures." + params.Status),
    params.Error,
		nowTime(),
	)
	SlackApi(text)
}

func (pusher Slack) FuturesCloseOrder(params FuturesOrderParams) {
	text := `
>%s
>{futures.side}：%s
>{futures.position_side}：%s
>{futures.price}：%f
>{futures.quantity}：%f
>{futures.leverage}：%f
>{futures.profit}：%f
>{futures.remarks}：%s
>{futures.status}：%s
>{futures.error}：%s
>{futures.time}：%s

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
    lang.Lang("futures." + params.Status),
    params.Error,
		nowTime(),
	)
	SlackApi(text)
}

func (pusher Slack) FuturesNotice(params FuturesNoticeParams) {
	text := `
>%s
>{futures.side}：%s
>{futures.position_side}：%s
>{futures.price}：%f
>{futures.auto_order}：%s
>{futures.time}：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("futures." + params.Side),
    lang.Lang("futures." + params.PositionSide),
    params.Price,
    params.AutoOrder,
    nowTime(),
  )
  SlackApi(text)
}

func (pusher Slack) FuturesListenKlineBase(params FuturesListenParams) {
	text := `
>%s
>{futures.change_percent}：%.6f
>{futures.price}：%f
>{futures.remarks}：%s
>{futures.time}：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    params.ChangePercent,
    params.Price,
    params.Remarks,
    nowTime(),
  )
  SlackApi(text)
}

func (pusher Slack) FuturesListenKlineKc(params FuturesListenParams) {
	text := `
>%s
>{futures.side}：%s
>{futures.now_price}：%f
>{futures.stop_loss_price}：%f
>{futures.target_half_profit_price}：%f
>{futures.target_all_profit_price}：%f
>{futures.desired_price}：%f
>{futures.time}：%s

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
  SlackApi(text)
}

func (pusher Slack) FuturesListenKlineCustom(params FuturesListenParams) {
  text := `
>%s
>{futures.side}：%s
>{futures.now_price}：%f
>{futures.strategy_name}：%s
>{futures.time}：%s

>%s

> author <sorry510sf@gmail.com>`
  
  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("futures." + params.PositionSide),
    params.NowPrice,
    params.StrategyName,
    nowTime(),
    params.Remarks,
  )
  SlackApi(text)
}

func (pusher Slack) FuturesListenFundingRate(params FuturesListenParams) {
	text := `
>%s
>{futures.side}：%s
>{futures.funding_rate}：%.2f%%
>{futures.price}：%f
>{futures.remarks}：%s
>{futures.time}：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("futures." + params.PositionSide),
    params.FundingRate,
    params.Price,
    params.Remarks,
    nowTime(),
  )
  SlackApi(text)
}

func (pusher Slack) SpotOrder(params SpotOrderParams) {
	text := `
>%s
>{spot.side}：%s
>{spot.price}：%f
>{spot.quantity}：%f
>{spot.remarks}：%s
>{spot.status}：%s
>{spot.error}：%s
>{spot.time}：%s

> author <sorry510sf@gmail.com>`
  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("spot." + params.Side),
    params.Price,
    params.Quantity,
    params.Remarks,
    lang.Lang("spot." + params.Status),
    params.Error,
    nowTime(),
  )
  SlackApi(text)
}

func (pusher Slack) SpotNotice(params SpotNoticeParams) {
	text := `
>%s
>{spot.side}：%s
>{spot.price}：%f
>{spot.auto_order}：%s
>{spot.time}：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("spot." + params.Side),
    params.Price,
    params.AutoOrder,
    nowTime(),
  )
  SlackApi(text)
}

func (pusher Slack) SpotListenKlineBase(params SpotListenParams) {
	text := `
>%s
>{spot.change_percent}：%.6f
>{spot.price}：%f
>{spot.remarks}：%s
>{spot.time}：%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    params.ChangePercent,
    params.Price,
    params.Remarks,
    nowTime(),
  )
  SlackApi(text)
}

func (pusher Slack) FuturesCustomStrategyTest(params FuturesTestParams) {
	text := `
>%s
>{futures.side}：%s
>{futures.strategy_name}：%s
>{futures.time}：%s

>%s

> author <sorry510sf@gmail.com>`

  text = fmt.Sprintf(lang.LangMatch(text),
    params.Symbol + params.Title,
    lang.Lang("futures." + params.PositionSide),
    params.StrategyName,
    nowTime(),
    params.Remarks,
  )
  SlackApi(text)
}
