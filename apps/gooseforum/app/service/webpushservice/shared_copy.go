package webpushservice

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
