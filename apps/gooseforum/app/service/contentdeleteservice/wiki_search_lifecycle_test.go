package contentdeleteservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"github.com/meilisearch/meilisearch-go"
)

// TestDeleteAllUserContentRemovesWikiSearchIndex 回归 PR313 最新 P1：
// 账户注销必须通过统一 Wiki 话题删除生命周期清理段落索引，不能先删 wiki_pages
// 再调用 DeleteTopicByUser，导致 DeleteTopicAs 找不到页面而留下陈旧文档。
func TestDeleteAllUserContentRemovesWikiSearchIndex(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	if !meiliconnect.IsAvailable() {
		t.Skip("Meilisearch not available; Wiki deletion index regression not testable")
	}
	configureWikiSearchIndexForLifecycleTest(t)

	const (
		authorID = uint64(9_500_001)
		topicID  = uint64(9_500_100)
	)
	pageID := seedWikiTopic(t, conn, topicID, authorID)
	page := wikiPages.Get(pageID)
	page.ParaAnchors = `[{"index":1,"anchor":"s-1","headingId":"intro","headingText":"简介","text":"账户注销遗留内容"}]`
	if err := wikiPages.Save(&page); err != nil {
		t.Fatalf("save wiki search page: %v", err)
	}
	t.Cleanup(func() { _ = searchservice.DeleteWikiPageDocuments(pageID) })

	if err := searchservice.IndexWikiPageDocuments(pageID); err != nil {
		t.Fatalf("index wiki page: %v", err)
	}
	waitWikiSearchTasks(t)
	assertWikiSearchPagePresent(t, pageID, "账户注销遗留内容")

	// 模拟旧账户注销路径已经先软删页面、但尚未清理搜索索引的中间状态。
	// 统一删除生命周期仍必须通过 unscoped page lookup 找到 pageId 并清理索引。
	if err := wikiPages.Delete(pageID); err != nil {
		t.Fatalf("soft-delete wiki page before account close: %v", err)
	}
	if err := DeleteAllUserContent(authorID); err != nil {
		t.Fatalf("DeleteAllUserContent: %v", err)
	}
	waitWikiSearchTasks(t)
	assertWikiSearchPageAbsent(t, pageID, "账户注销遗留内容")
}

func configureWikiSearchIndexForLifecycleTest(t *testing.T) {
	t.Helper()
	index := meiliconnect.GetClient().Index(searchservice.WikiPageIndex)
	searchable := []string{"title", "heading", "paragraph"}
	if _, err := index.UpdateSearchableAttributes(&searchable); err != nil {
		t.Fatalf("configure wiki searchable attributes: %v", err)
	}
	filterable := []any{"pageId", "namespace"}
	if _, err := index.UpdateFilterableAttributes(&filterable); err != nil {
		t.Fatalf("configure wiki filterable attributes: %v", err)
	}
	waitWikiSearchTasks(t)
}

func waitWikiSearchTasks(t *testing.T) {
	t.Helper()
	client := meiliconnect.GetClient()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		tasks, err := client.GetTasks(&meilisearch.TasksQuery{
			IndexUIDS: []string{searchservice.WikiPageIndex},
		})
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		settled := true
		for _, task := range tasks.Results {
			if task.Status == meilisearch.TaskStatusEnqueued || task.Status == meilisearch.TaskStatusProcessing {
				settled = false
				break
			}
			if task.Status == meilisearch.TaskStatusFailed {
				t.Fatalf("wiki search task failed: %s", task.Error)
			}
		}
		if settled {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("wiki search tasks did not settle within 10s")
}

func assertWikiSearchPagePresent(t *testing.T, pageID uint64, query string) {
	t.Helper()
	response, err := searchservice.SearchWikiPageHits(query, 20)
	if err != nil {
		t.Fatalf("search %q: %v", query, err)
	}
	for _, hit := range response.Hits {
		if hit.PageId == pageID {
			return
		}
	}
	t.Fatalf("page %d was not found for %q", pageID, query)
}

func assertWikiSearchPageAbsent(t *testing.T, pageID uint64, query string) {
	t.Helper()
	response, err := searchservice.SearchWikiPageHits(query, 20)
	if err != nil {
		t.Fatalf("search %q: %v", query, err)
	}
	for _, hit := range response.Hits {
		if hit.PageId == pageID {
			t.Fatalf("page %d still found for %q after deletion: %+v", pageID, query, hit)
		}
	}
}
