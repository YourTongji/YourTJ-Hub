package topicaccessservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

func TestCanViewPublicAndOwnerDraftTopics(t *testing.T) {
	public := topics.Entity{
		Id:               1,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		VisibilityStatus: topics.VisibilityActive,
	}
	if !CanView(&public, 0) {
		t.Fatal("anonymous viewer should see a public topic")
	}

	draft := public
	draft.Id = 2
	draft.Status = 0
	draft.UserId = 42
	if CanView(&draft, 0) || CanView(&draft, 41) {
		t.Fatal("draft leaked to a non-owner")
	}
	if !CanView(&draft, 42) {
		t.Fatal("draft owner should retain normal read access")
	}
}

func TestCanViewRejectsPurgedTopics(t *testing.T) {
	entity := topics.Entity{
		Id:               3,
		UserId:           42,
		VisibilityStatus: topics.VisibilityUserDeleted,
		RetentionStatus:  topics.RetentionPurged,
	}
	if CanView(&entity, 42) {
		t.Fatal("purged topic must remain unavailable to its former owner")
	}
}

// 隐私擦除与版主删除是「仅版主只读」状态：匿名与作者本人都不可见，保留通道
// 只供作用域内版主取证/审计（放行路径由 linkpreviewservice 的服务层测试以真实
// 版主数据端到端覆盖，这里用匿名/作者两个 viewer 钉住拒绝面）。
func TestCanViewHidesModerationOnlyStatesFromAuthorAndAnonymous(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		visibility string
	}{
		{"anonymized", topics.VisibilityAccountAnonymized},
		{"moderator removed", topics.VisibilityModeratorRemoved},
	} {
		entity := topics.Entity{
			Id:               11,
			UserId:           42,
			Status:           1,
			ProcessStatus:    topics.ProcessStatusNormal,
			VisibilityStatus: testCase.visibility,
		}
		if CanView(&entity, 0) {
			t.Fatalf("%s: anonymous viewer got access", testCase.name)
		}
		if CanView(&entity, 42) {
			t.Fatalf("%s: author retains access to a moderation-only state", testCase.name)
		}
	}
}

// 处理状态（封禁/待审）的豁免面只有 TopicsManager 与分类版主：匿名与作者都不
// 在通道内。TopicsManager 的放行路径同样由 linkpreviewservice 的服务层测试
// 以真实角色/权限数据端到端覆盖。
func TestCanViewProcessedTopicExemptsOnlyManagers(t *testing.T) {
	blocked := topics.Entity{
		Id:               12,
		UserId:           42,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusBlocked,
		VisibilityStatus: topics.VisibilityActive,
	}
	if CanView(&blocked, 0) {
		t.Fatal("anonymous viewer got access to a processed topic")
	}
	if CanView(&blocked, 42) {
		t.Fatal("author bypassed the processing state")
	}
}
