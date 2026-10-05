package httpnotifyservice

import (
	"errors"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// EventWebhookTest 管理端「测试发送」的事件名：不能订阅，只在手动测试时发送（issue #1049）。
const EventWebhookTest = "webhook.test"

// WebhookTestData generic 通道测试事件的数据。
type WebhookTestData struct {
	BaseURI      string `json:"baseUri"`
	EndpointID   string `json:"endpointId"`
	EndpointName string `json:"endpointName"`
	Message      string `json:"message"`
}

// SendTest 同步向单个 Endpoint 发送一条测试消息，返回可展示给管理员的失败原因
// （已去掉请求 URL，飞书地址是凭据）。不受总开关、启用状态和订阅事件限制，
// 也不计入失败次数与异常停用：测试是为了在保存前排查问题。
func SendTest(endpoint pageConfig.HttpNotifyEndpoint, baseURI string, now time.Time) error {
	channel := channelFor(endpoint)
	body, err := channel.testBody(endpoint, baseURI, now)
	if err != nil {
		return err
	}
	req, err := channel.buildRequest(endpoint, EventWebhookTest, deliveryID(), now.Unix(), body)
	if err != nil {
		return errors.New(deliveryErrorMessage(err))
	}
	resp, err := sendRequest(req, endpointTimeout(endpoint))
	if err != nil {
		return errors.New(deliveryErrorMessage(err))
	}
	defer resp.Body.Close()
	if err := channel.checkResponse(resp); err != nil {
		return errors.New(deliveryErrorMessage(err))
	}
	return nil
}
