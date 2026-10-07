package api

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentcommentservice"
)

// AgentCommentPolicyVo 管理端「Agent 评论策略」全局开关回显。
type AgentCommentPolicyVo struct {
	AllowAgentComments bool `json:"allowAgentComments"`
}

// GetAgentCommentPolicy 读取全局 Agent 评论开关（未保存过时回内置默认：允许）。
func GetAgentCommentPolicy(req component.BetterRequest[component.Null]) component.Response {
	config := hotdataserve.GetAgentCommentPolicyConfigCache()
	return component.SuccessResponse(AgentCommentPolicyVo{AllowAgentComments: config.AllowAgentComments})
}

type SaveAgentCommentPolicyReq struct {
	AllowAgentComments *bool `json:"allowAgentComments" validate:"required"`
}

// SaveAgentCommentPolicy 保存全局 Agent 评论开关（热生效，无需重启）。
func SaveAgentCommentPolicy(req component.BetterRequest[SaveAgentCommentPolicyReq]) component.Response {
	config := pageConfig.AgentCommentPolicyConfig{AllowAgentComments: *req.Params.AllowAgentComments}
	if err := agentcommentservice.SaveGlobalPolicy(config); err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

type SetAgentCommentTopicPolicyReq struct {
	TopicId  uint64 `json:"topicId" validate:"required"`
	Disabled bool   `json:"disabled"`
}

// SetAgentCommentTopicPolicy 设置单个主题的 Agent 评论禁止标记。
func SetAgentCommentTopicPolicy(req component.BetterRequest[SetAgentCommentTopicPolicyReq]) component.Response {
	topic := topics.Get(req.Params.TopicId)
	if topic.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}
	if err := agentcommentservice.SetTopicPolicy(topic.Id, req.Params.Disabled); err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponse(component.DataMap{
		"topicId":              topic.Id,
		"agentCommentDisabled": req.Params.Disabled,
	})
}
