package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/transform"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/vo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
)

func withOwnPendingTopics(public []*vo.TopicsSimpleVo, userID uint64, page int) []*vo.TopicsSimpleVo {
	if userID == 0 {
		return public
	}
	candidates := topics.PendingByAuthor(userID, 30)
	contentTypes := map[uint64]int8{}
	for _, topic := range candidates {
		post := posts.Get(topic.FirstPostId)
		publicationservice.OwnerSnapshot(topic, &post, userID, false)
		contentTypes[topic.Id] = post.ContentType
	}
	private := transform.Topics2Vo(candidates, hotdataserve.CategoryMap())
	byID := map[uint64]*vo.TopicsSimpleVo{}
	for _, item := range private {
		item.ContentType = contentTypes[item.Id]
		byID[item.Id] = item
	}
	result := make([]*vo.TopicsSimpleVo, 0, len(public)+len(private))
	seen := map[uint64]bool{}
	// First-page private entries provide immediate feedback even when not indexed.
	if page == 1 {
		for _, item := range private {
			result = append(result, item)
			seen[item.Id] = true
		}
	}
	for _, item := range public {
		if item == nil || seen[item.Id] {
			continue
		}
		if candidate := byID[item.Id]; candidate != nil {
			result = append(result, candidate)
		} else {
			result = append(result, item)
		}
	}
	return result
}
