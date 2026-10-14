package nativepushservice

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/webpushservice"
)

// nativePayload 是下发给移动端 App 的展示负载：title/body 为推送文案，
// route 为 App 内路由（如 /p/123）。文案复用 webpushservice 的共享文案表
// （同一事件 × 语言只维护一份），route 由 web 形态深链转换为 App 路由。
type nativePayload struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	Route       string `json:"route"`
	SourceKind  string `json:"sourceKind"`
	SourceId    uint64 `json:"sourceId"`
	RecipientId uint64 `json:"recipientId"`
	Channel     string `json:"channel"`
	Thread      string `json:"thread"`
}

// mobileFallbackRoute 是无法映射到 App 路由时的兜底页（通知中心）。
const mobileFallbackRoute = "/notifications"

// buildNativePayload 按通知类型 × 语言渲染原生推送负载。返回 nil 表示该类型
// 无可推送文案（未知类型），调用方跳过该用户。
func buildNativePayload(notification eventNotification.Entity, locale string) *nativePayload {
	copy := webpushservice.BuildPushCopy(notification, webpushservice.NormalizeLang(locale))
	if copy == nil {
		return nil
	}
	return &nativePayload{
		Title: copy.Title,
		Body:  copy.Body,
		Route: webRouteToMobile(copy.URL),
	}
}

// webRouteToMobile 把 web 深链（/p/post/{topicId}[/{postNo}]、/u/{userId}、
// /wiki/...）映射为移动端 App 路由（router.dart：/p/:postId、/u/:userId；
// wiki 保留合法页面路径，未知路径回落通知中心）。楼层号复用 TopicPage 的 postNo 查询参数；
// 只有旧 postId 锚点而没有楼层号时回退话题。
func webRouteToMobile(webURL string) string {
	path := strings.TrimSpace(webURL)
	parsed, err := url.Parse(path)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return mobileFallbackRoute
	}
	if strings.Contains(path, "\\") || strings.IndexFunc(parsed.Path, func(r rune) bool { return r <= 0x20 || r == 0x7f }) >= 0 {
		return mobileFallbackRoute
	}
	if parsed.Path == "/chat" || parsed.Path == "/messages" {
		id, err := strconv.ParseUint(parsed.Query().Get("userId"), 10, 64)
		if err == nil && id > 0 {
			return "/chat?userId=" + strconv.FormatUint(id, 10)
		}
		return mobileFallbackRoute
	}
	if strings.HasPrefix(parsed.Path, "/wiki/") && parsed.RawQuery == "" && parsed.Fragment == "" {
		// RawPath keeps an encoded separator that Path decoding already turned
		// into a real one; reject it before splitting into segments.
		if strings.Contains(strings.ToLower(parsed.RawPath), "%2f") {
			return mobileFallbackRoute
		}
		for _, part := range strings.Split(strings.TrimPrefix(parsed.Path, "/wiki/"), "/") {
			if part == "" || part == "." || part == ".." || strings.ContainsAny(part, "\\?#%") {
				return mobileFallbackRoute
			}
		}
		return parsed.EscapedPath()
	}
	if path == "/settings?tab=content" {
		return "/my-content"
	}
	if path == "" {
		return mobileFallbackRoute
	}
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	segs := strings.Split(strings.Trim(path, "/"), "/")
	if len(segs) >= 2 {
		switch segs[0] {
		case "p":
			if segs[1] == "post" && len(segs) >= 3 {
				return mobileTopicRoute(segs[2:])
			}
		case "topics":
			return mobileTopicRoute(segs[1:])
		case "u":
			id, err := strconv.ParseUint(segs[1], 10, 64)
			if err != nil || id == 0 {
				return mobileFallbackRoute
			}
			return "/u/" + strconv.FormatUint(id, 10)
		}
	}
	return mobileFallbackRoute
}

func mobileTopicRoute(segments []string) string {
	id, err := strconv.ParseUint(segments[0], 10, 64)
	if err != nil || id == 0 {
		return mobileFallbackRoute
	}
	route := "/p/" + strconv.FormatUint(id, 10)
	if len(segments) > 1 {
		if floor, err := strconv.ParseUint(segments[1], 10, 64); err == nil && floor > 0 {
			route += "?postNo=" + strconv.FormatUint(floor, 10)
		}
	}
	return route
}

func (p *nativePayload) Data() map[string]string {
	return map[string]string{"route": p.Route, "sourceKind": p.SourceKind, "sourceId": fmt.Sprint(p.SourceId), "recipientId": fmt.Sprint(p.RecipientId), "channel": p.Channel, "thread": p.Thread}
}
func channelFor(group string) string {
	switch group {
	case "chat":
		return "yourtj_chat_v2"
	case "mentionsAndReplies":
		return "yourtj_mentions_and_replies_v2"
	case "system":
		return "yourtj_system_v2"
	default:
		return "yourtj_activity_v2"
	}
}
