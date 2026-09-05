package webpushservice

import (
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
)

// 文案表完整性：4 语言 × 全部 7 事件类型的 body 均非空；badge 文案必须保留
// {badge} 占位符（发送前用徽章名替换）；genericTitle 4 语言非空。
func TestCopyTableComplete(t *testing.T) {
	langs := []string{"zh", "en", "ja", "de"}
	eventTypes := []string{
		eventNotification.EventTypeComment,
		eventNotification.EventTypePostReply,
		eventNotification.EventTypeTopicPost,
		eventNotification.EventTypeFollow,
		eventNotification.EventTypeBadge,
		eventNotification.EventTypeLike,
		eventNotification.EventTypeWikiUpdated,
	}
	for _, lang := range langs {
		if genericTitle(lang) == "" {
			t.Errorf("genericTitle(%q) is empty", lang)
		}
		for _, eventType := range eventTypes {
			body := bodyText(lang, eventType)
			if body == "" {
				t.Errorf("bodyText(%q, %q) is empty", lang, eventType)
			}
			if eventType == eventNotification.EventTypeBadge && !strings.Contains(body, "{badge}") {
				t.Errorf("badge body(%q) missing {badge} placeholder: %q", lang, body)
			}
		}
	}
}

// 未知语言回落 zh；zh/en/ja/de 原样保留（大小写不敏感、接受短码）。
func TestNormalizeLangFallback(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"zh", "zh"},
		{"en", "en"},
		{"ja", "ja"},
		{"de", "de"},
		{"EN", "en"},
		{"ZH", "zh"},
		{"", "zh"},
		{"fr", "zh"},
		{"de-DE", "de"},
		{"it", "zh"},
	}
	for _, c := range cases {
		if got := normalizeLang(c.in); got != c.want {
			t.Errorf("normalizeLang(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// 未知事件类型与未归一化语言都没有文案：bodyText 返回空串，调用方据此跳过推送。
func TestBodyTextEmptyForUnknown(t *testing.T) {
	if got := bodyText("zh", "unknown_type"); got != "" {
		t.Errorf("unknown type body = %q, want empty", got)
	}
	if got := bodyText("fr", eventNotification.EventTypeComment); got != "" {
		t.Errorf("unhandled lang body = %q, want empty", got)
	}
}

func TestBuildPushContentComment(t *testing.T) {
	notification := eventNotification.Entity{
		EventType: eventNotification.EventTypeComment,
		Payload: eventNotification.NotificationPayload{
			TopicId:    1001,
			TopicTitle: "话题标题",
			PostId:     2001,
			PostNo:     3,
		},
	}
	content := buildPushContent(notification, "zh")
	if content == nil {
		t.Fatal("comment content is nil")
	}
	if content.Body != "评论了你的内容" {
		t.Errorf("comment body = %q, want zh comment copy", content.Body)
	}
	wantURL := urlconfig.PostDetail(1001) + "/3"
	if content.URL != wantURL {
		t.Errorf("comment url = %q, want %q", content.URL, wantURL)
	}
	if content.Title != "话题标题" {
		t.Errorf("comment title = %q, want payload TopicTitle", content.Title)
	}
	if content.Icon == "" {
		t.Error("comment icon is empty")
	}
}

// 标题超长截断到 80 rune + 省略号；PostNo 缺失时回退 postId 锚点。
func TestBuildPushContentTruncateAndAnchorFallback(t *testing.T) {
	notification := eventNotification.Entity{
		EventType: eventNotification.EventTypeComment,
		Payload: eventNotification.NotificationPayload{
			TopicId:    1001,
			TopicTitle: strings.Repeat("长标题", 50), // 100 rune
			PostId:     2001,
			PostNo:     0,
		},
	}
	content := buildPushContent(notification, "zh")
	if content == nil {
		t.Fatal("comment content is nil")
	}
	runes := []rune(content.Title)
	if len(runes) != 81 || !strings.HasSuffix(content.Title, "…") {
		t.Errorf("truncated title = %q (len %d), want 80 runes + …", content.Title, len(runes))
	}
	wantURL := urlconfig.PostDetail(1001) + "#post-2001"
	if content.URL != wantURL {
		t.Errorf("anchor fallback url = %q, want %q", content.URL, wantURL)
	}
}

func TestBuildPushContentWikiUsesProfileURL(t *testing.T) {
	notification := eventNotification.Entity{
		EventType: eventNotification.EventTypeWikiUpdated,
		Payload: eventNotification.NotificationPayload{
			Title: "《使用指南》更新",
			Extra: eventNotification.Extra{ProfileURL: "/wiki/guide/intro"},
		},
	}
	content := buildPushContent(notification, "zh")
	if content == nil {
		t.Fatal("wiki content is nil")
	}
	if content.URL != "/wiki/guide/intro" {
		t.Errorf("wiki url = %q, want payload ProfileURL", content.URL)
	}
	if content.Title != "《使用指南》更新" {
		t.Errorf("wiki title = %q, want payload Title", content.Title)
	}
}

func TestBuildPushContentBadgeReplacesPlaceholder(t *testing.T) {
	notification := eventNotification.Entity{
		EventType: eventNotification.EventTypeBadge,
		Payload: eventNotification.NotificationPayload{
			Extra: eventNotification.Extra{BadgeName: "灌水大师"},
		},
	}
	content := buildPushContent(notification, "zh")
	if content == nil {
		t.Fatal("badge content is nil")
	}
	if content.Body != "获得了「灌水大师」徽章" {
		t.Errorf("badge body = %q, want {badge} replaced with BadgeName", content.Body)
	}
	if strings.Contains(content.Body, "{badge}") {
		t.Errorf("badge body still contains placeholder: %q", content.Body)
	}
	if content.URL != urlconfig.Notifications() {
		t.Errorf("badge url = %q, want notifications page", content.URL)
	}
}

// 无文案的类型（system 未进文案表）不产出推送内容。
func TestBuildPushContentUnknownTypeNil(t *testing.T) {
	notification := eventNotification.Entity{EventType: eventNotification.EventTypeSystem}
	if content := buildPushContent(notification, "zh"); content != nil {
		t.Errorf("system content = %#v, want nil", content)
	}
}

// buildPushContent 期望归一化后的语言；未归一化语言无文案，返回 nil
// （归一化由 normalizeLang 在入队消费侧完成）。
func TestBuildPushContentUnnormalizedLangNil(t *testing.T) {
	notification := eventNotification.Entity{
		EventType: eventNotification.EventTypeComment,
		Payload: eventNotification.NotificationPayload{
			TopicId:    1001,
			TopicTitle: "话题标题",
			PostNo:     3,
		},
	}
	if content := buildPushContent(notification, "fr"); content != nil {
		t.Errorf("unhandled lang content = %#v, want nil", content)
	}
}
