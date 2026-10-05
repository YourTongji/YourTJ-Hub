package api

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/topicpolicyservice"
	"gorm.io/gorm"
)

type TopicAgentRepliesReq struct {
	TopicId              uint64 `json:"topicId" validate:"required"`
	AgentRepliesDisabled *bool  `json:"agentRepliesDisabled" validate:"required"`
}

func UpdateTopicAgentReplies(req component.BetterRequest[TopicAgentRepliesReq]) component.Response {
	topic, err := topicpolicyservice.SetAgentRepliesDisabled(betterRequestContext(req), req.Params.TopicId, req.UserId, *req.Params.AgentRepliesDisabled)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}
	if errors.Is(err, topicpolicyservice.ErrSettingDenied) {
		return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
	}
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	hotdataserve.InvalidateTopicListCacheForCategories(topic.CategoryIds...)
	return component.SuccessResponse(true)
}
