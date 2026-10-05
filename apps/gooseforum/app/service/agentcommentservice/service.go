// Package agentcommentservice owns the site-wide and per-topic Agent comment
// policy. The global switch is hot-read page configuration; the per-topic flag
// lives on topics.agent_comment_disabled and is maintained from the admin
// "Agent comment policy" panel.
package agentcommentservice

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// ErrAgentCommentDisabled 全站或该主题已禁止 Agent 评论；这是业务拒绝，
// 与凭据/可见性错误（inaccessible）区分，便于 Agent 判断不应重试。
var ErrAgentCommentDisabled = errors.New("agent comments disabled")

// AllowsAgentComment 判断该主题当前是否允许 Agent 发表评论。
func AllowsAgentComment(topic topics.Entity) bool {
	if !hotdataserve.GetAgentCommentPolicyConfigCache().AllowAgentComments {
		return false
	}
	return !topic.AgentCommentDisabled
}

// SetTopicPolicy 管理端设置单个主题的 Agent 评论禁止标记。
func SetTopicPolicy(topicID uint64, disabled bool) error {
	return topics.UpdateAgentCommentDisabled(topicID, disabled)
}
