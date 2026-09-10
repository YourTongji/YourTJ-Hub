// Package nativepushservice 实现服务端事件驱动的移动端原生推送通道
// （mobile Route A）：通知行创建成功后入 taskQueue outbox，专用 worker 异步向
// 该用户全部已注册设备（iOS APNs / Android JPush（兼容 FCM））发送系统推送。
// 与 webpushservice 同一套 outbox + worker 语义：推送是站内通知红点的
// best-effort 外投，失败绝不影响通知落库与业务请求；实例未配置 [push.apns] /
// [push.fcm] 时通道关闭（dev 从 main 快照同步的任务行 no-op，绝不外发）。
package nativepushservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushDevice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/sideshow/apns2"
	apnstoken "github.com/sideshow/apns2/token"
	"github.com/spf13/cast"
	"golang.org/x/oauth2/google"
)

// TaskTypeNativePush 是 nativepush outbox 任务类型前缀。任务 Type 为
// "nativepush.{notificationId}"，RunWorker 按前缀隔离领取（与 webpush. 互不干扰）。
const TaskTypeNativePush = "nativepush."

// fcmScope 是 FCM HTTP v1 API 所需的 OAuth2 scope（service-account JWT 换取）。
const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

// fcmSendURLPattern 是 FCM HTTP v1 发送端点（project id 来自配置）。
const fcmSendURLPattern = "https://fcm.googleapis.com/v1/projects/%s/messages:send"

// fcmHTTPClient 是 FCM 发送用 HTTP 客户端：FCM/Google token 端点为固定公网
// 域名，无用户可控 URL，不需要 webpush 的白名单拨号校验。
var fcmHTTPClient = &http.Client{Timeout: 10 * time.Second}

// errTokenStale 表示设备 token 在推送服务侧已失效（APNs 410 / BadDeviceToken、
// FCM 404 UNREGISTERED）：worker 收到后删除本地 push_device 行。
var errTokenStale = errors.New("nativepush: device token stale")

// APNsConfig 是 [push.apns] 配置段（token-based .p8 认证）。
type APNsConfig struct {
	KeyPath     string // .p8 密钥文件路径
	KeyID       string
	TeamID      string
	BundleID    string // APNs topic（bundle id）
	Environment string // sandbox | production
}

// LoadAPNsConfig 读取 [push.apns] 配置段。
func LoadAPNsConfig() APNsConfig {
	return APNsConfig{
		KeyPath:     strings.TrimSpace(preferences.GetString("push.apns.key_path", "")),
		KeyID:       strings.TrimSpace(preferences.GetString("push.apns.key_id", "")),
		TeamID:      strings.TrimSpace(preferences.GetString("push.apns.team_id", "")),
		BundleID:    strings.TrimSpace(preferences.GetString("push.apns.bundle_id", "")),
		Environment: strings.ToLower(strings.TrimSpace(preferences.GetString("push.apns.environment", ""))),
	}
}

// Enabled 返回 APNs 通道是否启用：凭据字段全非空且 environment 合法。
// 格式不符视为未配置（fail-closed）；密钥文件是否可读在发送时判定并仅告警。
func (c APNsConfig) Enabled() bool {
	if c.KeyPath == "" || c.KeyID == "" || c.TeamID == "" || c.BundleID == "" {
		return false
	}
	// LoadAPNsConfig 已归一化小写；直接构造（测试/后续调用方）也容忍大小写。
	environment := strings.ToLower(c.Environment)
	return environment == "sandbox" || environment == "production"
}

// FCMConfig 是 [push.fcm] 配置段。
type FCMConfig struct {
	CredentialsPath string // service-account JSON 文件路径
	ProjectID       string
}

// LoadFCMConfig 读取 [push.fcm] 配置段。
func LoadFCMConfig() FCMConfig {
	return FCMConfig{
		CredentialsPath: strings.TrimSpace(preferences.GetString("push.fcm.credentials_path", "")),
		ProjectID:       strings.TrimSpace(preferences.GetString("push.fcm.project_id", "")),
	}
}

// Enabled 返回 FCM 通道是否启用（凭据文件与 project id 均非空）。
func (c FCMConfig) Enabled() bool {
	return c.CredentialsPath != "" && c.ProjectID != ""
}

// Channels 是实例当前生效的原生推送通道集合。
type Channels struct {
	APNs  APNsConfig
	FCM   FCMConfig
	JPush JPushConfig
}

// LoadChannels 读取全部原生推送通道配置。
func LoadChannels() Channels {
	return Channels{APNs: LoadAPNsConfig(), FCM: LoadFCMConfig(), JPush: LoadJPushConfig()}
}

// Enabled 返回是否存在至少一条已启用通道。
func (c Channels) Enabled() bool {
	return c.APNs.Enabled() || c.FCM.Enabled() || c.JPush.Enabled()
}

// LogConfigStatus 在 serve 启动时输出原生推送通道状态。
func LogConfigStatus() {
	ch := LoadChannels()
	if !ch.Enabled() {
		slog.Info("nativepush: disabled (no [push.apns] / [push.fcm] / [push.jpush] configured)")
		return
	}
	slog.Info("nativepush: enabled", "apns", ch.APNs.Enabled(), "fcm", ch.FCM.Enabled(), "jpush", ch.JPush.Enabled())
}

// PushTask 是 outbox 任务负载：定位一条通知行。
type PushTask struct {
	UserId         uint64 `json:"userId"`
	NotificationId uint64 `json:"notificationId"`
}

// EnqueueNotification 在通知行创建成功后入队原生推送任务。非事务写入：通知行已
// 独立提交，入队失败只丢推送（站内红点兜底），绝不影响调用方。实例未启用
// 任何原生通道时直接跳过（dev 不产生任务行）。
func EnqueueNotification(userId uint64, notificationId uint64) {
	if userId == 0 || notificationId == 0 {
		return
	}
	if !LoadChannels().Enabled() {
		return
	}
	taskJSON, err := json.Marshal(PushTask{UserId: userId, NotificationId: notificationId})
	if err != nil {
		slog.Warn("nativepush: marshal push task failed", "userId", userId, "notificationId", notificationId, "err", err)
		return
	}
	if err := taskQueue.Create(&taskQueue.Entity{
		Type:     TaskTypeNativePush + cast.ToString(notificationId),
		Status:   taskQueue.StatusPending,
		TaskJson: string(taskJSON),
	}); err != nil {
		slog.Warn("nativepush: enqueue push task failed", "userId", userId, "notificationId", notificationId, "err", err)
	}
}

// RecoverStaleTasks 启动时回收上次进程遗留的 Running 推送任务（崩溃恢复，
// 与 webpush worker 同款）。
func RecoverStaleTasks() error {
	return taskQueue.RecoverStaleRunning(TaskTypeNativePush, taskQueue.LeaseDuration)
}

// CleanupTerminalTasks 删除指定时间前进入终态（Success/Failed）的推送任务行
// （每条通知一行 outbox，只置终态不清理会让 task_queue 无界增长）。由每日
// cron 与 webpush 清理共用同一执行点做有界保留清理。
func CleanupTerminalTasks(before time.Time, limit int) (int64, error) {
	return taskQueue.DeleteTerminalByTypePrefix(TaskTypeNativePush,
		[]int{int(taskQueue.StatusSuccess), int(taskQueue.StatusFailed)}, before, limit)
}

// RunPushTask 是原生推送 worker 的任务处理函数（backgroundservice.RunWorker
// 语义）：任务整体按 Success 收尾（attempted 语义，不依赖 taskQueue 重试——
// 部分设备成功后重试会造成重复投递）；单设备发送失败仅日志，由站内红点
// 兜底；设备 token 失效（APNs 410/BadDeviceToken、FCM 404 UNREGISTERED）
// 删除本地 push_device 行（镜像 webpush 404/410 清理）。
func RunPushTask(ctx context.Context, task *taskQueue.Entity) error {
	var payload PushTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		// 负载无法解析属于不可恢复数据错误：Success 收尾避免无限重试。
		slog.Warn("nativepush: malformed push task", "taskId", task.Id, "err", err)
		return nil
	}
	if payload.UserId == 0 || payload.NotificationId == 0 {
		return nil
	}

	// 实例未启用任何通道（dev 快照含 main 任务行时走此分支）：置 Success
	// 防误发，同时避免任务积压。启动时已打过状态日志，此处静默。
	ch := LoadChannels()
	if !ch.Enabled() {
		slog.Debug("nativepush: task skipped, channel disabled", "taskId", task.Id)
		return nil
	}

	notification := eventNotification.GetByID(payload.NotificationId)
	if notification.Id == 0 {
		// 通知行不存在（已删/迁移丢失）：无事件源，Success 跳过。
		slog.Debug("nativepush: notification gone, skip", "taskId", task.Id, "notificationId", payload.NotificationId)
		return nil
	}
	// 已读通知不再推送：用户已看到，避免冗余打扰（与 webpush 同口径）。
	if notification.IsRead {
		return nil
	}
	// 注销账号（软删）不再推送。
	if users.IsAccountClosed(payload.UserId) {
		return nil
	}

	devices := pushDevice.ListByUser(payload.UserId)
	if len(devices) == 0 {
		return nil
	}

	// 文案语言取账号 locale（原生设备注册不带语言属性；语言是账号级偏好，
	// 与 webpush 订阅级 lang 语义对齐后均收敛到同一四语言文案表）。
	locale := ""
	if user, err := users.Get(payload.UserId); err == nil {
		locale = user.Locale
	}
	msg := buildNativePayload(notification, locale)
	if msg == nil {
		return nil
	}

	// APNs token client 按任务构建：.p8 文件小、读取 + ECDSA 解析开销可忽略，
	// 且每次读取保证配置热更（换 key 后无需重启）；client 内部维护 HTTP/2
	// 连接池，worker 串行消费下不会反复建连。
	var apnsClient *apns2.Client
	if ch.APNs.Enabled() {
		client, err := newAPNsClient(ch.APNs)
		if err != nil {
			slog.Warn("nativepush: apns client init failed", "userId", payload.UserId, "err", err)
		} else {
			apnsClient = client
		}
	}
	// FCM OAuth2 access token 按任务获取一次（凭据 JSON 小；google 侧自动缓存
	// 直到过期，重复调用只在过期后重新签发）。
	var fcmAccessToken string
	if ch.FCM.Enabled() {
		token, err := fcmTokenSource(ctx, ch.FCM)
		if err != nil {
			slog.Warn("nativepush: fcm token init failed", "userId", payload.UserId, "err", err)
		} else {
			fcmAccessToken = token
		}
	}

	var sent, deleted int
	for _, dev := range devices {
		if dev == nil || dev.Token == "" {
			continue
		}
		var sendErr error
		switch dev.DeliveryProvider() {
		case "apns":
			if apnsClient == nil {
				continue
			}
			sendErr = sendAPNs(ctx, apnsClient, ch.APNs, dev.Token, msg)
		case "fcm":
			if fcmAccessToken == "" {
				continue
			}
			sendErr = sendFCM(ctx, fcmAccessToken, ch.FCM, dev.Token, msg)
		case "jpush":
			if !ch.JPush.Enabled() {
				continue
			}
			sendErr = sendJPush(ctx, ch.JPush, dev.Token, msg)

		default:
			continue
		}
		if sendErr != nil {
			if errors.Is(sendErr, errTokenStale) {
				// 410/404 等：设备在推送服务已失效，从本地删除，停止向死 token 发送。
				deleted++
				if delErr := pushDevice.DeleteByToken(dev.Token, payload.UserId); delErr != nil {
					slog.Warn("nativepush: delete stale device failed", "userId", payload.UserId, "err", delErr)
				}
				slog.Info("nativepush: device removed (token gone)", "userId", payload.UserId, "platform", dev.Platform)
				continue
			}
			if ctx.Err() != nil {
				// worker 关闭/租约取消：剩余设备未发送，把任务留给重试
				// （返回错误使 taskQueue 置 Retrying，下个进程续投）。
				return ctx.Err()
			}
			slog.Warn("nativepush: send failed", "userId", payload.UserId, "platform", dev.Platform, "err", sendErr)
			continue
		}
		sent++
	}
	slog.Debug("nativepush: task done", "taskId", task.Id, "userId", payload.UserId, "sent", sent, "deleted", deleted)
	return nil
}

// newAPNsClient 从 [push.apns] 配置构建 APNs token client（.p8 密钥认证），
// 并按 environment 选择 sandbox/production 端点。
func newAPNsClient(cfg APNsConfig) (*apns2.Client, error) {
	authKey, err := apnstoken.AuthKeyFromFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("load apns auth key: %w", err)
	}
	client := apns2.NewTokenClient(&apnstoken.Token{
		AuthKey: authKey,
		KeyID:   cfg.KeyID,
		TeamID:  cfg.TeamID,
	})
	if cfg.Environment == "sandbox" {
		return client.Development(), nil
	}
	return client.Production(), nil
}

// sendAPNs 发送一条 APNs 通知（aps.alert = title/body，自定义 route 字段携带
// 移动端 app 路由）。设备 token 失效（410 Unregistered/ExpiredToken，或 400
// BadDeviceToken）返回 errTokenStale 触发本地清理。
func sendAPNs(ctx context.Context, client *apns2.Client, cfg APNsConfig, deviceToken string, msg *nativePayload) error {
	notification := &apns2.Notification{
		DeviceToken: deviceToken,
		Topic:       cfg.BundleID,
		PushType:    apns2.PushTypeAlert,
		Payload: map[string]any{
			"aps": map[string]any{
				"alert": map[string]string{"title": msg.Title, "body": msg.Body},
				"sound": "default",
			},
			"route": msg.Route,
		},
	}
	resp, err := client.PushWithContext(ctx, notification)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("apns request failed: %w", err)
	}
	switch {
	case resp.Sent():
		return nil
	case resp.StatusCode == http.StatusGone:
		// 410 Unregistered / ExpiredToken：设备 token 已失效。
		return errTokenStale
	case resp.StatusCode == http.StatusBadRequest && resp.Reason == apns2.ReasonBadDeviceToken:
		// 400 BadDeviceToken：token 非法/与 environment 不匹配，本地删除。
		return errTokenStale
	default:
		// 403（provider token 配置错误）、429、5xx 等：记录并跳过。
		// 按设计不做自动重试（避免重复投递），站内红点兜底。
		slog.Warn("nativepush: unexpected apns response", "status", resp.StatusCode, "reason", resp.Reason)
		return fmt.Errorf("apns rejected notification: status=%d reason=%s", resp.StatusCode, resp.Reason)
	}
}

// fcmTokenSource 读取 service-account JSON 并换取 FCM OAuth2 access token。
func fcmTokenSource(ctx context.Context, cfg FCMConfig) (string, error) {
	raw, err := os.ReadFile(cfg.CredentialsPath)
	if err != nil {
		return "", fmt.Errorf("read fcm credentials: %w", err)
	}
	conf, err := google.JWTConfigFromJSON(raw, fcmScope)
	if err != nil {
		return "", fmt.Errorf("parse fcm credentials: %w", err)
	}
	tok, err := conf.TokenSource(ctx).Token()
	if err != nil {
		return "", fmt.Errorf("fetch fcm access token: %w", err)
	}
	return tok.AccessToken, nil
}

// sendFCM 发送一条 FCM HTTP v1 消息（notification.title/body +
// data.route）。注册 token 失效（404）返回 errTokenStale 触发本地清理。
func sendFCM(ctx context.Context, accessToken string, cfg FCMConfig, deviceToken string, msg *nativePayload) error {
	body := map[string]any{
		"message": map[string]any{
			"token": deviceToken,
			"notification": map[string]string{
				"title": msg.Title,
				"body":  msg.Body,
			},
			"data": map[string]string{"route": msg.Route},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal fcm message: %w", err)
	}
	endpoint := fmt.Sprintf(fcmSendURLPattern, url.PathEscape(cfg.ProjectID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build fcm request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := fcmHTTPClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("fcm request failed: %w", err)
	}
	// 读取并关闭响应体（连接复用需要排空）。
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusNotFound:
		// 404 UNREGISTERED：注册 token 不再有效，本地删除。
		return errTokenStale
	default:
		// 400（message/凭据错误）、429、5xx：记录并跳过。
		slog.Warn("nativepush: unexpected fcm response", "status", resp.StatusCode)
		return nil
	}
}
