// Package webpushservice 实现服务端事件驱动的 Web Push 发送通道（issue #444
// 第二通道）：通知行（event_notification）创建成功后入 taskQueue outbox，
// 专用 worker 异步向该用户全部浏览器订阅发送系统推送。页面完全关闭也能收到；
// 推送是站内通知红点的 best-effort 外投，失败绝不影响通知落库与业务请求。
package webpushservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushSubscription"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
	"github.com/spf13/cast"
)

// TaskTypePush 是 webpush outbox 任务类型前缀。任务 Type 为
// "webpush.{notificationId}"，RunWorker 按前缀隔离领取。
const TaskTypePush = "webpush."

// pushHTTPClient 是发送用 HTTP 客户端：推送服务为第三方外部 I/O，
// 必须带超时与连接复用，不能使用无超时默认 client。
var pushHTTPClient = &http.Client{Timeout: 10 * time.Second}

// VAPID 密钥解码后的字节长度：私钥 32B、公钥 65B（P-256 未压缩点）。
const (
	vapidPrivateKeyLen = 32
	vapidPublicKeyLen  = 65
)

// VapidConfig 是本实例的 VAPID 密钥配置（[webpush] 段，ADR-006 config 治理）。
// 密钥按实例配置：dev 不配密钥 ⇒ 通道关闭；dev 从 main 快照同步的
// task_queue/push_subscriptions 数据也绝不会外发（worker 无密钥即 no-op）。
type VapidConfig struct {
	PublicKey  string
	PrivateKey string
}

// LoadVapidConfig 读取 [webpush] 配置段。
func LoadVapidConfig() VapidConfig {
	return VapidConfig{
		PublicKey:  strings.TrimSpace(preferences.GetString("webpush.vapid_public_key", "")),
		PrivateKey: strings.TrimSpace(preferences.GetString("webpush.vapid_private_key", "")),
	}
}

// Enabled 返回实例是否启用推送：公钥可解码为 65B P-256 点且私钥可解码为
// 32B 时启用。格式不符视为未配置（fail-closed）。
func (c VapidConfig) Enabled() bool {
	if c.PublicKey == "" || c.PrivateKey == "" {
		return false
	}
	pub, err := base64.RawURLEncoding.DecodeString(c.PublicKey)
	if err != nil || len(pub) != vapidPublicKeyLen {
		return false
	}
	priv, err := base64.RawURLEncoding.DecodeString(c.PrivateKey)
	return err == nil && len(priv) == vapidPrivateKeyLen
}

// PublicKeyOrEmpty 返回公钥（供 GET push/config 返回 applicationServerKey）。
// 未配置/格式不符时返回空串。
func PublicKeyOrEmpty() string {
	cfg := LoadVapidConfig()
	if !cfg.Enabled() {
		return ""
	}
	return cfg.PublicKey
}

// LogConfigStatus 在 serve 启动时输出推送通道状态；密钥已配置但格式非法时
// 告警（便于运维发现渲染/粘贴错误），并明确通道被禁用。
func LogConfigStatus() {
	cfg := LoadVapidConfig()
	hasKeys := cfg.PublicKey != "" || cfg.PrivateKey != ""
	if !hasKeys {
		slog.Info("webpush: disabled (no [webpush] VAPID keys configured)")
		return
	}
	if !cfg.Enabled() {
		slog.Warn("webpush: VAPID keys configured but invalid (need 43-char base64url private / 87-char public); channel disabled")
		return
	}
	slog.Info("webpush: enabled")
}

// PushTask 是 outbox 任务负载：定位一条通知行。
type PushTask struct {
	UserId         uint64 `json:"userId"`
	NotificationId uint64 `json:"notificationId"`
}

// EnqueueNotification 在通知行创建成功后入队推送任务。非事务写入：通知行已
// 独立提交，入队失败只丢推送（站内红点兜底），绝不影响调用方。实例未启用
// 推送时直接跳过（dev 不产生任务行）。
func EnqueueNotification(userId uint64, notificationId uint64) {
	if userId == 0 || notificationId == 0 {
		return
	}
	if !LoadVapidConfig().Enabled() {
		return
	}
	taskJSON, err := json.Marshal(PushTask{UserId: userId, NotificationId: notificationId})
	if err != nil {
		slog.Warn("webpush: marshal push task failed", "userId", userId, "notificationId", notificationId, "err", err)
		return
	}
	if err := taskQueue.Create(&taskQueue.Entity{
		Type:     TaskTypePush + cast.ToString(notificationId),
		Status:   taskQueue.StatusPending,
		TaskJson: string(taskJSON),
	}); err != nil {
		slog.Warn("webpush: enqueue push task failed", "userId", userId, "notificationId", notificationId, "err", err)
	}
}

// RecoverStaleTasks 启动时回收上次进程遗留的 Running 推送任务（崩溃恢复，
// 与搜索 worker 同款）。
func RecoverStaleTasks() error {
	return taskQueue.RecoverStaleRunning(TaskTypePush, taskQueue.LeaseDuration)
}

// RunPushTask 是推送 worker 的任务处理函数（backgroundservice.RunWorker 语义）：
// 任务整体按 Success 收尾（attempted 语义，不依赖 taskQueue 重试——部分订阅
// 成功后重试会造成重复投递）；单订阅发送失败仅日志，由站内红点兜底。
func RunPushTask(ctx context.Context, task *taskQueue.Entity) error {
	var payload PushTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		// 负载无法解析属于不可恢复数据错误：Success 收尾避免无限重试。
		slog.Warn("webpush: malformed push task", "taskId", task.Id, "err", err)
		return nil
	}
	if payload.UserId == 0 || payload.NotificationId == 0 {
		return nil
	}

	// 实例未启用（dev 快照含 main 任务行时走此分支）：置 Success 防误发，
	// 同时避免任务积压。启动时已打过状态日志，此处静默。
	cfg := LoadVapidConfig()
	if !cfg.Enabled() {
		slog.Debug("webpush: task skipped, channel disabled", "taskId", task.Id)
		return nil
	}

	notification := eventNotification.GetByID(payload.NotificationId)
	if notification.Id == 0 {
		// 通知行不存在（已删/迁移丢失）：无事件源，Success 跳过。
		slog.Debug("webpush: notification gone, skip", "taskId", task.Id, "notificationId", payload.NotificationId)
		return nil
	}
	// 已读通知不再推送：用户已看到，避免冗余打扰（已读与 worker 消费间的
	// 竞态窗口产生的冗余为可接受噪声）。
	if notification.IsRead {
		return nil
	}
	// 注销账号（软删）不再推送。
	if users.IsAccountClosed(payload.UserId) {
		return nil
	}

	subs := pushSubscription.ListByUser(payload.UserId)
	if len(subs) == 0 {
		return nil
	}

	var sent, deleted int
	for _, sub := range subs {
		if sub == nil || sub.Endpoint == "" || sub.P256dh == "" || sub.Auth == "" {
			continue
		}
		// 语言是订阅级属性（同一用户的不同浏览器可能用不同界面语言）。
		content := buildPushContent(notification, normalizeLang(sub.Lang))
		if content == nil {
			continue
		}
		opts := &webpush.Options{
			Subscriber:      vapidSubscriber(),
			VAPIDPublicKey:  cfg.PublicKey,
			VAPIDPrivateKey: cfg.PrivateKey,
			TTL:             24 * 60 * 60, // 24h：离线覆盖优先（TTL 0=立即过期）
			AuthScheme:      authSchemeFor(sub.Endpoint),
			HTTPClient:      pushHTTPClient,
		}
		msg, err := json.Marshal(content)
		if err != nil {
			continue
		}
		resp, err := webpush.SendNotificationWithContext(ctx, msg, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{Auth: sub.Auth, P256dh: sub.P256dh},
		}, opts)
		if err != nil {
			slog.Warn("webpush: send failed", "userId", payload.UserId, "endpoint", redactEndpoint(sub.Endpoint), "err", err)
			continue
		}
		// 读取并关闭响应体（连接复用需要排空）。
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			sent++
			if err := pushSubscription.TouchActive(sub.Endpoint, time.Now()); err != nil {
				slog.Warn("webpush: touch subscription failed", "endpoint", redactEndpoint(sub.Endpoint), "err", err)
			}
		case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
			// 404/410：订阅在推送服务已失效（含 Chrome 270 天陈旧策略），
			// 从本地删除，停止向死 endpoint 发送。
			deleted++
			if err := pushSubscription.DeleteByEndpoint(sub.Endpoint); err != nil {
				slog.Warn("webpush: delete stale subscription failed", "endpoint", redactEndpoint(sub.Endpoint), "err", err)
			}
			slog.Info("webpush: subscription removed (endpoint gone)", "userId", payload.UserId)
		default:
			// 401/403（VAPID 配置错误）、429/5xx（限流/服务端）等：记录并跳过。
			// 按设计不做自动重试（避免重复投递），站内红点兜底。
			slog.Warn("webpush: unexpected push response", "userId", payload.UserId, "endpoint", redactEndpoint(sub.Endpoint), "status", resp.StatusCode)
		}
	}
	slog.Debug("webpush: task done", "taskId", task.Id, "userId", payload.UserId, "sent", sent, "deleted", deleted)
	return nil
}

// vapidSubscriber 返回 VAPID JWT 的 sub：优先实例 server.url（https），
// 否则退回 mailto 形式（webpush-go 自动补 mailto: 前缀）。
func vapidSubscriber() string {
	serverURL := strings.TrimSpace(preferences.GetString("server.url", ""))
	if strings.HasPrefix(serverURL, "https://") {
		return serverURL
	}
	return "no-reply@localhost"
}

// authSchemeFor 按推送端点域名选择 VAPID 认证 scheme：Chrome 的 FCM 端点
// 使用 WebPush scheme（独立 Crypto-Key 头），Firefox/Apple 走 RFC 8292 的
// vapid scheme。webpush-go 需 ≥2026-04 提交才支持 WebPush。
func authSchemeFor(endpoint string) webpush.AuthScheme {
	if strings.Contains(endpoint, "fcm.googleapis.com") {
		return webpush.WebPush
	}
	return webpush.Vapid
}

// redactEndpoint 把 endpoint 脱敏成 host 前缀，避免完整推送 URL（含订阅
// 标识）进日志。
func redactEndpoint(endpoint string) string {
	trimmed := strings.TrimPrefix(endpoint, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	if i := strings.IndexByte(trimmed, '/'); i >= 0 {
		trimmed = trimmed[:i]
	}
	if len(trimmed) > 24 {
		trimmed = trimmed[:24]
	}
	return trimmed
}

// pushContent 是下发给 Service Worker 的展示负载。
type pushContent struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Icon  string `json:"icon"`
}

const pushIconPath = "/static/pic/icon_300.webp"

// buildPushContent 按通知类型 × 语言渲染标题/正文/深链。服务端渲染短文案，
// 绝不携带正文预览（内容删除联动后仍可能残留于已入队消息，隐私上不可接受）。
// 返回 nil 表示该类型无可推送文案（未知类型）。
func buildPushContent(notification eventNotification.Entity, lang string) *pushContent {
	eventType := notification.EventType
	body := bodyText(lang, eventType)
	if body == "" {
		return nil
	}

	payload := notification.Payload
	title := ""
	url := ""

	// 深链：wiki 页 → ProfileURL；话题类 → 帖子详情（楼层号稳定优先）；
	// 关注 → 用户主页；徽章/其余 → 通知中心。
	switch eventType {
	case eventNotification.EventTypeWikiUpdated:
		url = payload.Extra.ProfileURL
		title = truncateTitle(payload.Title)
	case eventNotification.EventTypeFollow:
		url = urlconfig.User(payload.ActorId)
		if payload.Extra.FollowerName != "" {
			title = truncateTitle(payload.Extra.FollowerName)
		} else {
			title = truncateTitle(actorName(payload))
		}
	case eventNotification.EventTypeComment, eventNotification.EventTypePostReply,
		eventNotification.EventTypeTopicPost, eventNotification.EventTypeLike:
		url = topicURL(payload)
		title = truncateTitle(topicTitle(payload))
	default:
		url = urlconfig.Notifications()
	}

	if url == "" {
		url = urlconfig.Notifications()
	}
	if title == "" {
		title = genericTitle(lang)
	}

	// 徽章 body 含 {badge} 占位符，用徽章名替换。
	if eventType == eventNotification.EventTypeBadge {
		name := payload.Extra.BadgeName
		if name == "" {
			name = payload.Extra.BadgeCode
		}
		if name == "" {
			name = genericTitle(lang)
		}
		body = strings.ReplaceAll(body, "{badge}", name)
	}

	return &pushContent{Title: title, Body: body, URL: url, Icon: pushIconPath}
}

// topicURL 构造帖子详情深链（与通知列表 BuildNotificationPayload 同规则）：
// /p/post/{topicId}[/{postNo}] 或回退 #post-{postId}。
func topicURL(payload eventNotification.NotificationPayload) string {
	if payload.TopicId == 0 {
		return ""
	}
	base := urlconfig.PostDetail(payload.TopicId)
	if payload.PostNo > 0 {
		return fmt.Sprintf("%s/%d", base, payload.PostNo)
	}
	if payload.PostId > 0 {
		return fmt.Sprintf("%s#post-%d", base, payload.PostId)
	}
	return base
}

// topicTitle 返回通知 payload 的话题标题（写入时多数类型不携带，需要时
// best-effort 查 topics 表补全；查不到返回空串，由调用方回落通用标题）。
func topicTitle(payload eventNotification.NotificationPayload) string {
	if payload.TopicTitle != "" {
		return payload.TopicTitle
	}
	if payload.TopicId == 0 {
		return ""
	}
	topicMap, err := topics.GetMapByIds([]uint64{payload.TopicId})
	if err != nil {
		return ""
	}
	if topic, ok := topicMap[payload.TopicId]; ok {
		return topic.Title
	}
	return ""
}

// actorName 返回通知触发者用户名（follow 通知标题用）；查不到返回空串。
func actorName(payload eventNotification.NotificationPayload) string {
	if payload.ActorId == 0 {
		return ""
	}
	if payload.ActorName != "" {
		return payload.ActorName
	}
	userMap := users.GetMapByIds([]uint64{payload.ActorId})
	if user, ok := userMap[payload.ActorId]; ok {
		return user.Username
	}
	return ""
}

// truncateTitle 把标题截断到 80 rune（推送标题过长会折行或被系统截断）。
func truncateTitle(title string) string {
	if title == "" {
		return ""
	}
	runes := []rune(title)
	if len(runes) > 80 {
		return string(runes[:80]) + "…"
	}
	return title
}

// normalizeLang 把订阅语言收敛到服务端文案支持的四语言；未知回落 zh。
func normalizeLang(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "zh", "en", "ja", "it":
		return strings.ToLower(strings.TrimSpace(lang))
	default:
		return "zh"
	}
}
