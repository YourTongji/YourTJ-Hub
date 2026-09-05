package api

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushSubscription"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/webpushservice"
)

// PushConfigResp GET push/config 响应。
type PushConfigResp struct {
	Configured           bool   `json:"configured"`
	ApplicationServerKey string `json:"applicationServerKey,omitempty"`
}

// GetPushConfigReq GET push/config（无参数）。
type GetPushConfigReq struct{}

// GetPushConfig 返回实例 Web Push 通道状态与 VAPID 公钥。
// applicationServerKey 只在通道启用时返回（浏览器 subscribe 的
// PushSubscriptionOptions 需要 65B P-256 未压缩点 base64url）。
func GetPushConfig(req component.BetterRequest[GetPushConfigReq]) component.Response {
	key := webpushservice.PublicKeyOrEmpty()
	return component.SuccessResponse(PushConfigResp{
		Configured:           key != "",
		ApplicationServerKey: key,
	})
}

// PushSubscriptionBody 前端 PushSubscription.toJSON() 的订阅对象。
type PushSubscriptionBody struct {
	Endpoint string              `json:"endpoint" validate:"required"`
	Keys     PushSubscriptionKey `json:"keys" validate:"required"`
}

// PushSubscriptionKey 订阅加密密钥。
type PushSubscriptionKey struct {
	P256dh string `json:"p256dh" validate:"required"`
	Auth   string `json:"auth" validate:"required"`
}

// SubscribePushReq POST push/subscribe 请求。
type SubscribePushReq struct {
	Subscription PushSubscriptionBody `json:"subscription" validate:"required"`
	Lang         string               `json:"lang" validate:"omitempty,oneof=zh en ja de"`
}

// maxPushSubscriptionsPerUser 是单用户可存储的订阅数上限（review P1 fan-out
// 防护）：每浏览器/每设备一条订阅，理论上限按实际设备数 + 余量设定，超限时
// rep 层按 id 升序淘汰最旧行，使 worker 串行 fan-out 有界。
const maxPushSubscriptionsPerUser = 20

// SubscribePush 保存当前用户的一条浏览器订阅。
// endpoint 全局唯一：同一浏览器重复授权/换账号登录时把订阅收敛到当前用户。
// 订阅 endpoint 在存储前通过推送服务白名单校验（review P1 SSRF：未知/内网/
// IP 字面量 host 一律拒绝，见 webpushservice.ValidateEndpoint）。
func SubscribePush(req component.BetterRequest[SubscribePushReq]) component.Response {
	sub := req.Params.Subscription
	if err := webpushservice.ValidateEndpoint(sub.Endpoint); err != nil {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	if _, err := pushSubscription.UpsertCapped(req.UserId, sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth, req.Params.Lang, maxPushSubscriptionsPerUser); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(true)
}

// UnsubscribePushReq POST push/unsubscribe 请求。
type UnsubscribePushReq struct {
	Endpoint string `json:"endpoint" validate:"required"`
}

// UnsubscribePush 删除当前用户指定的订阅 endpoint（幂等）。
// 删除谓词同时限定 user_id + endpoint：endpoint 可能在快照与删除之间被其他
// 账号经 Upsert 接管，owner 限定避免误删新归属者的有效订阅；不属于自己的
// endpoint 按不存在处理（幂等成功，不泄露他人订阅存在性）。
func UnsubscribePush(req component.BetterRequest[UnsubscribePushReq]) component.Response {
	if err := pushSubscription.DeleteByEndpoint(req.Params.Endpoint, req.UserId); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(true)
}
