package sms

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"

	"github.com/DoOR-Team/goutils/djson"
)

func SendMessage(accessKeyId string, accessSecret string, parameters map[string]string, templateCode string, phone string) (*dysmsapi.SendSmsResponse, error) {
	// client, err := dysmsapi.NewClientWithAccessKey("cn-hangzhou", "<accessKeyId>", "<accessSecret>")
	client, err := dysmsapi.NewClientWithAccessKey("cn-hangzhou", accessKeyId, accessSecret)
	if err != nil {
		return nil, err
	}
	request := dysmsapi.CreateSendSmsRequest()
	request.Scheme = "https"

	// request.PhoneNumbers = "19957159875"
	request.PhoneNumbers = phone
	request.SignName = "雪浪河图"
	// request.TemplateCode = "SMS_198865340"
	request.TemplateCode = templateCode
	// request.TemplateParam = "{\"code\":\"123456\"}"
	request.TemplateParam = djson.ToJsonString(parameters)

	return client.SendSms(request)
	// return response, err
}
