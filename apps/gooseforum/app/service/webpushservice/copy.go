package webpushservice

// 服务端推送文案表：按通知事件类型 × 订阅语言渲染推送 body 与通用标题。
// 文案与前端通知模板（resource/src/locales/{zh,en,ja,de}.ts → notifications.templates.*）
// 语义对齐，但服务端独立维护（SW 零逻辑，不依赖前端 i18n 包）。
// 未知语言回落 zh；未知类型返回空串（调用方跳过推送）。

// bodyByLangType 推送正文（body）。badge 文案含 {badge} 占位符，
// 发送前用徽章名替换。
var bodyByLangType = map[string]map[string]string{
	"zh": {
		"comment":      "评论了你的内容",
		"post_reply":   "回复了你",
		"topic_post":   "在你关注的内容下发表了新回复",
		"follow":       "关注了你",
		"badge":        "获得了「{badge}」徽章",
		"like":         "赞了你的回复",
		"wiki_updated": "更新了你订阅的 wiki 页面",
	},
	"en": {
		"comment":      "commented on your topic",
		"post_reply":   "replied to you",
		"topic_post":   "posted in a topic you watch",
		"follow":       "followed you",
		"badge":        "earned the \"{badge}\" badge",
		"like":         "liked your reply",
		"wiki_updated": "updated a wiki page you are watching",
	},
	"ja": {
		"comment":      "あなたのトピックにコメントしました",
		"post_reply":   "あなたに返信しました",
		"topic_post":   "ウォッチ中のトピックに投稿しました",
		"follow":       "あなたをフォローしました",
		"badge":        "「{badge}」バッジを獲得しました",
		"like":         "あなたの返信にいいねしました",
		"wiki_updated": "ウォッチ中の wiki ページが更新されました",
	},
	"de": {
		"comment":      "hat dein Thema kommentiert",
		"post_reply":   "hat dir geantwortet",
		"topic_post":   "hat in einem Thema gepostet, dem du folgst",
		"follow":       "folgt dir jetzt",
		"badge":        "hat das Abzeichen \"{badge}\" erhalten",
		"like":         "hat deine Antwort mit \"Gefällt mir\" markiert",
		"wiki_updated": "hat eine Wiki-Seite aktualisiert, der du folgst",
	},
}

// genericTitleByLang 无话题标题/无 actor 时的推送标题兜底
// （对齐 locales notifications.newNotification）。
var genericTitleByLang = map[string]string{
	"zh": "有新的通知",
	"en": "New notification",
	"ja": "新しい通知があります",
	"de": "Neue Benachrichtigung",
}

// bodyText 返回指定语言与事件类型的正文；未知类型返回空串。
func bodyText(lang string, eventType string) string {
	return bodyByLangType[lang][eventType]
}

// genericTitle 返回指定语言的通用推送标题。
func genericTitle(lang string) string {
	return genericTitleByLang[lang]
}
