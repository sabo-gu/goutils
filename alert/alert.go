package alert

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/DoOR-Team/wechat_bot"
)

// 有多种机器人，提醒不同
// 分别叫，小红，小蓝，小火
// 小红：已知的业务错误异常报警
// 小蓝：已知的业务提醒报警
// 小火：未知的系统错误报警
// 小开：开发环境报警
var AlertChanelEnum = struct {
	RedFireControl  string
	BlueFireControl string
	FireControl     string
	DevelopAlert    string
}{
	RedFireControl:  "680cbd99-fc6d-416f-adc6-d285791eb340",
	BlueFireControl: "f269d927-dcc9-4e1c-9349-e9073b2e7b2f",
	FireControl:     "be1cfa28-c989-40bd-8dd9-e26e062c48af",
	DevelopAlert:    "dea8d4f2-4e1a-43bc-a51e-348a30a8b3cd",
}

const wxUrl = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key="

func AlertDingMsgWithConfig(module, text, alertType string) error {

	title := fmt.Sprintf("%s [%s] 异常，请相关同事注意！", viper.GetString("k8s_appname"), module)
	return SendWXMsg(title, text, alertType)
}

func AlertDingMsg(title, text, alertType string) error {
	err := SendWXMsg(title, text, alertType)
	if err != nil {
		return err
	}
	return SendWXMentionMessage(alertType)
}

// TODO 根据appname的注册人，获取管理人员并进行提醒
func SendWXMentionMessage(alertType string) error {
	return nil
}

func SendWXMsg(title string, text string, alertType string) error {
	msgContent := wechat_bot.MsgContent{
		Msgtype:  "markdown",
		Markdown: wechat_bot.Markdown{},
	}

	wechatUrl := wxUrl + alertType
	msgContent.Markdown.Content = fmt.Sprintf(`### %s
%s`, title, text)
	return wechat_bot.SendMsg(wechatUrl, msgContent)
}

func AutoFire() string {
	alertChan := AlertChanelEnum.FireControl
	if viper.GetString("k8s_namespace") != "production" {
		alertChan = AlertChanelEnum.DevelopAlert
	}
	return alertChan
}

func AlertRed() string {
	alertChan := AlertChanelEnum.RedFireControl
	if viper.GetString("k8s_namespace") != "production" {
		alertChan = AlertChanelEnum.DevelopAlert
	}
	return alertChan
}
func AlertBlue() string {
	alertChan := AlertChanelEnum.BlueFireControl
	if viper.GetString("k8s_namespace") != "production" {
		alertChan = AlertChanelEnum.DevelopAlert
	}
	return alertChan
}
