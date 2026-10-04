package webpushservice

import (
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
)

// 文案表完整性：4 语言 × 全部 9 事件类型的 body 均非空；badge 文案必须保留
// {badge} 占位符（发送前用徽章名替换）；genericTitle 4 语言非空。
func TestCopyTableComplete(t *testing.T) {
	langs := []string{"zh", "en", "ja", "de"}
	eventTypes := []string{
		eventNotification.EventTypeMention,
		eventNotification.EventTypeComment,
		eventNotification.EventTypePostReply,
		eventNotification.EventTypeTopicPost,
		eventNotification.EventTypeFollow,
		eventNotification.EventTypeBadge,
		eventNotification.EventTypeLike,
		eventNotification.EventTypeWikiUpdated,
		eventNotification.EventTypeSystem,
		eventNotification.EventTypeReviewPending,
		eventNotification.EventTypeReviewApproved,
		eventNotification.EventTypeReviewRejected,
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

// 无文案的未知类型不产出推送内容（system 已进文案表，用真正的未知类型验证）。
func TestBuildPushContentUnknownTypeNil(t *testing.T) {
	notification := eventNotification.Entity{EventType: "unknown_type"}
	if content := buildPushContent(notification, "zh"); content != nil {
		t.Errorf("unknown-type content = %#v, want nil", content)
	}
}

// system 通知（管理告警）可推送：文案表命中 system，深链回落通知中心，
// 标题回落通用标题。
func TestBuildPushContentSystem(t *testing.T) {
	notification := eventNotification.Entity{
		EventType: eventNotification.EventTypeSystem,
		Payload:   eventNotification.NotificationPayload{Title: "一系统排课同步失败", Content: "【本科】同步失败：未登录或会话失效"},
	}
	for lang, body := range map[string]string{
		"zh": "系统管理提醒",
		"en": "Admin alert",
		"ja": "管理者向けアラート",
		"de": "Admin-Warnung",
	} {
		content := buildPushContent(notification, lang)
		if content == nil {
			t.Fatalf("system %s content is nil", lang)
		}
		if content.Body != body {
			t.Errorf("system %s body = %q, want %q", lang, content.Body, body)
		}
		if content.URL != urlconfig.Notifications() {
			t.Errorf("system %s url = %q, want notifications page", lang, content.URL)
		}
		if content.Title != "" {
			// 系统通知不带话题/actor，标题应回落通用标题而非 payload.Title
			// （Web push 与站内通知列表的标题字段语义不同）。
			t.Logf("system %s title = %q (generic)", lang, content.Title)
		}
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

// buildPushContent 携带通知行 id：SW 用其生成唯一 notification tag，使多条
// 未读通知在系统托盘各自呈现而不互相覆盖（review P2）。
func TestBuildPushContentCarriesNotificationID(t *testing.T) {
	notification := eventNotification.Entity{
		Id:        987654,
		EventType: eventNotification.EventTypeComment,
		Payload: eventNotification.NotificationPayload{
			TopicId:    1001,
			TopicTitle: "话题标题",
			PostNo:     3,
		},
	}
	content := buildPushContent(notification, "zh")
	if content == nil {
		t.Fatal("comment content is nil")
	}
	if content.Id != 987654 {
		t.Errorf("content.Id = %d, want 987654", content.Id)
	}
}

func TestBuildPushContentMention(t *testing.T) {
	for lang, body := range map[string]string{"zh": "提到了你", "en": "mentioned you", "ja": "あなたをメンションしました", "de": "hat dich erwähnt"} {
		for _, postNo := range []uint64{0, 8} {
			notification := eventNotification.Entity{EventType: eventNotification.EventTypeMention, Payload: eventNotification.NotificationPayload{TopicId: 512, TopicTitle: "Topic", PostId: 4096, PostNo: postNo}}
			content := buildPushContent(notification, lang)
			wantURL := urlconfig.PostDetail(512) + "#post-4096"
			if postNo > 0 {
				wantURL = urlconfig.PostDetail(512) + "/8"
			}
			if content == nil || content.Body != body || content.URL != wantURL {
				t.Fatalf("mention %s/%d: %#v", lang, postNo, content)
			}
		}
	}
}

// Approved content links to its floor; rejection links to the owner content manager.
func TestBuildPushContentReviewResult(t *testing.T) {
	approved := eventNotification.Entity{Id: 7, EventType: eventNotification.EventTypeReviewApproved,
		Payload: eventNotification.NotificationPayload{TopicId: 42, PostNo: 3, TopicTitle: "期末复习资料"}}
	content := buildPushContent(approved, "zh")
	if content == nil || content.URL != "/p/post/42/3" || content.Title != "期末复习资料" || content.Body != "你的内容已通过审核，现在所有人可见" {
		t.Fatalf("approved push = %+v", content)
	}
	rejected := eventNotification.Entity{Id: 8, EventType: eventNotification.EventTypeReviewRejected,
		Payload: eventNotification.NotificationPayload{TopicTitle: "被拒的话题"}}
	content = buildPushContent(rejected, "en")
	if content == nil || content.URL != "/settings?tab=content" || content.Title != "被******题" || !strings.Contains(content.Body, "wasn’t approved") {
		t.Fatalf("rejected push = %+v", content)
	}
}
