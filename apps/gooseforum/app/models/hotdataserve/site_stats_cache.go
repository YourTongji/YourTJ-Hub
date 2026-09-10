package hotdataserve

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/cacheconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/vo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/dailyStats"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/samber/lo"
)

const siteStatsCacheTTL = 5 * time.Second

var siteStatisticsDataCache = &localcache.Cache[*vo.SiteStats]{MaxEntries: cacheconfig.Current().SiteStatistics}

func GetSiteStatisticsData() *vo.SiteStats {
	return siteStatisticsDataCache.GetOrLoad("", func() (*vo.SiteStats, error) {
		res := GetFriendLinksConfigCache()
		linksCount := lo.SumBy(res, func(group pageConfig.FriendLinksGroup) int {
			return len(group.Links)
		})
		return &vo.SiteStats{
			UserCount:       users.GetMaxId(),
			UserMonthCount:  dailyStats.GetCurrentMonthSum(dailyStats.StatTypeRegCount),
			TopicMaxID:      topics.GetMaxId(),
			TopicMonthCount: dailyStats.GetCurrentMonthSum(dailyStats.StatTypeTopicCount),
			PostMaxID:       posts.GetMaxId(),
			LinksCount:      linksCount,
		}, nil
	}, siteStatsCacheTTL)
}
