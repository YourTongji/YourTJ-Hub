package eventhandlers

import (
	"context"

	"github.com/leancodebox/GooseForum/app/models/forum/agentInbox"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/service/agentinboxservice"
)

// handleAgentMentionTopicPublished 为主题发布事件扫描 Agent 提及。
// 事件只在已发布内容上产生（topicStatus=1 且非待审），待审内容由审核
// 批准路径补发同一事件，无需特殊审核代码。
func handleAgentMentionTopicPublished(ctx context.Context, event *TopicPublishedEvent) error {
	if event == nil {
		return nil
	}
	return handleAgentMentionTopicEvent(ctx, event.Topic, event.FirstPost, agentInbox.EventTypeTopicPublished)
}

// handleAgentMentionTopicUpdated 为主题更新事件扫描 Agent 提及。
func handleAgentMentionTopicUpdated(ctx context.Context, event *TopicUpdatedEvent) error {
	if event == nil {
		return nil
	}
	return handleAgentMentionTopicEvent(ctx, event.Topic, event.FirstPost, agentInbox.EventTypeTopicUpdated)
}

// handleAgentMentionCommentCreated 为评论/回复创建事件扫描 Agent 提及。
func handleAgentMentionCommentCreated(ctx context.Context, event *CommentCreatedEvent) error {
	if event == nil {
		return nil
	}
	return agentinboxservice.HandleMentionEvent(ctx, agentinboxservice.MentionEventParams{
		EventType: agentInbox.EventTypePostCreated,
		TopicId:   event.TopicId,
		PostId:    event.PostId,
		ActorId:   event.UserId,
		Content:   event.Content,
		Preview:   TakeUpTo64Chars(event.Content),
	})
}

// handleAgentMentionTopicEvent 处理主题发布/更新事件：标题 + 首楼内容
// 一起参与提及扫描，预览取首楼内容（降级为标题）。
func handleAgentMentionTopicEvent(ctx context.Context, topic *topics.Entity, firstPost *posts.Entity, eventType string) error {
	if topic == nil {
		return nil
	}
	content := ""
	preview := topic.Title
	if firstPost != nil {
		content = firstPost.Content
		preview = TakeUpTo64Chars(content)
	}
	return agentinboxservice.HandleMentionEvent(ctx, agentinboxservice.MentionEventParams{
		EventType: eventType,
		TopicId:   topic.Id,
		PostId:    topic.FirstPostId,
		ActorId:   topic.UserId,
		Title:     topic.Title,
		Content:   content,
		Preview:   preview,
	})
}
