package unreadservice

import (
	"strconv"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/cacheconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
)

const statusTTL = 2 * time.Minute

var statusCache = localcache.Cache[Status]{MaxEntries: cacheconfig.Current().UnreadStatus}

type Status struct {
	Notifications          bool   `json:"notifications"`
	Messages               bool   `json:"messages"`
	LatestNotificationType string `json:"latestNotificationType,omitempty"`
	// LatestUnreadId 最新一条未读通知的 id：新通知使其前移、已读清零后置 0。
	// 它是单调信号——轮询型浏览器通知据此识别「未读期间又来了一条新的」，
	// 而不是只看「无未读→有未读」布尔翻转（issue #444 review）。
	LatestUnreadId uint64 `json:"latestUnreadId,omitempty"`
}

func GetStatus(userID uint64) Status {
	if userID == 0 {
		return Status{}
	}
	return statusCache.GetOrLoad(cacheKey(userID), func() (Status, error) {
		return loadStatus(userID), nil
	}, statusTTL)
}

func Invalidate(userID uint64) {
	if userID == 0 {
		return
	}
	statusCache.Delete(cacheKey(userID))
}

func loadStatus(userID uint64) Status {
	latest := eventNotification.GetLastUnread(userID)
	return Status{
		Notifications:          latest.Id != 0,
		Messages:               imUserChatConfigs.HasUnread(userID),
		LatestNotificationType: latest.EventType,
		LatestUnreadId:         latest.Id,
	}
}

func cacheKey(userID uint64) string {
	return "user:unread:status:" + strconv.FormatUint(userID, 10)
}
