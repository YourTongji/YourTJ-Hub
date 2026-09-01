package hotdataserve

import (
	"strconv"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/cacheconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/transform"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/vo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

const (
	maxCachedTopicPage = 32
	topicListCacheTTL  = 5 * time.Second
)

var topicListSorts = [...]string{"latest", "hot", "popular", "new"}

type TopicSimpleVoPage struct {
	Topics  []*vo.TopicsSimpleVo
	HasNext bool
}

var topicSimpleVoCache = &localcache.Cache[TopicSimpleVoPage]{MaxEntries: cacheconfig.Current().TopicList}

func GetLatestTopicsSimpleVoPaginated(page int, sort string) TopicSimpleVoPage {
	page = normalizeTopicPage(page)
	sort = normalizeTopicSort(sort)
	if !shouldCacheTopicPage(page) {
		return loadLatestTopicsSimpleVoPaginated(page, sort)
	}
	key := latestTopicsCacheKey(sort, page)
	return topicSimpleVoCache.GetOrLoad(key, func() (TopicSimpleVoPage, error) {
		return loadLatestTopicsSimpleVoPaginated(page, sort), nil
	}, topicListCacheTTL)
}

func GetTopicsByCategorySimpleVo(categoryId uint64, sort string, page int) TopicSimpleVoPage {
	page = normalizeTopicPage(page)
	sort = normalizeTopicSort(sort)
	if !shouldCacheTopicPage(page) {
		return loadTopicsByCategorySimpleVo(categoryId, sort, page)
	}
	key := topicsByCategoryCacheKey(categoryId, sort, page)
	return topicSimpleVoCache.GetOrLoad(key, func() (TopicSimpleVoPage, error) {
		return loadTopicsByCategorySimpleVo(categoryId, sort, page), nil
	}, topicListCacheTTL)
}

func normalizeTopicPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func normalizeTopicSort(sort string) string {
	switch sort {
	case "hot", "popular", "new":
		return sort
	default:
		return "latest"
	}
}

func shouldCacheTopicPage(page int) bool {
	return page <= maxCachedTopicPage
}

func latestTopicsCacheKey(sort string, page int) string {
	return "home:GetLatestTopics:" + sort + ":" + strconv.Itoa(page)
}

func topicsByCategoryCacheKey(categoryID uint64, sort string, page int) string {
	return "GetTopicsByCategory:" + strconv.FormatUint(categoryID, 10) + ":" + sort + ":" + strconv.Itoa(page)
}

func loadLatestTopicsSimpleVoPaginated(page int, sort string) TopicSimpleVoPage {
	res := topics.Page(topics.PageQuery{
		Page:         page,
		PageSize:     20,
		FilterStatus: true,
		Sort:         sort,
		TopicType:    topics.TopicTypePtr(topics.TopicTypeForum),
	})
	return TopicSimpleVoPage{
		Topics:  transform.Topics2Vo(topicEntitiesToPointers(res.Data), CategoryMap()),
		HasNext: res.HasNext,
	}
}

func loadTopicsByCategorySimpleVo(categoryId uint64, sort string, page int) TopicSimpleVoPage {
	res := topics.Page(topics.PageQuery{
		Page:         page,
		PageSize:     20,
		CategoryId:   categoryId,
		FilterStatus: true,
		Sort:         sort,
		TopicType:    topics.TopicTypePtr(topics.TopicTypeForum),
	})
	return TopicSimpleVoPage{
		Topics:  transform.Topics2Vo(topicEntitiesToPointers(res.Data), CategoryMap()),
		HasNext: res.HasNext,
	}
}

func topicEntitiesToPointers(data []topics.Entity) []*topics.Entity {
	res := make([]*topics.Entity, 0, len(data))
	for i := range data {
		res = append(res, &data[i])
	}
	return res
}

func ClearTopicListCache() {
	topicSimpleVoCache.Clear()
}

// InvalidateTopicListCacheForCategories invalidates home pages and category
// pages for categories touched by a topic mutation. Other category pages stay
// warm and are still protected from stale in-flight loads by localcache.
func InvalidateTopicListCacheForCategories(categoryIDs ...uint64) {
	for page := 1; page <= maxCachedTopicPage; page++ {
		for _, sort := range topicListSorts {
			topicSimpleVoCache.Delete(latestTopicsCacheKey(sort, page))
		}
	}

	seen := make(map[uint64]struct{}, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		if categoryID == 0 {
			continue
		}
		if _, ok := seen[categoryID]; ok {
			continue
		}
		seen[categoryID] = struct{}{}
		for page := 1; page <= maxCachedTopicPage; page++ {
			for _, sort := range topicListSorts {
				topicSimpleVoCache.Delete(topicsByCategoryCacheKey(categoryID, sort, page))
			}
		}
	}
}
