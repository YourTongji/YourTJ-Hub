package api

import (
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushDevice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushSubscription"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/nativepushservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/webpushservice"
)

// NativePushConfigResp GET push/config 中原生推送通道状态。
type NativePushConfigResp struct {
	APNsEnabled  bool `json:"apnsEnabled"`
	FCMEnabled   bool `json:"fcmEnabled"`
	JPushEnabled bool `json:"jpushEnabled"`
}

// PushConfigResp GET push/config 响应。
type PushConfigResp struct {
	Configured           bool                 `json:"configured"`
	ApplicationServerKey string               `json:"applicationServerKey,omitempty"`
	Native               NativePushConfigResp `json:"native"`
}

// GetPushConfigReq GET push/config（无参数）。
type GetPushConfigReq struct{}

// GetPushConfig 返回实例 Web Push 通道状态与 VAPID 公钥，以及原生推送
// （APNs/FCM）通道是否启用。
// applicationServerKey 只在 Web Push 启用时返回（浏览器 subscribe 的
// PushSubscriptionOptions 需要 65B P-256 未压缩点 base64url）。
func GetPushConfig(req component.BetterRequest[GetPushConfigReq]) component.Response {
	key := webpushservice.PublicKeyOrEmpty()
	channels := nativepushservice.LoadChannels()
	return component.SuccessResponse(PushConfigResp{
		Configured:           key != "",
		ApplicationServerKey: key,
		Native: NativePushConfigResp{
			APNsEnabled:  channels.APNs.Enabled(),
			FCMEnabled:   channels.FCM.Enabled(),
			JPushEnabled: channels.JPush.Enabled(),
		},
	})
}

// RegisterPushDeviceReq POST push/device/register 请求。
type RegisterPushDeviceReq struct {
	Provider string `json:"provider" validate:"omitempty,oneof=apns fcm jpush"`
	Platform string `json:"platform" validate:"required,oneof=ios android"`
	Token    string `json:"token" validate:"required,min=1,max=512"`
}

// maxPushDevicesPerUser 是单用户可存储的设备注册数上限（fan-out 有界防护，
// 与 maxPushSubscriptionsPerUser 同口径）：超出时按 id 升序淘汰最旧设备行。
const maxPushDevicesPerUser = 20

// RegisterPushDevice 保存当前用户的一台移动端设备注册（mobile Route A）。
// token 全局唯一：同一设备重新授权/换账号登录时把注册收敛到当前用户并刷新
// 最近注册时间（created_at 不变）。device token 是推送服务长期凭据，等同
// 会话凭据对待：仅本人可管理、账号注销即删。
func RegisterPushDevice(req component.BetterRequest[RegisterPushDeviceReq]) component.Response {
	token := strings.TrimSpace(req.Params.Token)
	if token == "" {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	provider := req.Params.Provider
	if provider == "" {
		if req.Params.Platform == "ios" {
			provider = "apns"
		} else {
			provider = "fcm"
		}
	}
	if (req.Params.Platform == "ios" && provider != "apns") ||
		(req.Params.Platform == "android" && provider == "apns") {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	if _, err := pushDevice.UpsertCapped(req.UserId, req.Params.Platform, token, maxPushDevicesPerUser, time.Now(), provider); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(true)
}

// UnregisterPushDeviceReq POST push/device/unregister 请求。
type UnregisterPushDeviceReq struct {
	Token string `json:"token" validate:"required,min=1,max=512"`
}

// UnregisterPushDevice 删除当前用户指定的设备注册（幂等）。
// 删除谓词同时限定 user_id + token：token 可能在快照与删除之间被其他账号
// 经 Upsert 接管，owner 限定避免误删新归属者的有效注册；不属于自己的 token
// 按不存在处理（幂等成功，不泄露他人注册存在性）。
func UnregisterPushDevice(req component.BetterRequest[UnregisterPushDeviceReq]) component.Response {
	token := strings.TrimSpace(req.Params.Token)
	if token == "" {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	if err := pushDevice.DeleteByToken(token, req.UserId); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(true)
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
