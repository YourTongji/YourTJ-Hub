package searchservice

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/connect/meiliconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/meilisearch/meilisearch-go"
)

// TestAggregateSearchE2E 验证聚合搜索端到端行为（真实 Meilisearch，TEST_MEILI_URL 门控）。
// 注意：展示数据由 DB 重构（索引只存候选 ID），因此测试必须真实写入 DB。
func TestAggregateSearchE2E(t *testing.T) {
	if os.Getenv("TEST_MEILI_URL") == "" {
		t.Skip("TEST_MEILI_URL not set; skipping aggregate search e2e test")
	}
	client := meiliconnect.GetClient()
	unique := time.Now().UnixNano() & 0x7fffffff
	suffix := fmt.Sprintf("_%d", unique%1000000)

	// 初始化 DB schema（测试库为内存库，需先建表）
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &category.Entity{}); err != nil {
		t.Fatalf("migrate test tables: %v", err)
	}
	// 配置 users 索引 searchableAttributes（单文档 upsert 不会重配索引）
	if err := configureUserIndex(client.Index(UserIndex)); err != nil {
		t.Fatalf("configure user index: %v", err)
	}
	waitIndexTask(t, client)

	// 1. 中文用户 + 拼音字段（真实写入 DB，索引只存候选 ID）
	user := users.MakeUser("zhangsan"+suffix, "password123", "zhangsan@test.com")
	user.Nickname = "张三"
	user.Bio = "同济大学学生"
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.Id == 0 {
		t.Fatal("user id not populated after create")
	}
	if _, err := BuildSingleUserSearchDocument(user); err != nil {
		t.Fatalf("build user doc: %v", err)
	}
	waitIndexTask(t, client)

	// 2. 冻结用户（应正常显示）
	frozenUser := users.MakeUser("lisi"+suffix, "password123", "lisi@test.com")
	frozenUser.Nickname = "李四"
	frozenUser.IsFrozen = 1
	if err := users.Create(frozenUser); err != nil {
		t.Fatalf("create frozen user: %v", err)
	}
	if _, err := BuildSingleUserSearchDocument(frozenUser); err != nil {
		t.Fatalf("build frozen user doc: %v", err)
	}
	waitIndexTask(t, client)

	// 3. 中文分类 + 拼音字段（真实写入 DB）
	cat := category.Entity{Name: fmt.Sprintf("校园生活%d", unique%1000000), Slug: "campus-life", Sort: int(unique % 100)}
	category.SaveOrCreateById(&cat)
	if cat.Id == 0 {
		t.Fatal("category id not populated after save")
	}
	if _, err := BuildSingleCategorySearchDocument(&cat); err != nil {
		t.Fatalf("build category doc: %v", err)
	}
	waitIndexTask(t, client)

	// 4. 拼音全拼命中用户昵称
	resp, err := AggregateSearch(AggregateSearchRequest{Query: "zhangsan", Scope: ScopeAll, Limit: 10})
	if err != nil {
		t.Fatalf("search zhangsan: %v", err)
	}
	if !containsUser(resp.Users, user.Id) {
		t.Fatalf("zhangsan should match user %d, got %+v", user.Id, resp.Users)
	}

	// 5. 首字母命中（nickname 张三 -> ZS）
	resp, err = AggregateSearch(AggregateSearchRequest{Query: "zs", Scope: ScopeUsers, Limit: 10})
	if err != nil {
		t.Fatalf("search zs: %v", err)
	}
	if !containsUser(resp.Users, user.Id) {
		t.Fatalf("initials 'zs' should match 张三, got %+v", resp.Users)
	}

	// 6. scope=users 只返回用户、不含分类
	resp, err = AggregateSearch(AggregateSearchRequest{Query: "校园", Scope: ScopeUsers, Limit: 10})
	if err != nil {
		t.Fatalf("search scope users: %v", err)
	}
	if len(resp.Categories) != 0 {
		t.Fatalf("scope users should not return categories, got %+v", resp.Categories)
	}

	// 7. 分类拼音命中（校园生活 -> xiaoyuanshenghuo）
	resp, err = AggregateSearch(AggregateSearchRequest{Query: "xiaoyuan", Scope: ScopeAll, Limit: 10})
	if err != nil {
		t.Fatalf("search xiaoyuan: %v", err)
	}
	if !containsCategory(resp.Categories, cat.Id) {
		t.Fatalf("pinyin 'xiaoyuan' should match 校园生活, got %+v", resp.Categories)
	}

	// 8. 冻结用户仍显示
	resp, err = AggregateSearch(AggregateSearchRequest{Query: "lisi", Scope: ScopeUsers, Limit: 10})
	if err != nil {
		t.Fatalf("search lisi: %v", err)
	}
	if !containsUser(resp.Users, frozenUser.Id) {
		t.Fatalf("frozen user should still appear, got %+v", resp.Users)
	}

	// 9. 部分降级：删除 users 索引 -> failedScopes 含 users，分类仍可搜
	if _, err := client.DeleteIndex(UserIndex); err != nil {
		t.Fatalf("delete users index: %v", err)
	}
	waitIndexTask(t, client)
	resp, err = AggregateSearch(AggregateSearchRequest{Query: "xiaoyuan", Scope: ScopeAll, Limit: 10})
	if err != nil {
		t.Fatalf("search after index deletion: %v", err)
	}
	if len(resp.FailedScopes) == 0 {
		t.Fatalf("failedScopes should be non-empty after users index deletion, got %+v", resp.FailedScopes)
	}
	if !containsCategory(resp.Categories, cat.Id) {
		t.Fatalf("categories should still return after users index deletion, got %+v", resp.Categories)
	}

	// 清理：恢复 users 索引（测试 DB 为内存库，无需清理数据）
	_, _ = BuildUserIndex()
	waitIndexTask(t, client)
}

func containsUser(users []UserSearchResult, id uint64) bool {
	for _, u := range users {
		if u.ID == id {
			return true
		}
	}
	return false
}

func containsCategory(cats []CategorySearchResult, id uint64) bool {
	for _, c := range cats {
		if c.ID == id {
			return true
		}
	}
	return false
}

func waitIndexTask(t *testing.T, client meilisearch.ServiceManager) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		tasks, err := client.GetTasks(&meilisearch.TasksQuery{
			IndexUIDS: []string{TopicIndex, UserIndex, CategoryIndex},
		})
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		done := true
		for _, task := range tasks.Results {
			if task.Status == meilisearch.TaskStatusEnqueued || task.Status == meilisearch.TaskStatusProcessing {
				done = false
				break
			}
			if task.Status == meilisearch.TaskStatusFailed {
				t.Fatalf("meilisearch task failed: %s", task.Error)
			}
		}
		if done {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("meilisearch tasks did not settle within 10s")
}
