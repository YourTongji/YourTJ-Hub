package nativepushservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
)

// buildNativePayload 按事件类型渲染标题/正文；route 为移动端 App 路由
// （/p/{topicId}），web 深链（/p/post/...）正确映射。
func TestBuildNativePayloadComment(t *testing.T) {
	notification := eventNotification.Entity{
		Id:        987654,
		EventType: eventNotification.EventTypeComment,
		Payload: eventNotification.NotificationPayload{
			TopicId:    1001,
			TopicTitle: "话题标题",
			PostId:     2001,
			PostNo:     3,
		},
	}
	msg := buildNativePayload(notification, "zh")
	if msg == nil {
		t.Fatal("comment payload is nil")
	}
	if msg.Body != "评论了你的内容" {
		t.Errorf("comment body = %q, want zh comment copy", msg.Body)
	}
	if msg.Title != "话题标题" {
		t.Errorf("comment title = %q, want payload TopicTitle", msg.Title)
	}
	// /p/post/1001/3（楼层深链）→ App /p/1001（楼层号无对应页面，收敛到话题级）。
	if msg.Route != "/p/1001" {
		t.Errorf("comment route = %q, want /p/1001", msg.Route)
	}
}

// 未知事件类型与未归一化语言：无文案，返回 nil（调用方跳过）。
// 未知事件类型返回 nil（调用方跳过）；未归一化语言回落 zh（与 webpush
// 文案表同款回落语义，不返回 nil——语言未知仍能推送，只是用默认文案）。
func TestBuildNativePayloadUnknownNil(t *testing.T) {
	notification := eventNotification.Entity{EventType: eventNotification.EventTypeSystem}
	if msg := buildNativePayload(notification, "zh"); msg != nil {
		t.Errorf("system payload = %#v, want nil", msg)
	}
	comment := eventNotification.Entity{
		EventType: eventNotification.EventTypeComment,
		Payload: eventNotification.NotificationPayload{
			TopicId:    1001,
			TopicTitle: "话题标题",
			PostNo:     3,
		},
	}
	if msg := buildNativePayload(comment, "fr"); msg == nil {
		t.Fatal("unknown-lang payload = nil, want zh fallback")
	} else if msg.Body != "评论了你的内容" {
		t.Errorf("unknown-lang body = %q, want zh fallback body", msg.Body)
	}
}

// wiki 通知深链（/wiki/...）无 App 对应页面：回落通知中心。
func TestBuildNativePayloadWikiFallback(t *testing.T) {
	notification := eventNotification.Entity{
		EventType: eventNotification.EventTypeWikiUpdated,
		Payload: eventNotification.NotificationPayload{
			Title: "《使用指南》更新",
			Extra: eventNotification.Extra{ProfileURL: "/wiki/guide/intro"},
		},
	}
	msg := buildNativePayload(notification, "zh")
	if msg == nil {
		t.Fatal("wiki payload is nil")
	}
	if msg.Title != "《使用指南》更新" {
		t.Errorf("wiki title = %q, want payload Title", msg.Title)
	}
	if msg.Route != mobileFallbackRoute {
		t.Errorf("wiki route = %q, want %q", msg.Route, mobileFallbackRoute)
	}
}

// webRouteToMobile 映射：话题/帖子/用户主页映射到 App 路由；wiki/未知/空回落。
func TestWebRouteToMobile(t *testing.T) {
	cases := []struct {
		webURL string
		want   string
	}{
		{"", mobileFallbackRoute},
		{"/p/post/1001", "/p/1001"},
		{"/p/post/1001/3", "/p/1001"},
		{"/p/post/1001#post-2001", "/p/1001"},
		{"/topics/42", "/p/42"},
		{"/topics/42/1", "/p/42"},
		{"/u/7", "/u/7"},
		{"/wiki/guide/intro", mobileFallbackRoute},
		{"/notifications", mobileFallbackRoute},
		{"https://forum.example.com/p/post/1001", mobileFallbackRoute}, // 全 URL 非站内路径
	}
	for _, c := range cases {
		if got := webRouteToMobile(c.webURL); got != c.want {
			t.Errorf("webRouteToMobile(%q) = %q, want %q", c.webURL, got, c.want)
		}
	}
}
