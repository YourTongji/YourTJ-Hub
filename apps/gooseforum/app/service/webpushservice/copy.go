package webpushservice

// 服务端推送文案表：按通知事件类型 × 订阅语言渲染推送 body 与通用标题。
// 文案与前端通知模板（resource/src/locales/{zh,en,ja,de}.ts → notifications.templates.*）
// 语义对齐，但服务端独立维护（SW 零逻辑，不依赖前端 i18n 包）。
// 未知语言回落 zh；未知类型返回空串（调用方跳过推送）。

// bodyByLangType 推送正文（body）。badge 文案含 {badge} 占位符，
// 发送前用徽章名替换。
var bodyByLangType = map[string]map[string]string{
	"zh": {
		"comment":         "评论了你的内容",
		"post_reply":      "回复了你",
		"topic_post":      "在你关注的内容下发表了新回复",
		"follow":          "关注了你",
		"badge":           "获得了「{badge}」徽章",
		"like":            "赞了你的回复",
		"wiki_updated":    "更新了你订阅的 wiki 页面",
		"mention":         "提到了你",
		"system":          "系统管理提醒",
		"review_pending":  "你的内容正在等待人工审核",
		"review_approved": "你的内容已通过审核，现在所有人可见",
		"review_rejected": "本次提交未通过审核，可在内容管理中修改重提",
	},
	"en": {
		"comment":         "commented on your topic",
		"post_reply":      "replied to you",
		"topic_post":      "posted in a topic you watch",
		"follow":          "followed you",
		"badge":           "earned the \"{badge}\" badge",
		"like":            "liked your reply",
		"wiki_updated":    "updated a wiki page you are watching",
		"mention":         "mentioned you",
		"system":          "Admin alert",
		"review_pending":  "Your post is awaiting manual review",
		"review_approved": "Your post was approved and is now visible to everyone",
		"review_rejected": "Your submission wasn’t approved. Edit and resubmit it from content management.",
	},
	"ja": {
		"comment":         "あなたのトピックにコメントしました",
		"post_reply":      "あなたに返信しました",
		"topic_post":      "ウォッチ中のトピックに投稿しました",
		"follow":          "あなたをフォローしました",
		"badge":           "「{badge}」バッジを獲得しました",
		"like":            "あなたの返信にいいねしました",
		"wiki_updated":    "ウォッチ中の wiki ページが更新されました",
		"mention":         "あなたをメンションしました",
		"system":          "管理者向けアラート",
		"review_pending":  "投稿は手動確認を待っています",
		"review_approved": "投稿が承認され、すべての人に表示されるようになりました",
		"review_rejected": "今回の投稿は承認されませんでした。コンテンツ管理から修正して再送信できます。",
	},
	"de": {
		"comment":         "hat dein Thema kommentiert",
		"post_reply":      "hat dir geantwortet",
		"topic_post":      "hat in einem Thema gepostet, dem du folgst",
		"follow":          "folgt dir jetzt",
		"badge":           "hat das Abzeichen \"{badge}\" erhalten",
		"like":            "hat deine Antwort mit \"Gefällt mir\" markiert",
		"wiki_updated":    "hat eine Wiki-Seite aktualisiert, der du folgst",
		"mention":         "hat dich erwähnt",
		"system":          "Admin-Warnung",
		"review_pending":  "Dein Beitrag wartet auf manuelle Prüfung",
		"review_approved": "Dein Beitrag wurde freigegeben und ist jetzt für alle sichtbar",
		"review_rejected": "Deine Einreichung wurde nicht freigegeben. Du kannst sie in der Inhaltsverwaltung bearbeiten und erneut einreichen.",
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
