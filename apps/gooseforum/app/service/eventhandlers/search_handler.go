package eventhandlers

import (
	"context"

	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/searchservice"
)

// TopicPublishedEvent 主题发布事件
type TopicPublishedEvent struct {
	Topic     *topics.Entity
	FirstPost *posts.Entity
}

func (event *TopicPublishedEvent) Subject() (uint64, uint64, string) {
	if event == nil {
		return 0, 0, ""
	}
	if event.Topic != nil {
		return event.Topic.Id, event.Topic.UserId, event.Topic.Title
	}
	return 0, 0, ""
}

// handleTopicPublished 更新已发布主题搜索索引
func handleTopicPublished(ctx context.Context, event *TopicPublishedEvent) error {
	if event == nil {
		return nil
	}
	if event.Topic != nil {
		_, err := searchservice.BuildSingleTopicSearchDocument(event.Topic, event.FirstPost)
		return err
	}
	return nil
}

// TopicUpdatedEvent 主题更新事件
type TopicUpdatedEvent struct {
	Topic     *topics.Entity
	FirstPost *posts.Entity
}

// handleTopicUpdated 更新主题搜索索引
func handleTopicUpdated(ctx context.Context, event *TopicUpdatedEvent) error {
	if event == nil {
		return nil
	}
	if event.Topic != nil {
		_, err := searchservice.BuildSingleTopicSearchDocument(event.Topic, event.FirstPost)
		return err
	}
	return nil
}

// TopicDeletedEvent 主题删除事件
type TopicDeletedEvent struct {
	Topic *topics.Entity
}

func (event *TopicDeletedEvent) Subject() (uint64, uint64, string) {
	if event == nil {
		return 0, 0, ""
	}
	if event.Topic != nil {
		return event.Topic.Id, event.Topic.UserId, event.Topic.Title
	}
	return 0, 0, ""
}

// handleTopicDeleted 删除主题搜索索引
func handleTopicDeleted(ctx context.Context, event *TopicDeletedEvent) error {
	if event == nil || event.Topic == nil {
		return nil
	}
	_, err := searchservice.BuildSingleTopicSearchDocument(event.Topic, nil)
	return err
}

// UserSearchIndexUpdatedEvent 用户可搜字段变更（昵称/简介/用户名）后更新搜索索引
type UserSearchIndexUpdatedEvent struct {
	UserId uint64
}

func (event *UserSearchIndexUpdatedEvent) Subject() (uint64, uint64, string) {
	if event == nil {
		return 0, 0, ""
	}
	return event.UserId, 0, ""
}

// handleUserSearchIndexUpdated 根据用户当前状态 upsert 或删除搜索文档
func handleUserSearchIndexUpdated(ctx context.Context, event *UserSearchIndexUpdatedEvent) error {
	if event == nil || event.UserId == 0 {
		return nil
	}
	user, err := users.Get(event.UserId)
	if err != nil {
		// 用户不存在（如软删）→ 删除索引文档
		_, delErr := searchservice.DeleteUserSearchDocument(event.UserId)
		return delErr
	}
	_, err = searchservice.BuildSingleUserSearchDocument(&user)
	return err
}

// handleUserSignUpSearchIndex 注册事件后把新用户加入搜索索引
// （复用 UserSignUpEvent，自动覆盖 password/OAuth/OIDC 三条注册路径）
func handleUserSignUpSearchIndex(ctx context.Context, event *UserSignUpEvent) error {
	if event == nil || event.UserId == 0 {
		return nil
	}
	user, err := users.Get(event.UserId)
	if err != nil {
		return nil
	}
	_, err = searchservice.BuildSingleUserSearchDocument(&user)
	return err
}

// CategorySearchIndexUpdatedEvent 分类新增/更新后同步搜索索引
type CategorySearchIndexUpdatedEvent struct {
	CategoryId uint64
}

func (event *CategorySearchIndexUpdatedEvent) Subject() (uint64, uint64, string) {
	if event == nil {
		return 0, 0, ""
	}
	return event.CategoryId, 0, ""
}

// handleCategorySearchIndexUpdated upsert 分类搜索文档
func handleCategorySearchIndexUpdated(ctx context.Context, event *CategorySearchIndexUpdatedEvent) error {
	if event == nil || event.CategoryId == 0 {
		return nil
	}
	entity := category.Get(event.CategoryId)
	if entity.Id == 0 {
		return nil
	}
	_, err := searchservice.BuildSingleCategorySearchDocument(&entity)
	return err
}

// CategorySearchIndexDeletedEvent 分类删除后移除搜索索引
type CategorySearchIndexDeletedEvent struct {
	CategoryId uint64
}

func (event *CategorySearchIndexDeletedEvent) Subject() (uint64, uint64, string) {
	if event == nil {
		return 0, 0, ""
	}
	return event.CategoryId, 0, ""
}

// handleCategorySearchIndexDeleted 删除分类搜索文档
func handleCategorySearchIndexDeleted(ctx context.Context, event *CategorySearchIndexDeletedEvent) error {
	if event == nil || event.CategoryId == 0 {
		return nil
	}
	_, err := searchservice.DeleteCategorySearchDocument(event.CategoryId)
	return err
}
