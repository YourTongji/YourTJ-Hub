package cmd

import "testing"

// TestBuildSeedTopicRequestsPublishTopics 回归 issue #645：seed 话题必须以
// TopicStatus=1（已发布）写入，否则首页 FilterStatus=true 会过滤掉 status=0
// 的种子话题，本地预览看不到任何内容。
func TestBuildSeedTopicRequestsPublishTopics(t *testing.T) {
	userIds := []uint64{10, 20, 30}

	reqs := buildSeedTopicRequests(userIds)
	if len(reqs) != 2 {
		t.Fatalf("buildSeedTopicRequests() returned %d requests, want 2", len(reqs))
	}

	for i, req := range reqs {
		if req.Params.TopicStatus != 1 {
			t.Errorf("topic %d TopicStatus = %d, want 1 (published)", i, req.Params.TopicStatus)
		}
		if req.UserId == 0 {
			t.Errorf("topic %d UserId = 0, want a demo user id", i)
		}
		if len(req.Params.CategoryId) != 1 || req.Params.CategoryId[0] != 1 {
			t.Errorf("topic %d CategoryId = %v, want [1]", i, req.Params.CategoryId)
		}
	}
}