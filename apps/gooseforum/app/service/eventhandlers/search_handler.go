package eventhandlers

import (
	"context"

	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
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
