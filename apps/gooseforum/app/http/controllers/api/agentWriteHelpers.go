package api

import (
	"errors"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWrites"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentcommentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentwriteservice"
)

func agentWriteFailure(err error) component.Response {
	switch {
	case errors.Is(err, agentWrites.ErrConflict):
		return component.BuildResponse(http.StatusConflict, component.FailDataCode("agent.write.idempotencyConflict", nil))
	case errors.Is(err, agentwriteservice.ErrInvalidKey):
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	case errors.Is(err, agentwriteservice.ErrInstanceUnconfigured):
		return component.FailResponseCode("agent.events.instance_unconfigured", nil)
	case errors.Is(err, agentservice.ErrAgentTokenInvalid), errors.Is(err, agentservice.ErrAgentDisabled):
		return agentWriteAuthFailure()
	case errors.Is(err, users.ErrInteractionBlocked):
		return agentEventFailure(agenteventservice.ErrInaccessible)
	case errors.Is(err, agentcommentservice.ErrAgentCommentDisabled):
		return component.FailResponseCode(component.MessageTopicAgentCommentDisabled, nil)
	default:
		return agentEventFailure(err)
	}
}

// Reconstruct only the still-visible resource. The ledger contains no content
// body or response snapshot that can outlive deletion, moderation or anonymity.
func agentReplayResponse(userID uint64, entry *agentWrites.Entry) component.Response {
	p := posts.Get(entry.PostID)
	t := topics.Get(entry.TopicID)
	if p.Id == 0 || t.Id == 0 || p.UserId != userID || p.IsAnonymous || p.VisibilityStatus != posts.VisibilityActive || p.ProcessStatus == posts.ProcessStatusBlocked || t.VisibilityStatus != topics.VisibilityActive || t.Status != 1 || t.ProcessStatus == topics.ProcessStatusBlocked {
		return agentEventFailure(agenteventservice.ErrInaccessible)
	}
	pending := p.ProcessStatus == posts.ProcessStatusPending || t.ProcessStatus == topics.ProcessStatusPending
	if entry.Operation == "topic" {
		return publishSuccess(t.Id, pending, false)
	}
	return publishSuccess(map[string]any{"id": p.Id, "postNo": p.PostNo, "renderedContent": p.RenderedHTML, "isAnswer": isAnswerPost(p.ReplyToPostId, &t)}, pending, false)
}
