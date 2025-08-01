package models

type NotifyConfig struct {
	ID int64 `orm:"column(id)" json:"id"`
	Enable int `orm:"column(enable)" json:"enable"` // 是否启用 1:启用, 0:禁用
	Channel string `orm:"column(channel)" json:"channel"` // dingding, slack
	Module string `orm:"column(module)" json:"module"` // futures, futures_test, futures_position_convert, coin_notice, coin_listen, funding_rate, new_coin_rush
	DingDingToken string `orm:"column(dingding_token)" json:"dingding_token"`
	DingDingWord string `orm:"column(dingding_word)" json:"dingding_word"`
	SlackToken string `orm:"column(slack_token)" json:"slack_token"`
	SlackChannelId string `orm:"column(slack_channel_id)" json:"slack_channel_id"`
	CreateTime int64 `orm:"column(createTime)" json:"createTime"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}

func (u *NotifyConfig) TableName() string {
    return "notify_config"
}