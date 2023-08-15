package tracing

import (
	"fmt"
	"strings"

	"github.com/DoOR-Team/goutils/alert"
)

type DingtalkWebhookRequest struct {
	Msgtype  string                         `json:"msgtype"`
	Markdown DingtalkWebhookRequestMarkdown `json:"markdown"`
}

type DingtalkWebhookRequestMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

const maxAlertMessageLen = 1000

func alertDing(title, subTitle, msg string, urlTitle, url string) error {
	msg = strings.ReplaceAll(msg, "\n", "\n>")
	title = "服务名 " + title
	text := fmt.Sprintf(`#### %s
报错，[%s >>>](%s)
>%s`, subTitle, urlTitle, url, msg)
	if len(text) > maxAlertMessageLen { // 太长的话切割
		text = text[0:maxAlertMessageLen] + fmt.Sprintf("... [>>>](%s)", url)
	}
	// 调试
	return alert.AlertDingMsg(title,
		text,
		alert.AutoFire(),
	)
}
