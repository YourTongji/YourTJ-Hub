package api

import (
	"fmt"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/topicpolicyservice"
)

func TestAgentWriteFailurePreservesAuthorReplyPolicyCode(t *testing.T) {
	response := agentWriteFailure(fmt.Errorf("write transaction: %w", topicpolicyservice.ErrAgentRepliesDisabled))
	if response.Data.Code != component.FAIL || response.Data.MessageCode != component.MessageTopicAgentRepliesDisabled {
		t.Fatalf("author policy response = %+v", response)
	}
}
