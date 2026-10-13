package topics

import "testing"

// TestPublicScopePredicatesAreCanonical 固定规范谓词的逐字文本：公开列表、关注流、
// 排行榜与内容贡献统计共用它，文档与 API 契约也按此原文描述口径；
// 任何改动都必须同步 docs/ 与 packages/api-contract。
func TestPublicScopePredicatesAreCanonical(t *testing.T) {
	if got, want := PublicScopePredicate("topics"),
		"topics.status = 1 AND topics.process_status = 0 AND topics.visibility_status = 'ACTIVE' AND topics.deleted_at IS NULL AND topics.topic_type = 0"; got != want {
		t.Fatalf("PublicScopePredicate = %q, want %q", got, want)
	}
	if got, want := FirstPostVisiblePredicate("topics"),
		"(SELECT posts.process_status FROM posts WHERE posts.id = topics.first_post_id AND posts.topic_id = topics.id AND posts.deleted_at IS NULL AND posts.visibility_status = 'ACTIVE') = 0"; got != want {
		t.Fatalf("FirstPostVisiblePredicate = %q, want %q", got, want)
	}
}
