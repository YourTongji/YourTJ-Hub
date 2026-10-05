package httpnotifyservice

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// notifyChannel 通知通道适配器（issue #1049）：事件生产者只产出领域事件与安全摘要，
// 通道决定请求体、签名与“对方是否真正接收”的判定。新增 Slack/企业微信等通道只需
// 实现本接口，不改变事件生产者。
type notifyChannel interface {
	// supportsEvent 通道能否投递该事件名（卡片通道只渲染审批事件）。
	supportsEvent(eventName string) bool
	// requiresApproval 通道是否只能渲染审批摘要（Message.Approval 为空时跳过）。
	requiresApproval() bool
	encode(endpoint pageConfig.HttpNotifyEndpoint, alt Alternative, approval *ApprovalPayload, now time.Time) ([]byte, error)
	// testBody 管理端「测试发送」的请求体：让管理员在保存前确认地址、签名与展示效果。
	testBody(endpoint pageConfig.HttpNotifyEndpoint, baseURI string, now time.Time) ([]byte, error)
	buildRequest(endpoint pageConfig.HttpNotifyEndpoint, eventName string, deliveryID string, timestamp int64, body []byte) (*http.Request, error)
	// checkResponse 非 nil 即计为失败，进入 failureCount/lastError/异常停用。
	checkResponse(resp *http.Response) error
}

func channelFor(endpoint pageConfig.HttpNotifyEndpoint) notifyChannel {
	switch pageConfig.NormalizeHttpNotifyChannel(endpoint.ChannelType) {
	case pageConfig.HttpNotifyChannelFeishu:
		return feishuChannel{}
	case pageConfig.HttpNotifyChannelAstrBot:
		return astrbotChannel{}
	default:
		return genericChannel{}
	}
}

// genericChannel 通用 Webhook：Envelope JSON + X-Goose-* 头 + 可选 HMAC 签名，2xx 即成功。
type genericChannel struct{}

func (genericChannel) supportsEvent(string) bool { return true }

func (genericChannel) requiresApproval() bool { return false }

func (genericChannel) encode(_ pageConfig.HttpNotifyEndpoint, alt Alternative, _ *ApprovalPayload, now time.Time) ([]byte, error) {
	return json.Marshal(Envelope{Event: alt.Event, Timestamp: now.Unix(), Data: alt.Data})
}

// testBody 发送 webhook.test 事件：与正式事件同样带 X-Goose-* 头与签名，接收方可借此验签。
func (genericChannel) testBody(endpoint pageConfig.HttpNotifyEndpoint, baseURI string, now time.Time) ([]byte, error) {
	return json.Marshal(Envelope{Event: EventWebhookTest, Timestamp: now.Unix(), Data: WebhookTestData{
		BaseURI:      baseURI,
		EndpointID:   endpoint.Id,
		EndpointName: endpoint.Name,
		Message:      "This is a test delivery from the HTTP notification settings.",
	}})
}

func (genericChannel) buildRequest(endpoint pageConfig.HttpNotifyEndpoint, eventName string, deliveryID string, timestamp int64, body []byte) (*http.Request, error) {
	return buildRequest(endpoint, eventName, deliveryID, timestamp, body)
}

func (genericChannel) checkResponse(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}

// IsApprovalEvent 事件名是否属于版主审批通知（含“全部举报”聚合兼容事件）。
func IsApprovalEvent(eventName string) bool {
	switch eventName {
	case EventReportCreated,
		EventReviewTopicRequested, EventReviewPostRequested,
		EventReportTopicCreated, EventReportPostCreated,
		EventReportChatMessageCreated, EventReportCourseReviewCreated:
		return true
	default:
		return false
	}
}
