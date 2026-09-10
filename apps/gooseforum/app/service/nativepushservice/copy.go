package nativepushservice

import (
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/webpushservice"
)

// nativePayload 是下发给移动端 App 的展示负载：title/body 为推送文案，
// route 为 App 内路由（如 /p/123）。文案复用 webpushservice 的共享文案表
// （同一事件 × 语言只维护一份），route 由 web 形态深链转换为 App 路由。
type nativePayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Route string `json:"route"`
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
// App 尚无 wiki/楼层页，wiki 与未知路径回落通知中心）。楼层号与锚点在
// 移动端无对应页面，按话题级 /p/{topicId} 收敛。
func webRouteToMobile(webURL string) string {
	path := strings.TrimSpace(webURL)
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
				return "/p/" + segs[2]
			}
		case "topics":
			return "/p/" + segs[1]
		case "u":
			return "/u/" + segs[1]
		}
	}
	return mobileFallbackRoute
}
