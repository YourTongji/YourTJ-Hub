package webpushservice

import "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"

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
		"badge":           "你获得了「{badge}」徽章",
		"like":            "赞了你的回复",
		"wiki_updated":    "更新了你订阅的 wiki 页面",
		"mention":         "提到了你",
		"system":          "系统管理提醒",
		"review_pending":  "你的内容正在等待人工审核",
		"review_approved": "你的内容已通过审核，现在所有人可见",
		"review_rejected": "本次提交未通过审核，可在内容管理修改重提；符合条件的内容也可申诉",
	},
	"en": {
		"comment":         "commented on your topic",
		"post_reply":      "replied to you",
		"topic_post":      "posted in a topic you watch",
		"follow":          "followed you",
		"badge":           "You earned the \"{badge}\" badge",
		"like":            "liked your reply",
		"wiki_updated":    "updated a wiki page you are watching",
		"mention":         "mentioned you",
		"system":          "Admin alert",
		"review_pending":  "Your post is awaiting manual review",
		"review_approved": "Your post was approved and is now visible to everyone",
		"review_rejected": "Your submission wasn’t approved. Edit and resubmit it from content management; eligible items can also be appealed.",
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
		"review_rejected": "今回の投稿は承認されませんでした。コンテンツ管理から修正して再送信できます。対象の項目は異議も申し立てられます。",
	},
	"de": {
		"comment":         "hat dein Thema kommentiert",
		"post_reply":      "hat dir geantwortet",
		"topic_post":      "hat in einem Thema gepostet, dem du folgst",
		"follow":          "folgt dir jetzt",
		"badge":           "Du hast das Abzeichen \"{badge}\" erhalten",
		"like":            "hat deine Antwort mit \"Gefällt mir\" markiert",
		"wiki_updated":    "hat eine Wiki-Seite aktualisiert, der du folgst",
		"mention":         "hat dich erwähnt",
		"system":          "Admin-Warnung",
		"review_pending":  "Dein Beitrag wartet auf manuelle Prüfung",
		"review_approved": "Dein Beitrag wurde freigegeben und ist jetzt für alle sichtbar",
		"review_rejected": "Deine Einreichung wurde nicht freigegeben. In der Inhaltsverwaltung kannst du sie bearbeiten und erneut einreichen; für berechtigte Einträge ist auch ein Einspruch möglich.",
	},
}

// appealCopyByLang 审核申诉推送（#1114）：按通知模板区分，[0] 标题写结论，[1] 正文写下一步。
// 驳回/失效与审核拒绝一样不带内容标题（脱敏）。
var appealCopyByLang = map[string]map[string][2]string{
	"zh": {
		eventNotification.TemplateAppealSubmitted: {"申诉已提交", "版主复核后会通知你结果"},
		eventNotification.TemplateAppealApproved:  {"申诉通过，内容已公开", "现在所有人都能看到它"},
		eventNotification.TemplateAppealRejected:  {"申诉未通过", "版主的说明在内容管理里，可以修改后重新提交"},
		eventNotification.TemplateAppealStale:     {"申诉已失效", "内容在复核前有改动或已不可见，这次申诉不再处理"},
	},
	"en": {
		eventNotification.TemplateAppealSubmitted: {"Appeal submitted", "We’ll notify you when a moderator decides"},
		eventNotification.TemplateAppealApproved:  {"Appeal approved — your post is public", "Everyone can see it now"},
		eventNotification.TemplateAppealRejected:  {"Appeal not approved", "See the moderator’s note in content management. You can edit and resubmit."},
		eventNotification.TemplateAppealStale:     {"Appeal closed", "The post changed or became unavailable before review, so this appeal won’t be decided"},
	},
	"ja": {
		eventNotification.TemplateAppealSubmitted: {"異議申し立てを受け付けました", "モデレーターの判断が出たらお知らせします"},
		eventNotification.TemplateAppealApproved:  {"異議が認められ、投稿が公開されました", "すべての人に表示されるようになりました"},
		eventNotification.TemplateAppealRejected:  {"異議は認められませんでした", "モデレーターの説明はコンテンツ管理で確認できます。修正して再送信できます。"},
		eventNotification.TemplateAppealStale:     {"異議申し立ては無効になりました", "確認前に投稿が変更されたか表示できなくなったため、今回の申し立ては処理されません"},
	},
	"de": {
		eventNotification.TemplateAppealSubmitted: {"Einspruch eingereicht", "Wir benachrichtigen dich, sobald die Moderation entschieden hat"},
		eventNotification.TemplateAppealApproved:  {"Einspruch angenommen – dein Beitrag ist öffentlich", "Er ist jetzt für alle sichtbar"},
		eventNotification.TemplateAppealRejected:  {"Einspruch abgelehnt", "Die Begründung findest du in der Inhaltsverwaltung. Du kannst den Beitrag bearbeiten und erneut einreichen."},
		eventNotification.TemplateAppealStale:     {"Einspruch hinfällig", "Der Beitrag wurde vor der Prüfung geändert oder ist nicht mehr verfügbar, daher wird nicht entschieden"},
	},
}

// appealCopy returns the appeal title/body for a notification template, if it is one.
func appealCopy(lang, templateKey string) ([2]string, bool) {
	if !eventNotification.IsAppealTemplate(templateKey) {
		return [2]string{}, false
	}
	text, ok := appealCopyByLang[lang][templateKey]
	if !ok {
		text, ok = appealCopyByLang["zh"][templateKey]
	}
	return text, ok
}

// genericTitleByLang 无话题标题/无 actor 时的推送标题兜底
// （参考 locales notifications.fallback；服务端独立维护，文案不逐字相同）。
var genericTitleByLang = map[string]string{
	"zh": "你收到一条系统通知",
	"en": "You received a system notification",
	"ja": "システム通知が届きました",
	"de": "Du hast eine Systembenachrichtigung erhalten",
}

// bodyText 返回指定语言与事件类型的正文；未知类型返回空串。
func bodyText(lang string, eventType string) string {
	return bodyByLangType[lang][eventType]
}

// genericTitle 返回指定语言的通用推送标题。
func genericTitle(lang string) string {
	return genericTitleByLang[lang]
}
