package eventhandlers

import (
	"context"

	"github.com/leancodebox/GooseForum/app/service/llmsservice"
)

func handleLLMSTopicPublished(context.Context, *TopicPublishedEvent) error {
	llmsservice.ClearCache()
	return nil
}

func handleLLMSTopicUpdated(context.Context, *TopicUpdatedEvent) error {
	llmsservice.ClearCache()
	return nil
}

func handleLLMSTopicDeleted(context.Context, *TopicDeletedEvent) error {
	llmsservice.ClearCache()
	return nil
}

func handleLLMSCommentCreated(context.Context, *CommentCreatedEvent) error {
	llmsservice.ClearCache()
	return nil
}

func handleLLMSCategoryUpdated(context.Context, *CategorySearchIndexUpdatedEvent) error {
	llmsservice.ClearCache()
	return nil
}

func handleLLMSCategoryDeleted(context.Context, *CategorySearchIndexDeletedEvent) error {
	llmsservice.ClearCache()
	return nil
}
