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
	Lang         string               `json:"lang" validate:"omitempty,oneof=zh en ja it"`
}

// SubscribePush 保存当前用户的一条浏览器订阅。
// endpoint 全局唯一：同一浏览器重复授权/换账号登录时把订阅收敛到当前用户。
func SubscribePush(req component.BetterRequest[SubscribePushReq]) component.Response {
	sub := req.Params.Subscription
	if err := pushSubscription.Upsert(req.UserId, sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth, req.Params.Lang); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(true)
}

// UnsubscribePushReq POST push/unsubscribe 请求。
type UnsubscribePushReq struct {
	Endpoint string `json:"endpoint" validate:"required"`
}

// UnsubscribePush 删除指定 endpoint 的订阅（幂等）。
// 只删除属于当前用户的订阅：先按 user_id+endpoint 定位，防止越权删他人订阅。
func UnsubscribePush(req component.BetterRequest[UnsubscribePushReq]) component.Response {
	// DeleteByEndpoint 本身按 endpoint 删；为防越权，先确认该 endpoint 属于当前用户。
	owned := false
	for _, sub := range pushSubscription.ListByUser(req.UserId) {
		if sub != nil && sub.Endpoint == req.Params.Endpoint {
			owned = true
			break
		}
	}
	if !owned {
		// 不是自己的订阅：按不存在处理（幂等成功，不泄露他人订阅存在性）。
		return component.SuccessResponse(true)
	}
	if err := pushSubscription.DeleteByEndpoint(req.Params.Endpoint); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(true)
}
