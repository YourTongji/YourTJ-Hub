package eventhandlers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/connect/meiliconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/searchservice"
	"github.com/meilisearch/meilisearch-go"
)

// TestSearchSyncE2E 验证事件驱动索引同步（真实 Meilisearch，TEST_MEILI_URL 门控）。
// 覆盖：注册（UserSignUpEvent handler）→ 可搜；改昵称/用户名（UserSearchIndexUpdatedEvent）
// → 新值可搜旧值不可搜；分类改名（CategorySearchIndexUpdatedEvent）→ 新名可搜旧名不可搜；
// 分类删除（CategorySearchIndexDeletedEvent）→ 不再命中。
func TestSearchSyncE2E(t *testing.T) {
	if os.Getenv("TEST_MEILI_URL") == "" {
		t.Skip("TEST_MEILI_URL not set; skipping search sync e2e test")
	}

	ctx := context.Background()
	client := meiliconnect.GetClient()
	unique := time.Now().UnixNano() & 0x7fffffff
	suffix := fmt.Sprintf("_%d", unique%1000000)

	// 初始化 DB schema（测试库为内存库，需先建表）
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &category.Entity{}); err != nil {
		t.Fatalf("migrate test tables: %v", err)
	}

	// 1. 注册（UserSignUpEvent handler）→ 新用户可搜
	user := users.MakeUser("alice"+suffix, "password123", "alice@test.com")
	user.Nickname = "爱丽丝"
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := handleUserSignUpSearchIndex(ctx, &UserSignUpEvent{UserId: user.Id, Username: user.Username}); err != nil {
		t.Fatalf("handleUserSignUpSearchIndex: %v", err)
	}
	waitIndexTask(t, client)
	if !userSearchable(t, user.Id, user.Username) {
		t.Fatalf("registered user %d (%s) should be searchable", user.Id, user.Username)
	}

	// 2. 改昵称（UserSearchIndexUpdatedEvent）→ 新昵称可搜、旧昵称不可搜。
	//    使用完全不重叠的昵称，避免 Meilisearch 中文分词让旧词作为子词元命中新文档。
	oldNickname := user.Nickname
	newNickname := "小蓝"
	user.Nickname = newNickname
	if err := users.Save(user); err != nil {
		t.Fatalf("save user: %v", err)
	}
	if err := handleUserSearchIndexUpdated(ctx, &UserSearchIndexUpdatedEvent{UserId: user.Id}); err != nil {
		t.Fatalf("handleUserSearchIndexUpdated: %v", err)
	}
	waitIndexTask(t, client)
	if !userSearchable(t, user.Id, newNickname) {
		t.Fatalf("new nickname %q should be searchable", newNickname)
	}
	if userSearchable(t, user.Id, oldNickname) {
		t.Fatalf("old nickname %q should no longer be searchable", oldNickname)
	}

	// 3. 改用户名（UserSearchIndexUpdatedEvent）→ 新用户名可搜、旧用户名不可搜
	oldUsername := user.Username
	newUsername := "bob" + suffix
	user.Username = newUsername
	if err := users.Save(user); err != nil {
		t.Fatalf("save user: %v", err)
	}
	if err := handleUserSearchIndexUpdated(ctx, &UserSearchIndexUpdatedEvent{UserId: user.Id}); err != nil {
		t.Fatalf("handleUserSearchIndexUpdated: %v", err)
	}
	waitIndexTask(t, client)
	if !userSearchable(t, user.Id, newUsername) {
		t.Fatalf("new username %q should be searchable", newUsername)
	}
	if userSearchable(t, user.Id, oldUsername) {
		t.Fatalf("old username %q should no longer be searchable", oldUsername)
	}

	// 4. 分类创建（CategorySearchIndexUpdatedEvent）→ 新分类可搜
	cat := category.Entity{Name: fmt.Sprintf("校园热点%d", unique%1000000), Slug: "test-cat", Sort: int(unique % 100)}
	category.SaveOrCreateById(&cat)
	if cat.Id == 0 {
		t.Fatal("category id not populated after save")
	}
	if err := handleCategorySearchIndexUpdated(ctx, &CategorySearchIndexUpdatedEvent{CategoryId: cat.Id}); err != nil {
		t.Fatalf("handleCategorySearchIndexUpdated: %v", err)
	}
	waitIndexTask(t, client)
	if !categorySearchable(t, cat.Id, cat.Name) {
		t.Fatalf("created category %q should be searchable", cat.Name)
	}

	// 5. 分类改名（CategorySearchIndexUpdatedEvent）→ 新名可搜、旧名不可搜（完全不重叠）
	oldCatName := cat.Name
	newCatName := fmt.Sprintf("二手交易%d", unique%1000000)
	cat.Name = newCatName
	category.SaveOrCreateById(&cat)
	if err := handleCategorySearchIndexUpdated(ctx, &CategorySearchIndexUpdatedEvent{CategoryId: cat.Id}); err != nil {
		t.Fatalf("handleCategorySearchIndexUpdated: %v", err)
	}
	waitIndexTask(t, client)
	if !categorySearchable(t, cat.Id, newCatName) {
		t.Fatalf("renamed category %q should be searchable", newCatName)
	}
	if categorySearchable(t, cat.Id, oldCatName) {
		t.Fatalf("old category name %q should no longer be searchable", oldCatName)
	}

	// 6. 分类删除（CategorySearchIndexDeletedEvent）→ 不再命中
	if err := handleCategorySearchIndexDeleted(ctx, &CategorySearchIndexDeletedEvent{CategoryId: cat.Id}); err != nil {
		t.Fatalf("handleCategorySearchIndexDeleted: %v", err)
	}
	waitIndexTask(t, client)
	if categorySearchable(t, cat.Id, newCatName) {
		t.Fatalf("deleted category %q should no longer be searchable", newCatName)
	}
}

// userSearchable 通过 AggregateSearch 检查用户是否命中。
func userSearchable(t *testing.T, id uint64, query string) bool {
	t.Helper()
	resp, err := searchservice.AggregateSearch(searchservice.AggregateSearchRequest{Query: query, Scope: searchservice.ScopeUsers, Limit: 10})
	if err != nil {
		t.Fatalf("AggregateSearch(%q): %v", query, err)
	}
	for _, u := range resp.Users {
		if u.ID == id {
			return true
		}
	}
	return false
}

// categorySearchable 通过 AggregateSearch 检查分类是否命中。
func categorySearchable(t *testing.T, id uint64, query string) bool {
	t.Helper()
	resp, err := searchservice.AggregateSearch(searchservice.AggregateSearchRequest{Query: query, Scope: searchservice.ScopeCategories, Limit: 10})
	if err != nil {
		t.Fatalf("AggregateSearch(%q): %v", query, err)
	}
	for _, c := range resp.Categories {
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
			IndexUIDS: []string{searchservice.TopicIndex, searchservice.UserIndex, searchservice.CategoryIndex},
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
