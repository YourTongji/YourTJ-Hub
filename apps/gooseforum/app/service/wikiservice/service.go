package wikiservice

import (
	"log/slog"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/unreadservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

// ActionResult 通用动作成功响应（契约 {ok:true}）。
type ActionResult struct {
	Ok bool `json:"ok"`
}

// HasPageManagerPermission 判断用户是否拥有 PageManager（含 Admin）权限。
func HasPageManagerPermission(userId uint64) bool {
	if userId == 0 {
		return false
	}
	roleID, ok := userservice.GetUserRoleId(userId)
	if !ok {
		return false
	}
	return permission.CheckRole(roleID, permission.PageManager)
}

// wikiNotifyThrottleWindow 同页面 wiki_updated 通知的节流窗口（review 决策：
// 写即发布后每次编辑都是发布，watcher 会收到大量通知；窗口内只发首条）。
const wikiNotifyThrottleWindow = 10 * time.Minute

// notifyWatchersThrottled 节流后通知 watcher：窗口内已有通知则跳过本次。
func notifyWatchersThrottled(topicId uint64, pagePath string, title string, editorId uint64) {
	latest := eventNotification.GetLatestByTopicAndType(topicId, eventNotification.EventTypeWikiUpdated)
	if latest.Id != 0 && time.Since(latest.CreatedAt) < wikiNotifyThrottleWindow {
		return
	}
	notifyWatchers(topicId, pagePath, title, editorId)
}

// notifyWatchers 给全部 watcher 发送 wiki_updated 通知。
func notifyWatchers(topicId uint64, pagePath string, title string, editorId uint64) {
	after := uint64(0)
	for {
		watchers := topicUserAction.ListActiveWatchUserIDsAfter(topicId, after, nil, 500)
		if len(watchers) == 0 {
			return
		}
		notifications := make([]*eventNotification.Entity, 0, len(watchers))
		for _, userId := range watchers {
			if userId == editorId {
				continue
			}
			notifications = append(notifications, &eventNotification.Entity{
				UserId:    userId,
				EventType: eventNotification.EventTypeWikiUpdated,
				TopicID:   topicId,
				Payload: eventNotification.NotificationPayload{
					Title:       title,
					Content:     title,
					TemplateKey: eventNotification.TemplateWikiUpdated,
					TemplateParams: eventNotification.NotificationTemplateParams{
						Preview: title,
					},
					ActorId:    editorId,
					TopicId:    topicId,
					TopicTitle: title,
					Extra: eventNotification.Extra{
						ProfileURL: "/wiki/" + pagePath,
					},
				},
			})
		}
		if len(notifications) > 0 {
			if err := eventNotification.CreateBatch(notifications, 100); err != nil {
				slog.Warn("wiki: notify watchers failed", "topicId", topicId, "error", err)
			} else {
				for _, userId := range watchers {
					if userId != editorId {
						unreadservice.Invalidate(userId)
					}
				}
			}
		}
		after = watchers[len(watchers)-1]
		if len(watchers) < 500 {
			return
		}
	}
}
