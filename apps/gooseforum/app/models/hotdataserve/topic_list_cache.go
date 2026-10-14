package hotdataserve

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"strconv"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/cacheconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/transform"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/vo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
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

// GetLatestTopicsSimpleVoPaginated returns the shared public home list. A
// viewer with any block relation gets a viewer-scoped query that bypasses the
// shared cache; everyone else keeps the cached, viewer-independent list.
func GetLatestTopicsSimpleVoPaginated(viewerID uint64, page int, sort string) TopicSimpleVoPage {
	page = normalizeTopicPage(page)
	sort = normalizeTopicSort(sort)
	if !shouldCacheTopicPage(page) || viewerHasBlocks(viewerID) {
		return loadLatestTopicsSimpleVoPaginated(viewerID, page, sort)
	}
	key := latestTopicsCacheKey(sort, page)
	return topicSimpleVoCache.GetOrLoad(key, func() (TopicSimpleVoPage, error) {
		return loadLatestTopicsSimpleVoPaginated(0, page, sort), nil
	}, topicListCacheTTL)
}

// GetTopicsByCategorySimpleVo mirrors GetLatestTopicsSimpleVoPaginated for
// category pages: viewers with a block relation bypass the shared cache.
func GetTopicsByCategorySimpleVo(viewerID uint64, categoryId uint64, sort string, page int) TopicSimpleVoPage {
	page = normalizeTopicPage(page)
	sort = normalizeTopicSort(sort)
	if !shouldCacheTopicPage(page) || viewerHasBlocks(viewerID) {
		return loadTopicsByCategorySimpleVo(viewerID, categoryId, sort, page)
	}
	key := topicsByCategoryCacheKey(categoryId, sort, page)
	return topicSimpleVoCache.GetOrLoad(key, func() (TopicSimpleVoPage, error) {
		return loadTopicsByCategorySimpleVo(0, categoryId, sort, page), nil
	}, topicListCacheTTL)
}

// viewerHasBlocks decides whether a public list request needs the viewer-scoped
// path. The per-viewer answer is cached by the users model for a short TTL so a
// shared-cache hit performs no per-request existence query; block writes
// invalidate it, so changes still apply immediately. A storage failure fails
// open (shared cache): lists are public content, a brief unfiltered window is
// preferable to a blank page, and the failure is never cached as filtered
// output.
func viewerHasBlocks(viewerID uint64) bool {
	if viewerID == 0 {
		return false
	}
	has, err := users.HasBlockedPeersCached(viewerID)
	if err != nil {
		return false
	}
	return has
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
	return strconv.FormatBool(feedconfig.RankReady()) + ":" + feedconfig.Current().RankHash + ":" + "home:GetLatestTopics:" + sort + ":" + strconv.Itoa(page)
}

func topicsByCategoryCacheKey(categoryID uint64, sort string, page int) string {
	return strconv.FormatBool(feedconfig.RankReady()) + ":" + feedconfig.Current().RankHash + ":" + "GetTopicsByCategory:" + strconv.FormatUint(categoryID, 10) + ":" + sort + ":" + strconv.Itoa(page)
}

func loadLatestTopicsSimpleVoPaginated(viewerID uint64, page int, sort string) TopicSimpleVoPage {
	res := topics.Page(topics.PageQuery{
		Page:                   page,
		PageSize:               20,
		FilterStatus:           true,
		Sort:                   sort,
		TopicType:              topics.TopicTypePtr(topics.TopicTypeForum),
		ExcludeBlockedViewerID: viewerID,
	})
	return TopicSimpleVoPage{
		Topics:  transform.Topics2Vo(topicEntitiesToPointers(res.Data), CategoryMap()),
		HasNext: res.HasNext,
	}
}

func loadTopicsByCategorySimpleVo(viewerID uint64, categoryId uint64, sort string, page int) TopicSimpleVoPage {
	res := topics.Page(topics.PageQuery{
		Page:                   page,
		PageSize:               20,
		CategoryId:             categoryId,
		FilterStatus:           true,
		Sort:                   sort,
		TopicType:              topics.TopicTypePtr(topics.TopicTypeForum),
		ExcludeBlockedViewerID: viewerID,
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
