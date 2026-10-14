package webpushservice

import "fmt"
import "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"

// PushCopy 是跨推送通道共享的通知文案负载：Web Push 与原生推送（APNs/FCM）
// 复用同一份标题/正文/深链，避免两处文案与回查逻辑漂移。URL 为 web 形态
// （/p/post/{topicId}…），原生通道消费时自行转换为移动端 app 路由。
type PushCopy struct {
	Title string
	Body  string
	URL   string
}

// BuildPushCopy 按通知类型 × 语言渲染共享推送文案（webpush 通道内部同样
// 走 buildPushContent，文案表只维护一份）。返回 nil 表示该类型无可推送文案。
func BuildPushCopy(notification eventNotification.Entity, lang string) *PushCopy {
	content := buildPushContent(notification, lang)
	if content == nil {
		return nil
	}
	return &PushCopy{Title: content.Title, Body: content.Body, URL: content.URL}
}

// NormalizeLang 把用户/订阅语言收敛到服务端文案支持的四语言。
// 导出供原生推送按账号 locale 取文案语言。
func NormalizeLang(lang string) string {
	return normalizeLang(lang)
}

// BuildChatCopy never reads message text or private contact aliases.
func BuildChatCopy(senderID uint64, lang string) *PushCopy {
	name := actorName(eventNotification.NotificationPayload{ActorId: senderID})
	fallback := map[string]string{"zh": "一位用户", "en": "A user", "ja": "ユーザー", "de": "Ein Nutzer"}
	if name == "" {
		name = fallback[lang]
	}
	body := map[string]string{"zh": "发来一条私信", "en": "sent you a private message", "ja": "からプライベートメッセージが届きました", "de": "hat dir eine private Nachricht gesendet"}[lang]
	return &PushCopy{Title: truncateTitle(name) + " " + body, Body: map[string]string{"zh": "打开消息阅读", "en": "Open Messages to read it", "ja": "メッセージを開いて読む", "de": "Öffne Nachrichten zum Lesen"}[lang], URL: fmt.Sprintf("/messages?userId=%d", senderID)}
}
