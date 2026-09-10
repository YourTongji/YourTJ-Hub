package forum

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
)

func callWikiSearchJSON(t *testing.T, query string) WikiSearchJSONResp {
	t.Helper()
	req := component.BetterRequest[WikiSearchJSONReq]{
		Params: WikiSearchJSONReq{Q: query, Limit: 12},
	}
	resp := WikiSearchJSON(req)
	if resp.Code != 200 {
		t.Fatalf("WikiSearchJSON code=%d, want 200", resp.Code)
	}
	payload, ok := resp.Data.Result.(*WikiSearchJSONResp)
	if !ok {
		t.Fatalf("WikiSearchJSON result type=%T, want *WikiSearchJSONResp", resp.Data.Result)
	}
	return *payload
}

// TestWikiSearchJSONEmptyQuery 空 query 返回空结果且不触发搜索（searchUnavailable 恒 false）。
func TestWikiSearchJSONEmptyQuery(t *testing.T) {
	result := callWikiSearchJSON(t, "")
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("empty query should return empty result: %+v", result)
	}
	if result.SearchUnavailable {
		t.Fatal("empty query should not be unavailable")
	}
}

// TestWikiSearchJSONUnavailableFallback 搜索服务不可用（无 Meilisearch）时降级：
// 返回空结果并标记 searchUnavailable，不让调用方看到 500。
// 仅当 Meilisearch 不可用时断言；本地已启动 Meilisearch 则跳过。
func TestWikiSearchJSONUnavailableFallback(t *testing.T) {
	if meiliconnect.IsAvailable() {
		t.Skip("Meilisearch available; unavailable fallback not testable in this environment")
	}
	result := callWikiSearchJSON(t, "选课")
	if !result.SearchUnavailable {
		t.Fatalf("expected searchUnavailable fallback without Meilisearch, got %+v", result)
	}
	if len(result.Items) != 0 {
		t.Fatalf("unavailable fallback should have empty items, got %d", len(result.Items))
	}
}
